package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel/trace"
)

var (
	mu            sync.RWMutex
	defaultLogger *slog.Logger
	currentLevel  *slog.LevelVar
)

func init() {
	currentLevel = new(slog.LevelVar)
	currentLevel.Set(slog.LevelInfo)
	defaultLogger = NewLogger("default", currentLevel, os.Stdout)
}

// replaceAttr ensures consistent JSON fields matching logger-py and logger-ts:
// timestamp (RFC3339Nano), severity (DEBUG, INFO, WARN, ERROR), message, logger
func replaceAttr(groups []string, a slog.Attr) slog.Attr {
	if len(groups) == 0 {
		switch a.Key {
		case slog.TimeKey:
			a.Key = "timestamp"
			if t, ok := a.Value.Any().(time.Time); ok {
				a.Value = slog.StringValue(t.UTC().Format(time.RFC3339Nano))
			}
		case slog.LevelKey:
			a.Key = "severity"
			if lvl, ok := a.Value.Any().(slog.Level); ok {
				switch {
				case lvl < slog.LevelInfo:
					a.Value = slog.StringValue("DEBUG")
				case lvl < slog.LevelWarn:
					a.Value = slog.StringValue("INFO")
				case lvl < slog.LevelError:
					a.Value = slog.StringValue("WARN")
				default:
					a.Value = slog.StringValue("ERROR")
				}
			} else {
				a.Value = slog.StringValue(strings.ToUpper(a.Value.String()))
			}
		case slog.MessageKey:
			a.Key = "message"
		case "module":
			a.Key = "logger"
		}
	}
	return a
}

// DualModeHandler wraps an slog.Handler to group caller attributes under "context".
// If LOG_LEGACY_FLAT is true or 1, attributes are also mirrored to the root JSON object.
type DualModeHandler struct {
	inner      slog.Handler
	legacyFlat bool
}

// NewDualModeHandler creates a new DualModeHandler wrapping inner.
func NewDualModeHandler(inner slog.Handler) *DualModeHandler {
	legacy := os.Getenv("LOG_LEGACY_FLAT") == "true" || os.Getenv("LOG_LEGACY_FLAT") == "1"
	return &DualModeHandler{inner: inner, legacyFlat: legacy}
}

func (h *DualModeHandler) isLegacyFlat() bool {
	if h.legacyFlat {
		return true
	}
	v := os.Getenv("LOG_LEGACY_FLAT")
	return v == "true" || v == "1"
}

func (h *DualModeHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *DualModeHandler) Handle(ctx context.Context, r slog.Record) error {
	var userAttrs []slog.Attr
	r.Attrs(func(a slog.Attr) bool {
		userAttrs = append(userAttrs, a)
		return true
	})

	if len(userAttrs) > 0 {
		contextGroup := slog.Group("context", attrsToAny(userAttrs)...)
		newRecord := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
		if h.isLegacyFlat() {
			newRecord.AddAttrs(userAttrs...)
		}
		newRecord.AddAttrs(contextGroup)
		return h.inner.Handle(ctx, newRecord)
	}
	return h.inner.Handle(ctx, r)
}

func (h *DualModeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &DualModeHandler{inner: h.inner.WithAttrs(attrs), legacyFlat: h.legacyFlat}
}

func (h *DualModeHandler) WithGroup(name string) slog.Handler {
	return &DualModeHandler{inner: h.inner.WithGroup(name), legacyFlat: h.legacyFlat}
}

func attrsToAny(attrs []slog.Attr) []any {
	res := make([]any, len(attrs))
	for i, a := range attrs {
		res[i] = a
	}
	return res
}

type traceHandler struct {
	slog.Handler
}

func (h *traceHandler) Handle(ctx context.Context, r slog.Record) error {
	sc := trace.SpanContextFromContext(ctx)
	if sc.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}
	return h.Handler.Handle(ctx, r)
}

func (h *traceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &traceHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *traceHandler) WithGroup(name string) slog.Handler {
	return &traceHandler{Handler: h.Handler.WithGroup(name)}
}

// ParseLevel parses a string representation of a log level.
// If empty or unrecognized, it defaults to slog.LevelInfo.
func ParseLevel(lvl string) slog.Level {
	switch strings.ToUpper(strings.TrimSpace(lvl)) {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO", "":
		return slog.LevelInfo
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// SetLevel dynamically adjusts the global logging level across all active loggers.
// If lvl is empty, it defaults to "info".
func SetLevel(lvl string) {
	if lvl == "" {
		lvl = "info"
	}
	currentLevel.Set(ParseLevel(lvl))
}

// GetLevel returns the current active global logging level.
func GetLevel() slog.Level {
	return currentLevel.Level()
}

// NewLogger creates an slog.Logger instance with service identification and dynamic level filtering.
func NewLogger(service string, level slog.Leveler, w io.Writer) *slog.Logger {
	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level:       level,
		ReplaceAttr: replaceAttr,
	})
	dual := NewDualModeHandler(&traceHandler{Handler: handler})
	return slog.New(dual).With("service", service)
}

// SetupLogging configures the package-level default logger using the LOG_LEVEL env var or "info".
func SetupLogging(service string) *slog.Logger {
	lvl := os.Getenv("LOG_LEVEL")
	if lvl == "" {
		lvl = "info"
	}
	return SetupLoggingWithLevel(service, lvl)
}

// SetupLoggingWithLevel configures the package-level default logger with an explicit level.
// If levelStr is empty, it defaults to "info".
func SetupLoggingWithLevel(service, levelStr string) *slog.Logger {
	mu.Lock()
	defer mu.Unlock()

	if levelStr == "" {
		levelStr = "info"
	}
	currentLevel.Set(ParseLevel(levelStr))

	defaultLogger = NewLogger(service, currentLevel, os.Stdout)
	slog.SetDefault(defaultLogger)
	return defaultLogger
}

// Get returns the current default logger.
func Get() *slog.Logger {
	mu.RLock()
	defer mu.RUnlock()
	return defaultLogger
}

// SetDefault sets the package-level default logger in a thread-safe manner.
func SetDefault(l *slog.Logger) {
	if l == nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	defaultLogger = l
	slog.SetDefault(l)
}

func Debug(msg string, args ...any) {
	Get().Debug(msg, args...)
}

func Info(msg string, args ...any) {
	Get().Info(msg, args...)
}

func Warn(msg string, args ...any) {
	Get().Warn(msg, args...)
}

func Error(msg string, args ...any) {
	Get().Error(msg, args...)
}

func DebugContext(ctx context.Context, msg string, args ...any) {
	Get().DebugContext(ctx, msg, args...)
}

func InfoContext(ctx context.Context, msg string, args ...any) {
	Get().InfoContext(ctx, msg, args...)
}

func WarnContext(ctx context.Context, msg string, args ...any) {
	Get().WarnContext(ctx, msg, args...)
}

func ErrorContext(ctx context.Context, msg string, args ...any) {
	Get().ErrorContext(ctx, msg, args...)
}
