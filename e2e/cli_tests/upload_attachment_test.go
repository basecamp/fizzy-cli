package clitests

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/basecamp/fizzy-cli/e2e/harness"
)

type uploadRef struct {
	Path           string
	Filename       string
	SignedID       string
	AttachableSGID string
}

func fixtureFile(t *testing.T, name string) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	path := filepath.Join(wd, "..", "testdata", "fixtures", name)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skipf("test fixture not found at %s", path)
	}
	return path
}

func uploadFixture(t *testing.T, h *harness.Harness, name string) uploadRef {
	t.Helper()
	path := fixtureFile(t, name)
	result := h.Run("upload", "file", path)
	assertOK(t, result)

	ref := uploadRef{
		Path:           path,
		Filename:       filepath.Base(path),
		SignedID:       result.GetDataString("signed_id"),
		AttachableSGID: result.GetDataString("attachable_sgid"),
	}
	if ref.SignedID == "" {
		t.Fatal("expected signed_id in upload response")
	}
	if ref.AttachableSGID == "" {
		// Current attachment embedding uses attachable_sgid; keep the error explicit.
		t.Fatal("expected attachable_sgid in upload response")
	}
	return ref
}

func createCardWithAttachment(t *testing.T, h *harness.Harness, boardID string, ref uploadRef) int {
	t.Helper()
	description := fmt.Sprintf(`<action-text-attachment sgid="%s"></action-text-attachment>`, ref.AttachableSGID)
	result := h.Run("card", "create",
		"--board", boardID,
		"--title", fmt.Sprintf("Attachment Card %d", time.Now().UnixNano()),
		"--description", description,
	)
	assertOK(t, result)
	num := result.GetNumberFromLocation()
	if num == 0 {
		num = result.GetDataInt("number")
	}
	if num == 0 {
		t.Fatal("no card number in create response")
	}
	t.Cleanup(func() { newHarness(t).Run("card", "delete", strconv.Itoa(num)) })
	return num
}

func TestUploadFile(t *testing.T) {
	h := newHarness(t)
	for _, fixtureName := range []string{"test_image.png", "test_document.txt"} {
		fixtureName := fixtureName
		t.Run(fixtureName, func(t *testing.T) {
			ref := uploadFixture(t, h, fixtureName)
			if ref.SignedID == "" || ref.AttachableSGID == "" {
				t.Fatalf("expected both signed_id and attachable_sgid for %s", fixtureName)
			}
		})
	}
}

func TestUploadFileNotFound(t *testing.T) {
	h := newHarness(t)
	result := h.Run("upload", "file", "/path/to/nonexistent/file.png")
	if result.ExitCode == harness.ExitSuccess {
		t.Fatal("expected failure for non-existent file")
	}
	if result.Response != nil && result.Response.OK {
		t.Fatal("expected ok=false for non-existent file")
	}
}

func TestCardAttachmentRoundTrip(t *testing.T) {
	h := newHarness(t)
	boardID := createBoard(t, h)
	ref := uploadFixture(t, h, "test_image.png")
	cardNumber := createCardWithAttachment(t, h, boardID, ref)
	cardStr := strconv.Itoa(cardNumber)

	show := h.Run("card", "attachments", "show", cardStr)
	assertOK(t, show)
	attachments := show.GetDataArray()
	if len(attachments) == 0 {
		t.Fatal("expected at least one card attachment")
	}
	first := asMap(attachments[0])
	if first == nil {
		t.Fatalf("expected attachment object, got %T", attachments[0])
	}
	if got := mapValueString(first, "filename"); got != ref.Filename {
		t.Fatalf("expected filename %q, got %q", ref.Filename, got)
	}

	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "downloaded-card-attachment.png")
	download := h.Run("card", "attachments", "download", cardStr, "1", "-o", outputPath)
	assertOK(t, download)
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("expected downloaded file at %s: %v", outputPath, err)
	}
}

func TestCommentAttachmentRoundTrip(t *testing.T) {
	h := newHarness(t)
	boardID := createBoard(t, h)
	cardNumber := createCard(t, h, boardID)
	ref := uploadFixture(t, h, "test_image.png")
	body := fmt.Sprintf(`<action-text-attachment sgid="%s"></action-text-attachment>`, ref.AttachableSGID)
	commentID := createComment(t, h, cardNumber, body)
	cardStr := strconv.Itoa(cardNumber)

	show := h.Run("comment", "attachments", "show", "--card", cardStr)
	assertOK(t, show)
	attachments := show.GetDataArray()
	if len(attachments) == 0 {
		t.Fatal("expected at least one comment attachment")
	}
	first := asMap(attachments[0])
	if first == nil {
		t.Fatalf("expected attachment object, got %T", attachments[0])
	}
	if got := mapValueString(first, "filename"); got != ref.Filename {
		t.Fatalf("expected filename %q, got %q", ref.Filename, got)
	}
	if got := mapValueString(first, "comment_id"); got != commentID {
		t.Fatalf("expected comment_id %q, got %q", commentID, got)
	}

	combined := h.Run("card", "attachments", "show", cardStr, "--include-comments")
	assertOK(t, combined)
	if len(combined.GetDataArray()) == 0 {
		t.Fatal("expected card attachment view with --include-comments to include comment attachment")
	}

	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "downloaded-comment-attachment.png")
	download := h.Run("comment", "attachments", "download", "--card", cardStr, "1", "-o", outputPath)
	assertOK(t, download)
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("expected downloaded file at %s: %v", outputPath, err)
	}
}
