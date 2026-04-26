package clitests

import (
	"strconv"
	"testing"
)

func TestAccountShow(t *testing.T) {
	assertOK(t, newHarness(t).Run("account", "show"))
}

func TestAccountSettingsUpdateWithCurrentName(t *testing.T) {
	h := newHarness(t)
	show := h.Run("account", "show")
	assertOK(t, show)
	currentName := show.GetDataString("name")
	if currentName == "" {
		t.Skip("account show returned no name")
	}
	assertOK(t, h.Run("account", "settings-update", "--name", currentName))
	show = h.Run("account", "show")
	assertOK(t, show)
	if got := show.GetDataString("name"); got != currentName {
		t.Fatalf("expected account name %q after settings-update, got %q", currentName, got)
	}
}

func TestAccountEntropyWithCurrentValue(t *testing.T) {
	h := newHarness(t)
	show := h.Run("account", "show")
	assertOK(t, show)
	days := show.GetDataInt("auto_postpone_period_in_days")
	if days == 0 {
		days = 7
	}
	assertOK(t, h.Run("account", "entropy", "--auto_postpone_period_in_days", strconv.Itoa(days)))
	show = h.Run("account", "show")
	assertOK(t, show)
	if got := show.GetDataInt("auto_postpone_period_in_days"); got != days {
		t.Fatalf("expected auto_postpone_period_in_days=%d, got %d", days, got)
	}
}

func TestAccountJoinCodeShow(t *testing.T) {
	assertOK(t, newHarness(t).Run("account", "join-code-show"))
}

func TestAccountExportCreateShow(t *testing.T) {
	h := newHarness(t)
	create := h.Run("account", "export-create")
	assertOK(t, create)
	exportID := create.GetDataString("id")
	if exportID == "" {
		exportID = mapValueString(create.GetDataMap(), "id")
	}
	if exportID == "" {
		t.Fatal("expected export ID in export-create response")
	}
	show := h.Run("account", "export-show", exportID)
	assertOK(t, show)
	if got := mapValueString(show.GetDataMap(), "id"); got != exportID {
		t.Fatalf("expected export-show id %q, got %q", exportID, got)
	}
	if got := mapValueString(show.GetDataMap(), "status"); got == "" {
		t.Fatal("expected export status in export-show response")
	}
}

func TestUserList(t *testing.T) {
	result := newHarness(t).Run("user", "list")
	assertOK(t, result)
	if result.GetDataArray() == nil {
		t.Fatal("expected array response")
	}
}

func TestUserShowAndUpdateOwnProfile(t *testing.T) {
	h := newHarness(t)
	userID := currentUserID(t, h)
	show := h.Run("user", "show", userID)
	assertOK(t, show)
	currentName := show.GetDataString("name")
	if currentName == "" {
		t.Skip("user show returned no name")
	}
	assertOK(t, h.Run("user", "update", userID, "--name", currentName))
	show = h.Run("user", "show", userID)
	assertOK(t, show)
	if got := show.GetDataString("name"); got != currentName {
		t.Fatalf("expected user name %q after update, got %q", currentName, got)
	}
}

func TestUserAvatarUpdateAndRemove(t *testing.T) {
	h := newHarness(t)
	userID := currentUserID(t, h)
	fixturePath := fixtureFile(t, "test_image.png")

	show := h.Run("user", "show", userID)
	assertOK(t, show)
	avatarURL := show.GetDataString("avatar_url")
	if avatarURL == "" {
		t.Skip("user show returned no avatar_url")
	}
	initiallyAttached := avatarRedirects(t, avatarURL)
	if initiallyAttached {
		t.Cleanup(func() {
			assertOK(t, newHarness(t).Run("user", "update", userID, "--avatar", fixturePath))
			if !avatarRedirects(t, avatarURL) {
				t.Fatal("expected avatar to be restored")
			}
		})
	}

	assertOK(t, h.Run("user", "update", userID, "--avatar", fixturePath))
	if !avatarRedirects(t, avatarURL) {
		t.Fatal("expected uploaded avatar endpoint to redirect to an image blob")
	}

	assertOK(t, h.Run("user", "avatar-remove", userID))
	if avatarRedirects(t, avatarURL) {
		t.Fatal("expected avatar endpoint to fall back to generated SVG after removal")
	}
}

func TestUserExportCreateShow(t *testing.T) {
	h := newHarness(t)
	userID := currentUserID(t, h)

	create := h.Run("user", "export-create", userID)
	assertOK(t, create)
	exportID := create.GetDataString("id")
	if exportID == "" {
		exportID = mapValueString(create.GetDataMap(), "id")
	}
	if exportID == "" {
		t.Fatal("expected export ID in user export-create response")
	}

	show := h.Run("user", "export-show", userID, exportID)
	assertOK(t, show)
	if got := mapValueString(show.GetDataMap(), "id"); got != exportID {
		t.Fatalf("expected export-show id %q, got %q", exportID, got)
	}
	if got := mapValueString(show.GetDataMap(), "status"); got == "" {
		t.Fatal("expected export status in user export-show response")
	}
}
