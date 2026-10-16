package wsserver_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dangnmh/core-kit/pkg/wsserver"
	"github.com/gorilla/websocket"
)

func TestWSServer_BroadcastAndClientCount(t *testing.T) {
	t.Parallel()

	wsHub := wsserver.NewServer(
		wsserver.WithWriteWait(2*time.Second),
		wsserver.WithPongWait(2*time.Second),
	)
	defer func() { _ = wsHub.Stop() }()

	server := httptest.NewServer(http.HandlerFunc(wsHub.ServeHTTP))
	defer func() { server.Close() }()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	dialer := websocket.Dialer{HandshakeTimeout: 2 * time.Second}
	c1, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial c1: %v", err)
	}
	defer func() { _ = c1.Close() }()

	c2, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial c2: %v", err)
	}
	defer func() { _ = c2.Close() }()

	// Wait for connections to register
	for i := 0; i < 50; i++ {
		if wsHub.ClientCount() == 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	count := wsHub.ClientCount()
	if count != 2 {
		t.Fatalf("expected 2 clients, got %d", count)
	}

	payload := []byte(`{"type":"ping","data":"hello"}`)
	delivered := wsHub.Broadcast(payload)
	if delivered != 2 {
		t.Fatalf("expected delivered=2, got %d", delivered)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	checkClient := func(conn *websocket.Conn) {
		defer wg.Done()
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, msg, err := conn.ReadMessage()
		if err != nil {
			t.Errorf("read error: %v", err)
			return
		}
		if string(msg) != string(payload) {
			t.Errorf("expected %s, got %s", string(payload), string(msg))
		}
	}

	go checkClient(c1)
	go checkClient(c2)

	wg.Wait()

	// Disconnect c1
	_ = c1.Close()
	for i := 0; i < 50; i++ {
		if wsHub.ClientCount() == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	count = wsHub.ClientCount()
	if count != 1 {
		t.Fatalf("expected 1 client after c1 close, got %d", count)
	}
}

func TestWSServer_Stop(t *testing.T) {
	t.Parallel()

	wsHub := wsserver.NewServer()
	server := httptest.NewServer(http.HandlerFunc(wsHub.ServeHTTP))
	defer func() { server.Close() }()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	dialer := websocket.Dialer{HandshakeTimeout: 2 * time.Second}
	c, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer func() { _ = c.Close() }()

	for i := 0; i < 50; i++ {
		if wsHub.ClientCount() == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if err := wsHub.Stop(); err != nil {
		t.Fatalf("stop error: %v", err)
	}

	if wsHub.ClientCount() != 0 {
		t.Fatalf("expected 0 clients after stop, got %d", wsHub.ClientCount())
	}
}

func TestWSServer_StopWithReentrantOnDisconnect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		giveClients int
		wantCount   int
	}{
		{name: "single client reentrant ClientCount and Broadcast", giveClients: 1, wantCount: 0},
		{name: "multiple clients reentrant callbacks", giveClients: 4, wantCount: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var disconnectCalls atomic.Int32
			var observedCount atomic.Int32
			observedCount.Store(-1)

			var hub *wsserver.Server
			hub = wsserver.NewServer(
				wsserver.WithOnDisconnect(func(conn *websocket.Conn) {
					cnt := hub.ClientCount()
					observedCount.Store(int32(cnt))
					_ = hub.Broadcast([]byte(`{"type":"bye"}`))
					disconnectCalls.Add(1)
				}),
			)

			server := httptest.NewServer(http.HandlerFunc(hub.ServeHTTP))
			defer func() { server.Close() }()

			wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
			dialer := websocket.Dialer{HandshakeTimeout: 2 * time.Second}

			conns := make([]*websocket.Conn, 0, tt.giveClients)
			for i := 0; i < tt.giveClients; i++ {
				c, _, err := dialer.Dial(wsURL, nil)
				if err != nil {
					t.Fatalf("dial %d failed: %v", i, err)
				}
				conns = append(conns, c)
			}
			defer func() {
				for _, c := range conns {
					_ = c.Close()
				}
			}()

			for i := 0; i < 50; i++ {
				if hub.ClientCount() == tt.giveClients {
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			if hub.ClientCount() != tt.giveClients {
				t.Fatalf("expected %d clients, got %d", tt.giveClients, hub.ClientCount())
			}

			done := make(chan error, 1)
			go func() {
				done <- hub.Stop()
			}()

			select {
			case err := <-done:
				if err != nil {
					t.Fatalf("stop error: %v", err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("hub.Stop() deadlocked with reentrant onDisconnect callback")
			}

			if int(disconnectCalls.Load()) != tt.giveClients {
				t.Fatalf("expected %d disconnect calls, got %d", tt.giveClients, disconnectCalls.Load())
			}
			if int(observedCount.Load()) != tt.wantCount {
				t.Fatalf("expected ClientCount()=%d inside onDisconnect, got %d", tt.wantCount, observedCount.Load())
			}
		})
	}
}

func TestServer_SlowClientDoesNotBlockOthers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		give    int // number of broadcast messages
		want    int // fast client receives all
		wantErr bool
	}{
		{name: "one stalled client", give: 500, want: 500},
		{name: "burst exceeds slow queue", give: 2000, want: 2000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			disconnectCh := make(chan struct{}, 10)
			wsHub := wsserver.NewServer(
				wsserver.WithSendBuffer(64),
				wsserver.WithWriteWait(500*time.Millisecond),
				wsserver.WithOnDisconnect(func(conn *websocket.Conn) {
					select {
					case disconnectCh <- struct{}{}:
					default:
					}
				}),
			)
			defer func() { _ = wsHub.Stop() }()

			server := httptest.NewServer(http.HandlerFunc(wsHub.ServeHTTP))
			defer func() { server.Close() }()

			wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
			dialer := websocket.Dialer{HandshakeTimeout: 2 * time.Second}

			// Client A: Stalled client that never reads from socket
			stalledConn, _, err := dialer.Dial(wsURL, nil)
			if err != nil {
				t.Fatalf("stalled client dial failed: %v", err)
			}
			defer func() { _ = stalledConn.Close() }()
			if tcpConn, ok := stalledConn.UnderlyingConn().(*net.TCPConn); ok {
				_ = tcpConn.SetReadBuffer(2048)
			}

			// Client B: Fast client that continuously drains
			fastConn, _, err := dialer.Dial(wsURL, nil)
			if err != nil {
				t.Fatalf("fast client dial failed: %v", err)
			}
			defer func() { _ = fastConn.Close() }()

			// Wait for both clients to register
			for i := 0; i < 50; i++ {
				if wsHub.ClientCount() == 2 {
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			if wsHub.ClientCount() != 2 {
				t.Fatalf("expected 2 clients connected, got %d", wsHub.ClientCount())
			}

			var fastReceived atomic.Int64
			doneReading := make(chan struct{})

			// Fast reader loop
			go func() {
				defer close(doneReading)
				for {
					_ = fastConn.SetReadDeadline(time.Now().Add(5 * time.Second))
					_, _, err := fastConn.ReadMessage()
					if err != nil {
						return
					}
					if fastReceived.Add(1) == int64(tt.want) {
						return
					}
				}
			}()

			// Broadcast 16KB messages with realistic stream pacing
			payload := make([]byte, 16384)
			for i := 0; i < tt.give; i++ {
				wsHub.Broadcast(payload)
				time.Sleep(100 * time.Microsecond)
			}

			// Wait for fast client to finish reading
			select {
			case <-doneReading:
			case <-time.After(5 * time.Second):
				t.Fatalf("timeout waiting for fast client: received %d/%d", fastReceived.Load(), tt.want)
			}

			if int(fastReceived.Load()) != tt.want {
				t.Fatalf("fast client received %d, want %d", fastReceived.Load(), tt.want)
			}

			// Wait for stalled client eviction
			select {
			case <-disconnectCh:
			case <-time.After(2 * time.Second):
			}

			// Stalled client should have been evicted (ClientCount == 1)
			for i := 0; i < 50; i++ {
				if wsHub.ClientCount() == 1 {
					break
				}
				time.Sleep(10 * time.Millisecond)
			}

			if wsHub.ClientCount() != 1 {
				t.Fatalf("expected 1 client after evicting stalled client, got %d", wsHub.ClientCount())
			}
		})
	}
}

func TestServer_CheckOrigin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		origins    []string
		reqOrigin  string
		wantStatus int // 101 for switch protocols, 403 for forbidden
	}{
		{
			name:       "allowlist match exactly",
			origins:    []string{"http://localhost:5173", "http://example.com"},
			reqOrigin:  "http://localhost:5173",
			wantStatus: http.StatusSwitchingProtocols,
		},
		{
			name:       "allowlist match case insensitive",
			origins:    []string{"http://localhost:5173"},
			reqOrigin:  "HTTP://LOCALHOST:5173",
			wantStatus: http.StatusSwitchingProtocols,
		},
		{
			name:       "allowlist mismatch",
			origins:    []string{"http://localhost:5173"},
			reqOrigin:  "http://evil.com",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "allowlist non-browser client without Origin header",
			origins:    []string{"http://localhost:5173"},
			reqOrigin:  "",
			wantStatus: http.StatusSwitchingProtocols,
		},
		{
			name:       "wildcard allowlist",
			origins:    []string{"*"},
			reqOrigin:  "https://any-domain.com",
			wantStatus: http.StatusSwitchingProtocols,
		},
		{
			name:       "default same-origin rejects different origin",
			origins:    nil,
			reqOrigin:  "http://untrusted.com",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "default without Origin header accepted",
			origins:    nil,
			reqOrigin:  "",
			wantStatus: http.StatusSwitchingProtocols,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var opts []wsserver.Option
			if tt.origins != nil {
				opts = append(opts, wsserver.WithAllowedOrigins(tt.origins...))
			}
			hub := wsserver.NewServer(opts...)
			defer func() { _ = hub.Stop() }()

			server := httptest.NewServer(http.HandlerFunc(hub.ServeHTTP))
			defer func() { server.Close() }()

			wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
			header := make(http.Header)
			if tt.reqOrigin != "" {
				header.Set("Origin", tt.reqOrigin)
			}

			dialer := websocket.Dialer{HandshakeTimeout: 2 * time.Second}
			conn, resp, err := dialer.Dial(wsURL, header)
			if conn != nil {
				_ = conn.Close()
			}

			if tt.wantStatus == http.StatusSwitchingProtocols {
				if err != nil {
					t.Fatalf("expected successful connection, got error: %v", err)
				}
			} else {
				if resp == nil || resp.StatusCode != tt.wantStatus {
					var code int
					if resp != nil {
						code = resp.StatusCode
					}
					t.Fatalf("expected status %d, got %d (err: %v)", tt.wantStatus, code, err)
				}
			}
		})
	}
}

func TestServer_ConcurrentBroadcastAndDisconnect(t *testing.T) {
	t.Parallel()

	hub := wsserver.NewServer(
		wsserver.WithSendBuffer(32),
		wsserver.WithWriteWait(500*time.Millisecond),
	)
	defer func() { _ = hub.Stop() }()

	server := httptest.NewServer(http.HandlerFunc(hub.ServeHTTP))
	defer func() { server.Close() }()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	dialer := websocket.Dialer{HandshakeTimeout: 2 * time.Second}

	var conns []*websocket.Conn
	var connsMu sync.Mutex

	// Connect 10 clients
	for i := 0; i < 10; i++ {
		c, _, err := dialer.Dial(wsURL, nil)
		if err == nil {
			conns = append(conns, c)
			go func(conn *websocket.Conn) {
				for {
					_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
					if _, _, err := conn.ReadMessage(); err != nil {
						return
					}
				}
			}(c)
		}
	}

	defer func() {
		connsMu.Lock()
		for _, c := range conns {
			_ = c.Close()
		}
		connsMu.Unlock()
	}()

	var wg sync.WaitGroup
	// 32 concurrent broadcasters
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				hub.Broadcast([]byte(fmt.Sprintf("worker-%d-msg-%d", workerID, j)))
			}
		}(i)
	}

	// Simultaneously close some connections randomly
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < len(conns)/2; i++ {
			time.Sleep(10 * time.Millisecond)
			connsMu.Lock()
			if i < len(conns) && conns[i] != nil {
				_ = conns[i].Close()
			}
			connsMu.Unlock()
		}
	}()

	wg.Wait()
}

func TestServer_ConcurrentPingAndBroadcast(t *testing.T) {
	t.Parallel()

	hub := wsserver.NewServer(
		wsserver.WithSendBuffer(64),
		wsserver.WithWriteWait(500*time.Millisecond),
	)
	defer func() { _ = hub.Stop() }()

	server := httptest.NewServer(http.HandlerFunc(hub.ServeHTTP))
	defer func() { server.Close() }()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	dialer := websocket.Dialer{HandshakeTimeout: 2 * time.Second}

	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	var wg sync.WaitGroup

	// Client reader loop: consumes broadcast messages
	wg.Add(1)
	go func() {
		defer wg.Done()
		for ctx.Err() == nil {
			_ = conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}()

	// Client pinger: rapidly sends Ping control frames to trigger Pong response in Server
	wg.Add(1)
	go func() {
		defer wg.Done()
		for ctx.Err() == nil {
			_ = conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(200*time.Millisecond))
			time.Sleep(5 * time.Millisecond)
		}
	}()

	// Server broadcaster: rapidly broadcasts messages
	wg.Add(1)
	go func() {
		defer wg.Done()
		msg := []byte("broadcast-payload-1234567890")
		for ctx.Err() == nil {
			hub.Broadcast(msg)
			time.Sleep(2 * time.Millisecond)
		}
	}()

	wg.Wait()
}
