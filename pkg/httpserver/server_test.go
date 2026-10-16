package httpserver

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dangnmh/core-kit/pkg/httplog"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type MockRoutes struct{}

func (m *MockRoutes) RegisterRoutes(r *gin.Engine) {
	r.GET("/api/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}

type mockChecker struct {
	name string
	err  error
}

func (m *mockChecker) Name() string {
	return m.name
}

func (m *mockChecker) Ready(ctx context.Context) error {
	return m.err
}

func TestNewEngine(t *testing.T) {
	t.Parallel()

	engine := NewEngine(EngineParams{})
	if engine == nil {
		t.Fatal("expected non-nil engine")
	}

	engine.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	req, _ := http.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "pong" {
		t.Fatalf("expected body 'pong', got '%s'", w.Body.String())
	}
}

func TestNewEngine_WithLogEngine(t *testing.T) {
	t.Parallel()

	logEng, err := httplog.NewEngine(&httplog.Config{
		Default: httplog.GlobalConfig{
			MaskValue: "[REDACTED]",
		},
	})
	if err != nil {
		t.Fatalf("failed to create log engine: %v", err)
	}

	engine := NewEngine(EngineParams{
		LogEngine: logEng,
	})
	if engine == nil {
		t.Fatal("expected non-nil engine")
	}

	engine.POST("/test-log", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("POST", "/test-log", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}

func TestFxHTTPServerLifecycle(t *testing.T) {
	t.Parallel()

	var engine *gin.Engine
	app := fxtest.New(
		t,
		fx.Supply(Config{Port: 0}),
		fx.Provide(
			AsRoute(func() *MockRoutes {
				return &MockRoutes{}
			}),
		),
		Module,
		fx.Populate(&engine),
	)
	app.RequireStart()

	req, _ := http.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	app.RequireStop()
}

func TestFxHTTPServerLifecycle_WithLogEngine(t *testing.T) {
	t.Parallel()

	logEng, err := httplog.NewEngine(&httplog.Config{
		Default: httplog.GlobalConfig{
			MaskValue: "[REDACTED]",
		},
	})
	if err != nil {
		t.Fatalf("failed to create log engine: %v", err)
	}

	var engine *gin.Engine
	app := fxtest.New(
		t,
		fx.Supply(Config{Port: 0}),
		fx.Provide(
			func() httplog.Engine {
				return logEng
			},
			AsRoute(func() *MockRoutes {
				return &MockRoutes{}
			}),
		),
		Module,
		fx.Populate(&engine),
	)
	app.RequireStart()

	req, _ := http.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	app.RequireStop()
}

func TestNewHTTPServer_BindFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    func(t *testing.T) int
		wantErr bool
	}{
		{
			name: "free port",
			give: func(t *testing.T) int {
				return 0
			},
			wantErr: false,
		},
		{
			name: "port already bound",
			give: func(t *testing.T) int {
				ln, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					t.Fatalf("failed to listen on test port: %v", err)
				}
				t.Cleanup(func() { _ = ln.Close() })
				return ln.Addr().(*net.TCPAddr).Port
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			port := tt.give(t)
			app := fxtest.New(
				t,
				fx.Supply(Config{Port: port}),
				Module,
			)

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			err := app.Start(ctx)
			if (err != nil) != tt.wantErr {
				t.Fatalf("app.Start err = %v, wantErr = %v", err, tt.wantErr)
			}
			if err == nil {
				_ = app.Stop(ctx)
			}
		})
	}
}

func TestReadyz_States(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		ready      bool
		checkers   []ReadinessChecker
		wantStatus int
		wantBody   string
	}{
		{
			name:       "not ready before start",
			ready:      false,
			checkers:   nil,
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   `"status":"not_ready"`,
		},
		{
			name:       "ready with no checkers",
			ready:      true,
			checkers:   nil,
			wantStatus: http.StatusOK,
			wantBody:   `"status":"ok"`,
		},
		{
			name:  "ready with passing checker",
			ready: true,
			checkers: []ReadinessChecker{
				&mockChecker{name: "db", err: nil},
			},
			wantStatus: http.StatusOK,
			wantBody:   `"status":"ok"`,
		},
		{
			name:  "ready with failing checker",
			ready: true,
			checkers: []ReadinessChecker{
				&mockChecker{name: "db", err: errors.New("connection refused")},
			},
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   `"failed":["db"]`,
		},
		{
			name:  "ready with multiple failing checkers sorted",
			ready: true,
			checkers: []ReadinessChecker{
				&mockChecker{name: "redis", err: errors.New("timeout")},
				&mockChecker{name: "db", err: errors.New("refused")},
				&mockChecker{name: "broker", err: errors.New("offline")},
			},
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   `"failed":["broker","db","redis"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gin.SetMode(gin.TestMode)
			engine := gin.New()

			ready := &atomic.Bool{}
			ready.Store(tt.ready)
			registerHealthRoutes(engine, tt.checkers, ready)

			// 1. Test /healthz (liveness) always returns 200
			reqH, _ := http.NewRequest("GET", "/healthz", nil)
			wH := httptest.NewRecorder()
			engine.ServeHTTP(wH, reqH)
			if wH.Code != http.StatusOK {
				t.Fatalf("/healthz code = %d, want 200", wH.Code)
			}

			// 2. Test /readyz
			reqR, _ := http.NewRequest("GET", "/readyz", nil)
			wR := httptest.NewRecorder()
			engine.ServeHTTP(wR, reqR)

			if wR.Code != tt.wantStatus {
				t.Fatalf("/readyz code = %d, want %d", wR.Code, tt.wantStatus)
			}
			if !contains(wR.Body.String(), tt.wantBody) {
				t.Fatalf("/readyz body = %s, want containing %s", wR.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestCORS_NoCredentialsWithWildcard(t *testing.T) {
	t.Parallel()

	// 1. Wildcard CORS (empty AllowedOrigins)
	engineWildcard := NewEngine(EngineParams{
		Config: Config{AllowedOrigins: nil},
	})
	engineWildcard.GET("/data", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("OPTIONS", "/data", nil)
	req.Header.Set("Origin", "http://evil.example")
	req.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()
	engineWildcard.ServeHTTP(w, req)

	credHeader := w.Header().Get("Access-Control-Allow-Credentials")
	if credHeader == "true" {
		t.Fatalf("expected Access-Control-Allow-Credentials != 'true' when wildcard origins used, got '%s'", credHeader)
	}

	// 1b. Explicit wildcard string ["*"] in AllowedOrigins
	engineExplicitWildcard := NewEngine(EngineParams{
		Config: Config{AllowedOrigins: []string{"*"}},
	})
	engineExplicitWildcard.GET("/data", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	reqStar, _ := http.NewRequest("OPTIONS", "/data", nil)
	reqStar.Header.Set("Origin", "http://evil.example")
	reqStar.Header.Set("Access-Control-Request-Method", "GET")
	wStar := httptest.NewRecorder()
	engineExplicitWildcard.ServeHTTP(wStar, reqStar)

	if wStar.Header().Get("Access-Control-Allow-Credentials") == "true" {
		t.Fatalf("expected Access-Control-Allow-Credentials != 'true' when ['*'] used")
	}

	// 2. Restricted CORS (explicit AllowedOrigins)
	engineRestricted := NewEngine(EngineParams{
		Config: Config{AllowedOrigins: []string{"http://trusted.example"}},
	})
	engineRestricted.GET("/data", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Preflight from untrusted origin
	reqEvil, _ := http.NewRequest("OPTIONS", "/data", nil)
	reqEvil.Header.Set("Origin", "http://evil.example")
	reqEvil.Header.Set("Access-Control-Request-Method", "GET")
	wEvil := httptest.NewRecorder()
	engineRestricted.ServeHTTP(wEvil, reqEvil)

	if wEvil.Header().Get("Access-Control-Allow-Origin") == "http://evil.example" {
		t.Fatalf("expected Access-Control-Allow-Origin not to reflect untrusted origin")
	}

	// Preflight from trusted origin
	reqTrusted, _ := http.NewRequest("OPTIONS", "/data", nil)
	reqTrusted.Header.Set("Origin", "http://trusted.example")
	reqTrusted.Header.Set("Access-Control-Request-Method", "GET")
	wTrusted := httptest.NewRecorder()
	engineRestricted.ServeHTTP(wTrusted, reqTrusted)

	if wTrusted.Header().Get("Access-Control-Allow-Origin") != "http://trusted.example" {
		t.Fatalf("expected Access-Control-Allow-Origin 'http://trusted.example', got '%s'", wTrusted.Header().Get("Access-Control-Allow-Origin"))
	}
	if wTrusted.Header().Get("Access-Control-Allow-Credentials") == "true" {
		t.Fatalf("expected Access-Control-Allow-Credentials != 'true'")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
