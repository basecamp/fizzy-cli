package skills

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSkillContentNonEmpty(t *testing.T) {
	if len(Content) == 0 {
		t.Fatal("embedded SKILL.md is empty")
	}
}

func TestSkillContentFresh(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	canonicalPath := filepath.Join(filepath.Dir(thisFile), "..", "..", "skills", "fizzy", "SKILL.md")

	canonical, err := os.ReadFile(canonicalPath)
	if err != nil {
		t.Fatalf("Failed to read canonical SKILL.md at %s: %v", canonicalPath, err)
	}

	if string(Content) != string(canonical) {
		t.Fatal("internal/skills/SKILL.md is out of sync with skills/fizzy/SKILL.md. Run 'make sync-skill' to update.")
	}
}
