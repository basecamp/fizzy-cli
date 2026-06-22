package clitests

import (
	"strconv"
	"testing"
	"time"

	"github.com/basecamp/fizzy-cli/e2e/harness"
)

func TestCardList(t *testing.T) {
	assertOK(t, newHarness(t).Run("card", "list"))
}

func TestCardListOnBoard(t *testing.T) {
	result := newHarness(t).Run("card", "list", "--board", fixture.BoardID)
	assertOK(t, result)
	if result.GetDataArray() == nil {
		t.Fatal("expected array response")
	}
}

func TestCardListAll(t *testing.T) {
	assertOK(t, newHarness(t).Run("card", "list", "--board", fixture.BoardID, "--all"))
}

func TestCardShow(t *testing.T) {
	assertOK(t, newHarness(t).Run("card", "show", strconv.Itoa(fixture.CardNumber)))
}

func TestCardShowNotFound(t *testing.T) {
	assertResult(t, newHarness(t).Run("card", "show", "999999999"), harness.ExitNotFound)
}

func TestCardLifecycle(t *testing.T) {
	h := newHarness(t)
	num := createCard(t, h, fixture.BoardID)
	numStr := strconv.Itoa(num)
	currentUser := currentUserID(t, h)
	updatedTitle := "Updated Card"
	tagTitle := "cli-test"

	assertOK(t, h.Run("card", "update", numStr, "--title", updatedTitle))
	show := h.Run("card", "show", numStr)
	assertOK(t, show)
	if got := show.GetDataString("title"); got != updatedTitle {
		t.Fatalf("expected updated title %q, got %q", updatedTitle, got)
	}

	assertOK(t, h.Run("card", "column", numStr, "--column", fixture.ColumnID))
	show = h.Run("card", "show", numStr)
	assertOK(t, show)
	if got := mapValueString(asMap(show.GetDataMap()["column"]), "id"); got != fixture.ColumnID {
		t.Fatalf("expected card column %q, got %q", fixture.ColumnID, got)
	}

	// Watch/unwatch currently have no CLI readback path on card show/list.
	assertOK(t, h.Run("card", "watch", numStr))
	assertOK(t, h.Run("card", "unwatch", numStr))

	// Mark-read/mark-unread likewise have no card-scoped CLI readback state today.
	assertOK(t, h.Run("card", "mark-read", numStr))
	assertOK(t, h.Run("card", "mark-unread", numStr))

	assertOK(t, h.Run("card", "pin", numStr))
	pins := h.Run("pin", "list")
	assertOK(t, pins)
	if listMapByNumber(pins.GetDataArray(), num) == nil {
		t.Fatalf("expected pin list to include card #%d after pin", num)
	}
	assertOK(t, h.Run("card", "unpin", numStr))
	pins = h.Run("pin", "list")
	assertOK(t, pins)
	if listMapByNumber(pins.GetDataArray(), num) != nil {
		t.Fatalf("expected pin list to exclude card #%d after unpin", num)
	}

	assertOK(t, h.Run("card", "golden", numStr))
	show = h.Run("card", "show", numStr)
	assertOK(t, show)
	if !show.GetDataBool("golden") {
		t.Fatal("expected card to be golden after golden command")
	}
	assertOK(t, h.Run("card", "ungolden", numStr))
	show = h.Run("card", "show", numStr)
	assertOK(t, show)
	if show.GetDataBool("golden") {
		t.Fatal("expected card to no longer be golden after ungolden command")
	}

	assertOK(t, h.Run("card", "tag", numStr, "--tag", tagTitle))
	tags := h.Run("tag", "list")
	assertOK(t, tags)
	var tagID string
	for _, item := range tags.GetDataArray() {
		m := asMap(item)
		if mapValueString(m, "title") == tagTitle {
			tagID = mapValueString(m, "id")
			break
		}
	}
	if tagID == "" {
		t.Fatalf("expected tag list to include %q", tagTitle)
	}
	taggedCards := h.Run("card", "list", "--tag", tagID)
	assertOK(t, taggedCards)
	if listMapByNumber(taggedCards.GetDataArray(), num) == nil {
		t.Fatalf("expected card list for tag %q to include card #%d", tagTitle, num)
	}

	assertOK(t, h.Run("card", "self-assign", numStr))
	show = h.Run("card", "show", numStr)
	assertOK(t, show)
	if listMapByID(asSlice(show.GetDataMap()["assignees"]), currentUser) == nil {
		t.Fatalf("expected assignees to include current user %q", currentUser)
	}

	assertOK(t, h.Run("card", "close", numStr))
	show = h.Run("card", "show", numStr)
	assertOK(t, show)
	if !show.GetDataBool("closed") {
		t.Fatal("expected card to be closed after close command")
	}

	assertOK(t, h.Run("card", "reopen", numStr))
	show = h.Run("card", "show", numStr)
	assertOK(t, show)
	if show.GetDataBool("closed") {
		t.Fatal("expected card to be open after reopen command")
	}

	assertOK(t, h.Run("card", "postpone", numStr))
	show = h.Run("card", "show", numStr)
	assertOK(t, show)
	if !show.GetDataBool("postponed") {
		t.Fatal("expected card to be postponed after postpone command")
	}

	assertOK(t, h.Run("card", "untriage", numStr))
	show = h.Run("card", "show", numStr)
	assertOK(t, show)
	if show.GetDataBool("postponed") {
		t.Fatal("expected card to no longer be postponed after untriage command")
	}
}

func TestCardAssignToCurrentUser(t *testing.T) {
	h := newHarness(t)
	num := createCard(t, h, fixture.BoardID)
	userID := currentUserID(t, h)

	assertOK(t, h.Run("card", "assign", strconv.Itoa(num), "--user", userID))

	show := h.Run("card", "show", strconv.Itoa(num))
	assertOK(t, show)
	assignees := asSlice(show.GetDataMap()["assignees"])
	if len(assignees) == 0 {
		t.Fatal("expected assigned card to include assignees")
	}
	for _, item := range assignees {
		if mapValueString(asMap(item), "id") == userID {
			return
		}
	}
	t.Fatalf("expected assignees to include user %q", userID)
}

func TestCardImageRemove(t *testing.T) {
	h := newHarness(t)
	boardID := createBoard(t, h)
	ref := uploadFixture(t, h, "test_image.png")
	result := h.Run("card", "create",
		"--board", boardID,
		"--title", "Image Remove Card "+strconv.FormatInt(time.Now().UnixNano(), 10),
		"--image", ref.SignedID,
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

	showWithImage := h.Run("card", "show", strconv.Itoa(num))
	assertOK(t, showWithImage)
	if got := showWithImage.GetDataString("image_url"); got == "" {
		t.Fatal("expected image_url before image removal")
	}

	assertOK(t, h.Run("card", "image-remove", strconv.Itoa(num)))

	showWithoutImage := h.Run("card", "show", strconv.Itoa(num))
	assertOK(t, showWithoutImage)
	if got := showWithoutImage.GetDataString("image_url"); got != "" {
		t.Fatalf("expected image_url to be cleared, got %q", got)
	}
}

func TestCardMoveBetweenBoards(t *testing.T) {
	h := newHarness(t)
	destinationBoardID := createBoard(t, h)
	num := createCard(t, h, fixture.BoardID)
	assertOK(t, h.Run("card", "move", strconv.Itoa(num), "--to", destinationBoardID))
	show := h.Run("card", "show", strconv.Itoa(num))
	assertOK(t, show)
	if got := mapValueString(asMap(show.GetDataMap()["board"]), "id"); got != destinationBoardID {
		t.Fatalf("expected moved card board %q, got %q", destinationBoardID, got)
	}
}

func TestCardAttachmentsShow(t *testing.T) {
	assertOK(t, newHarness(t).Run("card", "attachments", "show", strconv.Itoa(fixture.CardNumber)))
}

func TestCardDelete(t *testing.T) {
	h := newHarness(t)
	num := createCard(t, h, fixture.BoardID)
	deleteResult := h.Run("card", "delete", strconv.Itoa(num))
	assertOK(t, deleteResult)
	if !deleteResult.GetDataBool("deleted") {
		t.Fatal("expected deleted=true")
	}
	assertResult(t, h.Run("card", "show", strconv.Itoa(num)), harness.ExitNotFound)
}

func TestCardCreateRoundTrip(t *testing.T) {
	h := newHarness(t)
	result := h.Run("card", "create", "--board", fixture.BoardID, "--title", "Round Trip Card", "--description", "created by cli tests")
	assertOK(t, result)
	num := result.GetNumberFromLocation()
	if num == 0 {
		num = result.GetDataInt("number")
	}
	if num == 0 {
		t.Fatal("no card number in create response")
	}
	t.Cleanup(func() { newHarness(t).Run("card", "delete", strconv.Itoa(num)) })
	show := h.Run("card", "show", strconv.Itoa(num))
	assertOK(t, show)
	if got := show.GetDataString("title"); got != "Round Trip Card" {
		t.Fatalf("expected card title %q, got %q", "Round Trip Card", got)
	}
	if got := mapValueString(asMap(show.GetDataMap()["board"]), "id"); got != fixture.BoardID {
		t.Fatalf("expected card board %q, got %q", fixture.BoardID, got)
	}
}

func TestCardCreateWithUniqueTitle(t *testing.T) {
	h := newHarness(t)
	title := "Unique Card " + strconv.FormatInt(time.Now().UnixNano(), 10)
	result := h.Run("card", "create", "--board", fixture.BoardID, "--title", title)
	assertOK(t, result)
	num := result.GetNumberFromLocation()
	if num == 0 {
		num = result.GetDataInt("number")
	}
	if num == 0 {
		t.Fatal("no card number in create response")
	}
	t.Cleanup(func() { newHarness(t).Run("card", "delete", strconv.Itoa(num)) })
	show := h.Run("card", "show", strconv.Itoa(num))
	assertOK(t, show)
	if got := show.GetDataString("title"); got != title {
		t.Fatalf("expected card title %q, got %q", title, got)
	}
}
