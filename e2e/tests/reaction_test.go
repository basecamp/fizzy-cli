package tests

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/basecamp/fizzy-cli/e2e/harness"
)

// createTestComment creates a comment for reaction tests
func createTestComment(t *testing.T, h *harness.Harness, cardNumber int) string {
	t.Helper()
	cardStr := strconv.Itoa(cardNumber)
	body := fmt.Sprintf("Comment for reactions %d", time.Now().UnixNano())
	result := h.Run("comment", "create", "--card", cardStr, "--body", body)
	if result.ExitCode != harness.ExitSuccess {
		t.Fatalf("failed to create test comment: %s\nstdout: %s", result.Stderr, result.Stdout)
	}
	// Create returns location - extract ID from it
	commentID := result.GetIDFromLocation()
	if commentID == "" {
		// Try data.id as fallback
		commentID = result.GetDataString("id")
	}
	if commentID == "" {
		t.Fatalf("no comment ID returned (location: %s)", result.GetLocation())
	}
	h.Cleanup.AddComment(commentID, cardNumber)
	return commentID
}

func TestReactionList(t *testing.T) {
	h := harness.New(t)
	defer h.Cleanup.CleanupAll(h)

	boardID := createTestBoard(t, h)
	cardNumber := createTestCard(t, h, boardID)
	commentID := createTestComment(t, h, cardNumber)
	cardStr := strconv.Itoa(cardNumber)

	t.Run("returns list of reactions for comment", func(t *testing.T) {
		result := h.Run("reaction", "list", "--card", cardStr, "--comment", commentID)

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

	t.Run("fails without --card option", func(t *testing.T) {
		result := h.Run("reaction", "list", "--comment", commentID)

		if result.ExitCode == harness.ExitSuccess {
			t.Error("expected non-zero exit code for missing required option")
		}
	})

}

// TestCardReactionList tests listing reactions directly on a card (no --comment flag)
func TestCardReactionList(t *testing.T) {
	h := harness.New(t)
	defer h.Cleanup.CleanupAll(h)

	boardID := createTestBoard(t, h)
	cardNumber := createTestCard(t, h, boardID)
	cardStr := strconv.Itoa(cardNumber)

	t.Run("returns list of reactions for card", func(t *testing.T) {
		result := h.Run("reaction", "list", "--card", cardStr)

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

	t.Run("fails without --card option", func(t *testing.T) {
		result := h.Run("reaction", "list")

		if result.ExitCode == harness.ExitSuccess {
			t.Error("expected non-zero exit code for missing required option")
		}
	})
}

func TestReactionCRUD(t *testing.T) {
	h := harness.New(t)
	defer h.Cleanup.CleanupAll(h)

	boardID := createTestBoard(t, h)
	cardNumber := createTestCard(t, h, boardID)
	commentID := createTestComment(t, h, cardNumber)
	cardStr := strconv.Itoa(cardNumber)

	var reactionID string

	t.Run("create reaction", func(t *testing.T) {
		result := h.Run("reaction", "create", "--card", cardStr, "--comment", commentID, "--content", "👍")

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

		// Note: Reaction create returns success but no location or data
		// We need to list reactions to get the ID for cleanup/deletion
		listResult := h.Run("reaction", "list", "--card", cardStr, "--comment", commentID)
		if listResult.ExitCode == harness.ExitSuccess && listResult.Response != nil {
			arr := listResult.GetDataArray()
			if len(arr) > 0 {
				// Get the last reaction (most recently created)
				lastReaction := arr[len(arr)-1].(map[string]any)
				if id, ok := lastReaction["id"].(string); ok {
					reactionID = id
					h.Cleanup.AddReaction(reactionID, cardNumber, commentID)
				}
			}
		}
		if reactionID == "" {
			t.Log("Warning: could not get reaction ID for cleanup")
		}
	})

	t.Run("delete reaction", func(t *testing.T) {
		if reactionID == "" {
			t.Skip("no reaction ID from create test")
		}

		result := h.Run("reaction", "delete", reactionID, "--card", cardStr, "--comment", commentID)

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
		h.Cleanup.Reactions = nil
	})
}

func TestReactionCreateMissingContent(t *testing.T) {
	h := harness.New(t)
	defer h.Cleanup.CleanupAll(h)

	boardID := createTestBoard(t, h)
	cardNumber := createTestCard(t, h, boardID)
	commentID := createTestComment(t, h, cardNumber)
	cardStr := strconv.Itoa(cardNumber)

	t.Run("fails without --content option for comment reaction", func(t *testing.T) {
		result := h.Run("reaction", "create", "--card", cardStr, "--comment", commentID)

		if result.ExitCode == harness.ExitSuccess {
			t.Error("expected non-zero exit code for missing required option")
		}
	})

	t.Run("fails without --content option for card reaction", func(t *testing.T) {
		result := h.Run("reaction", "create", "--card", cardStr)

		if result.ExitCode == harness.ExitSuccess {
			t.Error("expected non-zero exit code for missing required option")
		}
	})
}

// TestCardReactionCRUD tests creating and deleting reactions directly on cards
func TestCardReactionCRUD(t *testing.T) {
	h := harness.New(t)
	defer h.Cleanup.CleanupAll(h)

	boardID := createTestBoard(t, h)
	cardNumber := createTestCard(t, h, boardID)
	cardStr := strconv.Itoa(cardNumber)

	var reactionID string

	t.Run("create card reaction", func(t *testing.T) {
		result := h.Run("reaction", "create", "--card", cardStr, "--content", "🎉")

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

		// List reactions to get the ID for cleanup/deletion
		listResult := h.Run("reaction", "list", "--card", cardStr)
		if listResult.ExitCode == harness.ExitSuccess && listResult.Response != nil {
			arr := listResult.GetDataArray()
			if len(arr) > 0 {
				// Get the last reaction (most recently created)
				lastReaction := arr[len(arr)-1].(map[string]any)
				if id, ok := lastReaction["id"].(string); ok {
					reactionID = id
					h.Cleanup.AddCardReaction(reactionID, cardNumber)
				}
			}
		}
		if reactionID == "" {
			t.Log("Warning: could not get reaction ID for cleanup")
		}
	})

	t.Run("delete card reaction", func(t *testing.T) {
		if reactionID == "" {
			t.Skip("no reaction ID from create test")
		}

		result := h.Run("reaction", "delete", reactionID, "--card", cardStr)

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
		h.Cleanup.Reactions = nil
	})
}
