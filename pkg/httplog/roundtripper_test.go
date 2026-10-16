package httplog_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dangnmh/core-kit/pkg/httplog"
)

func TestRoundTripper_OutboundRequestAndResponseMasking(t *testing.T) {
	logBuf := &safeBuffer{}
	setupTestLogger(logBuf)

	type ServerReceived struct {
		APIKey       string `json:"api_key"`
		ClientSecret string `json:"client_secret"`
		Action       string `json:"action"`
	}

	var received ServerReceived
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = json.Unmarshal(body, &received)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"status": "success",
			"access_token": "secret_outbound_token_123",
			"phone_number": "0987654321",
			"temp_key": "ephemeral_secret_to_remove"
		}`))
	}))
	defer func() { ts.Close() }()

	cfg := &httplog.Config{
		Default: httplog.GlobalConfig{
			MaskValue: "[REDACTED]",
		},
		Rules: []httplog.RuleConfig{
			{
				Path:    "/v1/auth/*",
				Methods: []string{"POST"},
				Enable:  true,
				Request: &httplog.PayloadConfig{
					IgnoreFields: []httplog.FieldRule{
						{Field: "client_secret", Strategy: httplog.StrategyRedact},
						{Field: "api_key", Strategy: httplog.StrategyHashSHA256},
					},
				},
				Response: &httplog.PayloadConfig{
					IgnoreFields: []httplog.FieldRule{
						{Field: "access_token", Strategy: httplog.StrategyRedact},
						{Field: "phone_number", Strategy: httplog.StrategyPartialMask},
						{Field: "temp_key", Strategy: httplog.StrategyRemove},
					},
				},
			},
		},
	}

	engine, err := httplog.NewEngine(cfg)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	client := &http.Client{
		Transport: httplog.NewRoundTripper(http.DefaultTransport, engine),
	}

	reqPayload := `{"api_key":"my-api-key-999","client_secret":"raw_secret_xyz","action":"login"}`
	resp, err := client.Post(ts.URL+"/v1/auth/token", "application/json", bytes.NewBufferString(reqPayload))
	if err != nil {
		t.Fatalf("outbound request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// 1. Verify remote server received untouched original request data
	if received.ClientSecret != "raw_secret_xyz" {
		t.Fatalf("server received corrupted client_secret: got %q, want %q", received.ClientSecret, "raw_secret_xyz")
	}
	if received.APIKey != "my-api-key-999" {
		t.Fatalf("server received corrupted api_key: got %q, want %q", received.APIKey, "my-api-key-999")
	}
	if received.Action != "login" {
		t.Fatalf("server received corrupted action: got %q, want %q", received.Action, "login")
	}

	// 2. Verify client caller received untouched original response data
	resBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}
	var resData map[string]any
	if err := json.Unmarshal(resBytes, &resData); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}

	if resData["access_token"] != "secret_outbound_token_123" {
		t.Fatalf("client received redacted access_token: got %v", resData["access_token"])
	}
	if resData["phone_number"] != "0987654321" {
		t.Fatalf("client received redacted phone_number: got %v", resData["phone_number"])
	}
	if resData["temp_key"] != "ephemeral_secret_to_remove" {
		t.Fatalf("client received redacted temp_key: got %v", resData["temp_key"])
	}

	// 3. Verify logs contain sanitized request and response
	logs := logBuf.String()
	if !strings.Contains(logs, "Outbound HTTP completed") {
		t.Fatalf("expected 'Outbound HTTP completed' in logs, got: %s", logs)
	}
	// Raw secrets must NOT be in logs
	if strings.Contains(logs, "raw_secret_xyz") {
		t.Fatalf("logs contain plaintext client_secret! logs: %s", logs)
	}
	if strings.Contains(logs, "secret_outbound_token_123") {
		t.Fatalf("logs contain plaintext access_token! logs: %s", logs)
	}
	if strings.Contains(logs, "ephemeral_secret_to_remove") {
		t.Fatalf("logs contain removed temp_key! logs: %s", logs)
	}
	if strings.Contains(logs, "0987654321") {
		t.Fatalf("logs contain plaintext phone_number! logs: %s", logs)
	}
	// Sanitized representations must be in logs
	if !strings.Contains(logs, "[REDACTED]") {
		t.Fatalf("expected [REDACTED] in logs, got: %s", logs)
	}
	if !strings.Contains(logs, "sha256:") {
		t.Fatalf("expected hashed api_key in logs, got: %s", logs)
	}
	if !strings.Contains(logs, "098****321") {
		t.Fatalf("expected partially masked phone '098****321' in logs, got: %s", logs)
	}
	if !strings.Contains(logs, `"status":200`) && !strings.Contains(logs, `"status": 200`) {
		t.Fatalf("expected status 200 in logs, got: %s", logs)
	}
}

func TestRoundTripper_PreservesBodyStreams(t *testing.T) {
	logBuf := &safeBuffer{}
	setupTestLogger(logBuf)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body) // Echo raw body back
	}))
	defer func() { ts.Close() }()

	rt := httplog.NewRoundTripper(http.DefaultTransport, nil)
	client := &http.Client{Transport: rt}

	originalContent := "exact-payload-stream-verification-content-12345"
	resp, err := client.Post(ts.URL+"/echo", "text/plain", bytes.NewBufferString(originalContent))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	readContent, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}

	if string(readContent) != originalContent {
		t.Fatalf("body corrupted: got %q, want %q", string(readContent), originalContent)
	}
}

func TestRoundTripper_ConditionalResponseLogging_ErrorOnly(t *testing.T) {
	logBuf := &safeBuffer{}
	setupTestLogger(logBuf)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/ok" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"result":"should_not_be_logged"}`))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal_failure","secret_detail":"db_down"}`))
	}))
	defer func() { ts.Close() }()

	cfg := &httplog.Config{
		Default: httplog.GlobalConfig{
			MaskValue: "[REDACTED]",
		},
		Rules: []httplog.RuleConfig{
			{
				Path:    "/*",
				Methods: []string{"GET"},
				Enable:  true,
				Response: &httplog.PayloadConfig{
					LogResponseOn: "error_only",
					IgnoreFields: []httplog.FieldRule{
						{Field: "secret_detail", Strategy: httplog.StrategyRedact},
					},
				},
			},
		},
	}

	engine, err := httplog.NewEngine(cfg)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	client := &http.Client{
		Transport: httplog.NewRoundTripper(http.DefaultTransport, engine),
	}

	// 1. Success request: response body must NOT be logged
	respOk, err := client.Get(ts.URL + "/ok")
	if err != nil {
		t.Fatalf("get /ok failed: %v", err)
	}
	_ = respOk.Body.Close()

	logsAfterOk := logBuf.String()
	if strings.Contains(logsAfterOk, "should_not_be_logged") {
		t.Fatalf("200 response body was logged when error_only was configured! logs: %s", logsAfterOk)
	}

	// 2. Error request: response body MUST be logged and sanitized
	respErr, err := client.Get(ts.URL + "/error")
	if err != nil {
		t.Fatalf("get /error failed: %v", err)
	}
	_ = respErr.Body.Close()

	logsAfterErr := logBuf.String()
	if !strings.Contains(logsAfterErr, `"res_body":"{\"error\":\"internal_failure\",\"secret_detail\":\"[REDACTED]\"}"`) &&
		!strings.Contains(logsAfterErr, `\"secret_detail\":\"[REDACTED]\"`) {
		t.Fatalf("500 response was not properly logged with redaction! logs: %s", logsAfterErr)
	}
	if strings.Contains(logsAfterErr, "db_down") {
		t.Fatalf("logs contain plaintext secret_detail! logs: %s", logsAfterErr)
	}
}

type errRoundTripper struct{}

func (e *errRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, errors.New("simulated network connection timeout")
}

func TestRoundTripper_RoundTripError(t *testing.T) {
	logBuf := &safeBuffer{}
	setupTestLogger(logBuf)

	rt := httplog.NewRoundTripper(&errRoundTripper{}, nil)
	client := &http.Client{Transport: rt}

	req, _ := http.NewRequest(http.MethodGet, "http://localhost:9999/unreachable", nil)
	_, err := client.Do(req)
	if err == nil {
		t.Fatalf("expected error from errRoundTripper, got nil")
	}

	logs := logBuf.String()
	if !strings.Contains(logs, "Outbound HTTP failed") {
		t.Fatalf("expected 'Outbound HTTP failed' log, got: %s", logs)
	}
	if !strings.Contains(logs, "simulated network connection timeout") {
		t.Fatalf("expected error message in logs, got: %s", logs)
	}
}

func TestRoundTripper_NilEngineAndNilBaseSafeFallback(t *testing.T) {
	logBuf := &safeBuffer{}
	setupTestLogger(logBuf)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ping":"pong"}`))
	}))
	defer func() { ts.Close() }()

	// Both base and engine are nil: should use http.DefaultTransport safely
	rt := httplog.NewRoundTripper(nil, nil)
	client := &http.Client{Transport: rt}

	resp, err := client.Get(ts.URL + "/test-nil")
	if err != nil {
		t.Fatalf("unexpected error with nil engine/base: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != `{"ping":"pong"}` {
		t.Fatalf("got body %q, want {\"ping\":\"pong\"}", string(body))
	}

	logs := logBuf.String()
	if !strings.Contains(logs, "Outbound HTTP completed") {
		t.Fatalf("expected log entry even with nil engine, got: %s", logs)
	}
}

func TestRoundTripper_FormUrlEncodedMasking(t *testing.T) {
	logBuf := &safeBuffer{}
	setupTestLogger(logBuf)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer func() { ts.Close() }()

	cfg := &httplog.Config{
		Default: httplog.GlobalConfig{MaskValue: "[REDACTED]"},
		Rules: []httplog.RuleConfig{
			{
				Path:    "/oauth/token",
				Methods: []string{"POST"},
				Enable:  true,
				Request: &httplog.PayloadConfig{
					IgnoreFields: []httplog.FieldRule{
						{Field: "client_secret", Strategy: httplog.StrategyRedact},
					},
				},
			},
		},
	}

	engine, err := httplog.NewEngine(cfg)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	client := &http.Client{
		Transport: httplog.NewRoundTripper(http.DefaultTransport, engine),
	}

	formBody := "client_id=my_app&client_secret=super_secret_form_value&grant_type=client_credentials"
	resp, err := client.Post(ts.URL+"/oauth/token", "application/x-www-form-urlencoded", strings.NewReader(formBody))
	if err != nil {
		t.Fatalf("post form failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	logs := logBuf.String()
	if strings.Contains(logs, "super_secret_form_value") {
		t.Fatalf("logs contain plaintext form secret! logs: %s", logs)
	}
	if !strings.Contains(logs, "client_secret=%5BREDACTED%5D") && !strings.Contains(logs, "client_secret=[REDACTED]") {
		t.Fatalf("expected redacted form field in logs, got: %s", logs)
	}
}

func TestRoundTripper_DisabledOutboundLogging(t *testing.T) {
	logBuf := &safeBuffer{}
	setupTestLogger(logBuf)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"server_status":"ok"}`))
	}))
	defer func() { ts.Close() }()

	outboundDisabled := false
	cfg := &httplog.Config{
		Default: httplog.GlobalConfig{
			LogOutbound: &outboundDisabled,
		},
	}

	engine, err := httplog.NewEngine(cfg)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	if engine.LogOutboundEnabled() {
		t.Fatalf("expected LogOutboundEnabled to be false")
	}

	client := &http.Client{
		Transport: httplog.NewRoundTripper(http.DefaultTransport, engine),
	}

	resp, err := client.Get(ts.URL + "/test-outbound")
	if err != nil {
		t.Fatalf("get request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(resp.Body)
	if string(respBody) != `{"server_status":"ok"}` {
		t.Fatalf("unexpected body: %s", string(respBody))
	}

	logs := logBuf.String()
	if strings.Contains(logs, "Outbound HTTP completed") || strings.Contains(logs, "test-outbound") {
		t.Fatalf("expected NO outbound logs when LogOutbound is disabled, got: %s", logs)
	}
}

func TestRoundTripper_SkipWhenDisabledOrUnmatched(t *testing.T) {
	logBuf := &safeBuffer{}
	setupTestLogger(logBuf)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer func() { ts.Close() }()

	cfg := &httplog.Config{
		Default: httplog.GlobalConfig{
			MaskValue:    "[REDACTED]",
			LogUnmatched: false, // Default is false
		},
		Rules: []httplog.RuleConfig{
			{
				Path:    "/disabled/path",
				Methods: []string{"GET"},
				Enable:  false, // Explicitly disabled
			},
			{
				Path:    "/enabled/path",
				Methods: []string{"GET"},
				Enable:  true, // Explicitly enabled
			},
		},
	}

	engine, err := httplog.NewEngine(cfg)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	client := &http.Client{
		Transport: httplog.NewRoundTripper(http.DefaultTransport, engine),
	}

	// 1. Request to disabled route -> should NOT log
	resp, err := client.Get(ts.URL + "/disabled/path")
	if err != nil {
		t.Fatalf("disabled path request failed: %v", err)
	}
	_ = resp.Body.Close()
	if strings.Contains(logBuf.String(), "/disabled/path") {
		t.Fatalf("expected NO logs for disabled route, got: %s", logBuf.String())
	}

	// 2. Request to unmatched route when LogUnmatched is false -> should NOT log
	resp2, err := client.Get(ts.URL + "/unmatched/path")
	if err != nil {
		t.Fatalf("unmatched path request failed: %v", err)
	}
	_ = resp2.Body.Close()
	if strings.Contains(logBuf.String(), "/unmatched/path") {
		t.Fatalf("expected NO logs for unmatched route, got: %s", logBuf.String())
	}

	// 3. Request to enabled route -> SHOULD log
	resp3, err := client.Get(ts.URL + "/enabled/path")
	if err != nil {
		t.Fatalf("enabled path request failed: %v", err)
	}
	_ = resp3.Body.Close()
	if !strings.Contains(logBuf.String(), "/enabled/path") {
		t.Fatalf("expected logs for enabled route, got: %s", logBuf.String())
	}
}

func TestRoundTripper_LogWhenUnmatchedEnabled(t *testing.T) {
	logBuf := &safeBuffer{}
	setupTestLogger(logBuf)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer func() { ts.Close() }()

	cfg := &httplog.Config{
		Default: httplog.GlobalConfig{
			MaskValue:    "[REDACTED]",
			LogUnmatched: true, // Explicitly enabled
		},
		Rules: []httplog.RuleConfig{
			{
				Path:    "/disabled/path",
				Methods: []string{"GET"},
				Enable:  false,
			},
		},
	}

	engine, err := httplog.NewEngine(cfg)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	client := &http.Client{
		Transport: httplog.NewRoundTripper(http.DefaultTransport, engine),
	}

	resp, err := client.Get(ts.URL + "/any/unmatched/route")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	_ = resp.Body.Close()

	if !strings.Contains(logBuf.String(), "/any/unmatched/route") {
		t.Fatalf("expected logs when LogUnmatched is true, got: %s", logBuf.String())
	}
}
