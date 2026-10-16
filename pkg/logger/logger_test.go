package logger

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestLoggerJSONOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	l := NewLogger("test-service", slog.LevelInfo, buf)

	l.Info("Test message", "key1", "val1")

	var data map[string]any
	if err := json.Unmarshal(buf.Bytes(), &data); err != nil {
		t.Fatalf("Failed to parse JSON log output: %v", err)
	}

	if data["severity"] != "INFO" {
		t.Errorf("Expected severity INFO, got %v", data["severity"])
	}
	if data["message"] != "Test message" {
		t.Errorf("Expected message 'Test message', got %v", data["message"])
	}
	if data["service"] != "test-service" {
		t.Errorf("Expected service 'test-service', got %v", data["service"])
	}
	ctxObj, ok := data["context"].(map[string]any)
	if !ok {
		t.Fatalf("Expected context to be a map, got %v", data["context"])
	}
	if ctxObj["key1"] != "val1" {
		t.Errorf("Expected key1 'val1' in context, got %v", ctxObj["key1"])
	}
	if data["timestamp"] == nil {
		t.Errorf("Expected timestamp to be present")
	}
}

func TestLoggerTraceContext(t *testing.T) {
	buf := &bytes.Buffer{}
	l := NewLogger("test-service", slog.LevelInfo, buf)

	// Create valid span context
	traceID, _ := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	spanID, _ := trace.SpanIDFromHex("00f067aa0ba902b7")
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(t.Context(), sc)

	l.InfoContext(ctx, "Message with trace context", "action", "mine_block")

	var data map[string]any
	if err := json.Unmarshal(buf.Bytes(), &data); err != nil {
		t.Fatalf("Failed to parse JSON log output: %v", err)
	}

	if data["trace_id"] != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Errorf("Expected trace_id '4bf92f3577b34da6a3ce929d0e0e4736', got %v", data["trace_id"])
	}
	if data["span_id"] != "00f067aa0ba902b7" {
		t.Errorf("Expected span_id '00f067aa0ba902b7', got %v", data["span_id"])
	}
	ctxObj, ok := data["context"].(map[string]any)
	if !ok {
		t.Fatalf("Expected context to be a map, got %v", data["context"])
	}
	if ctxObj["action"] != "mine_block" {
		t.Errorf("Expected action 'mine_block' in context, got %v", ctxObj["action"])
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"INFO", slog.LevelInfo},
		{"", slog.LevelInfo}, // empty defaults to info
		{"warn", slog.LevelWarn},
		{"WARNING", slog.LevelWarn},
		{"error", slog.LevelError},
		{"ERROR", slog.LevelError},
		{"unknown", slog.LevelInfo},
	}

	for _, tc := range tests {
		got := ParseLevel(tc.input)
		if got != tc.expected {
			t.Errorf("ParseLevel(%q) = %v; want %v", tc.input, got, tc.expected)
		}
	}
}

func TestGlobalSetLevel(t *testing.T) {
	SetLevel("debug")
	if GetLevel() != slog.LevelDebug {
		t.Errorf("expected LevelDebug, got %v", GetLevel())
	}

	// Empty string must set to info
	SetLevel("")
	if GetLevel() != slog.LevelInfo {
		t.Errorf("expected LevelInfo when setting empty string, got %v", GetLevel())
	}

	SetLevel("warn")
	if GetLevel() != slog.LevelWarn {
		t.Errorf("expected LevelWarn, got %v", GetLevel())
	}

	SetLevel("error")
	if GetLevel() != slog.LevelError {
		t.Errorf("expected LevelError, got %v", GetLevel())
	}

	// Reset back to info
	SetLevel("info")
}

func TestDynamicLogLevelFiltering(t *testing.T) {
	buf := &bytes.Buffer{}
	lvlVar := new(slog.LevelVar)
	lvlVar.Set(slog.LevelInfo)

	logger := NewLogger("dynamic-test", lvlVar, buf)

	// At Info level, Debug messages should be suppressed
	logger.Debug("debug hidden message")
	if buf.Len() > 0 {
		t.Fatalf("expected debug message to be filtered out at info level, got: %s", buf.String())
	}

	// Info message should appear
	logger.Info("info message")
	if buf.Len() == 0 {
		t.Fatalf("expected info message to appear")
	}
	buf.Reset()

	// Switch to Debug level dynamically
	lvlVar.Set(slog.LevelDebug)
	logger.Debug("debug visible message")
	if buf.Len() == 0 {
		t.Fatalf("expected debug message to appear after switching to debug level")
	}
}

func TestDualModeHandler_ContextNesting(t *testing.T) {
	var buf bytes.Buffer
	lvl := new(slog.LevelVar)
	lvl.Set(slog.LevelInfo)
	l := NewLogger("test-svc", lvl, &buf)
	l.Info("user logged in", "user_id", "u-123", "ip", "127.0.0.1")

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
	assert.Equal(t, "test-svc", parsed["service"])
	assert.Equal(t, "user logged in", parsed["message"])

	// Invariant: context object exists and holds user_id and ip
	ctxObj, ok := parsed["context"].(map[string]any)
	require.True(t, ok, "context key must be a JSON object")
	assert.Equal(t, "u-123", ctxObj["user_id"])
	assert.Equal(t, "127.0.0.1", ctxObj["ip"])
	assert.Nil(t, parsed["user_id"])
	assert.Nil(t, parsed["ip"])
}

func TestDualModeHandler_LegacyFlatMirroring(t *testing.T) {
	t.Setenv("LOG_LEGACY_FLAT", "true")
	var buf bytes.Buffer
	lvl := new(slog.LevelVar)
	lvl.Set(slog.LevelInfo)
	l := NewLogger("test-svc", lvl, &buf)
	l.Info("order placed", "symbol", "BTC/USDT", "amount", 1.5)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
	assert.Equal(t, "test-svc", parsed["service"])
	assert.Equal(t, "order placed", parsed["message"])

	// Invariant: context object exists and holds symbol and amount
	ctxObj, ok := parsed["context"].(map[string]any)
	require.True(t, ok, "context key must be a JSON object")
	assert.Equal(t, "BTC/USDT", ctxObj["symbol"])
	assert.Equal(t, 1.5, ctxObj["amount"])

	// Invariant: attributes are ALSO mirrored at root when legacy flat is true
	assert.Equal(t, "BTC/USDT", parsed["symbol"])
	assert.Equal(t, 1.5, parsed["amount"])
}

func TestLoggerModuleMapping(t *testing.T) {
	var buf bytes.Buffer
	lvl := new(slog.LevelVar)
	lvl.Set(slog.LevelInfo)
	l := NewLogger("test-svc", lvl, &buf).With("module", "engine")
	l.Info("engine started")

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsed))
	assert.Equal(t, "engine", parsed["logger"])
}
