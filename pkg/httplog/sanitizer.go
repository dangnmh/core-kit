package httplog

import (
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strings"
)

// MaxRecursionDepth limits the recursion depth of JSON traversal to avoid stack overflow.
const MaxRecursionDepth = 32

// pathSegment records the field name and whether it points to an array during traversal.
type pathSegment struct {
	name    string
	isArray bool
}

// SummarizeBinary formats a summary descriptor for binary content.
func SummarizeBinary(contentType string, size int) string {
	return fmt.Sprintf("[BINARY %s, size: %d bytes]", contentType, size)
}

// SanitizeForm parses application/x-www-form-urlencoded body bytes, applies matching field rules,
// and returns the re-encoded form query string.
func SanitizeForm(body []byte, rules []CompiledFieldRule, maskVal string) string {
	if len(body) == 0 {
		return ""
	}

	values, err := url.ParseQuery(string(body))
	if err != nil {
		return string(body)
	}

	for k, vals := range values {
		var matchedRule *CompiledFieldRule

		// 1. Check exact/path match
		for i := range rules {
			r := &rules[i]
			if r.Type == PatternPath && r.RawPattern == k {
				matchedRule = r
				break
			}
		}

		// 2. Check prefix wildcard match
		if matchedRule == nil {
			for i := range rules {
				r := &rules[i]
				if r.Type == PatternPrefixWild {
					prefix := strings.ReplaceAll(r.PrefixPath, "[]", "")
					if prefix == k || strings.HasPrefix(k, prefix+".") {
						matchedRule = r
						break
					}
				}
			}
		}

		// 3. Check global key match
		if matchedRule == nil {
			for i := range rules {
				r := &rules[i]
				if r.Type == PatternGlobalKey && r.GlobalKey == k {
					matchedRule = r
					break
				}
			}
		}

		if matchedRule != nil {
			if matchedRule.Strategy == StrategyRemove {
				delete(values, k)
				continue
			}
			for i, v := range vals {
				masked, _ := ApplyStrategy(v, matchedRule.Strategy, maskVal)
				values[k][i] = fmt.Sprint(masked)
			}
		}
	}

	return values.Encode()
}

// SanitizeJSON recursively traverses a JSON body and masks or removes sensitive fields
// matching the provided compiled field rules. If the body is malformed or invalid JSON,
// it safely returns the raw body string without crashing.
func SanitizeJSON(body []byte, rules []CompiledFieldRule, maskVal string) string {
	if len(body) == 0 {
		return ""
	}

	var root any
	if err := json.Unmarshal(body, &root); err != nil {
		return string(body)
	}

	traverseJSON(root, nil, rules, maskVal, 0)

	sanitized, err := json.Marshal(root)
	if err != nil {
		return string(body)
	}

	return string(sanitized)
}

func traverseJSON(node any, currentPath []pathSegment, rules []CompiledFieldRule, maskVal string, depth int) {
	if depth > MaxRecursionDepth || node == nil {
		return
	}

	switch val := node.(type) {
	case map[string]any:
		var keysToRemove []string

		for k, child := range val {
			childIsArray := false
			if _, ok := child.([]any); ok {
				childIsArray = true
			}

			seg := pathSegment{name: k, isArray: childIsArray}
			fieldSegments := append(slices.Clone(currentPath), seg)

			// Compute dotPath (e.g. "data.user.profile")
			names := make([]string, len(fieldSegments))
			for i, s := range fieldSegments {
				names[i] = s.name
			}
			dotPath := strings.Join(names, ".")

			// Check matching rules
			matchedRule, isArrayTarget := findMatchingFieldRule(k, dotPath, fieldSegments, rules)
			if matchedRule != nil {
				if matchedRule.Strategy == StrategyRemove {
					if isArrayTarget {
						val[k] = []any{}
					} else {
						keysToRemove = append(keysToRemove, k)
					}
					continue
				}

				if isArrayTarget {
					if arr, ok := child.([]any); ok {
						newArr := make([]any, len(arr))
						for idx, item := range arr {
							masked, _ := ApplyStrategy(fmt.Sprint(item), matchedRule.Strategy, maskVal)
							newArr[idx] = masked
						}
						val[k] = newArr
					}
					continue
				}

				if matchedRule.Type == PatternPrefixWild {
					val[k] = maskVal
					continue
				}

				if !isContainer(child) {
					masked, remove := ApplyStrategy(fmt.Sprint(child), matchedRule.Strategy, maskVal)
					if remove {
						keysToRemove = append(keysToRemove, k)
					} else {
						val[k] = masked
					}
					continue
				}

				// If container matched a direct redact or mask rule
				val[k] = maskVal
				continue
			}

			// If no direct rule matched, traverse down into container
			if arr, ok := child.([]any); ok {
				for _, item := range arr {
					traverseJSON(item, fieldSegments, rules, maskVal, depth+1)
				}
				continue
			}

			if _, ok := child.(map[string]any); ok {
				traverseJSON(child, fieldSegments, rules, maskVal, depth+1)
				continue
			}
		}

		for _, k := range keysToRemove {
			delete(val, k)
		}

	case []any:
		for _, item := range val {
			traverseJSON(item, currentPath, rules, maskVal, depth+1)
		}
	}
}

func isContainer(v any) bool {
	switch v.(type) {
	case map[string]any, []any:
		return true
	default:
		return false
	}
}

func findMatchingFieldRule(
	key string,
	dotPath string,
	segments []pathSegment,
	rules []CompiledFieldRule,
) (*CompiledFieldRule, bool) {
	// 1. Check exact PatternPath matching
	for i := range rules {
		r := &rules[i]
		if r.Type == PatternPath {
			isTarget, isArrayTarget := matchPathRule(segments, r)
			if isTarget {
				return r, isArrayTarget
			}
		}
	}

	// 2. Check PatternPrefixWild matching
	for i := range rules {
		r := &rules[i]
		if r.Type == PatternPrefixWild {
			prefix := strings.ReplaceAll(r.PrefixPath, "[]", "")
			if prefix == dotPath || strings.HasPrefix(dotPath, prefix+".") {
				return r, false
			}
		}
	}

	// 3. Check PatternGlobalKey matching
	for i := range rules {
		r := &rules[i]
		if r.Type == PatternGlobalKey && r.GlobalKey == key {
			return r, false
		}
	}

	return nil, false
}

func matchPathRule(segments []pathSegment, r *CompiledFieldRule) (isTarget bool, isArrayTarget bool) {
	if r.Type != PatternPath || len(r.Tokens) == 0 {
		return false, false
	}

	if len(segments) != len(r.Tokens) {
		return false, false
	}

	for i := range segments {
		if segments[i].name != r.Tokens[i].Name {
			return false, false
		}
		// For intermediate segments, array status must match
		if i < len(segments)-1 {
			if r.Tokens[i].IsArray != segments[i].isArray {
				return false, false
			}
		}
	}

	lastToken := r.Tokens[len(r.Tokens)-1]
	if lastToken.IsArray {
		// Rule explicitly targets array elements; node must be an array
		if !segments[len(segments)-1].isArray {
			return false, false
		}
		return true, true
	}

	return true, false
}
