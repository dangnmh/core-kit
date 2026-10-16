package lifecycle

import (
	"context"
	"fmt"
	"time"
)

// Phase defines the sequential shutdown phase.
type Phase int

const (
	// PhasePreStop: Traffic shedding, mark readiness unready, stop discovery.
	PhasePreStop Phase = iota

	// PhaseStopIngress: Disconnect incoming sources (e.g. Minecraft bot client, webhook listeners).
	PhaseStopIngress

	// PhaseDrain: Flush and drain internal queues, ring buffers, batchers.
	PhaseDrain

	// PhaseStopEgress: Stop listening servers (WebSocket, HTTP) and close subscriber connections.
	PhaseStopEgress

	// PhasePostStop: Release resources, flush file tracers, close DB connections.
	PhasePostStop
)

// String returns the human-readable representation of the Phase.
func (p Phase) String() string {
	switch p {
	case PhasePreStop:
		return "PhasePreStop"
	case PhaseStopIngress:
		return "PhaseStopIngress"
	case PhaseDrain:
		return "PhaseDrain"
	case PhaseStopEgress:
		return "PhaseStopEgress"
	case PhasePostStop:
		return "PhasePostStop"
	default:
		return fmt.Sprintf("Phase(%d)", int(p))
	}
}

// Server represents a long-running service with concurrent Start and graceful Stop.
type Server interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// HookFunc is a lifecycle callback function executed during startup or phased shutdown.
type HookFunc func(ctx context.Context) error

// Hook holds metadata and callback logic for a registered lifecycle hook.
type Hook struct {
	Name    string
	Phase   Phase
	Timeout time.Duration
	Fn      HookFunc
}

// ServerEntry holds a registered server along with its descriptive name.
type ServerEntry struct {
	Name   string
	Server Server
}
