package httpserver

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dangnmh/core-kit/pkg/ttlcache"
)

const (
	HeaderRequestNonce     = "X-Request-Nonce"
	HeaderRequestTimestamp = "X-Request-Timestamp"
	DefaultNonceTTLSeconds = 300
)

var (
	// ErrNonceBadRequest indicates a missing, oversized, or expired X-Request-Nonce / X-Request-Timestamp header (HTTP 400).
	ErrNonceBadRequest = errors.New("httpserver: invalid or expired request nonce")
	// ErrNonceReplayed indicates that an X-Request-Nonce has already been consumed within its TTL window (HTTP 409).
	ErrNonceReplayed = errors.New("httpserver: replay attack detected")
)

// NonceStore provides atomic test-and-set anti-replay nonce tracking backed by core-kit/pkg/ttlcache.
type NonceStore struct {
	mu    sync.Mutex
	cache *ttlcache.TTLCache[string, int64]
}

// NewNonceStore initializes a thread-safe NonceStore with automatic TTL cleanup.
func NewNonceStore() *NonceStore {
	return &NonceStore{
		cache: ttlcache.New[string, int64](DefaultNonceTTLSeconds*time.Second, 5*time.Minute),
	}
}

// Close stops the underlying ttlcache background cleanup goroutine.
func (s *NonceStore) Close() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cache != nil {
		s.cache.Close()
	}
}

// CheckAndMark atomically checks whether nonce has already been consumed within its TTL window,
// and if not, marks it as consumed for the given ttl duration. Returns true if the nonce is fresh.
func (s *NonceStore) CheckAndMark(nonce string, ttl time.Duration) bool {
	if s == nil {
		return true
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.cache.Get(nonce); exists {
		return false
	}
	s.cache.Set(nonce, time.Now().UnixMilli(), ttl)
	return true
}

// ValidateNonceRequest validates X-Request-Nonce and X-Request-Timestamp headers on an HTTP request.
// Returns nil on valid/fresh nonce, ErrNonceBadRequest (wrapped) on missing/invalid/expired headers,
// or ErrNonceReplayed (wrapped) on replay.
func ValidateNonceRequest(r *http.Request, store *NonceStore, requireNonce bool, ttlSeconds int) error {
	if r == nil {
		return nil
	}
	if ttlSeconds <= 0 {
		ttlSeconds = DefaultNonceTTLSeconds
	}

	nonce := strings.TrimSpace(r.Header.Get(HeaderRequestNonce))
	tsHeader := strings.TrimSpace(r.Header.Get(HeaderRequestTimestamp))

	if nonce == "" {
		if requireNonce {
			return fmt.Errorf("missing %s header on write request: %w", HeaderRequestNonce, ErrNonceBadRequest)
		}
		return nil
	}

	if len(nonce) > 128 {
		return fmt.Errorf("%s exceeds maximum length of 128 characters: %w", HeaderRequestNonce, ErrNonceBadRequest)
	}

	if tsHeader != "" {
		tsVal, err := strconv.ParseInt(tsHeader, 10, 64)
		if err != nil || tsVal <= 0 {
			return fmt.Errorf("invalid %s header: %w", HeaderRequestTimestamp, ErrNonceBadRequest)
		}
		tsMs := tsVal
		if tsVal < 1e11 {
			tsMs = tsVal * 1000
		}
		nowMs := time.Now().UnixMilli()
		diffMs := nowMs - tsMs
		if diffMs < 0 {
			diffMs = -diffMs
		}
		maxSkewMs := int64(ttlSeconds) * 1000
		if diffMs > maxSkewMs {
			return fmt.Errorf("request timestamp expired or outside allowed %ds window: %w", ttlSeconds, ErrNonceBadRequest)
		}
	}

	ttl := time.Duration(ttlSeconds) * time.Second
	if store != nil && !store.CheckAndMark(nonce, ttl) {
		return fmt.Errorf("replay attack detected: request nonce has already been used: %w", ErrNonceReplayed)
	}

	return nil
}
