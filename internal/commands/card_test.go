package commands

import (
	"strings"
	"testing"

	"github.com/basecamp/fizzy-cli/internal/client"
	"github.com/basecamp/fizzy-cli/internal/errors"
)

func TestCardList(t *testing.T) {
	t.Run("returns list of cards", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data: []any{
				map[string]any{"id": "1", "title": "Card 1"},
				map[string]any{"id": "2", "title": "Card 2"},
			},
		}

		result := SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := cardListCmd.RunE(cardListCmd, []string{})
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
		if mock.GetWithPaginationCalls[0].Path != "/cards.json" {
			t.Errorf("expected path '/cards.json', got '%s'", mock.GetWithPaginationCalls[0].Path)
		}
	})

	t.Run("applies filters", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       []any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		cardListBoard = "123"
		cardListIndexedBy = "closed"
		err := cardListCmd.RunE(cardListCmd, []string{})
		cardListBoard = ""
		cardListIndexedBy = ""

		assertExitCode(t, err, 0)
		// Check that path contains filters
		path := mock.GetWithPaginationCalls[0].Path
		if path != "/cards.json?board_ids[]=123&indexed_by=closed" {
			t.Errorf("expected path with filters, got '%s'", path)
		}
	})

	t.Run("filters by pseudo column (not-now)", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       []any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		cfg.Board = "123"
		defer resetTest()

		cardListColumn = "not-now"
		err := cardListCmd.RunE(cardListCmd, []string{})
		cardListColumn = ""

		assertExitCode(t, err, 0)

		if mock.GetWithPaginationCalls[0].Path != "/cards.json?board_ids[]=123&indexed_by=not_now" {
			t.Errorf("expected indexed_by filter, got '%s'", mock.GetWithPaginationCalls[0].Path)
		}
	})

	t.Run("filters by real column server-side without client-side filtering", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data: []any{
				map[string]any{"id": "1", "title": "Column 1", "column_id": "col-1"},
				map[string]any{"id": "2", "title": "Column 2", "column_id": "col-2"},
			},
		}

		result := SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		cardListColumn = "col-1"
		err := cardListCmd.RunE(cardListCmd, []string{})
		cardListColumn = ""

		assertExitCode(t, err, 0)
		if mock.GetWithPaginationCalls[0].Path != "/cards.json?column_ids[]=col-1" {
			t.Errorf("expected server-side column_ids filter, got '%s'", mock.GetWithPaginationCalls[0].Path)
		}

		arr, ok := result.Response.Data.([]any)
		if !ok {
			t.Fatalf("expected array response data, got %T", result.Response.Data)
		}
		if len(arr) != 2 {
			t.Fatalf("expected server response to remain unfiltered client-side, got %d cards", len(arr))
		}
	})

	t.Run("filters by pseudo column maybe server-side without all", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data: []any{
				map[string]any{"id": "1", "title": "Triage", "column": nil},
				map[string]any{"id": "2", "title": "Unexpected extra", "column_id": "col-1"},
			},
		}

		result := SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		cardListColumn = "maybe"
		err := cardListCmd.RunE(cardListCmd, []string{})
		cardListColumn = ""

		assertExitCode(t, err, 0)
		if mock.GetWithPaginationCalls[0].Path != "/cards.json?indexed_by=maybe" {
			t.Errorf("expected server-side maybe filter, got '%s'", mock.GetWithPaginationCalls[0].Path)
		}

		arr, ok := result.Response.Data.([]any)
		if !ok {
			t.Fatalf("expected array response data, got %T", result.Response.Data)
		}
		if len(arr) != 2 {
			t.Fatalf("expected server response to remain unfiltered client-side, got %d cards", len(arr))
		}
	})

	t.Run("uses configured board as default filter", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       []any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		cfg.Board = "999"
		defer resetTest()

		err := cardListCmd.RunE(cardListCmd, []string{})

		assertExitCode(t, err, 0)
		if mock.GetWithPaginationCalls[0].Path != "/cards.json?board_ids[]=999" {
			t.Errorf("expected path '/cards.json?board_ids[]=999', got '%s'", mock.GetWithPaginationCalls[0].Path)
		}
	})

	t.Run("requires authentication", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("", "account", "https://api.example.com")
		defer resetTest()

		err := cardListCmd.RunE(cardListCmd, []string{})
		assertExitCode(t, err, errors.ExitAuthFailure)
	})

	t.Run("applies search filter", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       []any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		cardListSearch = "bug fix"
		err := cardListCmd.RunE(cardListCmd, []string{})
		cardListSearch = ""

		assertExitCode(t, err, 0)
		path := mock.GetWithPaginationCalls[0].Path
		if path != "/cards.json?terms[]=bug&terms[]=fix" {
			t.Errorf("expected path with search terms, got '%s'", path)
		}
	})

	t.Run("applies sort filter", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       []any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		cardListSort = "newest"
		err := cardListCmd.RunE(cardListCmd, []string{})
		cardListSort = ""

		assertExitCode(t, err, 0)
		path := mock.GetWithPaginationCalls[0].Path
		if path != "/cards.json?sorted_by=newest" {
			t.Errorf("expected path with sort, got '%s'", path)
		}
	})

	t.Run("applies creator filter", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       []any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		cardListCreator = "user-123"
		err := cardListCmd.RunE(cardListCmd, []string{})
		cardListCreator = ""

		assertExitCode(t, err, 0)
		path := mock.GetWithPaginationCalls[0].Path
		if path != "/cards.json?creator_ids[]=user-123" {
			t.Errorf("expected path with creator filter, got '%s'", path)
		}
	})

	t.Run("applies unassigned filter", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       []any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		cardListUnassigned = true
		err := cardListCmd.RunE(cardListCmd, []string{})
		cardListUnassigned = false

		assertExitCode(t, err, 0)
		path := mock.GetWithPaginationCalls[0].Path
		if path != "/cards.json?assignment_status=unassigned" {
			t.Errorf("expected path with unassigned filter, got '%s'", path)
		}
	})

	t.Run("applies created filter", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       []any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		cardListCreated = "thisweek"
		err := cardListCmd.RunE(cardListCmd, []string{})
		cardListCreated = ""

		assertExitCode(t, err, 0)
		path := mock.GetWithPaginationCalls[0].Path
		if path != "/cards.json?creation=thisweek" {
			t.Errorf("expected path with created filter, got '%s'", path)
		}
	})

	t.Run("applies closed filter", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       []any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		cardListClosed = "lastmonth"
		err := cardListCmd.RunE(cardListCmd, []string{})
		cardListClosed = ""

		assertExitCode(t, err, 0)
		path := mock.GetWithPaginationCalls[0].Path
		if path != "/cards.json?closure=lastmonth" {
			t.Errorf("expected path with closed filter, got '%s'", path)
		}
	})

	t.Run("combines multiple new filters", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       []any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		cardListBoard = "123"
		cardListSearch = "bug"
		cardListSort = "newest"
		cardListUnassigned = true
		err := cardListCmd.RunE(cardListCmd, []string{})
		cardListBoard = ""
		cardListSearch = ""
		cardListSort = ""
		cardListUnassigned = false

		assertExitCode(t, err, 0)
		path := mock.GetWithPaginationCalls[0].Path
		expected := "/cards.json?board_ids[]=123&terms[]=bug&sorted_by=newest&assignment_status=unassigned"
		if path != expected {
			t.Errorf("expected path '%s', got '%s'", expected, path)
		}
	})

	t.Run("combines column with other filters without changing command shape", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       []any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		cardListBoard = "123"
		cardListColumn = "col-1"
		cardListTag = "tag-1"
		cardListAssignee = "user-1"
		err := cardListCmd.RunE(cardListCmd, []string{})
		cardListBoard = ""
		cardListColumn = ""
		cardListTag = ""
		cardListAssignee = ""

		assertExitCode(t, err, 0)
		path := mock.GetWithPaginationCalls[0].Path
		expected := "/cards.json?board_ids[]=123&column_ids[]=col-1&tag_ids[]=tag-1&assignee_ids[]=user-1"
		if path != expected {
			t.Errorf("expected path '%s', got '%s'", expected, path)
		}
	})
}

func TestCardShow(t *testing.T) {
	t.Run("shows card by number", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetResponse = &client.APIResponse{
			StatusCode: 200,
			Data: map[string]any{
				"id":     "123",
				"number": 42,
				"title":  "Test Card",
			},
		}

		result := SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := cardShowCmd.RunE(cardShowCmd, []string{"42"})
		assertExitCode(t, err, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Response.OK {
			t.Error("expected success response")
		}
		if mock.GetCalls[0].Path != "/cards/42" {
			t.Errorf("expected path '/cards/42', got '%s'", mock.GetCalls[0].Path)
		}
	})

	t.Run("handles not found", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetError = errors.NewNotFoundError("Card not found")

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := cardShowCmd.RunE(cardShowCmd, []string{"999"})
		assertExitCode(t, err, errors.ExitNotFound)
	})
}

func TestCardCreate(t *testing.T) {
	t.Run("creates card with required fields", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{
			StatusCode: 201,
			Location:   "/cards/42",
			Data: map[string]any{
				"id":     "abc",
				"number": 42,
				"title":  "New Card",
			},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		cardCreateBoard = "123"
		cardCreateTitle = "New Card"
		err := cardCreateCmd.RunE(cardCreateCmd, []string{})
		cardCreateBoard = ""
		cardCreateTitle = ""

		assertExitCode(t, err, 0)
		if mock.PostCalls[0].Path != "/cards.json" {
			t.Errorf("expected path '/cards.json', got '%s'", mock.PostCalls[0].Path)
		}

		body := mock.PostCalls[0].Body.(map[string]any)
		if body["board_id"] != "123" {
			t.Errorf("expected board_id '123', got '%v'", body["board_id"])
		}
		if body["title"] != "New Card" {
			t.Errorf("expected title 'New Card', got '%v'", body["title"])
		}
	})

	t.Run("requires board flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		cardCreateBoard = ""
		cardCreateTitle = "Test"
		err := cardCreateCmd.RunE(cardCreateCmd, []string{})
		cardCreateTitle = ""

		assertExitCode(t, err, errors.ExitInvalidArgs)
	})

	t.Run("uses configured board when flag omitted", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{
			StatusCode: 201,
			Data:       map[string]any{"id": "abc", "number": 42},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		cfg.Board = "555"
		defer resetTest()

		cardCreateBoard = ""
		cardCreateTitle = "New Card"
		err := cardCreateCmd.RunE(cardCreateCmd, []string{})
		cardCreateTitle = ""

		assertExitCode(t, err, 0)
		body := mock.PostCalls[0].Body.(map[string]any)
		if body["board_id"] != "555" {
			t.Errorf("expected board_id '555', got '%v'", body["board_id"])
		}
	})

	t.Run("requires title flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		cardCreateBoard = "123"
		cardCreateTitle = ""
		err := cardCreateCmd.RunE(cardCreateCmd, []string{})
		cardCreateBoard = ""

		assertExitCode(t, err, errors.ExitInvalidArgs)
	})

	t.Run("includes optional fields", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{
			StatusCode: 201,
			Data:       map[string]any{"id": "abc", "number": 42},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		cardCreateBoard = "123"
		cardCreateTitle = "Test"
		cardCreateDescription = "<p>Description</p>"
		err := cardCreateCmd.RunE(cardCreateCmd, []string{})
		cardCreateBoard = ""
		cardCreateTitle = ""
		cardCreateDescription = ""

		assertExitCode(t, err, 0)

		body := mock.PostCalls[0].Body.(map[string]any)
		if body["description"] != "<p>Description</p>" {
			t.Errorf("expected description '<p>Description</p>', got '%v'", body["description"])
		}
	})

	t.Run("uploads and appends single inline attachment", func(t *testing.T) {
		tempDir := t.TempDir()
		attachPath := writeTestAttachmentFile(t, tempDir, "single.txt", "single")

		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{
			StatusCode: 201,
			Data:       map[string]any{"id": "abc", "number": 42},
		}
		mock.UploadFileResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       map[string]any{"attachable_sgid": "sgid-single"},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		cardCreateBoard = "123"
		cardCreateTitle = "Test"
		cardCreateDescription = "See attached"
		cardCreateAttach = []string{attachPath}
		err := cardCreateCmd.RunE(cardCreateCmd, []string{})
		cardCreateBoard = ""
		cardCreateTitle = ""
		cardCreateDescription = ""
		cardCreateAttach = nil

		assertExitCode(t, err, 0)

		if len(mock.UploadFileCalls) != 1 || mock.UploadFileCalls[0] != attachPath {
			t.Fatalf("unexpected upload calls: %#v", mock.UploadFileCalls)
		}

		body := mock.PostCalls[0].Body.(map[string]any)
		expected := strings.Join([]string{
			"See attached",
			`<action-text-attachment sgid="sgid-single"></action-text-attachment>`,
		}, "\n")
		if body["description"] != expected {
			t.Errorf("expected description %q, got %v", expected, body["description"])
		}
	})

	t.Run("uploads and appends multiple inline attachments in order", func(t *testing.T) {
		tempDir := t.TempDir()
		attachPath1 := writeTestAttachmentFile(t, tempDir, "first.txt", "first")
		attachPath2 := writeTestAttachmentFile(t, tempDir, "second.txt", "second")

		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{
			StatusCode: 201,
			Data:       map[string]any{"id": "abc", "number": 42},
		}
		mock.UploadFileResponses = []*client.APIResponse{
			{StatusCode: 200, Data: map[string]any{"attachable_sgid": "sgid-1"}},
			{StatusCode: 200, Data: map[string]any{"attachable_sgid": "sgid-2"}},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		cardCreateBoard = "123"
		cardCreateTitle = "Test"
		cardCreateAttach = []string{attachPath1, attachPath2}
		err := cardCreateCmd.RunE(cardCreateCmd, []string{})
		cardCreateBoard = ""
		cardCreateTitle = ""
		cardCreateAttach = nil

		assertExitCode(t, err, 0)

		body := mock.PostCalls[0].Body.(map[string]any)
		expected := strings.Join([]string{
			`<action-text-attachment sgid="sgid-1"></action-text-attachment>`,
			`<action-text-attachment sgid="sgid-2"></action-text-attachment>`,
		}, "\n")
		if body["description"] != expected {
			t.Errorf("expected description %q, got %v", expected, body["description"])
		}
	})
}

func TestCardUpdate(t *testing.T) {
	t.Run("updates card title", func(t *testing.T) {
		mock := NewMockClient()
		mock.PatchResponse = &client.APIResponse{
			StatusCode: 200,
			Data: map[string]any{
				"id":    "abc",
				"title": "Updated Title",
			},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		cardUpdateTitle = "Updated Title"
		err := cardUpdateCmd.RunE(cardUpdateCmd, []string{"42"})
		cardUpdateTitle = ""

		assertExitCode(t, err, 0)
		if mock.PatchCalls[0].Path != "/cards/42" {
			t.Errorf("expected path '/cards/42', got '%s'", mock.PatchCalls[0].Path)
		}
	})

	t.Run("uploads and appends inline attachments", func(t *testing.T) {
		tempDir := t.TempDir()
		attachPath := writeTestAttachmentFile(t, tempDir, "update.txt", "update")

		mock := NewMockClient()
		mock.PatchResponse = &client.APIResponse{StatusCode: 200, Data: map[string]any{"id": "abc"}}
		mock.UploadFileResponse = &client.APIResponse{StatusCode: 200, Data: map[string]any{"attachable_sgid": "sgid-update"}}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		cardUpdateDescription = "Updated body"
		cardUpdateAttach = []string{attachPath}
		err := cardUpdateCmd.RunE(cardUpdateCmd, []string{"42"})
		cardUpdateDescription = ""
		cardUpdateAttach = nil

		assertExitCode(t, err, 0)
		body := mock.PatchCalls[0].Body.(map[string]any)
		expected := strings.Join([]string{
			"Updated body",
			`<action-text-attachment sgid="sgid-update"></action-text-attachment>`,
		}, "\n")
		if body["description"] != expected {
			t.Errorf("expected description %q, got %v", expected, body["description"])
		}
	})

	t.Run("preserves existing description when only attach is provided", func(t *testing.T) {
		tempDir := t.TempDir()
		attachPath := writeTestAttachmentFile(t, tempDir, "update.txt", "update")

		mock := NewMockClient()
		mock.GetResponse = &client.APIResponse{
			StatusCode: 200,
			Data: map[string]any{
				"id":               "abc",
				"description_html": "<p>Existing description</p>",
			},
		}
		mock.PatchResponse = &client.APIResponse{StatusCode: 200, Data: map[string]any{"id": "abc"}}
		mock.UploadFileResponse = &client.APIResponse{StatusCode: 200, Data: map[string]any{"attachable_sgid": "sgid-update"}}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		cardUpdateAttach = []string{attachPath}
		err := cardUpdateCmd.RunE(cardUpdateCmd, []string{"42"})
		cardUpdateAttach = nil

		assertExitCode(t, err, 0)
		if len(mock.GetCalls) == 0 || mock.GetCalls[0].Path != "/cards/42" {
			t.Fatalf("expected existing card fetch before update, got %#v", mock.GetCalls)
		}
		body := mock.PatchCalls[0].Body.(map[string]any)
		expected := strings.Join([]string{
			"<p>Existing description</p>",
			`<action-text-attachment sgid="sgid-update"></action-text-attachment>`,
		}, "\n")
		if body["description"] != expected {
			t.Errorf("expected description %q, got %v", expected, body["description"])
		}
	})
}

func TestCardDelete(t *testing.T) {
	t.Run("deletes card", func(t *testing.T) {
		mock := NewMockClient()
		mock.DeleteResponse = &client.APIResponse{
			StatusCode: 204,
			Data:       map[string]any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := cardDeleteCmd.RunE(cardDeleteCmd, []string{"42"})
		assertExitCode(t, err, 0)

		if mock.DeleteCalls[0].Path != "/cards/42" {
			t.Errorf("expected path '/cards/42', got '%s'", mock.DeleteCalls[0].Path)
		}
	})
}

func TestCardClose(t *testing.T) {
	t.Run("closes card", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       map[string]any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := cardCloseCmd.RunE(cardCloseCmd, []string{"42"})
		assertExitCode(t, err, 0)

		if mock.PostCalls[0].Path != "/cards/42/closure.json" {
			t.Errorf("expected path '/cards/42/closure.json', got '%s'", mock.PostCalls[0].Path)
		}
	})
}

func TestCardReopen(t *testing.T) {
	t.Run("reopens card", func(t *testing.T) {
		mock := NewMockClient()
		mock.DeleteResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       map[string]any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := cardReopenCmd.RunE(cardReopenCmd, []string{"42"})
		assertExitCode(t, err, 0)

		if mock.DeleteCalls[0].Path != "/cards/42/closure.json" {
			t.Errorf("expected path '/cards/42/closure.json', got '%s'", mock.DeleteCalls[0].Path)
		}
	})
}

func TestCardPostpone(t *testing.T) {
	t.Run("postpones card", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       map[string]any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := cardPostponeCmd.RunE(cardPostponeCmd, []string{"42"})
		assertExitCode(t, err, 0)

		if mock.PostCalls[0].Path != "/cards/42/not_now.json" {
			t.Errorf("expected path '/cards/42/not_now.json', got '%s'", mock.PostCalls[0].Path)
		}
	})
}

func TestCardColumn(t *testing.T) {
	t.Run("moves card to column", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       map[string]any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		cardColumnColumn = "col-123"
		err := cardColumnCmd.RunE(cardColumnCmd, []string{"42"})
		cardColumnColumn = ""

		assertExitCode(t, err, 0)
		if mock.PostCalls[0].Path != "/cards/42/triage.json" {
			t.Errorf("expected path '/cards/42/triage.json', got '%s'", mock.PostCalls[0].Path)
		}

		body := mock.PostCalls[0].Body.(map[string]any)
		if body["column_id"] != "col-123" {
			t.Errorf("expected column_id 'col-123', got '%v'", body["column_id"])
		}
	})

	t.Run("moves card to pseudo columns", func(t *testing.T) {
		t.Run("not-now", func(t *testing.T) {
			mock := NewMockClient()
			mock.PostResponse = &client.APIResponse{StatusCode: 200, Data: map[string]any{}}

			SetTestModeWithSDK(mock)
			SetTestConfig("token", "account", "https://api.example.com")
			defer resetTest()

			cardColumnColumn = "not-now"
			err := cardColumnCmd.RunE(cardColumnCmd, []string{"42"})
			cardColumnColumn = ""

			assertExitCode(t, err, 0)
			if len(mock.PostCalls) != 1 || mock.PostCalls[0].Path != "/cards/42/not_now.json" {
				t.Errorf("expected post '/cards/42/not_now.json', got %+v", mock.PostCalls)
			}
		})

		t.Run("maybe", func(t *testing.T) {
			mock := NewMockClient()
			mock.DeleteResponse = &client.APIResponse{StatusCode: 200, Data: map[string]any{}}

			SetTestModeWithSDK(mock)
			SetTestConfig("token", "account", "https://api.example.com")
			defer resetTest()

			cardColumnColumn = "maybe"
			err := cardColumnCmd.RunE(cardColumnCmd, []string{"42"})
			cardColumnColumn = ""

			assertExitCode(t, err, 0)
			if len(mock.DeleteCalls) != 1 || mock.DeleteCalls[0].Path != "/cards/42/triage.json" {
				t.Errorf("expected delete '/cards/42/triage.json', got %+v", mock.DeleteCalls)
			}
		})

		t.Run("done", func(t *testing.T) {
			mock := NewMockClient()
			mock.PostResponse = &client.APIResponse{StatusCode: 200, Data: map[string]any{}}

			SetTestModeWithSDK(mock)
			SetTestConfig("token", "account", "https://api.example.com")
			defer resetTest()

			cardColumnColumn = "done"
			err := cardColumnCmd.RunE(cardColumnCmd, []string{"42"})
			cardColumnColumn = ""

			assertExitCode(t, err, 0)
			if len(mock.PostCalls) != 1 || mock.PostCalls[0].Path != "/cards/42/closure.json" {
				t.Errorf("expected post '/cards/42/closure.json', got %+v", mock.PostCalls)
			}
		})
	})

	t.Run("requires column flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		cardColumnColumn = ""
		err := cardColumnCmd.RunE(cardColumnCmd, []string{"42"})
		assertExitCode(t, err, errors.ExitInvalidArgs)
	})
}

func TestCardUntriage(t *testing.T) {
	t.Run("untriages card", func(t *testing.T) {
		mock := NewMockClient()
		mock.DeleteResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       map[string]any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := cardUntriageCmd.RunE(cardUntriageCmd, []string{"42"})
		assertExitCode(t, err, 0)

		if mock.DeleteCalls[0].Path != "/cards/42/triage.json" {
			t.Errorf("expected path '/cards/42/triage.json', got '%s'", mock.DeleteCalls[0].Path)
		}
	})
}

func TestCardAssign(t *testing.T) {
	t.Run("assigns user to card", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       map[string]any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		cardAssignUser = "user-123"
		err := cardAssignCmd.RunE(cardAssignCmd, []string{"42"})
		cardAssignUser = ""

		assertExitCode(t, err, 0)
		if mock.PostCalls[0].Path != "/cards/42/assignments.json" {
			t.Errorf("expected path '/cards/42/assignments.json', got '%s'", mock.PostCalls[0].Path)
		}

		body := mock.PostCalls[0].Body.(map[string]any)
		if body["assignee_id"] != "user-123" {
			t.Errorf("expected assignee_id 'user-123', got '%v'", body["assignee_id"])
		}
	})

	t.Run("requires user flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		cardAssignUser = ""
		err := cardAssignCmd.RunE(cardAssignCmd, []string{"42"})
		assertExitCode(t, err, errors.ExitInvalidArgs)
	})
}

func TestCardSelfAssign(t *testing.T) {
	t.Run("self-assigns card", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       map[string]any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := cardSelfAssignCmd.RunE(cardSelfAssignCmd, []string{"42"})
		assertExitCode(t, err, 0)

		if mock.PostCalls[0].Path != "/cards/42/self_assignment.json" {
			t.Errorf("expected path '/cards/42/self_assignment.json', got '%s'", mock.PostCalls[0].Path)
		}
	})
}

func TestCardTag(t *testing.T) {
	t.Run("tags card", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       map[string]any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		cardTagTag = "bug"
		err := cardTagCmd.RunE(cardTagCmd, []string{"42"})
		cardTagTag = ""

		assertExitCode(t, err, 0)
		if mock.PostCalls[0].Path != "/cards/42/taggings.json" {
			t.Errorf("expected path '/cards/42/taggings.json', got '%s'", mock.PostCalls[0].Path)
		}

		body := mock.PostCalls[0].Body.(map[string]any)
		if body["tag_title"] != "bug" {
			t.Errorf("expected tag_title 'bug', got '%v'", body["tag_title"])
		}
	})

	t.Run("requires tag flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		cardTagTag = ""
		err := cardTagCmd.RunE(cardTagCmd, []string{"42"})
		assertExitCode(t, err, errors.ExitInvalidArgs)
	})
}

func TestCardWatch(t *testing.T) {
	t.Run("watches card", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       map[string]any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := cardWatchCmd.RunE(cardWatchCmd, []string{"42"})
		assertExitCode(t, err, 0)

		if mock.PostCalls[0].Path != "/cards/42/watch.json" {
			t.Errorf("expected path '/cards/42/watch.json', got '%s'", mock.PostCalls[0].Path)
		}
	})
}

func TestCardUnwatch(t *testing.T) {
	t.Run("unwatches card", func(t *testing.T) {
		mock := NewMockClient()
		mock.DeleteResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       map[string]any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := cardUnwatchCmd.RunE(cardUnwatchCmd, []string{"42"})
		assertExitCode(t, err, 0)

		if mock.DeleteCalls[0].Path != "/cards/42/watch.json" {
			t.Errorf("expected path '/cards/42/watch.json', got '%s'", mock.DeleteCalls[0].Path)
		}
	})
}

func TestCardImageRemove(t *testing.T) {
	t.Run("removes card header image", func(t *testing.T) {
		mock := NewMockClient()
		mock.DeleteResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       map[string]any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := cardImageRemoveCmd.RunE(cardImageRemoveCmd, []string{"42"})
		assertExitCode(t, err, 0)

		if len(mock.DeleteCalls) != 1 {
			t.Fatalf("expected 1 delete call, got %d", len(mock.DeleteCalls))
		}
		if mock.DeleteCalls[0].Path != "/cards/42/image.json" {
			t.Errorf("expected path '/cards/42/image.json', got '%s'", mock.DeleteCalls[0].Path)
		}
	})

	t.Run("requires authentication", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("", "account", "https://api.example.com")
		defer resetTest()

		err := cardImageRemoveCmd.RunE(cardImageRemoveCmd, []string{"42"})
		assertExitCode(t, err, errors.ExitAuthFailure)
	})

	t.Run("handles not found error", func(t *testing.T) {
		mock := NewMockClient()
		mock.DeleteError = errors.NewNotFoundError("Card not found")

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := cardImageRemoveCmd.RunE(cardImageRemoveCmd, []string{"999"})
		assertExitCode(t, err, errors.ExitNotFound)
	})
}

func TestCardGolden(t *testing.T) {
	t.Run("marks card as golden", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       map[string]any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := cardGoldenCmd.RunE(cardGoldenCmd, []string{"42"})
		assertExitCode(t, err, 0)

		if len(mock.PostCalls) != 1 {
			t.Fatalf("expected 1 post call, got %d", len(mock.PostCalls))
		}
		if mock.PostCalls[0].Path != "/cards/42/goldness.json" {
			t.Errorf("expected path '/cards/42/goldness.json', got '%s'", mock.PostCalls[0].Path)
		}
	})

	t.Run("requires authentication", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("", "account", "https://api.example.com")
		defer resetTest()

		err := cardGoldenCmd.RunE(cardGoldenCmd, []string{"42"})
		assertExitCode(t, err, errors.ExitAuthFailure)
	})

	t.Run("handles not found error", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostError = errors.NewNotFoundError("Card not found")

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := cardGoldenCmd.RunE(cardGoldenCmd, []string{"999"})
		assertExitCode(t, err, errors.ExitNotFound)
	})
}

func TestCardUngolden(t *testing.T) {
	t.Run("removes golden status from card", func(t *testing.T) {
		mock := NewMockClient()
		mock.DeleteResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       map[string]any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := cardUngoldenCmd.RunE(cardUngoldenCmd, []string{"42"})
		assertExitCode(t, err, 0)

		if len(mock.DeleteCalls) != 1 {
			t.Fatalf("expected 1 delete call, got %d", len(mock.DeleteCalls))
		}
		if mock.DeleteCalls[0].Path != "/cards/42/goldness.json" {
			t.Errorf("expected path '/cards/42/goldness.json', got '%s'", mock.DeleteCalls[0].Path)
		}
	})

	t.Run("requires authentication", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("", "account", "https://api.example.com")
		defer resetTest()

		err := cardUngoldenCmd.RunE(cardUngoldenCmd, []string{"42"})
		assertExitCode(t, err, errors.ExitAuthFailure)
	})

	t.Run("handles not found error", func(t *testing.T) {
		mock := NewMockClient()
		mock.DeleteError = errors.NewNotFoundError("Card not found")

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := cardUngoldenCmd.RunE(cardUngoldenCmd, []string{"999"})
		assertExitCode(t, err, errors.ExitNotFound)
	})
}

func TestCardMove(t *testing.T) {
	t.Run("moves card to different board", func(t *testing.T) {
		mock := NewMockClient()
		mock.PatchResponse = &client.APIResponse{
			StatusCode: 200,
			Data: map[string]any{
				"id":       "abc",
				"number":   float64(42),
				"title":    "Test Card",
				"board_id": "board-456",
			},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		cardMoveBoard = "board-456"
		err := cardMoveCmd.RunE(cardMoveCmd, []string{"42"})
		cardMoveBoard = ""

		assertExitCode(t, err, 0)
		if len(mock.PatchCalls) != 1 {
			t.Errorf("expected 1 patch call, got %d", len(mock.PatchCalls))
		}
		if mock.PatchCalls[0].Path != "/cards/42/board.json" {
			t.Errorf("expected path '/cards/42/board.json', got '%s'", mock.PatchCalls[0].Path)
		}

		body := mock.PatchCalls[0].Body.(map[string]any)
		if body["board_id"] != "board-456" {
			t.Errorf("expected board_id 'board-456', got '%v'", body["board_id"])
		}
	})

	t.Run("requires --to flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		cardMoveBoard = ""
		err := cardMoveCmd.RunE(cardMoveCmd, []string{"42"})
		assertExitCode(t, err, errors.ExitInvalidArgs)
	})

	t.Run("handles not found error", func(t *testing.T) {
		mock := NewMockClient()
		mock.PatchError = errors.NewNotFoundError("Card not found")

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		cardMoveBoard = "board-456"
		err := cardMoveCmd.RunE(cardMoveCmd, []string{"999"})
		cardMoveBoard = ""

		assertExitCode(t, err, errors.ExitNotFound)
	})
}

func TestCardPublish(t *testing.T) {
	t.Run("publishes card", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := cardPublishCmd.RunE(cardPublishCmd, []string{"42"})
		assertExitCode(t, err, 0)

		if len(mock.PostCalls) != 1 {
			t.Fatalf("expected 1 post call, got %d", len(mock.PostCalls))
		}
		if mock.PostCalls[0].Path != "/cards/42/publish.json" {
			t.Errorf("expected path '/cards/42/publish.json', got '%s'", mock.PostCalls[0].Path)
		}
	})
}

func TestCardMarkRead(t *testing.T) {
	t.Run("marks card as read", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := cardMarkReadCmd.RunE(cardMarkReadCmd, []string{"42"})
		assertExitCode(t, err, 0)

		if len(mock.PostCalls) != 1 {
			t.Fatalf("expected 1 post call, got %d", len(mock.PostCalls))
		}
		if mock.PostCalls[0].Path != "/cards/42/reading.json" {
			t.Errorf("expected path '/cards/42/reading.json', got '%s'", mock.PostCalls[0].Path)
		}
	})
}

func TestCardMarkUnread(t *testing.T) {
	t.Run("marks card as unread", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := cardMarkUnreadCmd.RunE(cardMarkUnreadCmd, []string{"42"})
		assertExitCode(t, err, 0)

		if len(mock.DeleteCalls) != 1 {
			t.Fatalf("expected 1 delete call, got %d", len(mock.DeleteCalls))
		}
		if mock.DeleteCalls[0].Path != "/cards/42/reading.json" {
			t.Errorf("expected path '/cards/42/reading.json', got '%s'", mock.DeleteCalls[0].Path)
		}
	})
}
