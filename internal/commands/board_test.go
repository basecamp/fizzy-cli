package commands

import (
	"testing"

	"github.com/basecamp/fizzy-cli/internal/client"
	"github.com/basecamp/fizzy-cli/internal/errors"
)

func TestBoardList(t *testing.T) {
	t.Run("returns list of boards", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data: []any{
				map[string]any{"id": "1", "name": "Board 1"},
				map[string]any{"id": "2", "name": "Board 2"},
			},
		}

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		err := boardListCmd.RunE(boardListCmd, []string{})
		assertExitCode(t, err, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Response.OK {
			t.Error("expected success response")
		}
		if len(mock.GetWithPaginationCalls) != 1 {
			t.Errorf("expected 1 GetWithPagination call, got %d", len(mock.GetWithPaginationCalls))
		}
		if mock.GetWithPaginationCalls[0].Path != "/boards.json" {
			t.Errorf("expected path '/boards.json', got '%s'", mock.GetWithPaginationCalls[0].Path)
		}
	})

	t.Run("handles pagination", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       []any{},
			LinkNext:   "https://api.example.com/boards.json?page=2",
		}

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		boardListPage = 2
		boardListAll = false
		err := boardListCmd.RunE(boardListCmd, []string{})
		boardListPage = 0 // reset

		assertExitCode(t, err, 0)
	})

	t.Run("handles double-digit page numbers", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       []any{},
		}

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		boardListPage = 12
		boardListAll = false
		err := boardListCmd.RunE(boardListCmd, []string{})
		boardListPage = 0 // reset

		assertExitCode(t, err, 0)
		if mock.GetWithPaginationCalls[0].Path != "/boards.json?page=12" {
			t.Errorf("expected path '/boards.json?page=12', got '%s'", mock.GetWithPaginationCalls[0].Path)
		}
	})

	t.Run("requires authentication", func(t *testing.T) {
		mock := NewMockClient()
		SetTestMode(mock)
		SetTestConfig("", "account", "https://api.example.com") // No token
		defer ResetTestMode()

		err := boardListCmd.RunE(boardListCmd, []string{})
		assertExitCode(t, err, errors.ExitAuthFailure)
	})

	t.Run("requires account", func(t *testing.T) {
		mock := NewMockClient()
		SetTestMode(mock)
		SetTestConfig("token", "", "https://api.example.com") // No account
		defer ResetTestMode()

		err := boardListCmd.RunE(boardListCmd, []string{})
		assertExitCode(t, err, errors.ExitInvalidArgs)
	})
}

func TestBoardShow(t *testing.T) {
	t.Run("shows board by ID", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetResponse = &client.APIResponse{
			StatusCode: 200,
			Data: map[string]any{
				"id":   "123",
				"name": "Test Board",
			},
		}

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		err := boardShowCmd.RunE(boardShowCmd, []string{"123"})
		assertExitCode(t, err, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Response.OK {
			t.Error("expected success response")
		}
		if len(mock.GetCalls) != 1 {
			t.Errorf("expected 1 Get call, got %d", len(mock.GetCalls))
		}
		if mock.GetCalls[0].Path != "/boards/123.json" {
			t.Errorf("expected path '/boards/123.json', got '%s'", mock.GetCalls[0].Path)
		}
	})

	t.Run("handles not found", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetError = errors.NewNotFoundError("Board not found")

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		err := boardShowCmd.RunE(boardShowCmd, []string{"999"})
		assertExitCode(t, err, errors.ExitNotFound)
	})
}

func TestBoardCreate(t *testing.T) {
	t.Run("creates board with name", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{
			StatusCode: 201,
			Location:   "https://api.example.com/boards/456",
			Data:       map[string]any{"id": "456"},
		}
		mock.FollowLocationResponse = &client.APIResponse{
			StatusCode: 200,
			Data: map[string]any{
				"id":   "456",
				"name": "New Board",
			},
		}

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		boardCreateName = "New Board"
		err := boardCreateCmd.RunE(boardCreateCmd, []string{})
		boardCreateName = "" // reset

		assertExitCode(t, err, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Response.OK {
			t.Error("expected success response")
		}
		if len(mock.PostCalls) != 1 {
			t.Errorf("expected 1 Post call, got %d", len(mock.PostCalls))
		}
		if mock.PostCalls[0].Path != "/boards.json" {
			t.Errorf("expected path '/boards.json', got '%s'", mock.PostCalls[0].Path)
		}

		// Verify body contains board wrapper with name
		body, ok := mock.PostCalls[0].Body.(map[string]any)
		if !ok {
			t.Fatal("expected map body")
		}
		boardParams, ok := body["board"].(map[string]any)
		if !ok {
			t.Fatal("expected board wrapper in body")
		}
		if boardParams["name"] != "New Board" {
			t.Errorf("expected name 'New Board', got '%v'", boardParams["name"])
		}
	})

	t.Run("requires name flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		boardCreateName = ""
		err := boardCreateCmd.RunE(boardCreateCmd, []string{})
		assertExitCode(t, err, errors.ExitInvalidArgs)
	})

	t.Run("creates board with options", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{
			StatusCode: 201,
			Location:   "https://api.example.com/boards/789",
		}
		mock.FollowLocationResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       map[string]any{"id": "789"},
		}

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		boardCreateName = "Private Board"
		boardCreateAllAccess = "false"
		boardCreateAutoPostponePeriod = 7
		err := boardCreateCmd.RunE(boardCreateCmd, []string{})
		boardCreateName = ""
		boardCreateAllAccess = ""
		boardCreateAutoPostponePeriod = 0

		assertExitCode(t, err, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		body := mock.PostCalls[0].Body.(map[string]any)
		boardParams := body["board"].(map[string]any)
		if boardParams["all_access"] != false {
			t.Errorf("expected all_access false, got %v", boardParams["all_access"])
		}
		if boardParams["auto_postpone_period"] != 7 {
			t.Errorf("expected auto_postpone_period 7, got %v", boardParams["auto_postpone_period"])
		}
	})
}

func TestBoardUpdate(t *testing.T) {
	t.Run("updates board name", func(t *testing.T) {
		mock := NewMockClient()
		mock.PatchResponse = &client.APIResponse{
			StatusCode: 200,
			Data: map[string]any{
				"id":   "123",
				"name": "Updated Name",
			},
		}

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		boardUpdateName = "Updated Name"
		err := boardUpdateCmd.RunE(boardUpdateCmd, []string{"123"})
		boardUpdateName = ""

		assertExitCode(t, err, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(mock.PatchCalls) != 1 {
			t.Errorf("expected 1 Patch call, got %d", len(mock.PatchCalls))
		}
		if mock.PatchCalls[0].Path != "/boards/123.json" {
			t.Errorf("expected path '/boards/123.json', got '%s'", mock.PatchCalls[0].Path)
		}
	})

	t.Run("handles API error", func(t *testing.T) {
		mock := NewMockClient()
		mock.PatchError = errors.NewValidationError("Name is too long")

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		boardUpdateName = "Updated"
		err := boardUpdateCmd.RunE(boardUpdateCmd, []string{"123"})
		boardUpdateName = ""

		assertExitCode(t, err, errors.ExitValidation)
	})
}

func TestBoardDelete(t *testing.T) {
	t.Run("deletes board", func(t *testing.T) {
		mock := NewMockClient()
		mock.DeleteResponse = &client.APIResponse{
			StatusCode: 204,
			Data:       map[string]any{},
		}

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		err := boardDeleteCmd.RunE(boardDeleteCmd, []string{"123"})
		assertExitCode(t, err, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(mock.DeleteCalls) != 1 {
			t.Errorf("expected 1 Delete call, got %d", len(mock.DeleteCalls))
		}
		if mock.DeleteCalls[0].Path != "/boards/123.json" {
			t.Errorf("expected path '/boards/123.json', got '%s'", mock.DeleteCalls[0].Path)
		}
	})

	t.Run("handles not found", func(t *testing.T) {
		mock := NewMockClient()
		mock.DeleteError = errors.NewNotFoundError("Board not found")

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		err := boardDeleteCmd.RunE(boardDeleteCmd, []string{"999"})
		assertExitCode(t, err, errors.ExitNotFound)
	})
}
