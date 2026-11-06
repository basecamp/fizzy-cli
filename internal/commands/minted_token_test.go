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
	store := credstore.NewStore(credstore.StoreOptions{
		ServiceName:   "fizzy-minted-token-test",
		DisableEnvVar: "FIZZY_MINTED_TOKEN_NO_KR",
		FallbackDir:   t.TempDir(),
	})
	profileStore := profile.NewStore(filepath.Join(configDir, "config.json"))

	SetTestModeWithSDK(NewMockClient())
	SetTestCreds(store)
	SetTestProfiles(profileStore)
	t.Cleanup(resetTest)

	original := mintedTokenRevocationClient
	mintedTokenRevocationClient = func() *http.Client { return server.Client() }
	t.Cleanup(func() { mintedTokenRevocationClient = original })

	return &mintedTokenRig{server: server, fizzy: fizzy, store: store, profiles: profileStore}
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

			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w
			data := rig.logout(t, false)
			w.Close()
			os.Stderr = oldStderr
			stderr, _ := io.ReadAll(r)

			if data["revoked"] != false || data["revoke_error"] == nil {
				t.Errorf("logout data = %v, want revoked false with the error", data)
			}
			if !strings.Contains(string(stderr), "could not revoke the token fizzy signup created") {
				t.Errorf("stderr = %q, want the revocation warning", stderr)
			}
			if strings.Contains(string(stderr), "minted-token") || strings.Contains(fmt.Sprint(data["revoke_error"]), "minted-token") {
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
