package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/basecamp/fizzy-cli/e2e/harness"
)

// createTestCard creates a card for comment tests and adds it to cleanup
func createTestCard(t *testing.T, h *harness.Harness, boardID string) int {
	t.Helper()
	title := fmt.Sprintf("Comment Test Card %d", time.Now().UnixNano())
	result := h.Run("card", "create", "--board", boardID, "--title", title)
	if result.ExitCode != harness.ExitSuccess {
		t.Fatalf("failed to create test card: %s\nstdout: %s", result.Stderr, result.Stdout)
	}
	// Create returns location - extract number from it
	cardNumber := result.GetNumberFromLocation()
	if cardNumber == 0 {
		// Try data.number as fallback
		cardNumber = result.GetDataInt("number")
	}
	if cardNumber == 0 {
		t.Fatalf("no card number returned (location: %s)", result.GetLocation())
	}
	h.Cleanup.AddCard(cardNumber)
	return cardNumber
}

func TestCommentList(t *testing.T) {
	h := harness.New(t)
	defer h.Cleanup.CleanupAll(h)

	boardID := createTestBoard(t, h)
	cardNumber := createTestCard(t, h, boardID)
	cardStr := strconv.Itoa(cardNumber)

	t.Run("returns list of comments for card", func(t *testing.T) {
		result := h.Run("comment", "list", "--card", cardStr)

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

	t.Run("supports --all flag", func(t *testing.T) {
		result := h.Run("comment", "list", "--card", cardStr, "--all")

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d", harness.ExitSuccess, result.ExitCode)
		}
	})

	t.Run("fails without --card option", func(t *testing.T) {
		result := h.Run("comment", "list")

		if result.ExitCode == harness.ExitSuccess {
			t.Error("expected non-zero exit code for missing required option")
		}
	})
}

func TestCommentCRUD(t *testing.T) {
	h := harness.New(t)
	defer h.Cleanup.CleanupAll(h)

	boardID := createTestBoard(t, h)
	cardNumber := createTestCard(t, h, boardID)
	cardStr := strconv.Itoa(cardNumber)

	var commentID string
	commentBody := fmt.Sprintf("Test comment %d", time.Now().UnixNano())

	t.Run("create comment with body", func(t *testing.T) {
		result := h.Run("comment", "create", "--card", cardStr, "--body", commentBody)

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
		commentID = result.GetIDFromLocation()
		if commentID == "" {
			// Try data.id as fallback
			commentID = result.GetDataString("id")
		}
		if commentID == "" {
			t.Fatalf("expected comment ID in response (location: %s)", result.GetLocation())
		}

		h.Cleanup.AddComment(commentID, cardNumber)

		// Verify the body was actually saved by fetching the comment
		showResult := h.Run("comment", "show", commentID, "--card", cardStr)
		if showResult.ExitCode != harness.ExitSuccess {
			t.Fatalf("failed to show comment: %s", showResult.Stderr)
		}
		// Body is returned as an object with "plain_text" and "html" fields
		data := showResult.GetDataMap()
		if bodyObj, ok := data["body"].(map[string]any); ok {
			savedBody := bodyObj["plain_text"].(string)
			if savedBody != commentBody {
				t.Errorf("expected body %q, got %q", commentBody, savedBody)
			}
			// Verify HTML version is also returned
			savedHtml, ok := bodyObj["html"].(string)
			if !ok || savedHtml == "" {
				t.Error("expected body.html to be present")
			}
			// The HTML should contain our text
			if !strings.Contains(savedHtml, commentBody) {
				t.Errorf("expected body.html to contain %q, got %q", commentBody, savedHtml)
			}
		} else {
			t.Errorf("expected body object, got %T", data["body"])
		}
	})

	t.Run("create comment with body_file", func(t *testing.T) {
		wd, _ := os.Getwd()
		fixturePath := filepath.Join(wd, "..", "testdata", "fixtures", "test_document.txt")

		if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
			t.Skipf("test fixture not found at %s", fixturePath)
		}

		result := h.Run("comment", "create", "--card", cardStr, "--body_file", fixturePath)

		if result.ExitCode != harness.ExitSuccess {
			t.Fatalf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		// Create returns location - extract ID from it
		id := result.GetIDFromLocation()
		if id == "" {
			id = result.GetDataString("id")
		}
		if id != "" {
			h.Cleanup.AddComment(id, cardNumber)
		}

		if !result.Response.OK {
			t.Error("expected ok=true")
		}
	})

	t.Run("show comment", func(t *testing.T) {
		if commentID == "" {
			t.Skip("no comment ID from create test")
		}

		result := h.Run("comment", "show", commentID, "--card", cardStr)

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
		if id != commentID {
			t.Errorf("expected id %q, got %q", commentID, id)
		}
	})

	t.Run("update comment", func(t *testing.T) {
		if commentID == "" {
			t.Skip("no comment ID from create test")
		}

		newBody := commentBody + " updated"
		result := h.Run("comment", "update", commentID, "--card", cardStr, "--body", newBody)

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if result.Response == nil {
			t.Fatal("expected JSON response")
		}

		if !result.Response.OK {
			t.Error("expected ok=true")
		}

		// Verify the body was actually updated by fetching the comment
		showResult := h.Run("comment", "show", commentID, "--card", cardStr)
		if showResult.ExitCode != harness.ExitSuccess {
			t.Fatalf("failed to show comment: %s", showResult.Stderr)
		}
		// Body is returned as an object with "plain_text" and "html" fields
		data := showResult.GetDataMap()
		if bodyObj, ok := data["body"].(map[string]any); ok {
			savedBody := bodyObj["plain_text"].(string)
			if savedBody != newBody {
				t.Errorf("expected body %q after update, got %q", newBody, savedBody)
			}
			// Verify HTML version is also returned and updated
			savedHtml, ok := bodyObj["html"].(string)
			if !ok || savedHtml == "" {
				t.Error("expected body.html to be present")
			}
			// The HTML should contain our updated text
			if !strings.Contains(savedHtml, newBody) {
				t.Errorf("expected body.html to contain %q, got %q", newBody, savedHtml)
			}
		} else {
			t.Errorf("expected body object, got %T", data["body"])
		}
	})

	t.Run("delete comment", func(t *testing.T) {
		if commentID == "" {
			t.Skip("no comment ID from create test")
		}

		result := h.Run("comment", "delete", commentID, "--card", cardStr)

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
		if len(h.Cleanup.Comments) > 0 {
			h.Cleanup.Comments = h.Cleanup.Comments[:len(h.Cleanup.Comments)-1]
		}
	})
}

func TestCommentCreateMissingBody(t *testing.T) {
	h := harness.New(t)
	defer h.Cleanup.CleanupAll(h)

	boardID := createTestBoard(t, h)
	cardNumber := createTestCard(t, h, boardID)
	cardStr := strconv.Itoa(cardNumber)

	t.Run("fails without body or body_file", func(t *testing.T) {
		result := h.Run("comment", "create", "--card", cardStr)

		// Should fail - need either body or body_file
		if result.ExitCode == harness.ExitSuccess {
			// If it succeeded, it might have created an empty comment - clean up
			id := result.GetDataString("id")
			if id != "" {
				h.Cleanup.AddComment(id, cardNumber)
			}
			t.Error("expected failure without body")
		}
	})
}
