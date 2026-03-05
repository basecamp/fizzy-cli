package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/basecamp/fizzy-cli/e2e/harness"
)

// createCommentWithAttachment uploads a file, creates a card, then adds a comment with the attachment.
// Returns the card number, comment ID, and expected filename.
func createCommentWithAttachment(t *testing.T, h *harness.Harness, boardID string) (int, string, string) {
	t.Helper()

	wd, _ := os.Getwd()
	fixturePath := filepath.Join(wd, "..", "testdata", "fixtures", "test_image.png")

	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		t.Skipf("test fixture not found at %s", fixturePath)
	}

	// Upload the file
	uploadResult := h.Run("upload", "file", fixturePath)
	if uploadResult.ExitCode != harness.ExitSuccess {
		t.Fatalf("failed to upload file: %s\nstdout: %s", uploadResult.Stderr, uploadResult.Stdout)
	}

	attachableSGID := uploadResult.GetDataString("attachable_sgid")
	if attachableSGID == "" {
		t.Fatalf("no attachable_sgid returned from upload\nstdout: %s", uploadResult.Stdout)
	}

	// Create a plain card
	title := fmt.Sprintf("Comment Attachment Test %d", time.Now().UnixNano())
	cardResult := h.Run("card", "create", "--board", boardID, "--title", title)
	if cardResult.ExitCode != harness.ExitSuccess {
		t.Fatalf("failed to create card: %s\nstdout: %s", cardResult.Stderr, cardResult.Stdout)
	}

	cardNumber := cardResult.GetNumberFromLocation()
	if cardNumber == 0 {
		cardNumber = cardResult.GetDataInt("number")
	}
	if cardNumber == 0 {
		t.Fatalf("failed to get card number from create (location: %s)", cardResult.GetLocation())
	}
	h.Cleanup.AddCard(cardNumber)

	// Create a comment with the attachment
	body := fmt.Sprintf(`<action-text-attachment sgid="%s"></action-text-attachment>`, attachableSGID)
	commentResult := h.Run("comment", "create", "--card", strconv.Itoa(cardNumber), "--body", body)
	if commentResult.ExitCode != harness.ExitSuccess {
		t.Fatalf("failed to create comment with attachment: %s\nstdout: %s", commentResult.Stderr, commentResult.Stdout)
	}

	commentID := commentResult.GetIDFromLocation()
	if commentID == "" {
		commentID = commentResult.GetDataString("id")
	}
	if commentID != "" {
		h.Cleanup.AddComment(commentID, cardNumber)
	}

	return cardNumber, commentID, "test_image.png"
}

func TestCommentAttachmentsShow(t *testing.T) {
	h := harness.New(t)
	defer h.Cleanup.CleanupAll(h)

	boardID := createAttachmentTestBoard(t, h)

	t.Run("shows attachments from comments", func(t *testing.T) {
		cardNumber, _, expectedFilename := createCommentWithAttachment(t, h, boardID)

		result := h.Run("comment", "attachments", "show", "--card", strconv.Itoa(cardNumber))

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

		arr := result.GetDataArray()
		if arr == nil {
			t.Fatalf("expected array response, got: %v", result.Response.Data)
		}

		if len(arr) != 1 {
			t.Errorf("expected 1 attachment, got %d", len(arr))
		}

		if len(arr) > 0 {
			attachment, ok := arr[0].(map[string]any)
			if !ok {
				t.Fatalf("expected attachment to be a map, got %T", arr[0])
			}

			if filename := attachment["filename"].(string); filename != expectedFilename {
				t.Errorf("expected filename %q, got %q", expectedFilename, filename)
			}

			if index := int(attachment["index"].(float64)); index != 1 {
				t.Errorf("expected index 1, got %d", index)
			}

			if downloadURL := attachment["download_url"].(string); downloadURL == "" {
				t.Error("expected non-empty download_url")
			}

			if commentID, ok := attachment["comment_id"].(string); !ok || commentID == "" {
				t.Error("expected non-empty comment_id")
			}
		}
	})

	t.Run("returns empty for card with no comment attachments", func(t *testing.T) {
		// Create a card with a plain text comment
		title := fmt.Sprintf("No Comment Attachment %d", time.Now().UnixNano())
		cardResult := h.Run("card", "create", "--board", boardID, "--title", title)
		if cardResult.ExitCode != harness.ExitSuccess {
			t.Fatalf("failed to create card: %s", cardResult.Stderr)
		}

		cardNumber := cardResult.GetNumberFromLocation()
		if cardNumber == 0 {
			cardNumber = cardResult.GetDataInt("number")
		}
		h.Cleanup.AddCard(cardNumber)

		commentResult := h.Run("comment", "create", "--card", strconv.Itoa(cardNumber), "--body", "Just a plain comment")
		if commentResult.ExitCode != harness.ExitSuccess {
			t.Fatalf("failed to create comment: %s", commentResult.Stderr)
		}
		commentID := commentResult.GetIDFromLocation()
		if commentID == "" {
			commentID = commentResult.GetDataString("id")
		}
		if commentID != "" {
			h.Cleanup.AddComment(commentID, cardNumber)
		}

		result := h.Run("comment", "attachments", "show", "--card", strconv.Itoa(cardNumber))

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d", harness.ExitSuccess, result.ExitCode)
		}

		if !result.Response.OK {
			t.Errorf("expected ok=true, error: %+v", result.Response.Error)
		}

		arr := result.GetDataArray()
		if arr != nil && len(arr) != 0 {
			t.Errorf("expected empty array, got %d items", len(arr))
		}
	})

	t.Run("requires card flag", func(t *testing.T) {
		result := h.Run("comment", "attachments", "show")

		if result.ExitCode == harness.ExitSuccess {
			t.Error("expected non-zero exit code for missing --card flag")
		}
	})
}

func TestCommentAttachmentsDownload(t *testing.T) {
	h := harness.New(t)
	defer h.Cleanup.CleanupAll(h)

	boardID := createAttachmentTestBoard(t, h)

	t.Run("downloads single comment attachment by index", func(t *testing.T) {
		cardNumber, _, expectedFilename := createCommentWithAttachment(t, h, boardID)

		tempDir, err := os.MkdirTemp("", "fizzy-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		originalDir, _ := os.Getwd()
		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("failed to change to temp dir: %v", err)
		}
		defer os.Chdir(originalDir)

		result := h.Run("comment", "attachments", "download", "--card", strconv.Itoa(cardNumber), "1")

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s\nstdout: %s",
				harness.ExitSuccess, result.ExitCode, result.Stderr, result.Stdout)
		}

		if !result.Response.OK {
			t.Errorf("expected ok=true, error: %+v", result.Response.Error)
		}

		// Verify file was downloaded
		downloadedFile := filepath.Join(tempDir, expectedFilename)
		if _, err := os.Stat(downloadedFile); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", downloadedFile)
		}

		// Check response data
		data := result.GetDataMap()
		if data == nil {
			t.Fatal("expected data map in response")
		}

		downloaded := int(data["downloaded"].(float64))
		if downloaded != 1 {
			t.Errorf("expected downloaded=1, got %d", downloaded)
		}
	})

	t.Run("downloads all comment attachments", func(t *testing.T) {
		cardNumber, _, _ := createCommentWithAttachment(t, h, boardID)

		tempDir, err := os.MkdirTemp("", "fizzy-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		originalDir, _ := os.Getwd()
		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("failed to change to temp dir: %v", err)
		}
		defer os.Chdir(originalDir)

		result := h.Run("comment", "attachments", "download", "--card", strconv.Itoa(cardNumber))

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s\nstdout: %s",
				harness.ExitSuccess, result.ExitCode, result.Stderr, result.Stdout)
		}

		if !result.Response.OK {
			t.Errorf("expected ok=true, error: %+v", result.Response.Error)
		}

		data := result.GetDataMap()
		downloaded := int(data["downloaded"].(float64))
		if downloaded != 1 {
			t.Errorf("expected downloaded=1, got %d", downloaded)
		}
	})

	t.Run("downloads with custom output filename", func(t *testing.T) {
		cardNumber, _, _ := createCommentWithAttachment(t, h, boardID)

		tempDir, err := os.MkdirTemp("", "fizzy-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		originalDir, _ := os.Getwd()
		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("failed to change to temp dir: %v", err)
		}
		defer os.Chdir(originalDir)

		customFilename := "my_comment_download.png"
		result := h.Run("comment", "attachments", "download", "--card", strconv.Itoa(cardNumber), "1", "-o", customFilename)

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s",
				harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if !result.Response.OK {
			t.Errorf("expected ok=true, error: %+v", result.Response.Error)
		}

		downloadedFile := filepath.Join(tempDir, customFilename)
		if _, err := os.Stat(downloadedFile); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", downloadedFile)
		}

		data := result.GetDataMap()
		files := data["files"].([]any)
		if len(files) > 0 {
			fileInfo := files[0].(map[string]any)
			savedTo := fileInfo["saved_to"].(string)
			if savedTo != customFilename {
				t.Errorf("expected saved_to=%q, got %q", customFilename, savedTo)
			}
		}
	})

	t.Run("returns error for no comment attachments", func(t *testing.T) {
		title := fmt.Sprintf("No Comment Attachment %d", time.Now().UnixNano())
		cardResult := h.Run("card", "create", "--board", boardID, "--title", title, "--description", "No files here")
		if cardResult.ExitCode != harness.ExitSuccess {
			t.Fatalf("failed to create card: %s", cardResult.Stderr)
		}

		cardNumber := cardResult.GetNumberFromLocation()
		if cardNumber == 0 {
			cardNumber = cardResult.GetDataInt("number")
		}
		h.Cleanup.AddCard(cardNumber)

		result := h.Run("comment", "attachments", "download", "--card", strconv.Itoa(cardNumber))

		if result.ExitCode != harness.ExitNotFound {
			t.Errorf("expected exit code %d, got %d\nstdout: %s",
				harness.ExitNotFound, result.ExitCode, result.Stdout)
		}

		if result.Response != nil && result.Response.OK {
			t.Error("expected ok=false")
		}
	})

	t.Run("returns error for invalid attachment index", func(t *testing.T) {
		cardNumber, _, _ := createCommentWithAttachment(t, h, boardID)

		result := h.Run("comment", "attachments", "download", "--card", strconv.Itoa(cardNumber), "abc")

		if result.ExitCode != harness.ExitInvalidArgs {
			t.Errorf("expected exit code %d, got %d\nstdout: %s",
				harness.ExitInvalidArgs, result.ExitCode, result.Stdout)
		}

		if result.Response != nil && result.Response.OK {
			t.Error("expected ok=false")
		}
	})

	t.Run("returns error for attachment index out of range", func(t *testing.T) {
		cardNumber, _, _ := createCommentWithAttachment(t, h, boardID)

		result := h.Run("comment", "attachments", "download", "--card", strconv.Itoa(cardNumber), "99")

		if result.ExitCode != harness.ExitInvalidArgs {
			t.Errorf("expected exit code %d, got %d\nstdout: %s",
				harness.ExitInvalidArgs, result.ExitCode, result.Stdout)
		}

		if result.Response != nil && result.Response.OK {
			t.Error("expected ok=false")
		}
	})

	t.Run("requires card flag", func(t *testing.T) {
		result := h.Run("comment", "attachments", "download")

		if result.ExitCode == harness.ExitSuccess {
			t.Error("expected non-zero exit code for missing --card flag")
		}
	})
}

func TestCardAttachmentsIncludeComments(t *testing.T) {
	h := harness.New(t)
	defer h.Cleanup.CleanupAll(h)

	boardID := createAttachmentTestBoard(t, h)

	t.Run("show includes comment attachments with flag", func(t *testing.T) {
		cardNumber, _, expectedFilename := createCommentWithAttachment(t, h, boardID)

		result := h.Run("card", "attachments", "show", strconv.Itoa(cardNumber), "--include-comments")

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s\nstdout: %s",
				harness.ExitSuccess, result.ExitCode, result.Stderr, result.Stdout)
		}

		if !result.Response.OK {
			t.Errorf("expected ok=true, error: %+v", result.Response.Error)
		}

		arr := result.GetDataArray()
		if len(arr) != 1 {
			t.Errorf("expected 1 attachment, got %d", len(arr))
		}

		if len(arr) > 0 {
			attachment := arr[0].(map[string]any)
			if filename := attachment["filename"].(string); filename != expectedFilename {
				t.Errorf("expected filename %q, got %q", expectedFilename, filename)
			}
		}
	})

	t.Run("show excludes comment attachments without flag", func(t *testing.T) {
		cardNumber, _, _ := createCommentWithAttachment(t, h, boardID)

		result := h.Run("card", "attachments", "show", strconv.Itoa(cardNumber))

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d", harness.ExitSuccess, result.ExitCode)
		}

		if !result.Response.OK {
			t.Errorf("expected ok=true, error: %+v", result.Response.Error)
		}

		arr := result.GetDataArray()
		if arr != nil && len(arr) != 0 {
			t.Errorf("expected empty array (no description attachments), got %d items", len(arr))
		}
	})

	t.Run("download includes comment attachments with flag", func(t *testing.T) {
		cardNumber, _, expectedFilename := createCommentWithAttachment(t, h, boardID)

		tempDir, err := os.MkdirTemp("", "fizzy-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		originalDir, _ := os.Getwd()
		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("failed to change to temp dir: %v", err)
		}
		defer os.Chdir(originalDir)

		result := h.Run("card", "attachments", "download", strconv.Itoa(cardNumber), "--include-comments")

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s\nstdout: %s",
				harness.ExitSuccess, result.ExitCode, result.Stderr, result.Stdout)
		}

		if !result.Response.OK {
			t.Errorf("expected ok=true, error: %+v", result.Response.Error)
		}

		downloadedFile := filepath.Join(tempDir, expectedFilename)
		if _, err := os.Stat(downloadedFile); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", downloadedFile)
		}

		data := result.GetDataMap()
		downloaded := int(data["downloaded"].(float64))
		if downloaded != 1 {
			t.Errorf("expected downloaded=1, got %d", downloaded)
		}
	})
}
