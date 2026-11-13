package commands

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/basecamp/cli/credstore"
	"github.com/basecamp/cli/output"
	"github.com/basecamp/cli/profile"
	"github.com/basecamp/fizzy-cli/internal/config"
	"github.com/spf13/cobra"
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

func TestConfigShowVerboseAttributesAliasAccountToProfile(t *testing.T) {
	profileStore := profile.NewStore(filepath.Join(t.TempDir(), "config.json"))
	if err := profileStore.Create(&profile.Profile{
		Name:    "walter",
		BaseURL: config.DefaultAPIURL,
		Extra:   map[string]json.RawMessage{"account": json.RawMessage(`"1"`)},
	}); err != nil {
		t.Fatalf("create alias: %v", err)
	}
	SetTestProfiles(profileStore)
	SetTestConfig("token", "legacy", config.DefaultAPIURL)
	cfgProfile = "walter"
	defer resetTest()
	if err := resolveProfile(); err != nil {
		t.Fatalf("resolve profile: %v", err)
	}

	data := configShowData(true)
	account, ok := data["account"].(map[string]any)
	if !ok {
		t.Fatalf("account: expected object, got %#v", data["account"])
	}
	if account["value"] != "1" || account["source"] != "profile walter" {
		t.Fatalf("account: want value 1 from profile walter, got %#v", account)
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
			"account": json.RawMessage(`"1"`),
			"board":   json.RawMessage(`"profile-board"`),
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
	cfg.Account = "1"
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
	accountField := data["account"].(map[string]any)
	if accountField["value"] != "1" || accountField["source"] != "profile acme" {
		t.Fatalf("expected account 1 from profile acme, got %#v", accountField)
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

func TestConfigExplainAccountCandidatesFollowProfileFallback(t *testing.T) {
	configDir := t.TempDir()
	workDir := t.TempDir()
	config.SetTestConfigDir(configDir)
	config.SetTestWorkingDir(workDir)
	defer config.ResetTestConfigDir()
	defer config.ResetTestWorkingDir()

	if err := os.WriteFile(filepath.Join(workDir, config.LocalConfigFile), []byte("account: stale-local\n"), 0o600); err != nil {
		t.Fatalf("write local config: %v", err)
	}
	profileStore := profile.NewStore(filepath.Join(t.TempDir(), "config.json"))
	mock := NewMockClient()
	SetTestModeWithSDK(mock)
	SetTestProfiles(profileStore)
	t.Setenv("FIZZY_PROFILE", "new-account")
	cfg = config.Load()
	defer resetTest()

	if err := resolveProfile(); err != nil {
		t.Fatalf("resolve profile fallback: %v", err)
	}
	field := configExplainData()["account"].(configExplainField)
	if field.Value != "new-account" || field.Source != "env FIZZY_PROFILE" {
		t.Fatalf("effective account: want new-account from env, got %#v", field)
	}

	selected := ""
	for _, candidate := range field.Candidates {
		if candidate.Selected {
			selected = candidate.Source
		}
	}
	if selected != "env FIZZY_PROFILE" {
		t.Fatalf("selected account candidate: want env FIZZY_PROFILE, got %q (%#v)", selected, field.Candidates)
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

// `fizzy config trust` is what lets a local config's api_url apply, and
// `fizzy config untrust` takes it away again.
func TestConfigTrustAndUntrust(t *testing.T) {
	workDir := t.TempDir()
	config.SetTestConfigDir(t.TempDir())
	config.SetTestWorkingDir(workDir)
	defer config.ResetTestConfigDir()
	defer config.ResetTestWorkingDir()
	t.Setenv("FIZZY_API_URL", "")

	if err := os.WriteFile(filepath.Join(workDir, config.LocalConfigFile), []byte("api_url: https://fizzy.example.com\n"), 0o600); err != nil {
		t.Fatalf("write local config: %v", err)
	}
	if got := config.Load().APIURL; got != config.DefaultAPIURL {
		t.Fatalf("before trust: want the default api_url, got %q", got)
	}

	result := SetTestModeWithSDK(NewMockClient())
	defer resetTest()

	assertExitCode(t, configTrustCmd.RunE(configTrustCmd, []string{}), 0)
	data, _ := result.Response.Data.(map[string]any)
	if data["trusted"] != true || !strings.HasSuffix(data["path"].(string), config.LocalConfigFile) {
		t.Fatalf("trust: got %#v", result.Response.Data)
	}
	if got := config.Load().APIURL; got != "https://fizzy.example.com" {
		t.Fatalf("after trust: got %q", got)
	}

	assertExitCode(t, configUntrustCmd.RunE(configUntrustCmd, []string{}), 0)
	if got := config.Load().APIURL; got != config.DefaultAPIURL {
		t.Fatalf("after untrust: want the default api_url, got %q", got)
	}
}

func TestConfigTrustWithoutALocalConfigIsAUsageError(t *testing.T) {
	config.SetTestConfigDir(t.TempDir())
	config.SetTestWorkingDir(t.TempDir())
	defer config.ResetTestConfigDir()
	defer config.ResetTestWorkingDir()

	SetTestModeWithSDK(NewMockClient())
	defer resetTest()

	err := configTrustCmd.RunE(configTrustCmd, []string{})
	if err == nil || !strings.Contains(err.Error(), config.LocalConfigFile) {
		t.Fatalf("want an error naming the file, got %v", err)
	}
}

// Doctor says when a local config's api_url and token are being ignored, and
// does not attribute the effective values to it.
func TestDoctorWarnsOnAnUntrustedLocalConfig(t *testing.T) {
	workDir := t.TempDir()
	config.SetTestWorkingDir(workDir)
	config.SetTestConfigDir(t.TempDir())
	defer config.ResetTestWorkingDir()
	defer config.ResetTestConfigDir()

	if err := os.WriteFile(filepath.Join(workDir, config.LocalConfigFile), []byte("api_url: https://evil.example.com\ntoken: planted\n"), 0o600); err != nil {
		t.Fatalf("write local config: %v", err)
	}

	check := checkDoctorLocalConfig(false)
	if check.Status != "warn" || !strings.Contains(check.Message, "api_url and token ignored") || !strings.Contains(check.Hint, "fizzy config trust") {
		t.Fatalf("got %#v", check)
	}
	if local := appliedLocalConfig(); local == nil || local.APIURL != "" || local.Token != "" {
		t.Fatalf("the ignored values are not reported as the local config's, got %#v", local)
	}
}

// accountNamedProfiles sets up several stored profiles with no default, as
// logging out of the default profile leaves them, and the account "acme"
// configured.
func accountNamedProfiles(t *testing.T) {
	t.Helper()
	t.Setenv("FIZZY_API_URL", "")
	t.Setenv("FIZZY_PROFILE", "")
	t.Setenv("FIZZY_ACCOUNT", "")
	t.Setenv("FIZZY_TOKEN", "")
	t.Setenv("FIZZY_ACCOUNT_HOST_NO_KR", "1")
	config.SetTestConfigDir(t.TempDir())
	config.SetTestWorkingDir(t.TempDir())
	t.Cleanup(config.ResetTestConfigDir)
	t.Cleanup(config.ResetTestWorkingDir)

	storePath := filepath.Join(t.TempDir(), "config.json")
	stored := `{"profiles":{"acme":{"base_url":"https://fizzy.acme.example"},"other":{"base_url":"https://app.fizzy.do"}}}`
	if err := os.WriteFile(storePath, []byte(stored), 0o600); err != nil {
		t.Fatalf("write profile store: %v", err)
	}
	store := credstore.NewStore(credstore.StoreOptions{
		ServiceName:   "fizzy-account-host-test",
		DisableEnvVar: "FIZZY_ACCOUNT_HOST_NO_KR",
		FallbackDir:   t.TempDir(),
	})

	SetTestModeWithSDK(NewMockClient())
	SetTestProfiles(profile.NewStore(storePath))
	SetTestCreds(store)
	SetTestConfig("", "acme", "https://app.fizzy.do")
	t.Cleanup(resetTest)
}

// With several profiles and no default, no profile is chosen and the
// configured account names the credential. A cloned repo's local config can
// set that account, so the token it selects goes to its own profile's host,
// not to the host the YAML config or the default would give it.
func TestAnAccountNamedProfileKeepsItsTokenOnItsOwnHost(t *testing.T) {
	accountNamedProfiles(t)
	if err := credsSaveProfileToken("acme", "acme-token"); err != nil {
		t.Fatalf("save token: %v", err)
	}

	if err := resolveProfile(); err != nil {
		t.Fatalf("resolve profile: %v", err)
	}
	resolveToken()
	if activeProfile != "" {
		t.Fatalf("want no profile chosen, got %q", activeProfile)
	}
	if cfg.Token != "acme-token" {
		t.Fatalf("the account names the credential, got token %q", cfg.Token)
	}
	if cfg.APIURL != "https://fizzy.acme.example" {
		t.Errorf("want the profile's own host, got %q", cfg.APIURL)
	}

	// config explain shows that host as the selected candidate.
	field := configExplainData()["api_url"].(configExplainField)
	var selected []string
	for _, c := range field.Candidates {
		if c.Selected {
			selected = append(selected, c.Value)
		}
	}
	if len(selected) != 1 || selected[0] != "https://fizzy.acme.example" {
		t.Errorf("explain: want the profile's host selected, got %v", selected)
	}
}

// The host follows the credential. A profile with no token of its own does
// not pull a token from another source over to its host.
func TestAnAccountNamedProfileWithoutATokenDoesNotSetTheHost(t *testing.T) {
	accountNamedProfiles(t)

	if err := resolveProfile(); err != nil {
		t.Fatalf("resolve profile: %v", err)
	}
	if cfg.APIURL != "https://app.fizzy.do" {
		t.Errorf("want the configured host kept, got %q", cfg.APIURL)
	}
}

// config explain shows what an untrusted local config sets, as ignored.
func TestConfigExplainShowsIgnoredLocalSettings(t *testing.T) {
	accountNamedProfiles(t)
	workDir := t.TempDir()
	config.SetTestWorkingDir(workDir)
	if err := os.WriteFile(filepath.Join(workDir, config.LocalConfigFile), []byte("api_url: https://other.example.com\ntoken: planted\n"), 0o600); err != nil {
		t.Fatalf("write local config: %v", err)
	}

	data := configExplainData()
	for _, name := range []string{"api_url", "token"} {
		var local configExplainCandidate
		for _, c := range data[name].(configExplainField).Candidates {
			if c.Source == "local config" {
				local = c
			}
		}
		if local.Selected || !strings.Contains(local.Value, "ignored: the file is not trusted") {
			t.Errorf("%s: got %#v", name, local)
		}
		if strings.Contains(local.Value, "planted") {
			t.Errorf("%s: the token is shown: %q", name, local.Value)
		}
	}
}

// Trusting a file that cannot be read is not a usage error: only a request
// that names no file is.
func TestConfigTrustErrorCodes(t *testing.T) {
	config.SetTestConfigDir(t.TempDir())
	config.SetTestWorkingDir(t.TempDir())
	defer config.ResetTestConfigDir()
	defer config.ResetTestWorkingDir()
	SetTestModeWithSDK(NewMockClient())
	defer resetTest()

	code := func(err error) string {
		var e *output.Error
		if !errors.As(err, &e) {
			t.Fatalf("want an output error, got %v", err)
		}
		return e.Code
	}
	if got := code(configTrustCmd.RunE(configTrustCmd, []string{})); got != output.CodeUsage {
		t.Errorf("no local config: got %q", got)
	}
	missing := filepath.Join(t.TempDir(), config.LocalConfigFile)
	if got := code(configTrustCmd.RunE(configTrustCmd, []string{missing})); got != output.CodeNotFound {
		t.Errorf("missing file: got %q", got)
	}
	bad := filepath.Join(t.TempDir(), config.LocalConfigFile)
	if err := os.WriteFile(bad, []byte("api_url: [\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if got := code(configTrustCmd.RunE(configTrustCmd, []string{bad})); got != output.CodeAPI {
		t.Errorf("unreadable config: got %q", got)
	}
}

// The trust commands only change local trust records, so the profile, account
// and token checks do not apply to them.
func TestTrustCommandsAreLocal(t *testing.T) {
	for cmd, want := range map[*cobra.Command]bool{configTrustCmd: true, configUntrustCmd: true, configShowCmd: false, rootCmd: false} {
		if got := isLocalTrustCommand(cmd); got != want {
			t.Errorf("%s: got %v", cmd.CommandPath(), got)
		}
	}
}
