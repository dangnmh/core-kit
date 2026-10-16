package httplog

import (
	"strings"
)

// PatternType classifies the granularity and traversal behavior of a compiled field rule.
type PatternType int

const (
	// PatternGlobalKey matches a single field name at any nesting level.
	PatternGlobalKey PatternType = iota

	// PatternPrefixWild matches an object/subtree prefix and redacts all descendant elements.
	PatternPrefixWild

	// PatternPath matches an exact or array-indexed hierarchical path.
	PatternPath
)

// PathToken represents a single segment in a hierarchical path expression.
type PathToken struct {
	Name    string
	IsArray bool
}

// CompiledFieldRule stores the precompiled AST and strategy for a sensitive field mask rule.
type CompiledFieldRule struct {
	RawPattern string
	Type       PatternType
	Strategy   Strategy
	GlobalKey  string
	PrefixPath string
	Tokens     []PathToken
}

// CompiledRouteRule stores precompiled path, method, and masking rules for an HTTP route.
type CompiledRouteRule struct {
	PathPattern  string
	IsPrefix     bool
	PrefixPrefix string
	Methods      map[string]struct{} // empty set represents wildcard "*" (all methods)
	Enabled      bool
	ReqRules     []CompiledFieldRule
	ResRules     []CompiledFieldRule
	LogResOn     any
}

// CompileFieldRules compiles a slice of raw YAML FieldRules into high-performance CompiledFieldRules.
func CompileFieldRules(rules []FieldRule) []CompiledFieldRule {
	if len(rules) == 0 {
		return nil
	}

	res := make([]CompiledFieldRule, 0, len(rules))
	for _, r := range rules {
		trimmed := strings.TrimSpace(r.Field)
		if trimmed == "" {
			continue
		}

		strat := r.Strategy
		if strat == "" {
			strat = StrategyRedact
		}

		c := CompiledFieldRule{
			RawPattern: trimmed,
			Strategy:   strat,
		}

		// 1. Prefix Wildcard: ends with ".*" (e.g. "items.*", "user.meta.*")
		if strings.HasSuffix(trimmed, ".*") {
			c.Type = PatternPrefixWild
			c.PrefixPath = strings.TrimSuffix(trimmed, ".*")
			res = append(res, c)
			continue
		}

		// 2. Structured Path: contains "." or "[]" (e.g. "data.user.id", "items[].name")
		if strings.Contains(trimmed, ".") || strings.Contains(trimmed, "[]") {
			c.Type = PatternPath
			parts := strings.Split(trimmed, ".")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}

				if strings.HasSuffix(part, "[]") {
					c.Tokens = append(c.Tokens, PathToken{
						Name:    strings.TrimSuffix(part, "[]"),
						IsArray: true,
					})
				} else {
					c.Tokens = append(c.Tokens, PathToken{
						Name:    part,
						IsArray: false,
					})
				}
			}

			if len(c.Tokens) > 0 {
				res = append(res, c)
			}
			continue
		}

		// 3. Global Key: matches single field name anywhere in hierarchy
		c.Type = PatternGlobalKey
		c.GlobalKey = trimmed
		res = append(res, c)
	}

	return res
}
