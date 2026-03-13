package commands

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// commandInfo describes a command for structured output.
type commandInfo struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Flags       []flagInfo    `json:"flags,omitempty"`
	Subcommands []commandInfo `json:"subcommands,omitempty"`
}

type flagInfo struct {
	Name        string `json:"name"`
	Shorthand   string `json:"shorthand,omitempty"`
	Type        string `json:"type"`
	Default     string `json:"default,omitempty"`
	Description string `json:"description"`
}

// commandsCmd emits a flat catalog of all commands with their flags.
var commandsCmd = &cobra.Command{
	Use:   "commands",
	Short: "List all available commands",
	Long:  "Lists all available commands with their flags in JSON format.",
	RunE: func(cmd *cobra.Command, args []string) error {
		catalog := walkCommands(rootCmd, "fizzy")
		printSuccess(catalog)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(commandsCmd)
}

// walkCommands recursively builds a command catalog.
func walkCommands(cmd *cobra.Command, prefix string) []commandInfo {
	var result []commandInfo
	for _, sub := range cmd.Commands() {
		if sub.Hidden || sub.Name() == "help" || sub.Name() == "completion" {
			continue
		}
		fullName := prefix + " " + sub.Name()
		info := commandInfo{
			Name:        fullName,
			Description: sub.Short,
			Flags:       collectFlags(sub),
		}
		children := walkCommands(sub, fullName)
		if len(children) > 0 {
			info.Subcommands = children
		}
		result = append(result, info)
	}
	return result
}

// collectFlags returns non-hidden local flags for a command.
func collectFlags(cmd *cobra.Command) []flagInfo {
	var flags []flagInfo
	cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			return
		}
		flags = append(flags, flagInfo{
			Name:        "--" + f.Name,
			Shorthand:   f.Shorthand,
			Type:        f.Value.Type(),
			Default:     f.DefValue,
			Description: f.Usage,
		})
	})
	return flags
}

// agentHelp outputs command help as structured JSON.
func agentHelp(cmd *cobra.Command, _ []string) {
	info := commandInfo{
		Name:        cmd.CommandPath(),
		Description: cmd.Short,
		Flags:       collectFlags(cmd),
	}

	// Add persistent flags from parent chain
	cmd.InheritedFlags().VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			return
		}
		info.Flags = append(info.Flags, flagInfo{
			Name:        "--" + f.Name,
			Shorthand:   f.Shorthand,
			Type:        f.Value.Type(),
			Default:     f.DefValue,
			Description: f.Usage,
		})
	})

	children := walkCommands(cmd, cmd.CommandPath())
	if len(children) > 0 {
		info.Subcommands = children
	}

	data, _ := json.MarshalIndent(info, "", "  ")
	fmt.Fprintln(outWriter, string(data))
}

// installAgentHelp sets the custom help function when --agent is active.
func installAgentHelp() {
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if cfgAgent {
			agentHelp(cmd, args)
			return
		}
		// Banner on root help only
		if cmd == rootCmd {
			printBanner()
		}
		// Fall back to Cobra's default help
		cmd.Root().SetHelpFunc(nil)
		_ = cmd.Help()
	})
}
