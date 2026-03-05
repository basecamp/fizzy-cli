package tests

import (
	"testing"

	"github.com/basecamp/fizzy-cli/e2e/harness"
)

func TestNotificationList(t *testing.T) {
	h := harness.New(t)

	t.Run("returns list of notifications", func(t *testing.T) {
		result := h.Run("notification", "list")

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if result.Response == nil {
			t.Fatalf("expected JSON response, got nil\nstdout: %s", result.Stdout)
		}

		if !result.Response.OK {
			t.Error("expected ok=true")
		}

		arr := result.GetDataArray()
		if arr == nil {
			t.Error("expected data to be an array")
		}
	})

	t.Run("supports --page option", func(t *testing.T) {
		result := h.Run("notification", "list", "--page", "1")

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d", harness.ExitSuccess, result.ExitCode)
		}

		if result.Response == nil || !result.Response.OK {
			t.Error("expected successful response")
		}
	})

	t.Run("supports --all flag", func(t *testing.T) {
		result := h.Run("notification", "list", "--all")

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d", harness.ExitSuccess, result.ExitCode)
		}
	})
}

func TestNotificationReadUnread(t *testing.T) {
	h := harness.New(t)

	// First get a notification ID from the list (if any exist)
	listResult := h.Run("notification", "list")
	if listResult.ExitCode != harness.ExitSuccess {
		t.Fatalf("failed to list notifications: %s", listResult.Stderr)
	}

	notifications := listResult.GetDataArray()
	if len(notifications) == 0 {
		t.Skip("no notifications available to test")
	}

	firstNotification, ok := notifications[0].(map[string]any)
	if !ok {
		t.Fatal("expected notification to be a map")
	}

	notificationID, ok := firstNotification["id"].(string)
	if !ok || notificationID == "" {
		t.Fatal("expected notification to have id")
	}

	t.Run("mark notification as read", func(t *testing.T) {
		result := h.Run("notification", "read", notificationID)

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if result.Response == nil {
			t.Fatal("expected JSON response")
		}

		if !result.Response.OK {
			t.Errorf("expected ok=true, error: %+v", result.Response.Error)
		}
	})

	t.Run("mark notification as unread", func(t *testing.T) {
		result := h.Run("notification", "unread", notificationID)

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if result.Response == nil {
			t.Fatal("expected JSON response")
		}

		if !result.Response.OK {
			t.Errorf("expected ok=true, error: %+v", result.Response.Error)
		}
	})
}

func TestNotificationTray(t *testing.T) {
	h := harness.New(t)

	t.Run("returns notification tray", func(t *testing.T) {
		result := h.Run("notification", "tray")

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if result.Response == nil {
			t.Fatalf("expected JSON response, got nil\nstdout: %s", result.Stdout)
		}

		if !result.Response.OK {
			t.Error("expected ok=true")
		}

		arr := result.GetDataArray()
		if arr == nil {
			t.Error("expected data to be an array")
		}
	})

	t.Run("supports --include-read flag", func(t *testing.T) {
		result := h.Run("notification", "tray", "--include-read")

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if result.Response == nil {
			t.Fatalf("expected JSON response, got nil\nstdout: %s", result.Stdout)
		}

		if !result.Response.OK {
			t.Error("expected ok=true")
		}

		arr := result.GetDataArray()
		if arr == nil {
			t.Error("expected data to be an array")
		}
	})
}

func TestNotificationReadAll(t *testing.T) {
	h := harness.New(t)

	t.Run("marks all notifications as read", func(t *testing.T) {
		result := h.Run("notification", "read-all")

		if result.ExitCode != harness.ExitSuccess {
			t.Errorf("expected exit code %d, got %d\nstderr: %s", harness.ExitSuccess, result.ExitCode, result.Stderr)
		}

		if result.Response == nil {
			t.Fatal("expected JSON response")
		}

		if !result.Response.OK {
			t.Errorf("expected ok=true, error: %+v", result.Response.Error)
		}
	})
}

func TestNotificationReadNotFound(t *testing.T) {
	h := harness.New(t)

	t.Run("returns not found for non-existent notification", func(t *testing.T) {
		result := h.Run("notification", "read", "non-existent-notification-id")

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
	})
}
