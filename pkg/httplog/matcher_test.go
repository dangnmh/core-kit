package httplog_test

import (
	"testing"

	"github.com/dangnmh/core-kit/pkg/httplog"
)

func TestMatcher_PathAndMethod(t *testing.T) {
	t.Parallel()

	cfg := &httplog.Config{
		Rules: []httplog.RuleConfig{
			{
				Path:    "/api/v1/auth/login",
				Methods: []string{"POST"},
			},
			{
				Path:    "/api/v1/users/*",
				Methods: []string{"*"},
			},
			{
				Path:    "/api/v1/any-method",
				Methods: []string{},
			},
		},
	}

	matcher, err := httplog.NewRouteMatcher(cfg)
	if err != nil {
		t.Fatalf("failed to compile matcher: %v", err)
	}

	// Exact match POST
	rule, ok := matcher.Match("POST", "/api/v1/auth/login")
	if !ok || rule.PathPattern != "/api/v1/auth/login" {
		t.Errorf("expected exact match for POST /api/v1/auth/login, got %v", rule)
	}

	// Case-insensitive method match "post"
	rule, ok = matcher.Match("post", "/api/v1/auth/login")
	if !ok || rule.PathPattern != "/api/v1/auth/login" {
		t.Errorf("expected match for lowercase post /api/v1/auth/login, got %v", rule)
	}

	// Method mismatch
	_, ok = matcher.Match("GET", "/api/v1/auth/login")
	if ok {
		t.Errorf("expected match fail for GET /api/v1/auth/login")
	}

	// Prefix match with wildcard method
	rule, ok = matcher.Match("DELETE", "/api/v1/users/123/profile")
	if !ok || rule.PathPattern != "/api/v1/users/*" {
		t.Errorf("expected prefix match for DELETE /api/v1/users/123/profile, got %v", rule)
	}

	// Empty methods list matches any method
	rule, ok = matcher.Match("PATCH", "/api/v1/any-method")
	if !ok || rule.PathPattern != "/api/v1/any-method" {
		t.Errorf("expected match for any-method with empty methods list, got %v", rule)
	}
}

func TestMatcher_FieldRuleCompilation(t *testing.T) {
	t.Parallel()

	rawRules := []httplog.FieldRule{
		{Field: "password", Strategy: httplog.StrategyRedact},
		{Field: "items.*", Strategy: httplog.StrategyRedact},
		{Field: "user.meta.*", Strategy: httplog.StrategyRedact},
		{Field: "data.user.profile.id_card", Strategy: httplog.StrategyPartialMask},
		{Field: "items[].name", Strategy: httplog.StrategyRedact},
		{Field: "orders[].items[].serial_number", Strategy: httplog.StrategyRemove},
		{Field: "tags[]", Strategy: httplog.StrategyRedact},
		{Field: "   ", Strategy: httplog.StrategyRedact}, // should be ignored
		{Field: "default_strategy"},                      // empty strategy defaults to redact
	}

	compiled := httplog.CompileFieldRules(rawRules)
	if len(compiled) != 8 {
		t.Fatalf("compiled count = %d, want 8", len(compiled))
	}

	// 0: Global key "password"
	if compiled[0].Type != httplog.PatternGlobalKey || compiled[0].GlobalKey != "password" {
		t.Errorf("rule 0 = %+v, want global key password", compiled[0])
	}
	if compiled[0].Strategy != httplog.StrategyRedact {
		t.Errorf("rule 0 strategy = %v, want redact", compiled[0].Strategy)
	}

	// 1: Prefix wildcard "items.*"
	if compiled[1].Type != httplog.PatternPrefixWild || compiled[1].PrefixPath != "items" {
		t.Errorf("rule 1 = %+v, want prefix items", compiled[1])
	}

	// 2: Nested prefix wildcard "user.meta.*"
	if compiled[2].Type != httplog.PatternPrefixWild || compiled[2].PrefixPath != "user.meta" {
		t.Errorf("rule 2 = %+v, want prefix user.meta", compiled[2])
	}

	// 3: Deep path "data.user.profile.id_card"
	if compiled[3].Type != httplog.PatternPath || len(compiled[3].Tokens) != 4 {
		t.Errorf("rule 3 = %+v, want 4 tokens", compiled[3])
	}
	if compiled[3].Strategy != httplog.StrategyPartialMask {
		t.Errorf("rule 3 strategy = %v, want partial_mask", compiled[3].Strategy)
	}
	for i, token := range compiled[3].Tokens {
		if token.IsArray {
			t.Errorf("rule 3 token %d (%s) should not be array", i, token.Name)
		}
	}

	// 4: Array path "items[].name"
	if compiled[4].Type != httplog.PatternPath || len(compiled[4].Tokens) != 2 {
		t.Errorf("rule 4 = %+v, want 2 tokens", compiled[4])
	}
	if !compiled[4].Tokens[0].IsArray || compiled[4].Tokens[0].Name != "items" {
		t.Errorf("rule 4 token 0 = %+v, want items[array]", compiled[4].Tokens[0])
	}
	if compiled[4].Tokens[1].IsArray || compiled[4].Tokens[1].Name != "name" {
		t.Errorf("rule 4 token 1 = %+v, want name[scalar]", compiled[4].Tokens[1])
	}

	// 5: Nested array path "orders[].items[].serial_number"
	if compiled[5].Type != httplog.PatternPath || len(compiled[5].Tokens) != 3 {
		t.Errorf("rule 5 = %+v, want 3 tokens", compiled[5])
	}
	if !compiled[5].Tokens[0].IsArray || compiled[5].Tokens[0].Name != "orders" {
		t.Errorf("rule 5 token 0 = %+v, want orders[array]", compiled[5].Tokens[0])
	}
	if !compiled[5].Tokens[1].IsArray || compiled[5].Tokens[1].Name != "items" {
		t.Errorf("rule 5 token 1 = %+v, want items[array]", compiled[5].Tokens[1])
	}
	if compiled[5].Tokens[2].IsArray || compiled[5].Tokens[2].Name != "serial_number" {
		t.Errorf("rule 5 token 2 = %+v, want serial_number[scalar]", compiled[5].Tokens[2])
	}
	if compiled[5].Strategy != httplog.StrategyRemove {
		t.Errorf("rule 5 strategy = %v, want remove", compiled[5].Strategy)
	}

	// 6: Single array token "tags[]"
	if compiled[6].Type != httplog.PatternPath || len(compiled[6].Tokens) != 1 {
		t.Errorf("rule 6 = %+v, want 1 token", compiled[6])
	}
	if !compiled[6].Tokens[0].IsArray || compiled[6].Tokens[0].Name != "tags" {
		t.Errorf("rule 6 token 0 = %+v, want tags[array]", compiled[6].Tokens[0])
	}

	// 7: Default strategy for empty strategy
	if compiled[7].Type != httplog.PatternGlobalKey || compiled[7].GlobalKey != "default_strategy" {
		t.Errorf("rule 7 = %+v, want global key default_strategy", compiled[7])
	}
	if compiled[7].Strategy != httplog.StrategyRedact {
		t.Errorf("rule 7 strategy = %v, want default redact", compiled[7].Strategy)
	}
}

func TestMatcher_RouteRules_ReqAndRes(t *testing.T) {
	t.Parallel()

	cfg := &httplog.Config{
		Default: httplog.GlobalConfig{
			LogResponseOn: "error_only",
		},
		Rules: []httplog.RuleConfig{
			{
				Path:    "/shared-only",
				Methods: []string{"POST"},
				IgnoreFields: []httplog.FieldRule{
					{Field: "shared_secret", Strategy: httplog.StrategyRedact},
				},
			},
			{
				Path:    "/independent",
				Methods: []string{"POST"},
				IgnoreFields: []httplog.FieldRule{
					{Field: "shared_id", Strategy: httplog.StrategyRedact},
				},
				Request: &httplog.PayloadConfig{
					IgnoreFields: []httplog.FieldRule{
						{Field: "password", Strategy: httplog.StrategyRedact},
					},
				},
				Response: &httplog.PayloadConfig{
					LogResponseOn: "all",
					IgnoreFields: []httplog.FieldRule{
						{Field: "access_token", Strategy: httplog.StrategyRemove},
					},
				},
			},
		},
	}

	matcher, err := httplog.NewRouteMatcher(cfg)
	if err != nil {
		t.Fatalf("failed to compile matcher: %v", err)
	}

	// Test 1: Route with shared fields only -> applied to both ReqRules and ResRules
	rule1, ok := matcher.Match("POST", "/shared-only")
	if !ok {
		t.Fatalf("expected match for /shared-only")
	}
	if len(rule1.ReqRules) != 1 || rule1.ReqRules[0].GlobalKey != "shared_secret" {
		t.Errorf("expected ReqRules to inherit shared_secret, got %+v", rule1.ReqRules)
	}
	if len(rule1.ResRules) != 1 || rule1.ResRules[0].GlobalKey != "shared_secret" {
		t.Errorf("expected ResRules to inherit shared_secret, got %+v", rule1.ResRules)
	}
	if rule1.LogResOn != "error_only" {
		t.Errorf("expected LogResOn = error_only, got %v", rule1.LogResOn)
	}

	// Test 2: Route with shared + independent request and response configs
	rule2, ok := matcher.Match("POST", "/independent")
	if !ok {
		t.Fatalf("expected match for /independent")
	}
	if len(rule2.ReqRules) != 2 {
		t.Fatalf("expected 2 ReqRules (shared + req), got %d", len(rule2.ReqRules))
	}
	if rule2.ReqRules[0].GlobalKey != "shared_id" || rule2.ReqRules[1].GlobalKey != "password" {
		t.Errorf("unexpected ReqRules: %+v", rule2.ReqRules)
	}

	if len(rule2.ResRules) != 2 {
		t.Fatalf("expected 2 ResRules (shared + res), got %d", len(rule2.ResRules))
	}
	if rule2.ResRules[0].GlobalKey != "shared_id" || rule2.ResRules[1].GlobalKey != "access_token" {
		t.Errorf("unexpected ResRules: %+v", rule2.ResRules)
	}
	if rule2.LogResOn != "all" {
		t.Errorf("expected LogResOn = all overridden by response config, got %v", rule2.LogResOn)
	}
}

func TestMatcher_PathMatching_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		cfgRules    []httplog.RuleConfig
		method      string
		path        string
		wantMatch   bool
		wantPattern string
	}{
		{
			name: "wildcard slash star matches root and child",
			cfgRules: []httplog.RuleConfig{
				{Path: "/*", Methods: []string{"GET"}},
			},
			method:      "GET",
			path:        "/anything/at/all",
			wantMatch:   true,
			wantPattern: "/*",
		},
		{
			name: "wildcard slash star matches exact slash",
			cfgRules: []httplog.RuleConfig{
				{Path: "/*", Methods: []string{"GET"}},
			},
			method:      "GET",
			path:        "/",
			wantMatch:   true,
			wantPattern: "/*",
		},
		{
			name: "single star matches anything",
			cfgRules: []httplog.RuleConfig{
				{Path: "*", Methods: []string{"GET"}},
			},
			method:      "GET",
			path:        "/some/path",
			wantMatch:   true,
			wantPattern: "*",
		},
		{
			name: "prefix users/* matches /users and /users/123",
			cfgRules: []httplog.RuleConfig{
				{Path: "/api/v1/users/*", Methods: []string{"GET"}},
			},
			method:      "GET",
			path:        "/api/v1/users/123",
			wantMatch:   true,
			wantPattern: "/api/v1/users/*",
		},
		{
			name: "prefix users/* does not match /api/v1/users_backup",
			cfgRules: []httplog.RuleConfig{
				{Path: "/api/v1/users/*", Methods: []string{"GET"}},
			},
			method:    "GET",
			path:      "/api/v1/users_backup",
			wantMatch: false,
		},
		{
			name: "exact path does not match prefix extension",
			cfgRules: []httplog.RuleConfig{
				{Path: "/api/v1/login", Methods: []string{"POST"}},
			},
			method:    "POST",
			path:      "/api/v1/login/extra",
			wantMatch: false,
		},
		{
			name: "method in multiple methods matches",
			cfgRules: []httplog.RuleConfig{
				{Path: "/api/v1/multi", Methods: []string{"GET", "POST"}},
			},
			method:      "POST",
			path:        "/api/v1/multi",
			wantMatch:   true,
			wantPattern: "/api/v1/multi",
		},
		{
			name: "method not in multiple methods does not match",
			cfgRules: []httplog.RuleConfig{
				{Path: "/api/v1/multi", Methods: []string{"GET", "POST"}},
			},
			method:    "DELETE",
			path:      "/api/v1/multi",
			wantMatch: false,
		},
		{
			name:      "no routes configured",
			cfgRules:  []httplog.RuleConfig{},
			method:    "GET",
			path:      "/anything",
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &httplog.Config{Rules: tt.cfgRules}
			matcher, err := httplog.NewRouteMatcher(cfg)
			if err != nil {
				t.Fatalf("failed to create matcher: %v", err)
			}

			rule, ok := matcher.Match(tt.method, tt.path)
			if ok != tt.wantMatch {
				t.Fatalf("Match(%s, %s) ok = %v, want %v", tt.method, tt.path, ok, tt.wantMatch)
			}
			if tt.wantMatch && rule.PathPattern != tt.wantPattern {
				t.Errorf("matched PathPattern = %s, want %s", rule.PathPattern, tt.wantPattern)
			}
		})
	}
}

func TestMatcher_NilHandling(t *testing.T) {
	t.Parallel()

	// NewRouteMatcher(nil) must return error
	_, err := httplog.NewRouteMatcher(nil)
	if err == nil {
		t.Errorf("expected error for nil config, got nil")
	}

	// Match on nil RouteMatcher returns false
	var nilMatcher *httplog.RouteMatcher
	rule, ok := nilMatcher.Match("GET", "/")
	if ok || rule != nil {
		t.Errorf("expected nil, false on nil RouteMatcher.Match")
	}
}

func TestMatcher_InboundAndOutboundIsolation(t *testing.T) {
	t.Parallel()

	cfg := &httplog.Config{
		Inbound: []httplog.RuleConfig{
			{
				Path:    "/api/inbound/login",
				Methods: []string{"POST"},
			},
		},
		Outbound: []httplog.RuleConfig{
			{
				Path:    "/api/outbound/generate",
				Methods: []string{"POST"},
			},
		},
		Rules: []httplog.RuleConfig{
			{
				Path:    "/api/inbound/login",
				Methods: []string{"POST"},
			},
			{
				Path:    "/api/outbound/generate",
				Methods: []string{"POST"},
			},
		},
	}

	matcher, err := httplog.NewRouteMatcher(cfg)
	if err != nil {
		t.Fatalf("failed to create route matcher: %v", err)
	}

	t.Run("inbound matches inbound route", func(t *testing.T) {
		rule, ok := matcher.MatchInbound("POST", "/api/inbound/login")
		if !ok || rule == nil || rule.PathPattern != "/api/inbound/login" {
			t.Errorf("expected MatchInbound to match /api/inbound/login, got %v", rule)
		}
	})

	t.Run("inbound does NOT match outbound route", func(t *testing.T) {
		rule, ok := matcher.MatchInbound("POST", "/api/outbound/generate")
		if ok || rule != nil {
			t.Errorf("expected MatchInbound to NOT match outbound route, got %v", rule)
		}
	})

	t.Run("outbound matches outbound route", func(t *testing.T) {
		rule, ok := matcher.MatchOutbound("POST", "/api/outbound/generate")
		if !ok || rule == nil || rule.PathPattern != "/api/outbound/generate" {
			t.Errorf("expected MatchOutbound to match /api/outbound/generate, got %v", rule)
		}
	})

	t.Run("outbound does NOT match inbound route", func(t *testing.T) {
		rule, ok := matcher.MatchOutbound("POST", "/api/inbound/login")
		if ok || rule != nil {
			t.Errorf("expected MatchOutbound to NOT match inbound route, got %v", rule)
		}
	})

	t.Run("general Match matches both", func(t *testing.T) {
		rule1, ok1 := matcher.Match("POST", "/api/inbound/login")
		if !ok1 || rule1 == nil {
			t.Errorf("expected general Match to match /api/inbound/login")
		}

		rule2, ok2 := matcher.Match("POST", "/api/outbound/generate")
		if !ok2 || rule2 == nil {
			t.Errorf("expected general Match to match /api/outbound/generate")
		}
	})

	t.Run("fallback to general rules when inbound/outbound slice is empty", func(t *testing.T) {
		legacyCfg := &httplog.Config{
			Rules: []httplog.RuleConfig{
				{Path: "/legacy/route", Methods: []string{"GET"}},
			},
		}
		legacyMatcher, err := httplog.NewRouteMatcher(legacyCfg)
		if err != nil {
			t.Fatalf("failed to create legacy matcher: %v", err)
		}

		ruleIn, okIn := legacyMatcher.MatchInbound("GET", "/legacy/route")
		if !okIn || ruleIn == nil {
			t.Errorf("expected MatchInbound fallback to general rules")
		}

		ruleOut, okOut := legacyMatcher.MatchOutbound("GET", "/legacy/route")
		if !okOut || ruleOut == nil {
			t.Errorf("expected MatchOutbound fallback to general rules")
		}
	})
}
