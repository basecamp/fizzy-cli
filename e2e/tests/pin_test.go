package tests

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/basecamp/fizzy-cli/e2e/harness"
)

func TestPinActions(t *testing.T) {
	h := harness.New(t)
	defer h.Cleanup.CleanupAll(h)

	boardID := createTestBoard(t, h)

	// Create a card for pin tests
	title := fmt.Sprintf("Pin Test Card %d", time.Now().UnixNano())
	result := h.Run("card", "create", "--board", boardID, "--title", title)
	if result.ExitCode != harness.ExitSuccess {
		t.Fatalf("failed to create test card: %s\nstdout: %s", result.Stderr, result.Stdout)
	}
	cardNumber := result.GetNumberFromLocation()
	if cardNumber == 0 {
		cardNumber = result.GetDataInt("number")
	}
	if cardNumber == 0 {
		t.Fatalf("failed to get card number from create (location: %s)", result.GetLocation())
	}
	h.Cleanup.AddCard(cardNumber)
	cardStr := strconv.Itoa(cardNumber)

	t.Run("pin card", func(t *testing.T) {
		result := h.Run("card", "pin", cardStr)

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if !result.Response.OK {
			t.Errorf("expected ok=true, error: %+v", result.Response.Error)
		}
	})

	t.Run("pin list includes pinned card", func(t *testing.T) {
		result := h.Run("pin", "list")

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if !result.Response.OK {
			t.Errorf("expected ok=true, error: %+v", result.Response.Error)
		}

		arr := result.GetDataArray()
		if arr == nil {
			t.Fatal("expected data to be an array")
		}

		// Find our pinned card in the list
		found := false
		for _, item := range arr {
			card, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if num, ok := card["number"].(float64); ok && int(num) == cardNumber {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected pinned card #%d to appear in pin list", cardNumber)
		}
	})

	t.Run("unpin card", func(t *testing.T) {
		result := h.Run("card", "unpin", cardStr)

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if !result.Response.OK {
			t.Errorf("expected ok=true, error: %+v", result.Response.Error)
		}
	})

	t.Run("pin list excludes unpinned card", func(t *testing.T) {
		result := h.Run("pin", "list")

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		arr := result.GetDataArray()
		if arr == nil {
			// Empty array is fine - card should not be there
			return
		}

		for _, item := range arr {
			card, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if num, ok := card["number"].(float64); ok && int(num) == cardNumber {
				t.Errorf("expected card #%d to NOT appear in pin list after unpinning", cardNumber)
			}
		}
	})
}

func TestPinList(t *testing.T) {
	h := harness.New(t)

	t.Run("returns list of pinned cards", func(t *testing.T) {
		result := h.Run("pin", "list")

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if result.Response == nil {
			t.Fatalf("expected JSON response, got nil\nstdout: %s", result.Stdout)
		}

		if !result.Response.OK {
			t.Error("expected ok=true")
		}

		// Data should be an array
		arr := result.GetDataArray()
		if arr == nil {
			t.Error("expected data to be an array")
		}
	})
}

func TestPinNotFound(t *testing.T) {
	h := harness.New(t)

	t.Run("pin non-existent card fails", func(t *testing.T) {
		result := h.Run("card", "pin", "999999999")

		if result.ExitCode == harness.ExitSuccess {
			t.Error("expected non-zero exit code for non-existent card")
		}

		if result.Response != nil && result.Response.OK {
			t.Error("expected ok=false")
		}
	})

	t.Run("unpin non-existent card fails", func(t *testing.T) {
		result := h.Run("card", "unpin", "999999999")

		if result.ExitCode == harness.ExitSuccess {
			t.Error("expected non-zero exit code for non-existent card")
		}

		if result.Response != nil && result.Response.OK {
			t.Error("expected ok=false")
		}
	})
}
