package commands

import (
	"testing"

	"github.com/basecamp/fizzy-cli/internal/client"
)

func TestNotificationList(t *testing.T) {
	t.Run("returns list of notifications", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data: []any{
				map[string]any{"id": "1", "message": "You have a notification"},
			},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := notificationListCmd.RunE(notificationListCmd, []string{})
		assertExitCode(t, err, 0)
		if mock.GetWithPaginationCalls[0].Path != "/notifications.json" {
			t.Errorf("expected path '/notifications.json', got '%s'", mock.GetWithPaginationCalls[0].Path)
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

		notificationListPage = 2
		notificationListAll = true
		err := notificationListCmd.RunE(notificationListCmd, []string{})
		notificationListPage = 0
		notificationListAll = false

		assertExitCode(t, err, 0)
		if mock.GetWithPaginationCalls[0].Path != "/notifications.json?page=2" {
			t.Errorf("expected path '/notifications.json?page=2', got '%s'", mock.GetWithPaginationCalls[0].Path)
		}
	})
}

func TestNotificationRead(t *testing.T) {
	t.Run("marks notification as read", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       map[string]any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := notificationReadCmd.RunE(notificationReadCmd, []string{"notif-1"})
		assertExitCode(t, err, 0)
		if mock.PostCalls[0].Path != "/notifications/notif-1/reading.json" {
			t.Errorf("expected path '/notifications/notif-1/reading.json', got '%s'", mock.PostCalls[0].Path)
		}
	})
}

func TestNotificationUnread(t *testing.T) {
	t.Run("marks notification as unread", func(t *testing.T) {
		mock := NewMockClient()
		mock.DeleteResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       map[string]any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := notificationUnreadCmd.RunE(notificationUnreadCmd, []string{"notif-1"})
		assertExitCode(t, err, 0)
		if mock.DeleteCalls[0].Path != "/notifications/notif-1/reading.json" {
			t.Errorf("expected path '/notifications/notif-1/reading.json', got '%s'", mock.DeleteCalls[0].Path)
		}
	})
}

func TestNotificationTray(t *testing.T) {
	t.Run("returns notification tray", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetResponse = &client.APIResponse{
			StatusCode: 200,
			Data: []any{
				map[string]any{"id": "1", "read": false},
				map[string]any{"id": "2", "read": false},
			},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := notificationTrayCmd.RunE(notificationTrayCmd, []string{})
		assertExitCode(t, err, 0)
		if mock.GetCalls[0].Path != "/notifications/tray.json" {
			t.Errorf("expected path '/notifications/tray.json', got '%s'", mock.GetCalls[0].Path)
		}
	})

	t.Run("includes read notifications with flag", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetResponse = &client.APIResponse{
			StatusCode: 200,
			Data: []any{
				map[string]any{"id": "1", "read": false},
				map[string]any{"id": "2", "read": true},
			},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		notificationTrayIncludeRead = true
		err := notificationTrayCmd.RunE(notificationTrayCmd, []string{})
		notificationTrayIncludeRead = false

		assertExitCode(t, err, 0)
		if mock.GetCalls[0].Path != "/notifications/tray.json?include_read=true" {
			t.Errorf("expected path '/notifications/tray.json?include_read=true', got '%s'", mock.GetCalls[0].Path)
		}
	})
}

func TestNotificationReadAll(t *testing.T) {
	t.Run("marks all notifications as read", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       map[string]any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := notificationReadAllCmd.RunE(notificationReadAllCmd, []string{})
		assertExitCode(t, err, 0)
		if mock.PostCalls[0].Path != "/notifications/bulk_reading.json" {
			t.Errorf("expected path '/notifications/bulk_reading.json', got '%s'", mock.PostCalls[0].Path)
		}
	})
}
