package tracer_test

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/dangnmh/core-kit/pkg/tracer"
)

func TestTracer_AppendAndUnified(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tracer-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	tr := tracer.New(tempDir, true)

	rec1 := map[string]any{"event": "step1", "val": 10}
	rec2 := map[string]any{"event": "step2", "val": 20}

	if err := tr.AppendUnified("step1.jsonl", "all.jsonl", rec1); err != nil {
		t.Fatalf("append error: %v", err)
	}
	if err := tr.AppendUnified("step2.jsonl", "all.jsonl", rec2); err != nil {
		t.Fatalf("append error: %v", err)
	}

	// Verify all.jsonl contains 2 lines
	allPath := filepath.Join(tempDir, "all.jsonl")
	f, err := os.Open(allPath)
	if err != nil {
		t.Fatalf("failed to open all.jsonl: %v", err)
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	var lines []map[string]any
	for scanner.Scan() {
		var m map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &m); err != nil {
			t.Fatalf("failed to parse json line: %v", err)
		}
		lines = append(lines, m)
	}

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines in unified file, got %d", len(lines))
	}
	if lines[0]["event"] != "step1" || lines[1]["event"] != "step2" {
		t.Fatalf("unexpected events: %v", lines)
	}
}
