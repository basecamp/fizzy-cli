package commands

import (
	"testing"

	"github.com/basecamp/fizzy-cli/internal/client"
	"github.com/basecamp/fizzy-cli/internal/errors"
)

func TestCardPin(t *testing.T) {
	t.Run("pins a card", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       map[string]any{},
		}

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		err := cardPinCmd.RunE(cardPinCmd, []string{"42"})
		assertExitCode(t, err, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(mock.PostCalls) != 1 {
			t.Fatalf("expected 1 post call, got %d", len(mock.PostCalls))
		}
		if mock.PostCalls[0].Path != "/cards/42/pin.json" {
			t.Errorf("expected path '/cards/42/pin.json', got '%s'", mock.PostCalls[0].Path)
		}
	})

	t.Run("requires authentication", func(t *testing.T) {
		mock := NewMockClient()
		SetTestMode(mock)
		SetTestConfig("", "account", "https://api.example.com")
		defer ResetTestMode()

		err := cardPinCmd.RunE(cardPinCmd, []string{"42"})
		assertExitCode(t, err, errors.ExitAuthFailure)
	})

	t.Run("handles not found error", func(t *testing.T) {
		mock := NewMockClient()
		mock.PostError = errors.NewNotFoundError("Card not found")

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		err := cardPinCmd.RunE(cardPinCmd, []string{"999"})
		assertExitCode(t, err, errors.ExitNotFound)
	})
}

func TestCardUnpin(t *testing.T) {
	t.Run("unpins a card", func(t *testing.T) {
		mock := NewMockClient()
		mock.DeleteResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       map[string]any{},
		}

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		err := cardUnpinCmd.RunE(cardUnpinCmd, []string{"42"})
		assertExitCode(t, err, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(mock.DeleteCalls) != 1 {
			t.Fatalf("expected 1 delete call, got %d", len(mock.DeleteCalls))
		}
		if mock.DeleteCalls[0].Path != "/cards/42/pin.json" {
			t.Errorf("expected path '/cards/42/pin.json', got '%s'", mock.DeleteCalls[0].Path)
		}
	})

	t.Run("requires authentication", func(t *testing.T) {
		mock := NewMockClient()
		SetTestMode(mock)
		SetTestConfig("", "account", "https://api.example.com")
		defer ResetTestMode()

		err := cardUnpinCmd.RunE(cardUnpinCmd, []string{"42"})
		assertExitCode(t, err, errors.ExitAuthFailure)
	})

	t.Run("handles not found error", func(t *testing.T) {
		mock := NewMockClient()
		mock.DeleteError = errors.NewNotFoundError("Card not found")

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		err := cardUnpinCmd.RunE(cardUnpinCmd, []string{"999"})
		assertExitCode(t, err, errors.ExitNotFound)
	})
}

func TestPinList(t *testing.T) {
	t.Run("returns list of pinned cards", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetResponse = &client.APIResponse{
			StatusCode: 200,
			Data: []any{
				map[string]any{"id": "1", "title": "Pinned Card 1"},
				map[string]any{"id": "2", "title": "Pinned Card 2"},
			},
		}

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		err := pinListCmd.RunE(pinListCmd, []string{})
		assertExitCode(t, err, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Response.OK {
			t.Error("expected success response")
		}
		if len(mock.GetCalls) != 1 {
			t.Fatalf("expected 1 get call, got %d", len(mock.GetCalls))
		}
		if mock.GetCalls[0].Path != "/my/pins.json" {
			t.Errorf("expected path '/my/pins.json', got '%s'", mock.GetCalls[0].Path)
		}
		if result.Response.Summary != "2 pinned cards" {
			t.Errorf("expected summary '2 pinned cards', got '%s'", result.Response.Summary)
		}
	})

	t.Run("returns empty list", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       []any{},
		}

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		err := pinListCmd.RunE(pinListCmd, []string{})
		assertExitCode(t, err, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Response.Summary != "0 pinned cards" {
			t.Errorf("expected summary '0 pinned cards', got '%s'", result.Response.Summary)
		}
	})

	t.Run("requires authentication", func(t *testing.T) {
		mock := NewMockClient()
		SetTestMode(mock)
		SetTestConfig("", "account", "https://api.example.com")
		defer ResetTestMode()

		err := pinListCmd.RunE(pinListCmd, []string{})
		assertExitCode(t, err, errors.ExitAuthFailure)
	})

	t.Run("requires account", func(t *testing.T) {
		mock := NewMockClient()
		SetTestMode(mock)
		SetTestConfig("token", "", "https://api.example.com")
		defer ResetTestMode()

		err := pinListCmd.RunE(pinListCmd, []string{})
		assertExitCode(t, err, errors.ExitInvalidArgs)
	})

	t.Run("handles API error", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetError = errors.NewError("Server error")

		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		err := pinListCmd.RunE(pinListCmd, []string{})
		assertExitCode(t, err, errors.ExitError)
	})
}
