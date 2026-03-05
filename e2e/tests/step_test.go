package tests

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/basecamp/fizzy-cli/e2e/harness"
)

func TestStepCRUD(t *testing.T) {
	h := harness.New(t)
	defer h.Cleanup.CleanupAll(h)

	boardID := createTestBoard(t, h)
	cardNumber := createTestCard(t, h, boardID)
	cardStr := strconv.Itoa(cardNumber)

	var stepID string
	stepContent := fmt.Sprintf("Test step %d", time.Now().UnixNano())

	t.Run("create step", func(t *testing.T) {
		result := h.Run("step", "create", "--card", cardStr, "--content", stepContent)

		if result.ExitCode != harness.ExitSuccess {
			t.Fatalf("expected exit code %d, got %d\nstderr: %s\nstdout: %s",
				harness.ExitSuccess, result.ExitCode, result.Stderr, result.Stdout)
		}

		if result.Response == nil {
			t.Fatalf("expected JSON response, got nil\nstdout: %s", result.Stdout)
		}

		if !result.Response.OK {
			t.Errorf("expected ok=true, error: %+v", result.Response.Error)
		}

		// Create returns location - extract ID from it
		stepID = result.GetIDFromLocation()
		if stepID == "" {
			// Try data.id as fallback
			stepID = result.GetDataString("id")
		}
		if stepID == "" {
			t.Fatalf("expected step ID in response (location: %s)", result.GetLocation())
		}

		h.Cleanup.AddStep(stepID, cardNumber)

		// Note: Create returns location, not data with content
		// Verify the step exists via show command
		showResult := h.Run("step", "show", stepID, "--card", cardStr)
		if showResult.ExitCode == harness.ExitSuccess {
			content := showResult.GetDataString("content")
			if content != stepContent {
				t.Errorf("expected content %q, got %q", stepContent, content)
			}
		}
	})

	t.Run("create step as completed", func(t *testing.T) {
		content := fmt.Sprintf("Completed step %d", time.Now().UnixNano())
		result := h.Run("step", "create", "--card", cardStr, "--content", content, "--completed")

		if result.ExitCode != harness.ExitSuccess {
			t.Fatalf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		// Create returns location - extract ID from it
		id := result.GetIDFromLocation()
		if id == "" {
			id = result.GetDataString("id")
		}
		if id != "" {
			h.Cleanup.AddStep(id, cardNumber)
		}

		if !result.Response.OK {
			t.Error("expected ok=true")
		}

		// Note: Create returns location, not data - verify via show
		if id != "" {
			showResult := h.Run("step", "show", id, "--card", cardStr)
			if showResult.ExitCode == harness.ExitSuccess {
				completed := showResult.GetDataBool("completed")
				if !completed {
					t.Error("expected completed=true")
				}
			}
		}
	})

	t.Run("show step", func(t *testing.T) {
		if stepID == "" {
			t.Skip("no step ID from create test")
		}

		result := h.Run("step", "show", stepID, "--card", cardStr)

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if result.Response == nil {
			t.Fatal("expected JSON response")
		}

		if !result.Response.OK {
			t.Error("expected ok=true")
		}

		id := result.GetDataString("id")
		if id != stepID {
			t.Errorf("expected id %q, got %q", stepID, id)
		}
	})

	t.Run("update step content", func(t *testing.T) {
		if stepID == "" {
			t.Skip("no step ID from create test")
		}

		newContent := stepContent + " updated"
		result := h.Run("step", "update", stepID, "--card", cardStr, "--content", newContent)

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if result.Response == nil {
			t.Fatal("expected JSON response")
		}

		if !result.Response.OK {
			t.Error("expected ok=true")
		}

		content := result.GetDataString("content")
		if content != newContent {
			t.Errorf("expected content %q, got %q", newContent, content)
		}
	})

	t.Run("update step to completed", func(t *testing.T) {
		if stepID == "" {
			t.Skip("no step ID from create test")
		}

		result := h.Run("step", "update", stepID, "--card", cardStr, "--completed")

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if result.Response == nil {
			t.Fatal("expected JSON response")
		}

		if !result.Response.OK {
			t.Error("expected ok=true")
		}

		completed := result.GetDataBool("completed")
		if !completed {
			t.Error("expected completed=true")
		}
	})

	t.Run("update step to not completed", func(t *testing.T) {
		if stepID == "" {
			t.Skip("no step ID from create test")
		}

		result := h.Run("step", "update", stepID, "--card", cardStr, "--not_completed")

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if result.Response == nil {
			t.Fatal("expected JSON response")
		}

		if !result.Response.OK {
			t.Error("expected ok=true")
		}

		completed := result.GetDataBool("completed")
		if completed {
			t.Error("expected completed=false")
		}
	})

	t.Run("delete step", func(t *testing.T) {
		if stepID == "" {
			t.Skip("no step ID from create test")
		}

		result := h.Run("step", "delete", stepID, "--card", cardStr)

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if result.Response == nil {
			t.Fatal("expected JSON response")
		}

		if !result.Response.OK {
			t.Error("expected ok=true")
		}

		deleted := result.GetDataBool("deleted")
		if !deleted {
			t.Error("expected deleted=true")
		}

		// Remove from cleanup since we deleted it
		if len(h.Cleanup.Steps) > 0 {
			h.Cleanup.Steps = h.Cleanup.Steps[1:]
		}
	})
}

func TestStepCreateMissingContent(t *testing.T) {
	h := harness.New(t)
	defer h.Cleanup.CleanupAll(h)

	boardID := createTestBoard(t, h)
	cardNumber := createTestCard(t, h, boardID)
	cardStr := strconv.Itoa(cardNumber)

	t.Run("fails without --content option", func(t *testing.T) {
		result := h.Run("step", "create", "--card", cardStr)

		if result.ExitCode == harness.ExitSuccess {
			t.Error("expected non-zero exit code for missing required option")
		}
	})
}

func TestStepShowNotFound(t *testing.T) {
	h := harness.New(t)
	defer h.Cleanup.CleanupAll(h)

	boardID := createTestBoard(t, h)
	cardNumber := createTestCard(t, h, boardID)
	cardStr := strconv.Itoa(cardNumber)

	t.Run("returns not found for non-existent step", func(t *testing.T) {
		result := h.Run("step", "show", "non-existent-step-id", "--card", cardStr)

		if result.ExitCode != harness.ExitNotFound {
			t.Errorf("expected exit code %d, got %d\nstdout: %s",
				harness.ExitNotFound, result.ExitCode, result.Stdout)
		}

		if result.Response == nil {
			t.Fatal("expected JSON response")
		}

		if result.Response.OK {
			t.Error("expected ok=false")
		}
	})
}
