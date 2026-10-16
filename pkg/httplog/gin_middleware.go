package httplog

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/dangnmh/core-kit/pkg/logger"
	"github.com/gin-gonic/gin"
)

// bodyLogWriter intercepts and buffers bytes written through Gin's ResponseWriter.
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

// Write intercepts writes to the underlying response writer while buffering a copy for logging.
func (w *bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// WriteString intercepts string writes to the underlying response writer while buffering a copy for logging.
func (w *bodyLogWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

// GinMiddleware creates a Gin middleware that sanitizes request and response bodies according
// to the compiled rules in the provided httplog Engine, and emits structured JSON logs.
func GinMiddleware(engine Engine) gin.HandlerFunc {
	return func(c *gin.Context) {
		if engine != nil && !engine.LogInboundEnabled() {
			c.Next()
			return
		}

		path := c.Request.URL.Path
		method := c.Request.Method

		var rule *CompiledRouteRule
		var reqRules []CompiledFieldRule
		var resRules []CompiledFieldRule

		if engine != nil {
			var matched bool
			rule, matched = engine.MatchInbound(method, path)
			if !matched {
				rule, matched = engine.Match(method, path)
			}
			if matched && rule != nil {
				if !rule.Enabled {
					c.Next()
					return
				}
				reqRules = rule.ReqRules
				resRules = rule.ResRules
			} else {
				if !engine.LogUnmatchedEnabled() {
					c.Next()
					return
				}
			}
		}

		// Read and preserve request body so downstream handlers can read it.
		var reqBodyBytes []byte
		if c.Request.Body != nil && c.Request.Body != http.NoBody {
			var err error
			reqBodyBytes, err = io.ReadAll(c.Request.Body)
			if err == nil {
				c.Request.Body = io.NopCloser(bytes.NewBuffer(reqBodyBytes))
			}
		}

		// Wrap Gin response writer to capture output body bytes.
		blw := &bodyLogWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBuffer(nil),
		}
		c.Writer = blw

		start := time.Now()
		c.Next()
		latency := time.Since(start)

		// Sanitize request body.
		reqContentType := c.ContentType()
		if reqContentType == "" {
			reqContentType = c.Request.Header.Get("Content-Type")
		}

		var reqBodySanitized string
		if engine != nil {
			reqBodySanitized = engine.SanitizeBody(reqContentType, reqBodyBytes, reqRules)
		} else {
			reqBodySanitized = string(reqBodyBytes)
		}

		// Evaluate response logging condition and sanitize response body.
		status := c.Writer.Status()
		var resBodySanitized string
		shouldLogRes := true
		if engine != nil {
			shouldLogRes = engine.ShouldLogResponse(status, rule)
		}
		if shouldLogRes {
			resContentType := c.Writer.Header().Get("Content-Type")
			if engine != nil {
				resBodySanitized = engine.SanitizeBody(resContentType, blw.body.Bytes(), resRules)
			} else {
				resBodySanitized = blw.body.String()
			}
		}

		logger.InfoContext(c.Request.Context(), "HTTP Request",
			"direction", "inbound",
			"method", method,
			"path", path,
			"status", status,
			"latency_ms", latency.Milliseconds(),
			"client_ip", c.ClientIP(),
			"req_body", reqBodySanitized,
			"res_body", resBodySanitized,
		)
	}
}
