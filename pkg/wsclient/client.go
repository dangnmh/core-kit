package wsclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"sync"
	"time"

	"github.com/dangnmh/core-kit/pkg/logger"
	"github.com/gorilla/websocket"
)

// ErrNotConnected indicates that Send was called without an active WebSocket connection.
var ErrNotConnected = errors.New("websocket not connected")

type MessageHandler func(payload []byte)

type Options struct {
	Header         http.Header
	ReconnectWait  time.Duration
	MaxReconnect   time.Duration
	PingInterval   time.Duration
	PongWait       time.Duration
	WriteWait      time.Duration
	ReadLimit      int64 // 0 = 1 MiB default
	OnConnected    func()
	OnDisconnected func(err error)
}

type Option func(*Options)

func WithHeader(header http.Header) Option {
	return func(o *Options) {
		o.Header = header
	}
}

func WithReconnectWait(initial, max time.Duration) Option {
	return func(o *Options) {
		o.ReconnectWait = initial
		o.MaxReconnect = max
	}
}

func WithHeartbeat(pingInterval, pongWait time.Duration) Option {
	return func(o *Options) {
		o.PingInterval = pingInterval
		o.PongWait = pongWait
	}
}

func WithConnectHooks(onConnected func(), onDisconnected func(error)) Option {
	return func(o *Options) {
		o.OnConnected = onConnected
		o.OnDisconnected = onDisconnected
	}
}

func WithReadLimit(n int64) Option {
	return func(o *Options) {
		o.ReadLimit = n
	}
}

type Client struct {
	url     string
	handler MessageHandler
	options Options

	mu     sync.Mutex // protects conn, closed
	conn   *websocket.Conn
	closed bool

	writeMu sync.Mutex // serializes all writes to conn (data + control)

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func New(url string, handler MessageHandler, opts ...Option) *Client {
	options := Options{
		Header:        nil,
		ReconnectWait: 1 * time.Second,
		MaxReconnect:  30 * time.Second,
		PingInterval:  20 * time.Second,
		PongWait:      30 * time.Second,
		WriteWait:     5 * time.Second,
		ReadLimit:     1024 * 1024,
	}

	for _, opt := range opts {
		opt(&options)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Client{
		url:     url,
		handler: handler,
		options: options,
		ctx:     ctx,
		cancel:  cancel,
	}
}

func (c *Client) Start() {
	c.wg.Add(1)
	go c.lifecycleLoop()
}

// Close safely shuts down the WebSocket client and waits for goroutines to exit.
// Note: Close should not be called synchronously from inside a MessageHandler or OnDisconnected callback;
// use CloseAsync for non-blocking shutdown from callbacks.
func (c *Client) Close() {
	c.closeInternal()
	c.wg.Wait()
}

// CloseAsync initiates shutdown without waiting for background loops to exit.
// Safe to call from within handlers or callbacks.
func (c *Client) CloseAsync() {
	c.closeInternal()
}

func (c *Client) closeInternal() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	c.cancel()
	conn := c.conn
	c.mu.Unlock()

	if conn != nil {
		c.writeMu.Lock()
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
		_ = conn.Close()
		c.writeMu.Unlock()
	}
}

func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil
}

// writeFrame is the single path for all writes; callers must NOT hold c.mu.
func (c *Client) writeFrame(conn *websocket.Conn, msgType int, data []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	writeWait := c.options.WriteWait
	if writeWait <= 0 {
		writeWait = 5 * time.Second
	}
	if err := conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		return fmt.Errorf("websocket set write deadline: %w", err)
	}
	if err := conn.WriteMessage(msgType, data); err != nil {
		return fmt.Errorf("websocket write: %w", err)
	}
	return nil
}

func (c *Client) Send(payload []byte) error {
	c.mu.Lock()
	if c.closed || c.conn == nil {
		c.mu.Unlock()
		return ErrNotConnected
	}
	conn := c.conn
	c.mu.Unlock()

	return c.writeFrame(conn, websocket.TextMessage, payload)
}

// SendPayload marshals a typed payload to JSON and sends it over WebSocket.
func (c *Client) SendPayload[T any](payload T) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal websocket payload: %w", err)
	}
	return c.Send(data)
}

func (c *Client) SendJSON(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.Send(data)
}

func (c *Client) lifecycleLoop() {
	defer c.wg.Done()
	reconnectWait := c.options.ReconnectWait
	if reconnectWait <= 0 {
		reconnectWait = 1 * time.Second
	}
	maxReconnect := c.options.MaxReconnect
	if maxReconnect < reconnectWait {
		maxReconnect = reconnectWait
	}
	backoff := reconnectWait

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		conn, _, err := websocket.DefaultDialer.DialContext(c.ctx, c.url, c.options.Header)
		if err != nil {
			logger.Warn("WebSocket connection failed, retrying...", "url", c.url, "error", err, "backoff", backoff)
			sleep := reconnectWait
			if int64(backoff) > 0 {
				sleep = time.Duration(rand.Int64N(int64(backoff)))
			}
			select {
			case <-c.ctx.Done():
				return
			case <-time.After(sleep):
			}
			backoff = min(backoff*2, maxReconnect)
			continue
		}

		readLimit := c.options.ReadLimit
		if readLimit <= 0 {
			readLimit = 1024 * 1024
		}
		conn.SetReadLimit(readLimit)

		c.mu.Lock()
		c.conn = conn
		c.mu.Unlock()

		backoff = reconnectWait
		logger.Info("WebSocket client connected", "url", c.url)

		if c.options.OnConnected != nil {
			c.options.OnConnected()
		}

		readErr := c.readAndHeartbeatLoop(conn)

		c.mu.Lock()
		isClosed := c.closed
		c.conn = nil
		c.mu.Unlock()
		_ = conn.Close()

		if errors.Is(readErr, websocket.ErrReadLimit) {
			logger.Warn("WebSocket read limit exceeded", "limit", readLimit, "error", readErr)
		} else if !isClosed {
			logger.Warn("WebSocket client disconnected, reconnecting...", "url", c.url, "error", readErr)
		} else {
			logger.Info("WebSocket client disconnected on close", "url", c.url)
		}

		if c.options.OnDisconnected != nil {
			c.options.OnDisconnected(readErr)
		}

		if isClosed {
			return
		}
	}
}

func (c *Client) readAndHeartbeatLoop(conn *websocket.Conn) error {
	_ = conn.SetReadDeadline(time.Now().Add(c.options.PongWait))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(c.options.PongWait))
		return nil
	})

	done := make(chan struct{})

	var pingWg sync.WaitGroup
	if c.options.PingInterval > 0 {
		pingWg.Add(1)
		go func() {
			defer pingWg.Done()
			ticker := time.NewTicker(c.options.PingInterval)
			defer ticker.Stop()

			for {
				select {
				case <-done:
					return
				case <-c.ctx.Done():
					return
				case <-ticker.C:
					if err := c.writeFrame(conn, websocket.PingMessage, nil); err != nil {
						_ = conn.Close()
						return
					}
				}
			}
		}()
	}

	var readErr error
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			readErr = err
			break
		}
		if c.handler != nil {
			c.handler(message)
		}
	}

	close(done)
	pingWg.Wait()
	return readErr
}
