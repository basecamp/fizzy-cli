package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/basecamp/fizzy-cli/e2e/harness"
)

func TestBoardList(t *testing.T) {
	h := harness.New(t)

	t.Run("returns list of boards", func(t *testing.T) {
		result := h.Run("board", "list")

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if result.Response == nil {
			t.Fatalf("expected JSON response, got nil\nstdout: %s", result.Stdout)
		}

		if !result.Response.OK {
			t.Error("expected ok=true")
		}

		// Data should be an array (may be empty)
		arr := result.GetDataArray()
		if arr == nil {
			t.Error("expected data to be an array")
		}
	})

	t.Run("supports pagination with --page", func(t *testing.T) {
		result := h.Run("board", "list", "--page", "1")

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d", harness.ExitSuccess, result.ExitCode)
		}

		if result.Response == nil {
			t.Fatal("expected JSON response")
		}

		// Pagination may or may not have results depending on data
		// Just verify the command works
		if !result.Response.OK {
			t.Error("expected ok=true")
		}
	})

	t.Run("supports --all flag for fetching all pages", func(t *testing.T) {
		result := h.Run("board", "list", "--all")

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d", harness.ExitSuccess, result.ExitCode)
		}

		if result.Response == nil {
			t.Fatal("expected JSON response")
		}

		// When using --all, pagination should show no next page
		if ctx := result.Response.Context; ctx != nil {
			if pagination, ok := ctx["pagination"].(map[string]interface{}); ok {
				if hasNext, _ := pagination["has_next"].(bool); hasNext {
					t.Error("with --all, expected has_next=false")
				}
			}
		}
	})
}

func TestBoardCRUD(t *testing.T) {
	h := harness.New(t)
	defer h.Cleanup.CleanupAll(h)

	var boardID string
	boardName := fmt.Sprintf("Test Board %d", time.Now().UnixNano())

	t.Run("create board with name", func(t *testing.T) {
		result := h.Run("board", "create", "--name", boardName)

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

		// Create returns location, not data - extract ID from location
		boardID = result.GetIDFromLocation()
		if boardID == "" {
			// Try data.id as fallback
			boardID = result.GetDataString("id")
		}
		if boardID == "" {
			t.Fatalf("expected board ID in response (location: %s)", result.GetLocation())
		}

		h.Cleanup.AddBoard(boardID)

		// Verify the name was actually saved by fetching the board
		showResult := h.Run("board", "show", boardID)
		if showResult.ExitCode != harness.ExitSuccess {
			t.Fatalf("failed to show board: %s", showResult.Stderr)
		}
		savedName := showResult.GetDataString("name")
		if savedName != boardName {
			t.Errorf("expected name %q, got %q", boardName, savedName)
		}
	})

	t.Run("show board by ID", func(t *testing.T) {
		if boardID == "" {
			t.Skip("no board ID from create test")
		}

		result := h.Run("board", "show", boardID)

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
		if id != boardID {
			t.Errorf("expected id %q, got %q", boardID, id)
		}
	})

	t.Run("update board name", func(t *testing.T) {
		if boardID == "" {
			t.Skip("no board ID from create test")
		}

		newName := boardName + " Updated"
		result := h.Run("board", "update", boardID, "--name", newName)

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if result.Response == nil {
			t.Fatal("expected JSON response")
		}

		if !result.Response.OK {
			t.Error("expected ok=true")
		}

		// Note: Update returns success but no data - verify via show
		showResult := h.Run("board", "show", boardID)
		if showResult.ExitCode == harness.ExitSuccess {
			name := showResult.GetDataString("name")
			if name != newName {
				t.Errorf("expected name %q after update, got %q", newName, name)
			}
		}
	})

	t.Run("delete board", func(t *testing.T) {
		if boardID == "" {
			t.Skip("no board ID from create test")
		}

		result := h.Run("board", "delete", boardID)

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
		h.Cleanup.Boards = nil
	})
}

func TestBoardCreateWithOptions(t *testing.T) {
	h := harness.New(t)
	defer h.Cleanup.CleanupAll(h)

	t.Run("create board with all_access=false", func(t *testing.T) {
		name := fmt.Sprintf("Private Board %d", time.Now().UnixNano())
		result := h.Run("board", "create", "--name", name, "--all_access", "false")

		if result.ExitCode != harness.ExitSuccess {
			t.Fatalf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		boardID := result.GetDataString("id")
		if boardID != "" {
			h.Cleanup.AddBoard(boardID)
		}

		if result.Response == nil || !result.Response.OK {
			t.Error("expected successful response")
		}
	})

	t.Run("create board with auto_postpone_period", func(t *testing.T) {
		name := fmt.Sprintf("Auto Postpone Board %d", time.Now().UnixNano())
		result := h.Run("board", "create", "--name", name, "--auto_postpone_period", "7")

		if result.ExitCode != harness.ExitSuccess {
			t.Fatalf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		boardID := result.GetDataString("id")
		if boardID != "" {
			h.Cleanup.AddBoard(boardID)
		}

		if result.Response == nil || !result.Response.OK {
			t.Error("expected successful response")
		}
	})
}

func TestBoardCreateMissingName(t *testing.T) {
	h := harness.New(t)

	t.Run("fails without required --name option", func(t *testing.T) {
		result := h.Run("board", "create")

		// Should fail with error exit code (1, 2, or 6)
		if result.ExitCode == harness.ExitSuccess {
			t.Error("expected non-zero exit code for missing required option")
		}
	})
}

func TestBoardShowNotFound(t *testing.T) {
	h := harness.New(t)

	t.Run("returns not found for non-existent board", func(t *testing.T) {
		result := h.Run("board", "show", "non-existent-board-id-12345")

		if result.ExitCode != harness.ExitNotFound {
			t.Errorf("expected exit code %d, got %d\nstdout: %s\nstderr: %s",
				harness.ExitNotFound, result.ExitCode, result.Stdout, result.Stderr)
		}

		if result.Response == nil {
			t.Fatal("expected JSON response")
		}

		if result.Response.OK {
			t.Error("expected ok=false")
		}

		if result.Response.Error == "" {
			t.Error("expected error in response")
		} else if result.Response.Code != "not_found" {
			t.Errorf("expected error code not_found, got %s", result.Response.Code)
		}
	})
}

func TestBoardDeleteNotFound(t *testing.T) {
	h := harness.New(t)

	t.Run("returns not found for non-existent board", func(t *testing.T) {
		result := h.Run("board", "delete", "non-existent-board-id-12345")

		if result.ExitCode != harness.ExitNotFound {
			t.Errorf("expected exit code %d, got %d", harness.ExitNotFound, result.ExitCode)
		}

		if result.Response == nil {
			t.Fatal("expected JSON response")
		}

		if result.Response.OK {
			t.Error("expected ok=false")
		}
	})
}
