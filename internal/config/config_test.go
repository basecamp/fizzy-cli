package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultAPIURL(t *testing.T) {
	if DefaultAPIURL != "https://app.fizzy.do" {
		t.Errorf("expected DefaultAPIURL 'https://app.fizzy.do', got '%s'", DefaultAPIURL)
	}
}

func TestLoad_DefaultValues(t *testing.T) {
	// Clear environment variables
	os.Unsetenv("FIZZY_TOKEN")
	os.Unsetenv("FIZZY_ACCOUNT")
	os.Unsetenv("FIZZY_API_URL")

	// Use a temp home directory to avoid loading real config
	origHome := os.Getenv("HOME")
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	cfg := Load()

	if cfg.APIURL != DefaultAPIURL {
		t.Errorf("expected APIURL '%s', got '%s'", DefaultAPIURL, cfg.APIURL)
	}
	if cfg.Token != "" {
		t.Errorf("expected Token to be empty, got '%s'", cfg.Token)
	}
	if cfg.Account != "" {
		t.Errorf("expected Account to be empty, got '%s'", cfg.Account)
	}
}

func TestLoad_FromEnvironment(t *testing.T) {
	// Use a temp home directory
	origHome := os.Getenv("HOME")
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	// Set environment variables
	os.Setenv("FIZZY_TOKEN", "env-token-123")
	os.Setenv("FIZZY_ACCOUNT", "env-account-456")
	os.Setenv("FIZZY_API_URL", "https://custom.api.url")
	os.Setenv("FIZZY_BOARD", "env-board-789")
	defer func() {
		os.Unsetenv("FIZZY_TOKEN")
		os.Unsetenv("FIZZY_ACCOUNT")
		os.Unsetenv("FIZZY_API_URL")
		os.Unsetenv("FIZZY_BOARD")
	}()

	cfg := Load()

	if cfg.Token != "env-token-123" {
		t.Errorf("expected Token 'env-token-123', got '%s'", cfg.Token)
	}
	if cfg.Account != "env-account-456" {
		t.Errorf("expected Account 'env-account-456', got '%s'", cfg.Account)
	}
	if cfg.APIURL != "https://custom.api.url" {
		t.Errorf("expected APIURL 'https://custom.api.url', got '%s'", cfg.APIURL)
	}
	if cfg.Board != "env-board-789" {
		t.Errorf("expected Board 'env-board-789', got '%s'", cfg.Board)
	}
}

func TestLoad_FromConfigFile(t *testing.T) {
	// Clear environment variables
	os.Unsetenv("FIZZY_TOKEN")
	os.Unsetenv("FIZZY_ACCOUNT")
	os.Unsetenv("FIZZY_API_URL")

	// Create temp home directory with config file
	origHome := os.Getenv("HOME")
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	// Create config directory and file
	configDir := filepath.Join(tempDir, ".fizzy")
	os.MkdirAll(configDir, 0700)
	configFile := filepath.Join(configDir, "config.yaml")

	configContent := `token: file-token-789
account: file-account-012
api_url: https://file.api.url
board: file-board-345
`
	os.WriteFile(configFile, []byte(configContent), 0600)

	cfg := Load()

	if cfg.Token != "file-token-789" {
		t.Errorf("expected Token 'file-token-789', got '%s'", cfg.Token)
	}
	if cfg.Account != "file-account-012" {
		t.Errorf("expected Account 'file-account-012', got '%s'", cfg.Account)
	}
	if cfg.APIURL != "https://file.api.url" {
		t.Errorf("expected APIURL 'https://file.api.url', got '%s'", cfg.APIURL)
	}
	if cfg.Board != "file-board-345" {
		t.Errorf("expected Board 'file-board-345', got '%s'", cfg.Board)
	}
}

func TestLoad_EmptyAPIURLDefaults(t *testing.T) {
	// Clear environment variables
	os.Unsetenv("FIZZY_TOKEN")
	os.Unsetenv("FIZZY_ACCOUNT")
	os.Unsetenv("FIZZY_API_URL")

	// Create temp home directory with config file
	origHome := os.Getenv("HOME")
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	configDir := filepath.Join(tempDir, ".fizzy")
	os.MkdirAll(configDir, 0700)
	configFile := filepath.Join(configDir, "config.yaml")

	configContent := `token: file-token
account: file-account
api_url: ""
`
	os.WriteFile(configFile, []byte(configContent), 0600)

	cfg := Load()

	if cfg.APIURL != DefaultAPIURL {
		t.Errorf("expected APIURL '%s', got '%s'", DefaultAPIURL, cfg.APIURL)
	}
}

func TestLoad_EnvOverridesFile(t *testing.T) {
	// Create temp home directory with config file
	origHome := os.Getenv("HOME")
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	// Create config file
	configDir := filepath.Join(tempDir, ".fizzy")
	os.MkdirAll(configDir, 0700)
	configFile := filepath.Join(configDir, "config.yaml")
	configContent := `token: file-token
account: file-account
api_url: https://file.api.url
`
	os.WriteFile(configFile, []byte(configContent), 0600)

	// Set environment variable (should override file)
	os.Setenv("FIZZY_TOKEN", "env-token-override")
	defer os.Unsetenv("FIZZY_TOKEN")

	cfg := Load()

	// Token should come from env
	if cfg.Token != "env-token-override" {
		t.Errorf("expected Token 'env-token-override', got '%s'", cfg.Token)
	}
	// Account should come from file
	if cfg.Account != "file-account" {
		t.Errorf("expected Account 'file-account', got '%s'", cfg.Account)
	}
}

func TestConfigPath(t *testing.T) {
	origHome := os.Getenv("HOME")
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := filepath.Join(tempDir, ".config", "fizzy", "config.yaml")
	if path != expected {
		t.Errorf("expected path '%s', got '%s'", expected, path)
	}

	// Check that directory was created
	configDir := filepath.Join(tempDir, ".config", "fizzy")
	info, err := os.Stat(configDir)
	if err != nil {
		t.Fatalf("config directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected config directory to be a directory")
	}
}

func TestConfig_Save(t *testing.T) {
	origHome := os.Getenv("HOME")
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	cfg := &Config{
		Token:   "saved-token",
		Account: "saved-account",
		APIURL:  "https://saved.api.url",
	}

	err := cfg.Save()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Read back the file
	configFile := filepath.Join(tempDir, ".config", "fizzy", "config.yaml")
	data, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "saved-token") {
		t.Error("expected config to contain 'saved-token'")
	}
	if !strings.Contains(content, "saved-account") {
		t.Error("expected config to contain 'saved-account'")
	}
	if !strings.Contains(content, "https://saved.api.url") {
		t.Error("expected config to contain 'https://saved.api.url'")
	}
}

func TestConfigPath_PrefersExistingAlternatePath(t *testing.T) {
	origHome := os.Getenv("HOME")
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	// Create config in alternate location only.
	altDir := filepath.Join(tempDir, ".config", "fizzy")
	if err := os.MkdirAll(altDir, 0700); err != nil {
		t.Fatalf("failed to create alt config dir: %v", err)
	}
	altFile := filepath.Join(altDir, "config.yaml")
	if err := os.WriteFile(altFile, []byte("token: alt\n"), 0600); err != nil {
		t.Fatalf("failed to write alt config: %v", err)
	}

	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != altFile {
		t.Errorf("expected path '%s', got '%s'", altFile, path)
	}
}

func TestDelete(t *testing.T) {
	origHome := os.Getenv("HOME")
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	// Create a config file
	cfg := &Config{Token: "test"}
	cfg.Save()

	// Verify it exists
	configFile := filepath.Join(tempDir, ".config", "fizzy", "config.yaml")
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Fatal("config file should exist before delete")
	}

	// Delete it
	err := Delete()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it's gone
	if _, err := os.Stat(configFile); !os.IsNotExist(err) {
		t.Error("config file should not exist after delete")
	}
}

func TestDelete_NonExistent(t *testing.T) {
	origHome := os.Getenv("HOME")
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	// Delete should not error if file doesn't exist
	err := Delete()
	if err != nil {
		t.Fatalf("unexpected error when deleting non-existent file: %v", err)
	}
}

func TestExists(t *testing.T) {
	origHome := os.Getenv("HOME")
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	// Should not exist initially
	if Exists() {
		t.Error("expected Exists() to return false initially")
	}

	// Create config
	cfg := &Config{Token: "test"}
	cfg.Save()

	// Should exist now
	if !Exists() {
		t.Error("expected Exists() to return true after save")
	}
}

func TestGlobalConfigPaths(t *testing.T) {
	origHome := os.Getenv("HOME")
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	paths := globalConfigPaths()

	if len(paths) != 2 {
		t.Fatalf("expected 2 config paths, got %d", len(paths))
	}

	expected1 := filepath.Join(tempDir, ".config", "fizzy", "config.yaml")
	expected2 := filepath.Join(tempDir, ".fizzy", "config.yaml")

	if paths[0] != expected1 {
		t.Errorf("expected first path '%s', got '%s'", expected1, paths[0])
	}
	if paths[1] != expected2 {
		t.Errorf("expected second path '%s', got '%s'", expected2, paths[1])
	}
}

func TestLoad_AlternateConfigPath(t *testing.T) {
	// Clear environment variables
	os.Unsetenv("FIZZY_TOKEN")
	os.Unsetenv("FIZZY_ACCOUNT")
	os.Unsetenv("FIZZY_API_URL")

	origHome := os.Getenv("HOME")
	tempDir := t.TempDir()
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", origHome)

	// Create config in alternate location
	configDir := filepath.Join(tempDir, ".config", "fizzy")
	os.MkdirAll(configDir, 0700)
	configFile := filepath.Join(configDir, "config.yaml")

	configContent := `token: alt-token
account: alt-account
`
	os.WriteFile(configFile, []byte(configContent), 0600)

	cfg := Load()

	if cfg.Token != "alt-token" {
		t.Errorf("expected Token 'alt-token', got '%s'", cfg.Token)
	}
	if cfg.Account != "alt-account" {
		t.Errorf("expected Account 'alt-account', got '%s'", cfg.Account)
	}
}

// Tests for local project config (.fizzy.yaml)

func TestLocalConfigFile(t *testing.T) {
	if LocalConfigFile != ".fizzy.yaml" {
		t.Errorf("expected LocalConfigFile '.fizzy.yaml', got '%s'", LocalConfigFile)
	}
}

func TestLoad_LocalConfigOverridesGlobal(t *testing.T) {
	// Clear environment variables
	os.Unsetenv("FIZZY_TOKEN")
	os.Unsetenv("FIZZY_ACCOUNT")
	os.Unsetenv("FIZZY_API_URL")

	// Setup temp directories
	origHome := os.Getenv("HOME")
	homeDir := t.TempDir()
	projectDir := t.TempDir()
	os.Setenv("HOME", homeDir)
	defer os.Setenv("HOME", origHome)

	// Create global config
	globalConfigDir := filepath.Join(homeDir, ".fizzy")
	os.MkdirAll(globalConfigDir, 0700)
	globalContent := `token: global-token
account: global-account
api_url: https://global.api.url
board: global-board
`
	os.WriteFile(filepath.Join(globalConfigDir, "config.yaml"), []byte(globalContent), 0600)

	// Create local config in project directory
	localContent := `account: local-account
api_url: https://local.api.url
board: local-board
`
	os.WriteFile(filepath.Join(projectDir, LocalConfigFile), []byte(localContent), 0600)

	// Set working directory to project
	SetTestWorkingDir(projectDir)
	defer ResetTestWorkingDir()

	cfg := Load()

	// Token should come from global (local doesn't override empty values)
	if cfg.Token != "global-token" {
		t.Errorf("expected Token 'global-token', got '%s'", cfg.Token)
	}
	// Account should come from local
	if cfg.Account != "local-account" {
		t.Errorf("expected Account 'local-account', got '%s'", cfg.Account)
	}
	// APIURL should come from local
	if cfg.APIURL != "https://local.api.url" {
		t.Errorf("expected APIURL 'https://local.api.url', got '%s'", cfg.APIURL)
	}
	if cfg.Board != "local-board" {
		t.Errorf("expected Board 'local-board', got '%s'", cfg.Board)
	}
}

func TestLoad_LocalConfigInParentDirectory(t *testing.T) {
	// Clear environment variables
	os.Unsetenv("FIZZY_TOKEN")
	os.Unsetenv("FIZZY_ACCOUNT")
	os.Unsetenv("FIZZY_API_URL")

	// Setup temp directories
	origHome := os.Getenv("HOME")
	homeDir := t.TempDir()
	os.Setenv("HOME", homeDir)
	defer os.Setenv("HOME", origHome)

	// Create project structure: projectDir/subdir/deep
	projectDir := t.TempDir()
	deepDir := filepath.Join(projectDir, "subdir", "deep")
	os.MkdirAll(deepDir, 0755)

	// Create local config in project root (not in deepDir)
	localContent := `account: parent-account
api_url: https://parent.api.url
`
	os.WriteFile(filepath.Join(projectDir, LocalConfigFile), []byte(localContent), 0600)

	// Set working directory to deep subdirectory
	SetTestWorkingDir(deepDir)
	defer ResetTestWorkingDir()

	cfg := Load()

	// Should find config in parent directory
	if cfg.Account != "parent-account" {
		t.Errorf("expected Account 'parent-account', got '%s'", cfg.Account)
	}
	if cfg.APIURL != "https://parent.api.url" {
		t.Errorf("expected APIURL 'https://parent.api.url', got '%s'", cfg.APIURL)
	}
}

func TestLoad_EnvOverridesLocalConfig(t *testing.T) {
	// Setup temp directories
	origHome := os.Getenv("HOME")
	homeDir := t.TempDir()
	projectDir := t.TempDir()
	os.Setenv("HOME", homeDir)
	defer os.Setenv("HOME", origHome)

	// Create local config
	localContent := `token: local-token
account: local-account
`
	os.WriteFile(filepath.Join(projectDir, LocalConfigFile), []byte(localContent), 0600)

	// Set working directory
	SetTestWorkingDir(projectDir)
	defer ResetTestWorkingDir()

	// Set environment variable (should override local)
	os.Setenv("FIZZY_TOKEN", "env-token-override")
	defer os.Unsetenv("FIZZY_TOKEN")

	cfg := Load()

	// Token should come from env
	if cfg.Token != "env-token-override" {
		t.Errorf("expected Token 'env-token-override', got '%s'", cfg.Token)
	}
	// Account should come from local config
	if cfg.Account != "local-account" {
		t.Errorf("expected Account 'local-account', got '%s'", cfg.Account)
	}
}

func TestLoad_LocalConfigEmptyValuesDoNotOverride(t *testing.T) {
	// Clear environment variables
	os.Unsetenv("FIZZY_TOKEN")
	os.Unsetenv("FIZZY_ACCOUNT")
	os.Unsetenv("FIZZY_API_URL")

	// Setup temp directories
	origHome := os.Getenv("HOME")
	homeDir := t.TempDir()
	projectDir := t.TempDir()
	os.Setenv("HOME", homeDir)
	defer os.Setenv("HOME", origHome)

	// Create global config with all values
	globalConfigDir := filepath.Join(homeDir, ".fizzy")
	os.MkdirAll(globalConfigDir, 0700)
	globalContent := `token: global-token
account: global-account
api_url: https://global.api.url
`
	os.WriteFile(filepath.Join(globalConfigDir, "config.yaml"), []byte(globalContent), 0600)

	// Create local config with only account (token and api_url empty)
	localContent := `account: local-account
`
	os.WriteFile(filepath.Join(projectDir, LocalConfigFile), []byte(localContent), 0600)

	// Set working directory
	SetTestWorkingDir(projectDir)
	defer ResetTestWorkingDir()

	cfg := Load()

	// Token should remain from global (local value is empty)
	if cfg.Token != "global-token" {
		t.Errorf("expected Token 'global-token', got '%s'", cfg.Token)
	}
	// Account should come from local
	if cfg.Account != "local-account" {
		t.Errorf("expected Account 'local-account', got '%s'", cfg.Account)
	}
	// APIURL should remain from global (local value is empty)
	if cfg.APIURL != "https://global.api.url" {
		t.Errorf("expected APIURL 'https://global.api.url', got '%s'", cfg.APIURL)
	}
}

func TestLoad_NoLocalConfig(t *testing.T) {
	// Clear environment variables
	os.Unsetenv("FIZZY_TOKEN")
	os.Unsetenv("FIZZY_ACCOUNT")
	os.Unsetenv("FIZZY_API_URL")

	// Setup temp directories
	origHome := os.Getenv("HOME")
	homeDir := t.TempDir()
	projectDir := t.TempDir()
	os.Setenv("HOME", homeDir)
	defer os.Setenv("HOME", origHome)

	// Create global config only
	globalConfigDir := filepath.Join(homeDir, ".fizzy")
	os.MkdirAll(globalConfigDir, 0700)
	globalContent := `token: global-token
account: global-account
`
	os.WriteFile(filepath.Join(globalConfigDir, "config.yaml"), []byte(globalContent), 0600)

	// Set working directory (no local config exists)
	SetTestWorkingDir(projectDir)
	defer ResetTestWorkingDir()

	cfg := Load()

	// Should use global values
	if cfg.Token != "global-token" {
		t.Errorf("expected Token 'global-token', got '%s'", cfg.Token)
	}
	if cfg.Account != "global-account" {
		t.Errorf("expected Account 'global-account', got '%s'", cfg.Account)
	}
}

func TestLocalConfigPath(t *testing.T) {
	projectDir := t.TempDir()

	// No local config initially
	SetTestWorkingDir(projectDir)
	defer ResetTestWorkingDir()

	path := LocalConfigPath()
	if path != "" {
		t.Errorf("expected empty path when no local config, got '%s'", path)
	}

	// Create local config
	os.WriteFile(filepath.Join(projectDir, LocalConfigFile), []byte("account: test"), 0600)

	path = LocalConfigPath()
	expected := filepath.Join(projectDir, LocalConfigFile)
	if path != expected {
		t.Errorf("expected path '%s', got '%s'", expected, path)
	}
}

func TestLocalConfigPath_FindsInParent(t *testing.T) {
	// Create project structure
	projectDir := t.TempDir()
	deepDir := filepath.Join(projectDir, "src", "components")
	os.MkdirAll(deepDir, 0755)

	// Create local config in project root
	os.WriteFile(filepath.Join(projectDir, LocalConfigFile), []byte("account: project"), 0600)

	// Set working directory to deep subdirectory
	SetTestWorkingDir(deepDir)
	defer ResetTestWorkingDir()

	path := LocalConfigPath()
	expected := filepath.Join(projectDir, LocalConfigFile)
	if path != expected {
		t.Errorf("expected path '%s', got '%s'", expected, path)
	}
}

func TestSetTestWorkingDir(t *testing.T) {
	projectDir := t.TempDir()

	// Create local config
	os.WriteFile(filepath.Join(projectDir, LocalConfigFile), []byte("account: test"), 0600)

	// Before setting test dir, LocalConfigPath uses real working dir
	ResetTestWorkingDir()

	// After setting test dir, it should use that
	SetTestWorkingDir(projectDir)
	defer ResetTestWorkingDir()

	path := LocalConfigPath()
	if path != filepath.Join(projectDir, LocalConfigFile) {
		t.Errorf("SetTestWorkingDir did not work, got path '%s'", path)
	}
}

func TestSetTestConfigDir(t *testing.T) {
	configDir := t.TempDir()

	SetTestConfigDir(configDir)
	defer ResetTestConfigDir()

	paths := globalConfigPaths()
	if len(paths) != 1 {
		t.Fatalf("expected 1 path with test config dir, got %d", len(paths))
	}

	expected := filepath.Join(configDir, "config.yaml")
	if paths[0] != expected {
		t.Errorf("expected path '%s', got '%s'", expected, paths[0])
	}
}

func TestHTTPWarning(t *testing.T) {
	t.Run("warns on non-localhost HTTP URL", func(t *testing.T) {
		// Capture stderr
		origStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w
		defer func() { os.Stderr = origStderr }()

		cfg := &Config{APIURL: "http://example.com/api"}
		ensureAPIURL(cfg)

		w.Close()
		var buf bytes.Buffer
		buf.ReadFrom(r)
		if !strings.Contains(buf.String(), "Warning") {
			t.Error("expected HTTPS warning for http://example.com")
		}
	})

	t.Run("no warning for localhost", func(t *testing.T) {
		origStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w
		defer func() { os.Stderr = origStderr }()

		cfg := &Config{APIURL: "http://localhost:3000"}
		ensureAPIURL(cfg)

		w.Close()
		var buf bytes.Buffer
		buf.ReadFrom(r)
		if strings.Contains(buf.String(), "Warning") {
			t.Error("should not warn for localhost")
		}
	})

	t.Run("no warning for 127.0.0.1", func(t *testing.T) {
		origStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w
		defer func() { os.Stderr = origStderr }()

		cfg := &Config{APIURL: "http://127.0.0.1:3000"}
		ensureAPIURL(cfg)

		w.Close()
		var buf bytes.Buffer
		buf.ReadFrom(r)
		if strings.Contains(buf.String(), "Warning") {
			t.Error("should not warn for 127.0.0.1")
		}
	})

	t.Run("no warning for IPv6 loopback with port", func(t *testing.T) {
		origStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w
		defer func() { os.Stderr = origStderr }()

		cfg := &Config{APIURL: "http://[::1]:3000"}
		ensureAPIURL(cfg)

		w.Close()
		var buf bytes.Buffer
		buf.ReadFrom(r)
		if strings.Contains(buf.String(), "Warning") {
			t.Error("should not warn for [::1]:3000")
		}
	})

	t.Run("no warning for HTTPS", func(t *testing.T) {
		origStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w
		defer func() { os.Stderr = origStderr }()

		cfg := &Config{APIURL: "https://example.com"}
		ensureAPIURL(cfg)

		w.Close()
		var buf bytes.Buffer
		buf.ReadFrom(r)
		if strings.Contains(buf.String(), "Warning") {
			t.Error("should not warn for HTTPS")
		}
	})
}

func TestFullConfigPriorityChain(t *testing.T) {
	// This test validates the full priority chain:
	// CLI flags > env vars > local config > global config > defaults
	// (We can't test CLI flags here, but we test the rest)

	// Setup temp directories
	origHome := os.Getenv("HOME")
	homeDir := t.TempDir()
	projectDir := t.TempDir()
	os.Setenv("HOME", homeDir)
	defer os.Setenv("HOME", origHome)

	// Create global config
	globalConfigDir := filepath.Join(homeDir, ".fizzy")
	os.MkdirAll(globalConfigDir, 0700)
	globalContent := `token: global-token
account: global-account
api_url: https://global.api.url
`
	os.WriteFile(filepath.Join(globalConfigDir, "config.yaml"), []byte(globalContent), 0600)

	// Create local config (overrides some values)
	localContent := `account: local-account
`
	os.WriteFile(filepath.Join(projectDir, LocalConfigFile), []byte(localContent), 0600)

	// Set env var (overrides local and global)
	os.Setenv("FIZZY_API_URL", "https://env.api.url")
	defer os.Unsetenv("FIZZY_API_URL")

	// Clear other env vars
	os.Unsetenv("FIZZY_TOKEN")
	os.Unsetenv("FIZZY_ACCOUNT")

	SetTestWorkingDir(projectDir)
	defer ResetTestWorkingDir()

	cfg := Load()

	// Token: from global (no local, no env)
	if cfg.Token != "global-token" {
		t.Errorf("expected Token 'global-token' (from global), got '%s'", cfg.Token)
	}
	// Account: from local (overrides global)
	if cfg.Account != "local-account" {
		t.Errorf("expected Account 'local-account' (from local), got '%s'", cfg.Account)
	}
	// APIURL: from env (overrides local and global)
	if cfg.APIURL != "https://env.api.url" {
		t.Errorf("expected APIURL 'https://env.api.url' (from env), got '%s'", cfg.APIURL)
	}
}
