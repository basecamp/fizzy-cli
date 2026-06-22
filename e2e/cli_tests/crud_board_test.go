package clitests

import (
	"fmt"
	"testing"
	"time"

	"github.com/basecamp/fizzy-cli/e2e/harness"
)

func TestBoardList(t *testing.T) {
	h := newHarness(t)
	result := h.Run("board", "list")
	assertOK(t, result)
	if result.GetDataArray() == nil {
		t.Fatal("expected array response")
	}
}

func TestBoardListAll(t *testing.T) {
	assertOK(t, newHarness(t).Run("board", "list", "--all"))
}

func TestBoardListPaginated(t *testing.T) {
	assertOK(t, newHarness(t).Run("board", "list", "--page", "1"))
}

func TestBoardShow(t *testing.T) {
	result := newHarness(t).Run("board", "show", fixture.BoardID)
	assertOK(t, result)
	if got := result.GetDataString("id"); got != fixture.BoardID {
		t.Fatalf("expected board id %q, got %q", fixture.BoardID, got)
	}
}

func TestBoardShowNotFound(t *testing.T) {
	assertResult(t, newHarness(t).Run("board", "show", "nonexistent-board-id-99999"), harness.ExitNotFound)
}

func TestBoardCreateUpdateDelete(t *testing.T) {
	h := newHarness(t)
	boardID := createBoard(t, h)
	updatedName := fmt.Sprintf("Updated Board %d", time.Now().UnixNano())
	result := h.Run("board", "update", boardID, "--name", updatedName)
	assertOK(t, result)

	show := h.Run("board", "show", boardID)
	assertOK(t, show)
	if got := show.GetDataString("name"); got != updatedName {
		t.Fatalf("expected updated board name %q, got %q", updatedName, got)
	}

	deleteResult := h.Run("board", "delete", boardID)
	assertOK(t, deleteResult)
	if !deleteResult.GetDataBool("deleted") {
		t.Fatal("expected deleted=true")
	}
	assertResult(t, h.Run("board", "show", boardID), harness.ExitNotFound)
}

func TestBoardPublishUnpublish(t *testing.T) {
	h := newHarness(t)
	boardID := createBoard(t, h)
	publish := h.Run("board", "publish", boardID)
	assertOK(t, publish)
	publicURL := publish.GetDataString("public_url")
	if publicURL == "" {
		t.Fatal("expected public_url in publish response")
	}

	showPublished := h.Run("board", "show", boardID)
	assertOK(t, showPublished)
	if got := showPublished.GetDataString("public_url"); got != publicURL {
		t.Fatalf("expected published board public_url %q, got %q", publicURL, got)
	}

	assertOK(t, h.Run("board", "unpublish", boardID))
	showUnpublished := h.Run("board", "show", boardID)
	assertOK(t, showUnpublished)
	if got := showUnpublished.GetDataString("public_url"); got != "" {
		t.Fatalf("expected public_url to be cleared after unpublish, got %q", got)
	}
}

func TestBoardEntropy(t *testing.T) {
	h := newHarness(t)
	boardID := createBoard(t, h)
	show := h.Run("board", "show", boardID)
	assertOK(t, show)
	currentDays := show.GetDataInt("auto_postpone_period_in_days")
	if currentDays == 0 {
		currentDays = 30
	}
	assertOK(t, h.Run("board", "entropy", boardID, "--auto_postpone_period_in_days", fmt.Sprintf("%d", currentDays)))
	show = h.Run("board", "show", boardID)
	assertOK(t, show)
	if got := show.GetDataInt("auto_postpone_period_in_days"); got != currentDays {
		t.Fatalf("expected auto_postpone_period_in_days=%d, got %d", currentDays, got)
	}
}

func TestBoardViews(t *testing.T) {
	h := newHarness(t)
	boardID := createBoard(t, h)

	streamCard := createCard(t, h, boardID)
	closedCard := createCard(t, h, boardID)
	postponedCard := createCard(t, h, boardID)
	assertOK(t, h.Run("card", "close", fmt.Sprintf("%d", closedCard)))
	assertOK(t, h.Run("card", "postpone", fmt.Sprintf("%d", postponedCard)))

	stream := h.Run("board", "stream", "--board", boardID)
	assertOK(t, stream)
	if listMapByNumber(stream.GetDataArray(), streamCard) == nil {
		t.Fatalf("expected stream view to include card #%d", streamCard)
	}
	if listMapByNumber(stream.GetDataArray(), closedCard) != nil {
		t.Fatalf("expected stream view to exclude closed card #%d", closedCard)
	}
	if listMapByNumber(stream.GetDataArray(), postponedCard) != nil {
		t.Fatalf("expected stream view to exclude postponed card #%d", postponedCard)
	}

	closed := h.Run("board", "closed", "--board", boardID)
	assertOK(t, closed)
	if listMapByNumber(closed.GetDataArray(), closedCard) == nil {
		t.Fatalf("expected closed view to include card #%d", closedCard)
	}

	postponed := h.Run("board", "postponed", "--board", boardID)
	assertOK(t, postponed)
	if listMapByNumber(postponed.GetDataArray(), postponedCard) == nil {
		t.Fatalf("expected postponed view to include card #%d", postponedCard)
	}
}

func TestBoardAccesses(t *testing.T) {
	h := newHarness(t)
	boardID := createBoard(t, h)

	result := h.Run("board", "accesses", "--board", boardID)
	assertOK(t, result)
	if got := result.GetDataString("board_id"); got != boardID {
		t.Fatalf("expected board_id %q, got %q", boardID, got)
	}
	if _, ok := result.GetDataMap()["all_access"]; !ok {
		t.Fatal("expected all_access in board accesses response")
	}
	if _, ok := result.GetDataMap()["users"]; !ok {
		t.Fatal("expected users in board accesses response")
	}
}

func TestBoardInvolvement(t *testing.T) {
	h := newHarness(t)
	boardID := createBoard(t, h)
	// There is currently no CLI command that reads back board involvement, so this
	// remains a command-contract check until the CLI exposes that state.
	assertOK(t, h.Run("board", "involvement", boardID, "--involvement", "watching"))
	assertOK(t, h.Run("board", "involvement", boardID, "--involvement", "access_only"))
}
