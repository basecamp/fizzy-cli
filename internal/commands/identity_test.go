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
