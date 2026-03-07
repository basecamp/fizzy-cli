package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/basecamp/cli/output"
	"github.com/basecamp/fizzy-cli/internal/harness"
	"github.com/basecamp/fizzy-cli/internal/skills"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

const skillFilename = "SKILL.md"

// SkillLocation represents a predefined skill installation target.
type SkillLocation struct {
	Name string
	Path string
}

var skillLocations = []SkillLocation{
	{Name: "Agents (Shared)", Path: "~/.agents/skills/fizzy/SKILL.md"},
	{Name: "Claude Code (Global)", Path: "~/.claude/skills/fizzy/SKILL.md"},
	{Name: "Claude Code (Project)", Path: ".claude/skills/fizzy/SKILL.md"},
	{Name: "OpenCode (Global)", Path: "~/.config/opencode/skill/fizzy/SKILL.md"},
	{Name: "OpenCode (Project)", Path: ".opencode/skill/fizzy/SKILL.md"},
	{Name: "Codex (Global)", Path: codexGlobalSkillPath()},
}

var skillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Manage the embedded agent skill file",
	Long:  "Print or install the SKILL.md embedded in this binary.",
	RunE:  runSkill,
}

func init() {
	rootCmd.AddCommand(skillCmd)
	skillCmd.AddCommand(newSkillInstallCmd())
}

func runSkill(cmd *cobra.Command, args []string) error {
	// Non-interactive: print skill content
	if IsMachineOutput() {
		_, err := fmt.Fprint(cmd.OutOrStdout(), string(skills.Content))
		return err
	}

	// Interactive: show agent picker wizard
	return runSkillWizard()
}

func newSkillInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install the fizzy agent skill",
		Long:  "Copies the embedded SKILL.md to ~/.agents/skills/fizzy/ and creates a symlink in ~/.claude/skills/fizzy (if Claude Code is detected).",
		RunE: func(cmd *cobra.Command, args []string) error {
			skillPath, err := installSkillFiles()
			if err != nil {
				return err
			}

			result := map[string]any{
				"skill_path": skillPath,
			}

			// Only create the Claude symlink if Claude is actually installed
			if harness.DetectClaude() {
				symlinkPath, notice, linkErr := linkSkillToClaude()
				if linkErr != nil {
					return linkErr
				}
				result["symlink_path"] = symlinkPath
				if notice != "" {
					result["notice"] = notice
				}
			}

			summary := "Fizzy skill installed"
			if out != nil {
				_ = out.OK(result, output.WithSummary(summary))
				captureResponse()
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Installed skill to %s\n", skillPath)
			return nil
		},
	}
}

// installSkillFiles writes the embedded SKILL.md to ~/.agents/skills/fizzy/
// and returns the path to the installed file.
func installSkillFiles() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home directory: %w", err)
	}

	skillDir := filepath.Join(home, ".agents", "skills", "fizzy")
	skillFile := filepath.Join(skillDir, skillFilename)

	if err := os.MkdirAll(skillDir, 0o755); err != nil { //nolint:gosec // G301: Skill files are not secrets
		return "", fmt.Errorf("creating skill directory: %w", err)
	}
	if err := os.WriteFile(skillFile, skills.Content, 0o644); err != nil { //nolint:gosec // G306: Skill files are not secrets
		return "", fmt.Errorf("writing skill file: %w", err)
	}

	return skillFile, nil
}

// runSkillWizard runs the interactive skill installation wizard.
func runSkillWizard() error {
	fmt.Println()
	fmt.Println("Fizzy Skill Installation")
	fmt.Println()

	// Build options
	options := make([]huh.Option[string], len(skillLocations)+1)
	for i, loc := range skillLocations {
		label := fmt.Sprintf("%s (%s)", loc.Name, loc.Path)
		options[i] = huh.NewOption(label, loc.Path)
	}
	options[len(skillLocations)] = huh.NewOption("Other (custom path)", "other")

	var selectedPath string
	err := huh.NewSelect[string]().
		Title("Where would you like to install the Fizzy skill?").
		Options(options...).
		Value(&selectedPath).
		Run()

	if err != nil {
		fmt.Println("Installation cancelled.")
		return nil //nolint:nilerr // user cancelled prompt
	}

	// Handle custom path
	if selectedPath == "other" {
		err = huh.NewInput().
			Title("Enter custom path").
			Description("You can enter a directory or full path ending in SKILL.md").
			Placeholder("/path/to/skills/fizzy/SKILL.md").
			Value(&selectedPath).
			Validate(func(s string) error {
				if s == "" {
					return fmt.Errorf("path is required")
				}
				return nil
			}).
			Run()

		if err != nil {
			fmt.Println("Installation cancelled.")
			return nil //nolint:nilerr // user cancelled prompt
		}

		selectedPath = normalizeSkillPath(selectedPath)
	}

	expandedPath := expandPath(selectedPath)

	// Check for existing file
	if fileExists(expandedPath) {
		var overwrite bool
		err = huh.NewConfirm().
			Title(fmt.Sprintf("File already exists at %s. Overwrite?", sanitizeLogValue(selectedPath))).
			Value(&overwrite).
			Run()

		if err != nil || !overwrite {
			fmt.Println("Installation cancelled.")
			return nil //nolint:nilerr // user cancelled or declined
		}
	}

	// Write to selected location
	dir := filepath.Dir(expandedPath)
	if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // G301: Skill files are not secrets
		return &output.Error{Code: output.CodeAPI, Message: fmt.Sprintf("creating directory: %v", err)}
	}
	if err := os.WriteFile(expandedPath, skills.Content, 0o644); err != nil { //nolint:gosec // G306: Skill files are not secrets
		return &output.Error{Code: output.CodeAPI, Message: fmt.Sprintf("writing skill file: %v", err)}
	}

	// Also write to canonical location (~/.agents/skills/fizzy/)
	home, homeErr := os.UserHomeDir()
	if homeErr == nil {
		canonicalDir := filepath.Join(home, ".agents", "skills", "fizzy")
		canonicalFile := filepath.Join(canonicalDir, skillFilename)
		if canonicalFile != expandedPath {
			_ = os.MkdirAll(canonicalDir, 0o755)                   //nolint:gosec // G301: Skill files are not secrets
			_ = os.WriteFile(canonicalFile, skills.Content, 0o644) //nolint:gosec // G306: Skill files are not secrets
		}
	}

	fmt.Println()
	fmt.Println("Fizzy skill installed successfully!")
	fmt.Println()
	fmt.Printf("Location: %s\n", sanitizeLogValue(expandedPath))

	return nil
}

// normalizeSkillPath appends fizzy/SKILL.md to directory paths.
// Explicit file paths (any .md) are left as-is.
func normalizeSkillPath(path string) string {
	path = strings.TrimSpace(path)

	// Already points to a .md file — respect the user's choice
	if strings.HasSuffix(strings.ToLower(path), ".md") {
		return path
	}

	// Directory ending in "fizzy" — just append SKILL.md
	if strings.HasSuffix(path, "fizzy") || strings.HasSuffix(path, "fizzy/") ||
		strings.HasSuffix(path, "fizzy\\") {
		return filepath.Join(path, skillFilename)
	}

	// Bare directory — append fizzy/SKILL.md
	return filepath.Join(path, "fizzy", skillFilename)
}

func codexGlobalSkillPath() string {
	codexHome := strings.TrimSpace(os.Getenv("CODEX_HOME"))
	if codexHome == "" {
		return "~/.codex/skills/fizzy/SKILL.md"
	}
	return filepath.Join(codexHome, "skills", "fizzy", skillFilename)
}

// expandPath expands ~ to home directory.
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") || strings.HasPrefix(path, "~\\") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return home
	}
	return path
}

// sanitizeLogValue strips control characters from a string before output.
func sanitizeLogValue(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' {
			return -1
		}
		return r
	}, s)
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// linkSkillToClaude creates a symlink at ~/.claude/skills/fizzy pointing to
// the baseline skill directory. Returns (symlinkPath, notice, error).
func linkSkillToClaude() (string, string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("getting home directory: %w", err)
	}

	skillDir := filepath.Join(home, ".agents", "skills", "fizzy")
	symlinkDir := filepath.Join(home, ".claude", "skills")
	symlinkPath := filepath.Join(symlinkDir, "fizzy")

	if err := os.MkdirAll(symlinkDir, 0o755); err != nil { //nolint:gosec // G301: Skill files are not secrets
		return "", "", fmt.Errorf("creating symlink directory: %w", err)
	}

	// Remove existing entry at symlink path (idempotent)
	_ = os.Remove(symlinkPath)

	symlinkTarget := filepath.Join("..", "..", ".agents", "skills", "fizzy")
	notice := ""
	if err := os.Symlink(symlinkTarget, symlinkPath); err != nil {
		// Fallback: copy skill files directly
		notice = fmt.Sprintf("symlink failed (%v), copied files instead", err)
		if copyErr := copySkillFiles(skillDir, symlinkPath); copyErr != nil {
			return "", "", fmt.Errorf("creating symlink: %w (copy fallback also failed: %w)", err, copyErr)
		}
	}

	return symlinkPath, notice, nil
}

func copySkillFiles(src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil { //nolint:gosec // G301: Skill files are not secrets
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			return fmt.Errorf("skill directory contains subdirectory %q; copy fallback only supports flat files", e.Name())
		}
		data, err := os.ReadFile(filepath.Join(src, e.Name()))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dst, e.Name()), data, 0o644); err != nil { //nolint:gosec // G306: Skill files are not secrets
			return err
		}
	}
	return nil
}

// baselineSkillInstalled returns true if ~/.agents/skills/fizzy/SKILL.md exists.
func baselineSkillInstalled() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(home, ".agents", "skills", "fizzy", "SKILL.md"))
	return err == nil
}
