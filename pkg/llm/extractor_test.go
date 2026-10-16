package llm

import (
	"strings"
	"testing"
	"unicode/utf8"
)

type TestOutput struct {
	Thought string `json:"thought"`
	Action  string `json:"action"`
	Score   int    `json:"score"`
}

func TestExtractJSON_DirectJSON(t *testing.T) {
	raw := `{"thought": "test thought", "action": "mine", "score": 10}`
	out, err := ExtractJSON[TestOutput](raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Action != "mine" || out.Score != 10 {
		t.Errorf("unexpected output: %+v", out)
	}
}

func TestExtractJSON_MarkdownFence(t *testing.T) {
	raw := "Here is my decision:\n```json\n{\n  \"thought\": \"fence test\",\n  \"action\": \"craft\",\n  \"score\": 5\n}\n```\nHope that helps!"
	out, err := ExtractJSON[TestOutput](raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Action != "craft" || out.Score != 5 {
		t.Errorf("unexpected output: %+v", out)
	}
}

func TestExtractJSON_ThinkingTags(t *testing.T) {
	raw := "<think>\nLet me think about what action to take. I should build.\n</think>\n```json\n{\"thought\": \"thinking tags test\", \"action\": \"build\", \"score\": 8}\n```"
	out, err := ExtractJSON[TestOutput](raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Action != "build" || out.Score != 8 {
		t.Errorf("unexpected output: %+v", out)
	}
}

func TestExtractJSON_SurroundingTextFallback(t *testing.T) {
	raw := "Sure! I decided: {\"thought\": \"fallback test\", \"action\": \"attack\", \"score\": 9} Have fun!"
	out, err := ExtractJSON[TestOutput](raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Action != "attack" || out.Score != 9 {
		t.Errorf("unexpected output: %+v", out)
	}
}

func TestExtractJSON_Invalid(t *testing.T) {
	raw := "No JSON here at all!"
	_, err := ExtractJSON[TestOutput](raw)
	if err == nil {
		t.Errorf("expected error for invalid input, got nil")
	}
}

func TestExtractJSON_ErrorTruncated(t *testing.T) {
	t.Parallel()

	// 10KB input containing multi-byte UTF-8 characters without valid JSON
	raw := strings.Repeat("Tiếng Việt có dấu và ký tự đặc biệt 🚀 ", 250)
	if len(raw) < 10000 {
		t.Fatalf("expected raw length >= 10000, got %d", len(raw))
	}

	_, err := ExtractJSON[TestOutput](raw)
	if err == nil {
		t.Fatalf("expected error for invalid JSON, got nil")
	}

	errMsg := err.Error()
	if len(errMsg) >= 600 {
		t.Fatalf("expected truncated error message length < 600, got %d", len(errMsg))
	}
	if !utf8.ValidString(errMsg) {
		t.Fatalf("expected valid UTF-8 error message")
	}
}

func TestExtractJSON_TruncateRunes_Vietnamese(t *testing.T) {
	t.Parallel()

	t.Run("short vietnamese string not truncated", func(t *testing.T) {
		raw := "Xin chào thế giới 🚀 (không có json)"
		_, err := ExtractJSON[TestOutput](raw)
		if err == nil {
			t.Fatalf("expected error for invalid input, got nil")
		}
		errMsg := err.Error()
		if !utf8.ValidString(errMsg) {
			t.Fatalf("expected valid UTF-8 error message, got %s", errMsg)
		}
		if strings.Contains(errMsg, "...") {
			t.Fatalf("expected short message not to be truncated with '...', got %s", errMsg)
		}
		if !strings.Contains(errMsg, raw) {
			t.Fatalf("expected error message to contain %q, got %s", raw, errMsg)
		}
	})

	t.Run("long vietnamese string truncated safely at rune boundary", func(t *testing.T) {
		unit := "Xin chào thế giới 🚀 "
		raw := strings.Repeat(unit, 20)
		_, err := ExtractJSON[TestOutput](raw)
		if err == nil {
			t.Fatalf("expected error for invalid input, got nil")
		}
		errMsg := err.Error()
		if !utf8.ValidString(errMsg) {
			t.Fatalf("expected valid UTF-8 error message, got %s", errMsg)
		}
		if !strings.Contains(errMsg, "...") {
			t.Fatalf("expected long error message to end with '...', got %s", errMsg)
		}
		if !strings.HasPrefix(errMsg, "extract json (len=") {
			t.Fatalf("expected error prefix 'extract json (len=', got %s", errMsg)
		}
	})
}
