package wsclient_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dangnmh/core-kit/pkg/wsclient"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func TestWSClientConnectAndReceive(t *testing.T) {
	t.Parallel()
	var receivedMsg string
	var mu sync.Mutex
	done := make(chan struct{})

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("upgrade error: %v", err)
			return
		}
		defer func() { _ = conn.Close() }()

		_ = conn.WriteMessage(websocket.TextMessage, []byte("hello from server"))

		time.Sleep(100 * time.Millisecond)
	}))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")

	client := wsclient.New(wsURL, func(payload []byte) {
		mu.Lock()
		receivedMsg = string(payload)
		mu.Unlock()
		select {
		case <-done:
		default:
			close(done)
		}
	}, wsclient.WithReconnectWait(50*time.Millisecond, 200*time.Millisecond))

	client.Start()
	defer client.Close()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for message")
	}

	mu.Lock()
	defer mu.Unlock()
	if receivedMsg != "hello from server" {
		t.Fatalf("expected 'hello from server', got %q", receivedMsg)
	}
}

func TestClient_ConcurrentSendAndPing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		give int // number of Send goroutines
		want int // total expected messages received by server
	}{
		{name: "single writer", give: 1, want: 100},
		{name: "many writers with 1ms ping", give: 16, want: 1600},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var serverReceived atomic.Int64
			allReceived := make(chan struct{})

			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				conn, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					return
				}
				defer func() { _ = conn.Close() }()

				for {
					_, _, err := conn.ReadMessage()
					if err != nil {
						return
					}
					if int(serverReceived.Add(1)) == tt.want {
						close(allReceived)
					}
				}
			}))
			defer ts.Close()

			wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")

			connected := make(chan struct{})
			client := wsclient.New(
				wsURL,
				nil,
				wsclient.WithHeartbeat(1*time.Millisecond, 2*time.Second),
				wsclient.WithConnectHooks(func() {
					select {
					case <-connected:
					default:
						close(connected)
					}
				}, nil),
			)

			client.Start()
			defer client.Close()

			select {
			case <-connected:
			case <-time.After(3 * time.Second):
				t.Fatal("timed out waiting for client to connect")
			}

			var wg sync.WaitGroup
			perGoroutine := tt.want / tt.give

			for i := 0; i < tt.give; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := 0; j < perGoroutine; j++ {
						if err := client.Send([]byte("msg")); err != nil {
							t.Errorf("Send error: %v", err)
							return
						}
					}
				}()
			}

			wg.Wait()

			select {
			case <-allReceived:
			case <-time.After(5 * time.Second):
				t.Fatalf("timed out waiting for all messages, got %d, want %d", serverReceived.Load(), tt.want)
			}

			if got := int(serverReceived.Load()); got != tt.want {
				t.Fatalf("got %d messages, want %d", got, tt.want)
			}
		})
	}
}

func TestClient_OnDisconnectedReceivesReadError(t *testing.T) {
	t.Parallel()

	disconnectedErr := make(chan error, 1)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		_ = conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(4001, "custom disconnect"),
			time.Now().Add(time.Second),
		)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")

	client := wsclient.New(
		wsURL,
		nil,
		wsclient.WithReconnectWait(100*time.Millisecond, 200*time.Millisecond),
		wsclient.WithConnectHooks(nil, func(err error) {
			select {
			case disconnectedErr <- err:
			default:
			}
		}),
	)

	client.Start()
	defer client.Close()

	select {
	case err := <-disconnectedErr:
		if err == nil {
			t.Fatal("expected non-nil error in OnDisconnected")
		}
		var closeErr *websocket.CloseError
		if !errors.As(err, &closeErr) {
			t.Fatalf("expected *websocket.CloseError, got %T: %v", err, err)
		}
		if closeErr.Code != 4001 {
			t.Fatalf("expected close code 4001, got %d", closeErr.Code)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for OnDisconnected callback")
	}
}

func TestClient_CloseIdempotent(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")

	connected := make(chan struct{})
	client := wsclient.New(
		wsURL,
		nil,
		wsclient.WithConnectHooks(func() {
			select {
			case <-connected:
			default:
				close(connected)
			}
		}, nil),
	)

	client.Start()

	select {
	case <-connected:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for client to connect")
	}

	// Concurrently call Close
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client.Close()
		}()
	}
	wg.Wait()

	// Calling Close again sequentially must also be safe
	client.Close()

	// Send after close must return ErrNotConnected
	err := client.Send([]byte("after close"))
	if !errors.Is(err, wsclient.ErrNotConnected) {
		t.Fatalf("expected ErrNotConnected, got %v", err)
	}
}

func TestClient_ReadLimitExceeded(t *testing.T) {
	t.Parallel()

	disconnectedErr := make(chan error, 1)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		// Send 2048 bytes
		largePayload := make([]byte, 2048)
		_ = conn.WriteMessage(websocket.TextMessage, largePayload)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http")

	// Set ReadLimit to 1024 bytes (1 KiB)
	client := wsclient.New(
		wsURL,
		nil,
		wsclient.WithReadLimit(1024),
		wsclient.WithReconnectWait(500*time.Millisecond, 1*time.Second),
		wsclient.WithConnectHooks(nil, func(err error) {
			select {
			case disconnectedErr <- err:
			default:
			}
		}),
	)

	client.Start()
	defer client.Close()

	select {
	case err := <-disconnectedErr:
		if err == nil {
			t.Fatal("expected non-nil error when read limit is exceeded")
		}
		if !errors.Is(err, websocket.ErrReadLimit) && !strings.Contains(err.Error(), "read limit") {
			t.Fatalf("expected ErrReadLimit, got %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for read limit disconnect")
	}
}
