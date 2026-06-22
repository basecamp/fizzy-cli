package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRenderRootHelp(t *testing.T) {
	configureCLIUX()

	var buf bytes.Buffer
	renderHelp(rootCmd, &buf)
	out := buf.String()

	for _, want := range []string{"CORE COMMANDS", "activity", "token", "GETTING STARTED", "DISCOVER", "FLAGS", "--profile", "LEARN MORE", "Use `fizzy commands` to see the full command catalog.", "implies --json"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected root help to contain %q, got:\n%s", want, out)
		}
	}
}

func TestRenderSubcommandHelpIncludesAliasesAndExamples(t *testing.T) {
	configureCLIUX()

	var buf bytes.Buffer
	renderHelp(authListCmd, &buf)
	out := buf.String()

	if !strings.Contains(out, "ALIASES") || !strings.Contains(out, "ls") {
		t.Fatalf("expected alias section in help, got:\n%s", out)
	}
	if !strings.Contains(out, "EXAMPLES") {
		t.Fatalf("expected examples section in help, got:\n%s", out)
	}
}

func TestRenderRootHelpOmitsCommonWorkflows(t *testing.T) {
	configureCLIUX()

	var buf bytes.Buffer
	renderHelp(rootCmd, &buf)
	out := buf.String()

	if strings.Contains(out, "COMMON WORKFLOWS") {
		t.Fatalf("expected root help to omit common workflows, got:\n%s", out)
	}
}

func TestRenderRunnableParentCommandHelpPreservesDirectUsage(t *testing.T) {
	configureCLIUX()

	tests := []struct {
		name string
		cmd  *cobra.Command
		want string
	}{
		{name: "signup", cmd: signupCmd, want: "  fizzy signup [flags]"},
		{name: "setup", cmd: setupCmd, want: "  fizzy setup [flags]"},
		{name: "skill", cmd: skillCmd, want: "  fizzy skill [flags]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			renderHelp(tt.cmd, &buf)
			out := buf.String()

			if !strings.Contains(out, tt.want) {
				t.Fatalf("expected help to contain %q, got:\n%s", tt.want, out)
			}
		})
	}
}

func TestRenderCommandsHelpMentionsJSONCatalog(t *testing.T) {
	configureCLIUX()

	var buf bytes.Buffer
	renderHelp(commandsCmd, &buf)
	out := buf.String()

	if !strings.Contains(out, "Use --json for a structured command catalog.") {
		t.Fatalf("expected commands help to mention --json catalog, got:\n%s", out)
	}
	if !strings.Contains(out, "EXAMPLES") || !strings.Contains(out, "$ fizzy commands --json") {
		t.Fatalf("expected commands help examples to include --json, got:\n%s", out)
	}
}

func TestRenderSubcommandHelpDoesNotRepeatJQFlag(t *testing.T) {
	configureCLIUX()

	var buf bytes.Buffer
	renderHelp(boardListCmd, &buf)
	out := buf.String()

	if strings.Contains(out, "--jq") {
		t.Fatalf("expected subcommand help to omit --jq, got:\n%s", out)
	}
}
