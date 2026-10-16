package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dangnmh/core-kit/pkg/logger"
)

// Manager orchestrates multi-server startup, health status, signal handling,
// and phased reverse-order graceful shutdown.
type Manager struct {
	mu              sync.RWMutex
	startupTimeout  time.Duration
	shutdownTimeout time.Duration
	hookTimeout     time.Duration
	signals         []os.Signal
	onForceKill     func()

	servers    []ServerEntry
	startHooks []Hook
	stopHooks  map[Phase][]Hook

	ready atomic.Bool
	live  atomic.Bool
}

// NewManager creates an initialized lifecycle Manager with sensible defaults.
func NewManager(opts ...Option) *Manager {
	m := &Manager{
		startupTimeout:  DefaultStartupTimeout,
		shutdownTimeout: DefaultShutdownTimeout,
		hookTimeout:     DefaultHookTimeout,
		signals:         defaultSignals(),
		stopHooks:       make(map[Phase][]Hook),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(m)
		}
	}

	return m
}

// RegisterServer registers a Server instance to be started concurrently.
func (m *Manager) RegisterServer(name string, s Server) {
	if s == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.servers = append(m.servers, ServerEntry{
		Name:   name,
		Server: s,
	})
}

// RegisterTyped registers any concrete service type T that implements Server.
func (m *Manager) RegisterTyped[T Server](name string, s T) {
	m.RegisterServer(name, s)
}

// Register registers any concrete service type T that implements Server into Manager.
func Register[T Server](m *Manager, name string, s T) {
	m.RegisterServer(name, s)
}

// RegisterHTTPServer registers a standard *http.Server instance.
func (m *Manager) RegisterHTTPServer(name string, srv *http.Server) {
	if srv == nil {
		return
	}
	m.RegisterServer(name, NewHTTPServerAdapter(srv))
}

// OnStart registers a hook to be executed sequentially during startup before servers launch.
func (m *Manager) OnStart(name string, fn HookFunc) {
	if fn == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startHooks = append(m.startHooks, Hook{
		Name:    name,
		Timeout: m.hookTimeout,
		Fn:      fn,
	})
}

// OnStop registers a hook to be executed during the specified shutdown phase.
// IMPORTANT: Within each phase, hooks run in reverse order of registration (LIFO).
// If components within the same phase have strict forward dependencies (e.g. worker drain
// before orchestrator close before websocket stop), consolidate them into a single hook
// function to make the sequential execution order explicit and deterministic.
func (m *Manager) OnStop(phase Phase, name string, fn HookFunc) {
	if fn == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopHooks[phase] = append(m.stopHooks[phase], Hook{
		Name:    name,
		Phase:   phase,
		Timeout: m.hookTimeout,
		Fn:      fn,
	})
}

// IsReady returns true when startup hooks and all servers have launched successfully.
func (m *Manager) IsReady() bool {
	return m.ready.Load()
}

// IsLive returns true while the manager is active (from Run start to shutdown completion).
func (m *Manager) IsLive() bool {
	return m.live.Load()
}

// MarkUnready explicitly sets readiness to false (useful for manual traffic shedding).
func (m *Manager) MarkUnready() {
	m.ready.Store(false)
}

// Run executes startup hooks, launches servers concurrently, traps OS signals,
// and coordinates phased graceful shutdown upon cancellation or termination.
func (m *Manager) Run(ctx context.Context) error {
	m.live.Store(true)
	defer m.live.Store(false)

	// 1. Setup Signal Trapping (buffer size 2 to support force-kill on 2nd signal)
	sigChan := make(chan os.Signal, 2)
	signal.Notify(sigChan, m.signals...)
	defer signal.Stop(sigChan)

	// 2. Execute OnStart hooks
	startCtx, startCancel := context.WithTimeout(ctx, m.startupTimeout)
	err := m.executeStartHooks(startCtx)
	startCancel()
	if err != nil {
		logger.Error("Startup hooks failed, aborting lifecycle launch", "error", err)
		return fmt.Errorf("execute start hooks: %w", err)
	}

	// 3. Start Registered Servers Concurrently
	m.mu.RLock()
	servers := make([]ServerEntry, len(m.servers))
	copy(servers, m.servers)
	m.mu.RUnlock()

	serverErrChan := make(chan error, len(servers))
	runCtx, runCancel := context.WithCancel(ctx)
	defer runCancel()

	for _, entry := range servers {
		go func(se ServerEntry) {
			defer func() {
				if r := recover(); r != nil {
					serverErrChan <- fmt.Errorf("server %q panicked: %v", se.Name, r)
				}
			}()

			logger.Info("Starting server", "server", se.Name)
			if err := se.Server.Start(runCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
				serverErrChan <- fmt.Errorf("server %q exited with error: %w", se.Name, err)
			}
		}(entry)
	}

	// 4. Mark Ready
	m.ready.Store(true)
	logger.Info("Lifecycle manager running and ready", "servers_count", len(servers))

	// 5. Wait for termination trigger (Signal, Context Cancel, or Server Crash)
	var triggerErr error
	select {
	case <-ctx.Done():
		logger.Info("Lifecycle context cancelled, initiating shutdown")
		triggerErr = ctx.Err()
	case sig := <-sigChan:
		logger.Info("Shutdown signal caught, initiating graceful teardown", "signal", sig.String())
	case err := <-serverErrChan:
		logger.Error("Server error detected, initiating fail-fast shutdown", "error", err)
		triggerErr = err
	}

	// 6. Traffic Shedding: Immediately mark unready
	m.ready.Store(false)
	runCancel()

	// 7. Setup Second-Signal Force Kill Listener
	shutdownDone := make(chan struct{})
	defer close(shutdownDone)

	go func() {
		select {
		case sig2 := <-sigChan:
			logger.Warn("Force shutdown signal received, terminating immediately", "signal", sig2.String())
			if m.onForceKill != nil {
				m.onForceKill()
			} else {
				os.Exit(1)
			}
		case <-shutdownDone:
			return
		}
	}()

	// 8. Execute Phased Graceful Shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), m.shutdownTimeout)
	defer shutdownCancel()

	shutdownErrs := m.executePhasedShutdown(shutdownCtx, servers)
	if len(shutdownErrs) > 0 {
		logger.Error("Errors encountered during shutdown", "errors_count", len(shutdownErrs))
	} else {
		logger.Info("Graceful shutdown completed successfully")
	}

	return errors.Join(triggerErr, errors.Join(shutdownErrs...))
}

// executeStartHooks executes registered OnStart hooks sequentially.
func (m *Manager) executeStartHooks(ctx context.Context) error {
	m.mu.RLock()
	hooks := make([]Hook, len(m.startHooks))
	copy(hooks, m.startHooks)
	m.mu.RUnlock()

	for _, h := range hooks {
		logger.Info("Executing OnStart hook", "name", h.Name)
		hookTimeout := h.Timeout
		if hookTimeout <= 0 {
			hookTimeout = m.hookTimeout
		}

		hookCtx, cancel := context.WithTimeout(ctx, hookTimeout)
		err := executeHookSafely(hookCtx, h)
		cancel()

		if err != nil {
			return fmt.Errorf("start hook %q: %w", h.Name, err)
		}
	}
	return nil
}

// executePhasedShutdown sequentially executes shutdown phases:
// PreStop -> StopIngress -> Drain -> StopEgress (including servers) -> PostStop.
func (m *Manager) executePhasedShutdown(ctx context.Context, servers []ServerEntry) []error {
	phases := []Phase{
		PhasePreStop,
		PhaseStopIngress,
		PhaseDrain,
		PhaseStopEgress,
		PhasePostStop,
	}

	var allErrs []error

	for _, phase := range phases {
		logger.Info("Executing shutdown phase", "phase", phase.String())

		// In PhaseStopEgress, stop all servers concurrently
		if phase == PhaseStopEgress && len(servers) > 0 {
			serverErrs := m.stopServers(ctx, servers)
			allErrs = append(allErrs, serverErrs...)
		}

		// Execute registered hooks for this phase in LIFO order
		phaseErrs := m.executePhaseHooks(ctx, phase)
		allErrs = append(allErrs, phaseErrs...)
	}

	return allErrs
}

// stopServers stops all servers concurrently with a per-server timeout context.
func (m *Manager) stopServers(ctx context.Context, servers []ServerEntry) []error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error

	for _, entry := range servers {
		wg.Add(1)
		go func(se ServerEntry) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					mu.Lock()
					errs = append(errs, fmt.Errorf("server %q stop panicked: %v", se.Name, r))
					mu.Unlock()
				}
			}()

			logger.Info("Stopping server", "server", se.Name)
			stopTimeout := m.hookTimeout
			if stopTimeout <= 0 {
				stopTimeout = DefaultHookTimeout
			}

			stopCtx, cancel := context.WithTimeout(ctx, stopTimeout)
			defer cancel()

			if err := se.Server.Stop(stopCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
				mu.Lock()
				errs = append(errs, fmt.Errorf("stop server %q: %w", se.Name, err))
				mu.Unlock()
			}
		}(entry)
	}

	wg.Wait()
	return errs
}

// executePhaseHooks executes hooks registered for a phase in LIFO order.
func (m *Manager) executePhaseHooks(ctx context.Context, phase Phase) []error {
	m.mu.RLock()
	rawHooks := m.stopHooks[phase]
	hooks := make([]Hook, len(rawHooks))
	copy(hooks, rawHooks)
	m.mu.RUnlock()

	var errs []error

	// Execute LIFO (reverse order of registration)
	for i := len(hooks) - 1; i >= 0; i-- {
		h := hooks[i]
		logger.Info("Executing OnStop hook", "phase", phase.String(), "name", h.Name)

		timeout := h.Timeout
		if timeout <= 0 {
			timeout = m.hookTimeout
		}

		hookCtx, cancel := context.WithTimeout(ctx, timeout)
		err := executeHookSafely(hookCtx, h)
		cancel()

		if err != nil {
			logger.Error("Error executing OnStop hook", "phase", phase.String(), "name", h.Name, "error", err)
			errs = append(errs, fmt.Errorf("hook %q [%s]: %w", h.Name, phase.String(), err))
		}
	}

	return errs
}

// executeHookSafely runs a hook function safely with panic recovery.
func executeHookSafely(ctx context.Context, h Hook) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("hook panicked: %v", r)
		}
	}()

	done := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Errorf("hook panicked: %v", r)
			}
		}()
		done <- h.Fn(ctx)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err = <-done:
		return err
	}
}
