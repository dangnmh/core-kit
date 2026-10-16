package lifecycle

import (
	"os"
	"syscall"
	"time"
)

// Option configures the Manager instance.
type Option func(*Manager)

// Default settings
const (
	DefaultStartupTimeout  = 15 * time.Second
	DefaultShutdownTimeout = 15 * time.Second
	DefaultHookTimeout     = 5 * time.Second
)

// WithStartupTimeout sets the maximum duration allowed for all OnStart hooks to complete.
func WithStartupTimeout(d time.Duration) Option {
	return func(m *Manager) {
		if d > 0 {
			m.startupTimeout = d
		}
	}
}

// WithShutdownTimeout sets the overall deadline for the entire phased shutdown process.
func WithShutdownTimeout(d time.Duration) Option {
	return func(m *Manager) {
		if d > 0 {
			m.shutdownTimeout = d
		}
	}
}

// WithHookTimeout sets the maximum duration allowed for an individual OnStop hook.
func WithHookTimeout(d time.Duration) Option {
	return func(m *Manager) {
		if d > 0 {
			m.hookTimeout = d
		}
	}
}

// WithSignals configures the OS signals that initiate graceful teardown.
func WithSignals(sigs ...os.Signal) Option {
	return func(m *Manager) {
		if len(sigs) > 0 {
			m.signals = sigs
		}
	}
}

// WithOnForceKill overrides the default os.Exit(1) behavior when a second signal is received.
// This is especially valuable in testing or custom teardown scenarios.
func WithOnForceKill(fn func()) Option {
	return func(m *Manager) {
		m.onForceKill = fn
	}
}

// defaultSignals returns SIGINT and SIGTERM as standard termination signals.
func defaultSignals() []os.Signal {
	return []os.Signal{os.Interrupt, syscall.SIGTERM}
}
