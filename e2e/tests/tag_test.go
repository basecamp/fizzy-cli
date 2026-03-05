package tests

import (
	"testing"

	"github.com/basecamp/fizzy-cli/e2e/harness"
)

func TestTagList(t *testing.T) {
	h := harness.New(t)

	t.Run("returns list of tags", func(t *testing.T) {
		result := h.Run("tag", "list")

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if result.Response == nil {
			t.Fatalf("expected JSON response, got nil\nstdout: %s", result.Stdout)
		}

		if !result.Response.OK {
			t.Error("expected ok=true")
		}

		arr := result.GetDataArray()
		if arr == nil {
			t.Error("expected data to be an array")
		}
	})

	t.Run("supports --page option", func(t *testing.T) {
		result := h.Run("tag", "list", "--page", "1")

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d", harness.ExitSuccess, result.ExitCode)
		}

		if result.Response == nil || !result.Response.OK {
			t.Error("expected successful response")
		}
	})

	t.Run("supports --all flag", func(t *testing.T) {
		result := h.Run("tag", "list", "--all")

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d", harness.ExitSuccess, result.ExitCode)
		}
	})
}
