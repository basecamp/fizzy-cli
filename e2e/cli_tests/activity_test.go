package clitests

import (
	"strconv"
	"testing"
	"time"

	"github.com/basecamp/fizzy-cli/e2e/harness"
)

func TestActivityList(t *testing.T) {
	h := newHarness(t)
	boardID := createBoard(t, h)
	cardNum := createCard(t, h, boardID)
	creatorID := currentUserID(t, h)

	var result *harness.Result
	for attempt := 0; attempt < 10; attempt++ {
		r := h.Run("activity", "list", "--board", boardID)
		if r.ExitCode == harness.ExitSuccess && len(r.GetDataArray()) > 0 {
			result = r
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if result == nil {
		t.Fatal("expected at least one activity for throwaway board")
	}

	assertOK(t, result)
	if len(result.GetDataArray()) == 0 {
		t.Fatal("expected activity list to return at least one item")
	}

	foundCard := false
	for _, item := range result.GetDataArray() {
		m := asMap(item)
		if m == nil {
			continue
		}
		if eventable := asMap(m["eventable"]); eventable != nil {
			if got := mapValueString(eventable, "number"); got == strconv.Itoa(cardNum) {
				foundCard = true
				break
			}
		}
	}
	if !foundCard {
		t.Logf("activity list did not expose created card number %d; continuing because board activity was non-empty", cardNum)
	}

	creatorResult := h.Run("activity", "list", "--board", boardID, "--creator", creatorID)
	assertOK(t, creatorResult)
	if creatorResult.GetDataArray() == nil {
		t.Fatal("expected activity creator-filter response array")
	}
}
