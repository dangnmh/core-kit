package meili

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdapter_LifecycleAndHealth(t *testing.T) {
	var nilAdapter *Adapter
	if nilAdapter.IsEnabled() {
		t.Fatal("expected nil adapter to be disabled")
	}
	if nilAdapter.Client() != nil || nilAdapter.Index() != nil {
		t.Fatal("expected nil client/index on nil adapter")
	}

	disabled := NewAdapter("", "", "", nil)
	if disabled.IsEnabled() {
		t.Fatal("expected empty host adapter to be disabled")
	}
	if disabled.IsHealthy(context.Background()) {
		t.Fatal("expected disabled adapter to be unhealthy")
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"available"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	adapter := NewAdapter(srv.URL, "master-key", "", srv.Client())
	if !adapter.IsEnabled() {
		t.Fatal("expected adapter with host to be enabled")
	}
	if !adapter.IsHealthy(context.Background()) {
		t.Fatal("expected adapter to report healthy against mock /health endpoint")
	}
}
