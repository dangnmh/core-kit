package llm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DebugRecord represents an LLM interaction entry written to JSONL.
type DebugRecord struct {
	Timestamp time.Time `json:"timestamp"`
	Component string    `json:"component,omitempty"`
	Model     string    `json:"model,omitempty"`
	Endpoint  string    `json:"endpoint,omitempty"`
	Prompt    any       `json:"prompt,omitempty"`
	Response  any       `json:"response,omitempty"`
	LatencyMs int64     `json:"latency_ms"`
	Error     string    `json:"error,omitempty"`
}

// DebugLogger writes DebugRecord entries as JSON Lines (jsonl) to a designated file.
type DebugLogger interface {
	Log(record DebugRecord) error
	Close() error
}

type fileDebugLogger struct {
	mu       sync.Mutex
	filePath string
	file     *os.File
}

// NewDebugLogger constructs a thread-safe DebugLogger that appends entries to filePath.
// If filePath is empty, it defaults to "data/llm_debug.jsonl".
func NewDebugLogger(filePath string) (DebugLogger, error) {
	if filePath == "" {
		filePath = filepath.Join("data", "llm_debug.jsonl")
	}

	dir := filepath.Dir(filePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create directory for llm debug log %q: %w", dir, err)
		}
	}

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("open llm debug log file %q: %w", filePath, err)
	}

	return &fileDebugLogger{
		filePath: filePath,
		file:     f,
	}, nil
}

func (l *fileDebugLogger) Log(record DebugRecord) error {
	if l == nil {
		return nil
	}
	if record.Timestamp.IsZero() {
		record.Timestamp = time.Now().UTC()
	}

	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal llm debug record: %w", err)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file == nil {
		f, err := os.OpenFile(l.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("re-open llm debug log file: %w", err)
		}
		l.file = f
	}

	data = append(data, '\n')
	if _, err := l.file.Write(data); err != nil {
		return fmt.Errorf("write llm debug record: %w", err)
	}

	return nil
}

func (l *fileDebugLogger) Close() error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		err := l.file.Close()
		l.file = nil
		return err
	}
	return nil
}

type noopDebugLogger struct{}

// NewNoopDebugLogger returns a no-op implementation of DebugLogger.
func NewNoopDebugLogger() DebugLogger {
	return &noopDebugLogger{}
}

func (n *noopDebugLogger) Log(record DebugRecord) error {
	return nil
}

func (n *noopDebugLogger) Close() error {
	return nil
}

// SafeLog logs a DebugRecord if logger is not nil.
func SafeLog(logger DebugLogger, record DebugRecord) error {
	if logger == nil {
		return nil
	}
	return logger.Log(record)
}
