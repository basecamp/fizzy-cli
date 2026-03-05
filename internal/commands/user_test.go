package commands

import (
	"testing"

	"github.com/basecamp/fizzy-cli/internal/client"
	"github.com/basecamp/fizzy-cli/internal/errors"
)

func TestUserList(t *testing.T) {
	t.Run("returns list of users", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data: []any{
				map[string]any{"id": "1", "name": "User 1"},
				map[string]any{"id": "2", "name": "User 2"},
			},
		}

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		err := userListCmd.RunE(userListCmd, []string{})
		assertExitCode(t, err, 0)
		if mock.GetWithPaginationCalls[0].Path != "/users.json" {
			t.Errorf("expected path '/users.json', got '%s'", mock.GetWithPaginationCalls[0].Path)
		}
	})
}

func TestUserShow(t *testing.T) {
	t.Run("shows user by ID", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetResponse = &client.APIResponse{
			StatusCode: 200,
			Data: map[string]any{
				"id":   "user-1",
				"name": "Test User",
			},
		}

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		err := userShowCmd.RunE(userShowCmd, []string{"user-1"})
		assertExitCode(t, err, 0)
		if mock.GetCalls[0].Path != "/users/user-1.json" {
			t.Errorf("expected path '/users/user-1.json', got '%s'", mock.GetCalls[0].Path)
		}
	})
}

func TestUserUpdate(t *testing.T) {
	t.Run("updates user name", func(t *testing.T) {
		mock := NewMockClient()
		mock.PatchResponse = &client.APIResponse{
			StatusCode: 204,
			Data:       map[string]any{},
		}

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		userUpdateName = "New Name"
		err := userUpdateCmd.RunE(userUpdateCmd, []string{"user-1"})
		userUpdateName = ""

		assertExitCode(t, err, 0)
		if len(mock.PatchCalls) != 1 {
			t.Fatalf("expected 1 patch call, got %d", len(mock.PatchCalls))
		}
		if mock.PatchCalls[0].Path != "/users/user-1.json" {
			t.Errorf("expected path '/users/user-1.json', got '%s'", mock.PatchCalls[0].Path)
		}

		body := mock.PatchCalls[0].Body.(map[string]any)
		userParams := body["user"].(map[string]any)
		if userParams["name"] != "New Name" {
			t.Errorf("expected name 'New Name', got '%v'", userParams["name"])
		}
	})

	t.Run("updates user avatar via multipart", func(t *testing.T) {
		mock := NewMockClient()
		mock.PatchMultipartResponse = &client.APIResponse{
			StatusCode: 204,
			Data:       map[string]any{},
		}

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		userUpdateAvatar = "/path/to/avatar.jpg"
		err := userUpdateCmd.RunE(userUpdateCmd, []string{"user-1"})
		userUpdateAvatar = ""

		assertExitCode(t, err, 0)
		if len(mock.PatchMultipartCalls) != 1 {
			t.Fatalf("expected 1 PatchMultipart call, got %d", len(mock.PatchMultipartCalls))
		}
		if mock.PatchMultipartCalls[0].Path != "/users/user-1.json" {
			t.Errorf("expected path '/users/user-1.json', got '%s'", mock.PatchMultipartCalls[0].Path)
		}
	})

	t.Run("updates name and avatar together via multipart", func(t *testing.T) {
		mock := NewMockClient()
		mock.PatchMultipartResponse = &client.APIResponse{
			StatusCode: 204,
			Data:       map[string]any{},
		}

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		userUpdateName = "New Name"
		userUpdateAvatar = "/path/to/avatar.jpg"
		err := userUpdateCmd.RunE(userUpdateCmd, []string{"user-1"})
		userUpdateName = ""
		userUpdateAvatar = ""

		assertExitCode(t, err, 0)
		if len(mock.PatchMultipartCalls) != 1 {
			t.Fatalf("expected 1 PatchMultipart call, got %d", len(mock.PatchMultipartCalls))
		}
		// When avatar is provided, should use PatchMultipart (not Patch)
		if len(mock.PatchCalls) != 0 {
			t.Errorf("expected 0 Patch calls when avatar is provided, got %d", len(mock.PatchCalls))
		}
	})

	t.Run("requires at least one flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		userUpdateName = ""
		userUpdateAvatar = ""
		err := userUpdateCmd.RunE(userUpdateCmd, []string{"user-1"})

		assertExitCode(t, err, errors.ExitInvalidArgs)
	})
}

func TestUserDeactivate(t *testing.T) {
	t.Run("deactivates user by ID", func(t *testing.T) {
		mock := NewMockClient()
		mock.DeleteResponse = &client.APIResponse{
			StatusCode: 204,
			Data:       map[string]any{},
		}

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		err := userDeactivateCmd.RunE(userDeactivateCmd, []string{"user-1"})
		assertExitCode(t, err, 0)
		if len(mock.DeleteCalls) != 1 {
			t.Fatalf("expected 1 delete call, got %d", len(mock.DeleteCalls))
		}
		if mock.DeleteCalls[0].Path != "/users/user-1.json" {
			t.Errorf("expected path '/users/user-1.json', got '%s'", mock.DeleteCalls[0].Path)
		}
	})

	t.Run("returns not found for non-existent user", func(t *testing.T) {
		mock := NewMockClient()
		mock.DeleteError = errors.NewNotFoundError("User not found")

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		err := userDeactivateCmd.RunE(userDeactivateCmd, []string{"non-existent-user"})
		assertExitCode(t, err, errors.ExitNotFound)
	})
}
