package httplog

import (
	"bytes"
	"encoding/json"
	"slices"
	"strings"
)

// Engine defines the core routing and payload redaction capabilities of httplog.
type Engine interface {
	Match(method, path string) (*CompiledRouteRule, bool)
	MatchInbound(method, path string) (*CompiledRouteRule, bool)
	MatchOutbound(method, path string) (*CompiledRouteRule, bool)
	SanitizeBody(contentType string, body []byte, rules []CompiledFieldRule) string
	ShouldLogResponse(statusCode int, rule *CompiledRouteRule) bool
	DefaultMaskValue() string
	MaxBodySize() int
	LogInboundEnabled() bool
	LogOutboundEnabled() bool
	LogUnmatchedEnabled() bool
}

type engineImpl struct {
	cfg     *Config
	matcher *RouteMatcher
}

// NewEngine constructs and precompiles a new Engine from the given Config.
func NewEngine(cfg *Config) (Engine, error) {
	if cfg == nil {
		return nil, ErrNilConfig
	}

	m, err := NewRouteMatcher(cfg)
	if err != nil {
		return nil, err
	}

	return &engineImpl{
		cfg:     cfg,
		matcher: m,
	}, nil
}

func (e *engineImpl) Match(method, path string) (*CompiledRouteRule, bool) {
	if e == nil || e.matcher == nil {
		return nil, false
	}
	return e.matcher.Match(method, path)
}

func (e *engineImpl) MatchInbound(method, path string) (*CompiledRouteRule, bool) {
	if e == nil || e.matcher == nil {
		return nil, false
	}
	return e.matcher.MatchInbound(method, path)
}

func (e *engineImpl) MatchOutbound(method, path string) (*CompiledRouteRule, bool) {
	if e == nil || e.matcher == nil {
		return nil, false
	}
	return e.matcher.MatchOutbound(method, path)
}

func (e *engineImpl) DefaultMaskValue() string {
	if e == nil || e.cfg == nil || e.cfg.Default.MaskValue == "" {
		return "[REDACTED]"
	}
	return e.cfg.Default.MaskValue
}

func (e *engineImpl) MaxBodySize() int {
	if e == nil || e.cfg == nil {
		return 0
	}
	return e.cfg.Default.MaxBodySize
}

func (e *engineImpl) LogInboundEnabled() bool {
	if e == nil || e.cfg == nil {
		return false
	}
	return e.cfg.IsLogInboundEnabled()
}

func (e *engineImpl) LogOutboundEnabled() bool {
	if e == nil || e.cfg == nil {
		return true
	}
	return e.cfg.IsLogOutboundEnabled()
}

func (e *engineImpl) LogUnmatchedEnabled() bool {
	if e == nil || e.cfg == nil {
		return false
	}
	return e.cfg.IsLogUnmatchedEnabled()
}

func (e *engineImpl) SanitizeBody(contentType string, body []byte, rules []CompiledFieldRule) string {
	if len(body) == 0 {
		return ""
	}

	lowerCT := strings.ToLower(strings.TrimSpace(contentType))
	trimmed := bytes.TrimSpace(body)
	isJSONBody := len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') && json.Valid(trimmed)

	var res string
	switch {
	case strings.Contains(lowerCT, "application/json") || strings.HasSuffix(lowerCT, "+json") || isJSONBody:
		res = SanitizeJSON(body, rules, e.DefaultMaskValue())
	case strings.Contains(lowerCT, "application/x-www-form-urlencoded"):
		res = SanitizeForm(body, rules, e.DefaultMaskValue())
	case strings.Contains(lowerCT, "multipart/form-data"):
		res = "[MULTIPART_FORM_DATA]"
	case strings.HasPrefix(lowerCT, "image/") ||
		strings.HasPrefix(lowerCT, "audio/") ||
		strings.HasPrefix(lowerCT, "video/") ||
		strings.Contains(lowerCT, "application/octet-stream") ||
		strings.Contains(lowerCT, "application/pdf") ||
		strings.Contains(lowerCT, "application/zip"):
		res = SummarizeBinary(contentType, len(body))
	default:
		res = string(body)
	}

	maxSize := e.MaxBodySize()
	if maxSize > 0 && len(res) > maxSize {
		res = res[:maxSize]
	}

	return res
}

func (e *engineImpl) ShouldLogResponse(statusCode int, rule *CompiledRouteRule) bool {
	var cond any
	if rule != nil && rule.LogResOn != nil {
		cond = rule.LogResOn
	} else if e != nil && e.cfg != nil && e.cfg.Default.LogResponseOn != nil {
		cond = e.cfg.Default.LogResponseOn
	} else {
		cond = "all"
	}

	if cond == nil {
		return true
	}

	switch v := cond.(type) {
	case string:
		lower := strings.ToLower(strings.TrimSpace(v))
		if lower == "" || lower == "all" {
			return true
		}
		if lower == "error_only" {
			return statusCode >= 400
		}
		return true
	case []int:
		return slices.Contains(v, statusCode)
	case []any:
		for _, item := range v {
			if num, ok := item.(int); ok && num == statusCode {
				return true
			}
		}
		return false
	default:
		return true
	}
}
