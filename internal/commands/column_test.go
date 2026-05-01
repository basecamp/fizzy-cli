package commands

import (
	"testing"

	"github.com/basecamp/fizzy-cli/internal/client"
	"github.com/basecamp/fizzy-cli/internal/errors"
)

func TestColumnList(t *testing.T) {
	t.Run("returns list of columns", func(t *testing.T) {
		mock := NewMockClient()
		mock.OnGet("/boards/123/columns.json", &client.APIResponse{
			StatusCode: 200,
			Data: []any{
				map[string]any{"id": "1", "name": "To Do", "color": map[string]any{"name": "Blue", "value": "var(--color-card-1)"}},
				map[string]any{"id": "2", "name": "In Progress", "color": map[string]any{"name": "Green", "value": "var(--color-card-2)"}},
			},
		})

		result := SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		columnListBoard = "123"
		err := columnListCmd.RunE(columnListCmd, []string{})
		columnListBoard = ""

		assertExitCode(t, err, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if mock.GetCalls[0].Path != "/boards/123/columns.json" {
			t.Errorf("expected path '/boards/123/columns.json', got '%s'", mock.GetCalls[0].Path)
		}

		arr, ok := result.Response.Data.([]any)
		if !ok {
			t.Fatalf("expected array response data, got %T", result.Response.Data)
		}
		if len(arr) != 5 {
			t.Fatalf("expected 5 columns (3 pseudo + 2 real), got %d", len(arr))
		}

		first := arr[0].(map[string]any)
		if first["id"] != "not-now" || first["name"] != "Not Now" {
			t.Errorf("expected first pseudo column Not Now, got %+v", first)
		}
		second := arr[1].(map[string]any)
		if second["id"] != "maybe" || second["name"] != "Maybe?" {
			t.Errorf("expected second pseudo column Maybe?, got %+v", second)
		}
		last := arr[len(arr)-1].(map[string]any)
		if last["id"] != "done" || last["name"] != "Done" {
			t.Errorf("expected last pseudo column Done, got %+v", last)
		}
	})

	t.Run("requires board flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		columnListBoard = ""
		err := columnListCmd.RunE(columnListCmd, []string{})
		assertExitCode(t, err, errors.ExitInvalidArgs)
	})

	t.Run("uses configured board when flag omitted", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       []any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		cfg.Board = "123"
		defer resetTest()

		columnListBoard = ""
		err := columnListCmd.RunE(columnListCmd, []string{})
		assertExitCode(t, err, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if mock.GetCalls[0].Path != "/boards/123/columns.json" {
			t.Errorf("expected path '/boards/123/columns.json', got '%s'", mock.GetCalls[0].Path)
		}
	})
}

func TestColumnShow(t *testing.T) {
	t.Run("shows column by ID", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetResponse = &client.APIResponse{
			StatusCode: 200,
			Data: map[string]any{
				"id":    "col-1",
				"name":  "In Progress",
				"color": map[string]any{"name": "Blue", "value": "var(--color-card-1)"},
			},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		columnShowBoard = "123"
		err := columnShowCmd.RunE(columnShowCmd, []string{"col-1"})
		columnShowBoard = ""

		assertExitCode(t, err, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if mock.GetCalls[0].Path != "/boards/123/columns/col-1" {
			t.Errorf("expected path '/boards/123/columns/col-1', got '%s'", mock.GetCalls[0].Path)
		}
	})

	t.Run("requires board flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		columnShowBoard = ""
		err := columnShowCmd.RunE(columnShowCmd, []string{"col-1"})
		assertExitCode(t, err, errors.ExitInvalidArgs)
	})

	t.Run("shows pseudo columns without board", func(t *testing.T) {
		mock := NewMockClient()
		result := SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		columnShowBoard = ""
		err := columnShowCmd.RunE(columnShowCmd, []string{"done"})
		assertExitCode(t, err, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, ok := result.Response.Data.(map[string]any)
		if !ok {
			t.Fatalf("expected map response data, got %T", result.Response.Data)
		}
		if data["id"] != "done" || data["name"] != "Done" {
			t.Errorf("expected pseudo Done column, got %+v", data)
		}
	})
}

func TestColumnCreate(t *testing.T) {
	t.Run("creates column with name", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{
			StatusCode: 201,
			Location:   "/columns/col-1",
			Data: map[string]any{
				"id":   "col-1",
				"name": "New Column",
			},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		columnCreateBoard = "123"
		columnCreateName = "New Column"
		err := columnCreateCmd.RunE(columnCreateCmd, []string{})
		columnCreateBoard = ""
		columnCreateName = ""

		assertExitCode(t, err, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if mock.PostCalls[0].Path != "/boards/123/columns.json" {
			t.Errorf("expected path '/boards/123/columns.json', got '%s'", mock.PostCalls[0].Path)
		}

		body := mock.PostCalls[0].Body.(map[string]any)
		if body["name"] != "New Column" {
			t.Errorf("expected name 'New Column', got '%v'", body["name"])
		}
	})

	t.Run("requires board flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		columnCreateBoard = ""
		columnCreateName = "Test"
		err := columnCreateCmd.RunE(columnCreateCmd, []string{})
		columnCreateName = ""

		assertExitCode(t, err, errors.ExitInvalidArgs)
	})

	t.Run("requires name flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		columnCreateBoard = "123"
		columnCreateName = ""
		err := columnCreateCmd.RunE(columnCreateCmd, []string{})
		columnCreateBoard = ""

		assertExitCode(t, err, errors.ExitInvalidArgs)
	})

	t.Run("includes optional color", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{
			StatusCode: 201,
			Data:       map[string]any{"id": "col-1", "name": "Test"},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		columnCreateBoard = "123"
		columnCreateName = "Test"
		columnCreateColor = "blue"
		err := columnCreateCmd.RunE(columnCreateCmd, []string{})
		columnCreateBoard = ""
		columnCreateName = ""
		columnCreateColor = ""

		assertExitCode(t, err, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		body := mock.PostCalls[0].Body.(map[string]any)
		if body["color"] != "blue" {
			t.Errorf("expected color 'blue', got '%v'", body["color"])
		}
	})
}

func TestColumnUpdate(t *testing.T) {
	t.Run("updates column name", func(t *testing.T) {
		mock := NewMockClient()
		mock.PatchResponse = &client.APIResponse{
			StatusCode: 200,
			Data: map[string]any{
				"id":   "col-1",
				"name": "Updated Column",
			},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		columnUpdateBoard = "123"
		columnUpdateName = "Updated Column"
		err := columnUpdateCmd.RunE(columnUpdateCmd, []string{"col-1"})
		columnUpdateBoard = ""
		columnUpdateName = ""

		assertExitCode(t, err, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if mock.PatchCalls[0].Path != "/boards/123/columns/col-1" {
			t.Errorf("expected path '/boards/123/columns/col-1', got '%s'", mock.PatchCalls[0].Path)
		}
	})

	t.Run("requires board flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		columnUpdateBoard = ""
		err := columnUpdateCmd.RunE(columnUpdateCmd, []string{"col-1"})
		assertExitCode(t, err, errors.ExitInvalidArgs)
	})
}

func TestColumnDelete(t *testing.T) {
	t.Run("deletes column", func(t *testing.T) {
		mock := NewMockClient()
		mock.DeleteResponse = &client.APIResponse{
			StatusCode: 204,
			Data:       map[string]any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		columnDeleteBoard = "123"
		err := columnDeleteCmd.RunE(columnDeleteCmd, []string{"col-1"})
		columnDeleteBoard = ""

		assertExitCode(t, err, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if mock.DeleteCalls[0].Path != "/boards/123/columns/col-1" {
			t.Errorf("expected path '/boards/123/columns/col-1', got '%s'", mock.DeleteCalls[0].Path)
		}
	})

	t.Run("requires board flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		columnDeleteBoard = ""
		err := columnDeleteCmd.RunE(columnDeleteCmd, []string{"col-1"})
		assertExitCode(t, err, errors.ExitInvalidArgs)
	})
}

func TestColumnMoveLeft(t *testing.T) {
	t.Run("moves column left", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := columnMoveLeftCmd.RunE(columnMoveLeftCmd, []string{"col-1"})
		assertExitCode(t, err, 0)

		if mock.PostCalls[0].Path != "/columns/col-1/left_position.json" {
			t.Errorf("expected path '/columns/col-1/left_position.json', got '%s'", mock.PostCalls[0].Path)
		}
	})

	t.Run("rejects pseudo columns", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := columnMoveLeftCmd.RunE(columnMoveLeftCmd, []string{"not-now"})
		assertExitCode(t, err, errors.ExitInvalidArgs)
	})
}

func TestColumnMoveRight(t *testing.T) {
	t.Run("moves column right", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := columnMoveRightCmd.RunE(columnMoveRightCmd, []string{"col-1"})
		assertExitCode(t, err, 0)

		if mock.PostCalls[0].Path != "/columns/col-1/right_position.json" {
			t.Errorf("expected path '/columns/col-1/right_position.json', got '%s'", mock.PostCalls[0].Path)
		}
	})

	t.Run("rejects pseudo columns", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := columnMoveRightCmd.RunE(columnMoveRightCmd, []string{"not-now"})
		assertExitCode(t, err, errors.ExitInvalidArgs)
	})
}
