package tests_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dangnmh/core-kit/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

// UnifiedLogEntry models the canonical log schema across Go, Python, and TypeScript.
type UnifiedLogEntry struct {
	Timestamp string         `json:"timestamp"`
	Severity  string         `json:"severity"`
	Service   string         `json:"service"`
	Logger    string         `json:"logger"`
	Message   string         `json:"message"`
	Caller    string         `json:"caller,omitempty"`
	TraceID   string         `json:"trace_id,omitempty"`
	SpanID    string         `json:"span_id,omitempty"`
	Context   map[string]any `json:"context,omitempty"`
}

const (
	expectedTraceID = "4bf92f3577b34da6a3ce929d0e0e4736"
	expectedSpanID  = "00f067aa0ba902b7"
	expectedService = "test-service"
	expectedUserID  = "u-123"
	expectedCluster = "us-east-1"
)

func TestE2E_PolyglotLoggerSchemaParity(t *testing.T) {
	t.Parallel()

	// 1. Go (pkg/logger)
	var goBuf bytes.Buffer
	lvl := new(slog.LevelVar)
	lvl.Set(slog.LevelInfo)
	goLogger := logger.NewLogger(expectedService, lvl, &goBuf).With("module", "go-e2e")

	tid, err := trace.TraceIDFromHex(expectedTraceID)
	require.NoError(t, err)
	sid, err := trace.SpanIDFromHex(expectedSpanID)
	require.NoError(t, err)
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    tid,
		SpanID:     sid,
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(t.Context(), sc)

	goLogger.InfoContext(ctx, "hello from go", "user_id", expectedUserID, "cluster", expectedCluster)

	var goEntry UnifiedLogEntry
	require.NoError(t, json.Unmarshal(goBuf.Bytes(), &goEntry), "Failed to parse Go log JSON")

	// Resolve repository root from tests directory
	repoRoot, err := filepath.Abs("..")
	require.NoError(t, err)

	// 2. Python (py/logger-py)
	pyCmd := exec.Command("python3", "-c", `
import sys, logging
sys.path.insert(0, "py/logger-py")
from logger_py.formatter import JsonFormatter
logger = logging.getLogger("py-e2e")
logger.setLevel(logging.INFO)
handler = logging.StreamHandler(sys.stdout)
handler.setFormatter(JsonFormatter(service_name="test-service"))
logger.addHandler(handler)
logger.info("hello from python", extra={"trace_id": "4bf92f3577b34da6a3ce929d0e0e4736", "span_id": "00f067aa0ba902b7", "user_id": "u-123", "cluster": "us-east-1"})
`)
	pyCmd.Dir = repoRoot
	pyOut, err := pyCmd.CombinedOutput()
	require.NoError(t, err, "Python logger execution failed: %s", string(pyOut))

	var pyEntry UnifiedLogEntry
	require.NoError(t, json.Unmarshal(pyOut, &pyEntry), "Failed to parse Python log JSON: %s", string(pyOut))

	// 3. TypeScript (ts/logger-ts via Bun)
	tsCmd := exec.Command("bun", "-e", `
import { createLogger } from "./ts/logger-ts/src/index";
const logger = createLogger({ service: "test-service", module: "ts-e2e" });
logger.info("hello from typescript", { trace_id: "4bf92f3577b34da6a3ce929d0e0e4736", span_id: "00f067aa0ba902b7", user_id: "u-123", cluster: "us-east-1" });
`)
	tsCmd.Dir = repoRoot
	tsOut, err := tsCmd.CombinedOutput()
	require.NoError(t, err, "TypeScript logger execution failed: %s", string(tsOut))

	var tsEntry UnifiedLogEntry
	require.NoError(t, json.Unmarshal(tsOut, &tsEntry), "Failed to parse TypeScript log JSON: %s", string(tsOut))

	// Cross-Language Assertions
	entries := []struct {
		runtime string
		entry   UnifiedLogEntry
		module  string
	}{
		{"Go", goEntry, "go-e2e"},
		{"Python", pyEntry, "py-e2e"},
		{"TypeScript", tsEntry, "ts-e2e"},
	}

	for _, tc := range entries {
		t.Run(tc.runtime, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, expectedService, tc.entry.Service, "%s service mismatch", tc.runtime)
			assert.Equal(t, "INFO", tc.entry.Severity, "%s severity mismatch", tc.runtime)
			assert.Equal(t, tc.module, tc.entry.Logger, "%s logger name mismatch", tc.runtime)
			assert.Equal(t, expectedTraceID, tc.entry.TraceID, "%s trace_id hoisting mismatch", tc.runtime)
			assert.Equal(t, expectedSpanID, tc.entry.SpanID, "%s span_id hoisting mismatch", tc.runtime)

			// Timestamp RFC3339 validation
			parsedTime, parseErr := time.Parse(time.RFC3339Nano, tc.entry.Timestamp)
			if parseErr != nil {
				parsedTime, parseErr = time.Parse(time.RFC3339, tc.entry.Timestamp)
			}
			assert.NoError(t, parseErr, "%s timestamp must be valid RFC3339 (got: %s)", tc.runtime, tc.entry.Timestamp)
			assert.WithinDuration(t, time.Now().UTC(), parsedTime, 10*time.Second, "%s timestamp drift", tc.runtime)

			// Context sub-object validation
			require.NotNil(t, tc.entry.Context, "%s context sub-object must not be nil", tc.runtime)
			assert.Equal(t, expectedUserID, tc.entry.Context["user_id"], "%s context.user_id mismatch", tc.runtime)
			assert.Equal(t, expectedCluster, tc.entry.Context["cluster"], "%s context.cluster mismatch", tc.runtime)

			// No emoji in message
			assert.False(t, strings.ContainsAny(tc.entry.Message, "🚀🔥✨🚨"), "%s message should not contain emojis", tc.runtime)
		})
	}
}
