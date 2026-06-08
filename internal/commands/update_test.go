package commands

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestCheckForUpdateFindsNewRelease(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "state.yml")
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != "https://api.github.com/repos/basecamp/fizzy-cli/releases/latest" {
			t.Fatalf("unexpected URL %s", req.URL.String())
		}
		if ua := req.Header.Get("User-Agent"); ua != "fizzy-cli/"+currentVersion() {
			t.Fatalf("User-Agent = %q, want fizzy-cli/%s", ua, currentVersion())
		}
		body := `{"tag_name":"v3.0.3","html_url":"https://github.com/basecamp/fizzy-cli/releases/tag/v3.0.3","published_at":"2026-03-02T21:19:42Z"}`
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}

	rel, err := checkForUpdate(context.Background(), client, stateFile, "v3.0.2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rel == nil || rel.Version != "v3.0.3" {
		t.Fatalf("release = %#v, want v3.0.3", rel)
	}
	if _, err := os.Stat(stateFile); err != nil {
		t.Fatalf("state file not written: %v", err)
	}
}

func TestCheckForUpdateRecoversCorruptState(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "state.yml")
	if err := os.WriteFile(stateFile, []byte("checked_for_update_at: ["), 0o600); err != nil {
		t.Fatalf("failed to write state: %v", err)
	}

	called := false
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		called = true
		body := `{"tag_name":"v3.0.3","html_url":"https://github.com/basecamp/fizzy-cli/releases/tag/v3.0.3","published_at":"2026-03-02T21:19:42Z"}`
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}

	rel, err := checkForUpdate(context.Background(), client, stateFile, "v3.0.2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected corrupt state to fetch latest release")
	}
	if rel == nil || rel.Version != "v3.0.3" {
		t.Fatalf("release = %#v, want v3.0.3", rel)
	}
}

func TestCheckForUpdateUsesRecentCachedRelease(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "state.yml")
	err := setUpdateStateEntry(stateFile, time.Now().Add(-time.Hour), releaseInfo{Version: "v3.0.3"})
	if err != nil {
		t.Fatalf("failed to write state: %v", err)
	}

	called := false
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		called = true
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
	})}

	rel, err := checkForUpdate(context.Background(), client, stateFile, "v3.0.2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rel == nil || rel.Version != "v3.0.3" {
		t.Fatalf("release = %#v, want cached v3.0.3", rel)
	}
	if called {
		t.Fatal("expected recent state to skip HTTP request")
	}
}

func TestCheckForUpdateSkipsCurrentCachedRelease(t *testing.T) {
	stateFile := filepath.Join(t.TempDir(), "state.yml")
	err := setUpdateStateEntry(stateFile, time.Now().Add(-time.Hour), releaseInfo{Version: "v3.0.3"})
	if err != nil {
		t.Fatalf("failed to write state: %v", err)
	}

	called := false
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		called = true
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
	})}

	rel, err := checkForUpdate(context.Background(), client, stateFile, "v3.0.3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rel != nil {
		t.Fatalf("release = %#v, want nil", rel)
	}
	if called {
		t.Fatal("expected recent state to skip HTTP request")
	}
}

func TestVersionGreaterThan(t *testing.T) {
	tests := []struct {
		latest  string
		current string
		want    bool
	}{
		{"v3.0.3", "v3.0.2", true},
		{"v3.0.3", "v3.0.3", false},
		{"v3.0.2", "v3.0.3", false},
		{"v3.1.0", "v3.1.0-2-gabcdef12", false},
		{"v3.1.1", "v3.1.0-2-gabcdef12", true},
		{"v3.0.3", "dev", false},
	}

	for _, tt := range tests {
		if got := versionGreaterThan(tt.latest, tt.current); got != tt.want {
			t.Fatalf("versionGreaterThan(%q, %q) = %v, want %v", tt.latest, tt.current, got, tt.want)
		}
	}
}

func TestShouldCheckForUpdateHonorsOptOut(t *testing.T) {
	for _, name := range []string{"CODESPACES", "CI", "GITHUB_ACTIONS", "BUILDKITE", "CIRCLECI", "GITLAB_CI", "JENKINS_URL", "TEAMCITY_VERSION", "TF_BUILD"} {
		t.Setenv(name, "")
	}
	machineOutputChecker = func() bool { return false }
	terminalChecker = func(*os.File) bool { return true }
	defer func() {
		machineOutputChecker = IsMachineOutput
		terminalChecker = isTerminal
	}()

	if !shouldCheckForUpdate() {
		t.Fatal("expected update checks to be enabled before opt-out")
	}

	t.Setenv("FIZZY_NO_UPDATE_NOTIFIER", "1")
	if shouldCheckForUpdate() {
		t.Fatal("expected FIZZY_NO_UPDATE_NOTIFIER to disable update checks")
	}
}
