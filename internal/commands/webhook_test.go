package commands

import (
	"testing"

	"github.com/basecamp/fizzy-cli/internal/client"
	"github.com/basecamp/fizzy-cli/internal/errors"
)

func TestWebhookList(t *testing.T) {
	t.Run("returns list of webhooks", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data: []any{
				map[string]any{"id": "1", "name": "Webhook 1", "payload_url": "https://example.com/hook1", "active": true},
				map[string]any{"id": "2", "name": "Webhook 2", "payload_url": "https://example.com/hook2", "active": false},
			},
		}

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		webhookListBoard = "board-1"
		err := webhookListCmd.RunE(webhookListCmd, []string{})
		webhookListBoard = ""

		assertExitCode(t, err, 0)
		if !result.Response.OK {
			t.Error("expected success response")
		}
		if len(mock.GetWithPaginationCalls) != 1 {
			t.Errorf("expected 1 GetWithPagination call, got %d", len(mock.GetWithPaginationCalls))
		}
		if mock.GetWithPaginationCalls[0].Path != "/boards/board-1/webhooks.json" {
			t.Errorf("expected path '/boards/board-1/webhooks.json', got '%s'", mock.GetWithPaginationCalls[0].Path)
		}
	})

	t.Run("requires board", func(t *testing.T) {
		mock := NewMockClient()
		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		webhookListBoard = ""
		err := webhookListCmd.RunE(webhookListCmd, []string{})

		assertExitCode(t, err, errors.ExitInvalidArgs)
	})

	t.Run("requires authentication", func(t *testing.T) {
		mock := NewMockClient()
		SetTestMode(mock)
		SetTestConfig("", "account", "https://api.example.com")
		defer ResetTestMode()

		webhookListBoard = "board-1"
		err := webhookListCmd.RunE(webhookListCmd, []string{})
		webhookListBoard = ""

		assertExitCode(t, err, errors.ExitAuthFailure)
	})

	t.Run("handles pagination", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       []any{},
		}

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		webhookListBoard = "board-1"
		webhookListPage = 3
		err := webhookListCmd.RunE(webhookListCmd, []string{})
		webhookListBoard = ""
		webhookListPage = 0

		assertExitCode(t, err, 0)
		if mock.GetWithPaginationCalls[0].Path != "/boards/board-1/webhooks.json?page=3" {
			t.Errorf("expected path with page=3, got '%s'", mock.GetWithPaginationCalls[0].Path)
		}
	})
}

func TestWebhookDeliveries(t *testing.T) {
	t.Run("lists webhook deliveries", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data: []any{
				map[string]any{"id": "wd-1", "state": "ok", "created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:01Z"},
			},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		webhookDeliveriesBoard = "board-1"
		err := webhookDeliveriesCmd.RunE(webhookDeliveriesCmd, []string{"wh-1"})
		webhookDeliveriesBoard = ""
		webhookDeliveriesPage = 0
		webhookDeliveriesAll = false

		assertExitCode(t, err, 0)
		if mock.GetWithPaginationCalls[0].Path != "/boards/board-1/webhooks/wh-1/deliveries.json" {
			t.Errorf("expected path '/boards/board-1/webhooks/wh-1/deliveries.json', got '%s'", mock.GetWithPaginationCalls[0].Path)
		}
	})

	t.Run("handles page", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{StatusCode: 200, Data: []any{}}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		webhookDeliveriesBoard = "board-1"
		webhookDeliveriesPage = 2
		err := webhookDeliveriesCmd.RunE(webhookDeliveriesCmd, []string{"wh-1"})
		webhookDeliveriesBoard = ""
		webhookDeliveriesPage = 0

		assertExitCode(t, err, 0)
		if mock.GetWithPaginationCalls[0].Path != "/boards/board-1/webhooks/wh-1/deliveries.json?page=2" {
			t.Errorf("expected path with page=2, got '%s'", mock.GetWithPaginationCalls[0].Path)
		}
	})

	t.Run("handles all", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{StatusCode: 200, Data: []any{map[string]any{"id": "wd-1"}}}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		webhookDeliveriesBoard = "board-1"
		webhookDeliveriesAll = true
		err := webhookDeliveriesCmd.RunE(webhookDeliveriesCmd, []string{"wh-1"})
		webhookDeliveriesBoard = ""
		webhookDeliveriesAll = false

		assertExitCode(t, err, 0)
		if mock.GetWithPaginationCalls[0].Path != "/boards/board-1/webhooks/wh-1/deliveries.json" {
			t.Errorf("expected path '/boards/board-1/webhooks/wh-1/deliveries.json', got '%s'", mock.GetWithPaginationCalls[0].Path)
		}
	})

	t.Run("requires board", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := webhookDeliveriesCmd.RunE(webhookDeliveriesCmd, []string{"wh-1"})
		assertExitCode(t, err, errors.ExitInvalidArgs)
	})
}

func TestWebhookShow(t *testing.T) {
	t.Run("shows webhook by ID", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetResponse = &client.APIResponse{
			StatusCode: 200,
			Data: map[string]any{
				"id":                 "wh-1",
				"name":               "Production",
				"payload_url":        "https://example.com/hook",
				"active":             true,
				"subscribed_actions": []any{"card_published", "card_closed"},
			},
		}

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		webhookShowBoard = "board-1"
		err := webhookShowCmd.RunE(webhookShowCmd, []string{"wh-1"})
		webhookShowBoard = ""

		assertExitCode(t, err, 0)
		if !result.Response.OK {
			t.Error("expected success response")
		}
		if mock.GetCalls[0].Path != "/boards/board-1/webhooks/wh-1.json" {
			t.Errorf("expected path '/boards/board-1/webhooks/wh-1.json', got '%s'", mock.GetCalls[0].Path)
		}
	})

	t.Run("handles not found", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetError = errors.NewNotFoundError("Webhook not found")

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		webhookShowBoard = "board-1"
		err := webhookShowCmd.RunE(webhookShowCmd, []string{"bad-id"})
		webhookShowBoard = ""

		assertExitCode(t, err, errors.ExitNotFound)
	})
}

func TestWebhookCreate(t *testing.T) {
	t.Run("creates webhook with name and url", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{
			StatusCode: 201,
			Location:   "https://api.example.com/boards/board-1/webhooks/wh-new",
			Data:       map[string]any{"id": "wh-new"},
		}
		mock.FollowLocationResponse = &client.APIResponse{
			StatusCode: 200,
			Data: map[string]any{
				"id":          "wh-new",
				"name":        "My Hook",
				"payload_url": "https://example.com/hook",
				"active":      true,
			},
		}

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		webhookCreateBoard = "board-1"
		webhookCreateName = "My Hook"
		webhookCreateURL = "https://example.com/hook"
		err := webhookCreateCmd.RunE(webhookCreateCmd, []string{})
		webhookCreateBoard = ""
		webhookCreateName = ""
		webhookCreateURL = ""

		assertExitCode(t, err, 0)
		if !result.Response.OK {
			t.Error("expected success response")
		}
		if mock.PostCalls[0].Path != "/boards/board-1/webhooks.json" {
			t.Errorf("expected path '/boards/board-1/webhooks.json', got '%s'", mock.PostCalls[0].Path)
		}

		body := mock.PostCalls[0].Body.(map[string]any)
		webhookParams := body["webhook"].(map[string]any)
		if webhookParams["name"] != "My Hook" {
			t.Errorf("expected name 'My Hook', got '%v'", webhookParams["name"])
		}
		if webhookParams["url"] != "https://example.com/hook" {
			t.Errorf("expected url 'https://example.com/hook', got '%v'", webhookParams["url"])
		}
	})

	t.Run("creates webhook with actions", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{
			StatusCode: 201,
			Location:   "https://api.example.com/boards/board-1/webhooks/wh-new",
		}
		mock.FollowLocationResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       map[string]any{"id": "wh-new"},
		}

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		webhookCreateBoard = "board-1"
		webhookCreateName = "My Hook"
		webhookCreateURL = "https://example.com/hook"
		webhookCreateActions = []string{"card_published", "card_closed"}
		err := webhookCreateCmd.RunE(webhookCreateCmd, []string{})
		webhookCreateBoard = ""
		webhookCreateName = ""
		webhookCreateURL = ""
		webhookCreateActions = nil

		assertExitCode(t, err, 0)

		body := mock.PostCalls[0].Body.(map[string]any)
		webhookParams := body["webhook"].(map[string]any)
		actions := webhookParams["subscribed_actions"].([]string)
		if len(actions) != 2 || actions[0] != "card_published" || actions[1] != "card_closed" {
			t.Errorf("expected actions [card_published, card_closed], got %v", actions)
		}
	})

	t.Run("requires name flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		webhookCreateBoard = "board-1"
		webhookCreateName = ""
		webhookCreateURL = "https://example.com/hook"
		err := webhookCreateCmd.RunE(webhookCreateCmd, []string{})
		webhookCreateBoard = ""
		webhookCreateURL = ""

		assertExitCode(t, err, errors.ExitInvalidArgs)
	})

	t.Run("requires url flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		webhookCreateBoard = "board-1"
		webhookCreateName = "My Hook"
		webhookCreateURL = ""
		err := webhookCreateCmd.RunE(webhookCreateCmd, []string{})
		webhookCreateBoard = ""
		webhookCreateName = ""

		assertExitCode(t, err, errors.ExitInvalidArgs)
	})
}

func TestWebhookUpdate(t *testing.T) {
	t.Run("updates webhook name", func(t *testing.T) {
		mock := NewMockClient()
		mock.PatchResponse = &client.APIResponse{
			StatusCode: 200,
			Data: map[string]any{
				"id":   "wh-1",
				"name": "Updated Name",
			},
		}

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		webhookUpdateBoard = "board-1"
		webhookUpdateName = "Updated Name"
		err := webhookUpdateCmd.RunE(webhookUpdateCmd, []string{"wh-1"})
		webhookUpdateBoard = ""
		webhookUpdateName = ""

		assertExitCode(t, err, 0)
		if mock.PatchCalls[0].Path != "/boards/board-1/webhooks/wh-1.json" {
			t.Errorf("expected path '/boards/board-1/webhooks/wh-1.json', got '%s'", mock.PatchCalls[0].Path)
		}

		body := mock.PatchCalls[0].Body.(map[string]any)
		webhookParams := body["webhook"].(map[string]any)
		if webhookParams["name"] != "Updated Name" {
			t.Errorf("expected name 'Updated Name', got '%v'", webhookParams["name"])
		}
	})

	t.Run("updates webhook actions", func(t *testing.T) {
		mock := NewMockClient()
		mock.PatchResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       map[string]any{"id": "wh-1"},
		}

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		webhookUpdateBoard = "board-1"
		webhookUpdateActions = []string{"card_closed"}
		err := webhookUpdateCmd.RunE(webhookUpdateCmd, []string{"wh-1"})
		webhookUpdateBoard = ""
		webhookUpdateActions = nil

		assertExitCode(t, err, 0)

		body := mock.PatchCalls[0].Body.(map[string]any)
		webhookParams := body["webhook"].(map[string]any)
		actions := webhookParams["subscribed_actions"].([]string)
		if len(actions) != 1 || actions[0] != "card_closed" {
			t.Errorf("expected actions [card_closed], got %v", actions)
		}
	})

	t.Run("handles API error", func(t *testing.T) {
		mock := NewMockClient()
		mock.PatchError = errors.NewValidationError("Invalid webhook")

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		webhookUpdateBoard = "board-1"
		webhookUpdateName = "Test"
		err := webhookUpdateCmd.RunE(webhookUpdateCmd, []string{"wh-1"})
		webhookUpdateBoard = ""
		webhookUpdateName = ""

		assertExitCode(t, err, errors.ExitValidation)
	})
}

func TestWebhookDelete(t *testing.T) {
	t.Run("deletes webhook", func(t *testing.T) {
		mock := NewMockClient()
		mock.DeleteResponse = &client.APIResponse{
			StatusCode: 204,
			Data:       map[string]any{},
		}

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		webhookDeleteBoard = "board-1"
		err := webhookDeleteCmd.RunE(webhookDeleteCmd, []string{"wh-1"})
		webhookDeleteBoard = ""

		assertExitCode(t, err, 0)
		if mock.DeleteCalls[0].Path != "/boards/board-1/webhooks/wh-1.json" {
			t.Errorf("expected path '/boards/board-1/webhooks/wh-1.json', got '%s'", mock.DeleteCalls[0].Path)
		}
	})

	t.Run("handles not found", func(t *testing.T) {
		mock := NewMockClient()
		mock.DeleteError = errors.NewNotFoundError("Webhook not found")

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		webhookDeleteBoard = "board-1"
		err := webhookDeleteCmd.RunE(webhookDeleteCmd, []string{"bad-id"})
		webhookDeleteBoard = ""

		assertExitCode(t, err, errors.ExitNotFound)
	})
}

func TestWebhookReactivate(t *testing.T) {
	t.Run("reactivates webhook", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{
			StatusCode: 201,
			Data: map[string]any{
				"id":     "wh-1",
				"active": true,
			},
		}

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		webhookReactivateBoard = "board-1"
		err := webhookReactivateCmd.RunE(webhookReactivateCmd, []string{"wh-1"})
		webhookReactivateBoard = ""

		assertExitCode(t, err, 0)
		if mock.PostCalls[0].Path != "/boards/board-1/webhooks/wh-1/activation.json" {
			t.Errorf("expected path '/boards/board-1/webhooks/wh-1/activation.json', got '%s'", mock.PostCalls[0].Path)
		}
	})

	t.Run("requires board", func(t *testing.T) {
		mock := NewMockClient()
		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		webhookReactivateBoard = ""
		err := webhookReactivateCmd.RunE(webhookReactivateCmd, []string{"wh-1"})

		assertExitCode(t, err, errors.ExitInvalidArgs)
	})
}
