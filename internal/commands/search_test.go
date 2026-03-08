package commands

import (
	"testing"

	"github.com/basecamp/fizzy-cli/internal/client"
	"github.com/basecamp/fizzy-cli/internal/errors"
)

func TestSearch(t *testing.T) {
	t.Run("searches cards with single term", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data: []any{
				map[string]any{"id": "1", "title": "Bug fix"},
			},
		}

		result := SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := searchCmd.RunE(searchCmd, []string{"bug"})
		assertExitCode(t, err, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Response.OK {
			t.Error("expected success response")
		}
		path := mock.GetWithPaginationCalls[0].Path
		if path != "/cards.json?terms[]=bug" {
			t.Errorf("expected path '/cards.json?terms[]=bug', got '%s'", path)
		}
	})

	t.Run("searches cards with multiple terms", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       []any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := searchCmd.RunE(searchCmd, []string{"login error"})
		assertExitCode(t, err, 0)

		path := mock.GetWithPaginationCalls[0].Path
		if path != "/cards.json?terms[]=login&terms[]=error" {
			t.Errorf("expected path with multiple terms, got '%s'", path)
		}
	})

	t.Run("combines search with board filter", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       []any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		searchBoard = "123"
		err := searchCmd.RunE(searchCmd, []string{"bug"})
		searchBoard = ""

		assertExitCode(t, err, 0)
		path := mock.GetWithPaginationCalls[0].Path
		if path != "/cards.json?terms[]=bug&board_ids[]=123" {
			t.Errorf("expected path with board filter, got '%s'", path)
		}
	})

	t.Run("applies sort parameter", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       []any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		searchSort = "newest"
		err := searchCmd.RunE(searchCmd, []string{"bug"})
		searchSort = ""

		assertExitCode(t, err, 0)
		path := mock.GetWithPaginationCalls[0].Path
		if path != "/cards.json?terms[]=bug&sorted_by=newest" {
			t.Errorf("expected path with sort, got '%s'", path)
		}
	})

	t.Run("applies indexed-by parameter", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       []any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		searchIndexedBy = "closed"
		err := searchCmd.RunE(searchCmd, []string{"bug"})
		searchIndexedBy = ""

		assertExitCode(t, err, 0)
		path := mock.GetWithPaginationCalls[0].Path
		if path != "/cards.json?terms[]=bug&indexed_by=closed" {
			t.Errorf("expected path with indexed_by, got '%s'", path)
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
}
