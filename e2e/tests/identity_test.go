package tests

import (
	"testing"

	"github.com/basecamp/fizzy-cli/e2e/harness"
)

func TestIdentityShow(t *testing.T) {
	h := harness.New(t)

	t.Run("returns current identity with valid token", func(t *testing.T) {
		result := h.Run("identity", "show")

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if result.Response == nil {
			t.Fatalf("expected JSON response, got nil\nstdout: %s", result.Stdout)
		}

		if !result.Response.OK {
			t.Error("expected ok=true")
		}

		data := result.GetDataMap()
		if data == nil {
			t.Fatal("expected data map")
		}

		// Should have accounts array
		accounts, ok := data["accounts"].([]any)
		if !ok {
			t.Error("expected accounts array in response")
		}

		if len(accounts) == 0 {
			t.Error("expected at least one account")
		}

		// First account should have an id and slug
		if len(accounts) > 0 {
			firstAccount, ok := accounts[0].(map[string]any)
			if !ok {
				t.Error("expected account to be a map")
			} else {
				if _, ok := firstAccount["id"]; !ok {
					t.Error("expected account to have id")
				}
				if _, ok := firstAccount["slug"]; !ok {
					t.Error("expected account to have slug")
				}
			}
		}
	})
}

func TestIdentityShowWithInvalidToken(t *testing.T) {
	cfg := harness.LoadConfig()
	if cfg.Token == "" || cfg.Account == "" {
		t.Skip("FIZZY_TEST_TOKEN or FIZZY_TEST_ACCOUNT not set")
	}

	// Create harness with invalid token
	h := harness.NewWithConfig(t, &harness.Config{
		BinaryPath: cfg.BinaryPath,
		Token:      "invalid-token-12345",
		Account:    cfg.Account,
		APIURL:     cfg.APIURL,
	})

	t.Run("returns auth error with invalid token", func(t *testing.T) {
		result := h.Run("identity", "show")

		if result.ExitCode != harness.ExitAuthFailure {
			t.Errorf("expected exit code %d, got %d", harness.ExitAuthFailure, result.ExitCode)
		}

		if result.Response == nil {
			t.Fatalf("expected JSON response, got nil\nstdout: %s", result.Stdout)
		}

		if result.Response.OK {
			t.Error("expected ok=false")
		}

		if result.Response.Error == "" {
			t.Error("expected error in response")
		} else if result.Response.Code != "auth_required" {
			t.Errorf("expected error code AUTH_ERROR, got %s", result.Response.Code)
		}
	})
}
