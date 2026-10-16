package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dangnmh/core-kit/pkg/httplog"
	"github.com/dangnmh/core-kit/pkg/logger"
	"github.com/dangnmh/core-kit/pkg/tracer"
	"github.com/dangnmh/transport"
)

var ErrBodyTooLarge = errors.New("httpclient: response body exceeds limit")

const (
	DefaultMaxResponseBytes int64 = 8 * 1024 * 1024 // 8 MiB
	DefaultBaseWait               = 200 * time.Millisecond
	DefaultMaxWait                = 5 * time.Second
	DefaultMaxRetryAfter          = 10 * time.Second
)

type RetryPolicy struct {
	MaxRetries         int
	BaseWait           time.Duration // default 200ms
	MaxWait            time.Duration // default 5s
	MaxRetryAfter      time.Duration // default 10s
	RetryNonIdempotent bool          // default false
}

func (p RetryPolicy) withDefaults() RetryPolicy {
	cp := p
	if cp.BaseWait <= 0 {
		cp.BaseWait = DefaultBaseWait
	}
	if cp.MaxWait <= 0 {
		cp.MaxWait = DefaultMaxWait
	}
	if cp.MaxRetryAfter <= 0 {
		cp.MaxRetryAfter = DefaultMaxRetryAfter
	}
	return cp
}

type Response struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

func (r *Response) IsSuccess() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

func (r *Response) JSON(target any) error {
	return json.Unmarshal(r.Body, target)
}

func (r *Response) String() string {
	return string(r.Body)
}

type Client struct {
	baseURL          string
	httpClient       *http.Client
	defaultHeaders   map[string]string
	retryPolicy      RetryPolicy
	maxResponseBytes int64
	baseTransport    http.RoundTripper
	logEnabled       bool
	logOptions       []transport.LogOption
	logEngine        httplog.Engine
}

type Option func(*Client)

func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = strings.TrimRight(url, "/")
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// WithRetries sets retry count and wait duration for backward compatibility.
func WithRetries(retries int, wait time.Duration) Option {
	return func(c *Client) {
		c.retryPolicy.MaxRetries = retries
		if wait > 0 {
			c.retryPolicy.BaseWait = wait
		}
	}
}

// WithRetryPolicy configures detailed retry behavior.
func WithRetryPolicy(p RetryPolicy) Option {
	return func(c *Client) {
		c.retryPolicy = p
	}
}

// WithMaxResponseBytes configures the maximum allowed response body size (default 8 MiB).
func WithMaxResponseBytes(n int64) Option {
	return func(c *Client) {
		if n > 0 {
			c.maxResponseBytes = n
		}
	}
}

func WithDefaultHeader(key, value string) Option {
	return func(c *Client) {
		c.defaultHeaders[key] = value
	}
}

// WithTransport sets a custom base http.RoundTripper before transport log wrapping.
func WithTransport(tp http.RoundTripper) Option {
	return func(c *Client) {
		c.baseTransport = tp
	}
}

// WithTransportLog enables transport-level logging with custom dangnmh/transport options.
func WithTransportLog(opts ...transport.LogOption) Option {
	return func(c *Client) {
		c.logEnabled = true
		c.logOptions = append(c.logOptions, opts...)
	}
}

// WithoutTransportLog disables transport-level request/response logging.
func WithoutTransportLog() Option {
	return func(c *Client) {
		c.logEnabled = false
	}
}

// WithLogEngine configures the client to sanitize and log outbound HTTP requests and responses
// using the provided httplog.Engine.
func WithLogEngine(engine httplog.Engine) Option {
	return func(c *Client) {
		c.logEngine = engine
		if engine != nil {
			c.logEnabled = false
		}
	}
}

// WithHTTPClient allows injecting an existing *http.Client without mutating caller's client.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		if client != nil {
			cp := *client
			c.httpClient = &cp
		}
	}
}

// WithPoolConfig configures the underlying *http.Client using a tuned PoolConfig
// (supporting LocalAddr EIP binding, ProxyURL, HTTP/2 warm connection pooling, and 429 retry).
func WithPoolConfig(cfg PoolConfig) Option {
	return WithHTTPClient(NewPool(cfg))
}

func New(opts ...Option) *Client {
	c := &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		defaultHeaders:   make(map[string]string),
		maxResponseBytes: DefaultMaxResponseBytes,
		retryPolicy: RetryPolicy{
			MaxRetries:    0,
			BaseWait:      DefaultBaseWait,
			MaxWait:       DefaultMaxWait,
			MaxRetryAfter: DefaultMaxRetryAfter,
		},
		logEnabled: true,
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	baseTp := c.baseTransport
	if baseTp == nil {
		if c.httpClient.Transport != nil {
			baseTp = c.httpClient.Transport
		} else if dt, ok := http.DefaultTransport.(*http.Transport); ok {
			baseTp = dt.Clone()
		} else {
			baseTp = http.DefaultTransport
		}
	}

	if c.logEnabled && c.logEngine == nil && httplog.Default() != nil {
		c.logEngine = httplog.Default()
	}

	if c.logEngine != nil {
		c.httpClient.Transport = httplog.NewRoundTripper(baseTp, c.logEngine)
	} else if c.logEnabled {
		defaultMatcherCfg := transport.MatcherConfig{
			OnStatus:       []int{transport.NumberZero}, // Log all status codes (2xx, 3xx, 4xx, 5xx)
			WhiteListPaths: []string{transport.ConsCharStar},
			BlackListPaths: []string{},
		}
		logOpts := []transport.LogOption{
			transport.LogOptionLogger(logger.Get()),
			transport.LogOptionMatcherConfig(defaultMatcherCfg),
		}
		logOpts = append(logOpts, c.logOptions...)
		c.httpClient.Transport = transport.NewTransportLog(baseTp, logOpts...)
	} else {
		c.httpClient.Transport = baseTp
	}

	return c
}

// HTTPClient returns the underlying *http.Client instance.
func (c *Client) HTTPClient() *http.Client {
	return c.httpClient
}

func (c *Client) buildURL(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if c.baseURL == "" {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return c.baseURL + path
}

func isIdempotent(method string, headers http.Header) bool {
	switch strings.ToUpper(method) {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete, http.MethodOptions, "TRACE":
		return true
	}
	if headers != nil {
		if strings.TrimSpace(headers.Get("Idempotency-Key")) != "" || strings.TrimSpace(headers.Get("X-Idempotency-Key")) != "" {
			return true
		}
	}
	return false
}

func isDialError(err error) bool {
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return opErr.Op == "dial"
	}
	return false
}

func retryAfter(h http.Header, now time.Time) (time.Duration, bool) {
	if h == nil {
		return 0, false
	}
	val := strings.TrimSpace(h.Get("Retry-After"))
	if val == "" {
		return 0, false
	}

	// 1. Seconds integer (RFC 9110 §10.2.3)
	if seconds, err := strconv.ParseInt(val, 10, 64); err == nil {
		if seconds <= 0 {
			return 0, false
		}
		// Guard against int64 nanosecond overflow (math.MaxInt64 / time.Second ≈ 9.22e9 seconds ≈ 292 years)
		const maxSeconds = int64(time.Duration(1<<63-1) / time.Second)
		if seconds > maxSeconds {
			return time.Duration(1<<63 - 1), true
		}
		return time.Duration(seconds) * time.Second, true
	}

	// 2. HTTP-date (RFC 1123, RFC 850, asctime)
	if t, err := http.ParseTime(val); err == nil {
		diff := t.Sub(now)
		if diff <= 0 {
			return 0, false
		}
		return diff, true
	}

	return 0, false
}

func (c *Client) shouldRetry(method string, h http.Header, resp *http.Response, err error) bool {
	idemp := c.retryPolicy.RetryNonIdempotent || isIdempotent(method, h)

	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return false
		}
		if idemp || isDialError(err) {
			return true
		}
		return false
	}

	if resp == nil {
		return false
	}

	switch resp.StatusCode {
	case http.StatusTooManyRequests, // 429
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout:      // 504
		return idemp
	default:
		return false
	}
}

// calculateBackoff calculates wait time using Retry-After or full-jitter exponential backoff.
// TODO: Migrate to PKG-07 (resilience.Backoff) once merged.
func (c *Client) calculateBackoff(attempt int, resp *http.Response) time.Duration {
	p := c.retryPolicy.withDefaults()
	if resp != nil {
		if ra, ok := retryAfter(resp.Header, time.Now()); ok {
			if ra > p.MaxRetryAfter {
				return p.MaxRetryAfter
			}
			return ra
		}
	}

	maxWait := p.MaxWait
	baseWait := p.BaseWait
	capWait := baseWait
	if attempt > 0 {
		for i := 0; i < attempt; i++ {
			capWait *= 2
			if capWait >= maxWait || capWait <= 0 {
				capWait = maxWait
				break
			}
		}
	}
	if capWait > maxWait || capWait <= 0 {
		capWait = maxWait
	}

	return time.Duration(rand.Int64N(capWait.Nanoseconds()))
}

func (c *Client) Do(ctx context.Context, method, path string, body io.Reader, headers map[string]string) (*Response, error) {
	fullURL := c.buildURL(path)

	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			return nil, fmt.Errorf("read request body: %w", err)
		}
	}

	p := c.retryPolicy.withDefaults()
	attempts := 1 + p.MaxRetries
	if attempts < 1 {
		attempts = 1
	}

	var lastErr error

	for i := 0; i < attempts; i++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		var reqBody io.Reader
		if bodyBytes != nil {
			reqBody = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}

		for k, v := range c.defaultHeaders {
			req.Header.Set(k, v)
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		tracer.InjectHTTP(ctx, req.Header)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if i < attempts-1 && c.shouldRetry(method, req.Header, nil, err) {
				waitDur := c.calculateBackoff(i, nil)
				logger.DebugContext(ctx, "Retrying HTTP request after error", "url", fullURL, "attempt", i+1, "wait", waitDur, "error", err)
				timer := time.NewTimer(waitDur)
				select {
				case <-ctx.Done():
					timer.Stop()
					return nil, ctx.Err()
				case <-timer.C:
				}
				continue
			}
			return nil, fmt.Errorf("request failed after %d attempts: %w", i+1, lastErr)
		}

		maxBytes := c.maxResponseBytes
		if maxBytes <= 0 {
			maxBytes = DefaultMaxResponseBytes
		}

		respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
		_ = resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("read response body: %w", err)
			if i < attempts-1 && c.shouldRetry(method, req.Header, nil, err) {
				waitDur := c.calculateBackoff(i, resp)
				logger.DebugContext(ctx, "Retrying HTTP request after body read error", "url", fullURL, "attempt", i+1, "wait", waitDur, "error", err)
				timer := time.NewTimer(waitDur)
				select {
				case <-ctx.Done():
					timer.Stop()
					return nil, ctx.Err()
				case <-timer.C:
				}
				continue
			}
			return nil, fmt.Errorf("request failed after %d attempts: %w", i+1, lastErr)
		}

		if int64(len(respBody)) > maxBytes {
			return nil, ErrBodyTooLarge
		}

		res := &Response{
			StatusCode: resp.StatusCode,
			Header:     resp.Header,
			Body:       respBody,
		}

		if i < attempts-1 && c.shouldRetry(method, req.Header, resp, nil) {
			lastErr = fmt.Errorf("server error status %d (body %d bytes)", resp.StatusCode, len(respBody))
			waitDur := c.calculateBackoff(i, resp)
			logger.DebugContext(ctx, "Retrying HTTP request after server status", "url", fullURL, "attempt", i+1, "status", resp.StatusCode, "wait", waitDur)
			timer := time.NewTimer(waitDur)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, ctx.Err()
			case <-timer.C:
			}
			continue
		}

		return res, nil
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", attempts, lastErr)
}

func (c *Client) Get(ctx context.Context, path string, headers ...map[string]string) (*Response, error) {
	var h map[string]string
	if len(headers) > 0 {
		h = headers[0]
	}
	return c.Do(ctx, http.MethodGet, path, nil, h)
}

func (c *Client) GetJSON(ctx context.Context, path string, target any, headers ...map[string]string) error {
	resp, err := c.Get(ctx, path, headers...)
	if err != nil {
		return err
	}
	if !resp.IsSuccess() {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.String())
	}
	return resp.JSON(target)
}

func (c *Client) Post(ctx context.Context, path string, body io.Reader, contentType string, headers ...map[string]string) (*Response, error) {
	h := make(map[string]string)
	if contentType != "" {
		h["Content-Type"] = contentType
	}
	if len(headers) > 0 {
		for k, v := range headers[0] {
			h[k] = v
		}
	}
	return c.Do(ctx, http.MethodPost, path, body, h)
}

func (c *Client) PostJSON(ctx context.Context, path string, reqBody any, target any, headers ...map[string]string) error {
	var bodyReader io.Reader
	if reqBody != nil {
		data, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	resp, err := c.Post(ctx, path, bodyReader, "application/json", headers...)
	if err != nil {
		return err
	}

	if !resp.IsSuccess() {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.String())
	}

	if target != nil && len(resp.Body) > 0 {
		return resp.JSON(target)
	}
	return nil
}

// GetParsed performs a GET request using Client and unmarshals the JSON response into *T.
func (c *Client) GetParsed[T any](ctx context.Context, path string, headers ...map[string]string) (*T, error) {
	resp, err := c.Get(ctx, path, headers...)
	if err != nil {
		return nil, err
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.String())
	}
	var target T
	if err := resp.JSON(&target); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return &target, nil
}

// PostParsed performs a POST request with JSON payload using Client and unmarshals the response into *T.
func (c *Client) PostParsed[T any](ctx context.Context, path string, reqBody any, headers ...map[string]string) (*T, error) {
	var bodyReader io.Reader
	if reqBody != nil {
		data, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	resp, err := c.Post(ctx, path, bodyReader, "application/json", headers...)
	if err != nil {
		return nil, err
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.String())
	}

	var target T
	if len(resp.Body) > 0 {
		if err := resp.JSON(&target); err != nil {
			return nil, fmt.Errorf("unmarshal response: %w", err)
		}
	}
	return &target, nil
}

// Get performs a GET request using Client and unmarshals the JSON response into *T.
func Get[T any](ctx context.Context, c *Client, path string, headers ...map[string]string) (*T, error) {
	return c.GetParsed[T](ctx, path, headers...)
}

// Post performs a POST request with JSON payload using Client and unmarshals the response into *T.
func Post[T any](ctx context.Context, c *Client, path string, reqBody any, headers ...map[string]string) (*T, error) {
	return c.PostParsed[T](ctx, path, reqBody, headers...)
}
