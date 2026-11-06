package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/basecamp/cli/credstore"
	"github.com/basecamp/cli/profile"
	"github.com/basecamp/fizzy-cli/internal/config"
)

// fakeFizzyRevocation is Fizzy's POST /oauth/revocation, answering status and
// recording each token it was asked to revoke.
type fakeFizzyRevocation struct {
	t      *testing.T
	status int
	mu     sync.Mutex
	tokens []string
}

func (f *fakeFizzyRevocation) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || r.URL.Path != "/oauth/revocation" {
		f.t.Errorf("unexpected HTTP request: %s %s", r.Method, r.URL.Path)
		http.NotFound(w, r)
		return
	}
	raw, _ := io.ReadAll(r.Body)
	form, _ := url.ParseQuery(string(raw))
	f.mu.Lock()
	f.tokens = append(f.tokens, form.Get("token"))
	f.mu.Unlock()
	w.WriteHeader(f.status)
}

func (f *fakeFizzyRevocation) revoked() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.tokens...)
}

type mintedTokenRig struct {
	server   *httptest.Server
	fizzy    *fakeFizzyRevocation
	store    *credstore.Store
	credDir  string
	profiles *profile.Store
}

// newMintedTokenRig points the credential, profile and config stores at
// temporary directories and the revocation at a fake Fizzy answering status.
func newMintedTokenRig(t *testing.T, status int) *mintedTokenRig {
	t.Helper()
	fizzy := &fakeFizzyRevocation{t: t, status: status}
	server := httptest.NewServer(fizzy)
	t.Cleanup(server.Close)

	configDir := t.TempDir()
	config.SetTestConfigDir(configDir)
	t.Cleanup(config.ResetTestConfigDir)
	t.Setenv("FIZZY_MINTED_TOKEN_NO_KR", "1")
	credDir := filepath.Join(t.TempDir(), "credentials")
	store := credstore.NewStore(credstore.StoreOptions{
		ServiceName:   "fizzy-minted-token-test",
		DisableEnvVar: "FIZZY_MINTED_TOKEN_NO_KR",
		FallbackDir:   credDir,
	})
	profileStore := profile.NewStore(filepath.Join(configDir, "config.json"))

	SetTestModeWithSDK(NewMockClient())
	SetTestCreds(store)
	SetTestProfiles(profileStore)
	t.Cleanup(resetTest)

	original := mintedTokenRevocationClient
	mintedTokenRevocationClient = func() *http.Client { return server.Client() }
	t.Cleanup(func() { mintedTokenRevocationClient = original })

	return &mintedTokenRig{server: server, fizzy: fizzy, store: store, credDir: credDir, profiles: profileStore}
}

func (r *mintedTokenRig) logout(t *testing.T, all bool) map[string]any {
	t.Helper()
	flag := "false"
	if all {
		flag = "true"
	}
	authLogoutCmd.Flags().Set("all", flag)
	t.Cleanup(func() { authLogoutCmd.Flags().Set("all", "false") })
	if err := authLogoutCmd.RunE(authLogoutCmd, nil); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if lastResult == nil || lastResult.Response == nil {
		t.Fatal("logout printed no response")
	}
	data, _ := lastResult.Response.Data.(map[string]any)
	return data
}

// captureStderr returns what fn writes to os.Stderr, restoring it even when fn
// fails the test.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	oldStderr := os.Stderr
	os.Stderr = w
	t.Cleanup(func() { os.Stderr = oldStderr })
	done := make(chan []byte)
	go func() { b, _ := io.ReadAll(r); done <- b }()
	fn()
	os.Stderr = oldStderr
	w.Close()
	return string(<-done)
}

func TestLogoutRevokesTheTokenSignupCreated(t *testing.T) {
	rig := newMintedTokenRig(t, http.StatusOK)
	if err := saveSignupConfig("minted-token", "acct", rig.server.URL); err != nil {
		t.Fatalf("save signup config: %v", err)
	}
	p, err := rig.profiles.Get("acct")
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if strings.Contains(string(p.Extra[mintedTokenExtraKey]), "minted-token") {
		t.Fatal("the profile records the token itself, not its digest")
	}
	SetTestConfig("minted-token", "acct", rig.server.URL)

	data := rig.logout(t, false)
	if got := rig.fizzy.revoked(); len(got) != 1 || got[0] != "minted-token" {
		t.Errorf("revoked %d tokens, want the signup token once", len(got))
	}
	if data["revoked"] != true {
		t.Errorf("logout data = %v, want revoked", data)
	}
	if _, err := rig.store.Load("profile:acct"); err == nil {
		t.Error("the token is still stored after logout")
	}
}

// A token handed to `fizzy auth login` may be in use elsewhere: logout only
// forgets it. So does a profile whose signup token was replaced by a login.
func TestLogoutOnlyForgetsATokenTheCLIDidNotCreate(t *testing.T) {
	for name, setup := range map[string]func(*testing.T, *mintedTokenRig){
		"pasted token": func(t *testing.T, rig *mintedTokenRig) {
			if _, err := saveProfileCredentialState(profileCredentialSaveOptions{
				ProfileName: "acct", Account: "acct", BaseURL: rig.server.URL, Token: "pasted-token",
			}); err != nil {
				t.Fatal(err)
			}
		},
		"signup token replaced by a login": func(t *testing.T, rig *mintedTokenRig) {
			if err := saveSignupConfig("minted-token", "acct", rig.server.URL); err != nil {
				t.Fatal(err)
			}
			if _, err := saveProfileCredentialState(profileCredentialSaveOptions{
				ProfileName: "acct", Account: "acct", BaseURL: rig.server.URL, Token: "pasted-token",
			}); err != nil {
				t.Fatal(err)
			}
		},
		"stale mark beside a pasted token": func(t *testing.T, rig *mintedTokenRig) {
			// A mark left behind by a path that stores a token without
			// resetting it matches nothing.
			mark, _ := json.Marshal(mintedTokenDigest("minted-token"))
			if err := rig.profiles.Create(&profile.Profile{
				Name: "acct", BaseURL: rig.server.URL, Extra: map[string]json.RawMessage{mintedTokenExtraKey: mark},
			}); err != nil {
				t.Fatal(err)
			}
			if err := credsSaveProfileToken("acct", "pasted-token"); err != nil {
				t.Fatal(err)
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			rig := newMintedTokenRig(t, http.StatusOK)
			setup(t, rig)
			SetTestConfig("pasted-token", "acct", rig.server.URL)

			data := rig.logout(t, false)
			if got := rig.fizzy.revoked(); len(got) != 0 {
				t.Errorf("revoked %d tokens, want none", len(got))
			}
			if data["revoked"] != false {
				t.Errorf("logout data = %v, want revoked false", data)
			}
			if _, err := rig.store.Load("profile:acct"); err == nil {
				t.Error("the token is still stored after logout")
			}
		})
	}
}

// A Fizzy that can't confirm the revocation (an outage, a server without the
// endpoint) still gets the token forgotten here, and the failure reported.
func TestLogoutForgetsTheSignupTokenWhenRevocationFails(t *testing.T) {
	for _, status := range []int{http.StatusInternalServerError, http.StatusNotFound} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			rig := newMintedTokenRig(t, status)
			if err := saveSignupConfig("minted-token", "acct", rig.server.URL); err != nil {
				t.Fatal(err)
			}
			SetTestConfig("minted-token", "acct", rig.server.URL)

			var data map[string]any
			stderr := captureStderr(t, func() { data = rig.logout(t, false) })

			if data["revoked"] != false || data["revoke_error"] == nil {
				t.Errorf("logout data = %v, want revoked false with the error", data)
			}
			if !strings.Contains(stderr, "could not revoke the token fizzy signup created") {
				t.Errorf("stderr = %q, want the revocation warning", stderr)
			}
			if strings.Contains(stderr, "minted-token") || strings.Contains(fmt.Sprint(data["revoke_error"]), "minted-token") {
				t.Error("the warning printed the token")
			}
			if _, err := rig.store.Load("profile:acct"); err == nil {
				t.Error("the token is still stored after logout")
			}
		})
	}
}

func TestLogoutAllRevokesEverySignupToken(t *testing.T) {
	rig := newMintedTokenRig(t, http.StatusOK)
	if err := saveSignupConfig("minted-one", "one", rig.server.URL); err != nil {
		t.Fatal(err)
	}
	if err := saveSignupConfig("minted-two", "two", rig.server.URL); err != nil {
		t.Fatal(err)
	}
	if _, err := saveProfileCredentialState(profileCredentialSaveOptions{
		ProfileName: "three", Account: "three", BaseURL: rig.server.URL, Token: "pasted-three",
	}); err != nil {
		t.Fatal(err)
	}
	SetTestConfig("pasted-three", "three", rig.server.URL)

	data := rig.logout(t, true)
	got := strings.Join(rig.fizzy.revoked(), ",")
	if len(rig.fizzy.revoked()) != 2 || !strings.Contains(got, "minted-one") || !strings.Contains(got, "minted-two") {
		t.Errorf("revoked %d tokens, want both signup tokens and not the pasted one", len(rig.fizzy.revoked()))
	}
	if data["revoked"] != true {
		t.Errorf("logout data = %v, want revoked", data)
	}
}

// One revocation through and another refused is not a revoked logout.
func TestLogoutAllReportsAPartialRevocationAsNotRevoked(t *testing.T) {
	ok := newMintedTokenRig(t, http.StatusOK)
	refusing := &fakeFizzyRevocation{t: t, status: http.StatusInternalServerError}
	refusingServer := httptest.NewServer(refusing)
	defer refusingServer.Close()
	if err := saveSignupConfig("minted-one", "one", ok.server.URL); err != nil {
		t.Fatal(err)
	}
	if err := saveSignupConfig("minted-two", "two", refusingServer.URL); err != nil {
		t.Fatal(err)
	}
	mintedTokenRevocationClient = func() *http.Client { return &http.Client{} }
	SetTestConfig("minted-two", "two", refusingServer.URL)

	var data map[string]any
	stderr := captureStderr(t, func() { data = ok.logout(t, true) })
	if len(ok.fizzy.revoked()) != 1 || len(refusing.revoked()) != 1 {
		t.Fatalf("revocations = %d and %d, want one at each Fizzy", len(ok.fizzy.revoked()), len(refusing.revoked()))
	}
	if data["revoked"] != false || !strings.Contains(fmt.Sprint(data["revoke_error"]), "two:") {
		t.Errorf("logout data = %v, want revoked false naming profile two", data)
	}
	if !strings.Contains(stderr, `"two"`) {
		t.Errorf("stderr = %q, want a warning for profile two", stderr)
	}
}

// The token goes to the Fizzy that minted it: a redirect is a failed
// revocation, and its target hears nothing.
func TestLogoutDoesNotFollowARevocationRedirect(t *testing.T) {
	rig := newMintedTokenRig(t, http.StatusOK)
	target := &fakeFizzyRevocation{t: t, status: http.StatusOK}
	targetServer := httptest.NewServer(target)
	defer targetServer.Close()
	redirecting := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, targetServer.URL+"/oauth/revocation", http.StatusTemporaryRedirect)
	}))
	defer redirecting.Close()
	// The production client, not the rig's: its redirect policy is under test.
	mintedTokenRevocationClient = newMintedTokenRevocationClient
	if err := saveSignupConfig("minted-token", "acct", redirecting.URL); err != nil {
		t.Fatal(err)
	}
	SetTestConfig("minted-token", "acct", redirecting.URL)

	var data map[string]any
	captureStderr(t, func() { data = rig.logout(t, false) })
	if data["revoked"] != false || data["revoke_error"] == nil {
		t.Errorf("logout data = %v, want revoked false with the error", data)
	}
	if len(target.revoked()) != 0 {
		t.Error("the redirect target received the token")
	}
}

// A credential store that can't be read may still hold a live signup token:
// that is a failed revocation, warned about, not a token that isn't there.
// The warning comes before the cleanup, which fails here too.
func TestLogoutWarnsWhenTheSignupTokenCannotBeRead(t *testing.T) {
	rig := newMintedTokenRig(t, http.StatusOK)
	if err := saveSignupConfig("minted-token", "acct", rig.server.URL); err != nil {
		t.Fatal(err)
	}
	SetTestConfig("minted-token", "acct", rig.server.URL)
	backup := rig.credDir + "-backup"
	if err := os.Rename(rig.credDir, backup); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rig.credDir, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(rig.credDir); os.Rename(backup, rig.credDir) })

	var err error
	stderr := captureStderr(t, func() {
		authLogoutCmd.Flags().Set("all", "false")
		err = authLogoutCmd.RunE(authLogoutCmd, nil)
	})
	if len(rig.fizzy.revoked()) != 0 {
		t.Errorf("revoked %d tokens, want none", len(rig.fizzy.revoked()))
	}
	if !strings.Contains(stderr, "could not revoke the token fizzy signup created") {
		t.Errorf("stderr = %q, want the revocation warning (logout error: %v)", stderr, err)
	}
}

// The config-file copy signup falls back to still counts when the
// credential store can't be read.
func TestLogoutRevokesTheConfigFileCopyWhenTheStoreCannotBeRead(t *testing.T) {
	rig := newMintedTokenRig(t, http.StatusOK)
	if err := saveSignupConfig("minted-token", "acct", rig.server.URL); err != nil {
		t.Fatal(err)
	}
	global := config.LoadGlobal()
	global.Token = "minted-token"
	if err := global.Save(); err != nil {
		t.Fatal(err)
	}
	SetTestConfig("minted-token", "acct", rig.server.URL)
	backup := rig.credDir + "-backup"
	if err := os.Rename(rig.credDir, backup); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rig.credDir, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(rig.credDir); os.Rename(backup, rig.credDir) })

	stderr := captureStderr(t, func() {
		authLogoutCmd.Flags().Set("all", "false")
		_ = authLogoutCmd.RunE(authLogoutCmd, nil)
	})
	if got := rig.fizzy.revoked(); len(got) != 1 || got[0] != "minted-token" {
		t.Errorf("revoked %d tokens, want the config-file copy once", len(got))
	}
	if strings.Contains(stderr, "could not revoke") {
		t.Errorf("stderr = %q, want no revocation warning", stderr)
	}
}
