package commands

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/basecamp/cli/output"
	"github.com/spf13/cobra"
)

func TestUsageHelpCommand(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		want string
	}{
		{name: "nil falls back to root", cmd: "", want: "fizzy --help"},
		{name: "subcommand uses command path", cmd: "auth login", want: "fizzy auth login --help"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cmd *cobra.Command
			if tt.cmd != "" {
				parts := strings.Split(tt.cmd, " ")
				found, _, err := rootCmd.Find(parts)
				if err != nil {
					t.Fatalf("find command: %v", err)
				}
				cmd = found
			}

			if got := usageHelpCommand(cmd); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestPrintHumanErrorUsesCommandSpecificHelp(t *testing.T) {
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = w
	defer func() {
		os.Stderr = oldStderr
	}()

	cmd, _, err := rootCmd.Find([]string{"auth", "login"})
	if err != nil {
		t.Fatalf("find command: %v", err)
	}

	printHumanError(cmd, &output.Error{Code: output.CodeUsage, Message: "accepts 1 arg(s), received 0"})
	_ = w.Close()

	body, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stderr: %v", err)
	}

	out := string(body)
	if !strings.Contains(out, "Run `fizzy auth login --help` for usage.") {
		t.Fatalf("expected command-specific usage hint, got:\n%s", out)
	}
	if strings.Contains(out, "Run `fizzy --help` for usage.") {
		t.Fatalf("expected root usage hint to be omitted, got:\n%s", out)
	}
}
