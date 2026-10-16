package tracer_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dangnmh/core-kit/pkg/tracer"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func TestOTel_ContextPropagation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	tp, shutdown, err := tracer.InitOTel(ctx, tracer.OTelConfig{
		ServiceName: "test-service",
		Endpoint:    "",
		SampleRate:  1.0,
		Enabled:     true,
	})
	if err != nil {
		t.Fatalf("InitOTel error: %v", err)
	}
	defer func() { _ = shutdown(context.Background()) }()
	_ = tp

	// Start root span
	rootCtx, rootSpan := tracer.StartSpan(ctx, "test-tracer", "root-operation")
	defer rootSpan.End()

	traceID, spanID := tracer.ExtractTraceAndSpanID(rootCtx)
	if traceID == "" || spanID == "" {
		t.Fatalf("expected non-empty traceID and spanID, got traceID=%q, spanID=%q", traceID, spanID)
	}

	// Test W3C traceparent string generation
	tpStr := tracer.InjectTraceparent(rootCtx)
	if !strings.HasPrefix(tpStr, "00-") || !strings.Contains(tpStr, traceID) {
		t.Fatalf("invalid traceparent format: %s", tpStr)
	}

	// Test ExtractTraceparent
	restoredCtx := tracer.ExtractTraceparent(context.Background(), tpStr)
	restoredTraceID, restoredSpanID := tracer.ExtractTraceAndSpanID(restoredCtx)
	if restoredTraceID != traceID || restoredSpanID != spanID {
		t.Errorf("restored mismatch: want (%s, %s), got (%s, %s)", traceID, spanID, restoredTraceID, restoredSpanID)
	}

	// Test HTTP header injection & extraction
	header := make(http.Header)
	tracer.InjectHTTP(rootCtx, header)
	if header.Get("traceparent") == "" {
		t.Fatal("expected traceparent header to be present")
	}

	httpRestoredCtx := tracer.ExtractHTTP(context.Background(), header)
	httpTraceID, _ := tracer.ExtractTraceAndSpanID(httpRestoredCtx)
	if httpTraceID != traceID {
		t.Errorf("HTTP restored traceID mismatch: want %s, got %s", traceID, httpTraceID)
	}
}

func TestOTel_Exporter(t *testing.T) {
	t.Parallel()

	var receivedPayload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/traces" {
			http.NotFound(w, r)
			return
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedPayload)
		w.WriteHeader(http.StatusOK)
	}))
	defer func() { server.Close() }()

	ctx := context.Background()
	_, shutdown, err := tracer.InitOTel(ctx, tracer.OTelConfig{
		ServiceName: "exporter-test",
		Endpoint:    server.URL + "/v1/traces",
		SampleRate:  1.0,
		Enabled:     true,
	})
	if err != nil {
		t.Fatalf("InitOTel error: %v", err)
	}

	spanCtx, span := tracer.StartSpan(ctx, "test-tracer", "span-to-export",
		trace.WithAttributes(
			attribute.String("env", "unit-test"),
			attribute.Int("attempt", 1),
			attribute.Bool("success", true),
			attribute.Float64("score", 99.5),
		),
	)
	_ = spanCtx
	span.End()

	// Shutdown flushes the syncer
	_ = shutdown(context.Background())

	if receivedPayload == nil {
		t.Fatal("expected exporter to send payload to mock server")
	}

	resourceSpans, ok := receivedPayload["resourceSpans"].([]any)
	if !ok || len(resourceSpans) == 0 {
		t.Fatalf("invalid resourceSpans: %v", receivedPayload)
	}
}
