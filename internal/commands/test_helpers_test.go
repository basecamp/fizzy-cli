package commands

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/basecamp/cli/output"
	"github.com/basecamp/fizzy-cli/internal/client"
	"github.com/spf13/cobra"
)

// testHTTPServer holds the current httptest server for cleanup.
var testHTTPServer *httptest.Server

// SetTestModeWithSDK configures both the old mock client and a new httptest-backed SDK.
// The httptest server delegates to the MockClient so existing tests work unchanged.
// Call resetTest() (or defer resetTest()) to clean up.
func SetTestModeWithSDK(mock *MockClient) *CommandResult {
	// Set up old client for commands that still use getClient()
	result := SetTestMode(mock)

	// Create httptest server that delegates to the mock
	testHTTPServer = httptest.NewServer(mockHandler(mock))

	// Wire up SDK to point at the httptest server
	SetTestSDK(testHTTPServer.URL)

	// Set context on all commands so cmd.Context() doesn't return nil
	// when tests call RunE directly instead of through cobra Execute
	setContextOnAll(context.Background(), rootCmd)

	return result
}

// setContextOnAll recursively sets context on a command and all its children.
func setContextOnAll(ctx context.Context, cmd *cobra.Command) {
	cmd.SetContext(ctx)
	for _, child := range cmd.Commands() {
		setContextOnAll(ctx, child)
	}
}

// resetTest cleans up both the httptest server and test mode state.
func resetTest() {
	if testHTTPServer != nil {
		testHTTPServer.Close()
		testHTTPServer = nil
	}
	ResetTestMode()
}

// mockHandler creates an http.Handler that delegates to a MockClient.
func mockHandler(mock *MockClient) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Strip account prefix for account-scoped SDK paths: /<account>/path -> /path
		path := r.URL.Path
		if parts := strings.Split(strings.TrimPrefix(path, "/"), "/"); len(parts) > 1 {
			first := parts[0]
			if first != "my" && first != "signup" {
				path = "/" + strings.Join(parts[1:], "/")
			}
		}

		// Re-append query string
		if r.URL.RawQuery != "" {
			path += "?" + r.URL.RawQuery
		}

		var resp *client.APIResponse
		var err error

		switch r.Method {
		case "GET":
			// Record as a GetWithPagination call since many tests assert on that.
			mock.GetWithPaginationCalls = append(mock.GetWithPaginationCalls, MockCall{Path: path})
			// Use mock.Get which checks path-based responses, then falls back to GetResponse.
			resp, err = mock.Get(path)
			// When GetResponse is still the default empty map (test didn't override it),
			// fall back to GetWithPaginationResponse if available. This bridges the
			// old test pattern where list data was set on GetWithPaginationResponse.
			if err == nil && len(mock.getPathResponses) == 0 && isDefaultGetResponse(resp) {
				if mock.GetWithPaginationError != nil {
					resp = nil
					err = mock.GetWithPaginationError
				} else if mock.GetWithPaginationResponse != nil {
					resp = mock.GetWithPaginationResponse
				}
			}
		case "POST":
			var body any
			if r.Body != nil {
				data, _ := io.ReadAll(r.Body)
				if len(data) > 0 {
					_ = json.Unmarshal(data, &body)
				}
			}
			resp, err = mock.Post(path, body)
		case "PATCH":
			var body any
			if r.Body != nil {
				data, _ := io.ReadAll(r.Body)
				if len(data) > 0 {
					_ = json.Unmarshal(data, &body)
				}
			}
			resp, err = mock.Patch(path, body)
		case "PUT":
			var body any
			if r.Body != nil {
				data, _ := io.ReadAll(r.Body)
				if len(data) > 0 {
					_ = json.Unmarshal(data, &body)
				}
			}
			resp, err = mock.Put(path, body)
		case "DELETE":
			resp, err = mock.Delete(path)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err != nil {
			statusCode := http.StatusInternalServerError
			errMsg := err.Error()
			// Use HTTPStatus from output.Error if available
			var outputErr *output.Error
			if stderrors.As(err, &outputErr) && outputErr.HTTPStatus > 0 {
				statusCode = outputErr.HTTPStatus
			} else if strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "Not found") {
				statusCode = http.StatusNotFound
			} else if strings.Contains(strings.ToLower(errMsg), "unauthorized") {
				statusCode = http.StatusUnauthorized
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(statusCode)
			errResp := map[string]string{"error": errMsg}
			_ = json.NewEncoder(w).Encode(errResp)
			return
		}

		if resp == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Set Location header if present
		if resp.Location != "" {
			w.Header().Set("Location", resp.Location)
		}
		// Set Link header if present
		if resp.LinkNext != "" {
			w.Header().Set("Link", `<`+resp.LinkNext+`>; rel="next"`)
		}

		statusCode := resp.StatusCode
		if statusCode == 0 {
			statusCode = 200
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)

		if resp.Data != nil {
			_ = json.NewEncoder(w).Encode(resp.Data)
		} else {
			_, _ = w.Write([]byte("null"))
		}
	})
}

// isDefaultGetResponse checks if a response looks like the NewMockClient default
// GetResponse (status 200 with an empty map). Tests that explicitly set GetResponse
// with real data will not match this check.
func isDefaultGetResponse(resp *client.APIResponse) bool {
	if resp == nil {
		return false
	}
	m, ok := resp.Data.(map[string]any)
	return ok && len(m) == 0
}
