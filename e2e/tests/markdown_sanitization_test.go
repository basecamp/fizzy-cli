package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/basecamp/fizzy-cli/e2e/harness"
)

func TestMarkdownSanitization(t *testing.T) {
	h := harness.New(t)
	defer h.Cleanup.CleanupAll(h)

	boardID := createAttachmentTestBoard(t, h)

	t.Run("card description with backtick-wrapped action-text-attachment is escaped", func(t *testing.T) {
		title := fmt.Sprintf("Markdown Sanitization Card %d", time.Now().UnixNano())
		// Simulate an LLM writing documentation about attachments using markdown backticks
		description := `<br>2. Manually construct the ` + "`" + `<action-text-attachment sgid="..." content-type="application/vnd.actiontext.mention"></action-text-attachment>` + "`"

		result := h.Run("card", "create", "--board", boardID, "--title", title, "--description", description)
		if result.ExitCode != harness.ExitSuccess {
			t.Fatalf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		cardNumber := result.GetNumberFromLocation()
		if cardNumber == 0 {
			cardNumber = result.GetDataInt("number")
		}
		if cardNumber != 0 {
			h.Cleanup.AddCard(cardNumber)
		}

		// Fetch the card and verify the attachment tag was escaped, not parsed as a real attachment
		showResult := h.Run("card", "show", strconv.Itoa(cardNumber))
		if showResult.ExitCode != harness.ExitSuccess {
			t.Fatalf("failed to show card: %s", showResult.Stderr)
		}

		descHTML := showResult.GetDataString("description_html")

		// The action-text-attachment should be inside a <code> tag, escaped as &lt;action-text-attachment&gt;
		if strings.Contains(descHTML, `<action-text-attachment`) {
			t.Errorf("expected action-text-attachment to be escaped in description_html, but found raw tag:\n%s", descHTML)
		}
		if !strings.Contains(descHTML, `<code>`) {
			t.Errorf("expected backtick content to be converted to <code> tag:\n%s", descHTML)
		}
	})

	t.Run("comment body with backtick-wrapped action-text-attachment is escaped", func(t *testing.T) {
		// Create a plain card first
		title := fmt.Sprintf("Markdown Sanitization Comment Card %d", time.Now().UnixNano())
		cardResult := h.Run("card", "create", "--board", boardID, "--title", title)
		if cardResult.ExitCode != harness.ExitSuccess {
			t.Fatalf("failed to create card: %s\nstdout: %s", cardResult.Stderr, cardResult.Stdout)
		}
		cardNumber := cardResult.GetNumberFromLocation()
		if cardNumber == 0 {
			cardNumber = cardResult.GetDataInt("number")
		}
		if cardNumber != 0 {
			h.Cleanup.AddCard(cardNumber)
		}

		// Create a comment with backtick-wrapped attachment tag
		body := "Here is an example: `<action-text-attachment sgid=\"...\" content-type=\"application/vnd.actiontext.mention\"></action-text-attachment>`"

		commentResult := h.Run("comment", "create", "--card", strconv.Itoa(cardNumber), "--body", body)
		if commentResult.ExitCode != harness.ExitSuccess {
			t.Fatalf("failed to create comment: %s\nstdout: %s", commentResult.Stderr, commentResult.Stdout)
		}

		commentID := commentResult.GetIDFromLocation()
		if commentID == "" {
			commentID = commentResult.GetDataString("id")
		}
		if commentID != "" {
			h.Cleanup.AddComment(commentID, cardNumber)
		}

		// Fetch comments and verify the attachment tag was escaped
		listResult := h.Run("comment", "list", "--card", strconv.Itoa(cardNumber))
		if listResult.ExitCode != harness.ExitSuccess {
			t.Fatalf("failed to list comments: %s", listResult.Stderr)
		}

		comments := listResult.GetDataArray()
		if len(comments) == 0 {
			t.Fatal("expected at least one comment")
		}

		comment := comments[0].(map[string]any)
		bodyObj := comment["body"].(map[string]any)
		bodyHTML := bodyObj["html"].(string)

		if strings.Contains(bodyHTML, `<action-text-attachment`) {
			t.Errorf("expected action-text-attachment to be escaped in comment body html, but found raw tag:\n%s", bodyHTML)
		}
		if !strings.Contains(bodyHTML, `<code>`) {
			t.Errorf("expected backtick content to be converted to <code> tag:\n%s", bodyHTML)
		}
	})

	t.Run("real card attachment still works after markdown conversion", func(t *testing.T) {
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
			t.Fatalf("no attachable_sgid returned from upload")
		}

		// Create card with real attachment - raw HTML, no markdown
		title := fmt.Sprintf("Real Attachment Card %d", time.Now().UnixNano())
		description := fmt.Sprintf(`<action-text-attachment sgid="%s"></action-text-attachment>`, attachableSGID)

		cardResult := h.Run("card", "create", "--board", boardID, "--title", title, "--description", description)
		if cardResult.ExitCode != harness.ExitSuccess {
			t.Fatalf("failed to create card: %s\nstdout: %s", cardResult.Stderr, cardResult.Stdout)
		}
		cardNumber := cardResult.GetNumberFromLocation()
		if cardNumber == 0 {
			cardNumber = cardResult.GetDataInt("number")
		}
		if cardNumber != 0 {
			h.Cleanup.AddCard(cardNumber)
		}

		// Verify the attachment is actually there
		attachResult := h.Run("card", "attachments", "show", strconv.Itoa(cardNumber))
		if attachResult.ExitCode != harness.ExitSuccess {
			t.Fatalf("failed to show attachments: %s", attachResult.Stderr)
		}

		attachments := attachResult.GetDataArray()
		if len(attachments) == 0 {
			t.Error("expected at least one attachment on card")
		}
	})

	t.Run("real attachment with backtick-wrapped attachment tag in same description", func(t *testing.T) {
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
			t.Fatalf("no attachable_sgid returned from upload")
		}

		// Create card with a real attachment AND backtick-wrapped example tag
		title := fmt.Sprintf("Mixed Real and Backtick Attachment %d", time.Now().UnixNano())
		description := fmt.Sprintf("<p>See the image:</p><action-text-attachment sgid=\"%s\"></action-text-attachment><p>To embed attachments use: `<action-text-attachment sgid=\"...\" content-type=\"...\"></action-text-attachment>`</p>", attachableSGID)

		cardResult := h.Run("card", "create", "--board", boardID, "--title", title, "--description", description)
		if cardResult.ExitCode != harness.ExitSuccess {
			t.Fatalf("failed to create card: %s\nstdout: %s", cardResult.Stderr, cardResult.Stdout)
		}
		cardNumber := cardResult.GetNumberFromLocation()
		if cardNumber == 0 {
			cardNumber = cardResult.GetDataInt("number")
		}
		if cardNumber != 0 {
			h.Cleanup.AddCard(cardNumber)
		}

		// Verify the real attachment is there
		attachResult := h.Run("card", "attachments", "show", strconv.Itoa(cardNumber))
		if attachResult.ExitCode != harness.ExitSuccess {
			t.Fatalf("failed to show attachments: %s", attachResult.Stderr)
		}

		attachments := attachResult.GetDataArray()
		if len(attachments) != 1 {
			t.Errorf("expected exactly 1 real attachment, got %d", len(attachments))
		}

		// Verify the backtick content was escaped (check description_html)
		showResult := h.Run("card", "show", strconv.Itoa(cardNumber))
		if showResult.ExitCode != harness.ExitSuccess {
			t.Fatalf("failed to show card: %s", showResult.Stderr)
		}
		descHTML := showResult.GetDataString("description_html")
		if !strings.Contains(descHTML, "<code>") {
			t.Errorf("expected backtick content to be converted to <code> tag:\n%s", descHTML)
		}
	})

	t.Run("real comment attachment still works after markdown conversion", func(t *testing.T) {
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
			t.Fatalf("no attachable_sgid returned from upload")
		}

		// Create a plain card
		title := fmt.Sprintf("Real Comment Attachment Card %d", time.Now().UnixNano())
		cardResult := h.Run("card", "create", "--board", boardID, "--title", title)
		if cardResult.ExitCode != harness.ExitSuccess {
			t.Fatalf("failed to create card: %s\nstdout: %s", cardResult.Stderr, cardResult.Stdout)
		}
		cardNumber := cardResult.GetNumberFromLocation()
		if cardNumber == 0 {
			cardNumber = cardResult.GetDataInt("number")
		}
		if cardNumber != 0 {
			h.Cleanup.AddCard(cardNumber)
		}

		// Create comment with real attachment - raw HTML, no markdown
		body := fmt.Sprintf(`<action-text-attachment sgid="%s"></action-text-attachment>`, attachableSGID)
		commentResult := h.Run("comment", "create", "--card", strconv.Itoa(cardNumber), "--body", body)
		if commentResult.ExitCode != harness.ExitSuccess {
			t.Fatalf("failed to create comment: %s\nstdout: %s", commentResult.Stderr, commentResult.Stdout)
		}
		commentID := commentResult.GetIDFromLocation()
		if commentID == "" {
			commentID = commentResult.GetDataString("id")
		}
		if commentID != "" {
			h.Cleanup.AddComment(commentID, cardNumber)
		}

		// Verify the attachment is actually there
		attachResult := h.Run("comment", "attachments", "show", "--card", strconv.Itoa(cardNumber))
		if attachResult.ExitCode != harness.ExitSuccess {
			t.Fatalf("failed to show comment attachments: %s", attachResult.Stderr)
		}

		attachments := attachResult.GetDataArray()
		if len(attachments) == 0 {
			t.Error("expected at least one comment attachment")
		}
	})
}
