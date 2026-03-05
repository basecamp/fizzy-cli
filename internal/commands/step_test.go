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

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		stepShowCard = "42"
		RunTestCommand(func() {
			stepShowCmd.Run(stepShowCmd, []string{"step-1"})
		})
		stepShowCard = ""

		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", result.ExitCode)
		}
		if mock.GetCalls[0].Path != "/cards/42/steps/step-1.json" {
			t.Errorf("expected path '/cards/42/steps/step-1.json', got '%s'", mock.GetCalls[0].Path)
		}
	})

	t.Run("requires card flag", func(t *testing.T) {
		mock := NewMockClient()
		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		stepShowCard = ""
		RunTestCommand(func() {
			stepShowCmd.Run(stepShowCmd, []string{"step-1"})
		})

		if result.ExitCode != errors.ExitInvalidArgs {
			t.Errorf("expected exit code %d, got %d", errors.ExitInvalidArgs, result.ExitCode)
		}
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

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		stepCreateCard = "42"
		stepCreateContent = "New step"
		RunTestCommand(func() {
			stepCreateCmd.Run(stepCreateCmd, []string{})
		})
		stepCreateCard = ""
		stepCreateContent = ""

		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", result.ExitCode)
		}
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

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		stepCreateCard = "42"
		stepCreateContent = "Already done"
		stepCreateCompleted = true
		RunTestCommand(func() {
			stepCreateCmd.Run(stepCreateCmd, []string{})
		})
		stepCreateCard = ""
		stepCreateContent = ""
		stepCreateCompleted = false

		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", result.ExitCode)
		}

		body := mock.PostCalls[0].Body.(map[string]any)
		stepParams := body["step"].(map[string]any)
		if stepParams["completed"] != true {
			t.Errorf("expected completed true, got '%v'", stepParams["completed"])
		}
	})

	t.Run("requires card flag", func(t *testing.T) {
		mock := NewMockClient()
		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		stepCreateCard = ""
		stepCreateContent = "Test"
		RunTestCommand(func() {
			stepCreateCmd.Run(stepCreateCmd, []string{})
		})
		stepCreateContent = ""

		if result.ExitCode != errors.ExitInvalidArgs {
			t.Errorf("expected exit code %d, got %d", errors.ExitInvalidArgs, result.ExitCode)
		}
	})

	t.Run("requires content flag", func(t *testing.T) {
		mock := NewMockClient()
		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		stepCreateCard = "42"
		stepCreateContent = ""
		RunTestCommand(func() {
			stepCreateCmd.Run(stepCreateCmd, []string{})
		})
		stepCreateCard = ""

		if result.ExitCode != errors.ExitInvalidArgs {
			t.Errorf("expected exit code %d, got %d", errors.ExitInvalidArgs, result.ExitCode)
		}
	})
}

func TestStepUpdate(t *testing.T) {
	t.Run("updates step", func(t *testing.T) {
		mock := NewMockClient()
		mock.PatchResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       map[string]any{},
		}

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		stepUpdateCard = "42"
		stepUpdateContent = "Updated content"
		RunTestCommand(func() {
			stepUpdateCmd.Run(stepUpdateCmd, []string{"step-1"})
		})
		stepUpdateCard = ""
		stepUpdateContent = ""

		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", result.ExitCode)
		}
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

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		stepDeleteCard = "42"
		RunTestCommand(func() {
			stepDeleteCmd.Run(stepDeleteCmd, []string{"step-1"})
		})
		stepDeleteCard = ""

		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", result.ExitCode)
		}
		if mock.DeleteCalls[0].Path != "/cards/42/steps/step-1.json" {
			t.Errorf("expected path '/cards/42/steps/step-1.json', got '%s'", mock.DeleteCalls[0].Path)
		}
	})
}
