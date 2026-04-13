package commands

import (
	"strings"
	"testing"

	"github.com/basecamp/fizzy-cli/internal/client"
	"github.com/basecamp/fizzy-cli/internal/errors"
)

func TestCommentList(t *testing.T) {
	t.Run("returns list of comments", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data: []any{
				map[string]any{"id": "1", "body": map[string]any{"html": "Comment 1", "plain_text": "Comment 1"}},
				map[string]any{"id": "2", "body": map[string]any{"html": "Comment 2", "plain_text": "Comment 2"}},
			},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		commentListCard = "42"
		err := commentListCmd.RunE(commentListCmd, []string{})
		commentListCard = ""

		assertExitCode(t, err, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if mock.GetWithPaginationCalls[0].Path != "/cards/42/comments.json" {
			t.Errorf("expected path '/cards/42/comments.json', got '%s'", mock.GetWithPaginationCalls[0].Path)
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

		commentListCard = "42"
		commentListPage = 12
		commentListAll = false
		err := commentListCmd.RunE(commentListCmd, []string{})
		commentListCard = ""
		commentListPage = 0 // reset

		assertExitCode(t, err, 0)
		if mock.GetWithPaginationCalls[0].Path != "/cards/42/comments.json?page=12" {
			t.Errorf("expected path '/cards/42/comments.json?page=12', got '%s'", mock.GetWithPaginationCalls[0].Path)
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

		commentListCard = "42"
		commentListPage = 2
		commentListAll = true
		err := commentListCmd.RunE(commentListCmd, []string{})
		commentListCard = ""
		commentListPage = 0
		commentListAll = false

		assertExitCode(t, err, 0)
		if mock.GetWithPaginationCalls[0].Path != "/cards/42/comments.json?page=2" {
			t.Errorf("expected path '/cards/42/comments.json?page=2', got '%s'", mock.GetWithPaginationCalls[0].Path)
		}
	})

	t.Run("requires card flag for list", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		commentListCard = ""
		err := commentListCmd.RunE(commentListCmd, []string{})
		assertExitCode(t, err, errors.ExitInvalidArgs)
	})
}

func TestCommentShow(t *testing.T) {
	t.Run("shows comment by ID", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetResponse = &client.APIResponse{
			StatusCode: 200,
			Data: map[string]any{
				"id":   "comment-1",
				"body": map[string]any{"html": "This is a comment", "plain_text": "This is a comment"},
			},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		commentShowCard = "42"
		err := commentShowCmd.RunE(commentShowCmd, []string{"comment-1"})
		commentShowCard = ""

		assertExitCode(t, err, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if mock.GetCalls[0].Path != "/cards/42/comments/comment-1" {
			t.Errorf("expected path '/cards/42/comments/comment-1', got '%s'", mock.GetCalls[0].Path)
		}
	})

	t.Run("requires card flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		commentShowCard = ""
		err := commentShowCmd.RunE(commentShowCmd, []string{"comment-1"})
		assertExitCode(t, err, errors.ExitInvalidArgs)
	})
}

func TestCommentCreate(t *testing.T) {
	t.Run("creates comment with body", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{
			StatusCode: 201,
			Location:   "/comments/comment-1",
			Data: map[string]any{
				"id":   "comment-1",
				"body": map[string]any{"html": "New comment", "plain_text": "New comment"},
			},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		commentCreateCard = "42"
		commentCreateBody = "New comment"
		err := commentCreateCmd.RunE(commentCreateCmd, []string{})
		commentCreateCard = ""
		commentCreateBody = ""

		assertExitCode(t, err, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if mock.PostCalls[0].Path != "/cards/42/comments.json" {
			t.Errorf("expected path '/cards/42/comments.json', got '%s'", mock.PostCalls[0].Path)
		}

		body := mock.PostCalls[0].Body.(map[string]any)
		if body["body"] != "New comment" {
			t.Errorf("expected body 'New comment', got '%v'", body["body"])
		}
	})

	t.Run("requires card flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		commentCreateCard = ""
		commentCreateBody = "Test"
		err := commentCreateCmd.RunE(commentCreateCmd, []string{})
		commentCreateBody = ""

		assertExitCode(t, err, errors.ExitInvalidArgs)
	})

	t.Run("requires body, body_file, or attach", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		commentCreateCard = "42"
		commentCreateBody = ""
		commentCreateBodyFile = ""
		commentCreateAttach = nil
		err := commentCreateCmd.RunE(commentCreateCmd, []string{})
		commentCreateCard = ""

		assertExitCode(t, err, errors.ExitInvalidArgs)
	})

	t.Run("includes custom created_at", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{
			StatusCode: 201,
			Data:       map[string]any{"id": "comment-1", "body": map[string]any{"html": "Test", "plain_text": "Test"}},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		commentCreateCard = "42"
		commentCreateBody = "Test"
		commentCreateCreatedAt = "2020-01-01T00:00:00Z"
		err := commentCreateCmd.RunE(commentCreateCmd, []string{})
		commentCreateCard = ""
		commentCreateBody = ""
		commentCreateCreatedAt = ""

		assertExitCode(t, err, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		body := mock.PostCalls[0].Body.(map[string]any)
		if body["created_at"] != "2020-01-01T00:00:00Z" {
			t.Errorf("expected created_at '2020-01-01T00:00:00Z', got '%v'", body["created_at"])
		}
	})

	t.Run("uploads and appends single inline attachment", func(t *testing.T) {
		tempDir := t.TempDir()
		attachPath := writeTestAttachmentFile(t, tempDir, "single.txt", "single")

		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{
			StatusCode: 201,
			Data:       map[string]any{"id": "comment-1", "body": map[string]any{"html": "", "plain_text": ""}},
		}
		mock.UploadFileResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       map[string]any{"attachable_sgid": "sgid-single"},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		commentCreateCard = "42"
		commentCreateBody = "See attached"
		commentCreateAttach = []string{attachPath}
		err := commentCreateCmd.RunE(commentCreateCmd, []string{})
		commentCreateCard = ""
		commentCreateBody = ""
		commentCreateAttach = nil

		assertExitCode(t, err, 0)

		body := mock.PostCalls[0].Body.(map[string]any)
		expected := strings.Join([]string{
			"See attached",
			`<action-text-attachment sgid="sgid-single"></action-text-attachment>`,
		}, "\n")
		if body["body"] != expected {
			t.Errorf("expected body %q, got %v", expected, body["body"])
		}
	})

	t.Run("allows attachment-only comments and preserves order", func(t *testing.T) {
		tempDir := t.TempDir()
		attachPath1 := writeTestAttachmentFile(t, tempDir, "first.txt", "first")
		attachPath2 := writeTestAttachmentFile(t, tempDir, "second.txt", "second")

		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{
			StatusCode: 201,
			Data:       map[string]any{"id": "comment-1", "body": map[string]any{"html": "", "plain_text": ""}},
		}
		mock.UploadFileResponses = []*client.APIResponse{
			{StatusCode: 200, Data: map[string]any{"attachable_sgid": "sgid-1"}},
			{StatusCode: 200, Data: map[string]any{"attachable_sgid": "sgid-2"}},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		commentCreateCard = "42"
		commentCreateAttach = []string{attachPath1, attachPath2}
		err := commentCreateCmd.RunE(commentCreateCmd, []string{})
		commentCreateCard = ""
		commentCreateAttach = nil

		assertExitCode(t, err, 0)

		body := mock.PostCalls[0].Body.(map[string]any)
		expected := strings.Join([]string{
			`<action-text-attachment sgid="sgid-1"></action-text-attachment>`,
			`<action-text-attachment sgid="sgid-2"></action-text-attachment>`,
		}, "\n")
		if body["body"] != expected {
			t.Errorf("expected body %q, got %v", expected, body["body"])
		}
	})
}

func TestCommentUpdate(t *testing.T) {
	t.Run("updates comment body", func(t *testing.T) {
		mock := NewMockClient()
		mock.PatchResponse = &client.APIResponse{
			StatusCode: 200,
			Data: map[string]any{
				"id":   "comment-1",
				"body": map[string]any{"html": "Updated comment", "plain_text": "Updated comment"},
			},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		commentUpdateCard = "42"
		commentUpdateBody = "Updated comment"
		err := commentUpdateCmd.RunE(commentUpdateCmd, []string{"comment-1"})
		commentUpdateCard = ""
		commentUpdateBody = ""

		assertExitCode(t, err, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if mock.PatchCalls[0].Path != "/cards/42/comments/comment-1" {
			t.Errorf("expected path '/cards/42/comments/comment-1', got '%s'", mock.PatchCalls[0].Path)
		}
	})

	t.Run("uploads and appends inline attachments", func(t *testing.T) {
		tempDir := t.TempDir()
		attachPath := writeTestAttachmentFile(t, tempDir, "update.txt", "update")

		mock := NewMockClient()
		mock.PatchResponse = &client.APIResponse{StatusCode: 200, Data: map[string]any{"id": "comment-1"}}
		mock.UploadFileResponse = &client.APIResponse{StatusCode: 200, Data: map[string]any{"attachable_sgid": "sgid-update"}}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		commentUpdateCard = "42"
		commentUpdateBody = "Updated comment"
		commentUpdateAttach = []string{attachPath}
		err := commentUpdateCmd.RunE(commentUpdateCmd, []string{"comment-1"})
		commentUpdateCard = ""
		commentUpdateBody = ""
		commentUpdateAttach = nil

		assertExitCode(t, err, 0)
		body := mock.PatchCalls[0].Body.(map[string]any)
		expected := strings.Join([]string{
			"Updated comment",
			`<action-text-attachment sgid="sgid-update"></action-text-attachment>`,
		}, "\n")
		if body["body"] != expected {
			t.Errorf("expected body %q, got %v", expected, body["body"])
		}
	})

	t.Run("preserves existing body when only attach is provided", func(t *testing.T) {
		tempDir := t.TempDir()
		attachPath := writeTestAttachmentFile(t, tempDir, "update.txt", "update")

		mock := NewMockClient()
		mock.GetResponse = &client.APIResponse{
			StatusCode: 200,
			Data: map[string]any{
				"id": "comment-1",
				"body": map[string]any{
					"html":       "<p>Existing comment</p>",
					"plain_text": "Existing comment",
				},
			},
		}
		mock.PatchResponse = &client.APIResponse{StatusCode: 200, Data: map[string]any{"id": "comment-1"}}
		mock.UploadFileResponse = &client.APIResponse{StatusCode: 200, Data: map[string]any{"attachable_sgid": "sgid-update"}}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		commentUpdateCard = "42"
		commentUpdateAttach = []string{attachPath}
		err := commentUpdateCmd.RunE(commentUpdateCmd, []string{"comment-1"})
		commentUpdateCard = ""
		commentUpdateAttach = nil

		assertExitCode(t, err, 0)
		if len(mock.GetCalls) == 0 || mock.GetCalls[0].Path != "/cards/42/comments/comment-1" {
			t.Fatalf("expected existing comment fetch before update, got %#v", mock.GetCalls)
		}
		body := mock.PatchCalls[0].Body.(map[string]any)
		expected := strings.Join([]string{
			"<p>Existing comment</p>",
			`<action-text-attachment sgid="sgid-update"></action-text-attachment>`,
		}, "\n")
		if body["body"] != expected {
			t.Errorf("expected body %q, got %v", expected, body["body"])
		}
	})

	t.Run("requires card flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		commentUpdateCard = ""
		err := commentUpdateCmd.RunE(commentUpdateCmd, []string{"comment-1"})
		assertExitCode(t, err, errors.ExitInvalidArgs)
	})
}

func TestCommentDelete(t *testing.T) {
	t.Run("deletes comment", func(t *testing.T) {
		mock := NewMockClient()
		mock.DeleteResponse = &client.APIResponse{
			StatusCode: 204,
			Data:       map[string]any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		commentDeleteCard = "42"
		err := commentDeleteCmd.RunE(commentDeleteCmd, []string{"comment-1"})
		commentDeleteCard = ""

		assertExitCode(t, err, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if mock.DeleteCalls[0].Path != "/cards/42/comments/comment-1" {
			t.Errorf("expected path '/cards/42/comments/comment-1', got '%s'", mock.DeleteCalls[0].Path)
		}
	})

	t.Run("requires card flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		commentDeleteCard = ""
		err := commentDeleteCmd.RunE(commentDeleteCmd, []string{"comment-1"})
		assertExitCode(t, err, errors.ExitInvalidArgs)
	})
}
