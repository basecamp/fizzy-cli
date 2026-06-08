package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/basecamp/fizzy-cli/internal/config"
	version "github.com/hashicorp/go-version"
	"github.com/mattn/go-isatty"
	"gopkg.in/yaml.v3"
)

const fizzyUpdateRepo = "basecamp/fizzy-cli"

var gitDescribeSuffixRE = regexp.MustCompile(`\d+-\d+-g[a-f0-9]{8}$`)

var (
	updateHTTPClient     = &http.Client{Timeout: 5 * time.Second}
	updateCancel         context.CancelFunc
	updateMessage        chan *releaseInfo
	machineOutputChecker = IsMachineOutput
	terminalChecker      = isTerminal
)

type releaseInfo struct {
	Version     string    `json:"tag_name" yaml:"version"`
	URL         string    `json:"html_url" yaml:"url"`
	PublishedAt time.Time `json:"published_at" yaml:"published_at"`
}

type updateStateEntry struct {
	CheckedForUpdateAt time.Time   `yaml:"checked_for_update_at"`
	LatestRelease      releaseInfo `yaml:"latest_release"`
}

func startUpdateCheck() {
	if updateCancel != nil {
		updateCancel()
		updateCancel = nil
		updateMessage = nil
	}

	current := currentVersion()
	if !isUpdateableVersion(current) || !shouldCheckForUpdate() {
		return
	}

	stateDir, err := config.StateDir()
	if err != nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background()) //nolint:gosec // G118: cancel is retained and called after command execution
	updateCancel = cancel
	message := make(chan *releaseInfo, 1)
	updateMessage = message
	go func() {
		rel, err := checkForUpdate(ctx, updateHTTPClient, filepath.Join(stateDir, "state.yml"), current)
		if err != nil && cfgVerbose {
			fmt.Fprintf(os.Stderr, "warning: checking for update failed: %v\n", err)
		}
		message <- rel
	}()
}

func finishUpdateCheck() {
	if updateCancel == nil || updateMessage == nil {
		return
	}

	cancel := updateCancel
	message := updateMessage
	var rel *releaseInfo
	select {
	case rel = <-message:
	case <-time.After(200 * time.Millisecond):
	}
	cancel()
	updateCancel = nil
	updateMessage = nil
	if rel == nil {
		return
	}

	exe, _ := os.Executable()
	isHomebrew := isUnderHomebrew(exe)
	if isHomebrew && isRecentRelease(rel.PublishedAt) {
		return
	}

	fmt.Fprintf(os.Stderr, "\n\nA new release of Fizzy is available: %s → %s\n",
		strings.TrimPrefix(currentVersion(), "v"),
		strings.TrimPrefix(rel.Version, "v"))
	if isHomebrew {
		fmt.Fprintln(os.Stderr, "To upgrade, run: brew upgrade basecamp/tap/fizzy")
	} else {
		fmt.Fprintln(os.Stderr, "Upgrade with your package manager, or download it from:")
	}
	fmt.Fprintf(os.Stderr, "%s\n\n", rel.URL)
}

func shouldCheckForUpdate() bool {
	if os.Getenv("FIZZY_NO_UPDATE_NOTIFIER") != "" {
		return false
	}
	if os.Getenv("CODESPACES") != "" {
		return false
	}
	if isCI() {
		return false
	}
	if machineOutputChecker() {
		return false
	}
	return terminalChecker(os.Stderr)
}

func checkForUpdate(ctx context.Context, client *http.Client, stateFilePath, currentVersion string) (*releaseInfo, error) {
	stateEntry, err := getUpdateStateEntry(stateFilePath)
	if err != nil && !os.IsNotExist(err) {
		var pathErr *os.PathError
		if errors.As(err, &pathErr) {
			return nil, err
		}
		stateEntry = nil
	}
	if stateEntry != nil && time.Since(stateEntry.CheckedForUpdateAt).Hours() < 24 {
		if versionGreaterThan(stateEntry.LatestRelease.Version, currentVersion) {
			return &stateEntry.LatestRelease, nil
		}
		return nil, nil
	}

	rel, err := getLatestReleaseInfo(ctx, client, fizzyUpdateRepo)
	if err != nil {
		return nil, err
	}

	if err := setUpdateStateEntry(stateFilePath, time.Now(), *rel); err != nil {
		return nil, err
	}

	if versionGreaterThan(rel.Version, currentVersion) {
		return rel, nil
	}
	return nil, nil
}

func getLatestReleaseInfo(ctx context.Context, client *http.Client, repo string) (*releaseInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "fizzy-cli/"+currentVersion())

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, res.Body)
		_ = res.Body.Close()
	}()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP %d", res.StatusCode)
	}

	var rel releaseInfo
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

func getUpdateStateEntry(stateFilePath string) (*updateStateEntry, error) {
	content, err := os.ReadFile(stateFilePath)
	if err != nil {
		return nil, err
	}

	var stateEntry updateStateEntry
	if err := yaml.Unmarshal(content, &stateEntry); err != nil {
		return nil, err
	}
	return &stateEntry, nil
}

func setUpdateStateEntry(stateFilePath string, t time.Time, rel releaseInfo) error {
	content, err := yaml.Marshal(updateStateEntry{CheckedForUpdateAt: t, LatestRelease: rel})
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(stateFilePath), 0o700); err != nil {
		return err
	}
	return os.WriteFile(stateFilePath, content, 0o600)
}

func versionGreaterThan(v, w string) bool {
	w = gitDescribeSuffixRE.ReplaceAllStringFunc(w, func(m string) string {
		idx := strings.IndexRune(m, '-')
		n, _ := strconv.Atoi(m[:idx])
		return fmt.Sprintf("%d-pre.0", n+1)
	})

	vv, ve := version.NewVersion(v)
	vw, we := version.NewVersion(w)
	return ve == nil && we == nil && vv.GreaterThan(vw)
}

func isUpdateableVersion(v string) bool {
	v = strings.TrimSpace(v)
	if v == "" || v == "dev" || strings.Contains(v, "dirty") || strings.Contains(v, "-g") {
		return false
	}
	_, err := version.NewVersion(v)
	return err == nil
}

func isRecentRelease(publishedAt time.Time) bool {
	return !publishedAt.IsZero() && time.Since(publishedAt) < 24*time.Hour
}

func isUnderHomebrew(exePath string) bool {
	if exePath == "" {
		return false
	}
	brewExe, err := exec.LookPath("brew")
	if err != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	prefix, err := exec.CommandContext(ctx, brewExe, "--prefix").Output() //nolint:gosec // G204: brewExe comes from exec.LookPath
	if err != nil {
		return false
	}
	brewBinPrefix := filepath.Join(strings.TrimSpace(string(prefix)), "bin") + string(filepath.Separator)
	return strings.HasPrefix(exePath, brewBinPrefix)
}

func isCI() bool {
	for _, name := range []string{"CI", "GITHUB_ACTIONS", "BUILDKITE", "CIRCLECI", "GITLAB_CI", "JENKINS_URL", "TEAMCITY_VERSION", "TF_BUILD"} {
		if os.Getenv(name) != "" {
			return true
		}
	}
	return false
}

func isTerminal(f *os.File) bool {
	return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
}
