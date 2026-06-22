package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/basecamp/cli/credstore"
	"github.com/basecamp/cli/output"
	"github.com/basecamp/cli/profile"
	"github.com/basecamp/fizzy-cli/internal/config"
	"gopkg.in/yaml.v3"
)

func TestAuthLogin(t *testing.T) {
	t.Run("saves token to config file", func(t *testing.T) {
		tempDir := t.TempDir()
		config.SetTestConfigDir(tempDir)
		defer config.ResetTestConfigDir()

		mock := NewMockClient()
		result := SetTestModeWithSDK(mock)
		SetTestConfig("", "test-account", "https://app.fizzy.do")
		defer resetTest()

		err := authLoginCmd.RunE(authLoginCmd, []string{"test-token-123"})
		assertExitCode(t, err, 0)

		if !result.Response.OK {
			t.Error("expected success response")
		}

		// Verify config file was created with correct account
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

	t.Run("saves token to credstore under profile-scoped key", func(t *testing.T) {
		tempDir := t.TempDir()
		configDir := t.TempDir()
		profileDir := t.TempDir()

		config.SetTestConfigDir(configDir)
		defer config.ResetTestConfigDir()

		os.Setenv("FIZZY_TEST_NO_KR", "1")
		defer os.Unsetenv("FIZZY_TEST_NO_KR")
		store := credstore.NewStore(credstore.StoreOptions{
			ServiceName:   "fizzy-test",
			DisableEnvVar: "FIZZY_TEST_NO_KR",
			FallbackDir:   tempDir,
		})
		profileStore := profile.NewStore(filepath.Join(profileDir, "config.json"))

		mock := NewMockClient()
		result := SetTestModeWithSDK(mock)
		SetTestCreds(store)
		SetTestProfiles(profileStore)
		SetTestConfig("", "acme", "https://app.fizzy.do")
		defer resetTest()

		err := authLoginCmd.RunE(authLoginCmd, []string{"cred-token-456"})
		assertExitCode(t, err, 0)

		if !result.Response.OK {
			t.Error("expected success response")
		}

		// Token should be stored under "profile:acme"
		loaded, err := store.Load("profile:acme")
		if err != nil {
			t.Fatalf("expected token in credstore under 'profile:acme': %v", err)
		}
		var tokenStr string
		if err := json.Unmarshal(loaded, &tokenStr); err != nil {
			t.Fatalf("expected JSON-encoded token, got %q: %v", string(loaded), err)
		}
		if tokenStr != "cred-token-456" {
			t.Errorf("expected 'cred-token-456', got '%s'", tokenStr)
		}

		// Token should NOT be in YAML config
		configPath := filepath.Join(configDir, "config.yaml")
		if data, err := os.ReadFile(configPath); err == nil {
			var savedConfig config.Config
			yaml.Unmarshal(data, &savedConfig)
			if savedConfig.Token != "" {
				t.Errorf("expected empty token in YAML, got '%s'", savedConfig.Token)
			}
		}

		// Profile should exist in profile store
		p, err := profileStore.Get("acme")
		if err != nil {
			t.Fatalf("expected profile 'acme' in store: %v", err)
		}
		if p.BaseURL != "https://app.fizzy.do" {
			t.Errorf("expected base_url 'https://app.fizzy.do', got '%s'", p.BaseURL)
		}
	})

	t.Run("requires profile to be configured", func(t *testing.T) {
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("", "", "https://app.fizzy.do")
		defer resetTest()

		err := authLoginCmd.RunE(authLoginCmd, []string{"some-token"})
		if err == nil {
			t.Error("expected error when no profile configured")
		}
	})

	t.Run("preserves existing config values", func(t *testing.T) {
		tempDir := t.TempDir()
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
		SetTestModeWithSDK(mock)
		SetTestConfig("", "existing-account", "https://custom.api.com")
		defer resetTest()

		err := authLoginCmd.RunE(authLoginCmd, []string{"new-token"})
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
	t.Run("removes profile-scoped token from credstore", func(t *testing.T) {
		tempDir := t.TempDir()
		credDir := t.TempDir()
		profileDir := t.TempDir()

		config.SetTestConfigDir(tempDir)
		defer config.ResetTestConfigDir()

		os.Setenv("FIZZY_LOGOUT_NO_KR", "1")
		defer os.Unsetenv("FIZZY_LOGOUT_NO_KR")
		store := credstore.NewStore(credstore.StoreOptions{
			ServiceName:   "fizzy-logout-test",
			DisableEnvVar: "FIZZY_LOGOUT_NO_KR",
			FallbackDir:   credDir,
		})
		profileStore := profile.NewStore(filepath.Join(profileDir, "config.json"))
		profileStore.Create(&profile.Profile{Name: "acme", BaseURL: "https://app.fizzy.do"})

		// Save a token under profile-scoped key
		tokenData, _ := json.Marshal("my-token")
		store.Save("profile:acme", tokenData)

		// Save config
		cfg := &config.Config{Account: "acme"}
		cfgData, _ := yaml.Marshal(cfg)
		os.WriteFile(filepath.Join(tempDir, "config.yaml"), cfgData, 0600)

		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestCreds(store)
		SetTestProfiles(profileStore)
		SetTestConfig("my-token", "acme", "https://app.fizzy.do")
		defer resetTest()

		// Reset the --all flag
		authLogoutCmd.Flags().Set("all", "false")
		err := authLogoutCmd.RunE(authLogoutCmd, []string{})
		assertExitCode(t, err, 0)

		// Verify token removed from credstore
		if _, err := store.Load("profile:acme"); err == nil {
			t.Error("expected token to be removed from credstore")
		}

		// Verify profile removed from store
		if _, err := profileStore.Get("acme"); err == nil {
			t.Error("expected profile to be removed from store")
		}
	})

	t.Run("preserves legacy token key for downgrade compatibility", func(t *testing.T) {
		tempDir := t.TempDir()
		credDir := t.TempDir()
		profileDir := t.TempDir()

		config.SetTestConfigDir(tempDir)
		defer config.ResetTestConfigDir()

		os.Setenv("FIZZY_LOGOUT_LEGACY_NO_KR", "1")
		defer os.Unsetenv("FIZZY_LOGOUT_LEGACY_NO_KR")
		store := credstore.NewStore(credstore.StoreOptions{
			ServiceName:   "fizzy-logout-legacy-test",
			DisableEnvVar: "FIZZY_LOGOUT_LEGACY_NO_KR",
			FallbackDir:   credDir,
		})
		profileStore := profile.NewStore(filepath.Join(profileDir, "config.json"))
		profileStore.Create(&profile.Profile{Name: "acme", BaseURL: "https://app.fizzy.do"})

		// Simulate a migrated state: both legacy and profile-scoped keys exist
		tokenData, _ := json.Marshal("my-token")
		store.Save("token", tokenData)
		store.Save("profile:acme", tokenData)

		cfg := &config.Config{Account: "acme"}
		cfgData, _ := yaml.Marshal(cfg)
		os.WriteFile(filepath.Join(tempDir, "config.yaml"), cfgData, 0600)

		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestCreds(store)
		SetTestProfiles(profileStore)
		SetTestConfig("my-token", "acme", "https://app.fizzy.do")
		defer resetTest()

		authLogoutCmd.Flags().Set("all", "false")
		err := authLogoutCmd.RunE(authLogoutCmd, []string{})
		assertExitCode(t, err, 0)

		// Profile-scoped key should be removed
		if _, err := store.Load("profile:acme"); err == nil {
			t.Error("expected profile-scoped token to be removed")
		}

		// Legacy key should be preserved
		if _, err := store.Load("token"); err != nil {
			t.Error("expected legacy 'token' key to be preserved for downgrade compatibility")
		}
	})

	t.Run("logout --all removes all profiles", func(t *testing.T) {
		tempDir := t.TempDir()
		credDir := t.TempDir()
		profileDir := t.TempDir()

		config.SetTestConfigDir(tempDir)
		defer config.ResetTestConfigDir()

		os.Setenv("FIZZY_LOGOUTALL_NO_KR", "1")
		defer os.Unsetenv("FIZZY_LOGOUTALL_NO_KR")
		store := credstore.NewStore(credstore.StoreOptions{
			ServiceName:   "fizzy-logoutall-test",
			DisableEnvVar: "FIZZY_LOGOUTALL_NO_KR",
			FallbackDir:   credDir,
		})
		profileStore := profile.NewStore(filepath.Join(profileDir, "config.json"))
		profileStore.Create(&profile.Profile{Name: "acme", BaseURL: "https://app.fizzy.do"})
		profileStore.Create(&profile.Profile{Name: "other", BaseURL: "https://app.fizzy.do"})

		// Save tokens for two profiles
		t1, _ := json.Marshal("token1")
		t2, _ := json.Marshal("token2")
		store.Save("profile:acme", t1)
		store.Save("profile:other", t2)

		// Config
		cfg := &config.Config{Account: "acme"}
		cfgData, _ := yaml.Marshal(cfg)
		os.WriteFile(filepath.Join(tempDir, "config.yaml"), cfgData, 0600)

		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestCreds(store)
		SetTestProfiles(profileStore)
		SetTestConfig("token1", "acme", "https://app.fizzy.do")
		defer resetTest()

		authLogoutCmd.Flags().Set("all", "true")
		err := authLogoutCmd.RunE(authLogoutCmd, []string{})
		assertExitCode(t, err, 0)

		// Both tokens should be gone
		if _, err := store.Load("profile:acme"); err == nil {
			t.Error("expected acme token removed")
		}
		if _, err := store.Load("profile:other"); err == nil {
			t.Error("expected other token removed")
		}
	})

	t.Run("succeeds even if no config file exists", func(t *testing.T) {
		tempDir := t.TempDir()
		config.SetTestConfigDir(tempDir)
		defer config.ResetTestConfigDir()

		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestConfig("", "some-profile", "https://app.fizzy.do")
		defer resetTest()

		authLogoutCmd.Flags().Set("all", "false")
		err := authLogoutCmd.RunE(authLogoutCmd, []string{})
		assertExitCode(t, err, 0)
	})
}

func TestAuthStatus(t *testing.T) {
	t.Run("shows authenticated status when token exists", func(t *testing.T) {
		tempDir := t.TempDir()
		config.SetTestConfigDir(tempDir)
		defer config.ResetTestConfigDir()

		// Create config with token
		configData := "token: test-token\naccount: test-account"
		os.WriteFile(filepath.Join(tempDir, "config.yaml"), []byte(configData), 0600)

		mock := NewMockClient()
		result := SetTestModeWithSDK(mock)
		defer resetTest()

		err := authStatusCmd.RunE(authStatusCmd, []string{})
		assertExitCode(t, err, 0)

		if !result.Response.OK {
			t.Error("expected success response")
		}

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
		if data["profile"] != "test-account" {
			t.Errorf("expected profile='test-account', got %v", data["profile"])
		}
	})

	t.Run("shows unauthenticated status when no token", func(t *testing.T) {
		tempDir := t.TempDir()
		config.SetTestConfigDir(tempDir)
		defer config.ResetTestConfigDir()

		mock := NewMockClient()
		result := SetTestModeWithSDK(mock)
		defer resetTest()

		err := authStatusCmd.RunE(authStatusCmd, []string{})
		assertExitCode(t, err, 0)

		data, ok := result.Response.Data.(map[string]any)
		if !ok {
			t.Fatal("expected map response data")
		}
		if data["authenticated"] != false {
			t.Errorf("expected authenticated=false, got %v", data["authenticated"])
		}
	})

	t.Run("shows using_keyring field when credstore is set", func(t *testing.T) {
		mock := NewMockClient()
		result := SetTestModeWithSDK(mock)
		SetTestConfig("token", "account", "https://api.example.com")

		// Create a file-based credstore (env var disables keyring probe)
		tempDir := t.TempDir()
		os.Setenv("FIZZY_TEST_NO_KEYRING_ALWAYS", "1")
		defer os.Unsetenv("FIZZY_TEST_NO_KEYRING_ALWAYS")
		store := credstore.NewStore(credstore.StoreOptions{
			ServiceName:   "fizzy-test",
			DisableEnvVar: "FIZZY_TEST_NO_KEYRING_ALWAYS",
			FallbackDir:   tempDir,
		})
		SetTestCreds(store)
		defer resetTest()

		err := authStatusCmd.RunE(authStatusCmd, []string{})
		assertExitCode(t, err, 0)

		data, ok := result.Response.Data.(map[string]any)
		if !ok {
			t.Fatal("expected map response data")
		}
		if _, ok := data["using_keyring"]; !ok {
			t.Error("expected using_keyring field when credstore is set")
		}
	})

	t.Run("shows custom api_url when configured", func(t *testing.T) {
		tempDir := t.TempDir()
		config.SetTestConfigDir(tempDir)
		defer config.ResetTestConfigDir()

		// Create config with custom API URL
		configData := "token: test-token\napi_url: https://custom.fizzy.do"
		os.WriteFile(filepath.Join(tempDir, "config.yaml"), []byte(configData), 0600)

		mock := NewMockClient()
		result := SetTestModeWithSDK(mock)
		defer resetTest()

		err := authStatusCmd.RunE(authStatusCmd, []string{})
		assertExitCode(t, err, 0)

		data := result.Response.Data.(map[string]any)
		if data["api_url"] != "https://custom.fizzy.do" {
			t.Errorf("expected api_url='https://custom.fizzy.do', got %v", data["api_url"])
		}
	})
}

func TestAuthList(t *testing.T) {
	t.Run("lists authenticated profiles", func(t *testing.T) {
		credDir := t.TempDir()
		profileDir := t.TempDir()

		os.Setenv("FIZZY_LIST_NO_KR", "1")
		defer os.Unsetenv("FIZZY_LIST_NO_KR")
		store := credstore.NewStore(credstore.StoreOptions{
			ServiceName:   "fizzy-list-test",
			DisableEnvVar: "FIZZY_LIST_NO_KR",
			FallbackDir:   credDir,
		})
		profileStore := profile.NewStore(filepath.Join(profileDir, "config.json"))
		profileStore.Create(&profile.Profile{Name: "acme", BaseURL: "https://app.fizzy.do"})
		profileStore.Create(&profile.Profile{Name: "other", BaseURL: "https://staging.fizzy.do"})

		// Save tokens for two profiles
		t1, _ := json.Marshal("token1")
		t2, _ := json.Marshal("token2")
		store.Save("profile:acme", t1)
		store.Save("profile:other", t2)

		mock := NewMockClient()
		result := SetTestModeWithSDK(mock)
		SetTestCreds(store)
		SetTestProfiles(profileStore)
		defer resetTest()

		err := authListCmd.RunE(authListCmd, []string{})
		assertExitCode(t, err, 0)

		profiles, ok := result.Response.Data.([]any)
		if !ok {
			t.Fatal("expected array response data")
		}
		if len(profiles) != 2 {
			t.Fatalf("expected 2 profiles, got %d", len(profiles))
		}

		// Find the active profile (acme is default since it was created first)
		var activeFound bool
		for _, p := range profiles {
			entry := p.(map[string]any)
			if entry["active"] == true {
				activeFound = true
				if entry["profile"] != "acme" {
					t.Errorf("expected active profile 'acme', got %v", entry["profile"])
				}
			}
			if entry["has_token"] != true {
				t.Errorf("expected has_token=true for profile %v", entry["profile"])
			}
		}
		if !activeFound {
			t.Error("expected one active profile")
		}
	})

	t.Run("shows empty list when no profiles", func(t *testing.T) {
		mock := NewMockClient()
		result := SetTestModeWithSDK(mock)
		defer resetTest()

		err := authListCmd.RunE(authListCmd, []string{})
		assertExitCode(t, err, 0)

		profiles, ok := result.Response.Data.([]any)
		if !ok {
			t.Fatal("expected array response data")
		}
		if len(profiles) != 0 {
			t.Errorf("expected 0 profiles, got %d", len(profiles))
		}
	})

	t.Run("renders styled output with next steps", func(t *testing.T) {
		credDir := t.TempDir()
		profileDir := t.TempDir()

		os.Setenv("FIZZY_LIST_STYLED_NO_KR", "1")
		defer os.Unsetenv("FIZZY_LIST_STYLED_NO_KR")
		store := credstore.NewStore(credstore.StoreOptions{
			ServiceName:   "fizzy-list-styled-test",
			DisableEnvVar: "FIZZY_LIST_STYLED_NO_KR",
			FallbackDir:   credDir,
		})
		profileStore := profile.NewStore(filepath.Join(profileDir, "config.json"))
		profileStore.Create(&profile.Profile{Name: "acme", BaseURL: "https://app.fizzy.do"})
		t1, _ := json.Marshal("token1")
		store.Save("profile:acme", t1)

		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestCreds(store)
		SetTestProfiles(profileStore)
		SetTestFormat(output.FormatStyled)
		defer resetTest()

		err := authListCmd.RunE(authListCmd, []string{})
		assertExitCode(t, err, 0)

		raw := TestOutput()
		if !strings.Contains(raw, "Profile") {
			t.Fatalf("expected styled table output, got:\n%s", raw)
		}
		if !strings.Contains(raw, "Next steps:") {
			t.Fatalf("expected next steps section, got:\n%s", raw)
		}
	})
}

func TestAuthSwitch(t *testing.T) {
	t.Run("switches active profile", func(t *testing.T) {
		tempDir := t.TempDir()
		credDir := t.TempDir()
		profileDir := t.TempDir()

		config.SetTestConfigDir(tempDir)
		defer config.ResetTestConfigDir()

		os.Setenv("FIZZY_SWITCH_NO_KR", "1")
		defer os.Unsetenv("FIZZY_SWITCH_NO_KR")
		store := credstore.NewStore(credstore.StoreOptions{
			ServiceName:   "fizzy-switch-test",
			DisableEnvVar: "FIZZY_SWITCH_NO_KR",
			FallbackDir:   credDir,
		})
		profileStore := profile.NewStore(filepath.Join(profileDir, "config.json"))
		profileStore.Create(&profile.Profile{Name: "acme", BaseURL: "https://app.fizzy.do"})
		profileStore.Create(&profile.Profile{Name: "other", BaseURL: "https://staging.fizzy.do"})

		// Save token for the target profile
		tokenData, _ := json.Marshal("other-token")
		store.Save("profile:other", tokenData)

		cfg := &config.Config{Account: "acme"}
		cfgData, _ := yaml.Marshal(cfg)
		os.WriteFile(filepath.Join(tempDir, "config.yaml"), cfgData, 0600)

		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestCreds(store)
		SetTestProfiles(profileStore)
		SetTestConfig("acme-token", "acme", "https://app.fizzy.do")
		defer resetTest()

		err := authSwitchCmd.RunE(authSwitchCmd, []string{"other"})
		assertExitCode(t, err, 0)

		// Verify config was updated
		data, _ := os.ReadFile(filepath.Join(tempDir, "config.yaml"))
		var savedConfig config.Config
		yaml.Unmarshal(data, &savedConfig)

		if savedConfig.Account != "other" {
			t.Errorf("expected account 'other', got '%s'", savedConfig.Account)
		}
		if savedConfig.Board != "" {
			t.Errorf("expected board cleared on switch, got '%s'", savedConfig.Board)
		}

		// Verify profile store default was updated
		_, defaultName, _ := profileStore.List()
		if defaultName != "other" {
			t.Errorf("expected default profile 'other', got '%s'", defaultName)
		}
	})

	t.Run("fails for unknown profile", func(t *testing.T) {
		credDir := t.TempDir()

		os.Setenv("FIZZY_SWITCH2_NO_KR", "1")
		defer os.Unsetenv("FIZZY_SWITCH2_NO_KR")
		store := credstore.NewStore(credstore.StoreOptions{
			ServiceName:   "fizzy-switch2-test",
			DisableEnvVar: "FIZZY_SWITCH2_NO_KR",
			FallbackDir:   credDir,
		})

		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestCreds(store)
		SetTestConfig("", "acme", "https://app.fizzy.do")
		defer resetTest()

		err := authSwitchCmd.RunE(authSwitchCmd, []string{"nonexistent"})
		if err == nil {
			t.Error("expected error for unknown profile")
		}
	})
}

func TestProfileFlagTokenSelection(t *testing.T) {
	t.Run("resolveToken loads token for profile specified via flag", func(t *testing.T) {
		credDir := t.TempDir()
		profileDir := t.TempDir()

		os.Setenv("FIZZY_FLAGSEL_NO_KR", "1")
		defer os.Unsetenv("FIZZY_FLAGSEL_NO_KR")
		store := credstore.NewStore(credstore.StoreOptions{
			ServiceName:   "fizzy-flagsel-test",
			DisableEnvVar: "FIZZY_FLAGSEL_NO_KR",
			FallbackDir:   credDir,
		})
		profileStore := profile.NewStore(filepath.Join(profileDir, "config.json"))
		profileStore.Create(&profile.Profile{Name: "acme", BaseURL: "https://app.fizzy.do"})
		profileStore.Create(&profile.Profile{Name: "other", BaseURL: "https://app.fizzy.do"})

		// Save tokens for two profiles
		t1, _ := json.Marshal("acme-token")
		t2, _ := json.Marshal("other-token")
		store.Save("profile:acme", t1)
		store.Save("profile:other", t2)

		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestCreds(store)
		SetTestProfiles(profileStore)
		SetTestConfig("", "acme", "https://app.fizzy.do")
		defer resetTest()

		// Simulate --profile other flag: resolve profile first, then token
		cfgProfile = "other"
		defer func() { cfgProfile = "" }()

		if err := resolveProfile(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		resolveToken()

		if cfg.Token != "other-token" {
			t.Errorf("expected 'other-token' for --profile other, got '%s'", cfg.Token)
		}
		if cfg.Account != "other" {
			t.Errorf("expected account 'other' from profile resolution, got '%s'", cfg.Account)
		}
	})

	t.Run("resolveToken uses default profile when no flag", func(t *testing.T) {
		credDir := t.TempDir()
		profileDir := t.TempDir()

		os.Setenv("FIZZY_FLAGSEL2_NO_KR", "1")
		defer os.Unsetenv("FIZZY_FLAGSEL2_NO_KR")
		store := credstore.NewStore(credstore.StoreOptions{
			ServiceName:   "fizzy-flagsel2-test",
			DisableEnvVar: "FIZZY_FLAGSEL2_NO_KR",
			FallbackDir:   credDir,
		})
		profileStore := profile.NewStore(filepath.Join(profileDir, "config.json"))
		profileStore.Create(&profile.Profile{Name: "acme", BaseURL: "https://app.fizzy.do"})
		profileStore.Create(&profile.Profile{Name: "other", BaseURL: "https://app.fizzy.do"})

		t1, _ := json.Marshal("acme-token")
		t2, _ := json.Marshal("other-token")
		store.Save("profile:acme", t1)
		store.Save("profile:other", t2)

		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestCreds(store)
		SetTestProfiles(profileStore)
		SetTestConfig("", "", "https://app.fizzy.do")
		defer resetTest()

		// No --profile flag, "acme" is default (first created)
		cfgProfile = ""

		if err := resolveProfile(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		resolveToken()

		if cfg.Token != "acme-token" {
			t.Errorf("expected 'acme-token' for default profile, got '%s'", cfg.Token)
		}
	})

	t.Run("invalid --profile flag returns error", func(t *testing.T) {
		profileDir := t.TempDir()
		profileStore := profile.NewStore(filepath.Join(profileDir, "config.json"))
		profileStore.Create(&profile.Profile{Name: "acme", BaseURL: "https://app.fizzy.do"})

		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestProfiles(profileStore)
		SetTestConfig("", "acme", "https://app.fizzy.do")
		defer resetTest()

		cfgProfile = "nonexistent"
		defer func() { cfgProfile = "" }()

		err := resolveProfile()
		if err == nil {
			t.Error("expected error for invalid --profile flag")
		}
	})

	t.Run("invalid FIZZY_PROFILE env var returns error", func(t *testing.T) {
		profileDir := t.TempDir()
		profileStore := profile.NewStore(filepath.Join(profileDir, "config.json"))
		profileStore.Create(&profile.Profile{Name: "acme", BaseURL: "https://app.fizzy.do"})

		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestProfiles(profileStore)
		SetTestConfig("", "", "https://app.fizzy.do")
		defer resetTest()

		os.Setenv("FIZZY_PROFILE", "nonexistent")
		defer os.Unsetenv("FIZZY_PROFILE")

		err := resolveProfile()
		if err == nil {
			t.Error("expected error for invalid FIZZY_PROFILE env var")
		}
	})
}

func TestTokenMigrationToProfile(t *testing.T) {
	t.Run("migrates legacy single-key token to profile-scoped key", func(t *testing.T) {
		configDir := t.TempDir()
		credDir := t.TempDir()
		profileDir := t.TempDir()

		config.SetTestConfigDir(configDir)
		defer config.ResetTestConfigDir()

		os.WriteFile(filepath.Join(configDir, "config.yaml"),
			[]byte("account: acme"), 0600)

		os.Setenv("FIZZY_MIGRATE_NO_KR", "1")
		defer os.Unsetenv("FIZZY_MIGRATE_NO_KR")
		store := credstore.NewStore(credstore.StoreOptions{
			ServiceName:   "fizzy-migrate-test",
			DisableEnvVar: "FIZZY_MIGRATE_NO_KR",
			FallbackDir:   credDir,
		})
		profileStore := profile.NewStore(filepath.Join(profileDir, "config.json"))

		// Save a legacy token under the old "token" key
		legacyToken, _ := json.Marshal("migrate-me")
		store.Save("token", legacyToken)

		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestCreds(store)
		SetTestProfiles(profileStore)
		SetTestConfig("", "acme", "https://app.fizzy.do")
		defer resetTest()

		resolveToken()

		// Token should be available via the profile-scoped key
		loaded, err := store.Load("profile:acme")
		if err != nil {
			t.Fatalf("expected token in credstore under 'profile:acme': %v", err)
		}
		var tokenStr string
		json.Unmarshal(loaded, &tokenStr)
		if tokenStr != "migrate-me" {
			t.Errorf("expected 'migrate-me', got '%s'", tokenStr)
		}

		// Legacy key should be preserved for downgrade compatibility
		if _, err := store.Load("token"); err != nil {
			t.Error("expected legacy 'token' key to be preserved after migration")
		}

		// cfg.Token should be set
		if cfg.Token != "migrate-me" {
			t.Errorf("expected cfg.Token='migrate-me', got '%s'", cfg.Token)
		}

		// Profile should be created in the store
		if _, err := profileStore.Get("acme"); err != nil {
			t.Error("expected profile 'acme' to be created during migration")
		}
	})

	t.Run("migrates account-scoped token to profile-scoped key", func(t *testing.T) {
		configDir := t.TempDir()
		credDir := t.TempDir()
		profileDir := t.TempDir()

		config.SetTestConfigDir(configDir)
		defer config.ResetTestConfigDir()

		os.WriteFile(filepath.Join(configDir, "config.yaml"),
			[]byte("account: acme"), 0600)

		os.Setenv("FIZZY_MIGRATE_ACCT_NO_KR", "1")
		defer os.Unsetenv("FIZZY_MIGRATE_ACCT_NO_KR")
		store := credstore.NewStore(credstore.StoreOptions{
			ServiceName:   "fizzy-migrate-acct-test",
			DisableEnvVar: "FIZZY_MIGRATE_ACCT_NO_KR",
			FallbackDir:   credDir,
		})
		profileStore := profile.NewStore(filepath.Join(profileDir, "config.json"))

		// Save a token under the old account-scoped key "token:acme"
		acctToken, _ := json.Marshal("acct-migrate-me")
		store.Save("token:acme", acctToken)

		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestCreds(store)
		SetTestProfiles(profileStore)
		SetTestConfig("", "acme", "https://app.fizzy.do")
		defer resetTest()

		resolveToken()

		// Token should now also be under profile-scoped key
		loaded, err := store.Load("profile:acme")
		if err != nil {
			t.Fatalf("expected token under 'profile:acme': %v", err)
		}
		var tokenStr string
		json.Unmarshal(loaded, &tokenStr)
		if tokenStr != "acct-migrate-me" {
			t.Errorf("expected 'acct-migrate-me', got '%s'", tokenStr)
		}

		// cfg.Token should be set
		if cfg.Token != "acct-migrate-me" {
			t.Errorf("expected cfg.Token='acct-migrate-me', got '%s'", cfg.Token)
		}
	})

	t.Run("migrates YAML token to profile-scoped credstore key", func(t *testing.T) {
		configDir := t.TempDir()
		credDir := t.TempDir()
		profileDir := t.TempDir()

		config.SetTestConfigDir(configDir)
		defer config.ResetTestConfigDir()

		// Write a global config with a token (pre-credstore state)
		os.WriteFile(filepath.Join(configDir, "config.yaml"),
			[]byte("token: migrate-me\naccount: acme"), 0600)

		os.Setenv("FIZZY_MIGRATE2_NO_KR", "1")
		defer os.Unsetenv("FIZZY_MIGRATE2_NO_KR")
		store := credstore.NewStore(credstore.StoreOptions{
			ServiceName:   "fizzy-migrate2-test",
			DisableEnvVar: "FIZZY_MIGRATE2_NO_KR",
			FallbackDir:   credDir,
		})
		profileStore := profile.NewStore(filepath.Join(profileDir, "config.json"))

		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestCreds(store)
		SetTestProfiles(profileStore)
		SetTestConfig("migrate-me", "acme", "https://app.fizzy.do")
		defer resetTest()

		resolveToken()

		// Token should now be in credstore under profile key
		var tokenStr string
		loaded, err := store.Load("profile:acme")
		if err != nil {
			t.Fatalf("expected token in credstore after migration: %v", err)
		}
		json.Unmarshal(loaded, &tokenStr)
		if tokenStr != "migrate-me" {
			t.Errorf("expected 'migrate-me' in credstore, got '%s'", tokenStr)
		}

		// Global YAML config should have token cleared
		data, _ := os.ReadFile(filepath.Join(configDir, "config.yaml"))
		var savedConfig config.Config
		yaml.Unmarshal(data, &savedConfig)
		if savedConfig.Token != "" {
			t.Errorf("expected empty token in YAML after migration, got '%s'", savedConfig.Token)
		}
		if savedConfig.Account != "acme" {
			t.Errorf("expected account 'acme' preserved, got '%s'", savedConfig.Account)
		}
	})

	t.Run("does not migrate when profile-scoped token exists", func(t *testing.T) {
		credDir := t.TempDir()
		profileDir := t.TempDir()

		os.Setenv("FIZZY_MIGRATE3_NO_KR", "1")
		defer os.Unsetenv("FIZZY_MIGRATE3_NO_KR")
		store := credstore.NewStore(credstore.StoreOptions{
			ServiceName:   "fizzy-migrate3-test",
			DisableEnvVar: "FIZZY_MIGRATE3_NO_KR",
			FallbackDir:   credDir,
		})
		profileStore := profile.NewStore(filepath.Join(profileDir, "config.json"))

		// Pre-populate credstore with a profile-scoped token
		credToken, _ := json.Marshal("cred-token")
		store.Save("profile:acme", credToken)

		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestCreds(store)
		SetTestProfiles(profileStore)
		SetTestConfig("", "acme", "https://app.fizzy.do")
		defer resetTest()

		resolveToken()

		// cfg.Token should be the profile-scoped token
		if cfg.Token != "cred-token" {
			t.Errorf("expected 'cred-token' from credstore, got '%s'", cfg.Token)
		}
	})

	t.Run("does not migrate env-var token to credstore", func(t *testing.T) {
		configDir := t.TempDir()
		credDir := t.TempDir()
		profileDir := t.TempDir()

		config.SetTestConfigDir(configDir)
		defer config.ResetTestConfigDir()

		// Global YAML config has NO token — only env var provides one
		os.WriteFile(filepath.Join(configDir, "config.yaml"),
			[]byte("account: acme"), 0600)

		os.Setenv("FIZZY_MIGRATE4_NO_KR", "1")
		defer os.Unsetenv("FIZZY_MIGRATE4_NO_KR")
		store := credstore.NewStore(credstore.StoreOptions{
			ServiceName:   "fizzy-migrate4-test",
			DisableEnvVar: "FIZZY_MIGRATE4_NO_KR",
			FallbackDir:   credDir,
		})
		profileStore := profile.NewStore(filepath.Join(profileDir, "config.json"))

		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestCreds(store)
		SetTestProfiles(profileStore)
		// cfg.Token set via env-like source (not from global YAML)
		SetTestConfig("env-token", "acme", "https://app.fizzy.do")
		defer resetTest()

		resolveToken()

		// Credstore should remain empty — env tokens must not be persisted
		if _, err := store.Load("profile:acme"); err == nil {
			t.Error("env-var token should not be migrated to credstore")
		}
	})
}

func TestProfileResolution(t *testing.T) {
	t.Run("FIZZY_PROFILE env var sets account and BaseURL", func(t *testing.T) {
		profileDir := t.TempDir()
		profileStore := profile.NewStore(filepath.Join(profileDir, "config.json"))
		profileStore.Create(&profile.Profile{Name: "staging", BaseURL: "https://staging.fizzy.do"})

		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestProfiles(profileStore)
		SetTestConfig("", "", "https://app.fizzy.do")
		defer resetTest()

		os.Setenv("FIZZY_PROFILE", "staging")
		defer os.Unsetenv("FIZZY_PROFILE")

		if err := resolveProfile(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cfg.Account != "staging" {
			t.Errorf("expected account 'staging', got '%s'", cfg.Account)
		}
		if cfg.APIURL != "https://staging.fizzy.do" {
			t.Errorf("expected APIURL 'https://staging.fizzy.do', got '%s'", cfg.APIURL)
		}
	})

	t.Run("profile BaseURL overrides YAML config APIURL", func(t *testing.T) {
		profileDir := t.TempDir()
		profileStore := profile.NewStore(filepath.Join(profileDir, "config.json"))
		profileStore.Create(&profile.Profile{Name: "custom", BaseURL: "https://custom.example.com"})

		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestProfiles(profileStore)
		SetTestConfig("", "", "https://app.fizzy.do")
		defer resetTest()

		// Single profile auto-selects
		if err := resolveProfile(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cfg.APIURL != "https://custom.example.com" {
			t.Errorf("expected APIURL 'https://custom.example.com', got '%s'", cfg.APIURL)
		}
	})

	t.Run("FIZZY_API_URL env var beats profile BaseURL", func(t *testing.T) {
		profileDir := t.TempDir()
		profileStore := profile.NewStore(filepath.Join(profileDir, "config.json"))
		profileStore.Create(&profile.Profile{Name: "custom", BaseURL: "https://profile.example.com"})

		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestProfiles(profileStore)
		SetTestConfig("", "", "https://app.fizzy.do")
		defer resetTest()

		os.Setenv("FIZZY_API_URL", "https://env.example.com")
		defer os.Unsetenv("FIZZY_API_URL")
		// Simulate what config.Load() does: apply env var to cfg before profile resolution
		cfg.APIURL = "https://env.example.com"

		if err := resolveProfile(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Env var value should survive — resolveProfile must not overwrite it with profile BaseURL
		if cfg.APIURL != "https://env.example.com" {
			t.Errorf("expected APIURL 'https://env.example.com' (from env), got '%s'", cfg.APIURL)
		}
	})

	t.Run("FIZZY_BOARD env var beats profile board", func(t *testing.T) {
		profileDir := t.TempDir()
		profileStore := profile.NewStore(filepath.Join(profileDir, "config.json"))
		profileStore.Create(&profile.Profile{
			Name:    "withboard",
			BaseURL: "https://app.fizzy.do",
			Extra: map[string]json.RawMessage{
				"board": json.RawMessage(`"profile-board"`),
			},
		})

		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestProfiles(profileStore)
		SetTestConfig("", "", "https://app.fizzy.do")
		defer resetTest()

		os.Setenv("FIZZY_BOARD", "env-board")
		defer os.Unsetenv("FIZZY_BOARD")
		// Simulate what config.Load() does: apply env var to cfg before profile resolution
		cfg.Board = "env-board"

		if err := resolveProfile(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Env var value should survive — resolveProfile must not overwrite it with profile board
		if cfg.Board != "env-board" {
			t.Errorf("expected board 'env-board' (from env), got '%s'", cfg.Board)
		}
	})

	t.Run("profile board from Extra applies when no env var", func(t *testing.T) {
		profileDir := t.TempDir()
		profileStore := profile.NewStore(filepath.Join(profileDir, "config.json"))
		profileStore.Create(&profile.Profile{
			Name:    "withboard",
			BaseURL: "https://app.fizzy.do",
			Extra: map[string]json.RawMessage{
				"board": json.RawMessage(`"board-123"`),
			},
		})

		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestProfiles(profileStore)
		SetTestConfig("", "", "https://app.fizzy.do")
		defer resetTest()

		if err := resolveProfile(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cfg.Board != "board-123" {
			t.Errorf("expected board 'board-123', got '%s'", cfg.Board)
		}
	})

	t.Run("FIZZY_ACCOUNT works as fallback for FIZZY_PROFILE", func(t *testing.T) {
		profileDir := t.TempDir()
		profileStore := profile.NewStore(filepath.Join(profileDir, "config.json"))
		profileStore.Create(&profile.Profile{Name: "legacy-acct", BaseURL: "https://app.fizzy.do"})

		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestProfiles(profileStore)
		SetTestConfig("", "", "https://app.fizzy.do")
		defer resetTest()

		os.Setenv("FIZZY_ACCOUNT", "legacy-acct")
		defer os.Unsetenv("FIZZY_ACCOUNT")

		if err := resolveProfile(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cfg.Account != "legacy-acct" {
			t.Errorf("expected account 'legacy-acct' from FIZZY_ACCOUNT fallback, got '%s'", cfg.Account)
		}
	})
}

func TestEnsureProfileUpdatesExisting(t *testing.T) {
	t.Run("updates existing profile's BaseURL and board", func(t *testing.T) {
		profileDir := t.TempDir()
		profileStore := profile.NewStore(filepath.Join(profileDir, "config.json"))
		profileStore.Create(&profile.Profile{Name: "acme", BaseURL: "https://old.example.com"})

		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestProfiles(profileStore)
		defer resetTest()

		// Call ensureProfile with new settings
		ensureProfile("acme", "https://new.example.com", "new-board")

		p, err := profileStore.Get("acme")
		if err != nil {
			t.Fatalf("expected profile to exist: %v", err)
		}
		if p.BaseURL != "https://new.example.com" {
			t.Errorf("expected BaseURL 'https://new.example.com', got '%s'", p.BaseURL)
		}
		var board string
		if boardRaw, ok := p.Extra["board"]; ok {
			json.Unmarshal(boardRaw, &board)
		}
		if board != "new-board" {
			t.Errorf("expected board 'new-board', got '%s'", board)
		}
	})

	t.Run("overwrites self-hosted BaseURL with default on hosted re-signup", func(t *testing.T) {
		profileDir := t.TempDir()
		profileStore := profile.NewStore(filepath.Join(profileDir, "config.json"))
		profileStore.Create(&profile.Profile{
			Name:    "acme",
			BaseURL: "https://selfhosted.example.com",
			Extra: map[string]json.RawMessage{
				"board": json.RawMessage(`"old-board"`),
			},
		})

		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestProfiles(profileStore)
		defer resetTest()

		// Re-signup with default URL should overwrite the self-hosted URL
		ensureProfile("acme", config.DefaultAPIURL, "")

		p, err := profileStore.Get("acme")
		if err != nil {
			t.Fatalf("expected profile to exist: %v", err)
		}
		if p.BaseURL != config.DefaultAPIURL {
			t.Errorf("expected BaseURL '%s', got '%s'", config.DefaultAPIURL, p.BaseURL)
		}
		// Board should be preserved since we passed empty
		var board string
		if boardRaw, ok := p.Extra["board"]; ok {
			json.Unmarshal(boardRaw, &board)
		}
		if board != "old-board" {
			t.Errorf("expected board 'old-board' to be preserved, got '%s'", board)
		}
	})

	t.Run("preserves existing BaseURL when caller passes empty", func(t *testing.T) {
		profileDir := t.TempDir()
		profileStore := profile.NewStore(filepath.Join(profileDir, "config.json"))
		profileStore.Create(&profile.Profile{
			Name:    "acme",
			BaseURL: "https://custom.example.com",
		})

		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestProfiles(profileStore)
		defer resetTest()

		// Empty baseURL should preserve the existing one
		ensureProfile("acme", "", "")

		p, err := profileStore.Get("acme")
		if err != nil {
			t.Fatalf("expected profile to exist: %v", err)
		}
		if p.BaseURL != "https://custom.example.com" {
			t.Errorf("expected BaseURL 'https://custom.example.com', got '%s'", p.BaseURL)
		}
	})
}

func TestAuthLogoutAllCleansLegacyKeys(t *testing.T) {
	t.Run("removes legacy token:<account> keys on logout --all", func(t *testing.T) {
		tempDir := t.TempDir()
		credDir := t.TempDir()
		profileDir := t.TempDir()

		config.SetTestConfigDir(tempDir)
		defer config.ResetTestConfigDir()

		os.Setenv("FIZZY_LOGOUTALL_LEGACY_NO_KR", "1")
		defer os.Unsetenv("FIZZY_LOGOUTALL_LEGACY_NO_KR")
		store := credstore.NewStore(credstore.StoreOptions{
			ServiceName:   "fizzy-logoutall-legacy-test",
			DisableEnvVar: "FIZZY_LOGOUTALL_LEGACY_NO_KR",
			FallbackDir:   credDir,
		})
		profileStore := profile.NewStore(filepath.Join(profileDir, "config.json"))
		profileStore.Create(&profile.Profile{Name: "acme", BaseURL: "https://app.fizzy.do"})

		// Save tokens in ALL key formats
		tokenData, _ := json.Marshal("my-token")
		store.Save("token", tokenData)        // bare legacy
		store.Save("token:acme", tokenData)   // account-scoped legacy
		store.Save("profile:acme", tokenData) // profile-scoped

		cfg := &config.Config{Account: "acme"}
		cfgData, _ := yaml.Marshal(cfg)
		os.WriteFile(filepath.Join(tempDir, "config.yaml"), cfgData, 0600)

		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestCreds(store)
		SetTestProfiles(profileStore)
		SetTestConfig("my-token", "acme", "https://app.fizzy.do")
		defer resetTest()

		authLogoutCmd.Flags().Set("all", "true")
		err := authLogoutCmd.RunE(authLogoutCmd, []string{})
		assertExitCode(t, err, 0)

		// ALL key formats should be cleaned up
		if _, err := store.Load("token"); err == nil {
			t.Error("expected bare 'token' key removed")
		}
		if _, err := store.Load("token:acme"); err == nil {
			t.Error("expected legacy 'token:acme' key removed")
		}
		if _, err := store.Load("profile:acme"); err == nil {
			t.Error("expected 'profile:acme' key removed")
		}

		// Profile should be gone from store
		if _, err := profileStore.Get("acme"); err == nil {
			t.Error("expected profile removed from store")
		}
	})
}

// TestPrecedenceChainIntegration exercises the full precedence chain as
// PersistentPreRunE would: config.Load() → resolveProfile() → resolveToken()
// → flag overrides, wired up with real config files, credstore, and profile store.
func TestPrecedenceChainIntegration(t *testing.T) {
	t.Run("profile outranks YAML config for APIURL and board", func(t *testing.T) {
		tempDir := t.TempDir()
		credDir := t.TempDir()
		profileDir := t.TempDir()

		// Write YAML config with values that should be overridden
		config.SetTestConfigDir(tempDir)
		defer config.ResetTestConfigDir()
		yamlCfg := &config.Config{
			Account: "acme",
			APIURL:  "https://yaml.example.com",
			Board:   "yaml-board",
		}
		yamlData, _ := yaml.Marshal(yamlCfg)
		os.WriteFile(filepath.Join(tempDir, "config.yaml"), yamlData, 0600)

		// Profile with different BaseURL and board
		profileStore := profile.NewStore(filepath.Join(profileDir, "config.json"))
		profileStore.Create(&profile.Profile{
			Name:    "acme",
			BaseURL: "https://profile.example.com",
			Extra: map[string]json.RawMessage{
				"board": json.RawMessage(`"profile-board"`),
			},
		})
		profileStore.SetDefault("acme")

		// Credstore with token under profile key
		os.Setenv("FIZZY_PRECEDENCE_TEST_NO_KR", "1")
		defer os.Unsetenv("FIZZY_PRECEDENCE_TEST_NO_KR")
		store := credstore.NewStore(credstore.StoreOptions{
			ServiceName:   "fizzy-precedence-test",
			DisableEnvVar: "FIZZY_PRECEDENCE_TEST_NO_KR",
			FallbackDir:   credDir,
		})
		tokenData, _ := json.Marshal("cred-token")
		store.Save("profile:acme", tokenData)

		// Step 1: config.Load() — picks up YAML values
		loaded := config.Load()

		// Step 2: wire up package state as PersistentPreRunE would
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		defer resetTest()
		cfg = loaded
		SetTestCreds(store)
		SetTestProfiles(profileStore)

		// Step 3: resolveProfile() — profile overwrites YAML
		if err := resolveProfile(); err != nil {
			t.Fatalf("resolveProfile: %v", err)
		}

		// Step 4: resolveToken() — credstore overwrites YAML token
		resolveToken()

		if cfg.Account != "acme" {
			t.Errorf("account: want 'acme', got '%s'", cfg.Account)
		}
		if cfg.APIURL != "https://profile.example.com" {
			t.Errorf("APIURL: want profile value 'https://profile.example.com', got '%s'", cfg.APIURL)
		}
		if cfg.Board != "profile-board" {
			t.Errorf("board: want profile value 'profile-board', got '%s'", cfg.Board)
		}
		if cfg.Token != "cred-token" {
			t.Errorf("token: want credstore value 'cred-token', got '%s'", cfg.Token)
		}
	})

	t.Run("env vars beat profile for all fields", func(t *testing.T) {
		tempDir := t.TempDir()
		credDir := t.TempDir()
		profileDir := t.TempDir()

		config.SetTestConfigDir(tempDir)
		defer config.ResetTestConfigDir()

		profileStore := profile.NewStore(filepath.Join(profileDir, "config.json"))
		profileStore.Create(&profile.Profile{
			Name:    "acme",
			BaseURL: "https://profile.example.com",
			Extra: map[string]json.RawMessage{
				"board": json.RawMessage(`"profile-board"`),
			},
		})
		profileStore.SetDefault("acme")

		os.Setenv("FIZZY_PRECEDENCE_ENV_NO_KR", "1")
		defer os.Unsetenv("FIZZY_PRECEDENCE_ENV_NO_KR")
		store := credstore.NewStore(credstore.StoreOptions{
			ServiceName:   "fizzy-precedence-env-test",
			DisableEnvVar: "FIZZY_PRECEDENCE_ENV_NO_KR",
			FallbackDir:   credDir,
		})
		tokenData, _ := json.Marshal("cred-token")
		store.Save("profile:acme", tokenData)

		// Set env vars that should beat profile
		os.Setenv("FIZZY_API_URL", "https://env.example.com")
		defer os.Unsetenv("FIZZY_API_URL")
		os.Setenv("FIZZY_BOARD", "env-board")
		defer os.Unsetenv("FIZZY_BOARD")
		os.Setenv("FIZZY_TOKEN", "env-token")
		defer os.Unsetenv("FIZZY_TOKEN")

		// Step 1: config.Load() — picks up env vars
		loaded := config.Load()

		// Step 2: wire up
		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		defer resetTest()
		cfg = loaded
		SetTestCreds(store)
		SetTestProfiles(profileStore)

		// Step 3: resolveProfile()
		if err := resolveProfile(); err != nil {
			t.Fatalf("resolveProfile: %v", err)
		}

		// Step 4: resolveToken()
		resolveToken()

		if cfg.APIURL != "https://env.example.com" {
			t.Errorf("APIURL: want env value 'https://env.example.com', got '%s'", cfg.APIURL)
		}
		if cfg.Board != "env-board" {
			t.Errorf("board: want env value 'env-board', got '%s'", cfg.Board)
		}
		if cfg.Token != "env-token" {
			t.Errorf("token: want env value 'env-token', got '%s'", cfg.Token)
		}
	})

	t.Run("flag beats env and profile for APIURL", func(t *testing.T) {
		tempDir := t.TempDir()
		profileDir := t.TempDir()

		config.SetTestConfigDir(tempDir)
		defer config.ResetTestConfigDir()

		profileStore := profile.NewStore(filepath.Join(profileDir, "config.json"))
		profileStore.Create(&profile.Profile{
			Name:    "acme",
			BaseURL: "https://profile.example.com",
		})
		profileStore.SetDefault("acme")

		os.Setenv("FIZZY_API_URL", "https://env.example.com")
		defer os.Unsetenv("FIZZY_API_URL")

		loaded := config.Load()

		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		defer resetTest()
		cfg = loaded
		SetTestProfiles(profileStore)

		if err := resolveProfile(); err != nil {
			t.Fatalf("resolveProfile: %v", err)
		}

		// Simulate --api-url flag (same as PersistentPreRunE line 117-119)
		cfgAPIURL = "https://flag.example.com"
		defer func() { cfgAPIURL = "" }()
		if cfgAPIURL != "" {
			cfg.APIURL = cfgAPIURL
		}

		if cfg.APIURL != "https://flag.example.com" {
			t.Errorf("APIURL: want flag value 'https://flag.example.com', got '%s'", cfg.APIURL)
		}
	})
}
