package llm

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/dangnmh/core-kit/pkg/strutil"
)

var (
	thinkTagRegex = regexp.MustCompile(`(?s)<(?:think|thought)>.*?</(?:think|thought)>`)
	fenceRegex    = regexp.MustCompile("(?s)```(?:json)?\\s*([\\s\\S]*?)\\s*```")
)

// CleanText removes reasoning / thinking tags (<think>...</think>) from LLM raw outputs.
func CleanText(content string) string {
	text := thinkTagRegex.ReplaceAllString(content, "")
	return strings.TrimSpace(text)
}

// formatTruncatedError formats error messages using shared pkg/strutil rune truncation.
func formatTruncatedError(content string, maxRunes int, err error) error {
	truncated := strutil.TruncateRunes(content, maxRunes)
	if len(content) > len(truncated) {
		truncated += "..."
	}
	return fmt.Errorf("extract json (len=%d, head=%q): %w", len(content), truncated, err)
}

// ExtractJSON parses raw text from LLM response into any target struct T.
// Handles thinking tags, markdown code blocks (```json ... ```), and standalone JSON objects.
func ExtractJSON[T any](content string) (*T, error) {
	text := CleanText(content)
	if text == "" {
		return nil, fmt.Errorf("empty llm response content")
	}

	var lastErr error

	// 1. Direct JSON unmarshaling
	var direct T
	if err := json.Unmarshal([]byte(text), &direct); err == nil {
		return &direct, nil
	} else {
		lastErr = err
	}

	// 2. Markdown code fences: ```json ... ``` or ``` ... ```
	matches := fenceRegex.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) > 1 {
			block := strings.TrimSpace(match[1])
			var parsed T
			if err := json.Unmarshal([]byte(block), &parsed); err == nil {
				return &parsed, nil
			} else {
				lastErr = err
			}
		}
	}

	// 3. Fallback: Search for outer curly braces { ... }
	firstBrace := strings.Index(text, "{")
	lastBrace := strings.LastIndex(text, "}")
	if firstBrace != -1 && lastBrace != -1 && lastBrace > firstBrace {
		candidate := strings.TrimSpace(text[firstBrace : lastBrace+1])
		var parsed T
		if err := json.Unmarshal([]byte(candidate), &parsed); err == nil {
			return &parsed, nil
		} else {
			lastErr = err
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no valid JSON block found")
	}

	return nil, formatTruncatedError(content, 256, lastErr)
}
