package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/basecamp/fizzy-cli/e2e/harness"
)

func TestColumnList(t *testing.T) {
	h := harness.New(t)
	defer h.Cleanup.CleanupAll(h)

	boardID := createTestBoard(t, h)

	t.Run("returns list of columns for board", func(t *testing.T) {
		result := h.Run("column", "list", "--board", boardID)

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

	t.Run("fails without --board option when no default board configured", func(t *testing.T) {
		// Use a temp HOME so the global config (which may have a default board) is not found
		tmpHome := t.TempDir()
		result := h.RunWithEnv(map[string]string{"HOME": tmpHome}, "column", "list")

		if result.ExitCode == harness.ExitSuccess {
			t.Error("expected non-zero exit code for missing required option")
		}
	})
}

func TestColumnCRUD(t *testing.T) {
	h := harness.New(t)
	defer h.Cleanup.CleanupAll(h)

	boardID := createTestBoard(t, h)
	var columnID string
	columnName := fmt.Sprintf("Test Column %d", time.Now().UnixNano())

	t.Run("create column", func(t *testing.T) {
		result := h.Run("column", "create", "--board", boardID, "--name", columnName)

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
		columnID = result.GetIDFromLocation()
		if columnID == "" {
			// Try data.id as fallback
			columnID = result.GetDataString("id")
		}
		if columnID == "" {
			t.Fatalf("expected column ID in response (location: %s)", result.GetLocation())
		}

		h.Cleanup.AddColumn(columnID, boardID)

		// Note: Create returns location, not data with name
		// Verify the column exists via show command
		showResult := h.Run("column", "show", columnID, "--board", boardID)
		if showResult.ExitCode == harness.ExitSuccess {
			name := showResult.GetDataString("name")
			if name != columnName {
				t.Errorf("expected name %q, got %q", columnName, name)
			}
		}
	})

	t.Run("create column with color", func(t *testing.T) {
		name := fmt.Sprintf("Colored Column %d", time.Now().UnixNano())
		color := "var(--color-card-4)"
		result := h.Run("column", "create", "--board", boardID, "--name", name, "--color", color)

		if result.ExitCode != harness.ExitSuccess {
			t.Fatalf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		// Create returns location - extract ID from it
		id := result.GetIDFromLocation()
		if id == "" {
			id = result.GetDataString("id")
		}
		if id != "" {
			h.Cleanup.AddColumn(id, boardID)
		}

		if !result.Response.OK {
			t.Error("expected ok=true")
		}

		// Verify the name and color were actually saved
		if id != "" {
			showResult := h.Run("column", "show", id, "--board", boardID)
			if showResult.ExitCode != harness.ExitSuccess {
				t.Fatalf("failed to show column: %s", showResult.Stderr)
			}
			savedName := showResult.GetDataString("name")
			if savedName != name {
				t.Errorf("expected name %q, got %q", name, savedName)
			}
			// Color is returned as an object with "value" field
			data := showResult.GetDataMap()
			if colorObj, ok := data["color"].(map[string]any); ok {
				savedColorValue := colorObj["value"].(string)
				if savedColorValue != color {
					t.Errorf("expected color value %q, got %q", color, savedColorValue)
				}
			} else {
				t.Errorf("expected color object, got %T", data["color"])
			}
		}
	})

	t.Run("show column", func(t *testing.T) {
		if columnID == "" {
			t.Skip("no column ID from create test")
		}

		result := h.Run("column", "show", columnID, "--board", boardID)

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
		if id != columnID {
			t.Errorf("expected id %q, got %q", columnID, id)
		}
	})

	t.Run("update column", func(t *testing.T) {
		if columnID == "" {
			t.Skip("no column ID from create test")
		}

		newName := columnName + " Updated"
		result := h.Run("column", "update", columnID, "--board", boardID, "--name", newName)

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if result.Response == nil {
			t.Fatal("expected JSON response")
		}

		if !result.Response.OK {
			t.Error("expected ok=true")
		}

		// Note: Update may return success without data - verify via show
		showResult := h.Run("column", "show", columnID, "--board", boardID)
		if showResult.ExitCode == harness.ExitSuccess {
			name := showResult.GetDataString("name")
			if name != newName {
				t.Errorf("expected name %q, got %q", newName, name)
			}
		}
	})

	t.Run("delete column", func(t *testing.T) {
		if columnID == "" {
			t.Skip("no column ID from create test")
		}

		result := h.Run("column", "delete", columnID, "--board", boardID)

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

		// Remove the first column from cleanup since we deleted it
		if len(h.Cleanup.Columns) > 0 {
			h.Cleanup.Columns = h.Cleanup.Columns[1:]
		}
	})
}

func TestColumnShowNotFound(t *testing.T) {
	h := harness.New(t)
	defer h.Cleanup.CleanupAll(h)

	boardID := createTestBoard(t, h)

	t.Run("returns not found for non-existent column", func(t *testing.T) {
		result := h.Run("column", "show", "non-existent-column-id", "--board", boardID)

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
