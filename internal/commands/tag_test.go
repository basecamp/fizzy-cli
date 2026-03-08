package commands

import (
	"testing"

	"github.com/basecamp/fizzy-cli/internal/client"
)

func TestTagList(t *testing.T) {
	t.Run("returns list of tags", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data: []any{
				map[string]any{"id": "1", "title": "bug"},
				map[string]any{"id": "2", "title": "feature"},
			},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := tagListCmd.RunE(tagListCmd, []string{})
		assertExitCode(t, err, 0)
		if mock.GetWithPaginationCalls[0].Path != "/tags.json" {
			t.Errorf("expected path '/tags.json', got '%s'", mock.GetWithPaginationCalls[0].Path)
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

		tagListPage = 2
		tagListAll = true
		err := tagListCmd.RunE(tagListCmd, []string{})
		tagListPage = 0
		tagListAll = false

		assertExitCode(t, err, 0)
		if mock.GetWithPaginationCalls[0].Path != "/tags.json?page=2" {
			t.Errorf("expected path '/tags.json?page=2', got '%s'", mock.GetWithPaginationCalls[0].Path)
		}
	})
}
