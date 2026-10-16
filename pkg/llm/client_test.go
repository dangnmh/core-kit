package llm_test

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/dangnmh/core-kit/pkg/llm"
	"github.com/dangnmh/core-kit/pkg/ptr"
	"github.com/sashabaranov/go-openai"
)

func TestNewClient_Temperature(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		give *float32
		want float32
	}{
		{name: "nil defaults to 0.7", give: nil, want: 0.7},
		{name: "explicit zero kept", give: ptr.To[float32](0), want: 0},
		{name: "explicit 1.2 kept", give: ptr.To[float32](1.2), want: 1.2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := llm.NewClient(llm.Config{Temperature: tt.give})
			got := *c.Config().Temperature
			if got != tt.want {
				t.Fatalf("expected temperature %v, got %v", tt.want, got)
			}
		})
	}
}

func TestChat_TemperatureZeroSent(t *testing.T) {
	t.Parallel()

	var receivedTemp float64
	var hasTempKey bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var rawMap map[string]any
		if err := json.Unmarshal(body, &rawMap); err == nil {
			if v, ok := rawMap["temperature"]; ok {
				hasTempKey = true
				if f, ok := v.(float64); ok {
					receivedTemp = f
				}
			}
		}

		resp := openai.ChatCompletionResponse{
			Choices: []openai.ChatCompletionChoice{
				{Message: openai.ChatCompletionMessage{Role: "assistant", Content: "pong"}},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := llm.NewClient(llm.Config{
		BaseURL:     srv.URL,
		Temperature: ptr.To[float32](0),
	})

	_, err := c.Chat(context.Background(), []openai.ChatCompletionMessage{
		{Role: "user", Content: "ping"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !hasTempKey {
		t.Fatalf("expected 'temperature' key in request payload, but it was omitted")
	}
	if float32(receivedTemp) != math.SmallestNonzeroFloat32 {
		t.Fatalf("expected SmallestNonzeroFloat32 (%v), got %v", math.SmallestNonzeroFloat32, receivedTemp)
	}
}

func TestChat_RetriesTransientOnly(t *testing.T) {
	t.Parallel()

	t.Run("retry 503 then succeed", func(t *testing.T) {
		t.Parallel()
		var calls atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			count := calls.Add(1)
			if count == 1 {
				http.Error(w, `{"error":{"message":"service unavailable","type":"server_error"}}`, http.StatusServiceUnavailable)
				return
			}
			resp := openai.ChatCompletionResponse{
				Choices: []openai.ChatCompletionChoice{
					{Message: openai.ChatCompletionMessage{Role: "assistant", Content: "recovered"}},
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		}))
		defer srv.Close()

		c := llm.NewClient(llm.Config{
			BaseURL:    srv.URL,
			MaxRetries: 2,
		})

		content, err := c.Chat(context.Background(), []openai.ChatCompletionMessage{
			{Role: "user", Content: "hi"},
		})
		if err != nil {
			t.Fatalf("expected success on second attempt, got: %v", err)
		}
		if content != "recovered" {
			t.Fatalf("expected 'recovered', got %q", content)
		}
		if calls.Load() != 2 {
			t.Fatalf("expected 2 calls, got %d", calls.Load())
		}
	})

	t.Run("fail immediately on 400 without retry", func(t *testing.T) {
		t.Parallel()
		var calls atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			http.Error(w, `{"error":{"message":"bad request","type":"invalid_request_error"}}`, http.StatusBadRequest)
		}))
		defer srv.Close()

		c := llm.NewClient(llm.Config{
			BaseURL:    srv.URL,
			MaxRetries: 2,
		})

		_, err := c.Chat(context.Background(), []openai.ChatCompletionMessage{
			{Role: "user", Content: "hi"},
		})
		if err == nil {
			t.Fatalf("expected error on 400, got nil")
		}
		if calls.Load() != 1 {
			t.Fatalf("expected exactly 1 call (no retry on 400), got %d", calls.Load())
		}
	})

	t.Run("canceled context does not retry", func(t *testing.T) {
		t.Parallel()
		var calls atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			http.Error(w, `{"error":{"message":"overloaded","type":"server_error"}}`, http.StatusServiceUnavailable)
		}))
		defer srv.Close()

		c := llm.NewClient(llm.Config{
			BaseURL:    srv.URL,
			MaxRetries: 3,
		})

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := c.Chat(ctx, []openai.ChatCompletionMessage{
			{Role: "user", Content: "hi"},
		})
		if err == nil {
			t.Fatalf("expected error on canceled context")
		}
		if calls.Load() > 1 {
			t.Fatalf("expected at most 1 call on canceled context, got %d", calls.Load())
		}
	})
}
