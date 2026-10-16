package httpserver

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// BearerTokenAuth returns a Gin middleware that enforces Bearer token authentication.
// If token is empty or whitespace-only, it panics immediately at startup (CONST-SEC-01).
func BearerTokenAuth(token string) gin.HandlerFunc {
	trimmedToken := strings.TrimSpace(token)
	if trimmedToken == "" {
		panic("FATAL: httpserver.BearerTokenAuth initialized with empty or whitespace-only token; aborting startup")
	}
	expected := []byte(trimmedToken)

	return func(c *gin.Context) {
		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		const prefix = "bearer "
		if len(authHeader) < len(prefix) || !strings.EqualFold(authHeader[:len(prefix)], prefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "unauthorized",
			})
			return
		}
		provided := []byte(strings.TrimSpace(authHeader[len(prefix):]))
		if subtle.ConstantTimeCompare(provided, expected) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "unauthorized",
			})
			return
		}
		c.Next()
	}
}
