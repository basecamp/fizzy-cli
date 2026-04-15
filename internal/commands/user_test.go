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

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := userListCmd.RunE(userListCmd, []string{})
		assertExitCode(t, err, 0)
		if mock.GetWithPaginationCalls[0].Path != "/users.json" {
			t.Errorf("expected path '/users.json', got '%s'", mock.GetWithPaginationCalls[0].Path)
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

		userListPage = 2
		userListAll = true
		err := userListCmd.RunE(userListCmd, []string{})
		userListPage = 0
		userListAll = false

		assertExitCode(t, err, 0)
		if mock.GetWithPaginationCalls[0].Path != "/users.json?page=2" {
			t.Errorf("expected path '/users.json?page=2', got '%s'", mock.GetWithPaginationCalls[0].Path)
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

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := userShowCmd.RunE(userShowCmd, []string{"user-1"})
		assertExitCode(t, err, 0)
		if mock.GetCalls[0].Path != "/users/user-1" {
			t.Errorf("expected path '/users/user-1', got '%s'", mock.GetCalls[0].Path)
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

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		userUpdateName = "New Name"
		err := userUpdateCmd.RunE(userUpdateCmd, []string{"user-1"})
		userUpdateName = ""

		assertExitCode(t, err, 0)
		if len(mock.PatchCalls) != 1 {
			t.Fatalf("expected 1 patch call, got %d", len(mock.PatchCalls))
		}
		if mock.PatchCalls[0].Path != "/users/user-1" {
			t.Errorf("expected path '/users/user-1', got '%s'", mock.PatchCalls[0].Path)
		}

		body := mock.PatchCalls[0].Body.(map[string]any)
		if body["name"] != "New Name" {
			t.Errorf("expected name 'New Name', got '%v'", body["name"])
		}
	})

	t.Run("updates user avatar via multipart", func(t *testing.T) {
		mock := NewMockClient()
		mock.PatchMultipartResponse = &client.APIResponse{
			StatusCode: 204,
			Data:       map[string]any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

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

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

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
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

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

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := userDeactivateCmd.RunE(userDeactivateCmd, []string{"user-1"})
		assertExitCode(t, err, 0)
		if len(mock.DeleteCalls) != 1 {
			t.Fatalf("expected 1 delete call, got %d", len(mock.DeleteCalls))
		}
		if mock.DeleteCalls[0].Path != "/users/user-1" {
			t.Errorf("expected path '/users/user-1', got '%s'", mock.DeleteCalls[0].Path)
		}
	})

	t.Run("returns not found for non-existent user", func(t *testing.T) {
		mock := NewMockClient()
		mock.DeleteError = errors.NewNotFoundError("User not found")

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := userDeactivateCmd.RunE(userDeactivateCmd, []string{"non-existent-user"})
		assertExitCode(t, err, errors.ExitNotFound)
	})
}

func TestUserRole(t *testing.T) {
	t.Run("updates user role", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		userRoleRole = "admin"
		err := userRoleCmd.RunE(userRoleCmd, []string{"user-1"})
		userRoleRole = ""

		assertExitCode(t, err, 0)
		if mock.PatchCalls[0].Path != "/users/user-1/role.json" {
			t.Errorf("expected path '/users/user-1/role.json', got '%s'", mock.PatchCalls[0].Path)
		}

		body := mock.PatchCalls[0].Body.(map[string]any)
		if body["role"] != "admin" {
			t.Errorf("expected role 'admin', got '%v'", body["role"])
		}
	})

	t.Run("requires role flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		userRoleRole = ""
		err := userRoleCmd.RunE(userRoleCmd, []string{"user-1"})

		assertExitCode(t, err, errors.ExitInvalidArgs)
	})
}

func TestUserAvatarRemove(t *testing.T) {
	t.Run("removes user avatar", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := userAvatarRemoveCmd.RunE(userAvatarRemoveCmd, []string{"user-1"})
		assertExitCode(t, err, 0)

		if mock.DeleteCalls[0].Path != "/users/user-1/avatar" {
			t.Errorf("expected path '/users/user-1/avatar', got '%s'", mock.DeleteCalls[0].Path)
		}
	})
}

func TestUserExport(t *testing.T) {
	t.Run("creates user export", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{
			StatusCode: 201,
			Data: map[string]any{
				"id":     "export-1",
				"status": "queued",
			},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := userExportCreateCmd.RunE(userExportCreateCmd, []string{"user-1"})
		assertExitCode(t, err, 0)
		if mock.PostCalls[0].Path != "/users/user-1/data_exports.json" {
			t.Errorf("expected path '/users/user-1/data_exports.json', got '%s'", mock.PostCalls[0].Path)
		}
	})

	t.Run("shows user export", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetResponse = &client.APIResponse{
			StatusCode: 200,
			Data: map[string]any{
				"id":     "export-1",
				"status": "complete",
			},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := userExportShowCmd.RunE(userExportShowCmd, []string{"user-1", "export-1"})
		assertExitCode(t, err, 0)
		if mock.GetCalls[0].Path != "/users/user-1/data_exports/export-1" {
			t.Errorf("expected path '/users/user-1/data_exports/export-1', got '%s'", mock.GetCalls[0].Path)
		}
	})
}

func TestUserEmailChange(t *testing.T) {
	t.Run("requests email change", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{StatusCode: 204, Data: nil}

		result := SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		userEmailChangeRequestEmail = "new@example.com"
		err := userEmailChangeRequestCmd.RunE(userEmailChangeRequestCmd, []string{"user-1"})
		userEmailChangeRequestEmail = ""

		assertExitCode(t, err, 0)
		if mock.PostCalls[0].Path != "/users/user-1/email_addresses.json" {
			t.Errorf("expected path '/users/user-1/email_addresses.json', got '%s'", mock.PostCalls[0].Path)
		}
		body := mock.PostCalls[0].Body.(map[string]any)
		if body["email_address"] != "new@example.com" {
			t.Errorf("expected email_address 'new@example.com', got '%v'", body["email_address"])
		}
		data, ok := result.Response.Data.(map[string]any)
		if !ok || data["requested"] != true {
			t.Fatalf("expected explicit requested=true payload, got %#v", result.Response.Data)
		}
	})

	t.Run("confirms email change", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{StatusCode: 204, Data: nil}

		result := SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := userEmailChangeConfirmCmd.RunE(userEmailChangeConfirmCmd, []string{"user-1", "token-123"})

		assertExitCode(t, err, 0)
		if mock.PostCalls[0].Path != "/users/user-1/email_addresses/token-123/confirmation.json" {
			t.Errorf("expected confirmation path, got '%s'", mock.PostCalls[0].Path)
		}
		data, ok := result.Response.Data.(map[string]any)
		if !ok || data["confirmed"] != true {
			t.Fatalf("expected explicit confirmed=true payload, got %#v", result.Response.Data)
		}
	})

	t.Run("requires email flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		userEmailChangeRequestEmail = ""
		err := userEmailChangeRequestCmd.RunE(userEmailChangeRequestCmd, []string{"user-1"})
		assertExitCode(t, err, errors.ExitInvalidArgs)
	})

	t.Run("request requires authentication", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("", "account", "https://api.example.com")
		defer resetTest()

		userEmailChangeRequestEmail = "new@example.com"
		err := userEmailChangeRequestCmd.RunE(userEmailChangeRequestCmd, []string{"user-1"})
		userEmailChangeRequestEmail = ""
		assertExitCode(t, err, errors.ExitAuthFailure)
	})

	t.Run("confirm requires authentication", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("", "account", "https://api.example.com")
		defer resetTest()

		err := userEmailChangeConfirmCmd.RunE(userEmailChangeConfirmCmd, []string{"user-1", "token-123"})
		assertExitCode(t, err, errors.ExitAuthFailure)
	})
}

func TestUserPushSubscriptionCreate(t *testing.T) {
	t.Run("creates push subscription", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		pushSubCreateUser = "user-1"
		pushSubCreateEndpoint = "https://push.example.com"
		pushSubCreateP256dhKey = "key1"
		pushSubCreateAuthKey = "key2"
		err := userPushSubscriptionCreateCmd.RunE(userPushSubscriptionCreateCmd, []string{})
		pushSubCreateUser = ""
		pushSubCreateEndpoint = ""
		pushSubCreateP256dhKey = ""
		pushSubCreateAuthKey = ""

		assertExitCode(t, err, 0)
		if mock.PostCalls[0].Path != "/users/user-1/push_subscriptions.json" {
			t.Errorf("expected path '/users/user-1/push_subscriptions.json', got '%s'", mock.PostCalls[0].Path)
		}
	})

	t.Run("requires user flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		pushSubCreateUser = ""
		pushSubCreateEndpoint = "https://push.example.com"
		pushSubCreateP256dhKey = "key1"
		pushSubCreateAuthKey = "key2"
		err := userPushSubscriptionCreateCmd.RunE(userPushSubscriptionCreateCmd, []string{})
		pushSubCreateEndpoint = ""
		pushSubCreateP256dhKey = ""
		pushSubCreateAuthKey = ""

		assertExitCode(t, err, errors.ExitInvalidArgs)
	})

	t.Run("requires endpoint flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		pushSubCreateUser = "user-1"
		pushSubCreateEndpoint = ""
		pushSubCreateP256dhKey = "key1"
		pushSubCreateAuthKey = "key2"
		err := userPushSubscriptionCreateCmd.RunE(userPushSubscriptionCreateCmd, []string{})
		pushSubCreateUser = ""
		pushSubCreateP256dhKey = ""
		pushSubCreateAuthKey = ""

		assertExitCode(t, err, errors.ExitInvalidArgs)
	})

	t.Run("requires p256dh-key flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		pushSubCreateUser = "user-1"
		pushSubCreateEndpoint = "https://push.example.com"
		pushSubCreateP256dhKey = ""
		pushSubCreateAuthKey = "key2"
		err := userPushSubscriptionCreateCmd.RunE(userPushSubscriptionCreateCmd, []string{})
		pushSubCreateUser = ""
		pushSubCreateEndpoint = ""
		pushSubCreateAuthKey = ""

		assertExitCode(t, err, errors.ExitInvalidArgs)
	})

	t.Run("requires auth-key flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		pushSubCreateUser = "user-1"
		pushSubCreateEndpoint = "https://push.example.com"
		pushSubCreateP256dhKey = "key1"
		pushSubCreateAuthKey = ""
		err := userPushSubscriptionCreateCmd.RunE(userPushSubscriptionCreateCmd, []string{})
		pushSubCreateUser = ""
		pushSubCreateEndpoint = ""
		pushSubCreateP256dhKey = ""

		assertExitCode(t, err, errors.ExitInvalidArgs)
	})
}

func TestUserPushSubscriptionDelete(t *testing.T) {
	t.Run("deletes push subscription", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		pushSubDeleteUser = "user-1"
		err := userPushSubscriptionDeleteCmd.RunE(userPushSubscriptionDeleteCmd, []string{"sub-1"})
		pushSubDeleteUser = ""

		assertExitCode(t, err, 0)
		if mock.DeleteCalls[0].Path != "/users/user-1/push_subscriptions/sub-1" {
			t.Errorf("expected path '/users/user-1/push_subscriptions/sub-1', got '%s'", mock.DeleteCalls[0].Path)
		}
	})

	t.Run("requires user flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		pushSubDeleteUser = ""
		err := userPushSubscriptionDeleteCmd.RunE(userPushSubscriptionDeleteCmd, []string{"sub-1"})

		assertExitCode(t, err, errors.ExitInvalidArgs)
	})
}
