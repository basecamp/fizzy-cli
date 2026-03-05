package commands

import (
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
				map[string]any{"id": "1", "body": "Comment 1"},
				map[string]any{"id": "2", "body": "Comment 2"},
			},
		}

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		commentListCard = "42"
		RunTestCommand(func() {
			commentListCmd.Run(commentListCmd, []string{})
		})
		commentListCard = ""

		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", result.ExitCode)
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

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		commentListCard = "42"
		commentListPage = 12
		commentListAll = false
		RunTestCommand(func() {
			commentListCmd.Run(commentListCmd, []string{})
		})
		commentListCard = ""
		commentListPage = 0

		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", result.ExitCode)
		}
		if mock.GetWithPaginationCalls[0].Path != "/cards/42/comments.json?page=12" {
			t.Errorf("expected path '/cards/42/comments.json?page=12', got '%s'", mock.GetWithPaginationCalls[0].Path)
		}
	})

	t.Run("requires card flag", func(t *testing.T) {
		mock := NewMockClient()
		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		commentListCard = ""
		RunTestCommand(func() {
			commentListCmd.Run(commentListCmd, []string{})
		})

		if result.ExitCode != errors.ExitInvalidArgs {
			t.Errorf("expected exit code %d, got %d", errors.ExitInvalidArgs, result.ExitCode)
		}
	})
}

func TestCommentShow(t *testing.T) {
	t.Run("shows comment by ID", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetResponse = &client.APIResponse{
			StatusCode: 200,
			Data: map[string]any{
				"id":   "comment-1",
				"body": "This is a comment",
			},
		}

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		commentShowCard = "42"
		RunTestCommand(func() {
			commentShowCmd.Run(commentShowCmd, []string{"comment-1"})
		})
		commentShowCard = ""

		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", result.ExitCode)
		}
		if mock.GetCalls[0].Path != "/cards/42/comments/comment-1.json" {
			t.Errorf("expected path '/cards/42/comments/comment-1.json', got '%s'", mock.GetCalls[0].Path)
		}
	})

	t.Run("requires card flag", func(t *testing.T) {
		mock := NewMockClient()
		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		commentShowCard = ""
		RunTestCommand(func() {
			commentShowCmd.Run(commentShowCmd, []string{"comment-1"})
		})

		if result.ExitCode != errors.ExitInvalidArgs {
			t.Errorf("expected exit code %d, got %d", errors.ExitInvalidArgs, result.ExitCode)
		}
	})
}

func TestCommentCreate(t *testing.T) {
	t.Run("creates comment with body", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{
			StatusCode: 201,
			Location:   "https://api.example.com/comments/comment-1",
		}
		mock.FollowLocationResponse = &client.APIResponse{
			StatusCode: 200,
			Data: map[string]any{
				"id":   "comment-1",
				"body": "New comment",
			},
		}

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		commentCreateCard = "42"
		commentCreateBody = "New comment"
		RunTestCommand(func() {
			commentCreateCmd.Run(commentCreateCmd, []string{})
		})
		commentCreateCard = ""
		commentCreateBody = ""

		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", result.ExitCode)
		}
		if mock.PostCalls[0].Path != "/cards/42/comments.json" {
			t.Errorf("expected path '/cards/42/comments.json', got '%s'", mock.PostCalls[0].Path)
		}

		body := mock.PostCalls[0].Body.(map[string]any)
		comment := body["comment"].(map[string]any)
		if comment["body"] != "New comment" {
			t.Errorf("expected body 'New comment', got '%v'", comment["body"])
		}
	})

	t.Run("requires card flag", func(t *testing.T) {
		mock := NewMockClient()
		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		commentCreateCard = ""
		commentCreateBody = "Test"
		RunTestCommand(func() {
			commentCreateCmd.Run(commentCreateCmd, []string{})
		})
		commentCreateBody = ""

		if result.ExitCode != errors.ExitInvalidArgs {
			t.Errorf("expected exit code %d, got %d", errors.ExitInvalidArgs, result.ExitCode)
		}
	})

	t.Run("requires body or body_file", func(t *testing.T) {
		mock := NewMockClient()
		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		commentCreateCard = "42"
		commentCreateBody = ""
		commentCreateBodyFile = ""
		RunTestCommand(func() {
			commentCreateCmd.Run(commentCreateCmd, []string{})
		})
		commentCreateCard = ""

		if result.ExitCode != errors.ExitInvalidArgs {
			t.Errorf("expected exit code %d, got %d", errors.ExitInvalidArgs, result.ExitCode)
		}
	})

	t.Run("includes custom created_at", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{
			StatusCode: 201,
			Location:   "https://api.example.com/comments/comment-1",
		}
		mock.FollowLocationResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       map[string]any{},
		}

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		commentCreateCard = "42"
		commentCreateBody = "Test"
		commentCreateCreatedAt = "2020-01-01T00:00:00Z"
		RunTestCommand(func() {
			commentCreateCmd.Run(commentCreateCmd, []string{})
		})
		commentCreateCard = ""
		commentCreateBody = ""
		commentCreateCreatedAt = ""

		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", result.ExitCode)
		}

		body := mock.PostCalls[0].Body.(map[string]any)
		comment := body["comment"].(map[string]any)
		if comment["created_at"] != "2020-01-01T00:00:00Z" {
			t.Errorf("expected created_at '2020-01-01T00:00:00Z', got '%v'", comment["created_at"])
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
				"body": "Updated comment",
			},
		}

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		commentUpdateCard = "42"
		commentUpdateBody = "Updated comment"
		RunTestCommand(func() {
			commentUpdateCmd.Run(commentUpdateCmd, []string{"comment-1"})
		})
		commentUpdateCard = ""
		commentUpdateBody = ""

		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", result.ExitCode)
		}
		if mock.PatchCalls[0].Path != "/cards/42/comments/comment-1.json" {
			t.Errorf("expected path '/cards/42/comments/comment-1.json', got '%s'", mock.PatchCalls[0].Path)
		}
	})

	t.Run("requires card flag", func(t *testing.T) {
		mock := NewMockClient()
		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		commentUpdateCard = ""
		RunTestCommand(func() {
			commentUpdateCmd.Run(commentUpdateCmd, []string{"comment-1"})
		})

		if result.ExitCode != errors.ExitInvalidArgs {
			t.Errorf("expected exit code %d, got %d", errors.ExitInvalidArgs, result.ExitCode)
		}
	})
}

func TestCommentDelete(t *testing.T) {
	t.Run("deletes comment", func(t *testing.T) {
		mock := NewMockClient()
		mock.DeleteResponse = &client.APIResponse{
			StatusCode: 204,
			Data:       map[string]any{},
		}

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		commentDeleteCard = "42"
		RunTestCommand(func() {
			commentDeleteCmd.Run(commentDeleteCmd, []string{"comment-1"})
		})
		commentDeleteCard = ""

		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", result.ExitCode)
		}
		if mock.DeleteCalls[0].Path != "/cards/42/comments/comment-1.json" {
			t.Errorf("expected path '/cards/42/comments/comment-1.json', got '%s'", mock.DeleteCalls[0].Path)
		}
	})

	t.Run("requires card flag", func(t *testing.T) {
		mock := NewMockClient()
		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		commentDeleteCard = ""
		RunTestCommand(func() {
			commentDeleteCmd.Run(commentDeleteCmd, []string{"comment-1"})
		})

		if result.ExitCode != errors.ExitInvalidArgs {
			t.Errorf("expected exit code %d, got %d", errors.ExitInvalidArgs, result.ExitCode)
		}
	})
}
