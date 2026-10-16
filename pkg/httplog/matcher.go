package httplog

import (
	"errors"
	"strings"
)

// ErrNilConfig is returned when NewRouteMatcher receives a nil config.
var ErrNilConfig = errors.New("httplog config cannot be nil")

// RouteMatcher performs route matching against precompiled HTTP path and method rules.
type RouteMatcher struct {
	routes       []*CompiledRouteRule
	inbound      []*CompiledRouteRule
	outbound     []*CompiledRouteRule
	hasDirection bool
}

// compileRouteRules compiles a slice of RuleConfig into []*CompiledRouteRule.
func compileRouteRules(rules []RuleConfig, defaultLogResOn any) []*CompiledRouteRule {
	if len(rules) == 0 {
		return nil
	}

	compiledRoutes := make([]*CompiledRouteRule, 0, len(rules))
	for _, r := range rules {
		cr := &CompiledRouteRule{
			PathPattern: r.Path,
			Methods:     make(map[string]struct{}),
			Enabled:     r.IsEnabled(),
			LogResOn:    defaultLogResOn,
		}

		if cr.LogResOn == nil || cr.LogResOn == "" {
			cr.LogResOn = "all"
		}

		// Compile path prefix logic
		if strings.HasSuffix(r.Path, "/*") {
			cr.IsPrefix = true
			cr.PrefixPrefix = strings.TrimSuffix(r.Path, "/*")
		} else if strings.HasSuffix(r.Path, "*") {
			cr.IsPrefix = true
			cr.PrefixPrefix = strings.TrimSuffix(r.Path, "*")
		}

		// Compile methods whitelist
		for _, m := range r.Methods {
			upper := strings.ToUpper(strings.TrimSpace(m))
			if upper != "*" && upper != "" {
				cr.Methods[upper] = struct{}{}
			}
		}

		// Compile shared route ignore fields
		shared := CompileFieldRules(r.IgnoreFields)

		// Compile request rules
		if r.Request != nil {
			reqCompiled := CompileFieldRules(r.Request.IgnoreFields)
			cr.ReqRules = make([]CompiledFieldRule, 0, len(shared)+len(reqCompiled))
			cr.ReqRules = append(cr.ReqRules, shared...)
			cr.ReqRules = append(cr.ReqRules, reqCompiled...)
		} else {
			cr.ReqRules = make([]CompiledFieldRule, len(shared))
			copy(cr.ReqRules, shared)
		}

		// Compile response rules
		if r.Response != nil {
			resCompiled := CompileFieldRules(r.Response.IgnoreFields)
			cr.ResRules = make([]CompiledFieldRule, 0, len(shared)+len(resCompiled))
			cr.ResRules = append(cr.ResRules, shared...)
			cr.ResRules = append(cr.ResRules, resCompiled...)

			if r.Response.LogResponseOn != nil && r.Response.LogResponseOn != "" {
				cr.LogResOn = r.Response.LogResponseOn
			}
		} else {
			cr.ResRules = make([]CompiledFieldRule, len(shared))
			copy(cr.ResRules, shared)
		}

		compiledRoutes = append(compiledRoutes, cr)
	}

	return compiledRoutes
}

// NewRouteMatcher compiles all routes and field rules defined in the provided Config.
func NewRouteMatcher(cfg *Config) (*RouteMatcher, error) {
	if cfg == nil {
		return nil, ErrNilConfig
	}

	routes := compileRouteRules(cfg.Rules, cfg.Default.LogResponseOn)
	inbound := compileRouteRules(cfg.Inbound, cfg.Default.LogResponseOn)
	outbound := compileRouteRules(cfg.Outbound, cfg.Default.LogResponseOn)
	hasDirection := len(cfg.Inbound) > 0 || len(cfg.Outbound) > 0

	return &RouteMatcher{
		routes:       routes,
		inbound:      inbound,
		outbound:     outbound,
		hasDirection: hasDirection,
	}, nil
}

// Routes returns the slice of compiled route rules.
func (rm *RouteMatcher) Routes() []*CompiledRouteRule {
	if rm == nil {
		return nil
	}
	return rm.routes
}

// InboundRoutes returns the slice of compiled inbound route rules.
func (rm *RouteMatcher) InboundRoutes() []*CompiledRouteRule {
	if rm == nil {
		return nil
	}
	return rm.inbound
}

// OutboundRoutes returns the slice of compiled outbound route rules.
func (rm *RouteMatcher) OutboundRoutes() []*CompiledRouteRule {
	if rm == nil {
		return nil
	}
	return rm.outbound
}

func matchRouteList(routes []*CompiledRouteRule, method, path string) (*CompiledRouteRule, bool) {
	if len(routes) == 0 {
		return nil, false
	}

	method = strings.ToUpper(strings.TrimSpace(method))

	for _, r := range routes {
		// Check method restriction
		if len(r.Methods) > 0 {
			if _, ok := r.Methods[method]; !ok {
				continue
			}
		}

		// Check path matching
		if r.IsPrefix {
			if r.PrefixPrefix == "" {
				return r, true
			}
			if strings.HasSuffix(r.PathPattern, "/*") {
				if path == r.PrefixPrefix || strings.HasPrefix(path, r.PrefixPrefix+"/") {
					return r, true
				}
			} else {
				if strings.HasPrefix(path, r.PrefixPrefix) {
					return r, true
				}
			}
		} else {
			if r.PathPattern == path || strings.TrimSuffix(r.PathPattern, "/") == strings.TrimSuffix(path, "/") {
				return r, true
			}
		}
	}

	return nil, false
}

// Match finds the first compiled route matching the given HTTP method and request path.
func (rm *RouteMatcher) Match(method, path string) (*CompiledRouteRule, bool) {
	if rm == nil {
		return nil, false
	}
	return matchRouteList(rm.routes, method, path)
}

// MatchInbound finds the first compiled inbound route matching the given HTTP method and request path.
// When directional rules (inbound/outbound) are configured, matching is strictly restricted to inbound routes.
// If no directional rules are configured (legacy config), it falls back to general route matching.
func (rm *RouteMatcher) MatchInbound(method, path string) (*CompiledRouteRule, bool) {
	if rm == nil {
		return nil, false
	}
	if rm.hasDirection {
		return matchRouteList(rm.inbound, method, path)
	}
	return rm.Match(method, path)
}

// MatchOutbound finds the first compiled outbound route matching the given HTTP method and request path.
// When directional rules (inbound/outbound) are configured, matching is strictly restricted to outbound routes.
// If no directional rules are configured (legacy config), it falls back to general route matching.
func (rm *RouteMatcher) MatchOutbound(method, path string) (*CompiledRouteRule, bool) {
	if rm == nil {
		return nil, false
	}
	if rm.hasDirection {
		return matchRouteList(rm.outbound, method, path)
	}
	return rm.Match(method, path)
}
