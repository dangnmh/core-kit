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

func TestSlugify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"AutoForm AI", "autoform-ai"},
		{"Ứng dụng Quản lý Hóa đơn Đơn Giản", "ung-dung-quan-ly-hoa-don-don-gian"},
		{"Đà Nẵng Tech Hub 2026!", "da-nang-tech-hub-2026"},
		{"  ---Messy   String___With Symbols #@! --- ", "messy-string-with-symbols"},
		{"VibeInvoice AI — Trích Xuất Hóa Đơn Đỏ!", "vibeinvoice-ai-trich-xuat-hoa-don-do"},
		{"   ", ""},
		{"!@#$%^&*()", ""},
	}

	for _, tt := range tests {
		actual := strutil.Slugify(tt.input)
		if actual != tt.expected {
			t.Fatalf("Slugify(%q) = %q, expected %q", tt.input, actual, tt.expected)
		}
	}
}
