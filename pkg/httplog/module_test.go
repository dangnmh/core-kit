package httplog_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/dangnmh/core-kit/pkg/httplog"
	"go.uber.org/fx"
)

func TestProvideEngine_FallbackDefault(t *testing.T) {
	// Change working directory to a temporary directory without http_logger.yaml
	tmpDir := t.TempDir()
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	defer func() {
		_ = os.Chdir(origWd)
	}()

	engine := httplog.ProvideEngine()
	if engine == nil {
		t.Fatal("expected non-nil Engine from ProvideEngine()")
	}

	if engine.DefaultMaskValue() != "[REDACTED]" {
		t.Errorf("DefaultMaskValue = %q, want [REDACTED]", engine.DefaultMaskValue())
	}

	if engine.MaxBodySize() != 1048576 {
		t.Errorf("MaxBodySize = %d, want 1048576", engine.MaxBodySize())
	}
}

func TestProvideEngine_WithConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	yamlContent := `version: "1.0"
default:
  mask_value: "*****"
  max_body_size: 2048
rules:
  - path: "/api/test"
    methods: ["POST"]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "http_logger.yaml"), []byte(yamlContent), 0644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	defer func() {
		_ = os.Chdir(origWd)
	}()

	engine := httplog.ProvideEngine()
	if engine == nil {
		t.Fatal("expected non-nil Engine from ProvideEngine()")
	}

	if engine.DefaultMaskValue() != "*****" {
		t.Errorf("DefaultMaskValue = %q, want *****", engine.DefaultMaskValue())
	}

	if engine.MaxBodySize() != 2048 {
		t.Errorf("MaxBodySize = %d, want 2048", engine.MaxBodySize())
	}

	rule, matched := engine.Match("POST", "/api/test")
	if !matched || rule == nil {
		t.Errorf("expected route /api/test to match")
	}
}

func TestProvideEngine_InvalidConfigFileFallback(t *testing.T) {
	tmpDir := t.TempDir()
	invalidYAML := `invalid: [unclosed`
	if err := os.WriteFile(filepath.Join(tmpDir, "http_logger.yaml"), []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	defer func() {
		_ = os.Chdir(origWd)
	}()

	engine := httplog.ProvideEngine()
	if engine == nil {
		t.Fatal("expected non-nil Engine from ProvideEngine()")
	}

	if engine.DefaultMaskValue() != "[REDACTED]" {
		t.Errorf("DefaultMaskValue = %q, want [REDACTED]", engine.DefaultMaskValue())
	}
}

func TestModule_FxWiring(t *testing.T) {
	var capturedEngine httplog.Engine

	app := fx.New(
		httplog.Module,
		fx.Invoke(func(e httplog.Engine) {
			capturedEngine = e
		}),
	)

	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		t.Fatalf("app start failed: %v", err)
	}
	defer func() {
		_ = app.Stop(ctx)
	}()

	if capturedEngine == nil {
		t.Fatal("expected httplog.Engine to be injected via Fx")
	}
}
