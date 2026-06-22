package commands

import (
	"testing"

	"github.com/basecamp/fizzy-cli/internal/client"
	"github.com/basecamp/fizzy-cli/internal/errors"
)

func TestIdentityTimezoneUpdate(t *testing.T) {
	t.Run("updates timezone", func(t *testing.T) {
		mock := NewMockClient()
		mock.PatchResponse = &client.APIResponse{
			StatusCode: 200,
			Data: map[string]any{
				"timezone_name": "America/New_York",
				"updated_at":    "2026-06-03T21:15:00Z",
			},
		}

		result := SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		identityTimezoneUpdateTimezone = "America/New_York"
		defer func() {
			identityTimezoneUpdateTimezone = ""
			resetTest()
		}()

		err := identityTimezoneUpdateCmd.RunE(identityTimezoneUpdateCmd, []string{})
		assertExitCode(t, err, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(mock.PatchCalls) != 1 {
			t.Fatalf("expected 1 patch call, got %d", len(mock.PatchCalls))
		}
		if mock.PatchCalls[0].Path != "/my/timezone.json" {
			t.Errorf("expected path '/my/timezone.json', got %q", mock.PatchCalls[0].Path)
		}
		body, ok := mock.PatchCalls[0].Body.(map[string]any)
		if !ok {
			t.Fatalf("expected map body, got %#v", mock.PatchCalls[0].Body)
		}
		if body["timezone_name"] != "America/New_York" {
			t.Errorf("expected timezone_name body, got %#v", body)
		}
		if result.Response.Summary != "Timezone updated" {
			t.Errorf("expected timezone summary, got %q", result.Response.Summary)
		}
		if got := responseDataMap(t, result)["updated_at"]; got != "2026-06-03T21:15:00Z" {
			t.Errorf("expected timezone response body updated_at, got %#v", got)
		}
	})

	t.Run("falls back to requested timezone for empty response", func(t *testing.T) {
		mock := NewMockClient()
		mock.PatchResponse = &client.APIResponse{StatusCode: 204, Data: nil}

		result := SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		identityTimezoneUpdateTimezone = "America/New_York"
		defer func() {
			identityTimezoneUpdateTimezone = ""
			resetTest()
		}()

		err := identityTimezoneUpdateCmd.RunE(identityTimezoneUpdateCmd, []string{})
		assertExitCode(t, err, 0)

		if got := responseDataMap(t, result)["timezone_name"]; got != "America/New_York" {
			t.Errorf("expected fallback timezone_name, got %#v", got)
		}
	})

	t.Run("requires timezone", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		identityTimezoneUpdateTimezone = ""
		defer resetTest()

		err := identityTimezoneUpdateCmd.RunE(identityTimezoneUpdateCmd, []string{})
		assertExitCode(t, err, errors.ExitInvalidArgs)
	})

	t.Run("requires account", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("token", "", "https://api.example.com")
		identityTimezoneUpdateTimezone = "America/New_York"
		defer func() {
			identityTimezoneUpdateTimezone = ""
			resetTest()
		}()

		err := identityTimezoneUpdateCmd.RunE(identityTimezoneUpdateCmd, []string{})
		assertExitCode(t, err, errors.ExitInvalidArgs)
	})
}

func TestIdentityShow(t *testing.T) {
	t.Run("shows identity", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetResponse = &client.APIResponse{
			StatusCode: 200,
			Data: map[string]any{
				"id":    "user-123",
				"email": "test@example.com",
				"accounts": []any{
					map[string]any{"slug": "123456"},
				},
			},
		}

		SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer resetTest()

		err := identityShowCmd.RunE(identityShowCmd, []string{})
		assertExitCode(t, err, 0)
	})

	t.Run("requires authentication", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("", "", "https://api.example.com") // No token
		defer resetTest()

		err := identityShowCmd.RunE(identityShowCmd, []string{})
		assertExitCode(t, err, errors.ExitAuthFailure)
	})
}
