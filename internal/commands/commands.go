package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// commandInfo describes a command for structured output.
type commandInfo struct {
	Name        string        `json:"name"`
	Category    string        `json:"category,omitempty"`
	Description string        `json:"description"`
	Flags       []flagInfo    `json:"flags,omitempty"`
	Subcommands []commandInfo `json:"subcommands,omitempty"`
}

var commandCatalogOrder = []string{"core", "collaboration", "admin", "utilities"}

var commandCatalogTitles = map[string]string{
	"core":          "CORE COMMANDS",
	"collaboration": "COLLABORATION",
	"admin":         "ACCOUNT & ADMIN",
	"utilities":     "SETUP & TOOLS",
}

var commandCatalogGroups = map[string][]string{
	"core":          {"activity", "board", "card", "column", "comment", "search", "step"},
	"collaboration": {"notification", "pin", "reaction", "tag", "user"},
	"admin":         {"auth", "account", "identity", "token", "webhook", "upload", "migrate"},
	"utilities":     {"setup", "signup", "completion", "doctor", "config", "skill", "commands", "version"},
}

var commandCatalogCategory = func() map[string]string {
	m := make(map[string]string)
	for category, names := range commandCatalogGroups {
		for _, name := range names {
			m[name] = category
		}
	}
	return m
}()

type commandCatalogEntry struct {
	Name        string
	Description string
	Actions     []string
}

type flagInfo struct {
	Name        string `json:"name"`
	Shorthand   string `json:"shorthand,omitempty"`
	Type        string `json:"type"`
	Default     string `json:"default,omitempty"`
	Description string `json:"description"`
}

// commandsCmd emits a catalog of all commands with their flags.
var commandsCmd = &cobra.Command{
	Use:     "commands [filter]",
	Aliases: []string{"cmds"},
	Short:   "List all available commands",
	Long:    "Lists all available commands. Use --json for a structured command catalog.",
	Args:    cobra.MaximumNArgs(1),
	Example: "$ fizzy commands\n$ fizzy commands auth\n$ fizzy commands --json",
	RunE: func(cmd *cobra.Command, args []string) error {
		filter := ""
		if len(args) == 1 {
			filter = args[0]
		}

		catalog := walkCommands(rootCmd, "fizzy", filter)
		if isHumanOutput() {
			renderCommandsCatalog(outWriter, filter)
			captureResponse()
			return nil
		}
		printSuccess(catalog)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(commandsCmd)
}

// walkCommands recursively builds a command catalog.
func walkCommands(cmd *cobra.Command, prefix, filter string) []commandInfo {
	var result []commandInfo
	for _, sub := range cmd.Commands() {
		if sub.Hidden || sub.Name() == "help" {
			continue
		}
		fullName := prefix + " " + sub.Name()
		children := walkCommands(sub, fullName, filter)
		if !matchesCommandFilter(sub, fullName, filter) && len(children) == 0 {
			continue
		}
		info := commandInfo{
			Name:        fullName,
			Category:    commandCatalogCategory[sub.Name()],
			Description: sub.Short,
			Flags:       collectFlags(sub),
		}
		if len(children) > 0 {
			info.Subcommands = children
		}
		result = append(result, info)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
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
	sort.Slice(flags, func(i, j int) bool { return flags[i].Name < flags[j].Name })
	return flags
}

func matchesCommandFilter(cmd *cobra.Command, fullName, filter string) bool {
	if strings.TrimSpace(filter) == "" {
		return true
	}
	needle := strings.ToLower(strings.TrimSpace(filter))
	fields := []string{cmd.Name(), fullName, cmd.Short, cmd.Long}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), needle) {
			return true
		}
	}
	return false
}

func commandActions(cmd *cobra.Command) []string {
	var actions []string
	for _, sub := range cmd.Commands() {
		if sub.Hidden || sub.Name() == "help" {
			continue
		}
		actions = append(actions, sub.Name())
	}
	sort.Strings(actions)
	return actions
}

func renderCommandsCatalog(w io.Writer, filter string) {
	if w == nil {
		w = outWriter
	}
	if w == nil {
		return
	}

	registered := make(map[string]*cobra.Command)
	for _, sub := range rootCmd.Commands() {
		if sub.Hidden || sub.Name() == "help" {
			continue
		}
		registered[sub.Name()] = sub
	}

	grouped := make(map[string][]commandCatalogEntry)
	maxName := 0
	maxDesc := 0
	for _, category := range commandCatalogOrder {
		for _, name := range commandCatalogGroups[category] {
			cmd := registered[name]
			if cmd == nil {
				continue
			}
			if !matchesCommandFilter(cmd, rootCmd.CommandPath()+" "+name, filter) {
				continue
			}
			entry := commandCatalogEntry{Name: name, Description: cmd.Short, Actions: commandActions(cmd)}
			grouped[category] = append(grouped[category], entry)
			if len(entry.Name) > maxName {
				maxName = len(entry.Name)
			}
			if len(entry.Description) > maxDesc {
				maxDesc = len(entry.Description)
			}
		}
	}

	muted := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	showActions := strings.TrimSpace(filter) != ""
	printedAny := false
	for _, category := range commandCatalogOrder {
		entries := grouped[category]
		if len(entries) == 0 {
			continue
		}
		if printedAny {
			fmt.Fprintln(w)
		}
		printedAny = true
		fmt.Fprintln(w, commandCatalogTitles[category])
		for _, entry := range entries {
			line := fmt.Sprintf("  %-*s  %-*s", maxName, entry.Name, maxDesc, entry.Description)
			if showActions && len(entry.Actions) > 0 {
				line += "  " + muted.Render(strings.Join(entry.Actions, ", "))
			}
			fmt.Fprintln(w, line)
		}
	}

	if !printedAny {
		fmt.Fprintf(w, "No commands match %q\n", filter)
		return
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "LEARN MORE")
	fmt.Fprintln(w, "  fizzy <command> --help   Help for a specific command")
	fmt.Fprintln(w, "  fizzy commands --json    Structured command catalog")
	if strings.TrimSpace(filter) == "" {
		fmt.Fprintln(w, "  fizzy commands auth      Filter commands by name or description")
	} else {
		fmt.Fprintln(w, "  fizzy commands           Full command catalog")
	}
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

	children := walkCommands(cmd, cmd.CommandPath(), "")
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
		// Fall back to Cobra's default help
		cmd.Root().SetHelpFunc(nil)
		_ = cmd.Help()
	})
}
