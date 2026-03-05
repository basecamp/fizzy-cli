package commands

import (
	"testing"

	"github.com/basecamp/fizzy-cli/internal/client"
	"github.com/basecamp/fizzy-cli/internal/errors"
)

func TestColumnList(t *testing.T) {
	t.Run("returns list of columns", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetResponse = &client.APIResponse{
			StatusCode: 200,
			Data: []any{
				map[string]any{"id": "1", "name": "To Do"},
				map[string]any{"id": "2", "name": "In Progress"},
			},
		}

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		columnListBoard = "123"
		RunTestCommand(func() {
			columnListCmd.Run(columnListCmd, []string{})
		})
		columnListBoard = ""

		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", result.ExitCode)
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
		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		columnListBoard = ""
		RunTestCommand(func() {
			columnListCmd.Run(columnListCmd, []string{})
		})

		if result.ExitCode != errors.ExitInvalidArgs {
			t.Errorf("expected exit code %d, got %d", errors.ExitInvalidArgs, result.ExitCode)
		}
	})

	t.Run("uses configured board when flag omitted", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       []any{},
		}

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		cfg.Board = "123"
		defer ResetTestMode()

		columnListBoard = ""
		RunTestCommand(func() {
			columnListCmd.Run(columnListCmd, []string{})
		})

		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", result.ExitCode)
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
				"id":   "col-1",
				"name": "In Progress",
			},
		}

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		columnShowBoard = "123"
		RunTestCommand(func() {
			columnShowCmd.Run(columnShowCmd, []string{"col-1"})
		})
		columnShowBoard = ""

		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", result.ExitCode)
		}
		if mock.GetCalls[0].Path != "/boards/123/columns/col-1.json" {
			t.Errorf("expected path '/boards/123/columns/col-1.json', got '%s'", mock.GetCalls[0].Path)
		}
	})

	t.Run("requires board flag", func(t *testing.T) {
		mock := NewMockClient()
		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		columnShowBoard = ""
		RunTestCommand(func() {
			columnShowCmd.Run(columnShowCmd, []string{"col-1"})
		})

		if result.ExitCode != errors.ExitInvalidArgs {
			t.Errorf("expected exit code %d, got %d", errors.ExitInvalidArgs, result.ExitCode)
		}
	})

	t.Run("shows pseudo columns without board", func(t *testing.T) {
		mock := NewMockClient()
		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		columnShowBoard = ""
		RunTestCommand(func() {
			columnShowCmd.Run(columnShowCmd, []string{"done"})
		})

		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", result.ExitCode)
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
			Location:   "https://api.example.com/columns/col-1",
		}
		mock.FollowLocationResponse = &client.APIResponse{
			StatusCode: 200,
			Data: map[string]any{
				"id":   "col-1",
				"name": "New Column",
			},
		}

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		columnCreateBoard = "123"
		columnCreateName = "New Column"
		RunTestCommand(func() {
			columnCreateCmd.Run(columnCreateCmd, []string{})
		})
		columnCreateBoard = ""
		columnCreateName = ""

		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", result.ExitCode)
		}
		if mock.PostCalls[0].Path != "/boards/123/columns.json" {
			t.Errorf("expected path '/boards/123/columns.json', got '%s'", mock.PostCalls[0].Path)
		}

		body := mock.PostCalls[0].Body.(map[string]any)
		columnParams := body["column"].(map[string]any)
		if columnParams["name"] != "New Column" {
			t.Errorf("expected name 'New Column', got '%v'", columnParams["name"])
		}
	})

	t.Run("requires board flag", func(t *testing.T) {
		mock := NewMockClient()
		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		columnCreateBoard = ""
		columnCreateName = "Test"
		RunTestCommand(func() {
			columnCreateCmd.Run(columnCreateCmd, []string{})
		})
		columnCreateName = ""

		if result.ExitCode != errors.ExitInvalidArgs {
			t.Errorf("expected exit code %d, got %d", errors.ExitInvalidArgs, result.ExitCode)
		}
	})

	t.Run("requires name flag", func(t *testing.T) {
		mock := NewMockClient()
		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		columnCreateBoard = "123"
		columnCreateName = ""
		RunTestCommand(func() {
			columnCreateCmd.Run(columnCreateCmd, []string{})
		})
		columnCreateBoard = ""

		if result.ExitCode != errors.ExitInvalidArgs {
			t.Errorf("expected exit code %d, got %d", errors.ExitInvalidArgs, result.ExitCode)
		}
	})

	t.Run("includes optional color", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{
			StatusCode: 201,
			Location:   "https://api.example.com/columns/col-1",
		}
		mock.FollowLocationResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       map[string]any{},
		}

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		columnCreateBoard = "123"
		columnCreateName = "Test"
		columnCreateColor = "blue"
		RunTestCommand(func() {
			columnCreateCmd.Run(columnCreateCmd, []string{})
		})
		columnCreateBoard = ""
		columnCreateName = ""
		columnCreateColor = ""

		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", result.ExitCode)
		}

		body := mock.PostCalls[0].Body.(map[string]any)
		columnParams := body["column"].(map[string]any)
		if columnParams["color"] != "blue" {
			t.Errorf("expected color 'blue', got '%v'", columnParams["color"])
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

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		columnUpdateBoard = "123"
		columnUpdateName = "Updated Column"
		RunTestCommand(func() {
			columnUpdateCmd.Run(columnUpdateCmd, []string{"col-1"})
		})
		columnUpdateBoard = ""
		columnUpdateName = ""

		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", result.ExitCode)
		}
		if mock.PatchCalls[0].Path != "/boards/123/columns/col-1.json" {
			t.Errorf("expected path '/boards/123/columns/col-1.json', got '%s'", mock.PatchCalls[0].Path)
		}
	})

	t.Run("requires board flag", func(t *testing.T) {
		mock := NewMockClient()
		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		columnUpdateBoard = ""
		RunTestCommand(func() {
			columnUpdateCmd.Run(columnUpdateCmd, []string{"col-1"})
		})

		if result.ExitCode != errors.ExitInvalidArgs {
			t.Errorf("expected exit code %d, got %d", errors.ExitInvalidArgs, result.ExitCode)
		}
	})
}

func TestColumnDelete(t *testing.T) {
	t.Run("deletes column", func(t *testing.T) {
		mock := NewMockClient()
		mock.DeleteResponse = &client.APIResponse{
			StatusCode: 204,
			Data:       map[string]any{},
		}

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		columnDeleteBoard = "123"
		RunTestCommand(func() {
			columnDeleteCmd.Run(columnDeleteCmd, []string{"col-1"})
		})
		columnDeleteBoard = ""

		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", result.ExitCode)
		}
		if mock.DeleteCalls[0].Path != "/boards/123/columns/col-1.json" {
			t.Errorf("expected path '/boards/123/columns/col-1.json', got '%s'", mock.DeleteCalls[0].Path)
		}
	})

	t.Run("requires board flag", func(t *testing.T) {
		mock := NewMockClient()
		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		columnDeleteBoard = ""
		RunTestCommand(func() {
			columnDeleteCmd.Run(columnDeleteCmd, []string{"col-1"})
		})

		if result.ExitCode != errors.ExitInvalidArgs {
			t.Errorf("expected exit code %d, got %d", errors.ExitInvalidArgs, result.ExitCode)
		}
	})
}
