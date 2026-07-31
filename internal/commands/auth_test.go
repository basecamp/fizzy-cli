package commands

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

	t.Run("saves an alias separately from its account", func(t *testing.T) {
		credDir := t.TempDir()
		configDir := t.TempDir()
		profileDir := t.TempDir()

		config.SetTestConfigDir(configDir)
		defer config.ResetTestConfigDir()
		os.Setenv("FIZZY_ALIAS_LOGIN_NO_KR", "1")
		defer os.Unsetenv("FIZZY_ALIAS_LOGIN_NO_KR")
		store := credstore.NewStore(credstore.StoreOptions{
			ServiceName:   "fizzy-alias-login-test",
			DisableEnvVar: "FIZZY_ALIAS_LOGIN_NO_KR",
			FallbackDir:   credDir,
		})
		profileStore := profile.NewStore(filepath.Join(profileDir, "config.json"))

		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestCreds(store)
		SetTestProfiles(profileStore)
		SetTestConfig("", "walter", "https://app.fizzy.do")
		activeProfile = "walter"
		authLoginAccount = "1"
		defer resetTest()

		if err := authLoginCmd.RunE(authLoginCmd, []string{"walter-token"}); err != nil {
			t.Fatalf("login: %v", err)
		}

		p, err := profileStore.Get("walter")
		if err != nil {
			t.Fatalf("get walter profile: %v", err)
		}
		if account := profileAccount("walter", p); account != "1" {
			t.Errorf("profile account: want 1, got %q", account)
		}
		if _, err := store.Load("profile:walter"); err != nil {
			t.Fatalf("load aliased credential: %v", err)
		}
		globalCfg := config.LoadGlobal()
		if globalCfg.Account != "1" {
			t.Errorf("global account: want 1, got %q", globalCfg.Account)
		}
	})

	t.Run("rejects invalid aliases before saving credentials", func(t *testing.T) {
		os.Setenv("FIZZY_INVALID_ALIAS_NO_KR", "1")
		defer os.Unsetenv("FIZZY_INVALID_ALIAS_NO_KR")
		store := credstore.NewStore(credstore.StoreOptions{
			ServiceName:   "fizzy-invalid-alias-test",
			DisableEnvVar: "FIZZY_INVALID_ALIAS_NO_KR",
			FallbackDir:   t.TempDir(),
		})
		profileStore := profile.NewStore(filepath.Join(t.TempDir(), "config.json"))

		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestCreds(store)
		SetTestProfiles(profileStore)
		SetTestConfig("", "walter.agent", "https://app.fizzy.do")
		activeProfile = "walter.agent"
		authLoginAccount = "1"
		defer resetTest()

		err := authLoginCmd.RunE(authLoginCmd, []string{"agent-token"})
		if err == nil {
			t.Fatal("expected invalid profile error")
		}
		if _, err := store.Load("profile:walter.agent"); err == nil {
			t.Fatal("credential was saved for an invalid profile")
		}
	})

	t.Run("invalid selector does not migrate a legacy token", func(t *testing.T) {
		configDir := t.TempDir()
		config.SetTestConfigDir(configDir)
		config.SetTestWorkingDir(t.TempDir())
		defer config.ResetTestConfigDir()
		defer config.ResetTestWorkingDir()

		os.Setenv("FIZZY_INVALID_MIGRATION_NO_KR", "1")
		defer os.Unsetenv("FIZZY_INVALID_MIGRATION_NO_KR")
		store := credstore.NewStore(credstore.StoreOptions{
			ServiceName:   "fizzy-invalid-migration-test",
			DisableEnvVar: "FIZZY_INVALID_MIGRATION_NO_KR",
			FallbackDir:   t.TempDir(),
		})
		legacyToken, _ := json.Marshal("legacy-token")
		if err := store.Save("token", legacyToken); err != nil {
			t.Fatalf("save legacy token: %v", err)
		}
		profileStore := profile.NewStore(filepath.Join(configDir, "config.json"))
		if err := profileStore.Create(&profile.Profile{Name: "existing", BaseURL: config.DefaultAPIURL}); err != nil {
			t.Fatalf("create existing profile: %v", err)
		}

		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestCreds(store)
		SetTestProfiles(profileStore)
		SetTestConfig("", "existing", config.DefaultAPIURL)
		defer resetTest()

		_, err := runCobraWithArgs("auth", "login", "replacement-token", "--profile", "walter.agent", "--account", "1")
		if err == nil {
			t.Fatal("expected invalid profile error")
		}
		if _, err := store.Load("profile:walter.agent"); err == nil {
			t.Fatal("legacy token was migrated to an invalid profile")
		}
		if _, err := store.Load("token"); err != nil {
			t.Fatalf("legacy token was removed: %v", err)
		}
	})

	t.Run("restores a new profile when credential saving fails", func(t *testing.T) {
		configDir := t.TempDir()
		config.SetTestConfigDir(configDir)
		defer config.ResetTestConfigDir()

		blockedCredDir := filepath.Join(t.TempDir(), "blocked")
		if err := os.WriteFile(blockedCredDir, []byte("not a directory"), 0600); err != nil {
			t.Fatalf("create blocked credential path: %v", err)
		}
		t.Setenv("FIZZY_LOGIN_ROLLBACK_NEW_NO_KR", "1")
		store := credstore.NewStore(credstore.StoreOptions{
			ServiceName:   "fizzy-login-rollback-new-test",
			DisableEnvVar: "FIZZY_LOGIN_ROLLBACK_NEW_NO_KR",
			FallbackDir:   blockedCredDir,
		})
		profileStore := profile.NewStore(filepath.Join(configDir, "config.json"))
		if err := profileStore.Create(&profile.Profile{Name: "existing", BaseURL: config.DefaultAPIURL}); err != nil {
			t.Fatalf("create existing profile: %v", err)
		}

		SetTestCreds(store)
		SetTestProfiles(profileStore)
		SetTestConfig("", "1", config.DefaultAPIURL)
		activeProfile = "agent"
		authLoginAccount = "1"
		defer resetTest()

		err := authLoginCmd.RunE(authLoginCmd, []string{"replacement-token"})
		if err == nil {
			t.Fatal("expected credential save error")
		}
		allProfiles, defaultName, listErr := profileStore.List()
		if listErr != nil {
			t.Fatalf("list profiles: %v", listErr)
		}
		if _, ok := allProfiles["agent"]; ok {
			t.Fatal("failed login left the new profile behind")
		}
		if defaultName != "existing" {
			t.Fatalf("default profile: want existing, got %q", defaultName)
		}
	})

	for _, initialDefault := range []string{"other", "agent", ""} {
		name := "restores existing profile when credential saving fails"
		switch initialDefault {
		case "agent":
			name += " while already default"
		case "":
			name += " with no prior default"
		}
		t.Run(name, func(t *testing.T) {
			configDir := t.TempDir()
			config.SetTestConfigDir(configDir)
			defer config.ResetTestConfigDir()

			t.Setenv("FIZZY_LOGIN_ROLLBACK_EXISTING_NO_KR", "1")
			credDir := filepath.Join(t.TempDir(), "credentials")
			store := credstore.NewStore(credstore.StoreOptions{
				ServiceName:   "fizzy-login-rollback-existing-test",
				DisableEnvVar: "FIZZY_LOGIN_ROLLBACK_EXISTING_NO_KR",
				FallbackDir:   credDir,
			})
			oldToken, _ := json.Marshal("old-token")
			if err := store.Save("profile:agent", oldToken); err != nil {
				t.Fatalf("save old credential: %v", err)
			}
			backupCredDir := credDir + "-backup"
			if err := os.Rename(credDir, backupCredDir); err != nil {
				t.Fatalf("move credential directory: %v", err)
			}
			if err := os.WriteFile(credDir, []byte("not a directory"), 0600); err != nil {
				t.Fatalf("block credential directory: %v", err)
			}

			profileStore := profile.NewStore(filepath.Join(configDir, "config.json"))
			if err := profileStore.Create(&profile.Profile{Name: "other", BaseURL: config.DefaultAPIURL}); err != nil {
				t.Fatalf("create other profile: %v", err)
			}
			if err := profileStore.Create(&profile.Profile{
				Name:    "agent",
				BaseURL: "https://old.example.com",
				Extra: map[string]json.RawMessage{
					"account": json.RawMessage(`"old-account"`),
					"board":   json.RawMessage(`"old-board"`),
				},
			}); err != nil {
				t.Fatalf("create agent profile: %v", err)
			}
			switch initialDefault {
			case "agent":
				if err := profileStore.SetDefault("agent"); err != nil {
					t.Fatalf("set initial default: %v", err)
				}
			case "":
				if err := profileStore.Delete("other"); err != nil {
					t.Fatalf("clear initial default: %v", err)
				}
			}

			SetTestCreds(store)
			SetTestProfiles(profileStore)
			SetTestConfig("", "new-account", "https://new.example.com")
			activeProfile = "agent"
			authLoginAccount = "new-account"

			err := authLoginCmd.RunE(authLoginCmd, []string{"replacement-token"})
			if removeErr := os.Remove(credDir); removeErr != nil {
				t.Fatalf("unblock credential directory: %v", removeErr)
			}
			if renameErr := os.Rename(backupCredDir, credDir); renameErr != nil {
				t.Fatalf("restore credential directory: %v", renameErr)
			}
			defer resetTest()
			if err == nil {
				t.Fatal("expected credential save error")
			}

			restored, getErr := profileStore.Get("agent")
			if getErr != nil {
				t.Fatalf("get restored profile: %v", getErr)
			}
			if restored.BaseURL != "https://old.example.com" {
				t.Errorf("BaseURL: want old value, got %q", restored.BaseURL)
			}
			if got := profileAccount("agent", restored); got != "old-account" {
				t.Errorf("account: want old-account, got %q", got)
			}
			if got := string(restored.Extra["board"]); got != `"old-board"` {
				t.Errorf("board metadata: want old value, got %s", got)
			}
			_, defaultName, listErr := profileStore.List()
			if listErr != nil {
				t.Fatalf("list profiles: %v", listErr)
			}
			if defaultName != initialDefault {
				t.Errorf("default profile: want %q, got %q", initialDefault, defaultName)
			}
			data, loadErr := store.Load("profile:agent")
			if loadErr != nil {
				t.Fatalf("load restored credential: %v", loadErr)
			}
			if string(data) != string(oldToken) {
				t.Errorf("credential changed: want %s, got %s", oldToken, data)
			}
		})
	}

	t.Run("rejects unsafe account identifiers before saving", func(t *testing.T) {
		for _, account := range []string{"../other", "other?admin=1", "other%2Fadmin", " other"} {
			t.Run(account, func(t *testing.T) {
				t.Setenv("FIZZY_UNSAFE_ACCOUNT_NO_KR", "1")
				store := credstore.NewStore(credstore.StoreOptions{
					ServiceName:   "fizzy-unsafe-account-test",
					DisableEnvVar: "FIZZY_UNSAFE_ACCOUNT_NO_KR",
					FallbackDir:   t.TempDir(),
				})
				profileStore := profile.NewStore(filepath.Join(t.TempDir(), "config.json"))
				SetTestCreds(store)
				SetTestProfiles(profileStore)
				SetTestConfig("", account, config.DefaultAPIURL)
				activeProfile = "agent"
				authLoginAccount = account
				defer resetTest()

				if err := authLoginCmd.RunE(authLoginCmd, []string{"token"}); err == nil {
					t.Fatal("expected invalid account error")
				}
				if _, err := store.Load("profile:agent"); err == nil {
					t.Fatal("credential was saved for unsafe account")
				}
				if _, err := profileStore.Get("agent"); err == nil {
					t.Fatal("profile was saved for unsafe account")
				}
			})
		}
	})

	t.Run("global config failure restores profile and credential", func(t *testing.T) {
		configDir := t.TempDir()
		config.SetTestConfigDir(configDir)
		defer config.ResetTestConfigDir()
		if err := os.Mkdir(filepath.Join(configDir, "config.yaml"), 0700); err != nil {
			t.Fatalf("block global config: %v", err)
		}
		t.Setenv("FIZZY_LOGIN_YAML_FAILURE_NO_KR", "1")
		store := credstore.NewStore(credstore.StoreOptions{
			ServiceName:   "fizzy-login-yaml-failure-test",
			DisableEnvVar: "FIZZY_LOGIN_YAML_FAILURE_NO_KR",
			FallbackDir:   t.TempDir(),
		})
		profileStore := profile.NewStore(filepath.Join(t.TempDir(), "config.json"))
		if err := profileStore.Create(&profile.Profile{Name: "existing", BaseURL: config.DefaultAPIURL}); err != nil {
			t.Fatalf("create existing profile: %v", err)
		}
		SetTestCreds(store)
		SetTestProfiles(profileStore)
		SetTestConfig("", "1", config.DefaultAPIURL)
		activeProfile = "agent"
		authLoginAccount = "1"
		defer resetTest()

		if err := authLoginCmd.RunE(authLoginCmd, []string{"token"}); err == nil {
			t.Fatal("expected global config error")
		}
		if _, err := store.Load("profile:agent"); err == nil {
			t.Fatal("credential remained after failed login")
		}
		allProfiles, defaultName, err := profileStore.List()
		if err != nil {
			t.Fatalf("list profiles: %v", err)
		}
		if _, exists := allProfiles["agent"]; exists {
			t.Fatal("profile remained after failed login")
		}
		if defaultName != "existing" {
			t.Fatalf("default profile: want existing, got %q", defaultName)
		}
	})

	t.Run("does not save credentials when profile creation fails", func(t *testing.T) {
		configDir := t.TempDir()
		config.SetTestConfigDir(configDir)
		defer config.ResetTestConfigDir()

		os.Setenv("FIZZY_PROFILE_FAILURE_NO_KR", "1")
		defer os.Unsetenv("FIZZY_PROFILE_FAILURE_NO_KR")
		store := credstore.NewStore(credstore.StoreOptions{
			ServiceName:   "fizzy-profile-failure-test",
			DisableEnvVar: "FIZZY_PROFILE_FAILURE_NO_KR",
			FallbackDir:   t.TempDir(),
		})
		blockedParent := filepath.Join(t.TempDir(), "not-a-directory")
		if err := os.WriteFile(blockedParent, []byte("blocked"), 0o600); err != nil {
			t.Fatalf("create blocking file: %v", err)
		}
		profileStore := profile.NewStore(filepath.Join(blockedParent, "config.json"))

		mock := NewMockClient()
		SetTestModeWithSDK(mock)
		SetTestCreds(store)
		SetTestProfiles(profileStore)
		SetTestConfig("", "agent", "https://app.fizzy.do")
		activeProfile = "agent"
		authLoginAccount = "1"
		defer resetTest()

		err := authLoginCmd.RunE(authLoginCmd, []string{"agent-token"})
		if err == nil {
			t.Fatal("expected profile creation error")
		}
		if _, err := store.Load("profile:agent"); err == nil {
			t.Fatal("credential was saved without a profile")
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

func TestAuthLoginCreatesProfilesFromExplicitSelectors(t *testing.T) {
	for _, tt := range []struct {
		name        string
		profileName string
		profileArgs []string
		envProfile  string
		account     string
	}{
		{name: "flag alias", profileName: "agent", profileArgs: []string{"--profile", "agent"}, account: "1"},
		{name: "environment alias", profileName: "agent", envProfile: "agent", account: "1"},
		{name: "profile name defaults to account", profileName: "new-account", profileArgs: []string{"--profile", "new-account"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			configDir := t.TempDir()
			config.SetTestConfigDir(configDir)
			config.SetTestWorkingDir(t.TempDir())
			defer config.ResetTestConfigDir()
			defer config.ResetTestWorkingDir()

			os.Setenv("FIZZY_ALIAS_SELECTOR_NO_KR", "1")
			defer os.Unsetenv("FIZZY_ALIAS_SELECTOR_NO_KR")
			if tt.envProfile != "" {
				os.Setenv("FIZZY_PROFILE", tt.envProfile)
				defer os.Unsetenv("FIZZY_PROFILE")
			}
			store := credstore.NewStore(credstore.StoreOptions{
				ServiceName:   "fizzy-alias-selector-test-" + tt.name,
				DisableEnvVar: "FIZZY_ALIAS_SELECTOR_NO_KR",
				FallbackDir:   t.TempDir(),
			})
			profileStore := profile.NewStore(filepath.Join(configDir, "config.json"))
			if err := profileStore.Create(&profile.Profile{Name: "existing", BaseURL: "https://app.fizzy.do"}); err != nil {
				t.Fatalf("create existing profile: %v", err)
			}

			mock := NewMockClient()
			SetTestModeWithSDK(mock)
			SetTestCreds(store)
			SetTestProfiles(profileStore)
			SetTestConfig("", "existing", "https://app.fizzy.do")
			defer resetTest()

			args := make([]string, 0, 5+len(tt.profileArgs))
			args = append(args, "auth", "login", "agent-token")
			if tt.account != "" {
				args = append(args, "--account", tt.account)
			}
			args = append(args, tt.profileArgs...)
			if _, err := runCobraWithArgs(args...); err != nil {
				t.Fatalf("login with %s selector: %v", tt.name, err)
			}

			p, err := profileStore.Get(tt.profileName)
			if err != nil {
				t.Fatalf("get %s profile: %v", tt.profileName, err)
			}
			expectedAccount := firstNonEmpty(tt.account, tt.profileName)
			if account := profileAccount(tt.profileName, p); account != expectedAccount {
				t.Errorf("account: want %q, got %q", expectedAccount, account)
			}
			if _, err := store.Load("profile:" + tt.profileName); err != nil {
				t.Fatalf("load %s credential: %v", tt.profileName, err)
			}
		})
	}
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

func TestAuthLogoutReportsCredentialDeletionFailure(t *testing.T) {
	configDir := t.TempDir()
	config.SetTestConfigDir(configDir)
	defer config.ResetTestConfigDir()
	t.Setenv("FIZZY_LOGOUT_FAILURE_NO_KR", "1")

	credDir := filepath.Join(t.TempDir(), "credentials")
	store := credstore.NewStore(credstore.StoreOptions{
		ServiceName:   "fizzy-logout-failure-test",
		DisableEnvVar: "FIZZY_LOGOUT_FAILURE_NO_KR",
		FallbackDir:   credDir,
	})
	tokenData, _ := json.Marshal("agent-token")
	if err := store.Save("profile:agent", tokenData); err != nil {
		t.Fatalf("save credential: %v", err)
	}
	backupDir := credDir + "-backup"
	if err := os.Rename(credDir, backupDir); err != nil {
		t.Fatalf("move credential directory: %v", err)
	}
	if err := os.WriteFile(credDir, []byte("not a directory"), 0600); err != nil {
		t.Fatalf("block credential directory: %v", err)
	}
	profileStore := profile.NewStore(filepath.Join(configDir, "config.json"))
	if err := profileStore.Create(&profile.Profile{Name: "agent", BaseURL: config.DefaultAPIURL}); err != nil {
		t.Fatalf("create profile: %v", err)
	}

	SetTestModeWithSDK(NewMockClient())
	SetTestCreds(store)
	SetTestProfiles(profileStore)
	SetTestConfig("agent-token", "agent", config.DefaultAPIURL)
	activeProfile = "agent"
	err := authLogoutCmd.RunE(authLogoutCmd, nil)
	if removeErr := os.Remove(credDir); removeErr != nil {
		t.Fatalf("unblock credential directory: %v", removeErr)
	}
	if renameErr := os.Rename(backupDir, credDir); renameErr != nil {
		t.Fatalf("restore credential directory: %v", renameErr)
	}
	defer resetTest()
	if err == nil {
		t.Fatal("expected incomplete logout error")
	}
	if _, loadErr := store.Load("profile:agent"); loadErr != nil {
		t.Fatalf("credential should remain after failed deletion: %v", loadErr)
	}
}

func TestAuthLogoutUnknownExplicitProfilePreservesActiveProfile(t *testing.T) {
	configDir := t.TempDir()
	config.SetTestConfigDir(configDir)
	defer config.ResetTestConfigDir()
	t.Setenv("FIZZY_LOGOUT_UNKNOWN_NO_KR", "1")

	store := credstore.NewStore(credstore.StoreOptions{
		ServiceName:   "fizzy-logout-unknown-test",
		DisableEnvVar: "FIZZY_LOGOUT_UNKNOWN_NO_KR",
		FallbackDir:   t.TempDir(),
	})
	tokenData, _ := json.Marshal("active-token")
	if err := store.Save("profile:active", tokenData); err != nil {
		t.Fatalf("save active credential: %v", err)
	}
	profileStore := profile.NewStore(filepath.Join(configDir, "config.json"))
	if err := profileStore.Create(&profile.Profile{Name: "active", BaseURL: config.DefaultAPIURL}); err != nil {
		t.Fatalf("create active profile: %v", err)
	}
	if err := (&config.Config{Account: "active", APIURL: config.DefaultAPIURL}).Save(); err != nil {
		t.Fatalf("save global config: %v", err)
	}
	SetTestModeWithSDK(NewMockClient())
	SetTestCreds(store)
	SetTestProfiles(profileStore)
	SetTestConfig("active-token", "active", config.DefaultAPIURL)
	defer resetTest()

	if _, err := runCobraWithArgs("auth", "logout", "--profile", "typo"); err != nil {
		t.Fatalf("logout unknown profile: %v", err)
	}
	if _, err := store.Load("profile:active"); err != nil {
		t.Fatalf("active credential was deleted: %v", err)
	}
	if _, err := profileStore.Get("active"); err != nil {
		t.Fatalf("active profile was deleted: %v", err)
	}
}

func TestAuthLogoutAllReportsCredentialDeletionFailure(t *testing.T) {
	configDir := t.TempDir()
	config.SetTestConfigDir(configDir)
	defer config.ResetTestConfigDir()
	t.Setenv("FIZZY_LOGOUT_ALL_FAILURE_NO_KR", "1")

	credDir := filepath.Join(t.TempDir(), "credentials")
	store := credstore.NewStore(credstore.StoreOptions{
		ServiceName:   "fizzy-logout-all-failure-test",
		DisableEnvVar: "FIZZY_LOGOUT_ALL_FAILURE_NO_KR",
		FallbackDir:   credDir,
	})
	tokenData, _ := json.Marshal("agent-token")
	if err := store.Save("profile:agent", tokenData); err != nil {
		t.Fatalf("save credential: %v", err)
	}
	backupDir := credDir + "-backup"
	if err := os.Rename(credDir, backupDir); err != nil {
		t.Fatalf("move credential directory: %v", err)
	}
	if err := os.WriteFile(credDir, []byte("not a directory"), 0600); err != nil {
		t.Fatalf("block credential directory: %v", err)
	}
	profileStore := profile.NewStore(filepath.Join(configDir, "config.json"))
	if err := profileStore.Create(&profile.Profile{Name: "agent", BaseURL: config.DefaultAPIURL}); err != nil {
		t.Fatalf("create profile: %v", err)
	}
	SetTestCreds(store)
	SetTestProfiles(profileStore)
	err := authLogoutAll()
	if removeErr := os.Remove(credDir); removeErr != nil {
		t.Fatalf("unblock credential directory: %v", removeErr)
	}
	if renameErr := os.Rename(backupDir, credDir); renameErr != nil {
		t.Fatalf("restore credential directory: %v", renameErr)
	}
	defer resetTest()
	if err == nil {
		t.Fatal("expected incomplete logout error")
	}
	if _, loadErr := store.Load("profile:agent"); loadErr != nil {
		t.Fatalf("credential should remain after failed deletion: %v", loadErr)
	}
}

func TestAuthLogoutAllCleansInvalidProfileMetadata(t *testing.T) {
	configDir := t.TempDir()
	config.SetTestConfigDir(configDir)
	defer config.ResetTestConfigDir()
	t.Setenv("FIZZY_LOGOUT_INVALID_NO_KR", "1")

	store := credstore.NewStore(credstore.StoreOptions{
		ServiceName:   "fizzy-logout-invalid-test",
		DisableEnvVar: "FIZZY_LOGOUT_INVALID_NO_KR",
		FallbackDir:   t.TempDir(),
	})
	tokenData, _ := json.Marshal("agent-token")
	if err := store.Save("profile:agent", tokenData); err != nil {
		t.Fatalf("save credential: %v", err)
	}
	profileStore := profile.NewStore(filepath.Join(configDir, "config.json"))
	if err := profileStore.Create(&profile.Profile{
		Name:    "agent",
		BaseURL: config.DefaultAPIURL,
		Extra:   map[string]json.RawMessage{"account": json.RawMessage(`"../invalid"`)},
	}); err != nil {
		t.Fatalf("create invalid profile: %v", err)
	}
	SetTestModeWithSDK(NewMockClient())
	SetTestCreds(store)
	SetTestProfiles(profileStore)
	defer resetTest()

	if err := authLogoutAll(); err != nil {
		t.Fatalf("logout all: %v", err)
	}
	if _, err := store.Load("profile:agent"); err == nil {
		t.Fatal("invalid profile credential still exists")
	}
	if _, err := profileStore.Get("agent"); err == nil {
		t.Fatal("invalid profile still exists")
	}
}

func TestAuthLogoutAliasClearsActiveLegacyAccount(t *testing.T) {
	configDir := t.TempDir()
	config.SetTestConfigDir(configDir)
	config.SetTestWorkingDir(t.TempDir())
	defer config.ResetTestConfigDir()
	defer config.ResetTestWorkingDir()

	globalCfg := &config.Config{Account: "1", APIURL: "https://app.fizzy.do"}
	if err := globalCfg.Save(); err != nil {
		t.Fatalf("save global config: %v", err)
	}

	os.Setenv("FIZZY_ALIAS_LOGOUT_NO_KR", "1")
	defer os.Unsetenv("FIZZY_ALIAS_LOGOUT_NO_KR")
	store := credstore.NewStore(credstore.StoreOptions{
		ServiceName:   "fizzy-alias-logout-test",
		DisableEnvVar: "FIZZY_ALIAS_LOGOUT_NO_KR",
		FallbackDir:   t.TempDir(),
	})
	aliasToken, _ := json.Marshal("alias-token")
	legacyToken, _ := json.Marshal("legacy-token")
	if err := store.Save("profile:walter", aliasToken); err != nil {
		t.Fatalf("save alias token: %v", err)
	}
	if err := store.Save("token:1", legacyToken); err != nil {
		t.Fatalf("save legacy token: %v", err)
	}

	profileStore := profile.NewStore(filepath.Join(configDir, "config.json"))
	if err := profileStore.Create(&profile.Profile{
		Name:    "walter",
		BaseURL: "https://app.fizzy.do",
		Extra:   map[string]json.RawMessage{"account": json.RawMessage(`"1"`)},
	}); err != nil {
		t.Fatalf("create alias profile: %v", err)
	}

	mock := NewMockClient()
	SetTestModeWithSDK(mock)
	SetTestCreds(store)
	SetTestProfiles(profileStore)
	SetTestConfig("", "1", "https://app.fizzy.do")
	defer resetTest()

	if err := resolveProfile(); err != nil {
		t.Fatalf("resolve alias: %v", err)
	}
	resolveToken()
	if cfg.Token != "alias-token" {
		t.Fatalf("token before logout: want alias-token, got %q", cfg.Token)
	}
	if err := authLogoutCmd.RunE(authLogoutCmd, nil); err != nil {
		t.Fatalf("logout alias: %v", err)
	}

	// Simulate the next process invocation. The preserved account-scoped legacy
	// token must not become active after the selected alias is removed.
	cfg = config.Load()
	activeProfile = ""
	cfgProfile = ""
	if err := resolveProfile(); err != nil {
		t.Fatalf("resolve after logout: %v", err)
	}
	resolveToken()
	if cfg.Account != "" {
		t.Errorf("account after logout: want empty, got %q", cfg.Account)
	}
	if cfg.Token != "" {
		t.Errorf("token after logout: want empty, got %q", cfg.Token)
	}
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

		initialCfg := &config.Config{Account: "acme"}
		cfgData, _ := yaml.Marshal(initialCfg)
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
		if savedConfig.APIURL != "https://staging.fizzy.do" {
			t.Errorf("expected persisted target API URL, got %q", savedConfig.APIURL)
		}
		if cfg.APIURL != "https://staging.fizzy.do" {
			t.Errorf("expected target API URL to be applied, got %q", cfg.APIURL)
		}
		targetProfile, err := profileStore.Get("other")
		if err != nil {
			t.Fatalf("get target profile: %v", err)
		}
		if targetProfile.BaseURL != "https://staging.fizzy.do" {
			t.Errorf("expected target API URL to remain unchanged, got %q", targetProfile.BaseURL)
		}

		// Verify profile store default was updated
		_, defaultName, _ := profileStore.List()
		if defaultName != "other" {
			t.Errorf("expected default profile 'other', got '%s'", defaultName)
		}
	})

	for _, tt := range []struct {
		name      string
		envURL    string
		flagURL   string
		effective string
	}{
		{name: "preserves environment API URL override", envURL: "https://env.example.com", effective: "https://env.example.com"},
		{name: "preserves flag API URL override", envURL: "https://env.example.com", flagURL: "https://flag.example.com", effective: "https://flag.example.com"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			configDir := t.TempDir()
			config.SetTestConfigDir(configDir)
			defer config.ResetTestConfigDir()
			t.Setenv("FIZZY_API_URL", tt.envURL)
			t.Setenv("FIZZY_SWITCH_OVERRIDE_NO_KR", "1")

			store := credstore.NewStore(credstore.StoreOptions{
				ServiceName:   "fizzy-switch-override-test",
				DisableEnvVar: "FIZZY_SWITCH_OVERRIDE_NO_KR",
				FallbackDir:   t.TempDir(),
			})
			profileStore := profile.NewStore(filepath.Join(t.TempDir(), "config.json"))
			if err := profileStore.Create(&profile.Profile{Name: "acme", BaseURL: "https://app.fizzy.do"}); err != nil {
				t.Fatalf("create acme profile: %v", err)
			}
			if err := profileStore.Create(&profile.Profile{Name: "other", BaseURL: "https://profile.example.com"}); err != nil {
				t.Fatalf("create other profile: %v", err)
			}
			tokenData, _ := json.Marshal("other-token")
			if err := store.Save("profile:other", tokenData); err != nil {
				t.Fatalf("save other credential: %v", err)
			}

			SetTestModeWithSDK(NewMockClient())
			SetTestCreds(store)
			SetTestProfiles(profileStore)
			SetTestConfig("acme-token", "acme", tt.effective)
			cfgAPIURL = tt.flagURL
			defer func() { cfgAPIURL = "" }()
			defer resetTest()

			if err := authSwitchCmd.RunE(authSwitchCmd, []string{"other"}); err != nil {
				t.Fatalf("switch profile: %v", err)
			}
			if cfg.APIURL != tt.effective {
				t.Errorf("effective API URL: want %q, got %q", tt.effective, cfg.APIURL)
			}
			data, err := os.ReadFile(filepath.Join(configDir, "config.yaml"))
			if err != nil {
				t.Fatalf("read global config: %v", err)
			}
			var savedConfig config.Config
			if err := yaml.Unmarshal(data, &savedConfig); err != nil {
				t.Fatalf("parse global config: %v", err)
			}
			if savedConfig.APIURL != "https://profile.example.com" {
				t.Errorf("persisted API URL: want profile URL, got %q", savedConfig.APIURL)
			}
		})
	}

	t.Run("empty profile credential is rejected", func(t *testing.T) {
		t.Setenv("FIZZY_SWITCH_EMPTY_TOKEN_NO_KR", "1")
		store := credstore.NewStore(credstore.StoreOptions{
			ServiceName:   "fizzy-switch-empty-token-test",
			DisableEnvVar: "FIZZY_SWITCH_EMPTY_TOKEN_NO_KR",
			FallbackDir:   t.TempDir(),
		})
		emptyToken, _ := json.Marshal("")
		if err := store.Save("profile:agent", emptyToken); err != nil {
			t.Fatalf("save empty credential: %v", err)
		}
		profileStore := profile.NewStore(filepath.Join(t.TempDir(), "config.json"))
		if err := profileStore.Create(&profile.Profile{Name: "agent", BaseURL: config.DefaultAPIURL}); err != nil {
			t.Fatalf("create profile: %v", err)
		}
		SetTestModeWithSDK(NewMockClient())
		SetTestCreds(store)
		SetTestProfiles(profileStore)
		SetTestConfig("", "agent", config.DefaultAPIURL)
		defer resetTest()

		if err := authSwitchCmd.RunE(authSwitchCmd, []string{"agent"}); err == nil {
			t.Fatal("expected missing credential error")
		}
	})

	t.Run("alias requires a profile-scoped credential", func(t *testing.T) {
		t.Setenv("FIZZY_SWITCH_ALIAS_LEGACY_NO_KR", "1")
		store := credstore.NewStore(credstore.StoreOptions{
			ServiceName:   "fizzy-switch-alias-legacy-test",
			DisableEnvVar: "FIZZY_SWITCH_ALIAS_LEGACY_NO_KR",
			FallbackDir:   t.TempDir(),
		})
		legacyToken, _ := json.Marshal("legacy-token")
		if err := store.Save("token", legacyToken); err != nil {
			t.Fatalf("save legacy credential: %v", err)
		}
		profileStore := profile.NewStore(filepath.Join(t.TempDir(), "config.json"))
		if err := profileStore.Create(&profile.Profile{Name: "existing", BaseURL: config.DefaultAPIURL}); err != nil {
			t.Fatalf("create existing profile: %v", err)
		}
		if err := profileStore.Create(&profile.Profile{
			Name:    "agent",
			BaseURL: config.DefaultAPIURL,
			Extra:   map[string]json.RawMessage{"account": json.RawMessage(`"1"`)},
		}); err != nil {
			t.Fatalf("create alias: %v", err)
		}
		SetTestModeWithSDK(NewMockClient())
		SetTestCreds(store)
		SetTestProfiles(profileStore)
		SetTestConfig("", "existing", config.DefaultAPIURL)
		defer resetTest()

		if err := authSwitchCmd.RunE(authSwitchCmd, []string{"agent"}); err == nil {
			t.Fatal("expected missing alias credential error")
		}
		_, defaultName, err := profileStore.List()
		if err != nil {
			t.Fatalf("list profiles: %v", err)
		}
		if defaultName != "existing" {
			t.Fatalf("default profile changed to %q", defaultName)
		}
	})

	t.Run("recovers from an invalid default profile", func(t *testing.T) {
		configDir := t.TempDir()
		config.SetTestConfigDir(configDir)
		defer config.ResetTestConfigDir()
		t.Setenv("FIZZY_SWITCH_INVALID_DEFAULT_NO_KR", "1")
		store := credstore.NewStore(credstore.StoreOptions{
			ServiceName:   "fizzy-switch-invalid-default-test",
			DisableEnvVar: "FIZZY_SWITCH_INVALID_DEFAULT_NO_KR",
			FallbackDir:   t.TempDir(),
		})
		profileStore := profile.NewStore(filepath.Join(configDir, "config.json"))
		if err := profileStore.Create(&profile.Profile{
			Name:    "broken",
			BaseURL: config.DefaultAPIURL,
			Extra:   map[string]json.RawMessage{"account": json.RawMessage(`"../invalid"`)},
		}); err != nil {
			t.Fatalf("create broken profile: %v", err)
		}
		if err := profileStore.Create(&profile.Profile{Name: "good", BaseURL: config.DefaultAPIURL}); err != nil {
			t.Fatalf("create good profile: %v", err)
		}
		goodToken, _ := json.Marshal("good-token")
		if err := store.Save("profile:good", goodToken); err != nil {
			t.Fatalf("save good credential: %v", err)
		}
		SetTestModeWithSDK(NewMockClient())
		SetTestCreds(store)
		SetTestProfiles(profileStore)
		SetTestConfig("", "broken", config.DefaultAPIURL)
		defer resetTest()

		if _, err := runCobraWithArgs("auth", "switch", "good"); err != nil {
			t.Fatalf("switch from invalid default: %v", err)
		}
		_, defaultName, err := profileStore.List()
		if err != nil {
			t.Fatalf("list profiles: %v", err)
		}
		if defaultName != "good" {
			t.Fatalf("default profile: want good, got %q", defaultName)
		}
	})

	t.Run("global config failure restores previous default", func(t *testing.T) {
		configDir := t.TempDir()
		config.SetTestConfigDir(configDir)
		defer config.ResetTestConfigDir()
		if err := os.Mkdir(filepath.Join(configDir, "config.yaml"), 0700); err != nil {
			t.Fatalf("block global config: %v", err)
		}
		t.Setenv("FIZZY_SWITCH_YAML_FAILURE_NO_KR", "1")
		store := credstore.NewStore(credstore.StoreOptions{
			ServiceName:   "fizzy-switch-yaml-failure-test",
			DisableEnvVar: "FIZZY_SWITCH_YAML_FAILURE_NO_KR",
			FallbackDir:   t.TempDir(),
		})
		profileStore := profile.NewStore(filepath.Join(t.TempDir(), "config.json"))
		for _, name := range []string{"acme", "other"} {
			if err := profileStore.Create(&profile.Profile{Name: name, BaseURL: config.DefaultAPIURL}); err != nil {
				t.Fatalf("create profile %s: %v", name, err)
			}
			data, _ := json.Marshal(name + "-token")
			if err := store.Save("profile:"+name, data); err != nil {
				t.Fatalf("save credential %s: %v", name, err)
			}
		}
		SetTestModeWithSDK(NewMockClient())
		SetTestCreds(store)
		SetTestProfiles(profileStore)
		SetTestConfig("acme-token", "acme", config.DefaultAPIURL)
		defer resetTest()

		if err := authSwitchCmd.RunE(authSwitchCmd, []string{"other"}); err == nil {
			t.Fatal("expected global config error")
		}
		_, defaultName, err := profileStore.List()
		if err != nil {
			t.Fatalf("list profiles: %v", err)
		}
		if defaultName != "acme" {
			t.Fatalf("default profile: want acme, got %q", defaultName)
		}
	})

	t.Run("reconstructed profile inherits the effective API URL", func(t *testing.T) {
		configDir := t.TempDir()
		config.SetTestConfigDir(configDir)
		defer config.ResetTestConfigDir()
		t.Setenv("FIZZY_SWITCH_RECONSTRUCT_NO_KR", "1")
		store := credstore.NewStore(credstore.StoreOptions{
			ServiceName:   "fizzy-switch-reconstruct-test",
			DisableEnvVar: "FIZZY_SWITCH_RECONSTRUCT_NO_KR",
			FallbackDir:   t.TempDir(),
		})
		tokenData, _ := json.Marshal("other-token")
		if err := store.Save("profile:other", tokenData); err != nil {
			t.Fatalf("save target credential: %v", err)
		}
		profileStore := profile.NewStore(filepath.Join(t.TempDir(), "config.json"))
		if err := profileStore.Create(&profile.Profile{Name: "current", BaseURL: "https://self-hosted.example.com"}); err != nil {
			t.Fatalf("create current profile: %v", err)
		}
		SetTestModeWithSDK(NewMockClient())
		SetTestCreds(store)
		SetTestProfiles(profileStore)
		SetTestConfig("current-token", "current", "https://self-hosted.example.com")
		defer resetTest()

		if err := authSwitchCmd.RunE(authSwitchCmd, []string{"other"}); err != nil {
			t.Fatalf("switch reconstructed profile: %v", err)
		}
		reconstructed, err := profileStore.Get("other")
		if err != nil {
			t.Fatalf("get reconstructed profile: %v", err)
		}
		if reconstructed.BaseURL != "https://self-hosted.example.com" {
			t.Fatalf("BaseURL: want self-hosted URL, got %q", reconstructed.BaseURL)
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

func TestProfileAliasesUseSharedAccountWithDistinctTokens(t *testing.T) {
	type observedRequest struct {
		path          string
		authorization string
	}
	requests := make(chan observedRequest, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- observedRequest{path: r.URL.Path, authorization: r.Header.Get("Authorization")}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	credDir := t.TempDir()
	profileDir := t.TempDir()

	os.Setenv("FIZZY_ALIAS_NO_KR", "1")
	defer os.Unsetenv("FIZZY_ALIAS_NO_KR")
	store := credstore.NewStore(credstore.StoreOptions{
		ServiceName:   "fizzy-alias-test",
		DisableEnvVar: "FIZZY_ALIAS_NO_KR",
		FallbackDir:   credDir,
	})
	profileStore := profile.NewStore(filepath.Join(profileDir, "config.json"))
	for _, name := range []string{"walter", "walter2"} {
		if err := profileStore.Create(&profile.Profile{
			Name:    name,
			BaseURL: server.URL,
			Extra: map[string]json.RawMessage{
				"account": json.RawMessage(`"1"`),
			},
		}); err != nil {
			t.Fatalf("create profile %s: %v", name, err)
		}
	}

	for profileName, token := range map[string]string{
		"walter":  "walter-token",
		"walter2": "agent-token",
	} {
		data, _ := json.Marshal(token)
		if err := store.Save("profile:"+profileName, data); err != nil {
			t.Fatalf("save token for %s: %v", profileName, err)
		}
	}

	mock := NewMockClient()
	SetTestMode(mock)
	SetTestCreds(store)
	SetTestProfiles(profileStore)
	SetTestConfig("", "legacy-account", server.URL)
	defer resetTest()

	for _, tt := range []struct {
		profile string
		token   string
	}{
		{profile: "walter", token: "walter-token"},
		{profile: "walter2", token: "agent-token"},
	} {
		t.Run(tt.profile, func(t *testing.T) {
			cfgProfile = tt.profile
			cfg.Token = ""
			if err := resolveProfile(); err != nil {
				t.Fatalf("resolve profile: %v", err)
			}
			resolveToken()

			if cfg.Account != "1" {
				t.Errorf("account: want shared account '1', got %q", cfg.Account)
			}
			if cfg.Token != tt.token {
				t.Errorf("token: want %q, got %q", tt.token, cfg.Token)
			}

			if err := initSDK(boardListCmd, cfg.APIURL, cfg.Token, cfg.Account); err != nil {
				t.Fatalf("initialize SDK: %v", err)
			}
			if _, _, err := getSDK().Boards().List(context.Background(), "/boards.json"); err != nil {
				t.Fatalf("list boards: %v", err)
			}
			request := <-requests
			if request.path != "/1/boards.json" {
				t.Errorf("request path: want /1/boards.json, got %q", request.path)
			}
			if request.authorization != "Bearer "+tt.token {
				t.Errorf("authorization: want bearer token for %s, got %q", tt.profile, request.authorization)
			}
		})
	}
}

func TestAliasedProfileDoesNotBorrowLegacyCredentials(t *testing.T) {
	configDir := t.TempDir()
	config.SetTestConfigDir(configDir)
	defer config.ResetTestConfigDir()

	t.Setenv("FIZZY_ALIAS_LEGACY_NO_KR", "1")
	store := credstore.NewStore(credstore.StoreOptions{
		ServiceName:   "fizzy-alias-legacy-test",
		DisableEnvVar: "FIZZY_ALIAS_LEGACY_NO_KR",
		FallbackDir:   t.TempDir(),
	})
	legacyToken, _ := json.Marshal("legacy-token")
	if err := store.Save("token", legacyToken); err != nil {
		t.Fatalf("save legacy credential: %v", err)
	}
	if err := (&config.Config{Token: "yaml-token", Account: "1"}).Save(); err != nil {
		t.Fatalf("save legacy config: %v", err)
	}
	profileStore := profile.NewStore(filepath.Join(configDir, "config.json"))
	if err := profileStore.Create(&profile.Profile{
		Name:    "agent",
		BaseURL: config.DefaultAPIURL,
		Extra:   map[string]json.RawMessage{"account": json.RawMessage(`"1"`)},
	}); err != nil {
		t.Fatalf("create alias: %v", err)
	}

	SetTestCreds(store)
	SetTestProfiles(profileStore)
	SetTestConfig("yaml-token", "1", config.DefaultAPIURL)
	defer resetTest()

	if err := resolveProfile(); err != nil {
		t.Fatalf("resolve profile: %v", err)
	}
	resolveToken()
	if cfg.Token != "" {
		t.Fatalf("alias borrowed legacy credential %q", cfg.Token)
	}
	if _, err := store.Load("profile:agent"); err == nil {
		t.Fatal("legacy credential was migrated into alias")
	}
}

func TestProfileStoreReadFailureDoesNotMigrateLegacyCredential(t *testing.T) {
	t.Setenv("FIZZY_PROFILE_READ_FAILURE_NO_KR", "1")
	store := credstore.NewStore(credstore.StoreOptions{
		ServiceName:   "fizzy-profile-read-failure-test",
		DisableEnvVar: "FIZZY_PROFILE_READ_FAILURE_NO_KR",
		FallbackDir:   t.TempDir(),
	})
	legacyToken, _ := json.Marshal("legacy-token")
	if err := store.Save("token", legacyToken); err != nil {
		t.Fatalf("save legacy credential: %v", err)
	}
	profilePath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(profilePath, []byte(`{invalid`), 0600); err != nil {
		t.Fatalf("write malformed profile store: %v", err)
	}
	SetTestCreds(store)
	SetTestProfiles(profile.NewStore(profilePath))
	SetTestConfig("", "agent", config.DefaultAPIURL)
	cfgProfile = "agent"
	defer resetTest()

	if err := resolveProfile(); err == nil {
		t.Fatal("expected profile-store read error")
	}
	migrateLegacyToken("agent")
	if _, err := store.Load("profile:agent"); err == nil {
		t.Fatal("legacy credential migrated after profile-store read failure")
	}
}

func TestInvalidProfileAccountMetadataIsRejected(t *testing.T) {
	for _, raw := range []string{`{}`, `null`, `""`, `"   "`, `"../other"`, `"other?admin=1"`, `"other%2Fadmin"`} {
		t.Run(raw, func(t *testing.T) {
			profileStore := profile.NewStore(filepath.Join(t.TempDir(), "config.json"))
			if err := profileStore.Create(&profile.Profile{
				Name:    "agent",
				BaseURL: config.DefaultAPIURL,
				Extra:   map[string]json.RawMessage{"account": json.RawMessage(raw)},
			}); err != nil {
				t.Fatalf("create profile: %v", err)
			}
			SetTestProfiles(profileStore)
			SetTestConfig("token", "legacy", config.DefaultAPIURL)
			defer resetTest()

			if err := resolveProfile(); err == nil {
				t.Fatal("expected invalid account metadata error")
			}
		})
	}
}

func TestProfileAccountDefaultsToProfileName(t *testing.T) {
	p := &profile.Profile{Name: "6102600", BaseURL: "https://app.fizzy.do"}
	if account := profileAccount(p.Name, p); account != "6102600" {
		t.Errorf("account: want legacy profile name, got %q", account)
	}
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
		if err := ensureProfile("acme", "https://new.example.com", "new-board"); err != nil {
			t.Fatalf("ensure profile: %v", err)
		}

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
		if err := ensureProfile("acme", config.DefaultAPIURL, ""); err != nil {
			t.Fatalf("ensure profile: %v", err)
		}

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
		if err := ensureProfile("acme", "", ""); err != nil {
			t.Fatalf("ensure profile: %v", err)
		}

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
		profileStore.Create(&profile.Profile{Name: "walter", BaseURL: "https://app.fizzy.do", Extra: map[string]json.RawMessage{"account": json.RawMessage(`"1"`)}})
		profileStore.Create(&profile.Profile{Name: "jane", BaseURL: "https://app.fizzy.do", Extra: map[string]json.RawMessage{"account": json.RawMessage(`"2"`)}})

		// Save tokens in every key format, including legacy keys for aliased accounts.
		tokenData, _ := json.Marshal("my-token")
		store.Save("token", tokenData)          // bare legacy
		store.Save("token:acme", tokenData)     // account-scoped legacy
		store.Save("token:1", tokenData)        // aliased account legacy
		store.Save("token:2", tokenData)        // non-active aliased account legacy
		store.Save("profile:acme", tokenData)   // profile-scoped
		store.Save("profile:walter", tokenData) // aliased profile
		store.Save("profile:jane", tokenData)   // non-active aliased profile

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
		defer authLogoutCmd.Flags().Set("all", "false")
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
		for _, key := range []string{"token:1", "token:2", "profile:walter", "profile:jane"} {
			if _, err := store.Load(key); err == nil {
				t.Errorf("expected %q key removed", key)
			}
		}

		// Every profile should be gone from the store.
		for _, name := range []string{"acme", "walter", "jane"} {
			if _, err := profileStore.Get(name); err == nil {
				t.Errorf("expected profile %q removed", name)
			}
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
