package harness

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// SharedFixture is a pre-built world state created once per owner-only CLI
// test run. Resources are created via Execute() directly (no *testing.T
// required) so the fixture can be set up inside TestMain.
type SharedFixture struct {
	// configHome is an isolated temp directory used as HOME so the CLI reads
	// no config from the developer's real home directory.
	configHome string

	// BoardID is the root board for CLI tests.
	BoardID string

	// ColumnID is a custom column on BoardID.
	ColumnID string

	// CardNumber is a card created on BoardID.
	CardNumber int

	// CommentID is a comment left on CardNumber.
	CommentID string

	// StepID is a step on CardNumber.
	StepID string

	cfg *Config
}

// NewSharedFixture builds the shared fixture using the provided credentials.
// Returns an error describing the first setup step that fails.
func NewSharedFixture(cfg *Config) (*SharedFixture, error) {
	tmpDir, err := os.MkdirTemp("", "fizzy-e2e-fixture-*")
	if err != nil {
		return nil, fmt.Errorf("create fixture config home: %w", err)
	}
	f := &SharedFixture{cfg: cfg, configHome: tmpDir}
	if err := f.setup(); err != nil {
		if teardownErr := f.Teardown(); teardownErr != nil {
			_ = os.RemoveAll(tmpDir)
			return nil, fmt.Errorf("%w\ncleanup: %v", err, teardownErr)
		}
		return nil, err
	}
	return f, nil
}

// Teardown removes all fixture resources. Deleting the board cascades to all
// child resources (cards, columns, comments, steps, reactions).
func (f *SharedFixture) Teardown() error {
	var errs []string
	if f.BoardID != "" {
		r := f.run("board", "delete", f.BoardID)
		if r.ExitCode != ExitSuccess && r.ExitCode != ExitNotFound {
			errs = append(errs, fmt.Sprintf("delete fixture board %s: exit %d\nstderr: %s", f.BoardID, r.ExitCode, r.Stderr))
		}
	}
	if err := os.RemoveAll(f.configHome); err != nil {
		errs = append(errs, fmt.Sprintf("remove fixture config home %s: %v", f.configHome, err))
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "\n"))
	}
	return nil
}

func (f *SharedFixture) setup() error {
	boardName := fmt.Sprintf("CLI E2E Board %d", time.Now().UnixNano())
	r := f.run("board", "create", "--name", boardName)
	if r.ExitCode != ExitSuccess {
		return fmt.Errorf("create board: exit %d\nstderr: %s", r.ExitCode, r.Stderr)
	}
	f.BoardID = r.GetIDFromLocation()
	if f.BoardID == "" {
		f.BoardID = r.GetDataString("id")
	}
	if f.BoardID == "" {
		return fmt.Errorf("no board ID in create response (location: %q)", r.GetLocation())
	}

	r = f.run("column", "create", "--board", f.BoardID, "--name", "In Progress")
	if r.ExitCode != ExitSuccess {
		return fmt.Errorf("create column: exit %d\nstderr: %s", r.ExitCode, r.Stderr)
	}
	f.ColumnID = r.GetIDFromLocation()
	if f.ColumnID == "" {
		f.ColumnID = r.GetDataString("id")
	}
	if f.ColumnID == "" {
		return fmt.Errorf("no column ID in create response")
	}

	r = f.run("card", "create", "--board", f.BoardID, "--title", "Owner Test Card")
	if r.ExitCode != ExitSuccess {
		return fmt.Errorf("create card: exit %d\nstderr: %s", r.ExitCode, r.Stderr)
	}
	f.CardNumber = r.GetNumberFromLocation()
	if f.CardNumber == 0 {
		f.CardNumber = r.GetDataInt("number")
	}
	if f.CardNumber == 0 {
		return fmt.Errorf("no card number in create response")
	}

	r = f.run("comment", "create",
		"--card", strconv.Itoa(f.CardNumber),
		"--body", "Owner test comment")
	if r.ExitCode != ExitSuccess {
		return fmt.Errorf("create comment: exit %d\nstderr: %s", r.ExitCode, r.Stderr)
	}
	f.CommentID = r.GetIDFromLocation()
	if f.CommentID == "" {
		f.CommentID = r.GetDataString("id")
	}
	if f.CommentID == "" {
		return fmt.Errorf("no comment ID in create response")
	}

	r = f.run("step", "create",
		"--card", strconv.Itoa(f.CardNumber),
		"--content", "Test step")
	if r.ExitCode != ExitSuccess {
		return fmt.Errorf("create step: exit %d\nstderr: %s", r.ExitCode, r.Stderr)
	}
	f.StepID = r.GetIDFromLocation()
	if f.StepID == "" {
		f.StepID = r.GetDataString("id")
	}
	if f.StepID == "" {
		return fmt.Errorf("no step ID in create response")
	}

	return nil
}

func (f *SharedFixture) run(args ...string) *Result {
	fullArgs := make([]string, len(args), len(args)+4)
	copy(fullArgs, args)
	fullArgs = append(fullArgs, "--token", f.cfg.Token, "--api-url", f.cfg.APIURL)
	env := map[string]string{
		"FIZZY_PROFILE":    f.cfg.Account,
		"FIZZY_NO_KEYRING": "1",
		"HOME":             f.configHome,
		"XDG_CONFIG_HOME":  filepath.Join(f.configHome, "config"),
		"XDG_DATA_HOME":    filepath.Join(f.configHome, "data"),
		"XDG_STATE_HOME":   filepath.Join(f.configHome, "state"),
	}
	return Execute(f.cfg.BinaryPath, fullArgs, env)
}
