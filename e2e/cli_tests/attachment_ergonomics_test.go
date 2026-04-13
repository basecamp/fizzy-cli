package clitests

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestCardAttachFlag(t *testing.T) {
	h := newHarness(t)
	boardID := createBoard(t, h)
	imagePath := fixtureFile(t, "test_image.png")
	docPath := fixtureFile(t, "test_document.txt")

	t.Run("creates card with single attach and downloads it", func(t *testing.T) {
		title := fmt.Sprintf("Attach Flag Card %d", time.Now().UnixNano())
		result := h.Run("card", "create", "--board", boardID, "--title", title, "--attach", imagePath)
		assertOK(t, result)

		cardNumber := result.GetNumberFromLocation()
		if cardNumber == 0 {
			cardNumber = result.GetDataInt("number")
		}
		if cardNumber == 0 {
			t.Fatalf("failed to get card number from create (location: %s)", result.GetLocation())
		}

		showResult := h.Run("card", "attachments", "show", strconv.Itoa(cardNumber))
		assertOK(t, showResult)
		arr := showResult.GetDataArray()
		if len(arr) != 1 {
			t.Fatalf("expected 1 attachment, got %d", len(arr))
		}
		attachment := asMap(arr[0])
		if got := mapValueString(attachment, "filename"); got != "test_image.png" {
			t.Fatalf("expected filename test_image.png, got %v", got)
		}

		outputPath := filepath.Join(t.TempDir(), "test_image.png")
		downloadResult := h.Run("card", "attachments", "download", strconv.Itoa(cardNumber), "1", "-o", outputPath)
		assertOK(t, downloadResult)
		assertFileExists(t, outputPath)
	})

	t.Run("creates card with multiple attaches in order", func(t *testing.T) {
		title := fmt.Sprintf("Attach Flag Multi Card %d", time.Now().UnixNano())
		result := h.Run(
			"card", "create",
			"--board", boardID,
			"--title", title,
			"--description", "See attached files",
			"--attach", imagePath,
			"--attach", docPath,
		)
		assertOK(t, result)

		cardNumber := result.GetNumberFromLocation()
		if cardNumber == 0 {
			cardNumber = result.GetDataInt("number")
		}

		showResult := h.Run("card", "attachments", "show", strconv.Itoa(cardNumber))
		assertOK(t, showResult)
		arr := showResult.GetDataArray()
		if len(arr) != 2 {
			t.Fatalf("expected 2 attachments, got %d", len(arr))
		}
		if got := mapValueString(asMap(arr[0]), "filename"); got != "test_image.png" {
			t.Fatalf("expected first attachment test_image.png, got %v", got)
		}
		if got := mapValueString(asMap(arr[1]), "filename"); got != "test_document.txt" {
			t.Fatalf("expected second attachment test_document.txt, got %v", got)
		}
	})

	t.Run("works with description_file", func(t *testing.T) {
		descriptionFile := filepath.Join(t.TempDir(), "description.md")
		mustWriteFile(t, descriptionFile, []byte("See file-based content"))

		title := fmt.Sprintf("Attach Flag File Card %d", time.Now().UnixNano())
		result := h.Run(
			"card", "create",
			"--board", boardID,
			"--title", title,
			"--description_file", descriptionFile,
			"--attach", imagePath,
		)
		assertOK(t, result)

		cardNumber := result.GetNumberFromLocation()
		if cardNumber == 0 {
			cardNumber = result.GetDataInt("number")
		}

		showResult := h.Run("card", "attachments", "show", strconv.Itoa(cardNumber))
		assertOK(t, showResult)
		if got := len(showResult.GetDataArray()); got != 1 {
			t.Fatalf("expected 1 attachment from description_file flow, got %d", got)
		}
	})
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file at %s: %v", path, err)
	}
}

func mustWriteFile(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

func TestCommentAttachFlag(t *testing.T) {
	h := newHarness(t)
	cardNumber := createCard(t, h, fixture.BoardID)
	cardStr := strconv.Itoa(cardNumber)
	imagePath := fixtureFile(t, "test_image.png")
	docPath := fixtureFile(t, "test_document.txt")

	t.Run("creates attachment-only comment with single attach", func(t *testing.T) {
		before := h.Run("comment", "attachments", "show", "--card", cardStr)
		assertOK(t, before)
		beforeCount := len(before.GetDataArray())

		result := h.Run("comment", "create", "--card", cardStr, "--attach", imagePath)
		assertOK(t, result)

		showResult := h.Run("comment", "attachments", "show", "--card", cardStr)
		assertOK(t, showResult)
		arr := showResult.GetDataArray()
		if len(arr) != beforeCount+1 {
			t.Fatalf("expected %d comment attachments, got %d", beforeCount+1, len(arr))
		}
		added := asMap(arr[len(arr)-1])
		if got := mapValueString(added, "filename"); got != "test_image.png" {
			t.Fatalf("expected filename test_image.png, got %v", got)
		}

		outputPath := filepath.Join(t.TempDir(), "test_image.png")
		downloadResult := h.Run("comment", "attachments", "download", "--card", cardStr, strconv.Itoa(len(arr)), "-o", outputPath)
		assertOK(t, downloadResult)
		assertFileExists(t, outputPath)
	})

	t.Run("creates comment with multiple attaches in order", func(t *testing.T) {
		before := h.Run("comment", "attachments", "show", "--card", cardStr)
		assertOK(t, before)
		beforeCount := len(before.GetDataArray())

		result := h.Run(
			"comment", "create",
			"--card", cardStr,
			"--body", "See attached files",
			"--attach", imagePath,
			"--attach", docPath,
		)
		assertOK(t, result)

		showResult := h.Run("comment", "attachments", "show", "--card", cardStr)
		assertOK(t, showResult)
		arr := showResult.GetDataArray()
		if len(arr) != beforeCount+2 {
			t.Fatalf("expected %d comment attachments, got %d", beforeCount+2, len(arr))
		}

		lastTwo := arr[len(arr)-2:]
		if got := mapValueString(asMap(lastTwo[0]), "filename"); got != "test_image.png" {
			t.Fatalf("expected first new attachment test_image.png, got %v", got)
		}
		if got := mapValueString(asMap(lastTwo[1]), "filename"); got != "test_document.txt" {
			t.Fatalf("expected second new attachment test_document.txt, got %v", got)
		}
	})
}
