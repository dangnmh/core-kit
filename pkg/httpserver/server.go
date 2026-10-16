package httpserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dangnmh/core-kit/pkg/httplog"
	"github.com/dangnmh/core-kit/pkg/logger"
	"github.com/dangnmh/core-kit/pkg/tracer"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
)

// Config defines standard HTTP server configuration.
type Config struct {
	Port            int           `yaml:"port"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	IdleTimeout     time.Duration `yaml:"idle_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"` // default 10s
	AllowedOrigins  []string      `yaml:"allowed_origins"`  // empty = "*" WITHOUT credentials
}

// EngineParams holds the dependencies for creating the Gin engine.
type EngineParams struct {
	fx.In

	LogEngine httplog.Engine `optional:"true"`
	Config    Config         `optional:"true"`
}

// RouteRegistrar is an interface that route handlers implement to register their endpoints.
type RouteRegistrar interface {
	RegisterRoutes(router *gin.Engine)
}

// AsRoute is a convenience helper for fx.Provide to group route registrars.
func AsRoute(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(RouteRegistrar)),
		fx.ResultTags(`group:"routes"`),
	)
}

// ReadinessChecker is an interface that components implement to participate in the /readyz health check.
type ReadinessChecker interface {
	Name() string
	Ready(ctx context.Context) error
}

// AsReadinessChecker is a convenience helper for fx.Provide to group readiness checkers.
func AsReadinessChecker(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(ReadinessChecker)),
		fx.ResultTags(`group:"readiness"`),
	)
}

// MaxBytesMiddleware bounds incoming HTTP request bodies via http.MaxBytesReader (CONST-SEC-04).
func MaxBytesMiddleware(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}

// NewEngine creates a production-ready Gin Engine with Recovery, CORS, and Slog request logging.
func NewEngine(p EngineParams) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(MaxBytesMiddleware(10 << 20)) // 10MB limit (CONST-SEC-04)

	corsCfg := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization", "Accept", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type", "X-Word-Boundaries"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}

	hasWildcard := false
	for _, o := range p.Config.AllowedOrigins {
		if o == "*" {
			hasWildcard = true
			break
		}
	}
	if len(p.Config.AllowedOrigins) == 0 || hasWildcard {
		corsCfg.AllowAllOrigins = true
	} else {
		corsCfg.AllowOrigins = p.Config.AllowedOrigins
	}

	r.Use(cors.New(corsCfg))

	if p.LogEngine != nil {
		r.Use(httplog.GinMiddleware(p.LogEngine))
	}

	// OpenTelemetry Trace & Structured logging middleware
	r.Use(func(c *gin.Context) {
		ctx := tracer.ExtractHTTP(c.Request.Context(), c.Request.Header)
		path := c.Request.URL.Path
		spanCtx, span := tracer.StartSpan(ctx, "httpserver", fmt.Sprintf("HTTP %s %s", c.Request.Method, path),
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("http.method", c.Request.Method),
				attribute.String("http.path", path),
				attribute.String("client.ip", c.ClientIP()),
			),
		)
		defer span.End()

		c.Request = c.Request.WithContext(spanCtx)
		start := time.Now()
		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		span.SetAttributes(attribute.Int("http.status_code", status))
		if status >= 500 {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", status))
		}

		logger.DebugContext(spanCtx, "HTTP request completed",
			"method", c.Request.Method,
			"path", path,
			"status", status,
			"latency", latency.String(),
			"ip", c.ClientIP(),
		)
	})

	return r
}

// TraceMiddleware creates an explicit Gin middleware to extract W3C traceparent and start a server span.
func TraceMiddleware(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := tracer.ExtractHTTP(c.Request.Context(), c.Request.Header)
		path := c.Request.URL.Path
		spanCtx, span := tracer.StartSpan(ctx, serviceName, fmt.Sprintf("HTTP %s %s", c.Request.Method, path),
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("http.method", c.Request.Method),
				attribute.String("http.path", path),
				attribute.String("client.ip", c.ClientIP()),
			),
		)
		defer span.End()

		c.Request = c.Request.WithContext(spanCtx)
		c.Next()

		status := c.Writer.Status()
		span.SetAttributes(attribute.Int("http.status_code", status))
		if status >= 500 {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", status))
		}
	}
}

// Params holds the dependencies for starting the HTTP Server.
type Params struct {
	fx.In

	Lifecycle fx.Lifecycle
	Engine    *gin.Engine
	Config    Config             `optional:"true"`
	Routes    []RouteRegistrar   `group:"routes"`
	Readiness []ReadinessChecker `group:"readiness"`
}

// RegisterRoutes iterates over all injected route registrars and attaches them to the engine.
func RegisterRoutes(engine *gin.Engine, routes []RouteRegistrar) {
	for _, r := range routes {
		r.RegisterRoutes(engine)
	}
}

func registerHealthRoutes(engine *gin.Engine, checks []ReadinessChecker, ready *atomic.Bool) {
	hasHealthz := false
	hasReadyz := false

	for _, route := range engine.Routes() {
		if route.Path == "/healthz" {
			hasHealthz = true
		}
		if route.Path == "/readyz" {
			hasReadyz = true
		}
	}

	if !hasHealthz {
		engine.GET("/healthz", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
	}

	if !hasReadyz {
		engine.GET("/readyz", func(c *gin.Context) {
			if !ready.Load() {
				c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready"})
				return
			}

			if len(checks) == 0 {
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
				return
			}

			var (
				mu     sync.Mutex
				failed []string
				wg     sync.WaitGroup
			)

			for _, chk := range checks {
				wg.Add(1)
				go func(rc ReadinessChecker) {
					defer wg.Done()
					chkCtx, cancel := context.WithTimeout(c.Request.Context(), 1*time.Second)
					defer cancel()
					if err := rc.Ready(chkCtx); err != nil {
						mu.Lock()
						failed = append(failed, rc.Name())
						mu.Unlock()
					}
				}(chk)
			}
			wg.Wait()

			if len(failed) > 0 {
				slices.Sort(failed)
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"status": "unhealthy",
					"failed": failed,
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
	}
}

// NewHTTPServer constructs an http.Server and registers start/stop lifecycle hooks with Fx.
func NewHTTPServer(p Params) *http.Server {
	RegisterRoutes(p.Engine, p.Routes)

	ready := &atomic.Bool{}
	registerHealthRoutes(p.Engine, p.Readiness, ready)

	port := p.Config.Port
	var addr string
	if port < 0 {
		addr = fmt.Sprintf(":%d", port)
	} else if port == 0 {
		addr = ":0"
	} else {
		addr = fmt.Sprintf(":%d", port)
	}

	readTimeout := p.Config.ReadTimeout
	if readTimeout <= 0 {
		readTimeout = 10 * time.Second
	}
	writeTimeout := p.Config.WriteTimeout
	if writeTimeout <= 0 {
		writeTimeout = 15 * time.Second
	}
	idleTimeout := p.Config.IdleTimeout
	if idleTimeout <= 0 {
		idleTimeout = 60 * time.Second
	}
	shutdownTimeout := p.Config.ShutdownTimeout
	if shutdownTimeout <= 0 {
		shutdownTimeout = 10 * time.Second
	}

	srv := &http.Server{
		Addr:         addr,
		Handler:      p.Engine,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}

	p.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("HTTP server starting", "addr", srv.Addr)
			ln, err := net.Listen("tcp", srv.Addr)
			if err != nil {
				return fmt.Errorf("bind http listener %s: %w", srv.Addr, err)
			}
			srv.Addr = ln.Addr().String()
			ready.Store(true)
			logger.Info("HTTP server listening", "addr", srv.Addr)

			go func() {
				if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
					logger.Error("HTTP server stopped unexpectedly", "error", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			ready.Store(false)
			logger.Info("HTTP server shutting down gracefully...", "addr", srv.Addr)
			shutdownCtx, cancel := context.WithTimeout(ctx, shutdownTimeout)
			defer cancel()
			return srv.Shutdown(shutdownCtx)
		},
	})

	return srv
}

// Module provides the complete HTTP Server bundle for an Uber Fx application.
var Module = fx.Module("httpserver",
	fx.Provide(
		NewEngine,
		NewHTTPServer,
	),
	fx.Invoke(func(*http.Server) {}),
)
