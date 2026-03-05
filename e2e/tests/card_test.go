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

// createTestBoard creates a board for card tests and adds it to cleanup
func createTestBoard(t *testing.T, h *harness.Harness) string {
	t.Helper()
	name := fmt.Sprintf("Card Test Board %d", time.Now().UnixNano())
	result := h.Run("board", "create", "--name", name)
	if result.ExitCode != harness.ExitSuccess {
		t.Fatalf("failed to create test board: %s\nstdout: %s", result.Stderr, result.Stdout)
	}
	// Create returns location, not data - extract ID from location
	boardID := result.GetIDFromLocation()
	if boardID == "" {
		// Try data.id as fallback
		boardID = result.GetDataString("id")
	}
	if boardID == "" {
		t.Fatalf("no board ID returned (location: %s)", result.GetLocation())
	}
	h.Cleanup.AddBoard(boardID)
	return boardID
}

func TestCardList(t *testing.T) {
	h := harness.New(t)

	t.Run("returns list of cards", func(t *testing.T) {
		result := h.Run("card", "list")

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if result.Response == nil {
			t.Fatalf("expected JSON response, got nil\nstdout: %s", result.Stdout)
		}

		if !result.Response.OK {
			t.Error("expected ok=true")
		}

		// Data should be an array
		arr := result.GetDataArray()
		if arr == nil {
			t.Error("expected data to be an array")
		}
	})

	t.Run("supports --page option", func(t *testing.T) {
		result := h.Run("card", "list", "--page", "1")

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d", harness.ExitSuccess, result.ExitCode)
		}

		if result.Response == nil {
			t.Fatal("expected JSON response")
		}

		if !result.Response.OK {
			t.Error("expected ok=true")
		}
	})

	t.Run("supports --all flag", func(t *testing.T) {
		result := h.Run("card", "list", "--all")

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d", harness.ExitSuccess, result.ExitCode)
		}
	})
}

func TestCardListWithFilters(t *testing.T) {
	h := harness.New(t)
	defer h.Cleanup.CleanupAll(h)

	boardID := createTestBoard(t, h)

	t.Run("filters by board", func(t *testing.T) {
		// Create two cards on different boards to ensure the filter is effective.
		otherBoardID := createTestBoard(t, h)

		titleA := fmt.Sprintf("Board Filter A %d", time.Now().UnixNano())
		titleB := fmt.Sprintf("Board Filter B %d", time.Now().UnixNano())

		createdA := h.Run("card", "create", "--board", boardID, "--title", titleA)
		if createdA.ExitCode != harness.ExitSuccess {
			t.Fatalf("failed to create card on board A: %s\nstdout: %s", createdA.Stderr, createdA.Stdout)
		}
		cardA := createdA.GetNumberFromLocation()
		if cardA == 0 {
			cardA = createdA.GetDataInt("number")
		}
		if cardA != 0 {
			h.Cleanup.AddCard(cardA)
		}

		createdB := h.Run("card", "create", "--board", otherBoardID, "--title", titleB)
		if createdB.ExitCode != harness.ExitSuccess {
			t.Fatalf("failed to create card on board B: %s\nstdout: %s", createdB.Stderr, createdB.Stdout)
		}
		cardB := createdB.GetNumberFromLocation()
		if cardB == 0 {
			cardB = createdB.GetDataInt("number")
		}
		if cardB != 0 {
			h.Cleanup.AddCard(cardB)
		}

		result := h.Run("card", "list", "--board", boardID)

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d", harness.ExitSuccess, result.ExitCode)
		}

		if !result.Response.OK {
			t.Error("expected ok=true")
		}

		arr := result.GetDataArray()
		if arr == nil {
			t.Fatalf("expected array response\nstdout: %s", result.Stdout)
		}

		foundA := false
		foundB := false
		for _, item := range arr {
			card, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if title, ok := card["title"].(string); ok {
				if title == titleA {
					foundA = true
				}
				if title == titleB {
					foundB = true
				}
			}
		}

		if !foundA {
			t.Errorf("expected board-filtered list to include card created on board %s (title %q)", boardID, titleA)
		}
		if foundB {
			t.Errorf("expected board-filtered list to exclude card created on other board %s (title %q)", otherBoardID, titleB)
		}
	})

	t.Run("filters by status", func(t *testing.T) {
		result := h.Run("card", "list", "--indexed-by", "not_now")

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d", harness.ExitSuccess, result.ExitCode)
		}
	})
}

func TestCardCRUD(t *testing.T) {
	h := harness.New(t)
	defer h.Cleanup.CleanupAll(h)

	boardID := createTestBoard(t, h)
	var cardNumber int
	cardTitle := fmt.Sprintf("Test Card %d", time.Now().UnixNano())

	t.Run("create card with title", func(t *testing.T) {
		result := h.Run("card", "create", "--board", boardID, "--title", cardTitle)

		if result.ExitCode != harness.ExitSuccess {
			t.Fatalf("expected exit code %d, got %d\nstderr: %s\nstdout: %s",
				harness.ExitSuccess, result.ExitCode, result.Stderr, result.Stdout)
		}

		if result.Response == nil {
			t.Fatalf("expected JSON response, got nil\nstdout: %s", result.Stdout)
		}

		if !result.Response.OK {
			t.Errorf("expected ok=true, error: %+v", result.Response.Error)
		}

		// Create returns location - extract number from it
		cardNumber = result.GetNumberFromLocation()
		if cardNumber == 0 {
			// Try data.number as fallback
			cardNumber = result.GetDataInt("number")
		}
		if cardNumber == 0 {
			t.Fatalf("expected card number in response (location: %s)", result.GetLocation())
		}

		h.Cleanup.AddCard(cardNumber)

		// Note: Create returns location, not data with title
		// Verify the card exists via show command
		showResult := h.Run("card", "show", strconv.Itoa(cardNumber))
		if showResult.ExitCode == harness.ExitSuccess {
			title := showResult.GetDataString("title")
			if title != cardTitle {
				t.Errorf("expected title %q, got %q", cardTitle, title)
			}
		}
	})

	t.Run("show card by number", func(t *testing.T) {
		if cardNumber == 0 {
			t.Skip("no card number from create test")
		}

		result := h.Run("card", "show", strconv.Itoa(cardNumber))

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if result.Response == nil {
			t.Fatal("expected JSON response")
		}

		if !result.Response.OK {
			t.Error("expected ok=true")
		}

		num := result.GetDataInt("number")
		if num != cardNumber {
			t.Errorf("expected number %d, got %d", cardNumber, num)
		}
	})

	t.Run("update card title", func(t *testing.T) {
		if cardNumber == 0 {
			t.Skip("no card number from create test")
		}

		newTitle := cardTitle + " Updated"
		result := h.Run("card", "update", strconv.Itoa(cardNumber), "--title", newTitle)

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if result.Response == nil {
			t.Fatal("expected JSON response")
		}

		if !result.Response.OK {
			t.Error("expected ok=true")
		}

		title := result.GetDataString("title")
		if title != newTitle {
			t.Errorf("expected title %q, got %q", newTitle, title)
		}
	})

	t.Run("delete card", func(t *testing.T) {
		if cardNumber == 0 {
			t.Skip("no card number from create test")
		}

		result := h.Run("card", "delete", strconv.Itoa(cardNumber))

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if result.Response == nil {
			t.Fatal("expected JSON response")
		}

		if !result.Response.OK {
			t.Error("expected ok=true")
		}

		deleted := result.GetDataBool("deleted")
		if !deleted {
			t.Error("expected deleted=true")
		}

		// Remove from cleanup since we deleted it
		h.Cleanup.Cards = nil
	})
}

func TestCardCreateWithDescription(t *testing.T) {
	h := harness.New(t)
	defer h.Cleanup.CleanupAll(h)

	boardID := createTestBoard(t, h)

	t.Run("create card with description", func(t *testing.T) {
		title := fmt.Sprintf("Card with Description %d", time.Now().UnixNano())
		description := "<p>This is a <strong>test</strong> description.</p>"
		// API returns plain text version of description (HTML tags stripped)
		expectedDescPlainText := "This is a test description."

		result := h.Run("card", "create", "--board", boardID, "--title", title, "--description", description)

		if result.ExitCode != harness.ExitSuccess {
			t.Fatalf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		// Create returns location - extract number from it
		cardNumber := result.GetNumberFromLocation()
		if cardNumber == 0 {
			cardNumber = result.GetDataInt("number")
		}
		if cardNumber != 0 {
			h.Cleanup.AddCard(cardNumber)
		}

		if !result.Response.OK {
			t.Error("expected ok=true")
		}

		// Verify the description was actually saved by fetching the card
		if cardNumber != 0 {
			showResult := h.Run("card", "show", strconv.Itoa(cardNumber))
			if showResult.ExitCode != harness.ExitSuccess {
				t.Fatalf("failed to show card: %s", showResult.Stderr)
			}
			savedTitle := showResult.GetDataString("title")
			if savedTitle != title {
				t.Errorf("expected title %q, got %q", title, savedTitle)
			}
			// API returns plain text version
			savedDesc := showResult.GetDataString("description")
			if savedDesc != expectedDescPlainText {
				t.Errorf("expected description %q, got %q", expectedDescPlainText, savedDesc)
			}
			// Verify HTML version is also returned and contains our markup
			savedDescHtml := showResult.GetDataString("description_html")
			if savedDescHtml == "" {
				t.Error("expected description_html to be present")
			}
			// The HTML should contain our strong tag (Rails preserves it through Action Text)
			if !strings.Contains(savedDescHtml, "<strong>") && !strings.Contains(savedDescHtml, "test") {
				t.Errorf("expected description_html to contain markup, got %q", savedDescHtml)
			}
		}
	})

	t.Run("create card with description_file", func(t *testing.T) {
		// Get the path to the test document fixture
		wd, _ := os.Getwd()
		fixturePath := filepath.Join(wd, "..", "testdata", "fixtures", "test_document.txt")

		// Check if fixture exists
		if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
			t.Skipf("test fixture not found at %s", fixturePath)
		}

		title := fmt.Sprintf("Card from File %d", time.Now().UnixNano())
		result := h.Run("card", "create", "--board", boardID, "--title", title, "--description_file", fixturePath)

		if result.ExitCode != harness.ExitSuccess {
			t.Fatalf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		// Create returns location - extract number from it
		cardNumber := result.GetNumberFromLocation()
		if cardNumber == 0 {
			cardNumber = result.GetDataInt("number")
		}
		if cardNumber != 0 {
			h.Cleanup.AddCard(cardNumber)
		}

		if !result.Response.OK {
			t.Error("expected ok=true")
		}
	})
}

func TestCardActions(t *testing.T) {
	h := harness.New(t)
	defer h.Cleanup.CleanupAll(h)

	boardID := createTestBoard(t, h)

	// Create a card for action tests
	title := fmt.Sprintf("Action Test Card %d", time.Now().UnixNano())
	result := h.Run("card", "create", "--board", boardID, "--title", title)
	if result.ExitCode != harness.ExitSuccess {
		t.Fatalf("failed to create test card: %s\nstdout: %s", result.Stderr, result.Stdout)
	}
	// Create returns location - extract number from it
	cardNumber := result.GetNumberFromLocation()
	if cardNumber == 0 {
		cardNumber = result.GetDataInt("number")
	}
	if cardNumber == 0 {
		t.Fatalf("failed to get card number from create (location: %s)", result.GetLocation())
	}
	h.Cleanup.AddCard(cardNumber)
	cardStr := strconv.Itoa(cardNumber)

	t.Run("close card", func(t *testing.T) {
		result := h.Run("card", "close", cardStr)

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if !result.Response.OK {
			t.Errorf("expected ok=true, error: %+v", result.Response.Error)
		}
	})

	t.Run("reopen card", func(t *testing.T) {
		result := h.Run("card", "reopen", cardStr)

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if !result.Response.OK {
			t.Errorf("expected ok=true, error: %+v", result.Response.Error)
		}
	})

	t.Run("postpone card", func(t *testing.T) {
		result := h.Run("card", "postpone", cardStr)

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if !result.Response.OK {
			t.Errorf("expected ok=true, error: %+v", result.Response.Error)
		}
	})

	t.Run("watch card", func(t *testing.T) {
		result := h.Run("card", "watch", cardStr)

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if !result.Response.OK {
			t.Errorf("expected ok=true, error: %+v", result.Response.Error)
		}
	})

	t.Run("unwatch card", func(t *testing.T) {
		result := h.Run("card", "unwatch", cardStr)

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if !result.Response.OK {
			t.Errorf("expected ok=true, error: %+v", result.Response.Error)
		}
	})

	t.Run("tag card", func(t *testing.T) {
		tagName := fmt.Sprintf("test-tag-%d", time.Now().UnixNano())
		result := h.Run("card", "tag", cardStr, "--tag", tagName)

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if !result.Response.OK {
			t.Errorf("expected ok=true, error: %+v", result.Response.Error)
		}
	})

	t.Run("self-assign card", func(t *testing.T) {
		result := h.Run("card", "self-assign", cardStr)

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if !result.Response.OK {
			t.Errorf("expected ok=true, error: %+v", result.Response.Error)
		}
	})

	t.Run("self-assign card again to unassign", func(t *testing.T) {
		result := h.Run("card", "self-assign", cardStr)

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if !result.Response.OK {
			t.Errorf("expected ok=true, error: %+v", result.Response.Error)
		}
	})

	t.Run("golden card", func(t *testing.T) {
		result := h.Run("card", "golden", cardStr)

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if !result.Response.OK {
			t.Errorf("expected ok=true, error: %+v", result.Response.Error)
		}

		// Verify card is now golden
		showResult := h.Run("card", "show", cardStr)
		if showResult.ExitCode == harness.ExitSuccess {
			golden := showResult.GetDataBool("golden")
			if !golden {
				t.Error("expected card to be golden after marking as golden")
			}
		}
	})

	t.Run("ungolden card", func(t *testing.T) {
		result := h.Run("card", "ungolden", cardStr)

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if !result.Response.OK {
			t.Errorf("expected ok=true, error: %+v", result.Response.Error)
		}

		// Verify card is no longer golden
		showResult := h.Run("card", "show", cardStr)
		if showResult.ExitCode == harness.ExitSuccess {
			golden := showResult.GetDataBool("golden")
			if golden {
				t.Error("expected card to not be golden after removing golden status")
			}
		}
	})
}

func TestCardColumn(t *testing.T) {
	h := harness.New(t)
	defer h.Cleanup.CleanupAll(h)

	boardID := createTestBoard(t, h)

	// Create a column
	columnName := fmt.Sprintf("Test Column %d", time.Now().UnixNano())
	colResult := h.Run("column", "create", "--board", boardID, "--name", columnName)
	if colResult.ExitCode != harness.ExitSuccess {
		t.Fatalf("failed to create column: %s\nstdout: %s", colResult.Stderr, colResult.Stdout)
	}
	// Create returns location - extract ID from it
	columnID := colResult.GetIDFromLocation()
	if columnID == "" {
		columnID = colResult.GetDataString("id")
	}
	if columnID == "" {
		t.Fatalf("failed to get column ID (location: %s)", colResult.GetLocation())
	}
	h.Cleanup.AddColumn(columnID, boardID)

	// Create a card
	title := fmt.Sprintf("Column Test Card %d", time.Now().UnixNano())
	cardResult := h.Run("card", "create", "--board", boardID, "--title", title)
	if cardResult.ExitCode != harness.ExitSuccess {
		t.Fatalf("failed to create card: %s\nstdout: %s", cardResult.Stderr, cardResult.Stdout)
	}
	// Create returns location - extract number from it
	cardNumber := cardResult.GetNumberFromLocation()
	if cardNumber == 0 {
		cardNumber = cardResult.GetDataInt("number")
	}
	if cardNumber == 0 {
		t.Fatalf("failed to get card number (location: %s)", cardResult.GetLocation())
	}
	h.Cleanup.AddCard(cardNumber)
	cardStr := strconv.Itoa(cardNumber)

	t.Run("move card to column", func(t *testing.T) {
		result := h.Run("card", "column", cardStr, "--column", columnID)

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if !result.Response.OK {
			t.Errorf("expected ok=true, error: %+v", result.Response.Error)
		}
	})

	t.Run("untriage card (send back to triage)", func(t *testing.T) {
		result := h.Run("card", "untriage", cardStr)

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if !result.Response.OK {
			t.Errorf("expected ok=true, error: %+v", result.Response.Error)
		}
	})
}

func TestCardShowNotFound(t *testing.T) {
	h := harness.New(t)

	t.Run("returns not found for non-existent card", func(t *testing.T) {
		result := h.Run("card", "show", "999999999")

		if result.ExitCode != harness.ExitNotFound {
			t.Errorf("expected exit code %d, got %d\nstdout: %s",
				harness.ExitNotFound, result.ExitCode, result.Stdout)
		}

		if result.Response == nil {
			t.Fatal("expected JSON response")
		}

		if result.Response.OK {
			t.Error("expected ok=false")
		}

		if result.Response.Error == "" {
			t.Error("expected error in response")
		}
	})
}

func TestCardCreateMissingBoard(t *testing.T) {
	h := harness.New(t)

	t.Run("fails without required --board option when no default board configured", func(t *testing.T) {
		// Use a temp HOME so the global config (which may have a default board) is not found
		tmpHome := t.TempDir()
		result := h.RunWithEnv(map[string]string{"HOME": tmpHome}, "card", "create", "--title", "Test")

		// Should fail with error exit code
		if result.ExitCode == harness.ExitSuccess {
			t.Error("expected non-zero exit code for missing required option")
		}
	})
}

func TestCardCreateMissingTitle(t *testing.T) {
	h := harness.New(t)
	defer h.Cleanup.CleanupAll(h)

	boardID := createTestBoard(t, h)

	t.Run("fails without required --title option", func(t *testing.T) {
		result := h.Run("card", "create", "--board", boardID)

		// Should fail with error exit code
		if result.ExitCode == harness.ExitSuccess {
			t.Error("expected non-zero exit code for missing required option")
		}
	})
}

func TestCardHasAttachments(t *testing.T) {
	h := harness.New(t)
	defer h.Cleanup.CleanupAll(h)

	boardID := createTestBoard(t, h)

	t.Run("plain card has_attachments is false", func(t *testing.T) {
		title := fmt.Sprintf("No Attachments Card %d", time.Now().UnixNano())
		cardResult := h.Run("card", "create", "--board", boardID, "--title", title)
		if cardResult.ExitCode != harness.ExitSuccess {
			t.Fatalf("failed to create card: %s\nstdout: %s", cardResult.Stderr, cardResult.Stdout)
		}

		cardNumber := cardResult.GetNumberFromLocation()
		if cardNumber == 0 {
			cardNumber = cardResult.GetDataInt("number")
		}
		if cardNumber == 0 {
			t.Fatalf("failed to get card number (location: %s)", cardResult.GetLocation())
		}
		h.Cleanup.AddCard(cardNumber)
		cardStr := strconv.Itoa(cardNumber)

		showResult := h.Run("card", "show", cardStr)
		if showResult.ExitCode != harness.ExitSuccess {
			t.Fatalf("failed to show card: %s", showResult.Stderr)
		}

		hasAttachments := showResult.GetDataBool("has_attachments")
		if hasAttachments {
			t.Error("expected has_attachments=false for a card with no attachments")
		}
	})

	t.Run("card with inline attachment has_attachments is true", func(t *testing.T) {
		// Get the path to the test image fixture
		wd, _ := os.Getwd()
		fixturePath := filepath.Join(wd, "..", "testdata", "fixtures", "test_image.png")

		// Check if fixture exists
		if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
			t.Skipf("test fixture not found at %s", fixturePath)
		}

		// Upload the file to get an attachable_sgid
		uploadResult := h.Run("upload", "file", fixturePath)
		if uploadResult.ExitCode != harness.ExitSuccess {
			t.Fatalf("failed to upload file: %s\nstdout: %s", uploadResult.Stderr, uploadResult.Stdout)
		}

		sgid := uploadResult.GetDataString("attachable_sgid")
		if sgid == "" {
			t.Fatalf("no attachable_sgid returned from upload\nstdout: %s", uploadResult.Stdout)
		}

		// Create a card with an inline attachment in the description
		title := fmt.Sprintf("Attachment Card %d", time.Now().UnixNano())
		description := fmt.Sprintf(`<p>See image:</p><action-text-attachment sgid="%s"></action-text-attachment>`, sgid)
		cardResult := h.Run("card", "create", "--board", boardID, "--title", title, "--description", description)
		if cardResult.ExitCode != harness.ExitSuccess {
			t.Fatalf("failed to create card with attachment: %s\nstdout: %s", cardResult.Stderr, cardResult.Stdout)
		}

		cardNumber := cardResult.GetNumberFromLocation()
		if cardNumber == 0 {
			cardNumber = cardResult.GetDataInt("number")
		}
		if cardNumber == 0 {
			t.Fatalf("failed to get card number (location: %s)", cardResult.GetLocation())
		}
		h.Cleanup.AddCard(cardNumber)
		cardStr := strconv.Itoa(cardNumber)

		showResult := h.Run("card", "show", cardStr)
		if showResult.ExitCode != harness.ExitSuccess {
			t.Fatalf("failed to show card: %s", showResult.Stderr)
		}

		hasAttachments := showResult.GetDataBool("has_attachments")
		if !hasAttachments {
			t.Error("expected has_attachments=true for a card with an inline attachment")
		}
	})
}

func TestCardImageRemove(t *testing.T) {
	h := harness.New(t)
	defer h.Cleanup.CleanupAll(h)

	boardID := createTestBoard(t, h)

	t.Run("removes header image from card", func(t *testing.T) {
		// Get the path to the test image fixture
		wd, _ := os.Getwd()
		fixturePath := filepath.Join(wd, "..", "testdata", "fixtures", "test_image.png")

		// Check if fixture exists
		if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
			t.Skipf("test fixture not found at %s", fixturePath)
		}

		// Upload the file first to get a signed ID for the header image
		uploadResult := h.Run("upload", "file", fixturePath)
		if uploadResult.ExitCode != harness.ExitSuccess {
			t.Fatalf("failed to upload file: %s\nstdout: %s", uploadResult.Stderr, uploadResult.Stdout)
		}

		signedID := uploadResult.GetDataString("signed_id")
		if signedID == "" {
			t.Fatalf("no signed_id returned from upload\nstdout: %s", uploadResult.Stdout)
		}

		// Create a card with header image
		title := fmt.Sprintf("Image Remove Test Card %d", time.Now().UnixNano())
		cardResult := h.Run("card", "create", "--board", boardID, "--title", title, "--image", signedID)
		if cardResult.ExitCode != harness.ExitSuccess {
			t.Fatalf("failed to create card with image: %s\nstdout: %s", cardResult.Stderr, cardResult.Stdout)
		}

		cardNumber := cardResult.GetNumberFromLocation()
		if cardNumber == 0 {
			cardNumber = cardResult.GetDataInt("number")
		}
		if cardNumber == 0 {
			t.Fatalf("failed to get card number from create (location: %s)", cardResult.GetLocation())
		}
		h.Cleanup.AddCard(cardNumber)
		cardStr := strconv.Itoa(cardNumber)

		// Verify card has image initially
		showResult := h.Run("card", "show", cardStr)
		if showResult.ExitCode == harness.ExitSuccess {
			imageURL := showResult.GetDataString("image_url")
			if imageURL == "" {
				t.Log("Warning: card may not have image_url field set immediately after creation")
			}
		}

		// Remove the image
		result := h.Run("card", "image-remove", cardStr)

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if !result.Response.OK {
			t.Errorf("expected ok=true, error: %+v", result.Response.Error)
		}
	})

	t.Run("succeeds on card without image", func(t *testing.T) {
		// Create a card without image
		title := fmt.Sprintf("No Image Card %d", time.Now().UnixNano())
		cardResult := h.Run("card", "create", "--board", boardID, "--title", title)
		if cardResult.ExitCode != harness.ExitSuccess {
			t.Fatalf("failed to create card: %s", cardResult.Stderr)
		}

		cardNumber := cardResult.GetNumberFromLocation()
		if cardNumber == 0 {
			cardNumber = cardResult.GetDataInt("number")
		}
		h.Cleanup.AddCard(cardNumber)
		cardStr := strconv.Itoa(cardNumber)

		// Try to remove non-existent image (should still succeed or return appropriate error)
		result := h.Run("card", "image-remove", cardStr)

		// The API may return success or not found - either is acceptable
		if result.ExitCode != harness.ExitSuccess && result.ExitCode != harness.ExitNotFound {
			t.Errorf("expected exit code %d or %d, got %d\nstderr: %s",
				harness.ExitSuccess, harness.ExitNotFound, result.ExitCode, result.Stderr)
		}
	})

	t.Run("returns not found for non-existent card", func(t *testing.T) {
		result := h.Run("card", "image-remove", "999999999")

		if result.ExitCode != harness.ExitNotFound {
			t.Errorf("expected exit code %d, got %d\nstdout: %s",
				harness.ExitNotFound, result.ExitCode, result.Stdout)
		}

		if result.Response != nil && result.Response.OK {
			t.Error("expected ok=false")
		}
	})
}
