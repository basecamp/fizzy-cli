package commands

import (
	"testing"

	"github.com/basecamp/fizzy-cli/internal/client"
	"github.com/basecamp/fizzy-cli/internal/errors"
)

func TestStepShow(t *testing.T) {
	t.Run("shows step by ID", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetResponse = &client.APIResponse{
			StatusCode: 200,
			Data: map[string]any{
				"id":      "step-1",
				"content": "Review PR",
			},
		}

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		stepShowCard = "42"
		err := stepShowCmd.RunE(stepShowCmd, []string{"step-1"})
		stepShowCard = ""

		assertExitCode(t, err, 0)
		if mock.GetCalls[0].Path != "/cards/42/steps/step-1.json" {
			t.Errorf("expected path '/cards/42/steps/step-1.json', got '%s'", mock.GetCalls[0].Path)
		}
	})

	t.Run("requires card flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		stepShowCard = ""
		err := stepShowCmd.RunE(stepShowCmd, []string{"step-1"})

		assertExitCode(t, err, errors.ExitInvalidArgs)
	})
}

func TestStepCreate(t *testing.T) {
	t.Run("creates step", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{
			StatusCode: 201,
			Location:   "https://api.example.com/steps/step-1",
		}
		mock.FollowLocationResponse = &client.APIResponse{
			StatusCode: 200,
			Data: map[string]any{
				"id":      "step-1",
				"content": "New step",
			},
		}

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		stepCreateCard = "42"
		stepCreateContent = "New step"
		err := stepCreateCmd.RunE(stepCreateCmd, []string{})
		stepCreateCard = ""
		stepCreateContent = ""

		assertExitCode(t, err, 0)
		if mock.PostCalls[0].Path != "/cards/42/steps.json" {
			t.Errorf("expected path '/cards/42/steps.json', got '%s'", mock.PostCalls[0].Path)
		}

		body := mock.PostCalls[0].Body.(map[string]any)
		stepParams := body["step"].(map[string]any)
		if stepParams["content"] != "New step" {
			t.Errorf("expected content 'New step', got '%v'", stepParams["content"])
		}
	})

	t.Run("creates step with completed flag", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{
			StatusCode: 201,
			Location:   "https://api.example.com/steps/step-1",
		}
		mock.FollowLocationResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       map[string]any{},
		}

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		stepCreateCard = "42"
		stepCreateContent = "Already done"
		stepCreateCompleted = true
		err := stepCreateCmd.RunE(stepCreateCmd, []string{})
		stepCreateCard = ""
		stepCreateContent = ""
		stepCreateCompleted = false

		assertExitCode(t, err, 0)

		body := mock.PostCalls[0].Body.(map[string]any)
		stepParams := body["step"].(map[string]any)
		if stepParams["completed"] != true {
			t.Errorf("expected completed true, got '%v'", stepParams["completed"])
		}
	})

	t.Run("requires card flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		stepCreateCard = ""
		stepCreateContent = "Test"
		err := stepCreateCmd.RunE(stepCreateCmd, []string{})
		stepCreateContent = ""

		assertExitCode(t, err, errors.ExitInvalidArgs)
	})

	t.Run("requires content flag", func(t *testing.T) {
		mock := NewMockClient()
		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		stepCreateCard = "42"
		stepCreateContent = ""
		err := stepCreateCmd.RunE(stepCreateCmd, []string{})
		stepCreateCard = ""

		assertExitCode(t, err, errors.ExitInvalidArgs)
	})
}

func TestStepUpdate(t *testing.T) {
	t.Run("updates step", func(t *testing.T) {
		mock := NewMockClient()
		mock.PatchResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       map[string]any{},
		}

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		stepUpdateCard = "42"
		stepUpdateContent = "Updated content"
		err := stepUpdateCmd.RunE(stepUpdateCmd, []string{"step-1"})
		stepUpdateCard = ""
		stepUpdateContent = ""

		assertExitCode(t, err, 0)
		if mock.PatchCalls[0].Path != "/cards/42/steps/step-1.json" {
			t.Errorf("expected path '/cards/42/steps/step-1.json', got '%s'", mock.PatchCalls[0].Path)
		}
	})
}

func TestStepDelete(t *testing.T) {
	t.Run("deletes step", func(t *testing.T) {
		mock := NewMockClient()
		mock.DeleteResponse = &client.APIResponse{
			StatusCode: 204,
			Data:       map[string]any{},
		}

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		stepDeleteCard = "42"
		err := stepDeleteCmd.RunE(stepDeleteCmd, []string{"step-1"})
		stepDeleteCard = ""

		assertExitCode(t, err, 0)
		if mock.DeleteCalls[0].Path != "/cards/42/steps/step-1.json" {
			t.Errorf("expected path '/cards/42/steps/step-1.json', got '%s'", mock.DeleteCalls[0].Path)
		}
	})
}
