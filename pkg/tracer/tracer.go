package tracer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Tracer provides thread-safe structured JSONL audit trail logging.
type Tracer struct {
	mu      sync.Mutex
	logDir  string
	enabled bool
}

// New creates a new Tracer instance with the target directory.
func New(logDir string, enabled bool) *Tracer {
	if logDir == "" {
		logDir = "data/logs"
	}
	_ = os.MkdirAll(logDir, 0755)

	return &Tracer{
		logDir:  logDir,
		enabled: enabled,
	}
}

// GetISOTime returns the current epoch seconds and RFC3339Nano timestamp string.
func (t *Tracer) GetISOTime() (float64, string) {
	now := time.Now()
	nowEpoch := float64(now.UnixNano()) / 1e9
	isoStr := now.UTC().Format(time.RFC3339Nano)
	return nowEpoch, isoStr
}

// AppendJSONL marshals a record and appends it as a single line to target file in logDir.
func (t *Tracer) AppendJSONL(filename string, record any) error {
	if !t.enabled {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	data, err := json.Marshal(record)
	if err != nil {
		return err
	}

	path := filepath.Join(t.logDir, filename)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	data = append(data, '\n')
	_, err = f.Write(data)
	return err
}

// AppendUnified appends a record both to a specific filename and a unified master trace file.
func (t *Tracer) AppendUnified(filename string, unifiedFile string, record any) error {
	if !t.enabled {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal trace record: %w", err)
	}
	dataWithNL := append(data, '\n')

	path := filepath.Join(t.logDir, filename)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open trace file %q: %w", path, err)
	}
	if _, err := f.Write(dataWithNL); err != nil {
		_ = f.Close()
		return fmt.Errorf("write trace file %q: %w", path, err)
	}
	_ = f.Close()

	if unifiedFile != "" {
		upath := filepath.Join(t.logDir, unifiedFile)
		uf, err := os.OpenFile(upath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("open unified trace file %q: %w", upath, err)
		}
		if _, err := uf.Write(dataWithNL); err != nil {
			_ = uf.Close()
			return fmt.Errorf("write unified trace file %q: %w", upath, err)
		}
		_ = uf.Close()
	}

	return nil
}

// Enabled returns whether tracing is enabled.
func (t *Tracer) Enabled() bool {
	return t.enabled
}

// LogDir returns the output directory.
func (t *Tracer) LogDir() string {
	return t.logDir
}
