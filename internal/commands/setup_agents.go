package commands

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/basecamp/cli/output"
	"github.com/basecamp/fizzy-cli/internal/harness"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

// agentSetupHandler describes what a single agent's setup step does and how to run it.
type agentSetupHandler struct {
	Labels            []string
	Confirm           string
	Run               func(cmd *cobra.Command) error
	RunNonInteractive func(cmd *cobra.Command) error
}

// agentSetupHandlers maps agent ID -> setup handler.
var agentSetupHandlers = map[string]agentSetupHandler{
	"claude": {
		Labels: []string{
			"Add basecamp/claude-plugins marketplace to Claude Code",
			"Install the fizzy plugin for Claude Code",
		},
		Confirm:           "Set up Fizzy for your coding agents?",
		Run:               runClaudeSetup,
		RunNonInteractive: runClaudeSetupNonInteractive,
	},
}

func init() {
	for _, sub := range newSetupAgentCmds() {
		setupCmd.AddCommand(sub)
	}
}

// runClaudeSetup performs the Claude Code-specific setup steps
// (marketplace add + plugin install + skill symlink).
func runClaudeSetup(cmd *cobra.Command) error {
	w := cmd.OutOrStdout()

	// If the plugin is already installed, skip to skill link repair
	pluginOK := harness.CheckClaudePlugin().Status == "pass"
	if pluginOK {
		fmt.Fprintln(w, "  ✓ Claude Code plugin installed")
	} else {
		claudePath := harness.FindClaudeBinary()
		if claudePath == "" {
			fmt.Fprintln(w, "  Claude Code detected but binary not found in PATH.")
			fmt.Fprintln(w, "  Install the plugin manually:")
			fmt.Fprintf(w, "    claude plugin marketplace add %s\n", harness.ClaudeMarketplaceSource)
			fmt.Fprintf(w, "    claude plugin install %s\n", harness.ClaudePluginName)
		} else {
			ctx := cmd.Context()

			// Register the marketplace (best-effort — may already be registered)
			marketplaceCmd := exec.CommandContext(ctx, claudePath, "plugin", "marketplace", "add", harness.ClaudeMarketplaceSource) //nolint:gosec // G204: claudePath from exec.LookPath
			marketplaceCmd.Stdout = w
			marketplaceCmd.Stderr = cmd.ErrOrStderr()
			if err := marketplaceCmd.Run(); err != nil {
				fmt.Fprintf(w, "  ⚠ Marketplace registration failed: %s\n", err)
			} else {
				fmt.Fprintln(w, "  ✓ Marketplace registered")
			}

			// Install the plugin
			installCmd := exec.CommandContext(ctx, claudePath, "plugin", "install", harness.ClaudePluginName) //nolint:gosec // G204: claudePath from exec.LookPath
			installCmd.Stdout = w
			installCmd.Stderr = cmd.ErrOrStderr()
			if err := installCmd.Run(); err != nil {
				fmt.Fprintf(w, "  ⚠ Plugin install failed: %s\n", err)
				fmt.Fprintln(w, "  Try manually:")
				fmt.Fprintf(w, "    claude plugin marketplace add %s\n", harness.ClaudeMarketplaceSource)
				fmt.Fprintf(w, "    claude plugin install %s\n", harness.ClaudePluginName)
			} else {
				verify := harness.CheckClaudePlugin()
				if verify.Status == "pass" {
					fmt.Fprintln(w, "  ✓ Claude Code plugin installed")
				} else {
					fmt.Fprintln(w, "  ✗ Claude Code plugin may not have installed correctly")
				}
			}
		}
	}

	// Always attempt skill link repair
	if _, _, err := linkSkillToClaude(); err != nil {
		fmt.Fprintf(w, "  ⚠ Claude skill symlink failed: %s\n", err)
	}

	return nil
}

// setupAgents offers to set up detected coding agents during the setup wizard.
func setupAgents(cmd *cobra.Command) error {
	w := cmd.OutOrStdout()

	agents := harness.DetectedAgents()
	if len(agents) == 0 {
		return nil
	}

	// Check if all detected agents are already fully set up
	allGood := baselineSkillInstalled()
	if allGood {
		for _, a := range agents {
			if a.Checks == nil {
				continue
			}
			for _, c := range a.Checks() {
				if c.Status != "pass" {
					allGood = false
					break
				}
			}
			if !allGood {
				break
			}
		}
	}

	if allGood {
		for _, a := range agents {
			fmt.Fprintf(w, "  ✓ %s plugin installed\n", a.Name)
		}
		fmt.Fprintln(w)
		return nil
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Coding Agent Setup")
	fmt.Fprintln(w)

	// Show detected agents
	var names []string
	for _, a := range agents {
		names = append(names, a.Name)
	}
	fmt.Fprintf(w, "  Detected: %s\n", joinNames(names))
	fmt.Fprintln(w)

	// Build numbered list of what will happen
	fmt.Fprintln(w, "  This will:")
	step := 1
	fmt.Fprintf(w, "    %d. Install Fizzy agent skill to ~/.agents/skills/fizzy/\n", step)
	step++
	for _, a := range agents {
		handler, ok := agentSetupHandlers[a.ID]
		if !ok {
			continue
		}
		for _, label := range handler.Labels {
			fmt.Fprintf(w, "    %d. %s\n", step, label)
			step++
		}
	}
	fmt.Fprintln(w)

	var install bool
	err := huh.NewConfirm().
		Title("Set up Fizzy for your coding agents?").
		Value(&install).
		Run()

	if err != nil || !install {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "  You can set up agents later:")
		for _, a := range agents {
			if _, ok := agentSetupHandlers[a.ID]; ok {
				fmt.Fprintf(w, "    fizzy setup %s\n", a.ID)
			}
		}
		fmt.Fprintln(w)
		return nil //nolint:nilerr // user cancelled
	}

	fmt.Fprintln(w)

	// Install baseline skill
	if _, err := installSkillFiles(); err != nil {
		fmt.Fprintf(w, "  ⚠ Skill install failed: %s\n", err)
	} else {
		fmt.Fprintln(w, "  ✓ Agent skill installed")
	}

	// Run each detected agent's handler
	for _, a := range agents {
		handler, ok := agentSetupHandlers[a.ID]
		if !ok {
			continue
		}
		if err := handler.Run(cmd); err != nil {
			return err
		}
	}

	fmt.Fprintln(w)
	return nil
}

// runClaudeSetupNonInteractive attempts plugin install without prompts.
func runClaudeSetupNonInteractive(cmd *cobra.Command) error {
	var errs []string

	if check := harness.CheckClaudePlugin(); check.Status != "pass" {
		claudePath := harness.FindClaudeBinary()
		if claudePath != "" {
			ctx := cmd.Context()
			w := cmd.ErrOrStderr()

			// Best-effort marketplace registration
			marketplaceCmd := exec.CommandContext(ctx, claudePath, "plugin", "marketplace", "add", harness.ClaudeMarketplaceSource) //nolint:gosec // G204: claudePath from exec.LookPath
			marketplaceCmd.Stderr = w
			_ = marketplaceCmd.Run()

			// Install the plugin
			installCmd := exec.CommandContext(ctx, claudePath, "plugin", "install", harness.ClaudePluginName) //nolint:gosec // G204: claudePath from exec.LookPath
			installCmd.Stderr = w
			if err := installCmd.Run(); err != nil {
				errs = append(errs, fmt.Sprintf("plugin install: %s", err))
			}
		}
	}

	// Always attempt skill link repair
	if _, _, err := linkSkillToClaude(); err != nil {
		errs = append(errs, fmt.Sprintf("skill link: %s", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

// newSetupAgentCmds generates `setup <agent>` subcommands from the registry.
func newSetupAgentCmds() []*cobra.Command {
	var cmds []*cobra.Command
	for _, a := range harness.AllAgents() {
		agent := a // capture for closure
		handler, ok := agentSetupHandlers[agent.ID]
		if !ok {
			continue
		}
		h := handler // capture
		cmds = append(cmds, &cobra.Command{
			Use:   agent.ID,
			Short: fmt.Sprintf("Install the Fizzy plugin for %s", agent.Name),
			Long:  fmt.Sprintf("Set up the %s integration so %s can access Fizzy.", agent.Name, agent.Name),
			RunE: func(cmd *cobra.Command, args []string) error {
				// Always install baseline skill
				_, skillErr := installSkillFiles()

				var setupErrors []string
				if skillErr != nil {
					setupErrors = append(setupErrors, fmt.Sprintf("skill install: %s", skillErr))
				}

				if IsMachineOutput() {
					if h.RunNonInteractive != nil {
						if err := h.RunNonInteractive(cmd); err != nil {
							setupErrors = append(setupErrors, err.Error())
						}
					}
				} else {
					w := cmd.OutOrStdout()

					if skillErr != nil {
						fmt.Fprintf(w, "  ⚠ Skill install failed: %s\n", skillErr)
					} else {
						fmt.Fprintln(w, "  ✓ Agent skill installed")
					}

					if err := h.Run(cmd); err != nil {
						return err
					}

					fmt.Fprintf(w, "  Start a new %s session to use Fizzy commands.\n", agent.Name)
				}

				// Build structured result
				detected := agent.Detect != nil && agent.Detect()
				installed := false
				if detected && agent.Checks != nil {
					checks := agent.Checks()
					installed = len(checks) > 0
					for _, c := range checks {
						if c.Status != "pass" {
							installed = false
							break
						}
					}
				}

				summary := agent.Name + " plugin installed"
				if !detected {
					summary = agent.Name + " not detected"
				} else if !installed {
					summary = agent.Name + " plugin not installed"
				}

				result := map[string]any{
					"plugin_installed": installed,
					"agent_detected":   detected,
				}
				if len(setupErrors) > 0 {
					result["errors"] = setupErrors
					if installed {
						result["plugin_installed"] = false
						summary = agent.Name + " plugin not installed"
					}
				}

				if out != nil {
					_ = out.OK(result, output.WithSummary(summary))
					captureResponse()
				}
				return nil
			},
		})
	}
	return cmds
}

// joinNames joins names with commas and "and".
func joinNames(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	case 2:
		return names[0] + " and " + names[1]
	default:
		result := ""
		for i, n := range names {
			if i == len(names)-1 {
				result += "and " + n
			} else {
				result += n + ", "
			}
		}
		return result
	}
}
