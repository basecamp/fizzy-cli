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

		result := SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

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

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		boardListPage = 2
		boardListAll = false
		err := boardListCmd.RunE(boardListCmd, []string{})
		boardListPage = 0 // reset

		assertExitCode(t, err, 0)
	})

	t.Run("next page breadcrumb points to page 2 when page not specified", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       []any{map[string]any{"id": "1", "name": "Board 1"}},
			LinkNext:   "https://api.example.com/boards.json?page=2",
		}

		result := SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		boardListPage = 0
		boardListAll = false
		err := boardListCmd.RunE(boardListCmd, []string{})
		boardListPage = 0

		assertExitCode(t, err, 0)

		found := false
		for _, bc := range result.Response.Breadcrumbs {
			if bc.Action == "next" {
				found = true
				if bc.Cmd != "fizzy board list --page 2" {
					t.Errorf("expected next breadcrumb 'fizzy board list --page 2', got '%s'", bc.Cmd)
				}
			}
		}
		if !found {
			t.Error("expected 'next' breadcrumb but none found")
		}
	})

	t.Run("handles double-digit page numbers", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       []any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		boardListPage = 12
		boardListAll = false
		err := boardListCmd.RunE(boardListCmd, []string{})
		boardListPage = 0 // reset

		assertExitCode(t, err, 0)
		if mock.GetWithPaginationCalls[0].Path != "/boards.json?page=12" {
			t.Errorf("expected path '/boards.json?page=12', got '%s'", mock.GetWithPaginationCalls[0].Path)
		}
	})

	t.Run("passes page to GetAll", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       []any{map[string]any{"id": "1"}},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		boardListPage = 3
		boardListAll = true
		err := boardListCmd.RunE(boardListCmd, []string{})
		boardListPage = 0
		boardListAll = false

		assertExitCode(t, err, 0)
		if len(mock.GetWithPaginationCalls) == 0 {
			t.Fatal("expected at least one GET call")
		}
		gotPath := mock.GetWithPaginationCalls[0].Path
		if gotPath != "/boards.json?page=3" {
			t.Errorf("expected path '/boards.json?page=3', got '%s'", gotPath)
		}
	})

	t.Run("requires authentication", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("", "account", "https://api.example.com") // No token
		defer resetTest()

		err := boardListCmd.RunE(boardListCmd, []string{})
		assertExitCode(t, err, errors.ExitAuthFailure)
	})

	t.Run("requires account", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "", "https://api.example.com") // No account
		defer resetTest()

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

		result := SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

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

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := boardShowCmd.RunE(boardShowCmd, []string{"999"})
		assertExitCode(t, err, errors.ExitNotFound)
	})
}

func TestBoardShowBreadcrumbs(t *testing.T) {
	t.Run("shows publish breadcrumb for unpublished board", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetResponse = &client.APIResponse{
			StatusCode: 200,
			Data: map[string]any{
				"id":   "123",
				"name": "Unpublished Board",
			},
		}

		result := SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := boardShowCmd.RunE(boardShowCmd, []string{"123"})
		assertExitCode(t, err, 0)

		found := false
		for _, bc := range result.Response.Breadcrumbs {
			if bc.Action == "publish" {
				found = true
				if bc.Cmd != "fizzy board publish 123" {
					t.Errorf("expected cmd 'fizzy board publish 123', got '%s'", bc.Cmd)
				}
			}
			if bc.Action == "unpublish" {
				t.Error("did not expect unpublish breadcrumb for unpublished board")
			}
		}
		if !found {
			t.Error("expected 'publish' breadcrumb but none found")
		}
	})

	t.Run("shows unpublish breadcrumb for published board", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetResponse = &client.APIResponse{
			StatusCode: 200,
			Data: map[string]any{
				"id":         "123",
				"name":       "Published Board",
				"public_url": "https://app.fizzy.do/public/boards/test",
			},
		}

		result := SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := boardShowCmd.RunE(boardShowCmd, []string{"123"})
		assertExitCode(t, err, 0)

		found := false
		for _, bc := range result.Response.Breadcrumbs {
			if bc.Action == "unpublish" {
				found = true
				if bc.Cmd != "fizzy board unpublish 123" {
					t.Errorf("expected cmd 'fizzy board unpublish 123', got '%s'", bc.Cmd)
				}
			}
			if bc.Action == "publish" {
				t.Error("did not expect publish breadcrumb for published board")
			}
		}
		if !found {
			t.Error("expected 'unpublish' breadcrumb but none found")
		}
	})
}

func TestBoardCreate(t *testing.T) {
	t.Run("creates board with name", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{
			StatusCode: 201,
			Location:   "/boards/456",
			Data:       map[string]any{"id": "456"},
		}
		mock.OnGet("/boards/456", &client.APIResponse{
			StatusCode: 200,
			Data: map[string]any{
				"id":   "456",
				"name": "New Board",
			},
		})

		result := SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

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

		// Verify body contains name (flat — Rails wrap_parameters handles wrapping)
		body, ok := mock.PostCalls[0].Body.(map[string]any)
		if !ok {
			t.Fatal("expected map body")
		}
		if body["name"] != "New Board" {
			t.Errorf("expected name 'New Board', got '%v'", body["name"])
		}
	})

	t.Run("requires name flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		boardCreateName = ""
		err := boardCreateCmd.RunE(boardCreateCmd, []string{})
		assertExitCode(t, err, errors.ExitInvalidArgs)
	})

	t.Run("creates board with options", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{
			StatusCode: 201,
			Location:   "/boards/789",
			Data:       map[string]any{"id": "789", "name": "Private Board"},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		boardCreateName = "Private Board"
		boardCreateAllAccess = "false"
		boardCreateAutoPostponePeriodInDays = 7
		err := boardCreateCmd.RunE(boardCreateCmd, []string{})
		boardCreateName = ""
		boardCreateAllAccess = ""
		boardCreateAutoPostponePeriodInDays = 0

		assertExitCode(t, err, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		body := mock.PostCalls[0].Body.(map[string]any)
		// all_access=false is omitted from JSON (bool omitempty), which is equivalent to false
		if v, ok := body["all_access"]; ok && v != false {
			t.Errorf("expected all_access false or absent, got %v", body["all_access"])
		}
		if body["auto_postpone_period_in_days"] != float64(7) {
			t.Errorf("expected auto_postpone_period_in_days 7, got %v", body["auto_postpone_period_in_days"])
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

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

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
		if mock.PatchCalls[0].Path != "/boards/123" {
			t.Errorf("expected path '/boards/123', got '%s'", mock.PatchCalls[0].Path)
		}
	})

	t.Run("handles API error", func(t *testing.T) {
		mock := NewMockClient()
		mock.PatchError = errors.NewValidationError("Name is too long")

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

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

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := boardDeleteCmd.RunE(boardDeleteCmd, []string{"123"})
		assertExitCode(t, err, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(mock.DeleteCalls) != 1 {
			t.Errorf("expected 1 Delete call, got %d", len(mock.DeleteCalls))
		}
		if mock.DeleteCalls[0].Path != "/boards/123" {
			t.Errorf("expected path '/boards/123', got '%s'", mock.DeleteCalls[0].Path)
		}
	})

	t.Run("handles not found", func(t *testing.T) {
		mock := NewMockClient()
		mock.DeleteError = errors.NewNotFoundError("Board not found")

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := boardDeleteCmd.RunE(boardDeleteCmd, []string{"999"})
		assertExitCode(t, err, errors.ExitNotFound)
	})
}

func TestBoardPublish(t *testing.T) {
	t.Run("publishes board", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{
			StatusCode: 201,
			Data: map[string]any{
				"id":         "123",
				"name":       "Published Board",
				"public_url": "https://app.fizzy.do/public/boards/test",
			},
		}

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		err := boardPublishCmd.RunE(boardPublishCmd, []string{"123"})
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
		if mock.PostCalls[0].Path != "/boards/123/publication.json" {
			t.Errorf("expected path '/boards/123/publication.json', got '%s'", mock.PostCalls[0].Path)
		}
		if result.Response == nil || result.Response.Data == nil {
			t.Fatal("expected response data")
		}
		data, ok := result.Response.Data.(map[string]any)
		if !ok {
			t.Fatal("expected response data map")
		}
		if data["public_url"] != "https://app.fizzy.do/public/boards/test" {
			t.Errorf("expected public_url in response, got %v", data["public_url"])
		}
	})

	t.Run("handles API error", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostError = errors.NewForbiddenError("Only admins can publish boards")

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		err := boardPublishCmd.RunE(boardPublishCmd, []string{"123"})
		assertExitCode(t, err, errors.ExitForbidden)
	})
}

func TestBoardUnpublish(t *testing.T) {
	t.Run("unpublishes board", func(t *testing.T) {
		mock := NewMockClient()
		mock.DeleteResponse = &client.APIResponse{
			StatusCode: 204,
			Data:       map[string]any{},
		}

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		err := boardUnpublishCmd.RunE(boardUnpublishCmd, []string{"123"})
		assertExitCode(t, err, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Response.OK {
			t.Error("expected success response")
		}
		if len(mock.DeleteCalls) != 1 {
			t.Errorf("expected 1 Delete call, got %d", len(mock.DeleteCalls))
		}
		if mock.DeleteCalls[0].Path != "/boards/123/publication.json" {
			t.Errorf("expected path '/boards/123/publication.json', got '%s'", mock.DeleteCalls[0].Path)
		}
		if result.Response == nil || result.Response.Data == nil {
			t.Fatal("expected response data")
		}
		data, ok := result.Response.Data.(map[string]any)
		if !ok {
			t.Fatal("expected response data map")
		}
		if data["unpublished"] != true {
			t.Errorf("expected unpublished=true, got %v", data["unpublished"])
		}
	})

	t.Run("handles not found", func(t *testing.T) {
		mock := NewMockClient()
		mock.DeleteError = errors.NewNotFoundError("Board not found")

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		err := boardUnpublishCmd.RunE(boardUnpublishCmd, []string{"999"})
		assertExitCode(t, err, errors.ExitNotFound)
	})
}

func TestBoardCreateRejectsInvalidAutoPostponePeriod(t *testing.T) {
	mock := NewMockClient()
	SetTestModeWithSDK(mock)
	SetTestConfig("token", "account", "https://api.example.com")
	defer resetTest()

	boardCreateName = "Test"
	boardCreateAutoPostponePeriodInDays = 15
	err := boardCreateCmd.RunE(boardCreateCmd, []string{})
	boardCreateName = ""
	boardCreateAutoPostponePeriodInDays = 0

	assertExitCode(t, err, errors.ExitInvalidArgs)
}

func TestBoardEntropy(t *testing.T) {
	t.Run("updates board auto-postpone period", func(t *testing.T) {
		mock := NewMockClient()
		mock.PutResponse = &client.APIResponse{
			StatusCode: 200,
			Data: map[string]any{
				"id":                           "board-1",
				"name":                         "Test Board",
				"auto_postpone_period_in_days": float64(90),
			},
		}

		result := SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		boardEntropyAutoPostponePeriodInDays = 90
		err := boardEntropyCmd.RunE(boardEntropyCmd, []string{"board-1"})
		boardEntropyAutoPostponePeriodInDays = 0

		assertExitCode(t, err, 0)

		if !result.Response.OK {
			t.Error("expected success response")
		}
		if len(mock.PutCalls) != 1 {
			t.Errorf("expected 1 Put call, got %d", len(mock.PutCalls))
		}
		if mock.PutCalls[0].Path != "/boards/board-1/entropy.json" {
			t.Errorf("expected path '/boards/board-1/entropy.json', got '%s'", mock.PutCalls[0].Path)
		}
		body, ok := mock.PutCalls[0].Body.(map[string]any)
		if !ok {
			t.Fatal("expected map body")
		}
		if body["auto_postpone_period_in_days"] != float64(90) {
			t.Errorf("expected auto_postpone_period_in_days 90, got %v", body["auto_postpone_period_in_days"])
		}
	})

	t.Run("requires auto_postpone_period_in_days flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		boardEntropyAutoPostponePeriodInDays = 0
		err := boardEntropyCmd.RunE(boardEntropyCmd, []string{"board-1"})
		assertExitCode(t, err, errors.ExitInvalidArgs)
	})

	t.Run("rejects invalid period", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		boardEntropyAutoPostponePeriodInDays = 45
		err := boardEntropyCmd.RunE(boardEntropyCmd, []string{"board-1"})
		boardEntropyAutoPostponePeriodInDays = 0

		assertExitCode(t, err, errors.ExitInvalidArgs)
	})

	t.Run("handles not found", func(t *testing.T) {
		mock := NewMockClient()
		mock.PutError = errors.NewNotFoundError("Board not found")

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		boardEntropyAutoPostponePeriodInDays = 30
		err := boardEntropyCmd.RunE(boardEntropyCmd, []string{"nonexistent"})
		boardEntropyAutoPostponePeriodInDays = 0

		assertExitCode(t, err, errors.ExitNotFound)
	})
}

func TestBoardClosed(t *testing.T) {
	t.Run("lists closed cards", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data: []any{
				map[string]any{"id": "1", "number": float64(10), "title": "Closed Card"},
			},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		boardClosedBoard = "123"
		err := boardClosedCmd.RunE(boardClosedCmd, []string{})
		boardClosedBoard = ""
		boardClosedPage = 0
		boardClosedAll = false

		assertExitCode(t, err, 0)

		if len(mock.GetWithPaginationCalls) != 1 {
			t.Errorf("expected 1 GetWithPagination call, got %d", len(mock.GetWithPaginationCalls))
		}
		if mock.GetWithPaginationCalls[0].Path != "/boards/123/columns/closed.json" {
			t.Errorf("expected path '/boards/123/columns/closed.json', got '%s'", mock.GetWithPaginationCalls[0].Path)
		}
	})

	t.Run("requires board flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		boardClosedBoard = ""
		err := boardClosedCmd.RunE(boardClosedCmd, []string{})
		boardClosedBoard = ""
		boardClosedPage = 0
		boardClosedAll = false

		assertExitCode(t, err, errors.ExitInvalidArgs)
	})
}

func TestBoardPostponed(t *testing.T) {
	t.Run("lists postponed cards", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data: []any{
				map[string]any{"id": "1", "number": float64(20), "title": "Postponed Card"},
			},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		boardPostponedBoard = "123"
		err := boardPostponedCmd.RunE(boardPostponedCmd, []string{})
		boardPostponedBoard = ""
		boardPostponedPage = 0
		boardPostponedAll = false

		assertExitCode(t, err, 0)

		if len(mock.GetWithPaginationCalls) != 1 {
			t.Errorf("expected 1 GetWithPagination call, got %d", len(mock.GetWithPaginationCalls))
		}
		if mock.GetWithPaginationCalls[0].Path != "/boards/123/columns/not_now.json" {
			t.Errorf("expected path '/boards/123/columns/not_now.json', got '%s'", mock.GetWithPaginationCalls[0].Path)
		}
	})

	t.Run("requires board flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		boardPostponedBoard = ""
		err := boardPostponedCmd.RunE(boardPostponedCmd, []string{})
		boardPostponedBoard = ""
		boardPostponedPage = 0
		boardPostponedAll = false

		assertExitCode(t, err, errors.ExitInvalidArgs)
	})
}

func TestBoardStream(t *testing.T) {
	t.Run("lists stream cards", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data: []any{
				map[string]any{"id": "1", "number": float64(30), "title": "Stream Card"},
			},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		boardStreamBoard = "123"
		err := boardStreamCmd.RunE(boardStreamCmd, []string{})
		boardStreamBoard = ""
		boardStreamPage = 0
		boardStreamAll = false

		assertExitCode(t, err, 0)

		if len(mock.GetWithPaginationCalls) != 1 {
			t.Errorf("expected 1 GetWithPagination call, got %d", len(mock.GetWithPaginationCalls))
		}
		if mock.GetWithPaginationCalls[0].Path != "/boards/123/columns/stream.json" {
			t.Errorf("expected path '/boards/123/columns/stream.json', got '%s'", mock.GetWithPaginationCalls[0].Path)
		}
	})

	t.Run("requires board flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		boardStreamBoard = ""
		err := boardStreamCmd.RunE(boardStreamCmd, []string{})
		boardStreamBoard = ""
		boardStreamPage = 0
		boardStreamAll = false

		assertExitCode(t, err, errors.ExitInvalidArgs)
	})
}

func TestBoardInvolvement(t *testing.T) {
	t.Run("updates board involvement", func(t *testing.T) {
		mock := NewMockClient()
		mock.PatchResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       map[string]any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		boardInvolvementInvolvement = "involved"
		err := boardInvolvementCmd.RunE(boardInvolvementCmd, []string{"123"})
		boardInvolvementInvolvement = ""

		assertExitCode(t, err, 0)

		if len(mock.PatchCalls) != 1 {
			t.Errorf("expected 1 Patch call, got %d", len(mock.PatchCalls))
		}
		if mock.PatchCalls[0].Path != "/boards/123/involvement.json" {
			t.Errorf("expected path '/boards/123/involvement.json', got '%s'", mock.PatchCalls[0].Path)
		}
		body, ok := mock.PatchCalls[0].Body.(map[string]any)
		if !ok {
			t.Fatal("expected map body")
		}
		if body["involvement"] != "involved" {
			t.Errorf("expected involvement 'involved', got '%v'", body["involvement"])
		}
	})

	t.Run("requires involvement flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		boardInvolvementInvolvement = ""
		err := boardInvolvementCmd.RunE(boardInvolvementCmd, []string{"123"})
		boardInvolvementInvolvement = ""

		assertExitCode(t, err, errors.ExitInvalidArgs)
	})
}
