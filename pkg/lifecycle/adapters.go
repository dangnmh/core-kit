package lifecycle

import (
	"context"
	"errors"
	"net/http"
)

// ServerAdapter provides an in-line functional implementation of the Server interface.
type ServerAdapter struct {
	startFn func(ctx context.Context) error
	stopFn  func(ctx context.Context) error
}

// NewServerAdapter constructs a Server from functional closures.
func NewServerAdapter(
	startFn func(ctx context.Context) error,
	stopFn func(ctx context.Context) error,
) Server {
	return &ServerAdapter{
		startFn: startFn,
		stopFn:  stopFn,
	}
}

// Start executes the startup callback.
func (a *ServerAdapter) Start(ctx context.Context) error {
	if a.startFn != nil {
		return a.startFn(ctx)
	}
	return nil
}

// Stop executes the termination callback.
func (a *ServerAdapter) Stop(ctx context.Context) error {
	if a.stopFn != nil {
		return a.stopFn(ctx)
	}
	return nil
}

// HTTPServerAdapter wraps standard net/http.Server to conform to the Server interface.
type HTTPServerAdapter struct {
	server *http.Server
}

// NewHTTPServerAdapter constructs a Server wrapping an *http.Server.
func NewHTTPServerAdapter(srv *http.Server) Server {
	return &HTTPServerAdapter{server: srv}
}

// Start launches ListenAndServe, cleanly treating http.ErrServerClosed as a normal exit.
func (h *HTTPServerAdapter) Start(_ context.Context) error {
	if h.server == nil {
		return nil
	}
	err := h.server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// Stop gracefully shuts down the HTTP server with the provided context.
func (h *HTTPServerAdapter) Stop(ctx context.Context) error {
	if h.server == nil {
		return nil
	}
	return h.server.Shutdown(ctx)
}
