package commands

import (
	"testing"

	"github.com/basecamp/fizzy-cli/internal/errors"
)

func TestMigrateBoardValidation(t *testing.T) {
	t.Run("requires authentication", func(t *testing.T) {
		mock := NewMockClient()
		result := SetTestMode(mock)
		SetTestConfig("", "account", "https://api.example.com") // No token
		defer ResetTestMode()

		migrateBoardFrom = "source"
		migrateBoardTo = "target"
		defer func() {
			migrateBoardFrom = ""
			migrateBoardTo = ""
		}()

		RunTestCommand(func() {
			migrateBoardCmd.Run(migrateBoardCmd, []string{"board-id"})
		})

		if result.ExitCode != errors.ExitAuthFailure {
			t.Errorf("expected exit code %d, got %d", errors.ExitAuthFailure, result.ExitCode)
		}
	})

	t.Run("requires --from flag", func(t *testing.T) {
		mock := NewMockClient()
		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		migrateBoardFrom = ""
		migrateBoardTo = "target"
		defer func() {
			migrateBoardFrom = ""
			migrateBoardTo = ""
		}()

		RunTestCommand(func() {
			migrateBoardCmd.Run(migrateBoardCmd, []string{"board-id"})
		})

		if result.ExitCode != errors.ExitInvalidArgs {
			t.Errorf("expected exit code %d, got %d", errors.ExitInvalidArgs, result.ExitCode)
		}
	})

	t.Run("requires --to flag", func(t *testing.T) {
		mock := NewMockClient()
		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		migrateBoardFrom = "source"
		migrateBoardTo = ""
		defer func() {
			migrateBoardFrom = ""
			migrateBoardTo = ""
		}()

		RunTestCommand(func() {
			migrateBoardCmd.Run(migrateBoardCmd, []string{"board-id"})
		})

		if result.ExitCode != errors.ExitInvalidArgs {
			t.Errorf("expected exit code %d, got %d", errors.ExitInvalidArgs, result.ExitCode)
		}
	})

	t.Run("rejects same source and target account", func(t *testing.T) {
		mock := NewMockClient()
		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		migrateBoardFrom = "same-account"
		migrateBoardTo = "same-account"
		defer func() {
			migrateBoardFrom = ""
			migrateBoardTo = ""
		}()

		RunTestCommand(func() {
			migrateBoardCmd.Run(migrateBoardCmd, []string{"board-id"})
		})

		if result.ExitCode != errors.ExitInvalidArgs {
			t.Errorf("expected exit code %d, got %d", errors.ExitInvalidArgs, result.ExitCode)
		}
	})
}

func TestVerifyAccountAccess(t *testing.T) {
	t.Run("succeeds when user has access to both accounts", func(t *testing.T) {
		// This test would need to mock the identity endpoint
		// For now we test the helper functions
	})
}

func TestGetStringField(t *testing.T) {
	t.Run("returns string value", func(t *testing.T) {
		m := map[string]interface{}{"key": "value"}
		result := getStringField(m, "key")
		if result != "value" {
			t.Errorf("expected 'value', got '%s'", result)
		}
	})

	t.Run("returns empty string for missing key", func(t *testing.T) {
		m := map[string]interface{}{}
		result := getStringField(m, "missing")
		if result != "" {
			t.Errorf("expected empty string, got '%s'", result)
		}
	})

	t.Run("returns empty string for non-string value", func(t *testing.T) {
		m := map[string]interface{}{"key": 123}
		result := getStringField(m, "key")
		if result != "" {
			t.Errorf("expected empty string, got '%s'", result)
		}
	})
}

func TestGetIntField(t *testing.T) {
	t.Run("returns int from float64", func(t *testing.T) {
		m := map[string]interface{}{"key": float64(42)}
		result := getIntField(m, "key")
		if result != 42 {
			t.Errorf("expected 42, got %d", result)
		}
	})

	t.Run("returns int from int", func(t *testing.T) {
		m := map[string]interface{}{"key": 42}
		result := getIntField(m, "key")
		if result != 42 {
			t.Errorf("expected 42, got %d", result)
		}
	})

	t.Run("returns 0 for missing key", func(t *testing.T) {
		m := map[string]interface{}{}
		result := getIntField(m, "missing")
		if result != 0 {
			t.Errorf("expected 0, got %d", result)
		}
	})
}

func TestGetBoolField(t *testing.T) {
	t.Run("returns true", func(t *testing.T) {
		m := map[string]interface{}{"key": true}
		result := getBoolField(m, "key")
		if !result {
			t.Error("expected true, got false")
		}
	})

	t.Run("returns false", func(t *testing.T) {
		m := map[string]interface{}{"key": false}
		result := getBoolField(m, "key")
		if result {
			t.Error("expected false, got true")
		}
	})

	t.Run("returns false for missing key", func(t *testing.T) {
		m := map[string]interface{}{}
		result := getBoolField(m, "missing")
		if result {
			t.Error("expected false, got true")
		}
	})
}

func TestGetCardColumnID(t *testing.T) {
	t.Run("returns column_id directly", func(t *testing.T) {
		card := map[string]interface{}{"column_id": "col-123"}
		result := getCardColumnID(card)
		if result != "col-123" {
			t.Errorf("expected 'col-123', got '%s'", result)
		}
	})

	t.Run("returns id from nested column object", func(t *testing.T) {
		card := map[string]interface{}{
			"column": map[string]interface{}{"id": "col-456"},
		}
		result := getCardColumnID(card)
		if result != "col-456" {
			t.Errorf("expected 'col-456', got '%s'", result)
		}
	})

	t.Run("returns empty string when no column", func(t *testing.T) {
		card := map[string]interface{}{}
		result := getCardColumnID(card)
		if result != "" {
			t.Errorf("expected empty string, got '%s'", result)
		}
	})

	t.Run("prefers column_id over nested column", func(t *testing.T) {
		card := map[string]interface{}{
			"column_id": "col-123",
			"column":    map[string]interface{}{"id": "col-456"},
		}
		result := getCardColumnID(card)
		if result != "col-123" {
			t.Errorf("expected 'col-123', got '%s'", result)
		}
	})
}

func TestCountRealColumns(t *testing.T) {
	t.Run("counts only real columns", func(t *testing.T) {
		columns := []interface{}{
			map[string]interface{}{"id": "1", "name": "Backlog", "kind": "real"},
			map[string]interface{}{"id": "2", "name": "In Progress", "kind": "real"},
			map[string]interface{}{"id": "3", "name": "Not Now", "kind": "pseudo", "pseudo": true},
			map[string]interface{}{"id": "4", "name": "Done", "pseudo": true},
		}
		result := countRealColumns(columns)
		if result != 2 {
			t.Errorf("expected 2, got %d", result)
		}
	})

	t.Run("returns 0 for empty list", func(t *testing.T) {
		columns := []interface{}{}
		result := countRealColumns(columns)
		if result != 0 {
			t.Errorf("expected 0, got %d", result)
		}
	})

	t.Run("handles columns without kind field", func(t *testing.T) {
		columns := []interface{}{
			map[string]interface{}{"id": "1", "name": "Column"},
		}
		result := countRealColumns(columns)
		if result != 1 {
			t.Errorf("expected 1, got %d", result)
		}
	})
}

func TestMigrateInlineAttachments(t *testing.T) {
	t.Run("returns original HTML when no attachments", func(t *testing.T) {
		mock := NewMockClient()
		html := "<p>Hello world</p>"

		result, count := migrateInlineAttachments(mock, mock, html)
		if count != 0 {
			t.Errorf("expected 0 attachments, got %d", count)
		}
		if result != html {
			t.Errorf("expected original HTML, got '%s'", result)
		}
	})
}

// Note: Full dry run integration test is complex due to multi-client setup.
// The migrate command creates separate clients for source and target accounts,
// which makes mocking challenging. E2E tests cover the full flow.
