package clitests

import (
	"strconv"
	"testing"

	"github.com/basecamp/fizzy-cli/e2e/harness"
)

func TestBoardBoardScopedCommandsUseBoardFlag(t *testing.T) {
	h := newHarness(t)
	for name, args := range map[string][]string{
		"accesses":  {"board", "accesses", "--board", fixture.BoardID},
		"closed":    {"board", "closed", "--board", fixture.BoardID},
		"postponed": {"board", "postponed", "--board", fixture.BoardID},
		"stream":    {"board", "stream", "--board", fixture.BoardID},
	} {
		t.Run(name, func(t *testing.T) {
			assertOK(t, h.Run(args...))
		})
	}
}

func TestCardAttachmentsUseShowSubcommand(t *testing.T) {
	h := newHarness(t)
	assertOK(t, h.Run("card", "attachments", "show", strconv.Itoa(fixture.CardNumber)))
}

func TestColumnMoveUsesColumnIDOnly(t *testing.T) {
	h := newHarness(t)
	boardID := createBoard(t, h)
	leftID := createColumn(t, h, boardID, "Left")
	rightID := createColumn(t, h, boardID, "Right")
	assertOK(t, h.Run("column", "move-right", leftID))
	assertOK(t, h.Run("column", "move-left", rightID))
}

func TestNotificationReadUnreadUseNotificationID(t *testing.T) {
	h := newHarness(t)
	id := notificationID(t, h)
	assertOK(t, h.Run("notification", "read", id))
	assertOK(t, h.Run("notification", "unread", id))
}

func TestTagListDoesNotTakeBoardFlag(t *testing.T) {
	h := newHarness(t)
	result := h.Run("tag", "list", "--board", fixture.BoardID)
	assertResult(t, result, harness.ExitUsage)
}

func TestBoardCreateRequiresName(t *testing.T) {
	h := newHarness(t)
	assertResult(t, h.Run("board", "create"), harness.ExitUsage)
}

func TestCardCreateRequiresTitle(t *testing.T) {
	h := newHarness(t)
	assertResult(t, h.Run("card", "create", "--board", fixture.BoardID), harness.ExitUsage)
}

func TestNotificationUnreadRequiresID(t *testing.T) {
	h := newHarness(t)
	assertResult(t, h.Run("notification", "unread"), harness.ExitUsage)
}

func TestAccountEntropyRejectsInvalidZeroValue(t *testing.T) {
	h := newHarness(t)
	result := h.Run("account", "entropy", "--auto_postpone_period_in_days", "0")
	assertResult(t, result, harness.ExitUsage)
}
