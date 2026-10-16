package ptr

// To returns a pointer to the passed value v.
func To[T any](v T) *T {
	return &v
}

// Value returns the value of pointer p, or defaultVal if p is nil.
func Value[T any](p *T, defaultVal T) T {
	if p == nil {
		return defaultVal
	}
	return *p
}

// ValueOrZero returns the value of pointer p, or the zero value of T if p is nil.
func ValueOrZero[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}

// Equal returns true if both pointers are nil or point to equal values.
func Equal[T comparable](a, b *T) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
