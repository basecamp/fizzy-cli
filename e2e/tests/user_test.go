package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/basecamp/fizzy-cli/e2e/harness"
)

func TestUserList(t *testing.T) {
	h := harness.New(t)

	t.Run("returns list of users", func(t *testing.T) {
		result := h.Run("user", "list")

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if result.Response == nil {
			t.Fatalf("expected JSON response, got nil\nstdout: %s", result.Stdout)
		}

		if !result.Response.OK {
			t.Error("expected ok=true")
		}

		arr := result.GetDataArray()
		if arr == nil {
			t.Error("expected data to be an array")
		}

		// Should have at least one user (the authenticated user)
		if len(arr) == 0 {
			t.Error("expected at least one user")
		}
	})

	t.Run("supports --page option", func(t *testing.T) {
		result := h.Run("user", "list", "--page", "1")

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d", harness.ExitSuccess, result.ExitCode)
		}

		if result.Response == nil || !result.Response.OK {
			t.Error("expected successful response")
		}
	})

	t.Run("supports --all flag", func(t *testing.T) {
		result := h.Run("user", "list", "--all")

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d", harness.ExitSuccess, result.ExitCode)
		}
	})
}

func TestUserShow(t *testing.T) {
	h := harness.New(t)

	// First get a valid user ID from the list
	listResult := h.Run("user", "list")
	if listResult.ExitCode != harness.ExitSuccess {
		t.Fatalf("failed to list users: %s", listResult.Stderr)
	}

	users := listResult.GetDataArray()
	if len(users) == 0 {
		t.Skip("no users available")
	}

	firstUser, ok := users[0].(map[string]any)
	if !ok {
		t.Fatal("expected user to be a map")
	}

	userID, ok := firstUser["id"].(string)
	if !ok || userID == "" {
		t.Fatal("expected user to have id")
	}

	t.Run("returns user details", func(t *testing.T) {
		result := h.Run("user", "show", userID)

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if result.Response == nil {
			t.Fatal("expected JSON response")
		}

		if !result.Response.OK {
			t.Error("expected ok=true")
		}

		id := result.GetDataString("id")
		if id != userID {
			t.Errorf("expected id %q, got %q", userID, id)
		}
	})
}

func TestUserShowNotFound(t *testing.T) {
	h := harness.New(t)

	t.Run("returns not found for non-existent user", func(t *testing.T) {
		result := h.Run("user", "show", "non-existent-user-id-12345")

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
	})
}

func TestUserUpdate(t *testing.T) {
	cfg := harness.LoadConfig()
	if cfg.UserID == "" {
		t.Skip("FIZZY_TEST_USER_ID not set, skipping user update tests")
	}

	h := harness.New(t)
	userID := cfg.UserID

	// First get the current name so we can restore it
	showResult := h.Run("user", "show", userID)
	if showResult.ExitCode != harness.ExitSuccess {
		t.Fatalf("failed to show test user: %s", showResult.Stderr)
	}
	originalName := showResult.GetDataString("name")
	if originalName == "" {
		t.Fatal("expected test user to have a name")
	}

	t.Run("update user name", func(t *testing.T) {
		newName := originalName + " Updated"
		result := h.Run("user", "update", userID, "--name", newName)

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if !result.Response.OK {
			t.Errorf("expected ok=true, error: %+v", result.Response.Error)
		}

		// Verify the name was updated
		verifyResult := h.Run("user", "show", userID)
		if verifyResult.ExitCode == harness.ExitSuccess {
			name := verifyResult.GetDataString("name")
			if name != newName {
				t.Errorf("expected name %q, got %q", newName, name)
			}
		}
	})

	t.Run("restore original name", func(t *testing.T) {
		result := h.Run("user", "update", userID, "--name", originalName)

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if !result.Response.OK {
			t.Errorf("expected ok=true, error: %+v", result.Response.Error)
		}

		// Verify restored
		verifyResult := h.Run("user", "show", userID)
		if verifyResult.ExitCode == harness.ExitSuccess {
			name := verifyResult.GetDataString("name")
			if name != originalName {
				t.Errorf("expected name %q, got %q", originalName, name)
			}
		}
	})

	t.Run("update user avatar", func(t *testing.T) {
		wd, _ := os.Getwd()
		fixturePath := filepath.Join(wd, "..", "testdata", "fixtures", "test_image.png")
		if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
			t.Skipf("test fixture not found at %s", fixturePath)
		}

		result := h.Run("user", "update", userID, "--avatar", fixturePath)

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s\nstdout: %s", harness.ExitSuccess, result.ExitCode, result.Stderr, result.Stdout)
		}

		if !result.Response.OK {
			t.Errorf("expected ok=true, error: %+v", result.Response.Error)
		}

		// Verify user still has an avatar URL
		verifyResult := h.Run("user", "show", userID)
		if verifyResult.ExitCode == harness.ExitSuccess {
			avatarURL := verifyResult.GetDataString("avatar_url")
			if avatarURL == "" {
				t.Error("expected user to have an avatar_url after upload")
			}
		}
	})

	t.Run("update name and avatar together", func(t *testing.T) {
		wd, _ := os.Getwd()
		fixturePath := filepath.Join(wd, "..", "testdata", "fixtures", "test_image.png")
		if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
			t.Skipf("test fixture not found at %s", fixturePath)
		}

		newName := originalName + " WithAvatar"
		result := h.Run("user", "update", userID, "--name", newName, "--avatar", fixturePath)

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s\nstdout: %s", harness.ExitSuccess, result.ExitCode, result.Stderr, result.Stdout)
		}

		if !result.Response.OK {
			t.Errorf("expected ok=true, error: %+v", result.Response.Error)
		}

		// Verify name was updated
		verifyResult := h.Run("user", "show", userID)
		if verifyResult.ExitCode == harness.ExitSuccess {
			name := verifyResult.GetDataString("name")
			if name != newName {
				t.Errorf("expected name %q, got %q", newName, name)
			}
		}

		// Restore original name
		h.Run("user", "update", userID, "--name", originalName)
	})

	t.Run("update non-existent user returns not found", func(t *testing.T) {
		result := h.Run("user", "update", "non-existent-user-id-12345", "--name", "Nope")

		if result.ExitCode != harness.ExitNotFound {
			t.Errorf("expected exit code %d, got %d\nstdout: %s", harness.ExitNotFound, result.ExitCode, result.Stdout)
		}

		if result.Response == nil {
			t.Fatal("expected JSON response")
		}

		if result.Response.OK {
			t.Error("expected ok=false")
		}
	})
}

func TestUserDeactivate(t *testing.T) {
	cfg := harness.LoadConfig()
	if cfg.UserID == "" {
		t.Skip("FIZZY_TEST_USER_ID not set, skipping user deactivate tests")
	}

	h := harness.New(t)
	userID := cfg.UserID

	t.Run("deactivate non-existent user returns not found", func(t *testing.T) {
		result := h.Run("user", "deactivate", "non-existent-user-id-12345")

		if result.ExitCode != harness.ExitNotFound {
			t.Errorf("expected exit code %d, got %d\nstdout: %s", harness.ExitNotFound, result.ExitCode, result.Stdout)
		}

		if result.Response == nil {
			t.Fatal("expected JSON response")
		}

		if result.Response.OK {
			t.Error("expected ok=false")
		}
	})

	t.Run("deactivates user", func(t *testing.T) {
		// First verify the user exists
		showResult := h.Run("user", "show", userID)
		if showResult.ExitCode != harness.ExitSuccess {
			t.Fatalf("test user %s not found, cannot test deactivate: %s", userID, showResult.Stderr)
		}

		result := h.Run("user", "deactivate", userID)

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s\nstdout: %s", harness.ExitSuccess, result.ExitCode, result.Stderr, result.Stdout)
		}

		if result.Response == nil {
			t.Fatal("expected JSON response")
		}

		if !result.Response.OK {
			t.Errorf("expected ok=true, error: %+v", result.Response.Error)
		}

		// Verify the user is no longer accessible
		verifyResult := h.Run("user", "show", userID)
		if verifyResult.ExitCode != harness.ExitNotFound {
			t.Errorf("expected deactivated user to return not found, got exit code %d", verifyResult.ExitCode)
		}
	})
}
