package tracer

import (
	"context"
	"net/http"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// OTelConfig defines tracing configuration for OpenTelemetry.
type OTelConfig struct {
	ServiceName string  `yaml:"service_name" json:"service_name"`
	Endpoint    string  `yaml:"endpoint" json:"endpoint"`       // e.g. "http://jaeger:4318/v1/traces"
	SampleRate  float64 `yaml:"sample_rate" json:"sample_rate"` // 0.0 to 1.0 (default 1.0)
	Enabled     bool    `yaml:"enabled" json:"enabled"`
}

// DefaultOTelConfig returns standard default config.
func DefaultOTelConfig(serviceName string) OTelConfig {
	return OTelConfig{
		ServiceName: serviceName,
		Endpoint:    "http://jaeger:4318/v1/traces",
		SampleRate:  1.0,
		Enabled:     true,
	}
}

// Global text map propagator using W3C TraceContext and Baggage.
var Propagator = propagation.NewCompositeTextMapPropagator(
	propagation.TraceContext{},
	propagation.Baggage{},
)

func init() {
	otel.SetTextMapPropagator(Propagator)
}

// InitOTel initializes OpenTelemetry Provider and registers it globally.
func InitOTel(ctx context.Context, cfg OTelConfig) (*sdktrace.TracerProvider, func(context.Context) error, error) {
	if !cfg.Enabled || cfg.Endpoint == "" {
		tp := sdktrace.NewTracerProvider()
		otel.SetTracerProvider(tp)
		return tp, tp.Shutdown, nil
	}

	var sampler sdktrace.Sampler
	if cfg.SampleRate >= 1.0 {
		sampler = sdktrace.AlwaysSample()
	} else if cfg.SampleRate <= 0.0 {
		sampler = sdktrace.NeverSample()
	} else {
		sampler = sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SampleRate))
	}

	exporter := NewOTLPHTTPExporter(cfg.Endpoint, cfg.ServiceName)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sampler),
		sdktrace.WithSyncer(exporter),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(Propagator)

	return tp, tp.Shutdown, nil
}

// StartSpan starts a new OpenTelemetry span.
func StartSpan(ctx context.Context, tracerName, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	tr := otel.GetTracerProvider().Tracer(tracerName)
	return tr.Start(ctx, spanName, opts...)
}

// InjectHTTP writes W3C TraceContext into HTTP headers.
func InjectHTTP(ctx context.Context, header http.Header) {
	if header == nil {
		return
	}
	Propagator.Inject(ctx, propagation.HeaderCarrier(header))
}

// ExtractHTTP extracts W3C TraceContext from HTTP headers.
func ExtractHTTP(ctx context.Context, header http.Header) context.Context {
	if header == nil {
		return ctx
	}
	return Propagator.Extract(ctx, propagation.HeaderCarrier(header))
}

// InjectTraceparent returns W3C traceparent formatted string: 00-{traceid}-{spanid}-{flags}.
func InjectTraceparent(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	sc := span.SpanContext()
	if !sc.IsValid() {
		return ""
	}
	carrier := propagation.MapCarrier{}
	Propagator.Inject(ctx, carrier)
	return carrier.Get("traceparent")
}

// ExtractTraceparent restores context from a W3C traceparent string.
func ExtractTraceparent(ctx context.Context, traceparent string) context.Context {
	tp := strings.TrimSpace(traceparent)
	if tp == "" {
		return ctx
	}
	carrier := propagation.MapCarrier{
		"traceparent": tp,
	}
	return Propagator.Extract(ctx, carrier)
}

// ExtractTraceAndSpanID returns the hex trace ID and span ID from context.
func ExtractTraceAndSpanID(ctx context.Context) (traceID string, spanID string) {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return "", ""
	}
	return sc.TraceID().String(), sc.SpanID().String()
}
