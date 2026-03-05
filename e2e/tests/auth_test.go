package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/basecamp/fizzy-cli/e2e/harness"
)

func TestAuthStatus(t *testing.T) {
	cfg := harness.LoadConfig()
	if cfg.Token == "" {
		t.Skip("FIZZY_TEST_TOKEN not set")
	}

	t.Run("shows authenticated status with valid token in config", func(t *testing.T) {
		// Create a temp HOME with a config file containing the token
		tmpDir, err := os.MkdirTemp("", "fizzy-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		configDir := filepath.Join(tmpDir, ".fizzy")
		os.MkdirAll(configDir, 0755)
		configPath := filepath.Join(configDir, "config.yaml")
		os.WriteFile(configPath, []byte("token: "+cfg.Token+"\n"), 0600)

		result := harness.Execute(cfg.BinaryPath, []string{"auth", "status"}, map[string]string{
			"HOME": tmpDir,
		})

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

		authenticated, ok := data["authenticated"].(bool)
		if !ok || !authenticated {
			t.Error("expected authenticated=true")
		}
	})
}

func TestAuthStatusWithoutToken(t *testing.T) {
	cfg := harness.LoadConfig()
	if cfg.BinaryPath == "" {
		t.Skip("FIZZY_TEST_BINARY not set")
	}

	t.Run("shows not authenticated without token", func(t *testing.T) {
		// Create a temp HOME with no config file
		tmpDir, err := os.MkdirTemp("", "fizzy-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		result := harness.Execute(cfg.BinaryPath, []string{"auth", "status"}, map[string]string{
			"HOME": tmpDir,
		})

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

		authenticated, ok := data["authenticated"].(bool)
		if ok && authenticated {
			t.Error("expected authenticated=false or missing")
		}
	})
}

func TestAuthLogin(t *testing.T) {
	cfg := harness.LoadConfig()
	if cfg.Token == "" {
		t.Skip("FIZZY_TEST_TOKEN not set")
	}

	// Create a temporary config directory
	tmpDir, err := os.MkdirTemp("", "fizzy-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("saves token to config file", func(t *testing.T) {
		// Run login with HOME set to temp directory
		result := harness.Execute(cfg.BinaryPath, []string{"auth", "login", cfg.Token}, map[string]string{
			"HOME": tmpDir,
		})

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		// Check config file was created (preferred path is ~/.config/fizzy/config.yaml)
		preferredPath := filepath.Join(tmpDir, ".config", "fizzy", "config.yaml")
		legacyPath := filepath.Join(tmpDir, ".fizzy", "config.yaml")
		_, errPreferred := os.Stat(preferredPath)
		_, errLegacy := os.Stat(legacyPath)
		if os.IsNotExist(errPreferred) && os.IsNotExist(errLegacy) {
			t.Errorf("config file was not created at either %s or %s", preferredPath, legacyPath)
		}
	})
}

func TestAuthLogout(t *testing.T) {
	cfg := harness.LoadConfig()
	if cfg.Token == "" {
		t.Skip("FIZZY_TEST_TOKEN not set")
	}

	// Create a temporary config directory
	tmpDir, err := os.MkdirTemp("", "fizzy-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configDir := filepath.Join(tmpDir, ".fizzy")
	os.MkdirAll(configDir, 0755)

	// Create a config file
	configPath := filepath.Join(configDir, "config.yaml")
	os.WriteFile(configPath, []byte("token: test-token\n"), 0600)

	t.Run("removes config file on logout", func(t *testing.T) {
		result := harness.Execute(cfg.BinaryPath, []string{"auth", "logout"}, map[string]string{
			"HOME": tmpDir,
		})

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		// Check config file was removed
		if _, err := os.Stat(configPath); !os.IsNotExist(err) {
			t.Error("config file was not removed")
		}
	})

	t.Run("succeeds when already logged out", func(t *testing.T) {
		// Config file already removed, run logout again
		result := harness.Execute(cfg.BinaryPath, []string{"auth", "logout"}, map[string]string{
			"HOME": tmpDir,
		})

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d", harness.ExitSuccess, result.ExitCode)
		}
	})
}
