package commands

import (
	"fmt"
	"os"

	"github.com/basecamp/cli/output"
	"github.com/basecamp/fizzy-cli/internal/errors"
	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for fizzy.

To load completions:

Bash:
  $ source <(fizzy completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ fizzy completion bash > /etc/bash_completion.d/fizzy
  # macOS:
  $ fizzy completion bash > $(brew --prefix)/etc/bash_completion.d/fizzy

Zsh:
  $ source <(fizzy completion zsh)
  # To load completions for each session, execute once:
  $ fizzy completion zsh > "${fpath[1]}/_fizzy"

Fish:
  $ fizzy completion fish | source
  # To load completions for each session, execute once:
  $ fizzy completion fish > ~/.config/fish/completions/fizzy.fish

PowerShell:
  PS> fizzy completion powershell | Out-String | Invoke-Expression
  # To load completions for each session, add to your profile:
  PS> fizzy completion powershell > fizzy.ps1 && . fizzy.ps1
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfgJQ != "" {
			return errors.ErrJQNotSupported("completion script generation")
		}
		var err error
		switch args[0] {
		case "bash":
			err = cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			err = cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			err = cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			err = cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
		if err != nil {
			return &output.Error{Code: output.CodeAPI, Message: fmt.Sprintf("generating %s completions: %v", args[0], err)}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
