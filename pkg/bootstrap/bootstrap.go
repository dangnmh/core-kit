package bootstrap

import (
	"github.com/dangnmh/core-kit/pkg/logger"
)

// ExitConfig is the EX_CONFIG exit code (78) from sysexits.h indicating configuration error.
const ExitConfig = 78

// Exiter is a function type for process termination, injected to allow testing.
type Exiter func(code int)

// MustLoad executes the configuration loader function.
// If load returns an error, it logs a single structured Error and exits with ExitConfig (78).
// Returns the loaded configuration, or nil if exit was called (in tests where exit is stubbed).
func MustLoad[T any](service string, load func() (*T, error), exit Exiter) *T {
	cfg, err := load()
	if err != nil {
		logger.Error("Failed to load config", "service", service, "error", err)
		if exit != nil {
			exit(ExitConfig)
		}
		return nil
	}
	return cfg
}
