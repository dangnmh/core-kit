package ptr_test

import (
	"testing"

	"github.com/dangnmh/core-kit/pkg/ptr"
)

func TestPtr_To(t *testing.T) {
	t.Parallel()

	s := "hello"
	ps := ptr.To(s)
	if ps == nil || *ps != "hello" {
		t.Fatalf("expected pointer to 'hello', got %v", ps)
	}

	n := 42
	pn := ptr.To(n)
	if pn == nil || *pn != 42 {
		t.Fatalf("expected pointer to 42, got %v", pn)
	}
}

func TestPtr_Value(t *testing.T) {
	t.Parallel()

	var nilStr *string
	if val := ptr.Value(nilStr, "default"); val != "default" {
		t.Fatalf("expected default, got %s", val)
	}

	actual := "custom"
	if val := ptr.Value(&actual, "default"); val != "custom" {
		t.Fatalf("expected custom, got %s", val)
	}
}

func TestPtr_ValueOrZero(t *testing.T) {
	t.Parallel()

	var nilInt *int
	if val := ptr.ValueOrZero(nilInt); val != 0 {
		t.Fatalf("expected 0, got %d", val)
	}

	actual := 123
	if val := ptr.ValueOrZero(&actual); val != 123 {
		t.Fatalf("expected 123, got %d", val)
	}
}

func TestPtr_Equal(t *testing.T) {
	t.Parallel()

	var p1, p2 *int
	if !ptr.Equal(p1, p2) {
		t.Fatal("expected both nil to be equal")
	}

	val1, val2 := 10, 10
	val3 := 20

	if !ptr.Equal(&val1, &val2) {
		t.Fatal("expected pointers to 10 to be equal")
	}

	if ptr.Equal(&val1, &val3) {
		t.Fatal("expected pointers to 10 and 20 to NOT be equal")
	}

	if ptr.Equal(&val1, nil) {
		t.Fatal("expected pointer and nil to NOT be equal")
	}
}
