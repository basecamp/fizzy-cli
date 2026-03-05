package commands

import (
	"testing"

	"github.com/basecamp/fizzy-cli/internal/client"
	"github.com/basecamp/fizzy-cli/internal/errors"
)

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

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		RunTestCommand(func() {
			identityShowCmd.Run(identityShowCmd, []string{})
		})

		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", result.ExitCode)
		}
	})

	t.Run("requires authentication", func(t *testing.T) {
		mock := NewMockClient()
		result := SetTestMode(mock)
		SetTestConfig("", "", "https://api.example.com") // No token
		defer ResetTestMode()

		RunTestCommand(func() {
			identityShowCmd.Run(identityShowCmd, []string{})
		})

		if result.ExitCode != errors.ExitAuthFailure {
			t.Errorf("expected exit code %d, got %d", errors.ExitAuthFailure, result.ExitCode)
		}
	})
}
