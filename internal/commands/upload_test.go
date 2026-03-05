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

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		// Test with an existing file (mock_client_test.go exists for sure)
		err := uploadFileCmd.RunE(uploadFileCmd, []string{"mock_client_test.go"})
		assertExitCode(t, err, 0)
		if len(mock.UploadFileCalls) != 1 {
			t.Errorf("expected 1 UploadFile call, got %d", len(mock.UploadFileCalls))
		}
		if mock.UploadFileCalls[0] != "mock_client_test.go" {
			t.Errorf("expected file 'mock_client_test.go', got '%s'", mock.UploadFileCalls[0])
		}
	})

	t.Run("returns error for missing file", func(t *testing.T) {
		mock := NewMockClient()
		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		err := uploadFileCmd.RunE(uploadFileCmd, []string{"/nonexistent/file.png"})
		assertExitCode(t, err, errors.ExitError)
	})
}
