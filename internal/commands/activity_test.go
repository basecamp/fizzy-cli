package commands

import (
	"testing"

	"github.com/basecamp/fizzy-cli/internal/client"
	"github.com/basecamp/fizzy-cli/internal/errors"
)

func TestActivityList(t *testing.T) {
	t.Run("returns list of activities", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data: []any{
				map[string]any{"id": "1", "action": "card_created", "description": "Created a card"},
			},
		}

		result := SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := activityListCmd.RunE(activityListCmd, []string{})
		assertExitCode(t, err, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Response.OK {
			t.Error("expected success response")
		}
		if mock.GetWithPaginationCalls[0].Path != "/activities.json" {
			t.Errorf("expected path '/activities.json', got '%s'", mock.GetWithPaginationCalls[0].Path)
		}
	})

	t.Run("applies board filter", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{StatusCode: 200, Data: []any{}}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		activityListBoard = "board-123"
		err := activityListCmd.RunE(activityListCmd, []string{})
		activityListBoard = ""

		assertExitCode(t, err, 0)
		if mock.GetWithPaginationCalls[0].Path != "/activities.json?board_ids[]=board-123" {
			t.Errorf("expected board filter path, got '%s'", mock.GetWithPaginationCalls[0].Path)
		}
	})

	t.Run("applies creator filter", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{StatusCode: 200, Data: []any{}}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		activityListCreator = "user-123"
		err := activityListCmd.RunE(activityListCmd, []string{})
		activityListCreator = ""

		assertExitCode(t, err, 0)
		if mock.GetWithPaginationCalls[0].Path != "/activities.json?creator_ids[]=user-123" {
			t.Errorf("expected creator filter path, got '%s'", mock.GetWithPaginationCalls[0].Path)
		}
	})

	t.Run("passes page", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{StatusCode: 200, Data: []any{}}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		activityListPage = 3
		err := activityListCmd.RunE(activityListCmd, []string{})
		activityListPage = 0

		assertExitCode(t, err, 0)
		if mock.GetWithPaginationCalls[0].Path != "/activities.json?page=3" {
			t.Errorf("expected page path, got '%s'", mock.GetWithPaginationCalls[0].Path)
		}
	})

	t.Run("passes all", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{StatusCode: 200, Data: []any{map[string]any{"id": "1"}}}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		activityListAll = true
		err := activityListCmd.RunE(activityListCmd, []string{})
		activityListAll = false

		assertExitCode(t, err, 0)
		if mock.GetWithPaginationCalls[0].Path != "/activities.json" {
			t.Errorf("expected path '/activities.json', got '%s'", mock.GetWithPaginationCalls[0].Path)
		}
	})

	t.Run("combines board and creator filters", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{StatusCode: 200, Data: []any{}}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		activityListBoard = "board-123"
		activityListCreator = "user-123"
		err := activityListCmd.RunE(activityListCmd, []string{})
		activityListBoard = ""
		activityListCreator = ""

		assertExitCode(t, err, 0)
		if mock.GetWithPaginationCalls[0].Path != "/activities.json?board_ids[]=board-123&creator_ids[]=user-123" {
			t.Errorf("expected combined filter path, got '%s'", mock.GetWithPaginationCalls[0].Path)
		}
	})

	t.Run("next-page breadcrumb preserves active filters", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       []any{map[string]any{"id": "1"}},
			LinkNext:   "/activities.json?board_ids[]=board-123&creator_ids[]=user-123&page=2",
		}

		result := SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		activityListBoard = "board-123"
		activityListCreator = "user-123"
		err := activityListCmd.RunE(activityListCmd, []string{})
		activityListBoard = ""
		activityListCreator = ""

		assertExitCode(t, err, 0)

		var nextCmd string
		for _, b := range result.Response.Breadcrumbs {
			if b.Action == "next" {
				nextCmd = b.Cmd
				break
			}
		}
		expected := "fizzy activity list --board board-123 --creator user-123 --page 2"
		if nextCmd != expected {
			t.Errorf("expected next breadcrumb %q, got %q", expected, nextCmd)
		}
	})

	t.Run("rejects positional args", func(t *testing.T) {
		if err := activityListCmd.Args(activityListCmd, []string{"unexpected"}); err == nil {
			t.Fatal("expected positional args to be rejected")
		}
	})

	t.Run("requires authentication", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("", "account", "https://api.example.com")
		defer resetTest()

		err := activityListCmd.RunE(activityListCmd, []string{})
		assertExitCode(t, err, errors.ExitAuthFailure)
	})
}
