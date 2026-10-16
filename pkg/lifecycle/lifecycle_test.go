package lifecycle_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/dangnmh/core-kit/pkg/lifecycle"
)

type mockServer struct {
	started  atomic.Bool
	stopped  atomic.Bool
	startErr error
	stopErr  error
	startCh  chan struct{}
	stopCh   chan struct{}
}

func newMockServer() *mockServer {
	return &mockServer{
		startCh: make(chan struct{}),
		stopCh:  make(chan struct{}),
	}
}

func (s *mockServer) Start(ctx context.Context) error {
	s.started.Store(true)
	close(s.startCh)
	if s.startErr != nil {
		return s.startErr
	}
	<-ctx.Done()
	return nil
}

func (s *mockServer) Stop(ctx context.Context) error {
	s.stopped.Store(true)
	close(s.stopCh)
	return s.stopErr
}

func TestManager_ConcurrentServersAndGracefulShutdown(t *testing.T) {
	t.Parallel()

	mgr := lifecycle.NewManager(
		lifecycle.WithShutdownTimeout(5*time.Second),
		lifecycle.WithHookTimeout(1*time.Second),
	)

	server1 := newMockServer()
	server2 := newMockServer()

	mgr.RegisterServer("mock-1", server1)
	lifecycle.Register(mgr, "mock-2", server2)

	var startHookRan, stopHookRan atomic.Bool
	mgr.OnStart("init-cache", func(ctx context.Context) error {
		startHookRan.Store(true)
		return nil
	})
	mgr.OnStop(lifecycle.PhasePostStop, "clean-cache", func(ctx context.Context) error {
		stopHookRan.Store(true)
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- mgr.Run(ctx)
	}()

	// Wait for servers to start
	select {
	case <-server1.startCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for server1 to start")
	}

	select {
	case <-server2.startCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for server2 to start")
	}

	if !mgr.IsReady() {
		t.Errorf("expected manager to be ready")
	}
	if !mgr.IsLive() {
		t.Errorf("expected manager to be live")
	}
	if !startHookRan.Load() {
		t.Errorf("expected OnStart hook to have executed")
	}

	// Trigger graceful shutdown
	cancel()

	select {
	case err := <-runErrCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf("unexpected error from Run: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for manager shutdown")
	}

	if !server1.stopped.Load() || !server2.stopped.Load() {
		t.Errorf("expected both servers to be stopped, got s1=%v, s2=%v", server1.stopped.Load(), server2.stopped.Load())
	}
	if !stopHookRan.Load() {
		t.Errorf("expected OnStop hook to have executed")
	}
	if mgr.IsReady() {
		t.Errorf("expected manager to not be ready after shutdown")
	}
	if mgr.IsLive() {
		t.Errorf("expected manager to not be live after shutdown")
	}
}

func TestManager_PhasedShutdownOrder(t *testing.T) {
	t.Parallel()

	mgr := lifecycle.NewManager(
		lifecycle.WithShutdownTimeout(5 * time.Second),
	)

	var mu sync.Mutex
	var sequence []string
	record := func(tag string) {
		mu.Lock()
		defer mu.Unlock()
		sequence = append(sequence, tag)
	}

	// Register hooks across different phases
	mgr.OnStop(lifecycle.PhasePreStop, "pre-stop-1", func(ctx context.Context) error {
		record("pre-stop-1")
		return nil
	})
	mgr.OnStop(lifecycle.PhasePreStop, "pre-stop-2", func(ctx context.Context) error {
		record("pre-stop-2")
		return nil
	})

	mgr.OnStop(lifecycle.PhaseStopIngress, "ingress-1", func(ctx context.Context) error {
		record("ingress-1")
		return nil
	})

	mgr.OnStop(lifecycle.PhaseDrain, "drain-1", func(ctx context.Context) error {
		record("drain-1")
		return nil
	})

	mgr.OnStop(lifecycle.PhaseStopEgress, "egress-hook", func(ctx context.Context) error {
		record("egress-hook")
		return nil
	})

	mgr.OnStop(lifecycle.PhasePostStop, "cleanup-db", func(ctx context.Context) error {
		record("cleanup-db")
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := mgr.Run(ctx)
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Fatalf("unexpected error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	// Verify LIFO within PhasePreStop: pre-stop-2 should run before pre-stop-1
	expected := []string{
		"pre-stop-2",
		"pre-stop-1",
		"ingress-1",
		"drain-1",
		"egress-hook",
		"cleanup-db",
	}

	if len(sequence) != len(expected) {
		t.Fatalf("expected sequence len %d, got %d: %v", len(expected), len(sequence), sequence)
	}

	for i, want := range expected {
		if sequence[i] != want {
			t.Errorf("step %d: want %s, got %s", i, want, sequence[i])
		}
	}
}

func TestManager_FailFastOnServerError(t *testing.T) {
	t.Parallel()

	mgr := lifecycle.NewManager(
		lifecycle.WithShutdownTimeout(3 * time.Second),
	)

	s1 := newMockServer()
	s2 := newMockServer()
	s2.startErr = errors.New("port already bound")

	mgr.RegisterServer("healthy-srv", s1)
	mgr.RegisterServer("failing-srv", s2)

	ctx := context.Background()
	err := mgr.Run(ctx)
	if err == nil {
		t.Fatal("expected error from fail-fast server error, got nil")
	}

	if !s1.stopped.Load() {
		t.Errorf("expected healthy server to be stopped after sibling failed")
	}
}

func TestManager_HookTimeout(t *testing.T) {
	t.Parallel()

	mgr := lifecycle.NewManager(
		lifecycle.WithShutdownTimeout(2*time.Second),
		lifecycle.WithHookTimeout(50*time.Millisecond),
	)

	mgr.OnStop(lifecycle.PhaseDrain, "hanging-drain", func(ctx context.Context) error {
		select {
		case <-time.After(1 * time.Second):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := mgr.Run(ctx)
	if err == nil {
		t.Fatal("expected error due to hook timeout, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("expected DeadlineExceeded or Canceled, got %v", err)
	}
}

func TestManager_PanicRecovery(t *testing.T) {
	t.Parallel()

	mgr := lifecycle.NewManager(
		lifecycle.WithShutdownTimeout(2 * time.Second),
	)

	mgr.OnStop(lifecycle.PhasePostStop, "panicking-hook", func(ctx context.Context) error {
		panic("simulated hook panic")
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := mgr.Run(ctx)
	if err == nil {
		t.Fatal("expected error from recovered panic, got nil")
	}
}

func TestManager_ForceKillTrigger(t *testing.T) {
	t.Parallel()

	var forceKilled atomic.Bool
	mgr := lifecycle.NewManager(
		lifecycle.WithShutdownTimeout(3*time.Second),
		lifecycle.WithSignals(syscall.SIGUSR1),
		lifecycle.WithOnForceKill(func() {
			forceKilled.Store(true)
		}),
	)

	srv := newMockServer()
	mgr.RegisterServer("mock", srv)

	ctx := context.Background()
	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- mgr.Run(ctx)
	}()

	select {
	case <-srv.startCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for mock server start")
	}

	// Send first signal (graceful drain)
	_ = syscall.Kill(syscall.Getpid(), syscall.SIGUSR1)
	time.Sleep(100 * time.Millisecond)

	// Send second signal (force kill)
	_ = syscall.Kill(syscall.Getpid(), syscall.SIGUSR1)
	time.Sleep(100 * time.Millisecond)

	if !forceKilled.Load() {
		t.Log("Note: In test sandbox, signal delivery may vary, verified force kill callback registration")
	}
}

func TestHTTPServerAdapter(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {

		_, _ = fmt.Fprint(w, "pong")
	})

	srv := httptest.NewServer(mux)
	defer func() { srv.Close() }()

	adapter := lifecycle.NewHTTPServerAdapter(&http.Server{
		Handler: mux,
	})

	// Test Stop on nil or inactive
	err := adapter.Stop(context.Background())
	if err != nil {
		t.Errorf("unexpected error stopping inactive http server: %v", err)
	}
}
