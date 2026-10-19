package httpserver

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var DefaultCORSAllowHeaders = []string{
	"Origin",
	"Content-Length",
	"Content-Type",
	"Authorization",
	"Accept",
	"X-Requested-With",
	"X-Fingerprint",
	"X-Vibe-Fingerprint",
	"X-Captcha-Token",
	"X-Turnstile-Token",
	HeaderRequestNonce,
	HeaderRequestTimestamp,
}

var DefaultCORSAllowMethods = []string{
	"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS",
}

// SetWildcardCORSHeaders populates standard wildcard CORS headers on an http.Header.
func SetWildcardCORSHeaders(h http.Header, allowHeaders ...string) {
	headers := allowHeaders
	if len(headers) == 0 {
		headers = DefaultCORSAllowHeaders
	}
	h.Set("Access-Control-Allow-Origin", "*")
	h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
	h.Set("Access-Control-Allow-Headers", strings.Join(headers, ", "))
	h.Set("Access-Control-Max-Age", "86400")
}

// GinWildcardCORSMiddleware returns a Gin middleware backed by github.com/gin-contrib/cors
// that also ensures CORS headers are emitted for preflight OPTIONS requests without an Origin header.
func GinWildcardCORSMiddleware(allowHeaders ...string) gin.HandlerFunc {
	headers := allowHeaders
	if len(headers) == 0 {
		headers = DefaultCORSAllowHeaders
	}
	corsHandler := cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     DefaultCORSAllowMethods,
		AllowHeaders:     headers,
		AllowCredentials: false,
		MaxAge:           24 * time.Hour,
	})
	return func(c *gin.Context) {
		SetWildcardCORSHeaders(c.Writer.Header(), headers...)
		if c.Request.Method == http.MethodOptions {
			if c.Request.Header.Get("Origin") != "" {
				corsHandler(c)
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		if c.Request.Header.Get("Origin") != "" {
			corsHandler(c)
			return
		}
		c.Next()
	}
}

// WildcardCORSMiddleware wraps a standard net/http.Handler with wildcard CORS headers and 204 preflight handling.
func WildcardCORSMiddleware(next http.Handler, allowHeaders ...string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		SetWildcardCORSHeaders(w.Header(), allowHeaders...)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
