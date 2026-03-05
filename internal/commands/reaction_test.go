package commands

import (
	"testing"

	"github.com/basecamp/fizzy-cli/internal/client"
	"github.com/basecamp/fizzy-cli/internal/errors"
)

func TestReactionList(t *testing.T) {
	t.Run("returns list of reactions", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetResponse = &client.APIResponse{
			StatusCode: 200,
			Data: []any{
				map[string]any{"id": "1", "content": "\U0001f44d"},
			},
		}

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		reactionListCard = "42"
		reactionListComment = "comment-1"
		err := reactionListCmd.RunE(reactionListCmd, []string{})
		reactionListCard = ""
		reactionListComment = ""

		assertExitCode(t, err, 0)
		if mock.GetCalls[0].Path != "/cards/42/comments/comment-1/reactions.json" {
			t.Errorf("expected path '/cards/42/comments/comment-1/reactions.json', got '%s'", mock.GetCalls[0].Path)
		}
	})

	t.Run("requires card flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		reactionListCard = ""
		reactionListComment = "comment-1"
		err := reactionListCmd.RunE(reactionListCmd, []string{})
		reactionListComment = ""

		assertExitCode(t, err, errors.ExitInvalidArgs)
	})

	t.Run("lists card reactions without comment flag", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       []any{},
		}

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		reactionListCard = "42"
		reactionListComment = ""
		err := reactionListCmd.RunE(reactionListCmd, []string{})
		reactionListCard = ""

		assertExitCode(t, err, 0)
		if mock.GetCalls[0].Path != "/cards/42/reactions.json" {
			t.Errorf("expected path '/cards/42/reactions.json', got '%s'", mock.GetCalls[0].Path)
		}
	})
}

func TestReactionCreate(t *testing.T) {
	t.Run("creates comment reaction", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{
			StatusCode: 201,
			Data:       map[string]any{},
		}

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		reactionCreateCard = "42"
		reactionCreateComment = "comment-1"
		reactionCreateContent = "\U0001f44d"
		err := reactionCreateCmd.RunE(reactionCreateCmd, []string{})
		reactionCreateCard = ""
		reactionCreateComment = ""
		reactionCreateContent = ""

		assertExitCode(t, err, 0)
		if mock.PostCalls[0].Path != "/cards/42/comments/comment-1/reactions.json" {
			t.Errorf("expected path '/cards/42/comments/comment-1/reactions.json', got '%s'", mock.PostCalls[0].Path)
		}

		body := mock.PostCalls[0].Body.(map[string]any)
		if body["content"] != "\U0001f44d" {
			t.Errorf("expected content '\U0001f44d', got '%v'", body["content"])
		}
	})

	t.Run("creates card reaction without comment flag", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{
			StatusCode: 201,
			Data:       map[string]any{},
		}

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		reactionCreateCard = "42"
		reactionCreateComment = ""
		reactionCreateContent = "\U0001f389"
		err := reactionCreateCmd.RunE(reactionCreateCmd, []string{})
		reactionCreateCard = ""
		reactionCreateContent = ""

		assertExitCode(t, err, 0)
		if mock.PostCalls[0].Path != "/cards/42/reactions.json" {
			t.Errorf("expected path '/cards/42/reactions.json', got '%s'", mock.PostCalls[0].Path)
		}

		body := mock.PostCalls[0].Body.(map[string]any)
		if body["content"] != "\U0001f389" {
			t.Errorf("expected content '\U0001f389', got '%v'", body["content"])
		}
	})
}

func TestReactionDelete(t *testing.T) {
	t.Run("deletes comment reaction", func(t *testing.T) {
		mock := NewMockClient()
		mock.DeleteResponse = &client.APIResponse{
			StatusCode: 204,
			Data:       map[string]any{},
		}

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		reactionDeleteCard = "42"
		reactionDeleteComment = "comment-1"
		err := reactionDeleteCmd.RunE(reactionDeleteCmd, []string{"reaction-1"})
		reactionDeleteCard = ""
		reactionDeleteComment = ""

		assertExitCode(t, err, 0)
		if mock.DeleteCalls[0].Path != "/cards/42/comments/comment-1/reactions/reaction-1.json" {
			t.Errorf("expected path '/cards/42/comments/comment-1/reactions/reaction-1.json', got '%s'", mock.DeleteCalls[0].Path)
		}
	})

	t.Run("deletes card reaction without comment flag", func(t *testing.T) {
		mock := NewMockClient()
		mock.DeleteResponse = &client.APIResponse{
			StatusCode: 204,
			Data:       map[string]any{},
		}

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		reactionDeleteCard = "42"
		reactionDeleteComment = ""
		err := reactionDeleteCmd.RunE(reactionDeleteCmd, []string{"reaction-1"})
		reactionDeleteCard = ""

		assertExitCode(t, err, 0)
		if mock.DeleteCalls[0].Path != "/cards/42/reactions/reaction-1.json" {
			t.Errorf("expected path '/cards/42/reactions/reaction-1.json', got '%s'", mock.DeleteCalls[0].Path)
		}
	})
}
