package strutil_test

import (
	"testing"

	"github.com/dangnmh/core-kit/pkg/strutil"
)

func TestTruncateRunes(t *testing.T) {
	t.Parallel()

	// Vietnamese multi-byte string
	s := "Xin chào thế giới"
	if strutil.TruncateRunes(s, 8) != "Xin chào" {
		t.Fatalf("expected 'Xin chào', got %q", strutil.TruncateRunes(s, 8))
	}

	if strutil.TruncateRunes(s, 0) != "" {
		t.Fatal("expected empty string")
	}

	if strutil.TruncateRunes(s, 100) != s {
		t.Fatal("expected full string")
	}
}
