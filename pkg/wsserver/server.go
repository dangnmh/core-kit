package wsserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/dangnmh/core-kit/pkg/logger"
	"github.com/gorilla/websocket"
)

const (
	defaultWriteWait  = 10 * time.Second
	defaultPongWait   = 60 * time.Second
	defaultPingPeriod = (defaultPongWait * 9) / 10
	defaultMaxMsgSize = 512 * 1024
	defaultSendBuf    = 64
)

type client struct {
	conn    *websocket.Conn
	send    chan []byte // cap = s.sendBuf
	once    sync.Once   // closes send exactly once
	writeMu sync.Mutex  // serializes all writes (messages and control frames) to conn
}

// BroadcastResult contains metrics about a broadcast operation.
type BroadcastResult struct {
	Delivered int // number of clients where message was successfully enqueued
	Dropped   int // number of slow clients evicted due to full queue
}

// Option configures the Server.
type Option func(*Server)

// WithWriteWait sets the write timeout.
func WithWriteWait(d time.Duration) Option {
	return func(s *Server) {
		if d > 0 {
			s.writeWait = d
		}
	}
}

// WithPongWait sets the pong timeout and recalibrates ping period.
func WithPongWait(d time.Duration) Option {
	return func(s *Server) {
		if d > 0 {
			s.pongWait = d
			s.pingPeriod = (d * 9) / 10
		}
	}
}

// WithCheckOrigin sets custom WebSocket origin checking.
func WithCheckOrigin(fn func(r *http.Request) bool) Option {
	return func(s *Server) {
		s.upgrader.CheckOrigin = fn
	}
}

// WithAllowedOrigins sets the allowlist of origins permitted to establish WebSocket connections.
// If empty, gorilla's default same-origin check is enforced.
func WithAllowedOrigins(origins ...string) Option {
	return func(s *Server) {
		for _, o := range origins {
			trimmed := strings.TrimRight(strings.TrimSpace(o), "/")
			if trimmed != "" {
				s.origins = append(s.origins, trimmed)
			}
		}
	}
}

// WithSendBuffer sets per-client outbound message buffer size.
func WithSendBuffer(n int) Option {
	return func(s *Server) {
		if n > 0 {
			s.sendBuf = n
		}
	}
}

// WithMaxMessageSize sets max incoming frame size in bytes.
func WithMaxMessageSize(size int64) Option {
	return func(s *Server) {
		if size > 0 {
			s.maxMsgSize = size
		}
	}
}

// WithOnConnect sets callback on new client connection.
func WithOnConnect(fn func(*websocket.Conn)) Option {
	return func(s *Server) {
		s.onConnect = fn
	}
}

// WithOnDisconnect sets callback on client disconnect.
func WithOnDisconnect(fn func(*websocket.Conn)) Option {
	return func(s *Server) {
		s.onDisconnect = fn
	}
}

// Server implements a robust, thread-safe WebSocket broadcast hub with per-client send pumps.
type Server struct {
	mu           sync.RWMutex
	clients      map[*client]struct{}
	stopChan     chan struct{}
	stopOnce     sync.Once
	upgrader     websocket.Upgrader
	writeWait    time.Duration
	pongWait     time.Duration
	pingPeriod   time.Duration
	maxMsgSize   int64
	sendBuf      int
	origins      []string
	onConnect    func(*websocket.Conn)
	onDisconnect func(*websocket.Conn)
	wg           sync.WaitGroup
}

// NewServer initializes a WebSocket Server Hub.
func NewServer(opts ...Option) *Server {
	s := &Server{
		clients:    make(map[*client]struct{}),
		stopChan:   make(chan struct{}),
		writeWait:  defaultWriteWait,
		pongWait:   defaultPongWait,
		pingPeriod: defaultPingPeriod,
		maxMsgSize: defaultMaxMsgSize,
		sendBuf:    defaultSendBuf,
	}

	for _, opt := range opts {
		opt(s)
	}

	if s.sendBuf <= 0 {
		s.sendBuf = defaultSendBuf
	}

	// Origin checking logic:
	// If custom CheckOrigin was provided via WithCheckOrigin, keep it.
	// Otherwise:
	// - If s.origins is non-empty: check origin header against allowlist (case-insensitive).
	//   Requests without Origin header (non-browser clients) are allowed.
	// - If s.origins is empty: s.upgrader.CheckOrigin remains nil (gorilla default same-origin).
	if s.upgrader.CheckOrigin == nil && len(s.origins) > 0 {
		s.upgrader.CheckOrigin = func(r *http.Request) bool {
			origin := strings.TrimRight(strings.TrimSpace(r.Header.Get("Origin")), "/")
			if origin == "" {
				return true
			}
			for _, allowed := range s.origins {
				if allowed == "*" || strings.EqualFold(origin, allowed) {
					return true
				}
			}
			return false
		}
	}

	return s
}

// ClientCount returns the number of active connected clients.
func (s *Server) ClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.clients)
}

// BroadcastDetailed broadcasts payload to all connected clients without head-of-line blocking.
// If a client's send buffer is full, it is safely evicted as a slow consumer.
func (s *Server) BroadcastDetailed(data []byte) BroadcastResult {
	if len(data) == 0 {
		return BroadcastResult{}
	}

	s.mu.RLock()
	select {
	case <-s.stopChan:
		s.mu.RUnlock()
		return BroadcastResult{}
	default:
	}

	var slow []*client
	result := BroadcastResult{}

	for c := range s.clients {
		select {
		case c.send <- data:
			result.Delivered++
		default:
			slow = append(slow, c)
		}
	}
	s.mu.RUnlock()

	for _, c := range slow {
		if s.removeClient(c) {
			result.Dropped++
			logger.Warn("Evicted slow websocket client", "queue_cap", s.sendBuf)
		}
	}

	return result
}

// Broadcast sends binary or text payload to all connected clients.
// Returns the number of clients to which the message was successfully enqueued.
func (s *Server) Broadcast(data []byte) int {
	return s.BroadcastDetailed(data).Delivered
}

// BroadcastJSON marshals payload to JSON and broadcasts to all connected clients.
func (s *Server) BroadcastJSON[T any](payload T) (int, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("marshal websocket broadcast json: %w", err)
	}
	return s.Broadcast(data), nil
}

// Stop closes all active connections and waits for client pumps to terminate.
func (s *Server) Stop() error {
	s.stopOnce.Do(func() {
		s.mu.Lock()
		close(s.stopChan)
		disconnected := make([]*client, 0, len(s.clients))
		for c := range s.clients {
			delete(s.clients, c)
			c.once.Do(func() {
				close(c.send)
			})
			disconnected = append(disconnected, c)
		}
		s.mu.Unlock()

		for _, c := range disconnected {
			_ = c.conn.Close()
			if s.onDisconnect != nil {
				s.onDisconnect(c.conn)
			}
		}
		s.wg.Wait()
	})
	return nil
}

// ServeHTTP handles incoming HTTP requests and upgrades them to WebSocket.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Warn("WebSocket upgrade failed", "error", err.Error())
		return
	}

	c := &client{
		conn: conn,
		send: make(chan []byte, s.sendBuf),
	}

	conn.SetReadLimit(s.maxMsgSize)
	_ = conn.SetReadDeadline(time.Now().Add(s.pongWait))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(s.pongWait))
		return nil
	})
	conn.SetPingHandler(func(appData string) error {
		_ = conn.SetReadDeadline(time.Now().Add(s.pongWait))
		c.writeMu.Lock()
		defer c.writeMu.Unlock()
		return conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(s.writeWait))
	})

	s.mu.Lock()
	select {
	case <-s.stopChan:
		s.mu.Unlock()
		_ = conn.Close()
		return
	default:
		s.wg.Add(2)
		s.clients[c] = struct{}{}
	}
	active := len(s.clients)
	s.mu.Unlock()

	if s.onConnect != nil {
		s.onConnect(conn)
	}
	logger.Info("New WebSocket subscriber connected", "active", active)

	go s.writePump(c)
	go s.readPump(c)
}

func (s *Server) readPump(c *client) {
	defer s.wg.Done()
	defer s.removeClient(c)

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (s *Server) writePump(c *client) {
	defer s.wg.Done()
	ticker := time.NewTicker(s.pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case msg, ok := <-c.send:
			if !ok {
				c.writeMu.Lock()
				_ = c.conn.SetWriteDeadline(time.Now().Add(s.writeWait))
				_ = c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				c.writeMu.Unlock()
				return
			}
			c.writeMu.Lock()
			_ = c.conn.SetWriteDeadline(time.Now().Add(s.writeWait))
			err := c.conn.WriteMessage(websocket.TextMessage, msg)
			c.writeMu.Unlock()
			if err != nil {
				s.removeClient(c)
				return
			}
		case <-ticker.C:
			c.writeMu.Lock()
			_ = c.conn.SetWriteDeadline(time.Now().Add(s.writeWait))
			err := c.conn.WriteMessage(websocket.PingMessage, nil)
			c.writeMu.Unlock()
			if err != nil {
				s.removeClient(c)
				return
			}
		}
	}
}

func (s *Server) removeClient(c *client) bool {
	s.mu.Lock()
	if _, exists := s.clients[c]; !exists {
		s.mu.Unlock()
		return false
	}
	delete(s.clients, c)
	c.once.Do(func() {
		close(c.send)
	})
	active := len(s.clients)
	s.mu.Unlock()

	_ = c.conn.Close()
	if s.onDisconnect != nil {
		s.onDisconnect(c.conn)
	}
	logger.Info("WebSocket subscriber disconnected", "active", active)
	return true
}
