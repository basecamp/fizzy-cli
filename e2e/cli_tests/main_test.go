// Package clitests contains owner-only end-to-end tests for the Fizzy CLI.
//
// Required environment variables:
//   - FIZZY_TEST_TOKEN
//   - FIZZY_TEST_ACCOUNT
//
// Optional:
//   - FIZZY_TEST_API_URL
//   - FIZZY_TEST_BINARY
package clitests

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/basecamp/fizzy-cli/e2e/harness"
)

var (
	cfg     *harness.Config
	fixture *harness.SharedFixture
)

func TestMain(m *testing.M) {
	os.Exit(runMain(m))
}

func runMain(m *testing.M) int {
	cfg = harness.LoadConfig()
	if missing := cfg.MissingVars(); len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "Skipping CLI e2e tests — missing env vars: %v\n", missing)
		fmt.Fprintln(os.Stderr, "Set FIZZY_TEST_TOKEN and FIZZY_TEST_ACCOUNT to run these tests.")
		return 0
	}

	if !fileExists(cfg.BinaryPath) {
		repoRoot, err := harness.RepoRoot()
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return 1
		}
		tmpDir, err := os.MkdirTemp("", "fizzy-e2e-cli-build-*")
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return 1
		}
		defer os.RemoveAll(tmpDir)
		binPath := filepath.Join(tmpDir, "fizzy")
		cmd := exec.Command("go", "build", "-o", binPath, "./cmd/fizzy")
		cmd.Dir = repoRoot
		if out, err := cmd.CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "failed to build binary: %v\n%s\n", err, string(out))
			return 1
		}
		_ = os.Setenv("FIZZY_TEST_BINARY", binPath)
		cfg.BinaryPath = binPath
	}

	var err error
	fixture, err = harness.NewSharedFixture(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fixture setup failed: %v\n", err)
		return 1
	}
	printFixtureInfo()

	code := m.Run()

	if os.Getenv("FIZZY_E2E_KEEP_FIXTURE") == "1" {
		fmt.Fprintln(os.Stderr, "Keeping CLI e2e fixture (FIZZY_E2E_KEEP_FIXTURE=1)")
		printFixtureInfo()
		return code
	}
	if delay := teardownDelay(); delay > 0 {
		fmt.Fprintf(os.Stderr, "Delaying CLI e2e fixture teardown for %s\n", delay)
		printFixtureInfo()
		time.Sleep(delay)
	}
	if err := fixture.Teardown(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: fixture teardown error: %v\n", err)
	}
	return code
}

func printFixtureInfo() {
	if fixture == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "CLI e2e fixture board: %s\n", fixture.BoardID)
	if fixture.BoardID != "" {
		fmt.Fprintf(os.Stderr, "CLI e2e fixture board URL: %s/%s/boards/%s\n", strings.TrimRight(cfg.APIURL, "/"), cfg.Account, fixture.BoardID)
	}
	if fixture.CardNumber != 0 {
		fmt.Fprintf(os.Stderr, "CLI e2e fixture card URL: %s/%s/cards/%d\n", strings.TrimRight(cfg.APIURL, "/"), cfg.Account, fixture.CardNumber)
	}
}

func teardownDelay() time.Duration {
	raw := os.Getenv("FIZZY_E2E_TEARDOWN_DELAY")
	if raw == "" {
		return 0
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds <= 0 {
		fmt.Fprintf(os.Stderr, "ignoring invalid FIZZY_E2E_TEARDOWN_DELAY=%q\n", raw)
		return 0
	}
	return time.Duration(seconds) * time.Second
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}
