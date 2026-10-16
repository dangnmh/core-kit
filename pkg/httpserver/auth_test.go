package httpserver_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dangnmh/core-kit/pkg/httpserver"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestBearerTokenAuth(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		token    string
		give     string
		wantCode int
	}{
		{name: "valid", token: "s3cret", give: "Bearer s3cret", wantCode: http.StatusOK},
		{name: "valid case insensitive and trimmed", token: " s3cret\n", give: "bearer s3cret ", wantCode: http.StatusOK},
		{name: "wrong", token: "s3cret", give: "Bearer nope", wantCode: http.StatusUnauthorized},
		{name: "missing", token: "s3cret", give: "", wantCode: http.StatusUnauthorized},
		{name: "non-bearer scheme", token: "s3cret", give: "Basic s3cret", wantCode: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := gin.New()
			r.POST("/protected", httpserver.BearerTokenAuth(tt.token), func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"success": true})
			})

			req := httptest.NewRequest(http.MethodPost, "/protected", nil)
			if tt.give != "" {
				req.Header.Set("Authorization", tt.give)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Fatalf("got status %d, want %d (body=%s)", w.Code, tt.wantCode, w.Body.String())
			}
		})
	}
}

func TestBearerTokenAuth_FailClosedPanic(t *testing.T) {
	t.Parallel()

	emptyCases := []struct {
		name  string
		token string
	}{
		{name: "empty string", token: ""},
		{name: "spaces only", token: "   "},
		{name: "tabs and newlines", token: "\t\n  "},
	}

	expectedPanic := "FATAL: httpserver.BearerTokenAuth initialized with empty or whitespace-only token; aborting startup"

	for _, tc := range emptyCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.PanicsWithValue(t, expectedPanic, func() {
				httpserver.BearerTokenAuth(tc.token)
			})
		})
	}
}

func TestMaxBytesMiddleware(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(httpserver.MaxBytesMiddleware(10)) // 10 bytes limit
	r.POST("/upload", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.AbortWithStatus(http.StatusRequestEntityTooLarge)
			return
		}
		c.String(http.StatusOK, string(body))
	})

	// Case 1: Body within limit (5 bytes)
	req1 := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewBufferString("hello"))
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Case 2: Body exceeds limit (20 bytes)
	req2 := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewBufferString("this is too long of a body"))
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusRequestEntityTooLarge, w2.Code)
}
