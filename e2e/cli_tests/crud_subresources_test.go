package clitests

import (
	"strconv"
	"testing"
	"time"

	"github.com/basecamp/fizzy-cli/e2e/harness"
)

func TestColumnCRUDAndMove(t *testing.T) {
	h := newHarness(t)
	boardID := createBoard(t, h)
	leftID := createColumn(t, h, boardID, "Left")
	rightID := createColumn(t, h, boardID, "Right")

	list := h.Run("column", "list", "--board", boardID)
	assertOK(t, list)
	initial := list.GetDataArray()
	if initial == nil {
		t.Fatal("expected array response")
	}
	if listIndexByID(initial, leftID) == -1 || listIndexByID(initial, rightID) == -1 {
		t.Fatal("expected initial column list to include both columns")
	}

	show := h.Run("column", "show", leftID, "--board", boardID)
	assertOK(t, show)
	if got := show.GetDataString("name"); got != "Left" {
		t.Fatalf("expected column name %q, got %q", "Left", got)
	}

	assertOK(t, h.Run("column", "update", leftID, "--board", boardID, "--name", "Renamed Left"))
	show = h.Run("column", "show", leftID, "--board", boardID)
	assertOK(t, show)
	if got := show.GetDataString("name"); got != "Renamed Left" {
		t.Fatalf("expected renamed column, got %q", got)
	}

	assertOK(t, h.Run("column", "move-right", leftID))
	afterMoveRight := h.Run("column", "list", "--board", boardID)
	assertOK(t, afterMoveRight)
	if listIndexByID(afterMoveRight.GetDataArray(), leftID) <= listIndexByID(afterMoveRight.GetDataArray(), rightID) {
		t.Fatal("expected left column to appear after right column after move-right")
	}

	assertOK(t, h.Run("column", "move-left", leftID))
	afterMoveLeft := h.Run("column", "list", "--board", boardID)
	assertOK(t, afterMoveLeft)
	if listIndexByID(afterMoveLeft.GetDataArray(), leftID) >= listIndexByID(afterMoveLeft.GetDataArray(), rightID) {
		t.Fatal("expected left column to appear before right column after move-left")
	}

	assertOK(t, h.Run("column", "delete", rightID, "--board", boardID))
	assertResult(t, h.Run("column", "show", rightID, "--board", boardID), harness.ExitNotFound)
}

func TestCommentCRUD(t *testing.T) {
	h := newHarness(t)
	cardNum := fixture.CardNumber
	cardStr := strconv.Itoa(cardNum)

	list := h.Run("comment", "list", "--card", cardStr)
	assertOK(t, list)
	assertOK(t, h.Run("comment", "show", fixture.CommentID, "--card", cardStr))

	commentID := createComment(t, h, cardNum, "CLI comment")
	assertOK(t, h.Run("comment", "update", commentID, "--card", cardStr, "--body", "Updated CLI comment"))
	show := h.Run("comment", "show", commentID, "--card", cardStr)
	assertOK(t, show)
	if got := bodyPlainText(show.GetDataMap()); got != "Updated CLI comment" {
		t.Fatalf("expected updated comment body %q, got %q", "Updated CLI comment", got)
	}

	assertOK(t, h.Run("comment", "delete", commentID, "--card", cardStr))
	comments := h.Run("comment", "list", "--card", cardStr)
	assertOK(t, comments)
	if listMapByID(comments.GetDataArray(), commentID) != nil {
		t.Fatalf("expected deleted comment %q to be absent from list", commentID)
	}
}

func TestStepCRUD(t *testing.T) {
	h := newHarness(t)
	cardStr := strconv.Itoa(fixture.CardNumber)
	list := h.Run("step", "list", "--card", cardStr)
	assertOK(t, list)
	assertOK(t, h.Run("step", "show", fixture.StepID, "--card", cardStr))

	stepID := createStep(t, h, fixture.CardNumber, "CLI step")
	assertOK(t, h.Run("step", "update", stepID, "--card", cardStr, "--content", "Updated CLI step"))
	show := h.Run("step", "show", stepID, "--card", cardStr)
	assertOK(t, show)
	if got := show.GetDataString("content"); got != "Updated CLI step" {
		t.Fatalf("expected updated step content %q, got %q", "Updated CLI step", got)
	}

	assertOK(t, h.Run("step", "delete", stepID, "--card", cardStr))
	steps := h.Run("step", "list", "--card", cardStr)
	assertOK(t, steps)
	if listMapByID(steps.GetDataArray(), stepID) != nil {
		t.Fatalf("expected deleted step %q to be absent from list", stepID)
	}
}

func TestReactionCRUD(t *testing.T) {
	h := newHarness(t)
	userID := currentUserID(t, h)
	cardStr := strconv.Itoa(fixture.CardNumber)
	cardReactionContent := "+1"
	commentReactionContent := "heart"

	cardReactionsBefore := h.Run("reaction", "list", "--card", cardStr)
	assertOK(t, cardReactionsBefore)
	commentReactionsBefore := h.Run("reaction", "list", "--card", cardStr, "--comment", fixture.CommentID)
	assertOK(t, commentReactionsBefore)

	cardReaction := h.Run("reaction", "create", "--card", cardStr, "--content", cardReactionContent)
	assertOK(t, cardReaction)
	cardReactions := h.Run("reaction", "list", "--card", cardStr)
	assertOK(t, cardReactions)
	cardReactionID := addedReactionID(cardReactionsBefore.GetDataArray(), cardReactions.GetDataArray(), cardReactionContent, userID)
	if cardReactionID == "" {
		t.Fatal("expected exactly one created card reaction for the current user to appear in list")
	}
	t.Cleanup(func() {
		if cardReactionID != "" {
			newHarness(t).Run("reaction", "delete", cardReactionID, "--card", cardStr)
		}
	})
	if listMapByID(cardReactions.GetDataArray(), cardReactionID) == nil {
		t.Fatalf("expected card reaction list to include %q", cardReactionID)
	}
	deletedCardReactionID := cardReactionID
	assertOK(t, h.Run("reaction", "delete", cardReactionID, "--card", cardStr))
	cardReactionID = ""
	cardReactions = h.Run("reaction", "list", "--card", cardStr)
	assertOK(t, cardReactions)
	if listMapByID(cardReactions.GetDataArray(), deletedCardReactionID) != nil {
		t.Fatalf("expected deleted card reaction %q to be absent from list", deletedCardReactionID)
	}

	commentReaction := h.Run("reaction", "create", "--card", cardStr, "--comment", fixture.CommentID, "--content", commentReactionContent)
	assertOK(t, commentReaction)
	commentReactions := h.Run("reaction", "list", "--card", cardStr, "--comment", fixture.CommentID)
	assertOK(t, commentReactions)
	commentReactionID := addedReactionID(commentReactionsBefore.GetDataArray(), commentReactions.GetDataArray(), commentReactionContent, userID)
	if commentReactionID == "" {
		t.Fatal("expected exactly one created comment reaction for the current user to appear in list")
	}
	t.Cleanup(func() {
		if commentReactionID != "" {
			newHarness(t).Run("reaction", "delete", commentReactionID, "--card", cardStr, "--comment", fixture.CommentID)
		}
	})
	if listMapByID(commentReactions.GetDataArray(), commentReactionID) == nil {
		t.Fatalf("expected comment reaction list to include %q", commentReactionID)
	}
	deletedCommentReactionID := commentReactionID
	assertOK(t, h.Run("reaction", "delete", commentReactionID, "--card", cardStr, "--comment", fixture.CommentID))
	commentReactionID = ""
	commentReactions = h.Run("reaction", "list", "--card", cardStr, "--comment", fixture.CommentID)
	assertOK(t, commentReactions)
	if listMapByID(commentReactions.GetDataArray(), deletedCommentReactionID) != nil {
		t.Fatalf("expected deleted comment reaction %q to be absent from list", deletedCommentReactionID)
	}
}

func TestNotificationCommands(t *testing.T) {
	h := newHarness(t)
	assertOK(t, h.Run("notification", "list"))
	assertOK(t, h.Run("notification", "tray"))
	assertOK(t, h.Run("notification", "settings-show"))

	show := h.Run("notification", "settings-show")
	assertOK(t, show)
	currentFreq := show.GetDataString("bundle_email_frequency")
	if currentFreq == "" {
		currentFreq = "never"
	}
	assertOK(t, h.Run("notification", "settings-update", "--bundle-email-frequency", currentFreq))
	updatedSettings := h.Run("notification", "settings-show")
	assertOK(t, updatedSettings)
	if got := updatedSettings.GetDataString("bundle_email_frequency"); got != currentFreq {
		t.Fatalf("expected bundle_email_frequency %q, got %q", currentFreq, got)
	}

	id := notificationID(t, h)
	assertOK(t, h.Run("notification", "unread", id))
	tray := h.Run("notification", "tray", "--include-read")
	assertOK(t, tray)
	notif := listMapByID(tray.GetDataArray(), id)
	if notif == nil {
		t.Fatalf("expected notification tray to include %q", id)
	}
	if read, ok := notif["read"].(bool); !ok || read {
		t.Fatal("expected notification to be unread after unread command")
	}

	assertOK(t, h.Run("notification", "read", id))
	tray = h.Run("notification", "tray", "--include-read")
	assertOK(t, tray)
	notif = listMapByID(tray.GetDataArray(), id)
	if notif == nil {
		t.Fatalf("expected notification tray to include %q after read", id)
	}
	if read, ok := notif["read"].(bool); !ok || !read {
		t.Fatal("expected notification to be read after read command")
	}

	assertOK(t, h.Run("notification", "unread", id))
	assertOK(t, h.Run("notification", "read-all"))
	tray = h.Run("notification", "tray", "--include-read")
	assertOK(t, tray)
	notif = listMapByID(tray.GetDataArray(), id)
	if notif == nil {
		t.Fatalf("expected notification tray to include %q after read-all", id)
	}
	if read, ok := notif["read"].(bool); !ok || !read {
		t.Fatal("expected notification to be read after read-all")
	}
}

func TestTagAndPinLists(t *testing.T) {
	h := newHarness(t)
	cardNum := createCard(t, h, fixture.BoardID)
	cardStr := strconv.Itoa(cardNum)
	tagTitle := "cli-test"

	assertOK(t, h.Run("card", "tag", cardStr, "--tag", tagTitle))
	tagResult := h.Run("tag", "list")
	assertOK(t, tagResult)
	if tagResult.GetDataArray() == nil {
		t.Fatal("expected tag list array response")
	}
	var tagID string
	for _, item := range tagResult.GetDataArray() {
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
	if listMapByNumber(taggedCards.GetDataArray(), cardNum) == nil {
		t.Fatalf("expected tagged card list to include card #%d", cardNum)
	}

	assertOK(t, h.Run("card", "pin", cardStr))
	pinResult := h.Run("pin", "list")
	assertOK(t, pinResult)
	if pinResult.GetDataArray() == nil {
		t.Fatal("expected pin list array response")
	}
	if listMapByNumber(pinResult.GetDataArray(), cardNum) == nil {
		t.Fatalf("expected pin list to include card #%d", cardNum)
	}
	assertOK(t, h.Run("card", "unpin", cardStr))
}

func TestCommentAndStepCreationOnThrowawayCard(t *testing.T) {
	h := newHarness(t)
	cardNum := createCard(t, h, fixture.BoardID)
	cardStr := strconv.Itoa(cardNum)
	commentBody := "Throwaway card comment " + strconv.FormatInt(time.Now().UnixNano(), 10)
	stepContent := "Throwaway card step " + strconv.FormatInt(time.Now().UnixNano(), 10)
	commentID := createComment(t, h, cardNum, commentBody)
	stepID := createStep(t, h, cardNum, stepContent)
	if commentID == "" || stepID == "" {
		t.Fatal("expected comment and step IDs")
	}
	commentShow := h.Run("comment", "show", commentID, "--card", cardStr)
	assertOK(t, commentShow)
	if got := bodyPlainText(commentShow.GetDataMap()); got != commentBody {
		t.Fatalf("expected comment body %q, got %q", commentBody, got)
	}
	stepShow := h.Run("step", "show", stepID, "--card", cardStr)
	assertOK(t, stepShow)
	if got := stepShow.GetDataString("content"); got != stepContent {
		t.Fatalf("expected step content %q, got %q", stepContent, got)
	}
}
