package commands

import (
	"testing"

	"github.com/basecamp/fizzy-cli/internal/client"
	"github.com/basecamp/fizzy-cli/internal/errors"
)

func TestUploadFile(t *testing.T) {
	t.Run("uploads file", func(t *testing.T) {
		mock := NewMockClient()
		mock.UploadFileResponse = &client.APIResponse{
			StatusCode: 200,
			Data: map[string]any{
				"signed_id": "abc123",
			},
		}

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		// Test with an existing file (mock_client_test.go exists for sure)
		RunTestCommand(func() {
			uploadFileCmd.Run(uploadFileCmd, []string{"mock_client_test.go"})
		})

		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", result.ExitCode)
		}
		if len(mock.UploadFileCalls) != 1 {
			t.Errorf("expected 1 UploadFile call, got %d", len(mock.UploadFileCalls))
		}
		if mock.UploadFileCalls[0] != "mock_client_test.go" {
			t.Errorf("expected file 'mock_client_test.go', got '%s'", mock.UploadFileCalls[0])
		}
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		mock := NewMockClient()
		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		RunTestCommand(func() {
			uploadFileCmd.Run(uploadFileCmd, []string{"/nonexistent/file.png"})
		})

		if result.ExitCode != errors.ExitError {
			t.Errorf("expected exit code %d, got %d", errors.ExitError, result.ExitCode)
		}
	})
}
