package clitests

import (
	"strconv"
	"testing"
	"time"

	"github.com/basecamp/fizzy-cli/e2e/harness"
)

func TestAccessTokenCRUD(t *testing.T) {
	h := newHarness(t)
	description := "CLI Test Token " + strconv.FormatInt(time.Now().UnixNano(), 10)

	create := h.Run("token", "create", "--description", description, "--permission", "read")
	assertOK(t, create)
	tokenID := create.GetDataString("id")
	if tokenID == "" {
		t.Fatal("no token ID in create response")
	}
	deleted := false
	t.Cleanup(func() {
		if deleted {
			return
		}
		cleanupDelete := newHarness(t).Run("token", "delete", tokenID)
		if cleanupDelete.ExitCode != harness.ExitSuccess {
			t.Errorf("cleanup failed deleting token %q: exit=%d stdout=%s stderr=%s", tokenID, cleanupDelete.ExitCode, cleanupDelete.Stdout, cleanupDelete.Stderr)
			return
		}
		if !cleanupDelete.GetDataBool("deleted") {
			t.Errorf("cleanup delete for token %q did not report deleted=true", tokenID)
		}
	})
	if create.GetDataString("token") == "" {
		t.Fatal("expected raw token value in create response")
	}

	list := h.Run("token", "list")
	assertOK(t, list)
	found := false
	for _, item := range list.GetDataArray() {
		if mapValueString(asMap(item), "id") == tokenID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected token list to include %q", tokenID)
	}

	deleteResult := h.Run("token", "delete", tokenID)
	assertOK(t, deleteResult)
	deleted = true
	if !deleteResult.GetDataBool("deleted") {
		t.Fatal("expected deleted=true")
	}
}
