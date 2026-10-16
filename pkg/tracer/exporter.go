package tracer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// OTLPHTTPExporter exports spans via standard OTLP/HTTP JSON to Jaeger or OTel Collector.
type OTLPHTTPExporter struct {
	endpoint    string
	serviceName string
	client      *http.Client
}

// NewOTLPHTTPExporter creates a new OTLP/HTTP JSON exporter.
func NewOTLPHTTPExporter(endpoint, serviceName string) *OTLPHTTPExporter {
	if endpoint == "" {
		endpoint = "http://jaeger:4318/v1/traces"
	}
	return &OTLPHTTPExporter{
		endpoint:    endpoint,
		serviceName: serviceName,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// ExportSpans converts ReadOnlySpans to OTLP JSON payload and POSTs to the endpoint.
func (e *OTLPHTTPExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	if len(spans) == 0 {
		return nil
	}

	payload := e.buildOTLPPayload(spans)
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal otlp spans: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create otlp request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		// Non-blocking: fail quietly on network issues to not crash services
		return nil
	}
	defer func() { _ = resp.Body.Close() }()

	return nil
}

// Shutdown closes the exporter.
func (e *OTLPHTTPExporter) Shutdown(ctx context.Context) error {
	return nil
}

// buildOTLPPayload converts spans to OTLP JSON format.
func (e *OTLPHTTPExporter) buildOTLPPayload(spans []sdktrace.ReadOnlySpan) map[string]any {
	otlpSpans := make([]map[string]any, 0, len(spans))

	for _, s := range spans {
		sc := s.SpanContext()
		parent := s.Parent()

		spanObj := map[string]any{
			"traceId":           sc.TraceID().String(),
			"spanId":            sc.SpanID().String(),
			"name":              s.Name(),
			"kind":              int(s.SpanKind()),
			"startTimeUnixNano": strconv.FormatInt(s.StartTime().UnixNano(), 10),
			"endTimeUnixNano":   strconv.FormatInt(s.EndTime().UnixNano(), 10),
		}

		if parent.IsValid() {
			spanObj["parentSpanId"] = parent.SpanID().String()
		}

		// Convert attributes
		attrs := make([]map[string]any, 0, len(s.Attributes()))
		for _, attr := range s.Attributes() {
			attrs = append(attrs, map[string]any{
				"key":   string(attr.Key),
				"value": formatOTLPValue(attr.Value),
			})
		}
		if len(attrs) > 0 {
			spanObj["attributes"] = attrs
		}

		// Convert links
		if len(s.Links()) > 0 {
			links := make([]map[string]any, 0, len(s.Links()))
			for _, l := range s.Links() {
				if l.SpanContext.IsValid() {
					links = append(links, map[string]any{
						"traceId": l.SpanContext.TraceID().String(),
						"spanId":  l.SpanContext.SpanID().String(),
					})
				}
			}
			spanObj["links"] = links
		}

		// Status
		status := s.Status()
		spanObj["status"] = map[string]any{
			"code":        int(status.Code),
			"description": status.Description,
		}

		otlpSpans = append(otlpSpans, spanObj)
	}

	return map[string]any{
		"resourceSpans": []map[string]any{
			{
				"resource": map[string]any{
					"attributes": []map[string]any{
						{
							"key": "service.name",
							"value": map[string]any{
								"stringValue": e.serviceName,
							},
						},
					},
				},
				"scopeSpans": []map[string]any{
					{
						"scope": map[string]any{
							"name": e.serviceName,
						},
						"spans": otlpSpans,
					},
				},
			},
		},
	}
}

func formatOTLPValue(v attribute.Value) map[string]any {
	switch v.Type() {
	case attribute.BOOL:
		return map[string]any{"boolValue": v.AsBool()}
	case attribute.INT64:
		return map[string]any{"intValue": strconv.FormatInt(v.AsInt64(), 10)}
	case attribute.FLOAT64:
		return map[string]any{"doubleValue": v.AsFloat64()}
	default:
		return map[string]any{"stringValue": v.AsString()}
	}
}
