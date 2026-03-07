package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/basecamp/fizzy-cli/internal/skills"
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

func TestExpandPath(t *testing.T) {
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
		got := expandPath(tt.input)
		if got != tt.want {
			t.Errorf("expandPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestInstallSkillFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

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
	if string(got) != string(skills.Content) {
		t.Error("skill file content does not match embedded")
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

		// Pre-install skill files
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

		// Verify symlink target
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

		// Symlink still valid after second run
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

		// Pre-create a non-empty directory where the symlink would go
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

		// Verify SKILL.md was copied
		copied, err := os.ReadFile(filepath.Join(symlinkPath, "SKILL.md"))
		if err != nil {
			t.Fatal("SKILL.md not found in fallback copy location")
		}
		if string(copied) != string(skills.Content) {
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

	os.WriteFile(filepath.Join(src, "SKILL.md"), []byte("content"), 0o644)
	os.MkdirAll(filepath.Join(src, "subdir"), 0o755)

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
		os.MkdirAll(skillDir, 0o755)
		os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("content"), 0o644)

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

	if buf.String() != string(skills.Content) {
		t.Error("skill print output does not match embedded SKILL.md")
	}
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
