package httplog

import (
	"os"
	"sync"

	"github.com/dangnmh/core-kit/pkg/logger"
	"go.uber.org/fx"
)

var (
	defaultEngineMu sync.RWMutex
	defaultEngine   Engine
)

// SetDefault sets the global default httplog Engine.
func SetDefault(eng Engine) {
	defaultEngineMu.Lock()
	defer defaultEngineMu.Unlock()
	defaultEngine = eng
}

// Default returns the global default httplog Engine, or nil if not set.
func Default() Engine {
	defaultEngineMu.RLock()
	defer defaultEngineMu.RUnlock()
	return defaultEngine
}

// ProvideEngine loads "http_logger.yaml" from the working directory.
// If the file does not exist or fails to parse, it falls back to a safe default Config,
// returning an initialized Engine.
func ProvideEngine() Engine {
	configPath := "http_logger.yaml"
	if _, err := os.Stat(configPath); err != nil {
		// Default fallback config
		cfg := &Config{
			Default: GlobalConfig{
				MaskValue:   "[REDACTED]",
				MaxBodySize: 1048576,
			},
		}
		eng, _ := NewEngine(cfg)
		SetDefault(eng)
		return eng
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		logger.Error("Failed to load http_logger.yaml, using default", "error", err)
		cfg = &Config{
			Default: GlobalConfig{
				MaskValue:   "[REDACTED]",
				MaxBodySize: 1048576,
			},
		}
	}

	eng, err := NewEngine(cfg)
	if err != nil {
		logger.Error("Failed to build httplog engine, using default", "error", err)
		eng, _ = NewEngine(&Config{
			Default: GlobalConfig{
				MaskValue:   "[REDACTED]",
				MaxBodySize: 1048576,
			},
		})
	}
	SetDefault(eng)
	return eng
}

// Module provides the httplog Engine singleton to the Uber Fx container.
var Module = fx.Module("httplog",
	fx.Provide(ProvideEngine),
)
