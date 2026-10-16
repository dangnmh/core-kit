package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

type TestAppConfig struct {
	Server struct {
		Port int    `yaml:"port"`
		Host string `yaml:"host"`
	} `yaml:"server"`
	Feature struct {
		Enabled bool `yaml:"enabled"`
	} `yaml:"feature"`
}

func TestDefaultValuesWhenNoFile(t *testing.T) {
	defaults := &TestAppConfig{}
	defaults.Server.Port = 8080
	defaults.Server.Host = "127.0.0.1"

	cfg, path, err := Load[TestAppConfig](defaults, WithDefaultFiles("non_existent_file.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "" {
		t.Fatalf("expected empty path, got %q", path)
	}
	if cfg.Server.Port != 8080 {
		t.Fatalf("expected port 8080, got %d", cfg.Server.Port)
	}
	if cfg.Server.Host != "127.0.0.1" {
		t.Fatalf("expected host 127.0.0.1, got %q", cfg.Server.Host)
	}
}

func TestLoadFromEnvVar(t *testing.T) {
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "custom.yaml")
	content := `
server:
  port: 9090
  host: "0.0.0.0"
feature:
  enabled: true
`
	if err := os.WriteFile(yamlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write tmp file: %v", err)
	}

	t.Setenv("TEST_CONFIG_ENV", yamlPath)

	defaults := &TestAppConfig{}
	defaults.Server.Port = 8080

	cfg, path, err := Load[TestAppConfig](defaults, WithEnvVar("TEST_CONFIG_ENV"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != yamlPath {
		t.Fatalf("expected path %q, got %q", yamlPath, path)
	}
	if cfg.Server.Port != 9090 {
		t.Fatalf("expected port 9090, got %d", cfg.Server.Port)
	}
	if cfg.Server.Host != "0.0.0.0" {
		t.Fatalf("expected host 0.0.0.0, got %q", cfg.Server.Host)
	}
	if !cfg.Feature.Enabled {
		t.Fatalf("expected feature enabled true")
	}
}

func TestLoadInto(t *testing.T) {
	tmpDir := t.TempDir()
	yamlPath := filepath.Join(tmpDir, "test_into.yaml")
	content := `
server:
  port: 7070
`
	if err := os.WriteFile(yamlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write tmp file: %v", err)
	}

	cfg := &TestAppConfig{}
	cfg.Server.Port = 3000
	cfg.Server.Host = "localhost"

	path, err := LoadInto(cfg, WithDefaultFiles(yamlPath))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != yamlPath {
		t.Fatalf("expected path %q, got %q", yamlPath, path)
	}
	if cfg.Server.Port != 7070 {
		t.Fatalf("expected port 7070, got %d", cfg.Server.Port)
	}
	if cfg.Server.Host != "localhost" {
		t.Fatalf("expected host localhost, got %q", cfg.Server.Host)
	}
}

type ValidatedConfig struct {
	Port int    `yaml:"port" validate:"required,min=1,max=65535"`
	URL  string `yaml:"url" validate:"required"`
}

func TestValidate(t *testing.T) {
	// 1. Missing required field
	invalid := &ValidatedConfig{
		Port: 8080,
	}
	if err := Validate(invalid); err == nil {
		t.Fatalf("expected validation error for missing URL, got nil")
	}

	// 2. Port out of range
	invalidPort := &ValidatedConfig{
		Port: 99999,
		URL:  "http://example.com",
	}
	if err := Validate(invalidPort); err == nil {
		t.Fatalf("expected validation error for invalid port, got nil")
	}

	// 3. Valid struct
	valid := &ValidatedConfig{
		Port: 3020,
		URL:  "http://localhost:11434",
	}
	if err := Validate(valid); err != nil {
		t.Fatalf("unexpected validation error for valid struct: %v", err)
	}
}

type NestedSub struct {
	Level int `yaml:"level" env:"SUB_LEVEL"`
}

type NestedPtrSub struct {
	Name string `yaml:"name" env:"PTR_NAME"`
}

type EnvTestConfig struct {
	Port    int           `yaml:"port" env:"PORT"`
	Key     string        `yaml:"key" env:"KEY" validate:"required"`
	Timeout time.Duration `yaml:"timeout" env:"TIMEOUT"`
	Enabled bool          `yaml:"enabled" env:"ENABLED"`
	Rate    float64       `yaml:"rate" env:"RATE"`
	Ratio   *float32      `yaml:"ratio" env:"RATIO"`
	Tags    []string      `yaml:"tags" env:"TAGS"`
	Sub     NestedSub     `yaml:"sub"`
	PtrSub  *NestedPtrSub `yaml:"ptr_sub"`
}

func mapLookup(m map[string]string) func(string) (string, bool) {
	return func(k string) (string, bool) {
		v, ok := m[k]
		return v, ok
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "base.yaml")
	yamlContent := `
port: 8080
key: "from-yaml"
timeout: 5s
enabled: false
rate: 1.5
tags: ["default"]
sub:
  level: 1
`
	if err := os.WriteFile(yamlFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("write tmp yaml: %v", err)
	}

	ratioVal := float32(0.9)

	tests := []struct {
		name       string
		useYaml    bool
		giveEnv    map[string]string
		wantPort   int
		wantKey    string
		wantTime   time.Duration
		wantEnable bool
		wantRate   float64
		wantRatio  *float32
		wantTags   []string
		wantSubLvl int
		wantPtrSub string
		wantErr    bool
	}{
		{
			name:       "env overrides yaml scalar and duration",
			useYaml:    true,
			giveEnv:    map[string]string{"PORT": "9000", "KEY": "secret-key", "TIMEOUT": "2s"},
			wantPort:   9000,
			wantKey:    "secret-key",
			wantTime:   2 * time.Second,
			wantEnable: false,
			wantRate:   1.5,
			wantTags:   []string{"default"},
			wantSubLvl: 1,
			wantErr:    false,
		},
		{
			name:       "env overrides bool float pointer and tags slice",
			useYaml:    true,
			giveEnv:    map[string]string{"ENABLED": "true", "RATE": "3.14", "RATIO": "0.9", "TAGS": "a, b, c"},
			wantPort:   8080,
			wantKey:    "from-yaml",
			wantTime:   5 * time.Second,
			wantEnable: true,
			wantRate:   3.14,
			wantRatio:  &ratioVal,
			wantTags:   []string{"a", "b", "c"},
			wantSubLvl: 1,
			wantErr:    false,
		},
		{
			name:       "env overrides nested struct and allocates nested pointer",
			useYaml:    true,
			giveEnv:    map[string]string{"SUB_LEVEL": "42", "PTR_NAME": "nested-ptr"},
			wantPort:   8080,
			wantKey:    "from-yaml",
			wantTime:   5 * time.Second,
			wantEnable: false,
			wantRate:   1.5,
			wantTags:   []string{"default"},
			wantSubLvl: 42,
			wantPtrSub: "nested-ptr",
			wantErr:    false,
		},
		{
			name:    "bad int returns error",
			useYaml: true,
			giveEnv: map[string]string{"PORT": "invalid-int"},
			wantErr: true,
		},
		{
			name:    "bad bool returns error",
			useYaml: true,
			giveEnv: map[string]string{"ENABLED": "not-a-bool"},
			wantErr: true,
		},
		{
			name:    "bad duration returns error",
			useYaml: true,
			giveEnv: map[string]string{"TIMEOUT": "10x"},
			wantErr: true,
		},
		{
			name:    "bad float returns error",
			useYaml: true,
			giveEnv: map[string]string{"RATE": "abc"},
			wantErr: true,
		},
		{
			name:    "no file with missing required key still validates and errors",
			useYaml: false,
			giveEnv: map[string]string{"PORT": "3000"},
			wantErr: true,
		},
		{
			name:     "no file but env satisfies required key succeeds",
			useYaml:  false,
			giveEnv:  map[string]string{"KEY": "valid-key", "PORT": "3000"},
			wantPort: 3000,
			wantKey:  "valid-key",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := []Option{
				WithEnvLookup(mapLookup(tt.giveEnv)),
			}
			if tt.useYaml {
				opts = append(opts, WithFilePath(yamlFile))
			} else {
				opts = append(opts, WithDefaultFiles("non_existent_file.yaml"))
			}

			cfg, _, err := Load[EnvTestConfig](nil, opts...)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if cfg.Port != tt.wantPort {
				t.Errorf("Port = %d, want %d", cfg.Port, tt.wantPort)
			}
			if cfg.Key != tt.wantKey {
				t.Errorf("Key = %q, want %q", cfg.Key, tt.wantKey)
			}
			if tt.wantTime != 0 && cfg.Timeout != tt.wantTime {
				t.Errorf("Timeout = %v, want %v", cfg.Timeout, tt.wantTime)
			}
			if cfg.Enabled != tt.wantEnable {
				t.Errorf("Enabled = %v, want %v", cfg.Enabled, tt.wantEnable)
			}
			if tt.wantRate != 0 && cfg.Rate != tt.wantRate {
				t.Errorf("Rate = %v, want %v", cfg.Rate, tt.wantRate)
			}
			if tt.wantRatio != nil {
				if cfg.Ratio == nil || *cfg.Ratio != *tt.wantRatio {
					t.Errorf("Ratio = %v, want %v", cfg.Ratio, *tt.wantRatio)
				}
			}
			if len(tt.wantTags) > 0 {
				if len(cfg.Tags) != len(tt.wantTags) {
					t.Errorf("Tags len = %d, want %d", len(cfg.Tags), len(tt.wantTags))
				} else {
					for i := range tt.wantTags {
						if cfg.Tags[i] != tt.wantTags[i] {
							t.Errorf("Tags[%d] = %q, want %q", i, cfg.Tags[i], tt.wantTags[i])
						}
					}
				}
			}
			if tt.wantSubLvl != 0 && cfg.Sub.Level != tt.wantSubLvl {
				t.Errorf("Sub.Level = %d, want %d", cfg.Sub.Level, tt.wantSubLvl)
			}
			if tt.wantPtrSub != "" {
				if cfg.PtrSub == nil || cfg.PtrSub.Name != tt.wantPtrSub {
					t.Errorf("PtrSub.Name = %v, want %q", cfg.PtrSub, tt.wantPtrSub)
				}
			}
		})
	}
}
