package clitests

import (
	"strconv"
	"testing"
	"time"

	"github.com/basecamp/fizzy-cli/e2e/harness"
)

func TestWebhookCRUD(t *testing.T) {
	h := newHarness(t)
	boardID := createBoard(t, h)
	name := "CLI Test Hook " + strconv.FormatInt(time.Now().UnixNano(), 10)

	create := h.Run("webhook", "create",
		"--board", boardID,
		"--name", name,
		"--url", "https://example.com/fizzy-cli-webhook",
	)
	assertOK(t, create)
	webhookID := create.GetIDFromLocation()
	if webhookID == "" {
		webhookID = create.GetDataString("id")
	}
	if webhookID == "" {
		t.Fatal("no webhook ID in create response")
	}
	deleted := false
	t.Cleanup(func() {
		if !deleted {
			newHarness(t).Run("webhook", "delete", "--board", boardID, webhookID)
		}
	})

	show := h.Run("webhook", "show", "--board", boardID, webhookID)
	assertOK(t, show)
	if got := show.GetDataString("name"); got != name {
		t.Fatalf("expected webhook name %q, got %q", name, got)
	}
	if got := show.GetDataString("payload_url"); got == "" {
		t.Fatal("expected payload_url in webhook show response")
	}

	list := h.Run("webhook", "list", "--board", boardID)
	assertOK(t, list)
	found := false
	for _, item := range list.GetDataArray() {
		if mapValueString(asMap(item), "id") == webhookID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected webhook list to include %q", webhookID)
	}

	updatedName := name + " Updated"
	update := h.Run("webhook", "update", "--board", boardID, webhookID, "--name", updatedName, "--actions", "card_closed")
	assertOK(t, update)

	showUpdated := h.Run("webhook", "show", "--board", boardID, webhookID)
	assertOK(t, showUpdated)
	if got := showUpdated.GetDataString("name"); got != updatedName {
		t.Fatalf("expected updated webhook name %q, got %q", updatedName, got)
	}
	actions := asSlice(showUpdated.GetDataMap()["subscribed_actions"])
	if len(actions) != 1 || stringifyID(actions[0]) != "card_closed" {
		t.Fatalf("expected subscribed_actions [card_closed], got %v", actions)
	}

	deleteResult := h.Run("webhook", "delete", "--board", boardID, webhookID)
	assertOK(t, deleteResult)
	deleted = true
	if !deleteResult.GetDataBool("deleted") {
		t.Fatal("expected deleted=true")
	}
	assertResult(t, h.Run("webhook", "show", "--board", boardID, webhookID), harness.ExitNotFound)
}

func TestWebhookDeliveries(t *testing.T) {
	h := newHarness(t)
	boardID := createBoard(t, h)
	cardNum := createCard(t, h, boardID)
	name := "CLI Delivery Hook " + strconv.FormatInt(time.Now().UnixNano(), 10)

	create := h.Run("webhook", "create",
		"--board", boardID,
		"--name", name,
		"--url", "https://example.com/fizzy-cli-webhook-deliveries",
		"--actions", "card_closed",
	)
	assertOK(t, create)
	webhookID := create.GetIDFromLocation()
	if webhookID == "" {
		webhookID = create.GetDataString("id")
	}
	if webhookID == "" {
		t.Fatal("no webhook ID in create response")
	}
	t.Cleanup(func() {
		newHarness(t).Run("webhook", "delete", "--board", boardID, webhookID)
	})

	assertOK(t, h.Run("card", "close", strconv.Itoa(cardNum)))

	var deliveries *harness.Result
	for attempt := 0; attempt < 15; attempt++ {
		r := h.Run("webhook", "deliveries", "--board", boardID, webhookID)
		if r.ExitCode == harness.ExitSuccess && len(r.GetDataArray()) > 0 {
			deliveries = r
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if deliveries == nil {
		t.Fatal("expected at least one webhook delivery after triggering card_closed")
	}

	assertOK(t, deliveries)
	if len(deliveries.GetDataArray()) == 0 {
		t.Fatal("expected webhook deliveries to be non-empty")
	}
	first := asMap(deliveries.GetDataArray()[0])
	if mapValueString(first, "id") == "" {
		t.Fatal("expected delivery id")
	}
	if mapValueString(first, "state") == "" {
		t.Fatal("expected delivery state")
	}

	assertOK(t, h.Run("webhook", "deliveries", "--board", boardID, webhookID, "--all"))
}
