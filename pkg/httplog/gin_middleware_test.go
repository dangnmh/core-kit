package httplog_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/dangnmh/core-kit/pkg/httplog"
	"github.com/dangnmh/core-kit/pkg/logger"
	"github.com/gin-gonic/gin"
)

type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *safeBuffer) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *safeBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

func setupTestLogger(buf *safeBuffer) {
	l := logger.NewLogger("test-httplog", slog.LevelDebug, buf)
	logger.SetDefault(l)
}

func TestGinMiddleware_SanitizesRequestAndResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	logBuf := &safeBuffer{}
	setupTestLogger(logBuf)

	cfg := &httplog.Config{
		Default: httplog.GlobalConfig{
			MaskValue: "[REDACTED]",
		},
		Rules: []httplog.RuleConfig{
			{
				Path:    "/api/v1/auth",
				Methods: []string{"POST"},
				Enable:  true,
				Request: &httplog.PayloadConfig{
					IgnoreFields: []httplog.FieldRule{
						{Field: "password", Strategy: httplog.StrategyRedact},
						{Field: "credit_card", Strategy: httplog.StrategyPartialMask},
					},
				},
				Response: &httplog.PayloadConfig{
					IgnoreFields: []httplog.FieldRule{
						{Field: "access_token", Strategy: httplog.StrategyRedact},
						{Field: "refresh_token", Strategy: httplog.StrategyRemove},
					},
				},
			},
		},
	}

	engine, err := httplog.NewEngine(cfg)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	router := gin.New()
	router.Use(httplog.GinMiddleware(engine))

	type AuthRequest struct {
		Username   string `json:"username"`
		Password   string `json:"password"`
		CreditCard string `json:"credit_card"`
	}

	var downstreamReceived AuthRequest
	router.POST("/api/v1/auth", func(c *gin.Context) {
		if err := c.ShouldBindJSON(&downstreamReceived); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status":        "ok",
			"access_token":  "secret-jwt-token-123",
			"refresh_token": "secret-refresh-456",
			"user_id":       "user-999",
		})
	})

	reqPayload := `{"username":"alice","password":"super_secret_password","credit_card":"1234567812345678"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth", bytes.NewBufferString(reqPayload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// 1. Verify downstream received untouched original request data
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body: %s", w.Code, w.Body.String())
	}
	if downstreamReceived.Password != "super_secret_password" {
		t.Fatalf("downstream received corrupted password: got %q, want %q", downstreamReceived.Password, "super_secret_password")
	}
	if downstreamReceived.CreditCard != "1234567812345678" {
		t.Fatalf("downstream received corrupted credit card: got %q, want %q", downstreamReceived.CreditCard, "1234567812345678")
	}

	// 2. Verify client response is untouched
	var clientResp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &clientResp); err != nil {
		t.Fatalf("failed to parse client response: %v", err)
	}
	if clientResp["access_token"] != "secret-jwt-token-123" {
		t.Fatalf("client received redacted access_token: got %v", clientResp["access_token"])
	}
	if clientResp["refresh_token"] != "secret-refresh-456" {
		t.Fatalf("client received redacted refresh_token: got %v", clientResp["refresh_token"])
	}

	// 3. Verify logs contain sanitized request and response
	logs := logBuf.String()
	if !strings.Contains(logs, "HTTP Request") {
		t.Fatalf("expected 'HTTP Request' log, got: %s", logs)
	}
	// Password must NOT be present in plaintext in logs
	if strings.Contains(logs, "super_secret_password") {
		t.Fatalf("logs contain plaintext password! logs: %s", logs)
	}
	// Redacted password must appear in logs
	if !strings.Contains(logs, "[REDACTED]") {
		t.Fatalf("expected [REDACTED] in logs, got: %s", logs)
	}
	// Access token must NOT be present in plaintext in logs
	if strings.Contains(logs, "secret-jwt-token-123") {
		t.Fatalf("logs contain plaintext access_token! logs: %s", logs)
	}
	// Refresh token was marked with 'remove' strategy, so it should not appear
	if strings.Contains(logs, "secret-refresh-456") {
		t.Fatalf("logs contain removed refresh_token! logs: %s", logs)
	}
	// Credit card should be partially masked
	if strings.Contains(logs, "1234567812345678") {
		t.Fatalf("logs contain plaintext credit card! logs: %s", logs)
	}
	if !strings.Contains(logs, "1234********5678") {
		t.Fatalf("expected partially masked credit card '1234********5678' in logs, got: %s", logs)
	}
}

func TestGinMiddleware_PreservesRawBodyForMultipleDownstreamReads(t *testing.T) {
	gin.SetMode(gin.TestMode)

	logBuf := &safeBuffer{}
	setupTestLogger(logBuf)

	cfg := &httplog.Config{
		Default: httplog.GlobalConfig{MaskValue: "[REDACTED]"},
		Rules: []httplog.RuleConfig{
			{
				Path:    "/read-test",
				Methods: []string{"POST"},
				Request: &httplog.PayloadConfig{
					IgnoreFields: []httplog.FieldRule{
						{Field: "secret", Strategy: httplog.StrategyRedact},
					},
				},
			},
		},
	}

	engine, err := httplog.NewEngine(cfg)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	router := gin.New()
	router.Use(httplog.GinMiddleware(engine))

	var rawRead1 string
	router.POST("/read-test", func(c *gin.Context) {
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
		rawRead1 = string(bodyBytes)
		c.String(http.StatusOK, "read success")
	})

	origBody := `{"name":"item","secret":"sensitive_value"}`
	req := httptest.NewRequest(http.MethodPost, "/read-test", bytes.NewBufferString(origBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if rawRead1 != origBody {
		t.Fatalf("downstream read mismatch: got %q, want %q", rawRead1, origBody)
	}

	// Verify log was redacted
	logs := logBuf.String()
	if strings.Contains(logs, "sensitive_value") {
		t.Fatalf("log contains sensitive_value! logs: %s", logs)
	}
}

func TestGinMiddleware_ConditionalResponseLogging_ErrorOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)

	logBuf := &safeBuffer{}
	setupTestLogger(logBuf)

	cfg := &httplog.Config{
		Default: httplog.GlobalConfig{
			MaskValue:     "[REDACTED]",
			LogResponseOn: "error_only",
		},
		Rules: []httplog.RuleConfig{
			{
				Path:    "/error-only-route",
				Methods: []string{"GET"},
				Enable:  true,
			},
		},
	}

	engine, err := httplog.NewEngine(cfg)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	router := gin.New()
	router.Use(httplog.GinMiddleware(engine))

	router.GET("/error-only-route", func(c *gin.Context) {
		statusStr := c.Query("status")
		if statusStr == "500" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal crash detail"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": "success payload that should not be in response log"})
	})

	// Case 1: 200 OK -> response body must NOT be logged
	req1 := httptest.NewRequest(http.MethodGet, "/error-only-route?status=200", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w1.Code)
	}
	logs1 := logBuf.String()
	if strings.Contains(logs1, "success payload that should not be in response log") {
		t.Fatalf("200 OK response body should NOT be logged under error_only! logs: %s", logs1)
	}

	// Case 2: 500 InternalServerError -> response body MUST be logged
	req2 := httptest.NewRequest(http.MethodGet, "/error-only-route?status=500", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w2.Code)
	}
	logs2 := logBuf.String()
	if !strings.Contains(logs2, "internal crash detail") {
		t.Fatalf("500 response body MUST be logged under error_only! logs: %s", logs2)
	}
}

func TestGinMiddleware_NilEngineSafeFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)

	logBuf := &safeBuffer{}
	setupTestLogger(logBuf)

	router := gin.New()
	router.Use(httplog.GinMiddleware(nil)) // nil engine must not panic

	var handlerExecuted bool
	router.POST("/nil-test", func(c *gin.Context) {
		handlerExecuted = true
		c.String(http.StatusOK, "fallback ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/nil-test", bytes.NewBufferString(`{"ping":"pong"}`))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if !handlerExecuted {
		t.Fatalf("downstream handler was not executed")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "fallback ok" {
		t.Fatalf("expected body 'fallback ok', got %q", w.Body.String())
	}

	logs := logBuf.String()
	if !strings.Contains(logs, "HTTP Request") {
		t.Fatalf("expected logging fallback for nil engine, got: %s", logs)
	}
}

func TestGinMiddleware_NoMatchingRuleFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)

	logBuf := &safeBuffer{}
	setupTestLogger(logBuf)

	cfg := &httplog.Config{
		Default: httplog.GlobalConfig{
			MaskValue:     "[DEFAULT_MASK]",
			LogResponseOn: "all",
			LogUnmatched:  true,
		},
		Rules: []httplog.RuleConfig{
			{
				Path:    "/specific/route",
				Methods: []string{"GET"},
			},
		},
	}

	engine, err := httplog.NewEngine(cfg)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	router := gin.New()
	router.Use(httplog.GinMiddleware(engine))

	router.POST("/unmatched/path", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "hello unmatched"})
	})

	req := httptest.NewRequest(http.MethodPost, "/unmatched/path", bytes.NewBufferString(`{"unmatched":"data"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	logs := logBuf.String()
	if !strings.Contains(logs, "hello unmatched") {
		t.Fatalf("expected response logged for unmatched route under log_response_on: all, got: %s", logs)
	}
}

func TestGinMiddleware_FormUrlEncoded(t *testing.T) {
	gin.SetMode(gin.TestMode)

	logBuf := &safeBuffer{}
	setupTestLogger(logBuf)

	cfg := &httplog.Config{
		Default: httplog.GlobalConfig{MaskValue: "[REDACTED]"},
		Rules: []httplog.RuleConfig{
			{
				Path:    "/form-login",
				Methods: []string{"POST"},
				Enable:  true,
				Request: &httplog.PayloadConfig{
					IgnoreFields: []httplog.FieldRule{
						{Field: "secret_pin", Strategy: httplog.StrategyRedact},
					},
				},
			},
		},
	}

	engine, err := httplog.NewEngine(cfg)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	router := gin.New()
	router.Use(httplog.GinMiddleware(engine))

	var formPin string
	router.POST("/form-login", func(c *gin.Context) {
		formPin = c.PostForm("secret_pin")
		c.String(http.StatusOK, "form processed")
	})

	formData := "user=admin&secret_pin=9876"
	req := httptest.NewRequest(http.MethodPost, "/form-login", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if formPin != "9876" {
		t.Fatalf("downstream received corrupted form field: got %q, want %q", formPin, "9876")
	}

	logs := logBuf.String()
	if strings.Contains(logs, "secret_pin=9876") {
		t.Fatalf("log contains plaintext secret_pin! logs: %s", logs)
	}
	if !strings.Contains(logs, "secret_pin=%5BREDACTED%5D") && !strings.Contains(logs, "secret_pin=[REDACTED]") {
		t.Fatalf("expected redacted secret_pin in logs, got: %s", logs)
	}
}

func TestGinMiddleware_MultipartAndBinary(t *testing.T) {
	gin.SetMode(gin.TestMode)

	logBuf := &safeBuffer{}
	setupTestLogger(logBuf)

	cfg := &httplog.Config{
		Default: httplog.GlobalConfig{
			MaskValue:    "[REDACTED]",
			LogUnmatched: true,
		},
	}

	engine, err := httplog.NewEngine(cfg)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	router := gin.New()
	router.Use(httplog.GinMiddleware(engine))

	router.POST("/upload", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/octet-stream", []byte{0xDE, 0xAD, 0xBE, 0xEF})
	})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormField("file")
	_, _ = part.Write([]byte("dummy content"))
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	logs := logBuf.String()
	if !strings.Contains(logs, "[MULTIPART_FORM_DATA]") {
		t.Fatalf("expected [MULTIPART_FORM_DATA] in req_body logs, got: %s", logs)
	}
	if !strings.Contains(logs, "[BINARY application/octet-stream, size: 4 bytes]") {
		t.Fatalf("expected binary summary in res_body logs, got: %s", logs)
	}
}

func TestGinMiddleware_DisabledInboundLogging(t *testing.T) {
	gin.SetMode(gin.TestMode)

	logBuf := &safeBuffer{}
	setupTestLogger(logBuf)

	inboundDisabled := false
	cfg := &httplog.Config{
		Default: httplog.GlobalConfig{
			LogInbound: &inboundDisabled,
		},
	}

	engine, err := httplog.NewEngine(cfg)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	if engine.LogInboundEnabled() {
		t.Fatalf("expected LogInboundEnabled to be false")
	}

	router := gin.New()
	router.Use(httplog.GinMiddleware(engine))

	var handlerExecuted bool
	router.POST("/disabled-inbound", func(c *gin.Context) {
		handlerExecuted = true
		c.String(http.StatusOK, "inbound success")
	})

	req := httptest.NewRequest(http.MethodPost, "/disabled-inbound", bytes.NewBufferString(`{"field":"value"}`))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if !handlerExecuted {
		t.Fatalf("handler was not executed")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "inbound success" {
		t.Fatalf("expected response 'inbound success', got %q", w.Body.String())
	}

	logs := logBuf.String()
	if strings.Contains(logs, "HTTP Request") || strings.Contains(logs, "disabled-inbound") {
		t.Fatalf("expected NO inbound logs when LogInbound is disabled, got: %s", logs)
	}
}

func TestGinMiddleware_SkipWhenDisabledOrUnmatched(t *testing.T) {
	gin.SetMode(gin.TestMode)

	logBuf := &safeBuffer{}
	setupTestLogger(logBuf)

	inboundTrue := true
	cfg := &httplog.Config{
		Default: httplog.GlobalConfig{
			MaskValue:    "[REDACTED]",
			LogInbound:   &inboundTrue,
			LogUnmatched: false, // Default is false
		},
		Rules: []httplog.RuleConfig{
			{
				Path:    "/disabled-route",
				Methods: []string{"GET"},
				Enable:  false,
			},
			{
				Path:    "/enabled-route",
				Methods: []string{"GET"},
				Enable:  true,
			},
		},
	}

	engine, err := httplog.NewEngine(cfg)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	router := gin.New()
	router.Use(httplog.GinMiddleware(engine))

	router.GET("/disabled-route", func(c *gin.Context) {
		c.String(http.StatusOK, "ok disabled")
	})
	router.GET("/unmatched-route", func(c *gin.Context) {
		c.String(http.StatusOK, "ok unmatched")
	})
	router.GET("/enabled-route", func(c *gin.Context) {
		c.String(http.StatusOK, "ok enabled")
	})

	// 1. Disabled route -> no logs
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/disabled-route", nil)
	router.ServeHTTP(w1, req1)
	if strings.Contains(logBuf.String(), "/disabled-route") {
		t.Fatalf("expected NO logs for disabled route, got: %s", logBuf.String())
	}

	// 2. Unmatched route -> no logs
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/unmatched-route", nil)
	router.ServeHTTP(w2, req2)
	if strings.Contains(logBuf.String(), "/unmatched-route") {
		t.Fatalf("expected NO logs for unmatched route, got: %s", logBuf.String())
	}

	// 3. Enabled route -> should log
	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodGet, "/enabled-route", nil)
	router.ServeHTTP(w3, req3)
	if !strings.Contains(logBuf.String(), "/enabled-route") {
		t.Fatalf("expected logs for enabled route, got: %s", logBuf.String())
	}
}
