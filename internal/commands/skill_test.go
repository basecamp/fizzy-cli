package commands

import "testing"

func TestSanitizeLogValue(t *testing.T) {
	t.Run("strips newlines and tabs", func(t *testing.T) {
		input := "/safe/path\n/injected/line"
		result := sanitizeLogValue(input)
		if result != "/safe/path/injected/line" {
			t.Errorf("expected '/safe/path/injected/line', got '%s'", result)
		}
	})

	t.Run("strips carriage returns", func(t *testing.T) {
		input := "/safe/path\r\nOverwrite? yes"
		result := sanitizeLogValue(input)
		if result != "/safe/pathOverwrite? yes" {
			t.Errorf("expected '/safe/pathOverwrite? yes', got '%s'", result)
		}
	})

	t.Run("strips tabs", func(t *testing.T) {
		input := "/path/with\ttab"
		result := sanitizeLogValue(input)
		if result != "/path/withtab" {
			t.Errorf("expected '/path/withtab', got '%s'", result)
		}
	})

	t.Run("passes through clean paths", func(t *testing.T) {
		input := "/Users/test/.claude/skills/fizzy/SKILL.md"
		result := sanitizeLogValue(input)
		if result != input {
			t.Errorf("expected unchanged path, got '%s'", result)
		}
	})
}
