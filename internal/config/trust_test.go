package config

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// trustLocalConfig trusts the nearest local config, as `fizzy config trust` does.
func trustLocalConfig(t *testing.T) {
	t.Helper()
	if _, err := TrustLocal(""); err != nil {
		t.Fatalf("trust local config: %v", err)
	}
}

// localConfigHere points config loading at a fresh global config directory and
// a working directory holding a .fizzy.yaml with contents, and returns the
// file's path.
func localConfigHere(t *testing.T, contents string) string {
	t.Helper()
	for _, name := range []string{"FIZZY_TOKEN", "FIZZY_ACCOUNT", "FIZZY_API_URL", "FIZZY_BOARD"} {
		t.Setenv(name, "")
	}
	SetTestConfigDir(t.TempDir())
	t.Cleanup(ResetTestConfigDir)
	workDir := t.TempDir()
	SetTestWorkingDir(workDir)
	t.Cleanup(ResetTestWorkingDir)

	path := filepath.Join(workDir, LocalConfigFile)
	writeLocalConfig(t, path, contents)
	return path
}

func writeLocalConfig(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write local config: %v", err)
	}
}

// loadWithWarnings is Load and what it wrote to stderr.
func loadWithWarnings(t *testing.T) (*Config, string) {
	t.Helper()
	warnedUntrustedLocal.Clear()
	orig := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	defer r.Close()
	os.Stderr = w
	cfg := func() *Config {
		defer func() {
			os.Stderr = orig
			w.Close()
		}()
		return Load()
	}()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return cfg, buf.String()
}

// A local config that nobody trusted, as a cloned repo's is, sets the account
// and the board. Its api_url and token are ignored: they decide where a
// credential goes and which one.
func TestAnUntrustedLocalConfigSetsNoAPIURLOrToken(t *testing.T) {
	path := localConfigHere(t, "account: acme\nboard: roadmap\napi_url: https://evil.example.com\ntoken: planted\n")

	cfg, warning := loadWithWarnings(t)

	if cfg.APIURL != DefaultAPIURL {
		t.Errorf("api_url: want the default, got %q", cfg.APIURL)
	}
	if cfg.Token != "" {
		t.Errorf("token: want none, got %q", cfg.Token)
	}
	if cfg.Account != "acme" || cfg.Board != "roadmap" {
		t.Errorf("account and board apply without trust, got %q and %q", cfg.Account, cfg.Board)
	}
	for _, want := range []string{"api_url and token", path, "fizzy config trust"} {
		if !strings.Contains(warning, want) {
			t.Errorf("warning %q does not name %q", warning, want)
		}
	}
	if strings.Contains(warning, "planted") {
		t.Errorf("warning %q prints the token", warning)
	}
}

// Trust is given to what the file says, not where it is. A pull request that
// rewrites a trusted local config to name another host leaves it untrusted.
func TestARewrittenTrustedLocalConfigIsUntrusted(t *testing.T) {
	path := localConfigHere(t, "account: acme\napi_url: https://fizzy.example.com\n")
	trustLocalConfig(t)

	cfg, warning := loadWithWarnings(t)
	if cfg.APIURL != "https://fizzy.example.com" || warning != "" {
		t.Fatalf("a trusted api_url applies quietly, got %q and warning %q", cfg.APIURL, warning)
	}

	writeLocalConfig(t, path, "account: acme\napi_url: https://evil.example.com\n")
	cfg, warning = loadWithWarnings(t)
	if cfg.APIURL != DefaultAPIURL {
		t.Errorf("the rewritten api_url is ignored, got %q", cfg.APIURL)
	}
	if !strings.Contains(warning, "not trusted") {
		t.Errorf("want a warning, got %q", warning)
	}

	trustLocalConfig(t)
	if cfg, _ = loadWithWarnings(t); cfg.APIURL != "https://evil.example.com" {
		t.Errorf("until it is trusted again, got %q", cfg.APIURL)
	}
	if got := len(TrustedLocals()); got != 1 {
		t.Errorf("trusting again updates the entry, got %d entries", got)
	}
}

// A token added to a trusted file is as much a change as a new host.
func TestATokenAddedToATrustedLocalConfigIsUntrusted(t *testing.T) {
	path := localConfigHere(t, "api_url: https://fizzy.example.com\n")
	trustLocalConfig(t)

	writeLocalConfig(t, path, "api_url: https://fizzy.example.com\ntoken: planted\n")
	cfg, _ := loadWithWarnings(t)
	if cfg.Token != "" || cfg.APIURL != DefaultAPIURL {
		t.Errorf("want neither applied, got token %q and api_url %q", cfg.Token, cfg.APIURL)
	}
}

// Changing what an untrusted file may set anyway, or its layout, keeps trust.
func TestChangingAccountOrBoardKeepsTrust(t *testing.T) {
	path := localConfigHere(t, "account: acme\napi_url: https://fizzy.example.com\n")
	trustLocalConfig(t)

	writeLocalConfig(t, path, "# moved\nboard: roadmap\napi_url: \"https://fizzy.example.com\"\naccount: other\n")
	cfg, warning := loadWithWarnings(t)
	if cfg.APIURL != "https://fizzy.example.com" || warning != "" {
		t.Errorf("want the api_url still applied, got %q and warning %q", cfg.APIURL, warning)
	}
}

// A file with no api_url and no token has nothing to trust, and says nothing.
func TestALocalConfigWithoutGatedKeysNeedsNoTrust(t *testing.T) {
	localConfigHere(t, "account: acme\nboard: roadmap\n")

	cfg, warning := loadWithWarnings(t)
	if cfg.Account != "acme" || cfg.Board != "roadmap" || warning != "" {
		t.Errorf("got account %q, board %q, warning %q", cfg.Account, cfg.Board, warning)
	}
}

func TestUntrustLocalRemovesTrust(t *testing.T) {
	path := localConfigHere(t, "api_url: https://fizzy.example.com\n")
	trustLocalConfig(t)

	if _, removed, err := UntrustLocal(""); err != nil || !removed {
		t.Fatalf("untrust: removed %v, err %v", removed, err)
	}
	if cfg, _ := loadWithWarnings(t); cfg.APIURL != DefaultAPIURL {
		t.Errorf("api_url is ignored again, got %q", cfg.APIURL)
	}
	if _, removed, err := UntrustLocal(path); err != nil || removed {
		t.Errorf("untrusting twice: removed %v, err %v", removed, err)
	}
}

// Trust follows the file, not the name it was reached by.
func TestTrustFollowsASymlinkedPath(t *testing.T) {
	path := localConfigHere(t, "api_url: https://fizzy.example.com\n")
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(filepath.Dir(path), link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := TrustLocal(filepath.Join(link, LocalConfigFile)); err != nil {
		t.Fatalf("trust: %v", err)
	}
	if cfg, _ := loadWithWarnings(t); cfg.APIURL != "https://fizzy.example.com" {
		t.Errorf("want the api_url applied, got %q", cfg.APIURL)
	}
}

func TestTrustLocalWithoutALocalConfigFails(t *testing.T) {
	SetTestConfigDir(t.TempDir())
	t.Cleanup(ResetTestConfigDir)
	SetTestWorkingDir(t.TempDir())
	t.Cleanup(ResetTestWorkingDir)

	if _, err := TrustLocal(""); err == nil {
		t.Error("want an error")
	}
}

// The trust store stays in the preferred config directory: it does not follow
// the global config file to a legacy directory, where removing that file
// would leave every trusted entry behind.
func TestTheTrustStoreDoesNotFollowALegacyGlobalConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	legacy := filepath.Join(home, ".fizzy", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(legacy), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeLocalConfig(t, legacy, "account: acme\n")

	want := filepath.Join(home, ".config", "fizzy", trustFileName)
	if got, err := trustFilePath(); err != nil || got != want {
		t.Fatalf("with a legacy global config: got %q, %v", got, err)
	}
	if err := os.Remove(legacy); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if got, err := trustFilePath(); err != nil || got != want {
		t.Fatalf("without it: got %q, %v", got, err)
	}
}

// Commands that trust and untrust at the same time each keep what the others
// wrote: none saves a copy of the store it read before another's change.
func TestConcurrentTrustChangesKeepEachOther(t *testing.T) {
	localConfigHere(t, "account: acme\n")
	const files = 8
	paths := make([]string, files)
	for i := range paths {
		paths[i] = filepath.Join(t.TempDir(), LocalConfigFile)
		writeLocalConfig(t, paths[i], "api_url: https://fizzy.example.com\n")
	}

	errs := make(chan error, files)
	for _, path := range paths {
		go func() {
			_, err := TrustLocal(path)
			errs <- err
		}()
	}
	for range paths {
		if err := <-errs; err != nil {
			t.Fatalf("trust: %v", err)
		}
	}
	if got := len(TrustedLocals()); got != files {
		t.Fatalf("want %d trusted files, got %d", files, got)
	}
	if left, _ := filepath.Glob(filepath.Join(testConfigDir, trustFileName+".*")); len(left) != 0 {
		t.Fatalf("temporary or lock files left behind: %v", left)
	}
}

// A trusted file reached through a link is untrusted by that link even after
// its target is gone.
func TestUntrustThroughALinkWhoseTargetIsGone(t *testing.T) {
	localConfigHere(t, "account: acme\n")
	target := filepath.Join(t.TempDir(), LocalConfigFile)
	writeLocalConfig(t, target, "api_url: https://fizzy.example.com\n")
	link := filepath.Join(t.TempDir(), LocalConfigFile)
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if _, err := TrustLocal(link); err != nil {
		t.Fatalf("trust: %v", err)
	}
	if err := os.Remove(target); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, removed, err := UntrustLocal(link); err != nil || !removed {
		t.Fatalf("untrust: removed=%v, %v", removed, err)
	}
	if got := TrustedLocals(); len(got) != 0 {
		t.Fatalf("trust left behind: %#v", got)
	}
}

func TestDisplayPathDropsControlCharacters(t *testing.T) {
	if got := DisplayPath("/tmp/a\nTrusted /etc\x1b[2K/.fizzy.yaml"); got != "/tmp/aTrusted /etc[2K/.fizzy.yaml" {
		t.Fatalf("got %q", got)
	}
}
