package clitests

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/basecamp/fizzy-cli/e2e/harness"
)

func assertQuietList(t *testing.T, stdout string) {
	t.Helper()
	s := strings.TrimSpace(stdout)
	if s == "" || s == "null" {
		return
	}
	var arr []any
	if err := json.Unmarshal([]byte(s), &arr); err != nil {
		t.Fatalf("--quiet list: expected JSON array, got parse error: %v\nstdout: %s", err, s)
	}
	var envelope map[string]any
	if json.Unmarshal([]byte(s), &envelope) == nil {
		if _, hasOK := envelope["ok"]; hasOK {
			t.Fatal("--quiet list: output must not contain the 'ok' envelope key")
		}
	}
}

func assertQuietObject(t *testing.T, stdout string) {
	t.Helper()
	s := strings.TrimSpace(stdout)
	if s == "" || s == "null" {
		return
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(s), &obj); err != nil {
		t.Fatalf("--quiet object: expected JSON object, got parse error: %v\nstdout: %s", err, s)
	}
	if _, hasOK := obj["ok"]; hasOK {
		t.Fatal("--quiet object: output must not contain the 'ok' envelope key")
	}
}

func assertNonJSON(t *testing.T, stdout, flag string) {
	t.Helper()
	s := strings.TrimSpace(stdout)
	if s == "" {
		t.Fatalf("--%s: produced empty output", flag)
	}
	var v any
	if err := json.Unmarshal([]byte(s), &v); err == nil {
		t.Fatalf("--%s: output must not be valid JSON", flag)
	}
}

func assertIDsOnly(t *testing.T, stdout string) {
	t.Helper()
	s := strings.TrimSpace(stdout)
	if s == "" {
		return
	}
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "{") || strings.HasPrefix(line, "[") {
			t.Fatalf("--ids-only: line looks like JSON: %q", line)
		}
	}
}

func assertCount(t *testing.T, stdout string) {
	t.Helper()
	s := strings.TrimSpace(stdout)
	if s == "" {
		t.Fatal("--count: produced empty output")
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		t.Fatalf("--count: output %q is not an integer: %v", s, err)
	}
	if n < 0 {
		t.Fatalf("--count: got negative value %d", n)
	}
}

func assertLimitedList(t *testing.T, stdout string, max int) {
	t.Helper()
	s := strings.TrimSpace(stdout)
	if s == "" || s == "null" {
		return
	}
	var arr []any
	if err := json.Unmarshal([]byte(s), &arr); err != nil {
		t.Fatalf("--limit: expected JSON array: %v\nstdout: %s", err, s)
	}
	if len(arr) > max {
		t.Fatalf("--limit %d: got %d items", max, len(arr))
	}
}

func assertVerbose(t *testing.T, result *harness.Result) {
	t.Helper()
	if result.Response == nil {
		t.Fatal("--verbose: expected JSON response envelope")
	}
	if len(result.Response.Breadcrumbs) == 0 {
		t.Fatal("--verbose: expected at least one breadcrumb in response")
	}
}

func assertJQScalar(t *testing.T, stdout string) {
	t.Helper()
	if strings.TrimSpace(stdout) == "" {
		t.Fatal("--jq: produced empty output")
	}
}

type listFlagTest struct {
	name  string
	extra []string
	check func(t *testing.T, result *harness.Result)
}

type showFlagTest struct {
	name  string
	extra []string
	check func(t *testing.T, result *harness.Result)
}

func listFlagSuite() []listFlagTest {
	return []listFlagTest{
		{"json", []string{"--json"}, func(t *testing.T, r *harness.Result) {
			if r.Response == nil {
				t.Fatal("--json: expected JSON envelope")
			}
		}},
		{"quiet", []string{"--quiet"}, func(t *testing.T, r *harness.Result) { assertQuietList(t, r.Stdout) }},
		{"markdown", []string{"--markdown"}, func(t *testing.T, r *harness.Result) { assertNonJSON(t, r.Stdout, "markdown") }},
		{"styled", []string{"--styled"}, func(t *testing.T, r *harness.Result) { assertNonJSON(t, r.Stdout, "styled") }},
		{"ids-only", []string{"--ids-only"}, func(t *testing.T, r *harness.Result) { assertIDsOnly(t, r.Stdout) }},
		{"count", []string{"--count"}, func(t *testing.T, r *harness.Result) { assertCount(t, r.Stdout) }},
		{"limit-1", []string{"--quiet", "--limit", "1"}, func(t *testing.T, r *harness.Result) { assertLimitedList(t, r.Stdout, 1) }},
		{"verbose", []string{"--verbose"}, func(t *testing.T, r *harness.Result) { assertVerbose(t, r) }},
		{"jq-data-length", []string{"--jq", ".data | length"}, func(t *testing.T, r *harness.Result) { assertCount(t, r.Stdout) }},
		{"quiet-jq-length", []string{"--quiet", "--jq", "length"}, func(t *testing.T, r *harness.Result) { assertCount(t, r.Stdout) }},
	}
}

func showFlagSuite() []showFlagTest {
	return []showFlagTest{
		{"json", []string{"--json"}, func(t *testing.T, r *harness.Result) {
			if r.Response == nil {
				t.Fatal("--json: expected JSON envelope")
			}
		}},
		{"quiet", []string{"--quiet"}, func(t *testing.T, r *harness.Result) { assertQuietObject(t, r.Stdout) }},
		{"markdown", []string{"--markdown"}, func(t *testing.T, r *harness.Result) { assertNonJSON(t, r.Stdout, "markdown") }},
		{"styled", []string{"--styled"}, func(t *testing.T, r *harness.Result) { assertNonJSON(t, r.Stdout, "styled") }},
		{"ids-only", []string{"--ids-only"}, func(t *testing.T, r *harness.Result) { assertIDsOnly(t, r.Stdout) }},
		{"count", []string{"--count"}, func(t *testing.T, r *harness.Result) { assertCount(t, r.Stdout) }},
		{"verbose", []string{"--verbose"}, func(t *testing.T, r *harness.Result) { assertVerbose(t, r) }},
		{"jq-data-id", []string{"--jq", ".data.id"}, func(t *testing.T, r *harness.Result) { assertJQScalar(t, r.Stdout) }},
		{"quiet-jq-id", []string{"--quiet", "--jq", ".id"}, func(t *testing.T, r *harness.Result) { assertJQScalar(t, r.Stdout) }},
	}
}

func TestOutputContractListCommands(t *testing.T) {
	h := newHarness(t)
	cardNum := strconv.Itoa(fixture.CardNumber)
	cmds := []struct {
		name string
		args []string
	}{
		{"activity-list", []string{"activity", "list", "--board", fixture.BoardID}},
		{"board-list", []string{"board", "list"}},
		{"board-closed", []string{"board", "closed", "--board", fixture.BoardID}},
		{"board-postponed", []string{"board", "postponed", "--board", fixture.BoardID}},
		{"board-stream", []string{"board", "stream", "--board", fixture.BoardID}},
		{"card-list", []string{"card", "list", "--board", fixture.BoardID}},
		{"column-list", []string{"column", "list", "--board", fixture.BoardID}},
		{"comment-list", []string{"comment", "list", "--card", cardNum}},
		{"step-list", []string{"step", "list", "--card", cardNum}},
		{"reaction-list", []string{"reaction", "list", "--card", cardNum}},
		{"user-list", []string{"user", "list"}},
		{"notification-list", []string{"notification", "list"}},
		{"pin-list", []string{"pin", "list"}},
		{"tag-list", []string{"tag", "list"}},
		{"search", []string{"search", "test"}},
	}

	for _, cmd := range cmds {
		cmd := cmd
		t.Run(cmd.name, func(t *testing.T) {
			for _, f := range listFlagSuite() {
				f := f
				t.Run(f.name, func(t *testing.T) {
					args := append(append([]string(nil), cmd.args...), f.extra...)
					result := h.Run(args...)
					if result.ExitCode != harness.ExitSuccess {
						t.Fatalf("expected exit code 0, got %d\nstdout: %s\nstderr: %s", result.ExitCode, result.Stdout, result.Stderr)
					}
					f.check(t, result)
				})
			}
		})
	}
}

func TestOutputContractShowCommands(t *testing.T) {
	h := newHarness(t)
	cardNum := strconv.Itoa(fixture.CardNumber)
	cmds := []struct {
		name string
		args []string
	}{
		{"board-show", []string{"board", "show", fixture.BoardID}},
		{"card-show", []string{"card", "show", cardNum}},
		{"column-show", []string{"column", "show", fixture.ColumnID, "--board", fixture.BoardID}},
		{"comment-show", []string{"comment", "show", fixture.CommentID, "--card", cardNum}},
		{"step-show", []string{"step", "show", fixture.StepID, "--card", cardNum}},
	}

	for _, cmd := range cmds {
		cmd := cmd
		t.Run(cmd.name, func(t *testing.T) {
			for _, f := range showFlagSuite() {
				f := f
				t.Run(f.name, func(t *testing.T) {
					args := append(append([]string(nil), cmd.args...), f.extra...)
					result := h.Run(args...)
					if result.ExitCode != harness.ExitSuccess {
						t.Fatalf("expected exit code 0, got %d\nstdout: %s\nstderr: %s", result.ExitCode, result.Stdout, result.Stderr)
					}
					f.check(t, result)
				})
			}
		})
	}
}
