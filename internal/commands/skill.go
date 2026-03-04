package commands

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

const skillSourceURL = "https://raw.githubusercontent.com/basecamp/fizzy-cli/refs/heads/master/skills/fizzy/SKILL.md"
const skillFilename = "SKILL.md"

// SkillLocation represents a predefined skill installation location
type SkillLocation struct {
	Name        string
	Path        string
	Description string
}

var skillLocations = []SkillLocation{
	{
		Name:        "Claude Code (Global)",
		Path:        "~/.claude/skills/fizzy/SKILL.md",
		Description: "Available in all Claude Code projects",
	},
	{
		Name:        "Claude Code (Project)",
		Path:        ".claude/skills/fizzy/SKILL.md",
		Description: "Available only in this project",
	},
	{
		Name:        "OpenCode (Global)",
		Path:        "~/.config/opencode/skill/fizzy/SKILL.md",
		Description: "Available in all OpenCode projects",
	},
	{
		Name:        "OpenCode (Project)",
		Path:        ".opencode/skill/fizzy/SKILL.md",
		Description: "Available only in this project",
	},
	{
		Name:        "Codex (Global)",
		Path:        codexGlobalSkillPath(),
		Description: "Available in all Codex projects",
	},
}

var skillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Install Fizzy skill file",
	Long:  "Install the Fizzy SKILL.md file for use with Codex, Claude Code, or OpenCode.",
	Run:   runSkill,
}

func init() {
	rootCmd.AddCommand(skillCmd)
}

func runSkill(cmd *cobra.Command, args []string) {
	fmt.Println()
	fmt.Println("Fizzy Skill Installation")
	fmt.Println()

	// Build options for the select prompt
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
		os.Exit(0)
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
			os.Exit(0)
		}

		// Smart path handling
		selectedPath = normalizeSkillPath(selectedPath)
	}

	// Expand home directory
	expandedPath := expandPath(selectedPath)

	// Check if file already exists
	if fileExists(expandedPath) {
		var overwrite bool
		err = huh.NewConfirm().
			Title(fmt.Sprintf("File already exists at %s. Overwrite?", selectedPath)).
			Value(&overwrite).
			Run()

		if err != nil || !overwrite {
			fmt.Println("Installation cancelled.")
			os.Exit(0)
		}
	}

	// Download and install
	fmt.Print("Downloading skill file... ")
	content, err := downloadSkillFile()
	if err != nil {
		fmt.Println("✗")
		fmt.Printf("Error downloading skill file: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓")

	fmt.Print("Installing to " + selectedPath + "... ")
	err = installSkillFile(expandedPath, content)
	if err != nil {
		fmt.Println("✗")
		fmt.Printf("Error installing skill file: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✓")

	fmt.Println()
	fmt.Println("Fizzy skill installed successfully!")
	fmt.Println()
	fmt.Printf("Location: %s\n", expandedPath)
}

// normalizeSkillPath ensures the path ends with SKILL.md and has fizzy directory
func normalizeSkillPath(path string) string {
	path = strings.TrimSpace(path)

	// If path already ends with SKILL.md, return as is
	if strings.HasSuffix(path, skillFilename) {
		return path
	}

	// If path ends with .md but not SKILL.md, treat as invalid and append
	if strings.HasSuffix(strings.ToLower(path), ".md") {
		// User specified a different .md file, append SKILL.md to directory
		dir := filepath.Dir(path)
		return filepath.Join(dir, "fizzy", skillFilename)
	}

	// Check if path ends with fizzy directory
	if strings.HasSuffix(path, "fizzy") || strings.HasSuffix(path, "fizzy/") || strings.HasSuffix(path, "fizzy\\") {
		return filepath.Join(path, skillFilename)
	}

	// Path is a directory, add fizzy/SKILL.md
	return filepath.Join(path, "fizzy", skillFilename)
}

func codexGlobalSkillPath() string {
	codexHome := strings.TrimSpace(os.Getenv("CODEX_HOME"))
	if codexHome == "" {
		return "~/.codex/skills/fizzy/SKILL.md"
	}
	return filepath.Join(codexHome, "skills", "fizzy", skillFilename)
}

// expandPath expands ~ to home directory (works on both Unix and Windows)
func expandPath(path string) string {
	// Handle both ~/path (Unix) and ~\path (Windows)
	if strings.HasPrefix(path, "~/") || strings.HasPrefix(path, "~\\") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	// Handle bare ~ (just home directory)
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return home
	}
	return path
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// downloadSkillFile downloads the SKILL.md content from GitHub
func downloadSkillFile() ([]byte, error) {
	req, err := http.NewRequestWithContext(context.Background(), "GET", skillSourceURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return content, nil
}

// installSkillFile writes the skill file to the specified path
func installSkillFile(path string, content []byte) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(path, content, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
