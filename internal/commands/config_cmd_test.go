package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/basecamp/cli/output"
	"github.com/basecamp/cli/profile"
	"github.com/basecamp/fizzy-cli/internal/config"
)

func TestConfigShow(t *testing.T) {
	configDir := t.TempDir()
	profileDir := t.TempDir()
	config.SetTestConfigDir(configDir)
	defer config.ResetTestConfigDir()

	profileStore := profile.NewStore(filepath.Join(profileDir, "config.json"))
	if err := profileStore.Create(&profile.Profile{
		Name:    "acme",
		BaseURL: "https://profile.example.com",
		Extra: map[string]json.RawMessage{
			"board": json.RawMessage(`"board-123"`),
		},
	}); err != nil {
		t.Fatalf("create profile: %v", err)
	}
	if err := profileStore.SetDefault("acme"); err != nil {
		t.Fatalf("set default profile: %v", err)
	}

	mock := NewMockClient()
	result := SetTestModeWithSDK(mock)
	SetTestProfiles(profileStore)
	SetTestConfig("", "acme", "https://profile.example.com")
	t.Setenv("FIZZY_TOKEN", "env-token")
	cfg.Board = "board-123"
	defer resetTest()

	err := configShowCmd.RunE(configShowCmd, []string{})
	assertExitCode(t, err, 0)

	data, ok := result.Response.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected map response data, got %#v", result.Response.Data)
	}
	if data["profile"] != "acme" {
		t.Fatalf("expected profile acme, got %#v", data["profile"])
	}
	if data["api_url"] != "https://profile.example.com" {
		t.Fatalf("expected profile api url, got %#v", data["api_url"])
	}
	if data["board"] != "board-123" {
		t.Fatalf("expected board-123, got %#v", data["board"])
	}
	token, ok := data["token"].(map[string]any)
	if !ok {
		t.Fatalf("expected token map, got %#v", data["token"])
	}
	if token["configured"] != true {
		t.Fatalf("expected configured token, got %#v", token)
	}
	if token["source"] != "environment variable" {
		t.Fatalf("expected env token source, got %#v", token)
	}
	profiles, ok := data["profiles"].([]string)
	if !ok {
		items, okAny := data["profiles"].([]any)
		if !okAny {
			t.Fatalf("expected profiles slice, got %#v", data["profiles"])
		}
		profiles = make([]string, 0, len(items))
		for _, item := range items {
			profiles = append(profiles, item.(string))
		}
	}
	if len(profiles) != 1 || profiles[0] != "acme (default)" {
		t.Fatalf("expected saved profile list, got %#v", profiles)
	}
}

func TestConfigExplainShowsPrecedence(t *testing.T) {
	configDir := t.TempDir()
	workDir := t.TempDir()
	profileDir := t.TempDir()
	config.SetTestConfigDir(configDir)
	config.SetTestWorkingDir(workDir)
	defer config.ResetTestConfigDir()
	defer config.ResetTestWorkingDir()

	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("account: global\napi_url: https://global.example.com\nboard: global-board\n"), 0o600); err != nil {
		t.Fatalf("write global config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workDir, config.LocalConfigFile), []byte("account: local\nboard: local-board\n"), 0o600); err != nil {
		t.Fatalf("write local config: %v", err)
	}

	profileStore := profile.NewStore(filepath.Join(profileDir, "config.json"))
	if err := profileStore.Create(&profile.Profile{
		Name:    "acme",
		BaseURL: "https://profile.example.com",
		Extra: map[string]json.RawMessage{
			"board": json.RawMessage(`"profile-board"`),
		},
	}); err != nil {
		t.Fatalf("create profile: %v", err)
	}
	if err := profileStore.SetDefault("acme"); err != nil {
		t.Fatalf("set default profile: %v", err)
	}

	mock := NewMockClient()
	result := SetTestModeWithSDK(mock)
	SetTestProfiles(profileStore)
	t.Setenv("FIZZY_PROFILE", "acme")
	t.Setenv("FIZZY_API_URL", "https://env.example.com")
	cfg = config.Load()
	cfg.Account = "acme"
	cfg.APIURL = "https://env.example.com"
	cfg.Board = "profile-board"
	defer resetTest()

	err := configExplainCmd.RunE(configExplainCmd, []string{})
	assertExitCode(t, err, 0)

	data, ok := result.Response.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected map response data, got %#v", result.Response.Data)
	}

	profileField := data["profile"].(map[string]any)
	if profileField["source"] != "env FIZZY_PROFILE" {
		t.Fatalf("expected env profile source, got %#v", profileField)
	}
	apiURLField := data["api_url"].(map[string]any)
	if apiURLField["source"] != "env FIZZY_API_URL" {
		t.Fatalf("expected env api url source, got %#v", apiURLField)
	}
	boardField := data["board"].(map[string]any)
	if boardField["source"] != "profile acme" {
		t.Fatalf("expected profile board source, got %#v", boardField)
	}

	candidates := apiURLField["candidates"].([]any)
	var sawEnvSelected bool
	for _, item := range candidates {
		candidate := item.(map[string]any)
		if candidate["source"] == "env FIZZY_API_URL" && candidate["selected"] == true {
			sawEnvSelected = true
		}
	}
	if !sawEnvSelected {
		t.Fatalf("expected env API URL candidate to be selected, got %#v", candidates)
	}
}

func TestConfigShowVerboseIncludesProfiles(t *testing.T) {
	configDir := t.TempDir()
	profileDir := t.TempDir()
	config.SetTestConfigDir(configDir)
	defer config.ResetTestConfigDir()

	profileStore := profile.NewStore(filepath.Join(profileDir, "config.json"))
	if err := profileStore.Create(&profile.Profile{Name: "acme", BaseURL: "https://acme.example.com"}); err != nil {
		t.Fatalf("create profile: %v", err)
	}
	if err := profileStore.SetDefault("acme"); err != nil {
		t.Fatalf("set default: %v", err)
	}

	mock := NewMockClient()
	SetTestModeWithSDK(mock)
	SetTestProfiles(profileStore)
	SetTestFormat(output.FormatStyled)
	SetTestConfig("", "acme", "https://acme.example.com")
	cfgVerbose = true
	defer func() { cfgVerbose = false }()
	defer resetTest()

	err := configShowCmd.RunE(configShowCmd, []string{})
	assertExitCode(t, err, 0)

	raw := TestOutput()
	if !strings.Contains(raw, "acme") {
		t.Fatalf("expected saved profile 'acme' in verbose styled output, got:\n%s", raw)
	}
	if !strings.Contains(raw, "Saved Profiles") {
		t.Fatalf("expected 'Saved Profiles' label in verbose styled output, got:\n%s", raw)
	}
}

func TestConfigShowVerboseJSONIncludesProfiles(t *testing.T) {
	configDir := t.TempDir()
	profileDir := t.TempDir()
	config.SetTestConfigDir(configDir)
	defer config.ResetTestConfigDir()

	profileStore := profile.NewStore(filepath.Join(profileDir, "config.json"))
	if err := profileStore.Create(&profile.Profile{Name: "staging", BaseURL: "https://staging.example.com"}); err != nil {
		t.Fatalf("create profile: %v", err)
	}
	if err := profileStore.SetDefault("staging"); err != nil {
		t.Fatalf("set default: %v", err)
	}

	mock := NewMockClient()
	result := SetTestModeWithSDK(mock)
	SetTestProfiles(profileStore)
	SetTestConfig("", "staging", "https://staging.example.com")
	cfgVerbose = true
	defer func() { cfgVerbose = false }()
	defer resetTest()

	err := configShowCmd.RunE(configShowCmd, []string{})
	assertExitCode(t, err, 0)

	data, ok := result.Response.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected map response data, got %#v", result.Response.Data)
	}
	profiles, ok := data["profiles"]
	if !ok {
		t.Fatalf("expected 'profiles' key in verbose JSON response, got keys: %v", mapKeys(data))
	}
	items, ok := profiles.([]string)
	if !ok {
		anyItems, okAny := profiles.([]any)
		if !okAny {
			t.Fatalf("expected profiles slice, got %#v", profiles)
		}
		items = make([]string, 0, len(anyItems))
		for _, item := range anyItems {
			items = append(items, item.(string))
		}
	}
	if len(items) != 1 || !strings.Contains(items[0], "staging") {
		t.Fatalf("expected saved profile 'staging', got %#v", items)
	}
}

func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func TestConfigExplainStyledOutput(t *testing.T) {
	configDir := t.TempDir()
	config.SetTestConfigDir(configDir)
	defer config.ResetTestConfigDir()

	mock := NewMockClient()
	SetTestModeWithSDK(mock)
	SetTestFormat(output.FormatStyled)
	SetTestConfig("", "", testHTTPServer.URL)
	defer resetTest()

	err := configExplainCmd.RunE(configExplainCmd, []string{})
	assertExitCode(t, err, 0)

	raw := TestOutput()
	if !strings.Contains(raw, "Fizzy Config Explain") {
		t.Fatalf("expected explain heading, got:\n%s", raw)
	}
	if !strings.Contains(raw, "Next steps") {
		t.Fatalf("expected next steps section, got:\n%s", raw)
	}
}
