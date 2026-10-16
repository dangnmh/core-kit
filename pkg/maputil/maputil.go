package maputil

// GetString extracts a string value from a map, or returns empty string if not found or not a string.
func GetString(m map[string]any, key string, defVal ...string) string {
	if m == nil {
		if len(defVal) > 0 {
			return defVal[0]
		}
		return ""
	}
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	if len(defVal) > 0 {
		return defVal[0]
	}
	return ""
}

// GetFloat extracts a float64 value from a map, handling float64, float32, and int representations.
func GetFloat(m map[string]any, key string, defVal ...float64) float64 {
	d := 0.0
	if len(defVal) > 0 {
		d = defVal[0]
	}
	if m == nil {
		return d
	}
	v, ok := m[key]
	if !ok || v == nil {
		return d
	}
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	default:
		return d
	}
}

// GetInt extracts an int value from a map.
func GetInt(m map[string]any, key string, defVal ...int) int {
	d := 0
	if len(defVal) > 0 {
		d = defVal[0]
	}
	if m == nil {
		return d
	}
	v, ok := m[key]
	if !ok || v == nil {
		return d
	}
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	default:
		return d
	}
}

// GetBool extracts a bool value from a map.
func GetBool(m map[string]any, key string, defVal ...bool) bool {
	d := false
	if len(defVal) > 0 {
		d = defVal[0]
	}
	if m == nil {
		return d
	}
	v, ok := m[key]
	if !ok || v == nil {
		return d
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return d
}
