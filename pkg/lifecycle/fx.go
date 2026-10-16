package lifecycle

import (
	"context"
	"errors"

	"go.uber.org/fx"
)

// Module provides a shared Manager singleton wired directly to the Uber Fx lifecycle.
var Module = fx.Module("lifecycle",
	fx.Provide(func() *Manager {
		return NewManager()
	}),
	fx.Invoke(func(lc fx.Lifecycle, m *Manager) {
		m.AttachToFx(lc)
	}),
)

// AttachToFx attaches this Manager's startup and phased teardown to an Uber Fx Lifecycle.
func (m *Manager) AttachToFx(lc fx.Lifecycle) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			m.live.Store(true)
			if err := m.executeStartHooks(ctx); err != nil {
				return err
			}
			m.ready.Store(true)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			m.MarkUnready()
			defer m.live.Store(false)

			m.mu.RLock()
			servers := make([]ServerEntry, len(m.servers))
			copy(servers, m.servers)
			m.mu.RUnlock()

			errs := m.executePhasedShutdown(ctx, servers)
			if len(errs) > 0 {
				return errors.Join(errs...)
			}
			return nil
		},
	})
}
