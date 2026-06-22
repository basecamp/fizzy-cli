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

	cardNumStr := strconv.Itoa(cardNum)
	var result *harness.Result
	foundCard := false
	for attempt := 0; attempt < 10; attempt++ {
		r := h.Run("activity", "list", "--board", boardID)
		if r.ExitCode == harness.ExitSuccess {
			result = r
			for _, item := range r.GetDataArray() {
				m := asMap(item)
				if m == nil {
					continue
				}
				if eventable := asMap(m["eventable"]); eventable != nil {
					if mapValueString(eventable, "number") == cardNumStr {
						foundCard = true
						break
					}
				}
			}
			if foundCard {
				break
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	if result == nil {
		t.Fatal("expected at least one successful activity list call")
	}
	assertOK(t, result)
	if !foundCard {
		t.Fatalf("activity list did not expose created card number %d after retries", cardNum)
	}

	creatorResult := h.Run("activity", "list", "--board", boardID, "--creator", creatorID)
	assertOK(t, creatorResult)
	if creatorResult.GetDataArray() == nil {
		t.Fatal("expected activity creator-filter response array")
	}
}
