package httplog

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/dangnmh/core-kit/pkg/logger"
)

type logRoundTripper struct {
	base   http.RoundTripper
	engine Engine
}

// NewRoundTripper returns an http.RoundTripper that intercepts outbound HTTP requests,
// sanitizes request and response bodies using the provided Engine, preserves body streams
// for the caller, and emits structured logs.
func NewRoundTripper(base http.RoundTripper, engine Engine) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return &logRoundTripper{
		base:   base,
		engine: engine,
	}
}

// RoundTrip executes a single HTTP transaction and logs the request and response
// with sensitive data redacted according to the configured Engine rules.
func (rt *logRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if rt.engine != nil && !rt.engine.LogOutboundEnabled() {
		return rt.base.RoundTrip(req)
	}

	method := req.Method
	urlStr := ""
	path := ""
	if req.URL != nil {
		urlStr = req.URL.String()
		path = req.URL.Path
	}
	if path == "" {
		path = "/"
	}

	var rule *CompiledRouteRule
	var reqRules []CompiledFieldRule
	var resRules []CompiledFieldRule

	if rt.engine != nil {
		var matched bool
		rule, matched = rt.engine.MatchOutbound(method, path)
		if !matched {
			rule, matched = rt.engine.Match(method, path)
		}
		if matched && rule != nil {
			if !rule.Enabled {
				return rt.base.RoundTrip(req)
			}
			reqRules = rule.ReqRules
			resRules = rule.ResRules
		} else {
			if !rt.engine.LogUnmatchedEnabled() {
				return rt.base.RoundTrip(req)
			}
		}
	}

	var reqBody []byte
	if req.Body != nil && req.Body != http.NoBody {
		var err error
		reqBody, err = io.ReadAll(req.Body)
		if err == nil {
			req.Body = io.NopCloser(bytes.NewBuffer(reqBody))
			req.GetBody = func() (io.ReadCloser, error) {
				return io.NopCloser(bytes.NewBuffer(reqBody)), nil
			}
		}
	}

	start := time.Now()
	res, err := rt.base.RoundTrip(req)
	latency := time.Since(start)

	if err != nil {
		logger.ErrorContext(req.Context(), "Outbound HTTP failed",
			"direction", "outbound",
			"method", method,
			"url", urlStr,
			"error", err,
			"latency_ms", latency.Milliseconds(),
		)
		return nil, err
	}

	var resBody []byte
	var statusCode int
	var resContentType string

	if res != nil {
		statusCode = res.StatusCode
		if res.Body != nil {
			var readErr error
			resBody, readErr = io.ReadAll(res.Body)
			_ = res.Body.Close()
			if readErr == nil {
				res.Body = io.NopCloser(bytes.NewBuffer(resBody))
			}
		}
		if res.Header != nil {
			resContentType = res.Header.Get("Content-Type")
		}
	}

	reqContentType := ""
	if req.Header != nil {
		reqContentType = req.Header.Get("Content-Type")
	}

	var reqSanitized string
	if rt.engine != nil {
		reqSanitized = rt.engine.SanitizeBody(reqContentType, reqBody, reqRules)
	} else {
		reqSanitized = string(reqBody)
	}

	var resSanitized string
	shouldLogRes := true
	if rt.engine != nil {
		shouldLogRes = rt.engine.ShouldLogResponse(statusCode, rule)
	}
	if shouldLogRes && len(resBody) > 0 {
		if rt.engine != nil {
			resSanitized = rt.engine.SanitizeBody(resContentType, resBody, resRules)
		} else {
			resSanitized = string(resBody)
		}
	}

	logger.InfoContext(req.Context(), "Outbound HTTP completed",
		"direction", "outbound",
		"method", method,
		"url", urlStr,
		"status", statusCode,
		"latency_ms", latency.Milliseconds(),
		"req_body", reqSanitized,
		"res_body", resSanitized,
	)

	return res, nil
}
