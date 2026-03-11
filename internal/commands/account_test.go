package commands

import (
	"testing"

	"github.com/basecamp/fizzy-cli/internal/client"
	"github.com/basecamp/fizzy-cli/internal/errors"
)

func TestAccountShow(t *testing.T) {
	t.Run("shows account settings", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetResponse = &client.APIResponse{
			StatusCode: 200,
			Data: map[string]any{
				"id":                           "acc-1",
				"name":                         "37signals",
				"cards_count":                  float64(5),
				"auto_postpone_period_in_days": float64(30),
			},
		}

		result := SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := accountShowCmd.RunE(accountShowCmd, []string{})
		assertExitCode(t, err, 0)

		if !result.Response.OK {
			t.Error("expected success response")
		}
		if len(mock.GetCalls) != 1 {
			t.Errorf("expected 1 Get call, got %d", len(mock.GetCalls))
		}
		if mock.GetCalls[0].Path != "/account/settings.json" {
			t.Errorf("expected path '/account/settings.json', got '%s'", mock.GetCalls[0].Path)
		}
	})

	t.Run("requires authentication", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("", "account", "https://api.example.com")
		defer resetTest()

		err := accountShowCmd.RunE(accountShowCmd, []string{})
		assertExitCode(t, err, errors.ExitAuthFailure)
	})

	t.Run("requires account", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "", "https://api.example.com")
		defer resetTest()

		err := accountShowCmd.RunE(accountShowCmd, []string{})
		assertExitCode(t, err, errors.ExitInvalidArgs)
	})
}

func TestAccountEntropy(t *testing.T) {
	t.Run("updates account auto-postpone period", func(t *testing.T) {
		mock := NewMockClient()
		mock.PutResponse = &client.APIResponse{
			StatusCode: 200,
			Data: map[string]any{
				"id":                           "acc-1",
				"name":                         "37signals",
				"auto_postpone_period_in_days": float64(30),
			},
		}

		result := SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		accountEntropyAutoPostponePeriodInDays = 30
		err := accountEntropyCmd.RunE(accountEntropyCmd, []string{})
		accountEntropyAutoPostponePeriodInDays = 0

		assertExitCode(t, err, 0)

		if !result.Response.OK {
			t.Error("expected success response")
		}
		if len(mock.PutCalls) != 1 {
			t.Errorf("expected 1 Put call, got %d", len(mock.PutCalls))
		}
		if mock.PutCalls[0].Path != "/account/entropy.json" {
			t.Errorf("expected path '/account/entropy.json', got '%s'", mock.PutCalls[0].Path)
		}
		body, ok := mock.PutCalls[0].Body.(map[string]any)
		if !ok {
			t.Fatal("expected map body")
		}
		if body["auto_postpone_period_in_days"] != float64(30) {
			t.Errorf("expected auto_postpone_period_in_days 30, got %v", body["auto_postpone_period_in_days"])
		}
	})

	t.Run("requires auto_postpone_period_in_days flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		accountEntropyAutoPostponePeriodInDays = 0
		err := accountEntropyCmd.RunE(accountEntropyCmd, []string{})
		assertExitCode(t, err, errors.ExitInvalidArgs)
	})

	t.Run("rejects invalid period", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		accountEntropyAutoPostponePeriodInDays = 45
		err := accountEntropyCmd.RunE(accountEntropyCmd, []string{})
		accountEntropyAutoPostponePeriodInDays = 0

		assertExitCode(t, err, errors.ExitInvalidArgs)
	})

	t.Run("handles API error", func(t *testing.T) {
		mock := NewMockClient()
		mock.PutError = errors.NewForbiddenError("Admin role required")

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		accountEntropyAutoPostponePeriodInDays = 30
		err := accountEntropyCmd.RunE(accountEntropyCmd, []string{})
		accountEntropyAutoPostponePeriodInDays = 0

		assertExitCode(t, err, errors.ExitForbidden)
	})
}

func TestAccountSettingsUpdate(t *testing.T) {
	t.Run("updates account settings with name flag", func(t *testing.T) {
		mock := NewMockClient()

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		accountSettingsUpdateName = "New Name"
		err := accountSettingsUpdateCmd.RunE(accountSettingsUpdateCmd, []string{})
		accountSettingsUpdateName = ""

		assertExitCode(t, err, 0)
		if mock.PatchCalls[0].Path != "/account/settings.json" {
			t.Errorf("expected path '/account/settings.json', got '%s'", mock.PatchCalls[0].Path)
		}
		body := mock.PatchCalls[0].Body.(map[string]any)
		if body["name"] != "New Name" {
			t.Errorf("expected name 'New Name', got '%v'", body["name"])
		}
	})

	t.Run("requires name flag", func(t *testing.T) {
		mock := NewMockClient()

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		accountSettingsUpdateName = ""
		err := accountSettingsUpdateCmd.RunE(accountSettingsUpdateCmd, []string{})

		assertExitCode(t, err, errors.ExitInvalidArgs)
	})
}

func TestAccountExportCreate(t *testing.T) {
	t.Run("creates export", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{
			StatusCode: 201,
			Data:       map[string]any{"id": "export-1", "status": "pending"},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := accountExportCreateCmd.RunE(accountExportCreateCmd, []string{})
		assertExitCode(t, err, 0)
		if mock.PostCalls[0].Path != "/account/exports.json" {
			t.Errorf("expected path '/account/exports.json', got '%s'", mock.PostCalls[0].Path)
		}
	})
}

func TestAccountExportShow(t *testing.T) {
	t.Run("shows export by ID", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       map[string]any{"id": "export-1", "status": "completed"},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := accountExportShowCmd.RunE(accountExportShowCmd, []string{"export-1"})
		assertExitCode(t, err, 0)
		if mock.GetCalls[0].Path != "/account/exports/export-1" {
			t.Errorf("expected path '/account/exports/export-1', got '%s'", mock.GetCalls[0].Path)
		}
	})
}

func TestAccountJoinCodeShow(t *testing.T) {
	t.Run("shows join code", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       map[string]any{"code": "abc123", "usage_limit": float64(10)},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := accountJoinCodeShowCmd.RunE(accountJoinCodeShowCmd, []string{})
		assertExitCode(t, err, 0)
		if mock.GetCalls[0].Path != "/account/join_code.json" {
			t.Errorf("expected path '/account/join_code.json', got '%s'", mock.GetCalls[0].Path)
		}
	})
}

func TestAccountJoinCodeReset(t *testing.T) {
	t.Run("resets join code", func(t *testing.T) {
		mock := NewMockClient()
		mock.DeleteResponse = &client.APIResponse{
			StatusCode: 204,
			Data:       map[string]any{},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := accountJoinCodeResetCmd.RunE(accountJoinCodeResetCmd, []string{})
		assertExitCode(t, err, 0)
		if mock.DeleteCalls[0].Path != "/account/join_code.json" {
			t.Errorf("expected path '/account/join_code.json', got '%s'", mock.DeleteCalls[0].Path)
		}
	})
}

func TestAccountJoinCodeUpdate(t *testing.T) {
	t.Run("updates join code with usage-limit flag", func(t *testing.T) {
		mock := NewMockClient()

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		accountJoinCodeUpdateUsageLimit = 10
		err := accountJoinCodeUpdateCmd.RunE(accountJoinCodeUpdateCmd, []string{})
		accountJoinCodeUpdateUsageLimit = 0

		assertExitCode(t, err, 0)
		if mock.PatchCalls[0].Path != "/account/join_code.json" {
			t.Errorf("expected path '/account/join_code.json', got '%s'", mock.PatchCalls[0].Path)
		}
		body := mock.PatchCalls[0].Body.(map[string]any)
		if body["usage_limit"] == nil {
			t.Error("expected body to contain 'usage_limit'")
		}
	})

	t.Run("requires usage-limit flag", func(t *testing.T) {
		mock := NewMockClient()

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		accountJoinCodeUpdateUsageLimit = 0
		err := accountJoinCodeUpdateCmd.RunE(accountJoinCodeUpdateCmd, []string{})

		assertExitCode(t, err, errors.ExitInvalidArgs)
	})
}
