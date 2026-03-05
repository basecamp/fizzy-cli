package tests

import (
	"testing"

	"github.com/basecamp/fizzy-cli/e2e/harness"
)

func TestErrorAuthFailure(t *testing.T) {
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

	t.Run("returns auth exit code for auth failure", func(t *testing.T) {
		result := h.Run("board", "list")

		if result.ExitCode != harness.ExitAuthFailure {
			t.Errorf("expected exit code %d, got %d\nstdout: %s",
				harness.ExitAuthFailure, result.ExitCode, result.Stdout)
		}

		if result.Response == nil {
			t.Fatal("expected JSON response")
		}

		if result.Response.OK {
			t.Error("expected ok=false")
		}

		if result.Response.Error == "" {
			t.Error("expected error in response")
		}

		if result.Response.Code != "auth_required" {
			t.Errorf("expected error code auth_required, got %s", result.Response.Code)
		}
	})
}

func TestErrorNotFound(t *testing.T) {
	h := harness.New(t)

	testCases := []struct {
		name string
		args []string
	}{
		{"board show", []string{"board", "show", "non-existent-id"}},
		{"card show", []string{"card", "show", "999999999"}},
		{"user show", []string{"user", "show", "non-existent-id"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name+" returns not_found exit code", func(t *testing.T) {
			result := h.Run(tc.args...)

			if result.ExitCode != harness.ExitNotFound {
				t.Errorf("expected exit code %d, got %d\nstdout: %s",
					harness.ExitNotFound, result.ExitCode, result.Stdout)
			}

			if result.Response == nil {
				t.Fatal("expected JSON response")
			}

			if result.Response.OK {
				t.Error("expected ok=false")
			}

			if result.Response.Error == "" {
				t.Error("expected error in response")
			}

			if result.Response.Code != "not_found" {
				t.Errorf("expected error code not_found, got %s", result.Response.Code)
			}
		})
	}
}

func TestErrorNetworkFailure(t *testing.T) {
	cfg := harness.LoadConfig()
	if cfg.Token == "" || cfg.Account == "" {
		t.Skip("FIZZY_TEST_TOKEN or FIZZY_TEST_ACCOUNT not set")
	}

	// Create harness with invalid API URL (unreachable)
	h := harness.NewWithConfig(t, &harness.Config{
		BinaryPath: cfg.BinaryPath,
		Token:      cfg.Token,
		Account:    cfg.Account,
		APIURL:     "http://localhost:59999", // Unlikely to be listening
	})

	t.Run("returns network exit code for network error", func(t *testing.T) {
		result := h.Run("board", "list")

		if result.ExitCode != harness.ExitNetwork {
			t.Errorf("expected exit code %d, got %d\nstdout: %s\nstderr: %s",
				harness.ExitNetwork, result.ExitCode, result.Stdout, result.Stderr)
		}

		if result.Response == nil {
			t.Fatal("expected JSON response")
		}

		if result.Response.OK {
			t.Error("expected ok=false")
		}

		if result.Response.Error == "" {
			t.Error("expected error in response")
		}

		if result.Response.Code != "network" {
			t.Errorf("expected error code network, got %s", result.Response.Code)
		}
	})
}

func TestErrorResponseFormat(t *testing.T) {
	h := harness.New(t)

	t.Run("error response has correct structure", func(t *testing.T) {
		// Use an invalid board ID to trigger a not found error
		result := h.Run("board", "show", "non-existent-board-id")

		if result.Response == nil {
			t.Fatal("expected JSON response")
		}

		// Check response structure
		if result.Response.OK {
			t.Error("expected ok=false for error")
		}

		if result.Response.Error == "" {
			t.Fatal("expected error message in response")
		}

		// Error should have a code
		if result.Response.Code == "" {
			t.Error("expected error code")
		}
	})
}

func TestSuccessResponseFormat(t *testing.T) {
	h := harness.New(t)

	t.Run("success response has correct structure", func(t *testing.T) {
		result := h.Run("board", "list")

		if result.ExitCode != harness.ExitSuccess {
			t.Fatalf("expected exit code %d, got %d", harness.ExitSuccess, result.ExitCode)
		}

		if result.Response == nil {
			t.Fatal("expected JSON response")
		}

		// Check response structure
		if !result.Response.OK {
			t.Error("expected ok=true")
		}

		// Data should be present (can be empty array)
		if result.Response.Data == nil {
			t.Error("expected data in response")
		}

		// Error should be empty for success
		if result.Response.Error != "" {
			t.Error("expected no error in success response")
		}
	})
}
