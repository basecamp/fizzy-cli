package commands

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/basecamp/cli/surface"
)

// TestGenerateSurfaceSnapshot writes SURFACE.txt from the current command tree.
// Run with: go test ./internal/commands/ -run TestGenerateSurfaceSnapshot -generate
//
// Only runs when -generate flag is provided (see Makefile surface-snapshot target).
func TestGenerateSurfaceSnapshot(t *testing.T) {
	if os.Getenv("GENERATE_SURFACE") == "" {
		t.Skip("set GENERATE_SURFACE=1 to regenerate SURFACE.txt")
	}

	configureCLIUX()
	initAllHelpFlags(rootCmd)
	snapshot := surface.SnapshotString(rootCmd) + "\n"

	_, thisFile, _, _ := runtime.Caller(0)
	goldenPath := filepath.Join(filepath.Dir(thisFile), "..", "..", "SURFACE.txt")

	if err := os.WriteFile(goldenPath, []byte(snapshot), 0644); err != nil {
		t.Fatalf("Failed to write SURFACE.txt: %v", err)
	}
	t.Logf("Wrote %s", goldenPath)
}
