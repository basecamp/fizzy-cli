package commands

import (
	"testing"

	"github.com/basecamp/fizzy-cli/internal/client"
	"github.com/basecamp/fizzy-cli/internal/errors"
)

func TestSearch(t *testing.T) {
	t.Run("single-word query hits /search.json with q param", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data: []any{
				map[string]any{"id": "1", "number": float64(42), "title": "Bug fix"},
			},
		}

		result := SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := searchCmd.RunE(searchCmd, []string{"bug"})
		assertExitCode(t, err, 0)

		if !result.Response.OK {
			t.Error("expected success response")
		}
		if len(mock.GetWithPaginationCalls) != 1 {
			t.Fatalf("expected 1 GET call, got %d", len(mock.GetWithPaginationCalls))
		}
		if got := mock.GetWithPaginationCalls[0].Path; got != "/search.json?q=bug" {
			t.Errorf("expected '/search.json?q=bug', got '%s'", got)
		}
	})

	t.Run("multiple args joined into a single q string", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       []any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := searchCmd.RunE(searchCmd, []string{"login", "error"})
		assertExitCode(t, err, 0)

		if len(mock.GetWithPaginationCalls) != 1 {
			t.Fatalf("expected 1 GET call, got %d", len(mock.GetWithPaginationCalls))
		}
		if got := mock.GetWithPaginationCalls[0].Path; got != "/search.json?q=login+error" {
			t.Errorf("expected '/search.json?q=login+error', got '%s'", got)
		}
	})

	t.Run("query with special chars is URL-encoded", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       []any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := searchCmd.RunE(searchCmd, []string{"foo&bar=baz"})
		assertExitCode(t, err, 0)

		if len(mock.GetWithPaginationCalls) != 1 {
			t.Fatalf("expected 1 GET call, got %d", len(mock.GetWithPaginationCalls))
		}
		if got := mock.GetWithPaginationCalls[0].Path; got != "/search.json?q=foo%26bar%3Dbaz" {
			t.Errorf("expected URL-encoded q, got '%s'", got)
		}
	})

	t.Run("no default board injection", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       []any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		cfg.Board = "default-board-id"
		defer resetTest()

		err := searchCmd.RunE(searchCmd, []string{"bug"})
		assertExitCode(t, err, 0)

		if len(mock.GetWithPaginationCalls) != 1 {
			t.Fatalf("expected 1 GET call, got %d", len(mock.GetWithPaginationCalls))
		}
		if got := mock.GetWithPaginationCalls[0].Path; got != "/search.json?q=bug" {
			t.Errorf("expected no board params in path, got '%s'", got)
		}
	})

	t.Run("requires at least one arg", func(t *testing.T) {
		if err := searchCmd.Args(searchCmd, []string{}); err == nil {
			t.Error("expected error when no query args provided")
		}
	})

	t.Run("requires authentication", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("", "account", "https://api.example.com")
		defer resetTest()

		err := searchCmd.RunE(searchCmd, []string{"bug"})
		assertExitCode(t, err, errors.ExitAuthFailure)
	})

	t.Run("propagates not-found from server", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetError = errors.NewNotFoundError("not found")

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := searchCmd.RunE(searchCmd, []string{"bug"})
		assertExitCode(t, err, errors.ExitNotFound)
	})
}
