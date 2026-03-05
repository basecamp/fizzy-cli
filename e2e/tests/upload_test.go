package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/basecamp/fizzy-cli/e2e/harness"
)

func TestUploadFile(t *testing.T) {
	h := harness.New(t)

	// Get the path to the test image fixture
	wd, _ := os.Getwd()
	fixturePath := filepath.Join(wd, "..", "testdata", "fixtures", "test_image.png")

	// Check if fixture exists
	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		t.Skipf("test fixture not found at %s", fixturePath)
	}

	t.Run("uploads file and returns signed_id", func(t *testing.T) {
		result := h.Run("upload", "file", fixturePath)

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s\nstdout: %s",
				harness.ExitSuccess, result.ExitCode, result.Stderr, result.Stdout)
		}

		if result.Response == nil {
			t.Fatalf("expected JSON response, got nil\nstdout: %s", result.Stdout)
		}

		if !result.Response.OK {
			t.Errorf("expected ok=true, error: %+v", result.Response.Error)
		}

		signedID := result.GetDataString("signed_id")
		if signedID == "" {
			t.Error("expected signed_id in response")
		}
	})
}

func TestUploadTextFile(t *testing.T) {
	h := harness.New(t)

	// Get the path to the test document fixture
	wd, _ := os.Getwd()
	fixturePath := filepath.Join(wd, "..", "testdata", "fixtures", "test_document.txt")

	// Check if fixture exists
	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		t.Skipf("test fixture not found at %s", fixturePath)
	}

	t.Run("uploads text file and returns signed_id", func(t *testing.T) {
		result := h.Run("upload", "file", fixturePath)

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s\nstdout: %s",
				harness.ExitSuccess, result.ExitCode, result.Stderr, result.Stdout)
		}

		if result.Response == nil {
			t.Fatalf("expected JSON response, got nil\nstdout: %s", result.Stdout)
		}

		if !result.Response.OK {
			t.Errorf("expected ok=true, error: %+v", result.Response.Error)
		}

		signedID := result.GetDataString("signed_id")
		if signedID == "" {
			t.Error("expected signed_id in response")
		}
	})
}

func TestUploadFileNotFound(t *testing.T) {
	h := harness.New(t)

	t.Run("returns error for non-existent file", func(t *testing.T) {
		result := h.Run("upload", "file", "/path/to/nonexistent/file.png")

		// Should fail with validation error or general error
		if result.ExitCode == harness.ExitSuccess {
			t.Error("expected failure for non-existent file")
		}

		if result.Response != nil && result.Response.OK {
			t.Error("expected ok=false")
		}
	})
}

func TestUploadMissingPath(t *testing.T) {
	h := harness.New(t)

	t.Run("fails without file path argument", func(t *testing.T) {
		result := h.Run("upload", "file")

		// Should fail with invalid args
		if result.ExitCode != harness.ExitInvalidArgs && result.ExitCode != harness.ExitError {
			t.Errorf("expected exit code %d or %d, got %d",
				harness.ExitInvalidArgs, harness.ExitError, result.ExitCode)
		}
	})
}
