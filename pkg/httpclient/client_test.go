package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dangnmh/core-kit/pkg/httplog"
	"github.com/dangnmh/core-kit/pkg/logger"
	"github.com/dangnmh/transport"
	"go.opentelemetry.io/otel/trace"
)

func TestHTTPClientGetJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/test" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			return
		}
		http.NotFound(w, r)
	}))
	defer func() { ts.Close() }()

	client := New(WithBaseURL(ts.URL), WithTimeout(2*time.Second))

	var res map[string]string
	err := client.GetJSON(context.Background(), "/test", &res)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res["status"] != "ok" {
		t.Fatalf("expected status ok, got %v", res["status"])
	}
}

func TestHTTPClientPostJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "bad method", http.StatusBadRequest)
			return
		}
		var req map[string]string
		_ = json.NewDecoder(r.Body).Decode(&req)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"echo": req["msg"]})
	}))
	defer func() { ts.Close() }()

	client := New(WithBaseURL(ts.URL))

	var res map[string]string
	err := client.PostJSON(context.Background(), "/echo", map[string]string{"msg": "hello"}, &res)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res["echo"] != "hello" {
		t.Fatalf("expected echo hello, got %v", res["echo"])
	}
}

func TestHTTPClientGenericGet(t *testing.T) {
	type StatusResponse struct {
		Status string `json:"status"`
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(StatusResponse{Status: "ready"})
	}))
	defer func() { ts.Close() }()

	client := New(WithBaseURL(ts.URL))
	res, err := Get[StatusResponse](context.Background(), client, "/status")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil || res.Status != "ready" {
		t.Fatalf("expected status 'ready', got %+v", res)
	}
}

func TestHTTPClientGenericPost(t *testing.T) {
	type EchoResponse struct {
		Echo string `json:"echo"`
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]string
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(EchoResponse{Echo: req["text"]})
	}))
	defer func() { ts.Close() }()

	client := New(WithBaseURL(ts.URL))
	res, err := Post[EchoResponse](context.Background(), client, "/api/echo", map[string]string{"text": "generic-call"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil || res.Echo != "generic-call" {
		t.Fatalf("expected echo 'generic-call', got %+v", res)
	}
}

func TestHTTPClient_TransportLogging(t *testing.T) {
	var buf bytes.Buffer
	testLogger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "pong"})
	}))
	defer func() { ts.Close() }()

	client := New(
		WithBaseURL(ts.URL),
		WithTransportLog(transport.LogOptionLogger(testLogger)),
	)

	var res map[string]string
	err := client.GetJSON(context.Background(), "/ping", &res)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	loggedOutput := buf.String()
	if !strings.Contains(loggedOutput, "HTTP Request") {
		t.Errorf("expected 'HTTP Request' in log output, got: %s", loggedOutput)
	}
	if !strings.Contains(loggedOutput, "/ping") {
		t.Errorf("expected '/ping' path in log output, got: %s", loggedOutput)
	}
	if !strings.Contains(loggedOutput, "GET") {
		t.Errorf("expected 'GET' method in log output, got: %s", loggedOutput)
	}
	if !strings.Contains(loggedOutput, "status\":200") && !strings.Contains(loggedOutput, "\"status\": 200") {
		t.Errorf("expected status 200 in log output, got: %s", loggedOutput)
	}
}

func TestHTTPClient_SensitiveHeaderRedaction(t *testing.T) {
	var buf bytes.Buffer
	testLogger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"auth":"verified"}`))
	}))
	defer func() { ts.Close() }()

	client := New(
		WithBaseURL(ts.URL),
		WithTransportLog(transport.LogOptionLogger(testLogger)),
	)

	headers := map[string]string{
		"Authorization": "Bearer super-secret-key-12345",
	}

	resp, err := client.Get(context.Background(), "/secure", headers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.IsSuccess() {
		t.Fatalf("expected success response")
	}

	loggedOutput := buf.String()
	if strings.Contains(loggedOutput, "super-secret-key-12345") {
		t.Errorf("sensitive authorization header was not redacted! Output: %s", loggedOutput)
	}
	if !strings.Contains(loggedOutput, "[REDACTED]") {
		t.Errorf("expected [REDACTED] in logged headers, got: %s", loggedOutput)
	}
}

func TestHTTPClient_WithoutTransportLog(t *testing.T) {
	var buf bytes.Buffer
	testLogger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer func() { ts.Close() }()

	client := New(
		WithBaseURL(ts.URL),
		WithTransportLog(transport.LogOptionLogger(testLogger)),
		WithoutTransportLog(),
	)

	resp, err := client.Get(context.Background(), "/no-log")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.IsSuccess() {
		t.Fatalf("expected success")
	}

	if buf.Len() > 0 {
		t.Errorf("expected no log output when WithoutTransportLog is used, got: %s", buf.String())
	}
}

func TestHTTPClient_TraceContextPropagation(t *testing.T) {
	var capturedTraceparent string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedTraceparent = r.Header.Get("traceparent")
		w.WriteHeader(http.StatusOK)
	}))
	defer func() { ts.Close() }()

	traceID, _ := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	spanID, _ := trace.SpanIDFromHex("00f067aa0ba902b7")
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(t.Context(), sc)

	client := New(WithBaseURL(ts.URL))
	_, err := client.Get(ctx, "/trace-test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(capturedTraceparent, "4bf92f3577b34da6a3ce929d0e0e4736") {
		t.Errorf("expected traceparent to contain trace ID, got: %q", capturedTraceparent)
	}
	if !strings.Contains(capturedTraceparent, "00f067aa0ba902b7") {
		t.Errorf("expected traceparent to contain span ID, got: %q", capturedTraceparent)
	}
}

func TestHTTPClient_WithLogEngine(t *testing.T) {
	var buf bytes.Buffer
	testLogger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	origLogger := logger.Get()
	defer logger.SetDefault(origLogger)
	logger.SetDefault(testLogger)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "auth ok",
			"token":   "secret-response-token",
		})
	}))
	defer func() { ts.Close() }()

	cfg := &httplog.Config{
		Default: httplog.GlobalConfig{
			MaskValue: "[REDACTED]",
		},
		Rules: []httplog.RuleConfig{
			{
				Path:    "/auth/login",
				Methods: []string{"POST"},
				Enable:  true,
				Request: &httplog.PayloadConfig{
					IgnoreFields: []httplog.FieldRule{
						{Field: "password", Strategy: httplog.StrategyRedact},
					},
				},
				Response: &httplog.PayloadConfig{
					IgnoreFields: []httplog.FieldRule{
						{Field: "token", Strategy: httplog.StrategyRedact},
					},
				},
			},
		},
	}

	engine, err := httplog.NewEngine(cfg)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	client := New(
		WithBaseURL(ts.URL),
		WithLogEngine(engine),
	)

	var res map[string]string
	err = client.PostJSON(context.Background(), "/auth/login", map[string]string{
		"username": "tester",
		"password": "super-secret-password",
	}, &res)
	if err != nil {
		t.Fatalf("PostJSON failed: %v", err)
	}

	if res["token"] != "secret-response-token" {
		t.Fatalf("client received redacted response token: got %q", res["token"])
	}

	logs := buf.String()
	if !strings.Contains(logs, "Outbound HTTP completed") {
		t.Fatalf("expected 'Outbound HTTP completed' in logs, got: %s", logs)
	}
	if strings.Contains(logs, "super-secret-password") {
		t.Fatalf("logs contain plaintext password! logs: %s", logs)
	}
	if strings.Contains(logs, "secret-response-token") {
		t.Fatalf("logs contain plaintext response token! logs: %s", logs)
	}
	if !strings.Contains(logs, "[REDACTED]") {
		t.Fatalf("expected [REDACTED] in logs, got: %s", logs)
	}
}

func TestHTTPClient_WithLogEngine_NilEngine(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer func() { ts.Close() }()

	client := New(
		WithBaseURL(ts.URL),
		WithLogEngine(nil),
	)

	var res map[string]string
	err := client.GetJSON(context.Background(), "/nil-test", &res)
	if err != nil {
		t.Fatalf("unexpected error with WithLogEngine(nil): %v", err)
	}
	if res["status"] != "ok" {
		t.Fatalf("got status %q, want ok", res["status"])
	}
}

func TestHTTPClient_AutomaticDefaultEngine(t *testing.T) {
	var buf bytes.Buffer
	testLogger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	origLogger := logger.Get()
	defer logger.SetDefault(origLogger)
	logger.SetDefault(testLogger)

	cfg := &httplog.Config{
		Default: httplog.GlobalConfig{
			MaskValue: "[REDACTED]",
		},
		Rules: []httplog.RuleConfig{
			{
				Path:    "/auto/test",
				Methods: []string{"POST"},
				Enable:  true,
				Request: &httplog.PayloadConfig{
					IgnoreFields: []httplog.FieldRule{
						{Field: "secret_param", Strategy: httplog.StrategyRedact},
					},
				},
				Response: &httplog.PayloadConfig{
					IgnoreFields: []httplog.FieldRule{
						{Field: "secret_result", Strategy: httplog.StrategyRedact},
					},
				},
			},
		},
	}

	engine, err := httplog.NewEngine(cfg)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	origDefault := httplog.Default()
	defer httplog.SetDefault(origDefault)
	httplog.SetDefault(engine)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message":       "done",
			"secret_result": "ultra_secret_result_data",
		})
	}))
	defer func() { ts.Close() }()

	// Create client without WithLogEngine; it should automatically use httplog.Default()
	client := New(WithBaseURL(ts.URL))

	var res map[string]string
	err = client.PostJSON(context.Background(), "/auto/test", map[string]string{
		"action":       "test",
		"secret_param": "super_secret_param_data",
	}, &res)
	if err != nil {
		t.Fatalf("PostJSON failed: %v", err)
	}

	if res["secret_result"] != "ultra_secret_result_data" {
		t.Fatalf("client received redacted data: %q", res["secret_result"])
	}

	logs := buf.String()
	if !strings.Contains(logs, "Outbound HTTP completed") {
		t.Fatalf("expected 'Outbound HTTP completed' in logs, got: %s", logs)
	}
	if strings.Contains(logs, "super_secret_param_data") {
		t.Fatalf("logs contain plaintext secret_param: %s", logs)
	}
	if strings.Contains(logs, "ultra_secret_result_data") {
		t.Fatalf("logs contain plaintext secret_result: %s", logs)
	}
	if !strings.Contains(logs, "[REDACTED]") {
		t.Fatalf("expected [REDACTED] in logs, got: %s", logs)
	}
	// Verify that legacy transport log did not run
	if strings.Contains(logs, "HTTP Request") {
		t.Fatalf("legacy transport log should NOT run when httplog engine is active, got: %s", logs)
	}
}

func TestClient_RetryPolicy(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		give     string // method
		giveCode []int  // chuỗi status server trả
		giveKey  bool   // có Idempotency-Key
		want     int    // số request server nhận
		wantErr  bool
	}{
		{name: "GET retries 503 then ok", give: http.MethodGet, giveCode: []int{503, 200}, want: 2},
		{name: "POST no retry on 503", give: http.MethodPost, giveCode: []int{503, 200}, want: 1},
		{name: "POST with key retries", give: http.MethodPost, giveCode: []int{503, 200}, giveKey: true, want: 2},
		{name: "GET 404 no retry", give: http.MethodGet, giveCode: []int{404}, want: 1},
		{name: "429 honours Retry-After 0", give: http.MethodGet, giveCode: []int{429, 200}, want: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var reqCount atomic.Int32

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				idx := int(reqCount.Add(1)) - 1
				code := http.StatusOK
				if idx < len(tt.giveCode) {
					code = tt.giveCode[idx]
				}
				if code == http.StatusTooManyRequests {
					w.Header().Set("Retry-After", "0")
				}
				w.WriteHeader(code)
				_, _ = w.Write([]byte(`{"status":"test"}`))
			}))
			defer ts.Close()

			client := New(
				WithBaseURL(ts.URL),
				WithRetryPolicy(RetryPolicy{
					MaxRetries: 3,
					BaseWait:   time.Millisecond,
					MaxWait:    50 * time.Millisecond,
				}),
			)

			headers := make(map[string]string)
			if tt.giveKey {
				headers["Idempotency-Key"] = "idemp-key-test"
			}

			var body io.Reader
			if tt.give == http.MethodPost {
				body = strings.NewReader(`{"test":true}`)
			}

			_, err := client.Do(context.Background(), tt.give, "/test", body, headers)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Do() error = %v, wantErr %v", err, tt.wantErr)
			}
			if int(reqCount.Load()) != tt.want {
				t.Fatalf("server received %d requests, want %d", reqCount.Load(), tt.want)
			}
		})
	}
}

func TestClient_BodyLimit(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(strings.Repeat("a", 101)))
	}))
	defer ts.Close()

	client := New(
		WithBaseURL(ts.URL),
		WithMaxResponseBytes(100),
	)

	_, err := client.Get(context.Background(), "/overflow")
	if !errors.Is(err, ErrBodyTooLarge) {
		t.Fatalf("expected ErrBodyTooLarge, got %v", err)
	}
}

func TestClient_DoesNotMutateInjectedClient(t *testing.T) {
	t.Parallel()
	hc := &http.Client{}
	if hc.Transport != nil {
		t.Fatalf("precondition failed: hc.Transport should be nil")
	}

	_ = New(WithHTTPClient(hc))
	if hc.Transport != nil {
		t.Fatalf("hc.Transport was mutated by New(WithHTTPClient(hc)): %v", hc.Transport)
	}
}

func TestClient_ConcurrentDo(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ping":"pong"}`))
	}))
	defer ts.Close()

	client := New(
		WithBaseURL(ts.URL),
		WithRetryPolicy(RetryPolicy{
			MaxRetries: 2,
			BaseWait:   time.Millisecond,
		}),
	)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := client.Get(context.Background(), "/ping")
			if err != nil {
				t.Errorf("concurrent Get failed: %v", err)
				return
			}
			if !resp.IsSuccess() {
				t.Errorf("expected success status, got %d", resp.StatusCode)
			}
		}()
	}
	wg.Wait()
}

func TestClient_RetryAfterHeader(t *testing.T) {
	t.Parallel()

	// 1. Seconds format
	h := http.Header{}
	h.Set("Retry-After", "5")
	dur, ok := retryAfter(h, time.Now())
	if !ok || dur != 5*time.Second {
		t.Errorf("expected 5s, got %v (ok=%v)", dur, ok)
	}

	// 2. HTTP-Date in future (HTTP dates are always in UTC/GMT)
	futureTime := time.Now().UTC().Add(10 * time.Second).Truncate(time.Second)
	h.Set("Retry-After", futureTime.Format(http.TimeFormat))
	dur, ok = retryAfter(h, time.Now().UTC())
	if !ok || dur <= 0 || dur > 11*time.Second {
		t.Errorf("expected around 10s, got %v (ok=%v)", dur, ok)
	}

	// 3. Negative / 0 / past
	h.Set("Retry-After", "0")
	dur, ok = retryAfter(h, time.Now())
	if ok || dur != 0 {
		t.Errorf("expected false for 0, got %v (ok=%v)", dur, ok)
	}

	h.Set("Retry-After", "-1")
	dur, ok = retryAfter(h, time.Now())
	if ok || dur != 0 {
		t.Errorf("expected false for -1, got %v (ok=%v)", dur, ok)
	}
}

func TestClient_IsIdempotent(t *testing.T) {
	t.Parallel()

	if !isIdempotent(http.MethodGet, nil) {
		t.Errorf("GET should be idempotent")
	}
	if !isIdempotent(http.MethodPut, nil) {
		t.Errorf("PUT should be idempotent")
	}
	if !isIdempotent(http.MethodDelete, nil) {
		t.Errorf("DELETE should be idempotent")
	}
	if isIdempotent(http.MethodPost, nil) {
		t.Errorf("POST without headers should not be idempotent")
	}

	h := http.Header{}
	h.Set("Idempotency-Key", "key-123")
	if !isIdempotent(http.MethodPost, h) {
		t.Errorf("POST with Idempotency-Key should be idempotent")
	}

	h2 := http.Header{}
	h2.Set("X-Idempotency-Key", "key-456")
	if !isIdempotent(http.MethodPost, h2) {
		t.Errorf("POST with X-Idempotency-Key should be idempotent")
	}

	hWhitespace := http.Header{}
	hWhitespace.Set("Idempotency-Key", "   ")
	if isIdempotent(http.MethodPost, hWhitespace) {
		t.Errorf("POST with whitespace-only Idempotency-Key should NOT be idempotent")
	}
}

func TestClient_RetryAfter_OverflowProtection(t *testing.T) {
	t.Parallel()

	h := http.Header{}
	h.Set("Retry-After", "99999999999") // 100 billion seconds (would overflow int64 nanoseconds if unchecked)
	dur, ok := retryAfter(h, time.Now())
	if !ok {
		t.Fatalf("expected ok=true for huge seconds")
	}
	if dur <= 0 {
		t.Fatalf("expected positive duration, got %v (integer overflow detected!)", dur)
	}

	client := New(
		WithRetryPolicy(RetryPolicy{
			MaxRetryAfter: 10 * time.Second,
		}),
	)
	resp := &http.Response{Header: h}
	backoff := client.calculateBackoff(0, resp)
	if backoff != 10*time.Second {
		t.Fatalf("expected calculateBackoff to cap huge Retry-After at MaxRetryAfter (10s), got %v", backoff)
	}
}

func TestClient_CalculateBackoff_Attempts(t *testing.T) {
	t.Parallel()

	client := New(
		WithRetryPolicy(RetryPolicy{
			BaseWait: 100 * time.Millisecond,
			MaxWait:  1 * time.Second,
		}),
	)

	for _, att := range []int{-1, 0, 1, 5, 30, 50, 100} {
		dur := client.calculateBackoff(att, nil)
		if dur < 0 || dur > 1*time.Second {
			t.Errorf("attempt %d: expected backoff in [0, 1s], got %v", att, dur)
		}
	}
}
