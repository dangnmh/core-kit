package maputil_test

import (
	"testing"

	"github.com/dangnmh/core-kit/pkg/maputil"
)

func TestMapUtil_Extractions(t *testing.T) {
	t.Parallel()

	data := map[string]any{
		"str":   "hello",
		"num":   42,
		"flt":   3.14,
		"flt32": float32(1.5),
		"truth": true,
	}

	if maputil.GetString(data, "str") != "hello" {
		t.Fatal("expected hello")
	}
	if maputil.GetString(data, "missing", "fallback") != "fallback" {
		t.Fatal("expected fallback")
	}
	if maputil.GetInt(data, "num") != 42 {
		t.Fatal("expected 42")
	}
	if maputil.GetInt(data, "missing", 99) != 99 {
		t.Fatal("expected 99")
	}
	if maputil.GetFloat(data, "flt") != 3.14 {
		t.Fatal("expected 3.14")
	}
	if maputil.GetFloat(data, "flt32") != 1.5 {
		t.Fatal("expected 1.5")
	}
	if !maputil.GetBool(data, "truth") {
		t.Fatal("expected true")
	}
	if maputil.GetBool(data, "missing", true) != true {
		t.Fatal("expected default true")
	}
}
