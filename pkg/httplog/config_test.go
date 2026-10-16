package httplog_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dangnmh/core-kit/pkg/httplog"
	"gopkg.in/yaml.v3"
)

func TestStrategy_Apply(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		strategy   httplog.Strategy
		input      string
		maskValue  string
		wantVal    any
		wantRemove bool
	}{
		{
			name:      "default redact with default mask",
			strategy:  httplog.StrategyRedact,
			input:     "secret123",
			maskValue: "[REDACTED]",
			wantVal:   "[REDACTED]",
		},
		{
			name:      "redact with custom mask",
			strategy:  httplog.StrategyRedact,
			input:     "secret123",
			maskValue: "***CONFIDENTIAL***",
			wantVal:   "***CONFIDENTIAL***",
		},
		{
			name:      "redact with empty maskValue falls back to [REDACTED]",
			strategy:  httplog.StrategyRedact,
			input:     "secret123",
			maskValue: "",
			wantVal:   "[REDACTED]",
		},
		{
			name:      "phone partial mask 10 chars",
			strategy:  httplog.StrategyPartialMask,
			input:     "0987654321",
			maskValue: "[REDACTED]",
			wantVal:   "098****321",
		},
		{
			name:      "credit card partial mask 16 chars",
			strategy:  httplog.StrategyPartialMask,
			input:     "4111222233334444",
			maskValue: "[REDACTED]",
			wantVal:   "4111********4444",
		},
		{
			name:      "short partial mask empty",
			strategy:  httplog.StrategyPartialMask,
			input:     "",
			maskValue: "[REDACTED]",
			wantVal:   "",
		},
		{
			name:      "short partial mask 3 chars",
			strategy:  httplog.StrategyPartialMask,
			input:     "123",
			maskValue: "[REDACTED]",
			wantVal:   "***",
		},
		{
			name:      "short partial mask 4 chars",
			strategy:  httplog.StrategyPartialMask,
			input:     "1234",
			maskValue: "[REDACTED]",
			wantVal:   "****",
		},
		{
			name:      "partial mask 5 chars",
			strategy:  httplog.StrategyPartialMask,
			input:     "12345",
			maskValue: "[REDACTED]",
			wantVal:   "1***5",
		},
		{
			name:      "partial mask 6 chars",
			strategy:  httplog.StrategyPartialMask,
			input:     "123456",
			maskValue: "[REDACTED]",
			wantVal:   "1****6",
		},
		{
			name:      "partial mask 7 chars",
			strategy:  httplog.StrategyPartialMask,
			input:     "1234567",
			maskValue: "[REDACTED]",
			wantVal:   "123*567",
		},
		{
			name:      "email mask standard",
			strategy:  httplog.StrategyEmailMask,
			input:     "dangnmh@example.com",
			maskValue: "[REDACTED]",
			wantVal:   "d***h@example.com",
		},
		{
			name:      "email mask 1-char username",
			strategy:  httplog.StrategyEmailMask,
			input:     "a@example.com",
			maskValue: "[REDACTED]",
			wantVal:   "a***@example.com",
		},
		{
			name:      "email mask 2-char username",
			strategy:  httplog.StrategyEmailMask,
			input:     "ab@example.com",
			maskValue: "[REDACTED]",
			wantVal:   "a***@example.com",
		},
		{
			name:      "email mask 3-char username",
			strategy:  httplog.StrategyEmailMask,
			input:     "abc@example.com",
			maskValue: "[REDACTED]",
			wantVal:   "a***c@example.com",
		},
		{
			name:      "email mask invalid no @",
			strategy:  httplog.StrategyEmailMask,
			input:     "notanemail",
			maskValue: "[REDACTED]",
			wantVal:   "[REDACTED]",
		},
		{
			name:      "email mask empty user part",
			strategy:  httplog.StrategyEmailMask,
			input:     "@example.com",
			maskValue: "[REDACTED]",
			wantVal:   "[REDACTED]",
		},
		{
			name:      "email mask empty domain part",
			strategy:  httplog.StrategyEmailMask,
			input:     "user@",
			maskValue: "[REDACTED]",
			wantVal:   "[REDACTED]",
		},
		{
			name:       "remove strategy",
			strategy:   httplog.StrategyRemove,
			input:      "anything",
			maskValue:  "[REDACTED]",
			wantRemove: true,
			wantVal:    nil,
		},
		{
			name:      "sha256 hash strategy",
			strategy:  httplog.StrategyHashSHA256,
			input:     "user_123",
			maskValue: "[REDACTED]",
			wantVal:   "sha256:80fba0ae1c48e3978e43e4efc365e14e12ea0c830ba8ba5b9a2dafc7e3f2ab8b",
		},
		{
			name:      "unknown strategy fallback to redact",
			strategy:  httplog.Strategy("unknown_strategy"),
			input:     "user_123",
			maskValue: "[REDACTED]",
			wantVal:   "[REDACTED]",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotVal, gotRemove := httplog.ApplyStrategy(tt.input, tt.strategy, tt.maskValue)
			if gotRemove != tt.wantRemove {
				t.Fatalf("ApplyStrategy remove = %v, want %v", gotRemove, tt.wantRemove)
			}
			if !gotRemove && gotVal != tt.wantVal {
				t.Fatalf("ApplyStrategy val = %v, want %v", gotVal, tt.wantVal)
			}
		})
	}
}

func TestConfig_YAMLUnmarshal(t *testing.T) {
	t.Parallel()

	yamlData := `
version: "1.0"
default:
  mask_value: "[MASKED]"
  max_body_size: 2048
  log_response_on: "error_only"
  headers:
    - "authorization"
rules:
  - path: "/api/v1/auth"
    methods: ["POST"]
    ignore_fields:
      - "shared_token"
    request:
      ignore_fields:
        - "password"
        - field: "token"
          strategy: "remove"
        - field: "explicit_field_only"
    response:
      log_response_on: "all"
      ignore_fields:
        - field: "card_number"
          strategy: "partial_mask"
`
	var cfg httplog.Config
	err := yaml.Unmarshal([]byte(yamlData), &cfg)
	if err != nil {
		t.Fatalf("failed to unmarshal yaml: %v", err)
	}

	if cfg.Version != "1.0" {
		t.Errorf("version = %s, want 1.0", cfg.Version)
	}
	if cfg.Default.MaskValue != "[MASKED]" {
		t.Errorf("mask_value = %s, want [MASKED]", cfg.Default.MaskValue)
	}
	if cfg.Default.MaxBodySize != 2048 {
		t.Errorf("max_body_size = %d, want 2048", cfg.Default.MaxBodySize)
	}
	if cfg.Default.LogResponseOn != "error_only" {
		t.Errorf("log_response_on = %v, want error_only", cfg.Default.LogResponseOn)
	}
	if len(cfg.Default.Headers) != 1 || cfg.Default.Headers[0] != "authorization" {
		t.Errorf("headers = %+v, want [authorization]", cfg.Default.Headers)
	}

	if len(cfg.Rules) != 1 {
		t.Fatalf("rules count = %d, want 1", len(cfg.Rules))
	}

	rule := cfg.Rules[0]
	if rule.Path != "/api/v1/auth" {
		t.Errorf("path = %s, want /api/v1/auth", rule.Path)
	}
	if len(rule.Methods) != 1 || rule.Methods[0] != "POST" {
		t.Errorf("methods = %+v, want [POST]", rule.Methods)
	}
	if len(rule.IgnoreFields) != 1 || rule.IgnoreFields[0].Field != "shared_token" || rule.IgnoreFields[0].Strategy != httplog.StrategyRedact {
		t.Errorf("shared ignore_fields = %+v", rule.IgnoreFields)
	}

	reqRules := rule.Request.IgnoreFields
	if len(reqRules) != 3 {
		t.Fatalf("req rules count = %d, want 3", len(reqRules))
	}
	if reqRules[0].Field != "password" || reqRules[0].Strategy != httplog.StrategyRedact {
		t.Errorf("rule 0 = %+v, want field password, strategy redact", reqRules[0])
	}
	if reqRules[1].Field != "token" || reqRules[1].Strategy != httplog.StrategyRemove {
		t.Errorf("rule 1 = %+v, want field token, strategy remove", reqRules[1])
	}
	if reqRules[2].Field != "explicit_field_only" || reqRules[2].Strategy != httplog.StrategyRedact {
		t.Errorf("rule 2 = %+v, want field explicit_field_only, strategy redact", reqRules[2])
	}

	resRules := rule.Response.IgnoreFields
	if len(resRules) != 1 {
		t.Fatalf("res rules count = %d, want 1", len(resRules))
	}
	if resRules[0].Field != "card_number" || resRules[0].Strategy != httplog.StrategyPartialMask {
		t.Errorf("res rule 0 = %+v, want field card_number, strategy partial_mask", resRules[0])
	}
	if rule.Response.LogResponseOn != "all" {
		t.Errorf("response log_response_on = %v, want all", rule.Response.LogResponseOn)
	}
}

func TestConfig_YAMLUnmarshal_InvalidFieldRule(t *testing.T) {
	t.Parallel()

	invalidYAML := `
rules:
  - path: "/test"
    request:
      ignore_fields:
        - [1, 2, 3] # array instead of string or mapping
`
	var cfg httplog.Config
	err := yaml.Unmarshal([]byte(invalidYAML), &cfg)
	if err == nil {
		t.Fatalf("expected error unmarshaling invalid field rule, got nil")
	}
}

func TestLoadConfig(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	t.Run("valid file with defaults", func(t *testing.T) {
		t.Parallel()
		validPath := filepath.Join(tmpDir, "config_default.yaml")
		content := `
version: "1.0"
default:
  max_body_size: 1024
rules:
  - path: "/ping"
`
		if err := os.WriteFile(validPath, []byte(content), 0600); err != nil {
			t.Fatalf("failed to write temp config: %v", err)
		}

		cfg, err := httplog.LoadConfig(validPath)
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		if cfg.Default.MaskValue != "[REDACTED]" {
			t.Errorf("mask_value = %q, want default [REDACTED]", cfg.Default.MaskValue)
		}
		if cfg.Default.MaxBodySize != 1024 {
			t.Errorf("max_body_size = %d, want 1024", cfg.Default.MaxBodySize)
		}
		if len(cfg.Rules) != 1 || cfg.Rules[0].Path != "/ping" {
			t.Errorf("rules mismatch: %+v", cfg.Rules)
		}
	})

	t.Run("valid file with custom mask value", func(t *testing.T) {
		t.Parallel()
		customPath := filepath.Join(tmpDir, "config_custom.yaml")
		content := `
version: "1.0"
default:
  mask_value: "*****"
`
		if err := os.WriteFile(customPath, []byte(content), 0600); err != nil {
			t.Fatalf("failed to write temp config: %v", err)
		}

		cfg, err := httplog.LoadConfig(customPath)
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		if cfg.Default.MaskValue != "*****" {
			t.Errorf("mask_value = %q, want *****", cfg.Default.MaskValue)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		t.Parallel()
		nonExistentPath := filepath.Join(tmpDir, "non_existent.yaml")
		_, err := httplog.LoadConfig(nonExistentPath)
		if err == nil {
			t.Fatalf("expected error for non-existent file, got nil")
		}
	})

	t.Run("invalid yaml syntax", func(t *testing.T) {
		t.Parallel()
		invalidPath := filepath.Join(tmpDir, "invalid.yaml")
		if err := os.WriteFile(invalidPath, []byte(": not valid yaml: ["), 0600); err != nil {
			t.Fatalf("failed to write temp config: %v", err)
		}

		_, err := httplog.LoadConfig(invalidPath)
		if err == nil {
			t.Fatalf("expected error for invalid yaml syntax, got nil")
		}
	})
}

func TestConfig_InboundAndOutboundRules(t *testing.T) {
	t.Parallel()

	t.Run("rules mapping with inbound and outbound", func(t *testing.T) {
		t.Parallel()
		yamlData := `
version: "1.0"
default:
  mask_value: "[REDACTED]"
rules:
  inbound:
    - path: "/api/inbound/1"
      methods: ["GET"]
    - path: "/api/inbound/2"
      methods: ["POST"]
  outbound:
    - path: "/api/outbound/1"
      methods: ["POST"]
`
		cfg, err := httplog.ParseConfig([]byte(yamlData))
		if err != nil {
			t.Fatalf("ParseConfig failed: %v", err)
		}

		if len(cfg.InboundRules()) != 2 {
			t.Errorf("got %d inbound rules, want 2", len(cfg.InboundRules()))
		}
		if len(cfg.OutboundRules()) != 1 {
			t.Errorf("got %d outbound rules, want 1", len(cfg.OutboundRules()))
		}
		if len(cfg.AllRules()) != 3 {
			t.Errorf("got %d total rules, want 3", len(cfg.AllRules()))
		}
		if cfg.InboundRules()[0].Direction != "inbound" {
			t.Errorf("inbound rule direction = %q, want inbound", cfg.InboundRules()[0].Direction)
		}
		if cfg.OutboundRules()[0].Direction != "outbound" {
			t.Errorf("outbound rule direction = %q, want outbound", cfg.OutboundRules()[0].Direction)
		}
	})

	t.Run("top level inbound and outbound", func(t *testing.T) {
		t.Parallel()
		yamlData := `
version: "1.0"
inbound:
  - path: "/api/inbound"
    methods: ["GET"]
outbound:
  - path: "/api/outbound"
    methods: ["POST"]
`
		cfg, err := httplog.ParseConfig([]byte(yamlData))
		if err != nil {
			t.Fatalf("ParseConfig failed: %v", err)
		}

		if len(cfg.Inbound) != 1 || cfg.Inbound[0].Path != "/api/inbound" {
			t.Errorf("inbound mismatch: %+v", cfg.Inbound)
		}
		if len(cfg.Outbound) != 1 || cfg.Outbound[0].Path != "/api/outbound" {
			t.Errorf("outbound mismatch: %+v", cfg.Outbound)
		}
		if len(cfg.Rules) != 2 {
			t.Errorf("expected 2 total rules in cfg.Rules, got %d", len(cfg.Rules))
		}
	})

	t.Run("legacy flat rules list preserves backwards compatibility", func(t *testing.T) {
		t.Parallel()
		yamlData := `
version: "1.0"
rules:
  - path: "/legacy/1"
  - path: "/legacy/2"
`
		cfg, err := httplog.ParseConfig([]byte(yamlData))
		if err != nil {
			t.Fatalf("ParseConfig failed: %v", err)
		}

		if len(cfg.Rules) != 2 {
			t.Errorf("expected 2 rules, got %d", len(cfg.Rules))
		}
		if len(cfg.AllRules()) != 2 {
			t.Errorf("expected AllRules() = 2, got %d", len(cfg.AllRules()))
		}
	})
}
