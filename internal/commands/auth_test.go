package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/basecamp/fizzy-cli/internal/config"
	"gopkg.in/yaml.v3"
)

func TestAuthLogin(t *testing.T) {
	t.Run("saves token to config file", func(t *testing.T) {
		// Create temp directory for config
		tempDir, err := os.MkdirTemp("", "fizzy-test-*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tempDir)

		config.SetTestConfigDir(tempDir)
		defer config.ResetTestConfigDir()

		mock := NewMockClient()
		result := SetTestMode(mock)
		defer ResetTestMode()

		err = authLoginCmd.RunE(authLoginCmd, []string{"test-token-123"})
		assertExitCode(t, err, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Response.OK {
			t.Error("expected success response")
		}

		// Verify config file was created with correct token
		configPath := filepath.Join(tempDir, "config.yaml")
		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("config file not created: %v", err)
		}

		var savedConfig config.Config
		if err := yaml.Unmarshal(data, &savedConfig); err != nil {
			t.Fatalf("failed to parse config: %v", err)
		}

		if savedConfig.Token != "test-token-123" {
			t.Errorf("expected token 'test-token-123', got '%s'", savedConfig.Token)
		}
	})

	t.Run("preserves existing config values", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "fizzy-test-*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tempDir)

		config.SetTestConfigDir(tempDir)
		defer config.ResetTestConfigDir()

		// Create existing config with account
		existingConfig := &config.Config{
			Account: "existing-account",
			APIURL:  "https://custom.api.com",
		}
		existingData, _ := yaml.Marshal(existingConfig)
		os.WriteFile(filepath.Join(tempDir, "config.yaml"), existingData, 0600)

		mock := NewMockClient()
		SetTestMode(mock)
		defer ResetTestMode()

		err = authLoginCmd.RunE(authLoginCmd, []string{"new-token"})
		assertExitCode(t, err, 0)

		// Verify existing values preserved
		data, _ := os.ReadFile(filepath.Join(tempDir, "config.yaml"))
		var savedConfig config.Config
		yaml.Unmarshal(data, &savedConfig)

		if savedConfig.Token != "new-token" {
			t.Errorf("expected token 'new-token', got '%s'", savedConfig.Token)
		}
		if savedConfig.Account != "existing-account" {
			t.Errorf("expected account 'existing-account', got '%s'", savedConfig.Account)
		}
	})
}

func TestAuthLogout(t *testing.T) {
	t.Run("removes config file", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "fizzy-test-*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tempDir)

		config.SetTestConfigDir(tempDir)
		defer config.ResetTestConfigDir()

		// Create config file
		configPath := filepath.Join(tempDir, "config.yaml")
		os.WriteFile(configPath, []byte("token: test-token"), 0600)

		mock := NewMockClient()
		SetTestMode(mock)
		defer ResetTestMode()

		err = authLogoutCmd.RunE(authLogoutCmd, []string{})
		assertExitCode(t, err, 0)

		// Verify config file was removed
		if _, err := os.Stat(configPath); !os.IsNotExist(err) {
			t.Error("expected config file to be removed")
		}
	})

	t.Run("succeeds even if no config file exists", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "fizzy-test-*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tempDir)

		config.SetTestConfigDir(tempDir)
		defer config.ResetTestConfigDir()

		mock := NewMockClient()
		SetTestMode(mock)
		defer ResetTestMode()

		err = authLogoutCmd.RunE(authLogoutCmd, []string{})
		assertExitCode(t, err, 0)
	})
}

func TestAuthStatus(t *testing.T) {
	t.Run("shows authenticated status when token exists", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "fizzy-test-*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tempDir)

		config.SetTestConfigDir(tempDir)
		defer config.ResetTestConfigDir()

		// Create config with token
		configData := "token: test-token\naccount: test-account"
		os.WriteFile(filepath.Join(tempDir, "config.yaml"), []byte(configData), 0600)

		mock := NewMockClient()
		result := SetTestMode(mock)
		defer ResetTestMode()

		err = authStatusCmd.RunE(authStatusCmd, []string{})
		assertExitCode(t, err, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Response.OK {
			t.Error("expected success response")
		}

		// Check response data
		data, ok := result.Response.Data.(map[string]any)
		if !ok {
			t.Fatal("expected map response data")
		}
		if data["authenticated"] != true {
			t.Errorf("expected authenticated=true, got %v", data["authenticated"])
		}
		if data["token_configured"] != true {
			t.Errorf("expected token_configured=true, got %v", data["token_configured"])
		}
		if data["account"] != "test-account" {
			t.Errorf("expected account='test-account', got %v", data["account"])
		}
	})

	t.Run("shows unauthenticated status when no token", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "fizzy-test-*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tempDir)

		config.SetTestConfigDir(tempDir)
		defer config.ResetTestConfigDir()

		mock := NewMockClient()
		result := SetTestMode(mock)
		defer ResetTestMode()

		err = authStatusCmd.RunE(authStatusCmd, []string{})
		assertExitCode(t, err, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, ok := result.Response.Data.(map[string]any)
		if !ok {
			t.Fatal("expected map response data")
		}
		if data["authenticated"] != false {
			t.Errorf("expected authenticated=false, got %v", data["authenticated"])
		}
	})

	t.Run("shows custom api_url when configured", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "fizzy-test-*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tempDir)

		config.SetTestConfigDir(tempDir)
		defer config.ResetTestConfigDir()

		// Create config with custom API URL
		configData := "token: test-token\napi_url: https://custom.fizzy.do"
		os.WriteFile(filepath.Join(tempDir, "config.yaml"), []byte(configData), 0600)

		mock := NewMockClient()
		result := SetTestMode(mock)
		defer ResetTestMode()

		err = authStatusCmd.RunE(authStatusCmd, []string{})
		assertExitCode(t, err, 0)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data := result.Response.Data.(map[string]any)
		if data["api_url"] != "https://custom.fizzy.do" {
			t.Errorf("expected api_url='https://custom.fizzy.do', got %v", data["api_url"])
		}
	})
}
