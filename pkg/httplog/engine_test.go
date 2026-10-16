package httplog_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/dangnmh/core-kit/pkg/httplog"
)

func TestEngine_SanitizeJSON_AllSelectorsAndStrategies(t *testing.T) {
	t.Parallel()

	inputJSON := `{
		"name": "public_name",
		"password": "my_secret_password",
		"email": "dangnmh@example.com",
		"items": [
			{"id": 1, "name": "item_1", "secret": "secret_1"},
			{"id": 2, "name": "item_2", "secret": "secret_2"}
		],
		"orders": [
			{
				"order_id": "ORD-1",
				"items": [
					{"serial_number": "SN-999-AAA", "desc": "gear"},
					{"serial_number": "SN-888-BBB", "desc": "motor"}
				]
			}
		],
		"user": {
			"profile": {
				"id_card": "123456789",
				"phone": "0987654321",
				"account_id": "acc-998877"
			}
		},
		"meta": {
			"internal_id": 999,
			"token": "tok_xyz"
		},
		"tags": ["alpha", "beta", "gamma"]
	}`

	rules := []httplog.FieldRule{
		// 1. Global key + redact
		{Field: "password", Strategy: httplog.StrategyRedact},
		// 1b. Global key + email_mask
		{Field: "email", Strategy: httplog.StrategyEmailMask},
		// 2. Prefix wildcard + redact
		{Field: "meta.*", Strategy: httplog.StrategyRedact},
		// 3. Array path + redact
		{Field: "items[].secret", Strategy: httplog.StrategyRedact},
		// 4. Nested array path + remove
		{Field: "orders[].items[].serial_number", Strategy: httplog.StrategyRemove},
		// 5a. Exact path + partial_mask
		{Field: "user.profile.phone", Strategy: httplog.StrategyPartialMask},
		// 5b. Exact path + remove
		{Field: "user.profile.id_card", Strategy: httplog.StrategyRemove},
		// 5c. Exact path + hash_sha256
		{Field: "user.profile.account_id", Strategy: httplog.StrategyHashSHA256},
		// 6. Array of primitives + redact
		{Field: "tags[]", Strategy: httplog.StrategyRedact},
	}

	compiled := httplog.CompileFieldRules(rules)
	sanitized := httplog.SanitizeJSON([]byte(inputJSON), compiled, "[REDACTED]")

	var parsed map[string]any
	if err := json.Unmarshal([]byte(sanitized), &parsed); err != nil {
		t.Fatalf("failed to parse sanitized json: %v\nResult was: %s", err, sanitized)
	}

	// 1. Password must be redacted (global key + redact)
	if parsed["password"] != "[REDACTED]" {
		t.Errorf("password = %v, want [REDACTED]", parsed["password"])
	}

	// 1b. Email masked (global key + email_mask)
	if parsed["email"] != "d***h@example.com" {
		t.Errorf("email = %v, want d***h@example.com", parsed["email"])
	}

	// 2. meta.* must be redacted (prefix wildcard)
	if parsed["meta"] != "[REDACTED]" {
		t.Errorf("meta = %v, want [REDACTED]", parsed["meta"])
	}

	// 3. items[].secret must be redacted (array path)
	items, ok := parsed["items"].([]any)
	if !ok || len(items) != 2 {
		t.Fatalf("expected items array of length 2, got: %+v", parsed["items"])
	}
	item0 := items[0].(map[string]any)
	if item0["secret"] != "[REDACTED]" || item0["name"] != "item_1" || item0["id"] != float64(1) {
		t.Errorf("items[0] mismatch: %+v", item0)
	}
	item1 := items[1].(map[string]any)
	if item1["secret"] != "[REDACTED]" || item1["name"] != "item_2" || item1["id"] != float64(2) {
		t.Errorf("items[1] mismatch: %+v", item1)
	}

	// 4. orders[].items[].serial_number must be removed (nested array path + remove)
	orders, ok := parsed["orders"].([]any)
	if !ok || len(orders) != 1 {
		t.Fatalf("expected orders array of length 1, got: %+v", parsed["orders"])
	}
	ord0 := orders[0].(map[string]any)
	ordItems := ord0["items"].([]any)
	ordItem0 := ordItems[0].(map[string]any)
	if _, exists := ordItem0["serial_number"]; exists {
		t.Errorf("serial_number should have been removed from ordItems[0]: %+v", ordItem0)
	}
	if ordItem0["desc"] != "gear" {
		t.Errorf("ordItems[0].desc = %v, want gear", ordItem0["desc"])
	}

	// 5. user.profile exact paths
	user := parsed["user"].(map[string]any)
	profile := user["profile"].(map[string]any)

	// phone: partial_mask
	if profile["phone"] != "098****321" {
		t.Errorf("phone = %v, want 098****321", profile["phone"])
	}

	// id_card: removed
	if _, exists := profile["id_card"]; exists {
		t.Errorf("id_card should have been removed: %+v", profile)
	}

	// account_id: hash_sha256
	accountID, ok := profile["account_id"].(string)
	if !ok || !strings.HasPrefix(accountID, "sha256:") {
		t.Errorf("account_id = %v, want sha256:...", profile["account_id"])
	}

	// 6. tags[] array of primitives redacted
	tags, ok := parsed["tags"].([]any)
	if !ok || len(tags) != 3 {
		t.Fatalf("expected tags array of length 3, got: %+v", parsed["tags"])
	}
	for i, tag := range tags {
		if tag != "[REDACTED]" {
			t.Errorf("tags[%d] = %v, want [REDACTED]", i, tag)
		}
	}
}

func TestEngine_SanitizeJSON_EdgeCases(t *testing.T) {
	t.Parallel()

	// 1. Empty body
	emptyRes := httplog.SanitizeJSON([]byte(""), nil, "[REDACTED]")
	if emptyRes != "" {
		t.Errorf("empty body sanitized = %q, want empty", emptyRes)
	}

	// 2. Malformed JSON returns raw string without panicking
	malformed := []byte(`{"broken": json...`)
	malformedRes := httplog.SanitizeJSON(malformed, nil, "[REDACTED]")
	if malformedRes != string(malformed) {
		t.Errorf("malformed JSON = %q, want %q", malformedRes, string(malformed))
	}

	// 3. Root array of objects
	rootArrayJSON := `[{"token": "tok1", "val": 1}, {"token": "tok2", "val": 2}]`
	rules := httplog.CompileFieldRules([]httplog.FieldRule{
		{Field: "token", Strategy: httplog.StrategyRedact},
	})
	rootArrayRes := httplog.SanitizeJSON([]byte(rootArrayJSON), rules, "[REDACTED]")
	var parsedArr []map[string]any
	if err := json.Unmarshal([]byte(rootArrayRes), &parsedArr); err != nil {
		t.Fatalf("failed to unmarshal root array: %v", err)
	}
	if parsedArr[0]["token"] != "[REDACTED]" || parsedArr[1]["token"] != "[REDACTED]" {
		t.Errorf("root array tokens not redacted: %+v", parsedArr)
	}

	// 4. Global key matching a container object (remove and redact)
	containerJSON := `{"credentials": {"key": "secret"}, "keep": true}`
	containerRules := httplog.CompileFieldRules([]httplog.FieldRule{
		{Field: "credentials", Strategy: httplog.StrategyRemove},
	})
	containerRes := httplog.SanitizeJSON([]byte(containerJSON), containerRules, "[REDACTED]")
	var parsedContainer map[string]any
	if err := json.Unmarshal([]byte(containerRes), &parsedContainer); err != nil {
		t.Fatalf("failed to unmarshal container: %v", err)
	}
	if _, exists := parsedContainer["credentials"]; exists {
		t.Errorf("credentials object container should be removed: %+v", parsedContainer)
	}

	// 5. Deeply nested JSON exceeding MaxRecursionDepth (32) does not crash
	nested := make(map[string]any)
	curr := nested
	for i := 0; i < 40; i++ {
		next := make(map[string]any)
		curr["child"] = next
		curr = next
	}
	curr["secret"] = "deep_secret"
	nestedBytes, err := json.Marshal(nested)
	if err != nil {
		t.Fatalf("marshal nested: %v", err)
	}
	deepRules := httplog.CompileFieldRules([]httplog.FieldRule{
		{Field: "secret", Strategy: httplog.StrategyRedact},
	})
	// Must not panic or stack overflow
	deepRes := httplog.SanitizeJSON(nestedBytes, deepRules, "[REDACTED]")
	if deepRes == "" {
		t.Errorf("expected non-empty result from deep JSON traversal")
	}
}

func TestEngine_SanitizeForm(t *testing.T) {
	t.Parallel()

	form := "username=admin&password=secret123&phone=0987654321&token=xyz789&debug=remove_me"
	rules := httplog.CompileFieldRules([]httplog.FieldRule{
		{Field: "password", Strategy: httplog.StrategyRedact},
		{Field: "phone", Strategy: httplog.StrategyPartialMask},
		{Field: "token", Strategy: httplog.StrategyHashSHA256},
		{Field: "debug", Strategy: httplog.StrategyRemove},
	})

	res := httplog.SanitizeForm([]byte(form), rules, "[REDACTED]")

	// password should be redacted
	if !strings.Contains(res, "password=%5BREDACTED%5D") && !strings.Contains(res, "password=[REDACTED]") {
		t.Errorf("form sanitized password mismatch: %s", res)
	}

	// phone should be partially masked (098****321 -> url encoded: 098%2A%2A%2A%2A321 or 098****321)
	if !strings.Contains(res, "phone=098%2A%2A%2A%2A321") && !strings.Contains(res, "phone=098****321") {
		t.Errorf("form sanitized phone mismatch: %s", res)
	}

	// token should be sha256 hashed
	if !strings.Contains(res, "token=sha256") {
		t.Errorf("form sanitized token mismatch: %s", res)
	}

	// debug should be removed
	if strings.Contains(res, "debug=") {
		t.Errorf("form sanitized debug should have been removed: %s", res)
	}

	// username should be preserved
	if !strings.Contains(res, "username=admin") {
		t.Errorf("form sanitized username mismatch: %s", res)
	}

	// Empty form
	if httplog.SanitizeForm([]byte(""), rules, "[REDACTED]") != "" {
		t.Errorf("expected empty string for empty form")
	}
}

func TestEngine_SummarizeBinary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		contentType string
		size        int
		want        string
	}{
		{"audio/wav", 1024, "[BINARY audio/wav, size: 1024 bytes]"},
		{"image/png", 2048, "[BINARY image/png, size: 2048 bytes]"},
		{"application/octet-stream", 0, "[BINARY application/octet-stream, size: 0 bytes]"},
		{"application/pdf", 512000, "[BINARY application/pdf, size: 512000 bytes]"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.contentType, func(t *testing.T) {
			t.Parallel()
			got := httplog.SummarizeBinary(tt.contentType, tt.size)
			if got != tt.want {
				t.Fatalf("SummarizeBinary(%s, %d) = %q, want %q", tt.contentType, tt.size, got, tt.want)
			}
		})
	}
}

func TestEngine_ShouldLogResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		defaultLog any
		routeLog   any
		status     int
		wantLog    bool
	}{
		{
			name:       "default all logs 200",
			defaultLog: "all",
			status:     200,
			wantLog:    true,
		},
		{
			name:       "default all logs 500",
			defaultLog: "all",
			status:     500,
			wantLog:    true,
		},
		{
			name:       "default error_only suppresses 200",
			defaultLog: "error_only",
			status:     200,
			wantLog:    false,
		},
		{
			name:       "default error_only logs 400",
			defaultLog: "error_only",
			status:     400,
			wantLog:    true,
		},
		{
			name:       "default error_only logs 500",
			defaultLog: "error_only",
			status:     500,
			wantLog:    true,
		},
		{
			name:       "status codes list []int matches 400",
			defaultLog: []int{400, 500},
			status:     400,
			wantLog:    true,
		},
		{
			name:       "status codes list []int suppresses 404",
			defaultLog: []int{400, 500},
			status:     404,
			wantLog:    false,
		},
		{
			name:       "status codes list []any matches 500",
			defaultLog: []any{400, 500},
			status:     500,
			wantLog:    true,
		},
		{
			name:       "route rule overrides default error_only to all",
			defaultLog: "error_only",
			routeLog:   "all",
			status:     200,
			wantLog:    true,
		},
		{
			name:       "route rule overrides default all to error_only",
			defaultLog: "all",
			routeLog:   "error_only",
			status:     200,
			wantLog:    false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &httplog.Config{
				Default: httplog.GlobalConfig{
					LogResponseOn: tt.defaultLog,
				},
				Rules: []httplog.RuleConfig{
					{
						Path: "/test",
						Response: &httplog.PayloadConfig{
							LogResponseOn: tt.routeLog,
						},
					},
				},
			}

			eng, err := httplog.NewEngine(cfg)
			if err != nil {
				t.Fatalf("failed to create engine: %v", err)
			}

			rule, _ := eng.Match("GET", "/test")
			got := eng.ShouldLogResponse(tt.status, rule)
			if got != tt.wantLog {
				t.Errorf("ShouldLogResponse(%d) = %v, want %v", tt.status, got, tt.wantLog)
			}
		})
	}
}

func TestEngine_SanitizeBody_ContentTypesAndTruncation(t *testing.T) {
	t.Parallel()

	cfg := &httplog.Config{
		Default: httplog.GlobalConfig{
			MaskValue:   "[MASKED]",
			MaxBodySize: 100,
		},
		Rules: []httplog.RuleConfig{
			{
				Path:    "/api/test",
				Methods: []string{"POST"},
				Request: &httplog.PayloadConfig{
					IgnoreFields: []httplog.FieldRule{
						{Field: "secret", Strategy: httplog.StrategyRedact},
					},
				},
			},
		},
	}

	eng, err := httplog.NewEngine(cfg)
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	rule, ok := eng.Match("POST", "/api/test")
	if !ok {
		t.Fatalf("route match failed")
	}

	// 1. JSON sanitization with custom mask value
	jsonBody := []byte(`{"secret": "val", "data": "info"}`)
	sanitizedJSON := eng.SanitizeBody("application/json", jsonBody, rule.ReqRules)
	if !strings.Contains(sanitizedJSON, "[MASKED]") {
		t.Errorf("sanitized json does not contain custom mask value [MASKED]: %s", sanitizedJSON)
	}

	// 2. JSON with charset
	sanitizedJSONCharset := eng.SanitizeBody("application/json; charset=utf-8", jsonBody, rule.ReqRules)
	if !strings.Contains(sanitizedJSONCharset, "[MASKED]") {
		t.Errorf("sanitized json with charset mismatch: %s", sanitizedJSONCharset)
	}

	// 3. Problem+json
	sanitizedProblemJSON := eng.SanitizeBody("application/problem+json", jsonBody, rule.ReqRules)
	if !strings.Contains(sanitizedProblemJSON, "[MASKED]") {
		t.Errorf("sanitized problem+json mismatch: %s", sanitizedProblemJSON)
	}

	// 4. Form urlencoded
	formBody := []byte("secret=my_secret&user=john")
	sanitizedForm := eng.SanitizeBody("application/x-www-form-urlencoded", formBody, rule.ReqRules)
	if !strings.Contains(sanitizedForm, "%5BMASKED%5D") && !strings.Contains(sanitizedForm, "[MASKED]") {
		t.Errorf("sanitized form mismatch: %s", sanitizedForm)
	}

	// 5. Multipart form data
	sanitizedMultipart := eng.SanitizeBody("multipart/form-data; boundary=xyz", []byte("---xyz"), rule.ReqRules)
	if sanitizedMultipart != "[MULTIPART_FORM_DATA]" {
		t.Errorf("multipart summary = %q, want [MULTIPART_FORM_DATA]", sanitizedMultipart)
	}

	// 6. Binary types
	sanitizedAudio := eng.SanitizeBody("audio/mpeg", make([]byte, 2048), rule.ReqRules)
	if sanitizedAudio != "[BINARY audio/mpeg, size: 2048 bytes]" {
		t.Errorf("binary summary = %q, want [BINARY audio/mpeg, size: 2048 bytes]", sanitizedAudio)
	}

	// 7. Plain text / fallback
	plainText := []byte("hello plain text")
	sanitizedPlain := eng.SanitizeBody("text/plain", plainText, rule.ReqRules)
	if sanitizedPlain != "hello plain text" {
		t.Errorf("plain text = %q, want hello plain text", sanitizedPlain)
	}

	// 8. MaxBodySize truncation
	longText := []byte(strings.Repeat("A", 200))
	sanitizedLong := eng.SanitizeBody("text/plain", longText, rule.ReqRules)
	if len(sanitizedLong) != 100 {
		t.Errorf("truncated length = %d, want 100", len(sanitizedLong))
	}

	// 9. Empty body returns empty string
	if eng.SanitizeBody("application/json", nil, rule.ReqRules) != "" {
		t.Errorf("expected empty string for nil body")
	}
}

func TestEngine_NilHandling(t *testing.T) {
	t.Parallel()

	// NewEngine with nil config returns ErrNilConfig
	_, err := httplog.NewEngine(nil)
	if err == nil {
		t.Errorf("expected error when cfg is nil")
	}

	// Methods on nil *engineImpl
	var nilEng httplog.Engine
	_ = nilEng // compile check for interface
}

func TestEngine_SanitizeJSON_ObjectRedactionAndPrefixInArray(t *testing.T) {
	t.Parallel()

	inputJSON := `{
		"user": {
			"profile": {
				"name": "john",
				"age": 30
			},
			"credentials": {
				"hash": "abc123xyz"
			}
		},
		"orders": [
			{
				"meta": {
					"ip": "1.2.3.4",
					"agent": "curl"
				},
				"id": "ord-1"
			}
		]
	}`

	rules := []httplog.FieldRule{
		// Exact object path redact
		{Field: "user.profile", Strategy: httplog.StrategyRedact},
		// Exact object path remove
		{Field: "user.credentials", Strategy: httplog.StrategyRemove},
		// Prefix wildcard inside array
		{Field: "orders[].meta.*", Strategy: httplog.StrategyRedact},
	}

	compiled := httplog.CompileFieldRules(rules)
	sanitized := httplog.SanitizeJSON([]byte(inputJSON), compiled, "[REDACTED]")

	var parsed map[string]any
	if err := json.Unmarshal([]byte(sanitized), &parsed); err != nil {
		t.Fatalf("failed to parse json: %v\nResult: %s", err, sanitized)
	}

	user := parsed["user"].(map[string]any)

	// user.profile was redacted
	if user["profile"] != "[REDACTED]" {
		t.Errorf("user.profile = %v, want [REDACTED]", user["profile"])
	}

	// user.credentials was removed
	if _, exists := user["credentials"]; exists {
		t.Errorf("user.credentials should have been removed: %+v", user)
	}

	// orders[0].meta was redacted
	orders := parsed["orders"].([]any)
	ord0 := orders[0].(map[string]any)
	if ord0["meta"] != "[REDACTED]" {
		t.Errorf("orders[0].meta = %v, want [REDACTED]", ord0["meta"])
	}
	if ord0["id"] != "ord-1" {
		t.Errorf("orders[0].id = %v, want ord-1", ord0["id"])
	}
}

func TestEngine_SanitizeForm_PrefixWildcard(t *testing.T) {
	t.Parallel()

	form := "user.name=john&meta.token=tok123&meta.secret=sec456&common=ok"
	rules := httplog.CompileFieldRules([]httplog.FieldRule{
		{Field: "meta.*", Strategy: httplog.StrategyRedact},
	})

	res := httplog.SanitizeForm([]byte(form), rules, "[REDACTED]")
	if !strings.Contains(res, "meta.token=%5BREDACTED%5D") && !strings.Contains(res, "meta.token=[REDACTED]") {
		t.Errorf("form meta.token not redacted: %s", res)
	}
	if !strings.Contains(res, "meta.secret=%5BREDACTED%5D") && !strings.Contains(res, "meta.secret=[REDACTED]") {
		t.Errorf("form meta.secret not redacted: %s", res)
	}
	if !strings.Contains(res, "user.name=john") {
		t.Errorf("form user.name should be preserved: %s", res)
	}
}
