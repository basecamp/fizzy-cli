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

// createAttachmentTestBoard creates a board for attachment tests and adds it to cleanup
func createAttachmentTestBoard(t *testing.T, h *harness.Harness) string {
	t.Helper()
	name := fmt.Sprintf("Attachment Test Board %d", time.Now().UnixNano())
	result := h.Run("board", "create", "--name", name)
	if result.ExitCode != harness.ExitSuccess {
		t.Fatalf("failed to create test board: %s\nstdout: %s", result.Stderr, result.Stdout)
	}
	boardID := result.GetIDFromLocation()
	if boardID == "" {
		boardID = result.GetDataString("id")
	}
	if boardID == "" {
		t.Fatalf("no board ID returned (location: %s)", result.GetLocation())
	}
	h.Cleanup.AddBoard(boardID)
	return boardID
}

// createCardWithAttachment creates a card with an attachment for testing.
// Returns the card number and the expected filename.
func createCardWithAttachment(t *testing.T, h *harness.Harness, boardID string) (int, string) {
	t.Helper()

	// Get the path to the test image fixture
	wd, _ := os.Getwd()
	fixturePath := filepath.Join(wd, "..", "testdata", "fixtures", "test_image.png")

	// Check if fixture exists
	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		t.Skipf("test fixture not found at %s", fixturePath)
	}

	// Upload the file first
	uploadResult := h.Run("upload", "file", fixturePath)
	if uploadResult.ExitCode != harness.ExitSuccess {
		t.Fatalf("failed to upload file: %s\nstdout: %s", uploadResult.Stderr, uploadResult.Stdout)
	}

	attachableSGID := uploadResult.GetDataString("attachable_sgid")
	if attachableSGID == "" {
		t.Fatalf("no attachable_sgid returned from upload\nstdout: %s", uploadResult.Stdout)
	}

	// Create a card with the attachment in description
	title := fmt.Sprintf("Attachment Test Card %d", time.Now().UnixNano())
	description := fmt.Sprintf(`<action-text-attachment sgid="%s"></action-text-attachment>`, attachableSGID)

	cardResult := h.Run("card", "create", "--board", boardID, "--title", title, "--description", description)
	if cardResult.ExitCode != harness.ExitSuccess {
		t.Fatalf("failed to create card with attachment: %s\nstdout: %s", cardResult.Stderr, cardResult.Stdout)
	}

	cardNumber := cardResult.GetNumberFromLocation()
	if cardNumber == 0 {
		cardNumber = cardResult.GetDataInt("number")
	}
	if cardNumber == 0 {
		t.Fatalf("failed to get card number from create (location: %s)", cardResult.GetLocation())
	}

	return cardNumber, "test_image.png"
}

// createCardWithMultipleAttachments creates a card with two attachments.
func createCardWithMultipleAttachments(t *testing.T, h *harness.Harness, boardID string) (int, []string) {
	t.Helper()

	wd, _ := os.Getwd()
	imagePath := filepath.Join(wd, "..", "testdata", "fixtures", "test_image.png")
	docPath := filepath.Join(wd, "..", "testdata", "fixtures", "test_document.txt")

	// Check if fixtures exist
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		t.Skipf("test fixture not found at %s", imagePath)
	}
	if _, err := os.Stat(docPath); os.IsNotExist(err) {
		t.Skipf("test fixture not found at %s", docPath)
	}

	// Upload both files
	uploadImage := h.Run("upload", "file", imagePath)
	if uploadImage.ExitCode != harness.ExitSuccess {
		t.Fatalf("failed to upload image: %s", uploadImage.Stderr)
	}
	imageSGID := uploadImage.GetDataString("attachable_sgid")

	uploadDoc := h.Run("upload", "file", docPath)
	if uploadDoc.ExitCode != harness.ExitSuccess {
		t.Fatalf("failed to upload document: %s", uploadDoc.Stderr)
	}
	docSGID := uploadDoc.GetDataString("attachable_sgid")

	// Create card with both attachments
	title := fmt.Sprintf("Multi Attachment Test %d", time.Now().UnixNano())
	description := fmt.Sprintf(`<action-text-attachment sgid="%s"></action-text-attachment><p>Text between attachments</p><action-text-attachment sgid="%s"></action-text-attachment>`,
		imageSGID, docSGID)

	cardResult := h.Run("card", "create", "--board", boardID, "--title", title, "--description", description)
	if cardResult.ExitCode != harness.ExitSuccess {
		t.Fatalf("failed to create card: %s", cardResult.Stderr)
	}

	cardNumber := cardResult.GetNumberFromLocation()
	if cardNumber == 0 {
		cardNumber = cardResult.GetDataInt("number")
	}

	return cardNumber, []string{"test_image.png", "test_document.txt"}
}

func TestCardAttachmentsShow(t *testing.T) {
	h := harness.New(t)
	defer h.Cleanup.CleanupAll(h)

	boardID := createAttachmentTestBoard(t, h)

	t.Run("shows attachments on card with single attachment", func(t *testing.T) {
		cardNumber, expectedFilename := createCardWithAttachment(t, h, boardID)
		h.Cleanup.AddCard(cardNumber)

		result := h.Run("card", "attachments", "show", strconv.Itoa(cardNumber))

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

		// Check first attachment
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
		}
	})

	t.Run("shows multiple attachments", func(t *testing.T) {
		cardNumber, expectedFilenames := createCardWithMultipleAttachments(t, h, boardID)
		h.Cleanup.AddCard(cardNumber)

		result := h.Run("card", "attachments", "show", strconv.Itoa(cardNumber))

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s",
				harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if !result.Response.OK {
			t.Errorf("expected ok=true, error: %+v", result.Response.Error)
		}

		arr := result.GetDataArray()
		if len(arr) != 2 {
			t.Errorf("expected 2 attachments, got %d", len(arr))
		}

		// Verify filenames
		foundFilenames := make(map[string]bool)
		for _, item := range arr {
			attachment := item.(map[string]any)
			foundFilenames[attachment["filename"].(string)] = true
		}

		for _, expected := range expectedFilenames {
			if !foundFilenames[expected] {
				t.Errorf("expected to find attachment %q", expected)
			}
		}
	})

	t.Run("returns empty array for card without attachments", func(t *testing.T) {
		// Create a card without attachments
		title := fmt.Sprintf("No Attachment Card %d", time.Now().UnixNano())
		cardResult := h.Run("card", "create", "--board", boardID, "--title", title, "--description", "Just plain text")
		if cardResult.ExitCode != harness.ExitSuccess {
			t.Fatalf("failed to create card: %s", cardResult.Stderr)
		}

		cardNumber := cardResult.GetNumberFromLocation()
		if cardNumber == 0 {
			cardNumber = cardResult.GetDataInt("number")
		}
		h.Cleanup.AddCard(cardNumber)

		result := h.Run("card", "attachments", "show", strconv.Itoa(cardNumber))

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d", harness.ExitSuccess, result.ExitCode)
		}

		if !result.Response.OK {
			t.Errorf("expected ok=true, error: %+v", result.Response.Error)
		}

		// Data should be empty array or nil
		arr := result.GetDataArray()
		if arr != nil && len(arr) != 0 {
			t.Errorf("expected empty array, got %d items", len(arr))
		}
	})

	t.Run("returns not found for non-existent card", func(t *testing.T) {
		result := h.Run("card", "attachments", "show", "999999999")

		if result.ExitCode != harness.ExitNotFound {
			t.Errorf("expected exit code %d, got %d\nstdout: %s",
				harness.ExitNotFound, result.ExitCode, result.Stdout)
		}

		if result.Response != nil && result.Response.OK {
			t.Error("expected ok=false")
		}
	})
}

func TestCardAttachmentsDownload(t *testing.T) {
	h := harness.New(t)
	defer h.Cleanup.CleanupAll(h)

	boardID := createAttachmentTestBoard(t, h)

	t.Run("downloads single attachment by index", func(t *testing.T) {
		cardNumber, expectedFilename := createCardWithAttachment(t, h, boardID)
		h.Cleanup.AddCard(cardNumber)

		// Create temp directory for downloads
		tempDir, err := os.MkdirTemp("", "fizzy-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Change to temp directory and run download
		originalDir, _ := os.Getwd()
		if err := os.Chdir(tempDir); err != nil {
			t.Fatalf("failed to change to temp dir: %v", err)
		}
		defer os.Chdir(originalDir)

		result := h.Run("card", "attachments", "download", strconv.Itoa(cardNumber), "1")

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

	t.Run("downloads all attachments when no index specified", func(t *testing.T) {
		cardNumber, expectedFilenames := createCardWithMultipleAttachments(t, h, boardID)
		h.Cleanup.AddCard(cardNumber)

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

		result := h.Run("card", "attachments", "download", strconv.Itoa(cardNumber))

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s\nstdout: %s",
				harness.ExitSuccess, result.ExitCode, result.Stderr, result.Stdout)
		}

		if !result.Response.OK {
			t.Errorf("expected ok=true, error: %+v", result.Response.Error)
		}

		// Verify all files were downloaded
		for _, filename := range expectedFilenames {
			downloadedFile := filepath.Join(tempDir, filename)
			if _, err := os.Stat(downloadedFile); os.IsNotExist(err) {
				t.Errorf("expected file %s to exist", downloadedFile)
			}
		}

		// Check response data
		data := result.GetDataMap()
		downloaded := int(data["downloaded"].(float64))
		if downloaded != 2 {
			t.Errorf("expected downloaded=2, got %d", downloaded)
		}
	})

	t.Run("downloads with custom output filename", func(t *testing.T) {
		cardNumber, _ := createCardWithAttachment(t, h, boardID)
		h.Cleanup.AddCard(cardNumber)

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

		customFilename := "my_custom_download.png"
		result := h.Run("card", "attachments", "download", strconv.Itoa(cardNumber), "1", "-o", customFilename)

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s",
				harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if !result.Response.OK {
			t.Errorf("expected ok=true, error: %+v", result.Response.Error)
		}

		// Verify file was downloaded with custom name
		downloadedFile := filepath.Join(tempDir, customFilename)
		if _, err := os.Stat(downloadedFile); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", downloadedFile)
		}

		// Check response shows custom saved_to path
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

	t.Run("returns error for card with no attachments", func(t *testing.T) {
		// Create a card without attachments
		title := fmt.Sprintf("No Attachment Card %d", time.Now().UnixNano())
		cardResult := h.Run("card", "create", "--board", boardID, "--title", title, "--description", "No files here")
		if cardResult.ExitCode != harness.ExitSuccess {
			t.Fatalf("failed to create card: %s", cardResult.Stderr)
		}

		cardNumber := cardResult.GetNumberFromLocation()
		if cardNumber == 0 {
			cardNumber = cardResult.GetDataInt("number")
		}
		h.Cleanup.AddCard(cardNumber)

		result := h.Run("card", "attachments", "download", strconv.Itoa(cardNumber))

		if result.ExitCode != harness.ExitNotFound {
			t.Errorf("expected exit code %d, got %d\nstdout: %s",
				harness.ExitNotFound, result.ExitCode, result.Stdout)
		}

		if result.Response != nil && result.Response.OK {
			t.Error("expected ok=false")
		}
	})

	t.Run("returns error for invalid attachment index", func(t *testing.T) {
		cardNumber, _ := createCardWithAttachment(t, h, boardID)
		h.Cleanup.AddCard(cardNumber)

		result := h.Run("card", "attachments", "download", strconv.Itoa(cardNumber), "abc")

		if result.ExitCode != harness.ExitInvalidArgs {
			t.Errorf("expected exit code %d, got %d\nstdout: %s",
				harness.ExitInvalidArgs, result.ExitCode, result.Stdout)
		}

		if result.Response != nil && result.Response.OK {
			t.Error("expected ok=false")
		}
	})

	t.Run("returns error for attachment index out of range", func(t *testing.T) {
		cardNumber, _ := createCardWithAttachment(t, h, boardID)
		h.Cleanup.AddCard(cardNumber)

		result := h.Run("card", "attachments", "download", strconv.Itoa(cardNumber), "99")

		if result.ExitCode != harness.ExitInvalidArgs {
			t.Errorf("expected exit code %d, got %d\nstdout: %s",
				harness.ExitInvalidArgs, result.ExitCode, result.Stdout)
		}

		if result.Response != nil && result.Response.OK {
			t.Error("expected ok=false")
		}
	})

	t.Run("returns error for attachment index of zero", func(t *testing.T) {
		cardNumber, _ := createCardWithAttachment(t, h, boardID)
		h.Cleanup.AddCard(cardNumber)

		result := h.Run("card", "attachments", "download", strconv.Itoa(cardNumber), "0")

		if result.ExitCode != harness.ExitInvalidArgs {
			t.Errorf("expected exit code %d, got %d\nstdout: %s",
				harness.ExitInvalidArgs, result.ExitCode, result.Stdout)
		}

		if result.Response != nil && result.Response.OK {
			t.Error("expected ok=false")
		}
	})

	t.Run("returns not found for non-existent card", func(t *testing.T) {
		result := h.Run("card", "attachments", "download", "999999999", "1")

		if result.ExitCode != harness.ExitNotFound {
			t.Errorf("expected exit code %d, got %d\nstdout: %s",
				harness.ExitNotFound, result.ExitCode, result.Stdout)
		}

		if result.Response != nil && result.Response.OK {
			t.Error("expected ok=false")
		}
	})
}

func TestCardAttachmentsMissingArgs(t *testing.T) {
	h := harness.New(t)

	t.Run("show requires card number argument", func(t *testing.T) {
		result := h.Run("card", "attachments", "show")

		// Should fail with invalid args error
		if result.ExitCode == harness.ExitSuccess {
			t.Error("expected non-zero exit code for missing required argument")
		}
	})

	t.Run("download requires card number argument", func(t *testing.T) {
		result := h.Run("card", "attachments", "download")

		// Should fail with invalid args error
		if result.ExitCode == harness.ExitSuccess {
			t.Error("expected non-zero exit code for missing required argument")
		}
	})
}
