package llm_test

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/dangnmh/core-kit/pkg/llm"
)

func TestDebugLogger_SequentialWrite(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "logs", "llm_debug.jsonl")

	logger, err := llm.NewDebugLogger(logPath)
	if err != nil {
		t.Fatalf("failed to create debug logger: %v", err)
	}
	defer func() { _ = logger.Close() }()

	rec1 := llm.DebugRecord{
		Timestamp: time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC),
		Component: "llm-planner",
		Model:     "gpt-4o-mini",
		Endpoint:  "http://ai-proxy:8317/v1/chat/completions",
		Prompt: map[string]any{
			"system": "you are an assistant",
		},
		Response: map[string]any{
			"content": "hello world",
		},
		LatencyMs: 250,
	}

	rec2 := llm.DebugRecord{
		Timestamp: time.Date(2026, 9, 26, 12, 0, 1, 0, time.UTC),
		Component: "llm-executor",
		Model:     "qwen2.5-coder:1.5b",
		Endpoint:  "http://localhost:11434/api/generate",
		Prompt:    "mine 5 cobblestone",
		Response:  "mine_block",
		LatencyMs: 120,
		Error:     "",
	}

	if err := logger.Log(rec1); err != nil {
		t.Fatalf("failed to log rec1: %v", err)
	}
	if err := logger.Log(rec2); err != nil {
		t.Fatalf("failed to log rec2: %v", err)
	}

	// Read and verify file contents
	f, err := os.Open(logPath)
	if err != nil {
		t.Fatalf("failed to open log file: %v", err)
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("failed to read lines: %v", err)
	}

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	var parsed1, parsed2 llm.DebugRecord
	if err := json.Unmarshal([]byte(lines[0]), &parsed1); err != nil {
		t.Fatalf("line 1 is not valid JSON: %v", err)
	}
	if parsed1.Component != "llm-planner" || parsed1.LatencyMs != 250 {
		t.Errorf("unexpected parsed1: %+v", parsed1)
	}

	if err := json.Unmarshal([]byte(lines[1]), &parsed2); err != nil {
		t.Fatalf("line 2 is not valid JSON: %v", err)
	}
	if parsed2.Component != "llm-executor" || parsed2.LatencyMs != 120 {
		t.Errorf("unexpected parsed2: %+v", parsed2)
	}
}

func TestDebugLogger_ConcurrentWrite(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "llm_concurrent.jsonl")

	logger, err := llm.NewDebugLogger(logPath)
	if err != nil {
		t.Fatalf("failed to create debug logger: %v", err)
	}
	defer func() { _ = logger.Close() }()

	const count = 100
	var wg sync.WaitGroup
	wg.Add(count)

	for i := 0; i < count; i++ {
		idx := i
		go func() {
			defer wg.Done()
			err := logger.Log(llm.DebugRecord{
				Component: "test-concurrency",
				Model:     "test-model",
				LatencyMs: int64(idx),
			})
			if err != nil {
				t.Errorf("concurrent log error: %v", err)
			}
		}()
	}

	wg.Wait()

	// Verify all 100 lines are present and valid JSON
	f, err := os.Open(logPath)
	if err != nil {
		t.Fatalf("failed to open log file: %v", err)
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	lineCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		var rec llm.DebugRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("corrupted JSON line: %v (content: %q)", err, line)
		}
		lineCount++
	}

	if lineCount != count {
		t.Fatalf("expected %d lines, got %d", count, lineCount)
	}
}

func TestDebugLogger_NilAndNoop(t *testing.T) {
	t.Parallel()

	if err := llm.SafeLog(nil, llm.DebugRecord{}); err != nil {
		t.Errorf("expected SafeLog with nil logger to not error, got %v", err)
	}

	noop := llm.NewNoopDebugLogger()
	if err := noop.Log(llm.DebugRecord{Component: "noop"}); err != nil {
		t.Errorf("expected noop Log to not error, got %v", err)
	}
	if err := noop.Close(); err != nil {
		t.Errorf("expected noop Close to not error, got %v", err)
	}
}
