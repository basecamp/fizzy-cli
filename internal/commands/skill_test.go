package commands

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/basecamp/fizzy-cli/internal/config"
	"github.com/basecamp/fizzy-cli/skills"
)

func TestSanitizeLogValue(t *testing.T) {
	t.Run("strips newlines and tabs", func(t *testing.T) {
		input := "/safe/path\n/injected/line"
		result := sanitizeLogValue(input)
		if result != "/safe/path/injected/line" {
			t.Errorf("expected '/safe/path/injected/line', got '%s'", result)
		}
	})

	t.Run("strips carriage returns", func(t *testing.T) {
		input := "/safe/path\r\nOverwrite? yes"
		result := sanitizeLogValue(input)
		if result != "/safe/pathOverwrite? yes" {
			t.Errorf("expected '/safe/pathOverwrite? yes', got '%s'", result)
		}
	})

	t.Run("strips tabs", func(t *testing.T) {
		input := "/path/with\ttab"
		result := sanitizeLogValue(input)
		if result != "/path/withtab" {
			t.Errorf("expected '/path/withtab', got '%s'", result)
		}
	})

	t.Run("passes through clean paths", func(t *testing.T) {
		input := "/Users/test/.claude/skills/fizzy/SKILL.md"
		result := sanitizeLogValue(input)
		if result != input {
			t.Errorf("expected unchanged path, got '%s'", result)
		}
	})
}

func TestNormalizeSkillPath(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"~/.claude/skills/fizzy/SKILL.md", "~/.claude/skills/fizzy/SKILL.md"},
		{"/tmp/skills", filepath.Join("/tmp/skills", "fizzy", "SKILL.md")},
		{"/tmp/fizzy", filepath.Join("/tmp/fizzy", "SKILL.md")},
		{"/tmp/fizzy/", filepath.Join("/tmp/fizzy", "SKILL.md")},
		{"/tmp/other.md", "/tmp/other.md"},
		{"  ~/.claude/skills/fizzy/SKILL.md  ", "~/.claude/skills/fizzy/SKILL.md"},
	}
	for _, tt := range tests {
		got := normalizeSkillPath(tt.input)
		if got != tt.want {
			t.Errorf("normalizeSkillPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExpandSkillPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory")
	}

	tests := []struct {
		input, want string
	}{
		{"~/foo/bar", filepath.Join(home, "foo/bar")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
		{"~", home},
	}
	for _, tt := range tests {
		got := expandSkillPath(tt.input)
		if got != tt.want {
			t.Errorf("expandSkillPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestInstallSkillFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	SetVersion("1.2.3")
	defer SetVersion("dev")

	skillPath, err := installSkillFiles()
	if err != nil {
		t.Fatalf("installSkillFiles() error = %v", err)
	}

	expectedPath := filepath.Join(home, ".agents", "skills", "fizzy", "SKILL.md")
	if skillPath != expectedPath {
		t.Errorf("skillPath = %q, want %q", skillPath, expectedPath)
	}

	got, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("skill file not created: %v", err)
	}
	embedded, err := skills.FS.ReadFile("fizzy/SKILL.md")
	if err != nil {
		t.Fatalf("reading embedded skill: %v", err)
	}
	if string(got) != string(embedded) {
		t.Error("skill file content does not match embedded")
	}

	versionPath := filepath.Join(home, ".agents", "skills", "fizzy", installedVersionFile)
	versionData, err := os.ReadFile(versionPath)
	if err != nil {
		t.Fatalf("installed version file not created: %v", err)
	}
	if string(versionData) != "1.2.3" {
		t.Errorf("installed version = %q, want %q", versionData, "1.2.3")
	}
}

func TestInstallSkillFilesIdempotent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if _, err := installSkillFiles(); err != nil {
		t.Fatalf("first installSkillFiles() error = %v", err)
	}
	if _, err := installSkillFiles(); err != nil {
		t.Fatalf("second installSkillFiles() error = %v", err)
	}
}

func TestLinkSkillToClaude(t *testing.T) {
	t.Run("creates symlink", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)

		if _, err := installSkillFiles(); err != nil {
			t.Fatal(err)
		}

		symlinkPath, notice, err := linkSkillToClaude()
		if err != nil {
			t.Fatalf("linkSkillToClaude() error = %v", err)
		}

		expectedSymlink := filepath.Join(home, ".claude", "skills", "fizzy")
		if symlinkPath != expectedSymlink {
			t.Errorf("symlinkPath = %q, want %q", symlinkPath, expectedSymlink)
		}
		if notice != "" {
			t.Errorf("unexpected notice: %q", notice)
		}

		linkTarget, err := os.Readlink(symlinkPath)
		if err != nil {
			t.Fatalf("symlink not created: %v", err)
		}
		wantTarget := filepath.Join("..", "..", ".agents", "skills", "fizzy")
		if linkTarget != wantTarget {
			t.Errorf("symlink target = %q, want %q", linkTarget, wantTarget)
		}
	})

	t.Run("idempotent", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)

		if _, err := installSkillFiles(); err != nil {
			t.Fatal(err)
		}

		if _, _, err := linkSkillToClaude(); err != nil {
			t.Fatalf("first linkSkillToClaude() error = %v", err)
		}
		if _, _, err := linkSkillToClaude(); err != nil {
			t.Fatalf("second linkSkillToClaude() error = %v", err)
		}

		symlinkPath := filepath.Join(home, ".claude", "skills", "fizzy")
		if _, err := os.Readlink(symlinkPath); err != nil {
			t.Fatalf("symlink broken after second install: %v", err)
		}
	})

	t.Run("fallback on non-empty dir", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)

		if _, err := installSkillFiles(); err != nil {
			t.Fatal(err)
		}

		symlinkPath := filepath.Join(home, ".claude", "skills", "fizzy")
		if err := os.MkdirAll(symlinkPath, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(symlinkPath, "blocker.txt"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}

		_, notice, err := linkSkillToClaude()
		if err != nil {
			t.Fatalf("linkSkillToClaude() error = %v (fallback should have handled it)", err)
		}
		if notice == "" {
			t.Error("expected notice about fallback")
		}

		copied, err := os.ReadFile(filepath.Join(symlinkPath, "SKILL.md"))
		if err != nil {
			t.Fatal("SKILL.md not found in fallback copy location")
		}
		embedded, err := skills.FS.ReadFile("fizzy/SKILL.md")
		if err != nil {
			t.Fatalf("reading embedded skill: %v", err)
		}
		if string(copied) != string(embedded) {
			t.Error("fallback copy content does not match embedded")
		}
	})
}

func TestCopySkillFiles(t *testing.T) {
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "dest")

	if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte("skill content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "extra.txt"), []byte("extra"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := copySkillFiles(src, dst); err != nil {
		t.Fatalf("copySkillFiles() error = %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dst, "SKILL.md"))
	if err != nil {
		t.Fatalf("reading SKILL.md: %v", err)
	}
	if string(got) != "skill content" {
		t.Errorf("SKILL.md = %q, want %q", got, "skill content")
	}
}

func TestCopySkillFilesRejectsSubdirs(t *testing.T) {
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "dest")

	if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte("content"), 0o644); err != nil {
		t.Fatalf("writing SKILL.md: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(src, "subdir"), 0o755); err != nil {
		t.Fatalf("creating subdir: %v", err)
	}

	err := copySkillFiles(src, dst)
	if err == nil {
		t.Fatal("expected error for subdirectory in source")
	}
	if !strings.Contains(err.Error(), "subdirectory") {
		t.Errorf("error = %q, want subdirectory rejection message", err)
	}
}

func TestBaselineSkillInstalled(t *testing.T) {
	t.Run("true when installed", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)

		skillDir := filepath.Join(home, ".agents", "skills", "fizzy")
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatalf("creating skill directory: %v", err)
		}
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("content"), 0o644); err != nil {
			t.Fatalf("writing SKILL.md: %v", err)
		}

		if !baselineSkillInstalled() {
			t.Error("expected true when skill file exists")
		}
	})

	t.Run("false when not installed", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)

		if baselineSkillInstalled() {
			t.Error("expected false when skill file does not exist")
		}
	})
}

func TestInstalledSkillVersion(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if got := installedSkillVersion(); got != "" {
		t.Fatalf("installedSkillVersion() = %q, want empty string", got)
	}

	dir := filepath.Join(home, ".agents", "skills", "fizzy")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, installedVersionFile), []byte("1.2.3\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := installedSkillVersion(); got != "1.2.3" {
		t.Fatalf("installedSkillVersion() = %q, want %q", got, "1.2.3")
	}
}

func TestSkillPrintOutputMatchesEmbedded(t *testing.T) {
	defer ResetTestMode()
	ResetTestMode()
	cfgAgent = true

	var buf bytes.Buffer
	skillCmd.SetOut(&buf)
	defer skillCmd.SetOut(nil)

	err := runSkill(skillCmd, nil)
	if err != nil {
		t.Fatalf("skill print RunE() error = %v", err)
	}

	embedded, err := skills.FS.ReadFile("fizzy/SKILL.md")
	if err != nil {
		t.Fatalf("reading embedded skill: %v", err)
	}
	if buf.String() != string(embedded) {
		t.Error("skill print output does not match embedded SKILL.md")
	}
}

func TestSkillInstallRunE(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}

	cmd := newSkillInstallCmd()
	cmd.SetContext(context.Background())
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("RunE() error = %v", err)
	}

	skillFile := filepath.Join(home, ".agents", "skills", "fizzy", "SKILL.md")
	got, err := os.ReadFile(skillFile)
	if err != nil {
		t.Fatalf("skill file not created: %v", err)
	}
	embedded, err := skills.FS.ReadFile("fizzy/SKILL.md")
	if err != nil {
		t.Fatalf("reading embedded skill: %v", err)
	}
	if string(got) != string(embedded) {
		t.Error("skill file content does not match embedded")
	}

	symlinkPath := filepath.Join(home, ".claude", "skills", "fizzy")
	linkTarget, err := os.Readlink(symlinkPath)
	if err != nil {
		t.Fatalf("symlink not created: %v", err)
	}
	wantTarget := filepath.Join("..", "..", ".agents", "skills", "fizzy")
	if linkTarget != wantTarget {
		t.Errorf("symlink target = %q, want %q", linkTarget, wantTarget)
	}
}

func TestSkillInstallNoClaude(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("PATH", home)

	cmd := newSkillInstallCmd()
	cmd.SetContext(context.Background())
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("RunE() error = %v", err)
	}

	skillFile := filepath.Join(home, ".agents", "skills", "fizzy", "SKILL.md")
	if _, err := os.Stat(skillFile); err != nil {
		t.Errorf("skill file should exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude")); err == nil {
		t.Error("~/.claude should not be created when Claude is not detected")
	}
}

func TestRefreshSkillsIfVersionChanged(t *testing.T) {
	t.Run("sentinel missing refreshes and updates sentinel", func(t *testing.T) {
		home := t.TempDir()
		configDir := t.TempDir()
		t.Setenv("HOME", home)
		config.SetTestConfigDir(configDir)
		defer config.ResetTestConfigDir()
		SetVersion("1.0.0")
		defer SetVersion("dev")

		if _, err := installSkillFiles(); err != nil {
			t.Fatal(err)
		}

		if !RefreshSkillsIfVersionChanged() {
			t.Fatal("expected refresh when sentinel is missing")
		}

		sentinel, err := os.ReadFile(filepath.Join(configDir, ".last-run-version"))
		if err != nil {
			t.Fatalf("reading sentinel: %v", err)
		}
		if string(sentinel) != "1.0.0" {
			t.Fatalf("sentinel = %q, want %q", sentinel, "1.0.0")
		}
	})

	t.Run("matching sentinel does not refresh", func(t *testing.T) {
		home := t.TempDir()
		configDir := t.TempDir()
		t.Setenv("HOME", home)
		config.SetTestConfigDir(configDir)
		defer config.ResetTestConfigDir()
		SetVersion("1.0.0")
		defer SetVersion("dev")

		if _, err := installSkillFiles(); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(configDir, ".last-run-version"), []byte("1.0.0"), 0o644); err != nil {
			t.Fatal(err)
		}

		if RefreshSkillsIfVersionChanged() {
			t.Fatal("did not expect refresh when sentinel matches")
		}
	})

	t.Run("mismatched sentinel refreshes and updates installed version", func(t *testing.T) {
		home := t.TempDir()
		configDir := t.TempDir()
		t.Setenv("HOME", home)
		config.SetTestConfigDir(configDir)
		defer config.ResetTestConfigDir()
		SetVersion("2.0.0")
		defer SetVersion("dev")

		if _, err := installSkillFiles(); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(configDir, ".last-run-version"), []byte("1.0.0"), 0o644); err != nil {
			t.Fatal(err)
		}

		if !RefreshSkillsIfVersionChanged() {
			t.Fatal("expected refresh when sentinel mismatches")
		}
		if got := installedSkillVersion(); got != "2.0.0" {
			t.Fatalf("installedSkillVersion() = %q, want %q", got, "2.0.0")
		}
	})

	t.Run("dev build skips refresh", func(t *testing.T) {
		SetVersion("dev")
		if RefreshSkillsIfVersionChanged() {
			t.Fatal("did not expect refresh for dev build")
		}
	})
}

func TestRefreshAllInstalledSkills(t *testing.T) {
	t.Run("multiple locations", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("PATH", home)
		SetVersion("5.0.0")
		defer SetVersion("dev")

		embedded, err := skills.FS.ReadFile("fizzy/SKILL.md")
		if err != nil {
			t.Fatal(err)
		}

		baseline := filepath.Join(home, ".agents", "skills", "fizzy")
		if err := os.MkdirAll(baseline, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(baseline, "SKILL.md"), []byte("old"), 0o644); err != nil {
			t.Fatal(err)
		}

		claudeSkill := filepath.Join(home, ".claude", "skills", "fizzy")
		if err := os.MkdirAll(claudeSkill, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(claudeSkill, "SKILL.md"), []byte("old"), 0o644); err != nil {
			t.Fatal(err)
		}

		opencode := filepath.Join(home, ".config", "opencode", "skill", "fizzy")
		if err := os.MkdirAll(opencode, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(opencode, "SKILL.md"), []byte("old"), 0o644); err != nil {
			t.Fatal(err)
		}

		if !refreshAllInstalledSkills() {
			t.Fatal("expected refreshAllInstalledSkills to succeed")
		}

		for _, path := range []string{
			filepath.Join(baseline, "SKILL.md"),
			filepath.Join(claudeSkill, "SKILL.md"),
			filepath.Join(opencode, "SKILL.md"),
		} {
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("reading %s: %v", path, err)
			}
			if string(got) != string(embedded) {
				t.Fatalf("content mismatch at %s", path)
			}
		}

		if got := installedSkillVersion(); got != "5.0.0" {
			t.Fatalf("installedSkillVersion() = %q, want %q", got, "5.0.0")
		}
	})

	t.Run("skips absent locations", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("PATH", home)
		SetVersion("5.0.0")
		defer SetVersion("dev")

		baseline := filepath.Join(home, ".agents", "skills", "fizzy")
		if err := os.MkdirAll(baseline, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(baseline, "SKILL.md"), []byte("old"), 0o644); err != nil {
			t.Fatal(err)
		}

		if !refreshAllInstalledSkills() {
			t.Fatal("expected refreshAllInstalledSkills to succeed")
		}

		if _, err := os.Stat(filepath.Join(home, ".claude", "skills", "fizzy", "SKILL.md")); !os.IsNotExist(err) {
			t.Fatal("should not create skill at absent location")
		}
	})
}

func TestRepairClaudeSkillLink(t *testing.T) {
	t.Run("repairs broken symlink", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)

		if err := os.MkdirAll(filepath.Join(home, ".claude", "skills"), 0o755); err != nil {
			t.Fatal(err)
		}
		baseline := filepath.Join(home, ".agents", "skills", "fizzy")
		if err := os.MkdirAll(baseline, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(baseline, "SKILL.md"), []byte("skill"), 0o644); err != nil {
			t.Fatal(err)
		}

		symlinkPath := filepath.Join(home, ".claude", "skills", "fizzy")
		if err := os.Symlink("/nonexistent/target", symlinkPath); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(symlinkPath); !os.IsNotExist(err) {
			t.Fatal("expected broken symlink before repair")
		}

		repairClaudeSkillLink()

		if _, err := os.Stat(filepath.Join(symlinkPath, "SKILL.md")); err != nil {
			t.Fatalf("expected skill reachable through repaired symlink: %v", err)
		}
	})

	t.Run("healthy symlink left alone", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)

		baseline := filepath.Join(home, ".agents", "skills", "fizzy")
		if err := os.MkdirAll(baseline, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(baseline, "SKILL.md"), []byte("skill"), 0o644); err != nil {
			t.Fatal(err)
		}

		symlinkDir := filepath.Join(home, ".claude", "skills")
		if err := os.MkdirAll(symlinkDir, 0o755); err != nil {
			t.Fatal(err)
		}
		symlinkPath := filepath.Join(symlinkDir, "fizzy")
		if err := os.Symlink(baseline, symlinkPath); err != nil {
			t.Fatal(err)
		}

		targetBefore, err := os.Readlink(symlinkPath)
		if err != nil {
			t.Fatal(err)
		}
		repairClaudeSkillLink()
		targetAfter, err := os.Readlink(symlinkPath)
		if err != nil {
			t.Fatal(err)
		}
		if targetBefore != targetAfter {
			t.Fatalf("healthy symlink should not be modified: before %q after %q", targetBefore, targetAfter)
		}
	})
}

func TestJoinNames(t *testing.T) {
	tests := []struct {
		names []string
		want  string
	}{
		{nil, ""},
		{[]string{"A"}, "A"},
		{[]string{"A", "B"}, "A and B"},
		{[]string{"A", "B", "C"}, "A, B, and C"},
	}
	for _, tt := range tests {
		got := joinNames(tt.names)
		if got != tt.want {
			t.Errorf("joinNames(%v) = %q, want %q", tt.names, got, tt.want)
		}
	}
}
