package commands

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderRootHelp(t *testing.T) {
	configureCLIUX()

	var buf bytes.Buffer
	renderHelp(rootCmd, &buf)
	out := buf.String()

	for _, want := range []string{"CORE COMMANDS", "GLOBAL OUTPUT FLAGS", "LEARN MORE"} {
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
