package response

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/basecamp/fizzy-cli/internal/errors"
)

func TestSuccess(t *testing.T) {
	data := map[string]string{"name": "test"}
	resp := Success(data)

	if !resp.Success {
		t.Error("expected Success to be true")
	}
	if resp.Data == nil {
		t.Error("expected Data to be set")
	}
	if resp.Error != nil {
		t.Error("expected Error to be nil")
	}
	if resp.Meta == nil {
		t.Error("expected Meta to be set")
	}
	if _, ok := resp.Meta["timestamp"]; !ok {
		t.Error("expected Meta to contain timestamp")
	}
}

func TestSuccessWithLocation(t *testing.T) {
	data := map[string]string{"id": "123"}
	location := "https://example.com/resource/123"
	resp := SuccessWithLocation(data, location)

	if !resp.Success {
		t.Error("expected Success to be true")
	}
	if resp.Location != location {
		t.Errorf("expected Location '%s', got '%s'", location, resp.Location)
	}
}

func TestSuccessWithPagination(t *testing.T) {
	data := []string{"item1", "item2"}

	t.Run("with pagination", func(t *testing.T) {
		resp := SuccessWithPagination(data, true, "https://example.com/page2")

		if !resp.Success {
			t.Error("expected Success to be true")
		}
		if resp.Pagination == nil {
			t.Fatal("expected Pagination to be set")
		}
		if !resp.Pagination.HasNext {
			t.Error("expected HasNext to be true")
		}
		if resp.Pagination.NextURL != "https://example.com/page2" {
			t.Errorf("expected NextURL 'https://example.com/page2', got '%s'", resp.Pagination.NextURL)
		}
	})

	t.Run("without pagination", func(t *testing.T) {
		resp := SuccessWithPagination(data, false, "")

		if resp.Pagination != nil {
			t.Error("expected Pagination to be nil when no next page")
		}
	})
}

func TestError(t *testing.T) {
	cliErr := &errors.CLIError{
		Code:    "TEST_ERROR",
		Message: "test error message",
		Status:  400,
	}
	resp := Error(cliErr)

	if resp.Success {
		t.Error("expected Success to be false")
	}
	if resp.Data != nil {
		t.Error("expected Data to be nil")
	}
	if resp.Error == nil {
		t.Fatal("expected Error to be set")
	}
	if resp.Error.Code != "TEST_ERROR" {
		t.Errorf("expected Code 'TEST_ERROR', got '%s'", resp.Error.Code)
	}
	if resp.Error.Message != "test error message" {
		t.Errorf("expected Message 'test error message', got '%s'", resp.Error.Message)
	}
	if resp.Error.Status != 400 {
		t.Errorf("expected Status 400, got %d", resp.Error.Status)
	}
}

func TestErrorWithoutStatus(t *testing.T) {
	cliErr := &errors.CLIError{
		Code:    "ERROR",
		Message: "generic error",
	}
	resp := Error(cliErr)

	if resp.Error.Status != 0 {
		t.Errorf("expected Status 0, got %d", resp.Error.Status)
	}
}

func TestErrorFromError(t *testing.T) {
	t.Run("with CLIError", func(t *testing.T) {
		cliErr := errors.NewNotFoundError("not found")
		resp := ErrorFromError(cliErr)

		if resp.Error.Code != "NOT_FOUND" {
			t.Errorf("expected Code 'NOT_FOUND', got '%s'", resp.Error.Code)
		}
	})

	t.Run("with generic error", func(t *testing.T) {
		err := &testError{msg: "generic error"}
		resp := ErrorFromError(err)

		if resp.Error.Code != "ERROR" {
			t.Errorf("expected Code 'ERROR', got '%s'", resp.Error.Code)
		}
		if resp.Error.Message != "generic error" {
			t.Errorf("expected Message 'generic error', got '%s'", resp.Error.Message)
		}
	})
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestResponseJSONSerialization(t *testing.T) {
	t.Run("success response", func(t *testing.T) {
		resp := Success(map[string]string{"key": "value"})
		data, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var parsed Response
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if !parsed.Success {
			t.Error("expected Success to be true after parsing")
		}
	})

	t.Run("error response", func(t *testing.T) {
		resp := Error(errors.NewAuthError("unauthorized"))
		data, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var parsed Response
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if parsed.Success {
			t.Error("expected Success to be false after parsing")
		}
		if parsed.Error == nil {
			t.Fatal("expected Error to be set after parsing")
		}
		if parsed.Error.Code != "AUTH_ERROR" {
			t.Errorf("expected Code 'AUTH_ERROR', got '%s'", parsed.Error.Code)
		}
	})
}

func TestExitCode(t *testing.T) {
	tests := []struct {
		name     string
		resp     *Response
		expected int
	}{
		{
			name:     "success",
			resp:     Success(nil),
			expected: errors.ExitSuccess,
		},
		{
			name:     "auth error",
			resp:     Error(errors.NewAuthError("unauthorized")),
			expected: errors.ExitAuthFailure,
		},
		{
			name:     "forbidden",
			resp:     Error(errors.NewForbiddenError("forbidden")),
			expected: errors.ExitForbidden,
		},
		{
			name:     "not found",
			resp:     Error(errors.NewNotFoundError("not found")),
			expected: errors.ExitNotFound,
		},
		{
			name:     "validation error",
			resp:     Error(errors.NewValidationError("invalid")),
			expected: errors.ExitValidation,
		},
		{
			name:     "network error",
			resp:     Error(errors.NewNetworkError("connection failed")),
			expected: errors.ExitNetwork,
		},
		{
			name:     "invalid args",
			resp:     Error(errors.NewInvalidArgsError("missing flag")),
			expected: errors.ExitInvalidArgs,
		},
		{
			name:     "generic error",
			resp:     Error(errors.NewError("something went wrong")),
			expected: errors.ExitError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exitCode := tt.resp.ExitCode()
			if exitCode != tt.expected {
				t.Errorf("expected exit code %d, got %d", tt.expected, exitCode)
			}
		})
	}
}

func TestCreateMeta(t *testing.T) {
	meta := createMeta()

	if meta == nil {
		t.Fatal("expected meta to be set")
	}

	timestamp, ok := meta["timestamp"]
	if !ok {
		t.Fatal("expected timestamp to be set")
	}

	ts, ok := timestamp.(string)
	if !ok {
		t.Fatal("expected timestamp to be a string")
	}

	if len(ts) == 0 {
		t.Error("expected timestamp to be non-empty")
	}
}

func TestResponseOmitsEmptyFields(t *testing.T) {
	resp := Success(map[string]string{"key": "value"})
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	jsonStr := string(data)

	// Error should be omitted when nil
	if containsKey(jsonStr, "error") {
		t.Error("expected 'error' to be omitted from JSON")
	}

	// Location should be omitted when empty
	if containsKey(jsonStr, "location") {
		t.Error("expected 'location' to be omitted from JSON")
	}

	// Pagination should be omitted when nil
	if containsKey(jsonStr, "pagination") {
		t.Error("expected 'pagination' to be omitted from JSON")
	}
}

func containsKey(jsonStr, key string) bool {
	var m map[string]any
	json.Unmarshal([]byte(jsonStr), &m)
	_, ok := m[key]
	return ok
}

func TestPrintDoesNotEscapeHTML(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Create response with HTML content (like description_html or comment body.html)
	htmlContent := "<p>Hello <strong>World</strong></p>"
	resp := Success(map[string]string{"description_html": htmlContent})
	resp.Print()

	// Restore stdout and read captured output
	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify HTML is NOT escaped (the fix)
	// Before the fix, < would become \u003c and > would become \u003e
	if strings.Contains(output, `\u003c`) || strings.Contains(output, `\u003e`) {
		t.Errorf("HTML should not be escaped in output, got: %s", output)
	}

	// Verify the actual HTML tags are present
	if !strings.Contains(output, "<p>") || !strings.Contains(output, "<strong>") {
		t.Errorf("expected HTML tags to be preserved, got: %s", output)
	}
}

func TestNewBreadcrumb(t *testing.T) {
	bc := NewBreadcrumb("show", "fizzy card show 42", "View card details")

	if bc.Action != "show" {
		t.Errorf("expected Action 'show', got '%s'", bc.Action)
	}
	if bc.Cmd != "fizzy card show 42" {
		t.Errorf("expected Cmd 'fizzy card show 42', got '%s'", bc.Cmd)
	}
	if bc.Description != "View card details" {
		t.Errorf("expected Description 'View card details', got '%s'", bc.Description)
	}
}

func TestSuccessWithBreadcrumbs(t *testing.T) {
	data := map[string]string{"id": "42"}
	breadcrumbs := []Breadcrumb{
		NewBreadcrumb("show", "fizzy card show 42", "View card details"),
		NewBreadcrumb("comment", "fizzy comment create --card 42 --body \"text\"", "Add comment"),
	}

	resp := SuccessWithBreadcrumbs(data, "Card #42 created", breadcrumbs)

	if !resp.Success {
		t.Error("expected Success to be true")
	}
	if resp.Data == nil {
		t.Error("expected Data to be set")
	}
	if resp.Summary != "Card #42 created" {
		t.Errorf("expected Summary 'Card #42 created', got '%s'", resp.Summary)
	}
	if len(resp.Breadcrumbs) != 2 {
		t.Errorf("expected 2 breadcrumbs, got %d", len(resp.Breadcrumbs))
	}
	if resp.Breadcrumbs[0].Action != "show" {
		t.Errorf("expected first breadcrumb action 'show', got '%s'", resp.Breadcrumbs[0].Action)
	}
	if resp.Breadcrumbs[1].Action != "comment" {
		t.Errorf("expected second breadcrumb action 'comment', got '%s'", resp.Breadcrumbs[1].Action)
	}
	if resp.Meta == nil {
		t.Error("expected Meta to be set")
	}
}

func TestSuccessWithPaginationAndBreadcrumbs(t *testing.T) {
	data := []string{"item1", "item2"}
	breadcrumbs := []Breadcrumb{
		NewBreadcrumb("show", "fizzy card show <number>", "View card details"),
		NewBreadcrumb("next", "fizzy card list --page 2", "Next page"),
	}

	t.Run("with pagination and breadcrumbs", func(t *testing.T) {
		resp := SuccessWithPaginationAndBreadcrumbs(data, true, "https://example.com/page2", "10 cards", breadcrumbs)

		if !resp.Success {
			t.Error("expected Success to be true")
		}
		if resp.Pagination == nil {
			t.Fatal("expected Pagination to be set")
		}
		if !resp.Pagination.HasNext {
			t.Error("expected HasNext to be true")
		}
		if resp.Summary != "10 cards" {
			t.Errorf("expected Summary '10 cards', got '%s'", resp.Summary)
		}
		if len(resp.Breadcrumbs) != 2 {
			t.Errorf("expected 2 breadcrumbs, got %d", len(resp.Breadcrumbs))
		}
	})

	t.Run("without pagination but with breadcrumbs", func(t *testing.T) {
		resp := SuccessWithPaginationAndBreadcrumbs(data, false, "", "10 cards", breadcrumbs)

		if resp.Pagination != nil {
			t.Error("expected Pagination to be nil when no next page")
		}
		if len(resp.Breadcrumbs) != 2 {
			t.Errorf("expected 2 breadcrumbs, got %d", len(resp.Breadcrumbs))
		}
	})
}

func TestBreadcrumbsOmittedWhenEmpty(t *testing.T) {
	resp := Success(map[string]string{"key": "value"})
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	jsonStr := string(data)

	// Breadcrumbs should be omitted when nil/empty
	if containsKey(jsonStr, "breadcrumbs") {
		t.Error("expected 'breadcrumbs' to be omitted from JSON when empty")
	}
}

func TestBreadcrumbsIncludedWhenPresent(t *testing.T) {
	breadcrumbs := []Breadcrumb{
		NewBreadcrumb("show", "fizzy card show 42", "View card"),
	}
	resp := SuccessWithBreadcrumbs(map[string]string{"id": "42"}, "", breadcrumbs)
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	jsonStr := string(data)

	// Breadcrumbs should be present
	if !containsKey(jsonStr, "breadcrumbs") {
		t.Error("expected 'breadcrumbs' to be present in JSON")
	}

	// Verify the structure
	var parsed Response
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if len(parsed.Breadcrumbs) != 1 {
		t.Errorf("expected 1 breadcrumb after parsing, got %d", len(parsed.Breadcrumbs))
	}
	if parsed.Breadcrumbs[0].Action != "show" {
		t.Errorf("expected action 'show', got '%s'", parsed.Breadcrumbs[0].Action)
	}
	if parsed.Breadcrumbs[0].Cmd != "fizzy card show 42" {
		t.Errorf("expected cmd 'fizzy card show 42', got '%s'", parsed.Breadcrumbs[0].Cmd)
	}
	if parsed.Breadcrumbs[0].Description != "View card" {
		t.Errorf("expected description 'View card', got '%s'", parsed.Breadcrumbs[0].Description)
	}
}
