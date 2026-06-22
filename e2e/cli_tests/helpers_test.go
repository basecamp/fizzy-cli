package clitests

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/basecamp/fizzy-cli/e2e/harness"
)

func newHarness(t *testing.T) *harness.Harness {
	t.Helper()
	return harness.NewWithConfig(t, cfg)
}

func assertResult(t *testing.T, result *harness.Result, wantExit int) {
	t.Helper()
	if result.ExitCode != wantExit {
		t.Fatalf("expected exit code %d, got %d\nstdout: %s\nstderr: %s", wantExit, result.ExitCode, result.Stdout, result.Stderr)
	}
	if result.Response == nil {
		t.Fatalf("expected JSON response envelope, got none\nstdout: %s\nstderr: %s", result.Stdout, result.Stderr)
	}
	wantOK := wantExit == harness.ExitSuccess
	if result.Response.OK != wantOK {
		t.Fatalf("expected ok=%v, got %v (error: %q)", wantOK, result.Response.OK, result.Response.Error)
	}
}

func assertOK(t *testing.T, result *harness.Result) {
	t.Helper()
	assertResult(t, result, harness.ExitSuccess)
}

func createBoard(t *testing.T, h *harness.Harness) string {
	t.Helper()
	name := fmt.Sprintf("CLI Throwaway Board %d", time.Now().UnixNano())
	r := h.Run("board", "create", "--name", name)
	assertOK(t, r)
	id := r.GetIDFromLocation()
	if id == "" {
		id = r.GetDataString("id")
	}
	if id == "" {
		t.Fatal("no board ID in create response")
	}
	t.Cleanup(func() { newHarness(t).Run("board", "delete", id) })
	return id
}

func createColumn(t *testing.T, h *harness.Harness, boardID, name string) string {
	t.Helper()
	if name == "" {
		name = fmt.Sprintf("CLI Throwaway Column %d", time.Now().UnixNano())
	}
	r := h.Run("column", "create", "--board", boardID, "--name", name)
	assertOK(t, r)
	id := r.GetIDFromLocation()
	if id == "" {
		id = r.GetDataString("id")
	}
	if id == "" {
		t.Fatal("no column ID in create response")
	}
	t.Cleanup(func() { newHarness(t).Run("column", "delete", id, "--board", boardID) })
	return id
}

func createCard(t *testing.T, h *harness.Harness, boardID string) int {
	t.Helper()
	if boardID == "" {
		boardID = fixture.BoardID
	}
	r := h.Run("card", "create", "--board", boardID, "--title", fmt.Sprintf("CLI Throwaway Card %d", time.Now().UnixNano()))
	assertOK(t, r)
	num := r.GetNumberFromLocation()
	if num == 0 {
		num = r.GetDataInt("number")
	}
	if num == 0 {
		t.Fatal("no card number in create response")
	}
	t.Cleanup(func() { newHarness(t).Run("card", "delete", strconv.Itoa(num)) })
	return num
}

func createComment(t *testing.T, h *harness.Harness, cardNumber int, body string) string {
	t.Helper()
	if body == "" {
		body = fmt.Sprintf("CLI Throwaway Comment %d", time.Now().UnixNano())
	}
	r := h.Run("comment", "create", "--card", strconv.Itoa(cardNumber), "--body", body)
	assertOK(t, r)
	id := r.GetIDFromLocation()
	if id == "" {
		id = r.GetDataString("id")
	}
	if id == "" {
		t.Fatal("no comment ID in create response")
	}
	t.Cleanup(func() { newHarness(t).Run("comment", "delete", id, "--card", strconv.Itoa(cardNumber)) })
	return id
}

func createStep(t *testing.T, h *harness.Harness, cardNumber int, content string) string {
	t.Helper()
	if content == "" {
		content = fmt.Sprintf("CLI Throwaway Step %d", time.Now().UnixNano())
	}
	r := h.Run("step", "create", "--card", strconv.Itoa(cardNumber), "--content", content)
	assertOK(t, r)
	id := r.GetIDFromLocation()
	if id == "" {
		id = r.GetDataString("id")
	}
	if id == "" {
		t.Fatal("no step ID in create response")
	}
	t.Cleanup(func() { newHarness(t).Run("step", "delete", id, "--card", strconv.Itoa(cardNumber)) })
	return id
}

func stringifyID(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return strconv.Itoa(int(x))
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func mapValueString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if v, ok := m[key]; ok {
			if s := stringifyID(v); s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return ""
}

func asMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func asSlice(v any) []any {
	s, _ := v.([]any)
	return s
}

func listMapByID(items []any, id string) map[string]any {
	for _, item := range items {
		m := asMap(item)
		if m != nil && mapValueString(m, "id") == id {
			return m
		}
	}
	return nil
}

func listMapByNumber(items []any, number int) map[string]any {
	want := strconv.Itoa(number)
	for _, item := range items {
		m := asMap(item)
		if m != nil && mapValueString(m, "number") == want {
			return m
		}
	}
	return nil
}

func listIndexByID(items []any, id string) int {
	for i, item := range items {
		m := asMap(item)
		if m != nil && mapValueString(m, "id") == id {
			return i
		}
	}
	return -1
}

func listAddedID(before, after []any) string {
	seen := make(map[string]struct{}, len(before))
	for _, item := range before {
		m := asMap(item)
		if m == nil {
			continue
		}
		if id := mapValueString(m, "id"); id != "" {
			seen[id] = struct{}{}
		}
	}
	for _, item := range after {
		m := asMap(item)
		if m == nil {
			continue
		}
		id := mapValueString(m, "id")
		if id == "" {
			continue
		}
		if _, ok := seen[id]; !ok {
			return id
		}
	}
	return ""
}

func addedReactionID(before, after []any, content, userID string) string {
	seen := make(map[string]struct{}, len(before))
	for _, item := range before {
		m := asMap(item)
		if m == nil {
			continue
		}
		if id := mapValueString(m, "id"); id != "" {
			seen[id] = struct{}{}
		}
	}

	matches := []string{}
	for _, item := range after {
		m := asMap(item)
		if m == nil {
			continue
		}
		id := mapValueString(m, "id")
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		if mapValueString(m, "content") != content {
			continue
		}
		reacter := asMap(m["reacter"])
		if userID != "" && mapValueString(reacter, "id") != userID {
			continue
		}
		matches = append(matches, id)
	}
	if len(matches) == 1 {
		return matches[0]
	}
	return ""
}

func bodyPlainText(m map[string]any) string {
	if m == nil {
		return ""
	}
	body := asMap(m["body"])
	if body == nil {
		return ""
	}
	return mapValueString(body, "plain_text")
}

func currentUserID(t *testing.T, h *harness.Harness) string {
	t.Helper()
	identity := h.Run("identity", "show")
	assertOK(t, identity)
	data := identity.GetDataMap()
	if data == nil {
		t.Skip("identity response had no object payload")
	}

	// Prefer account-scoped ID from accounts[].user.id.
	if accounts := asSlice(data["accounts"]); len(accounts) > 0 {
		var fallback string
		for _, item := range accounts {
			account := asMap(item)
			if account == nil {
				continue
			}
			user := asMap(account["user"])
			if user == nil {
				continue
			}
			id := mapValueString(user, "id")
			if id == "" {
				continue
			}
			if fallback == "" {
				fallback = id
			}
			accountID := mapValueString(account, "id")
			slug := mapValueString(account, "slug")
			name := mapValueString(account, "name")
			if cfg.Account == accountID || cfg.Account == slug || cfg.Account == name {
				return id
			}
		}
		if fallback != "" {
			return fallback
		}
	}

	identityName := mapValueString(data, "name")
	identityEmail := mapValueString(data, "email", "email_address")
	users := h.Run("user", "list")
	assertOK(t, users)
	for _, item := range users.GetDataArray() {
		user := asMap(item)
		if user == nil {
			continue
		}
		name := mapValueString(user, "name")
		email := mapValueString(user, "email", "email_address")
		if identityEmail != "" && email == identityEmail {
			if id := mapValueString(user, "id"); id != "" {
				return id
			}
		}
		if identityName != "" && name == identityName {
			if id := mapValueString(user, "id"); id != "" {
				return id
			}
		}
	}

	t.Skip("could not determine current account-scoped user ID")
	return ""
}

func notificationID(t *testing.T, h *harness.Harness) string {
	t.Helper()
	for _, args := range [][]string{{"notification", "tray", "--include-read"}, {"notification", "list"}} {
		result := h.Run(args...)
		if result.ExitCode != harness.ExitSuccess {
			continue
		}
		for _, item := range result.GetDataArray() {
			m := asMap(item)
			if m == nil {
				continue
			}
			if id := mapValueString(m, "id"); id != "" {
				return id
			}
		}
	}
	t.Skip("no notifications available")
	return ""
}

func avatarRedirects(t *testing.T, avatarURL string) bool {
	t.Helper()
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	for attempt := 0; attempt < 10; attempt++ {
		req, err := http.NewRequest(http.MethodHead, avatarURL, nil)
		if err != nil {
			t.Fatalf("build avatar HEAD request: %v", err)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("fetch avatar headers: %v", err)
		}
		resp.Body.Close()
		switch {
		case resp.StatusCode == http.StatusOK:
			return false
		case resp.StatusCode/100 == 3:
			return true
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("avatar endpoint %s did not settle on 200 or 3xx", avatarURL)
	return false
}
