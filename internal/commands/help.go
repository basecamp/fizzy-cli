package commands

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const helpGroupAnnotation = "fizzy.help.group"

var cliUXConfigured bool

type helpLink struct {
	Command     string
	Description string
}

type commandGroup struct {
	Title    string
	Commands []*cobra.Command
}

func configureCLIUX() {
	if cliUXConfigured {
		return
	}
	cliUXConfigured = true

	applyHelpMetadata(rootCmd)
	installHumanHelp()
}

func applyHelpMetadata(root *cobra.Command) {
	setRootHelpMetadata(root)
	walkCommandTree(root, func(cmd *cobra.Command) {
		applyGenericAliases(cmd)
		if cmd.Example == "" {
			if ex := commandExamples[cmd.CommandPath()]; ex != "" {
				cmd.Example = ex
			}
		}
	})

	for group, names := range rootCommandGroups {
		for _, name := range names {
			if cmd := findSubcommand(root, name); cmd != nil {
				if cmd.Annotations == nil {
					cmd.Annotations = map[string]string{}
				}
				cmd.Annotations[helpGroupAnnotation] = group
			}
		}
	}
}

func installHumanHelp() {
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if cfgAgent {
			agentHelp(cmd, args)
			return
		}
		renderHelp(cmd, outWriter)
	})
}

func renderHelp(cmd *cobra.Command, w io.Writer) {
	if w == nil {
		w = outWriter
	}
	if w == nil {
		w = io.Discard
	}

	if cmd == rootCmd {
		renderRootHelp(cmd, w)
		return
	}
	renderCommandHelp(cmd, w)
}

func renderRootHelp(cmd *cobra.Command, w io.Writer) {
	cmd.InitDefaultHelpFlag()
	cmd.InitDefaultVersionFlag()

	fmt.Fprintln(w, strings.TrimSpace(cmd.Long))
	fmt.Fprintln(w)
	fmt.Fprintln(w, "USAGE")
	fmt.Fprintf(w, "  %s <command> [flags]\n", cmd.Name())

	groups := groupedRootCommands(cmd)
	for _, group := range groups {
		fmt.Fprintln(w)
		fmt.Fprintln(w, group.Title)
		printCommandList(w, group.Commands)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "FLAGS")
	printNamedFlags(w, cmd.PersistentFlags(), []string{"json", "jq", "quiet", "profile", "verbose"})
	if flags := rootLocalFlags(cmd); len(flags) > 0 {
		printFlags(w, flags)
	}

	if cmd.Example != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "EXAMPLES")
		printExampleBlock(w, cmd.Example)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "LEARN MORE")
	fmt.Fprintln(w, "  Use `fizzy commands` to see the full command catalog.")
	fmt.Fprintln(w, "  Use `fizzy <command> --help` for more information about a command.")
	fmt.Fprintln(w, "  Use `fizzy commands --json` for a structured command catalog.")
}

func renderCommandHelp(cmd *cobra.Command, w io.Writer) {
	cmd.InitDefaultHelpFlag()

	desc := strings.TrimSpace(cmd.Long)
	if desc == "" {
		desc = strings.TrimSpace(cmd.Short)
	}
	if desc != "" {
		fmt.Fprintln(w, desc)
		fmt.Fprintln(w)
	}

	subs := visibleSubcommands(cmd)
	usageLine := cmd.UseLine()
	if len(subs) > 0 && !cmd.Runnable() {
		usageLine = cmd.CommandPath() + " <command> [flags]"
	}

	fmt.Fprintln(w, "USAGE")
	fmt.Fprintf(w, "  %s\n", usageLine)

	if len(cmd.Aliases) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "ALIASES")
		fmt.Fprintf(w, "  %s\n", strings.Join(cmd.Aliases, ", "))
	}

	if len(subs) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "AVAILABLE COMMANDS")
		printCommandList(w, subs)
	}

	if flags := visibleFlags(cmd.LocalFlags()); len(flags) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "FLAGS")
		printFlags(w, flags)
	}

	if cmd.HasInheritedFlags() {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "GLOBAL OUTPUT FLAGS")
		printNamedFlags(w, cmd.InheritedFlags(), []string{"json", "quiet", "styled", "markdown", "ids-only", "count", "limit"})

		fmt.Fprintln(w)
		fmt.Fprintln(w, "GLOBAL CONFIG FLAGS")
		printNamedFlags(w, cmd.InheritedFlags(), []string{"profile", "token", "api-url", "verbose", "agent"})
	}

	if cmd.Example != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "EXAMPLES")
		printExampleBlock(w, cmd.Example)
	}

	if links := relatedCommands[cmd.CommandPath()]; len(links) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "SEE ALSO")
		for _, link := range links {
			fmt.Fprintf(w, "  %-34s %s\n", link.Command, link.Description)
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "LEARN MORE")
	fmt.Fprintf(w, "  Use `fizzy %s --help` for more information about this command.\n", strings.TrimPrefix(cmd.CommandPath(), "fizzy "))
}

func setRootHelpMetadata(cmd *cobra.Command) {
	cmd.Example = strings.Join([]string{
		"$ fizzy auth status",
		"$ fizzy doctor",
		"$ fizzy board list",
		"$ fizzy card show 42",
		"$ fizzy search \"billing bug\"",
	}, "\n")
}

func applyGenericAliases(cmd *cobra.Command) {
	switch cmd.Name() {
	case "list":
		appendAlias(cmd, "ls")
	case "show":
		appendAlias(cmd, "view")
	case "delete":
		appendAlias(cmd, "rm")
	}
}

func appendAlias(cmd *cobra.Command, alias string) {
	for _, existing := range cmd.Aliases {
		if existing == alias {
			return
		}
	}
	cmd.Aliases = append(cmd.Aliases, alias)
}

func walkCommandTree(cmd *cobra.Command, fn func(*cobra.Command)) {
	fn(cmd)
	for _, sub := range cmd.Commands() {
		if sub.Hidden {
			continue
		}
		walkCommandTree(sub, fn)
	}
}

func findSubcommand(cmd *cobra.Command, name string) *cobra.Command {
	for _, sub := range cmd.Commands() {
		if sub.Name() == name {
			return sub
		}
	}
	return nil
}

func groupedRootCommands(cmd *cobra.Command) []commandGroup {
	groups := make([]commandGroup, 0, len(rootCommandGroupsOrder))
	for _, key := range rootCommandGroupsOrder {
		title := rootCommandGroupTitles[key]
		var commands []*cobra.Command
		for _, sub := range visibleSubcommands(cmd) {
			if sub.Annotations[helpGroupAnnotation] == key {
				commands = append(commands, sub)
			}
		}
		if len(commands) == 0 {
			continue
		}
		sort.Slice(commands, func(i, j int) bool { return commands[i].Name() < commands[j].Name() })
		groups = append(groups, commandGroup{Title: title, Commands: commands})
	}
	return groups
}

func visibleSubcommands(cmd *cobra.Command) []*cobra.Command {
	var subs []*cobra.Command
	for _, sub := range cmd.Commands() {
		if sub.Hidden || sub.Name() == "help" {
			continue
		}
		subs = append(subs, sub)
	}
	sort.Slice(subs, func(i, j int) bool { return subs[i].Name() < subs[j].Name() })
	return subs
}

func printCommandList(w io.Writer, commands []*cobra.Command) {
	for _, sub := range commands {
		fmt.Fprintf(w, "  %-14s %s\n", sub.Name(), sub.Short)
	}
}

func printNamedFlags(w io.Writer, flags *pflag.FlagSet, names []string) {
	selected := make([]*pflag.Flag, 0, len(names))
	for _, name := range names {
		if f := flags.Lookup(name); f != nil && !f.Hidden {
			selected = append(selected, f)
		}
	}
	printFlags(w, selected)
}

func rootLocalFlags(cmd *cobra.Command) []*pflag.Flag {
	excluded := map[string]bool{
		"agent": true, "api-url": true, "count": true, "ids-only": true,
		"jq": true, "json": true, "limit": true, "markdown": true,
		"profile": true, "quiet": true, "styled": true, "token": true,
		"verbose": true,
	}

	flags := visibleFlags(cmd.Flags())
	result := make([]*pflag.Flag, 0, len(flags))
	for _, f := range flags {
		if excluded[f.Name] {
			continue
		}
		result = append(result, f)
	}
	return result
}

func visibleFlags(flags *pflag.FlagSet) []*pflag.Flag {
	var result []*pflag.Flag
	flags.VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			return
		}
		result = append(result, f)
	})
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func printFlags(w io.Writer, flags []*pflag.Flag) {
	for _, f := range flags {
		fmt.Fprintf(w, "  %-26s %s\n", flagDisplayName(f), flagUsage(f))
	}
}

func flagDisplayName(f *pflag.Flag) string {
	parts := make([]string, 0, 2)
	if f.Shorthand != "" {
		parts = append(parts, "-"+f.Shorthand)
	}
	name := "--" + f.Name
	if f.Value.Type() != "bool" {
		name += " <" + strings.ToUpper(strings.ReplaceAll(f.Name, "-", "_")) + ">"
	}
	parts = append(parts, name)
	return strings.Join(parts, ", ")
}

func flagUsage(f *pflag.Flag) string {
	usage := f.Usage
	if shouldShowDefault(f) {
		usage += fmt.Sprintf(" (default: %s)", f.DefValue)
	}
	return usage
}

func shouldShowDefault(f *pflag.Flag) bool {
	if f.DefValue == "" {
		return false
	}
	switch f.Value.Type() {
	case "bool":
		return f.DefValue == "true"
	case "int":
		return f.DefValue != "0"
	default:
		return true
	}
}

func printExampleBlock(w io.Writer, example string) {
	style := lipgloss.NewStyle()
	if f, ok := w.(*os.File); ok {
		if isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd()) {
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
		}
	}
	for _, line := range strings.Split(strings.TrimSpace(example), "\n") {
		fmt.Fprintf(w, "%s\n", style.Render("  "+line))
	}
}

var rootCommandGroupsOrder = []string{"core", "collaboration", "getting-started", "discover"}

var rootCommandGroupTitles = map[string]string{
	"core":            "CORE COMMANDS",
	"collaboration":   "COLLABORATION",
	"getting-started": "GETTING STARTED",
	"discover":        "DISCOVER",
}

var rootCommandGroups = map[string][]string{
	"core":            {"auth", "token", "activity", "board", "card", "search"},
	"collaboration":   {"comment", "notification"},
	"getting-started": {"setup", "signup"},
	"discover":        {"doctor", "config", "commands", "version"},
}

var commandExamples = map[string]string{
	"fizzy auth":              "$ fizzy auth status\n$ fizzy auth login TOKEN --profile acme",
	"fizzy auth status":       "$ fizzy auth status",
	"fizzy auth list":         "$ fizzy auth list\n$ fizzy auth switch acme",
	"fizzy activity":          "$ fizzy activity list\n$ fizzy activity list --board <id>",
	"fizzy activity list":     "$ fizzy activity list --board <id>\n$ fizzy activity list --creator <user_id>",
	"fizzy board":             "$ fizzy board list\n$ fizzy board show <id>",
	"fizzy board list":        "$ fizzy board list\n$ fizzy board list --page 2",
	"fizzy board show":        "$ fizzy board show <id>",
	"fizzy card":              "$ fizzy card list --board <id>\n$ fizzy card show <number>",
	"fizzy card list":         "$ fizzy card list --board <id>\n$ fizzy card list --board <id> --all",
	"fizzy card show":         "$ fizzy card show <number>",
	"fizzy card create":       "$ fizzy card create --board <id> --title \"Fix billing bug\"",
	"fizzy comment list":      "$ fizzy comment list --card <number>",
	"fizzy comment create":    "$ fizzy comment create --card <number> --body \"Looks good\"",
	"fizzy commands":          "$ fizzy commands\n$ fizzy commands --json",
	"fizzy config":            "$ fizzy config show\n$ fizzy config explain",
	"fizzy config show":       "$ fizzy config show\n$ fizzy config show --verbose",
	"fizzy config explain":    "$ fizzy config explain\n$ fizzy config explain --profile acme",
	"fizzy doctor":            "$ fizzy doctor\n$ fizzy doctor --profile acme\n$ fizzy doctor --all-profiles",
	"fizzy search":            "$ fizzy search \"billing bug\"\n$ fizzy search <card-id>",
	"fizzy notification":      "$ fizzy notification tray\n$ fizzy notification list",
	"fizzy notification tray": "$ fizzy notification tray",
	"fizzy user":              "$ fizzy user list\n$ fizzy user show <id>",
	"fizzy user list":         "$ fizzy user list\n$ fizzy user list --all",
	"fizzy setup":             "$ fizzy setup",
	"fizzy signup":            "$ fizzy signup start --email you@example.com",
	"fizzy version":           "$ fizzy version",
}

var relatedCommands = map[string][]helpLink{
	"fizzy auth": {
		{Command: "fizzy identity show", Description: "View your identity and accessible accounts"},
		{Command: "fizzy doctor", Description: "Run a full health check after logging in"},
		{Command: "fizzy board list", Description: "List boards after logging in"},
	},
	"fizzy auth status": {
		{Command: "fizzy doctor", Description: "Run a full CLI health check"},
		{Command: "fizzy identity show", Description: "View your identity"},
		{Command: "fizzy auth list", Description: "List saved profiles"},
	},
	"fizzy board": {
		{Command: "fizzy card list --board <id>", Description: "List cards on a board"},
		{Command: "fizzy column list --board <id>", Description: "List columns on a board"},
	},
	"fizzy board show": {
		{Command: "fizzy card list --board <id>", Description: "List cards on this board"},
		{Command: "fizzy column list --board <id>", Description: "List columns on this board"},
	},
	"fizzy card": {
		{Command: "fizzy search \"query\"", Description: "Search cards by text"},
		{Command: "fizzy comment list --card <number>", Description: "List comments on a card"},
	},
	"fizzy search": {
		{Command: "fizzy card list --board <id>", Description: "Browse cards on a board"},
		{Command: "fizzy card show <number>", Description: "Show a matching card"},
	},
	"fizzy config": {
		{Command: "fizzy config show", Description: "Show the effective configuration"},
		{Command: "fizzy config explain", Description: "Explain config precedence"},
		{Command: "fizzy auth list", Description: "List saved profiles"},
	},
	"fizzy config show": {
		{Command: "fizzy config explain", Description: "Explain why these values won"},
		{Command: "fizzy doctor", Description: "Run a full health check"},
	},
	"fizzy config explain": {
		{Command: "fizzy config show", Description: "Show only the effective values"},
		{Command: "fizzy auth list", Description: "List saved profiles"},
		{Command: "fizzy doctor", Description: "Run a full health check"},
	},
	"fizzy doctor": {
		{Command: "fizzy auth status", Description: "Check current authentication state"},
		{Command: "fizzy config explain", Description: "Explain config precedence"},
		{Command: "fizzy identity show", Description: "Verify identity and accessible accounts"},
		{Command: "fizzy setup", Description: "Repair or re-run interactive setup"},
	},
}
