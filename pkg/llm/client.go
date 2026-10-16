package llm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"time"

	"github.com/dangnmh/core-kit/pkg/logger"
	"github.com/dangnmh/core-kit/pkg/ptr"
	"github.com/dangnmh/core-kit/pkg/resilience"
	"github.com/sashabaranov/go-openai"
)

// Config holds LLM client connection and inference parameters.
type Config struct {
	BaseURL      string        `yaml:"base_url" env:"LLM_BASE_URL"`
	APIKey       string        `yaml:"api_key" env:"LLM_API_KEY"`
	Model        string        `yaml:"model" env:"LLM_MODEL"`
	Temperature  *float32      `yaml:"temperature" env:"LLM_TEMPERATURE" validate:"omitempty,gte=0,lte=2"` // nil = 0.7
	Timeout      time.Duration `yaml:"timeout" env:"LLM_TIMEOUT"`
	DebugLogPath string        `yaml:"debug_log_path" env:"LLM_DEBUG_LOG_PATH"`
	MaxRetries   int           `yaml:"max_retries" env:"LLM_MAX_RETRIES"` // default 2, only for transient errors
}

// ClientOption configures an LLM Client instance.
type ClientOption func(*Client)

// WithComponent sets the component identifier for debug logging.
func WithComponent(component string) ClientOption {
	return func(c *Client) {
		c.component = component
	}
}

// WithDebugLogger sets the DebugLogger instance for the client.
func WithDebugLogger(logger DebugLogger) ClientOption {
	return func(c *Client) {
		c.debugLogger = logger
	}
}

// Client wraps the OpenAI SDK client with resilience and custom configuration.
type Client struct {
	client      *openai.Client
	config      Config
	component   string
	debugLogger DebugLogger
}

// NewClient creates a new LLM Client.
func NewClient(cfg Config, opts ...ClientOption) *Client {
	if cfg.APIKey == "" {
		cfg.APIKey = "dummy-key" // Required by OpenAI SDK even when using local proxies (Ollama / CLIProxyAPI)
	}
	if cfg.Model == "" {
		cfg.Model = "gpt-4o-mini"
	}
	if cfg.Temperature == nil {
		cfg.Temperature = ptr.To[float32](0.7)
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	} else if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 2
	}

	clientConfig := openai.DefaultConfig(cfg.APIKey)
	if cfg.BaseURL != "" {
		clientConfig.BaseURL = cfg.BaseURL
	}
	clientConfig.HTTPClient = &http.Client{
		Timeout: cfg.Timeout,
	}

	c := &Client{
		client:    openai.NewClientWithConfig(clientConfig),
		config:    cfg,
		component: "llm-client",
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.debugLogger == nil && cfg.DebugLogPath != "" && cfg.DebugLogPath != "off" && cfg.DebugLogPath != "disabled" {
		if dl, err := NewDebugLogger(cfg.DebugLogPath); err == nil {
			c.debugLogger = dl
		} else {
			logger.Warn("Failed to initialize LLM debug logger", "path", cfg.DebugLogPath, "error", err)
		}
	}

	return c
}

// Config returns the current configuration.
func (c *Client) Config() Config {
	return c.config
}

// UnderlyingClient returns the raw openai.Client for custom calls if needed.
func (c *Client) UnderlyingClient() *openai.Client {
	return c.client
}

// DebugLogger returns the configured DebugLogger, or nil if none.
func (c *Client) DebugLogger() DebugLogger {
	return c.debugLogger
}

// Close closes any resources owned by Client, including DebugLogger.
func (c *Client) Close() error {
	if c.debugLogger != nil {
		return c.debugLogger.Close()
	}
	return nil
}

// isTransient determines whether an error from the LLM client call is retryable.
// 429, 5xx, and network timeouts are transient; 400, 401, and context cancellation are NOT.
func isTransient(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var apiErr *openai.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.HTTPStatusCode
		if code == http.StatusTooManyRequests || (code >= 500 && code <= 599) {
			return true
		}
		return false
	}

	var reqErr *openai.RequestError
	if errors.As(err, &reqErr) {
		code := reqErr.HTTPStatusCode
		if code == http.StatusTooManyRequests || (code >= 500 && code <= 599) {
			return true
		}
		if reqErr.Err != nil && isTransient(reqErr.Err) {
			return true
		}
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return true
		}
	}

	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}

	return false
}

// Chat performs a chat completion request with retry on transient errors and returns the raw response text.
func (c *Client) Chat(ctx context.Context, messages []openai.ChatCompletionMessage) (string, error) {
	temp := float32(0.7)
	if c.config.Temperature != nil {
		temp = *c.config.Temperature
	}
	if temp == 0 {
		temp = math.SmallestNonzeroFloat32
	}

	req := openai.ChatCompletionRequest{
		Model:       c.config.Model,
		Messages:    messages,
		Temperature: temp,
	}

	maxRetries := c.config.MaxRetries
	if maxRetries < 0 {
		maxRetries = 0
	}

	backoff := resilience.NewBackoff(100*time.Millisecond, 2*time.Second)

	var lastErr error
	var resp openai.ChatCompletionResponse
	var start time.Time
	var latency time.Duration

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			if err := backoff.Sleep(ctx, attempt-1); err != nil {
				return "", fmt.Errorf("llm chat retry aborted: %w", err)
			}
		}

		start = time.Now()
		resp, lastErr = c.client.CreateChatCompletion(ctx, req)
		latency = time.Since(start)

		if lastErr == nil {
			if len(resp.Choices) == 0 {
				lastErr = fmt.Errorf("llm returned no choices")
			}
			break
		}

		if !isTransient(lastErr) || ctx.Err() != nil || attempt == maxRetries {
			break
		}
	}

	var content string
	var errStr string
	if lastErr != nil {
		errStr = lastErr.Error()
	} else if len(resp.Choices) > 0 {
		content = resp.Choices[0].Message.Content
	}

	if c.debugLogger != nil {
		_ = c.debugLogger.Log(DebugRecord{
			Timestamp: start.UTC(),
			Component: c.component,
			Model:     c.config.Model,
			Endpoint:  c.config.BaseURL,
			Prompt:    messages,
			Response:  content,
			LatencyMs: latency.Milliseconds(),
			Error:     errStr,
		})
	}

	if lastErr != nil {
		return "", fmt.Errorf("llm chat completion failed: %w", lastErr)
	}

	return content, nil
}

// GenerateJSON executes a chat completion on Client and automatically parses the output into struct T.
func (c *Client) GenerateJSON[T any](ctx context.Context, messages []openai.ChatCompletionMessage) (*T, error) {
	content, err := c.Chat(ctx, messages)
	if err != nil {
		return nil, err
	}

	return ExtractJSON[T](content)
}

// GenerateJSON executes a chat completion using Client and automatically parses the output into struct T.
func GenerateJSON[T any](ctx context.Context, c *Client, messages []openai.ChatCompletionMessage) (*T, error) {
	return c.GenerateJSON[T](ctx, messages)
}
