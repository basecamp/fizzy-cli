package client

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/basecamp/fizzy-cli/internal/errors"
)

func TestNew(t *testing.T) {
	c := New("https://api.example.com", "token123", "account456")

	if c.BaseURL != "https://api.example.com" {
		t.Errorf("expected BaseURL 'https://api.example.com', got '%s'", c.BaseURL)
	}
	if c.Token != "token123" {
		t.Errorf("expected Token 'token123', got '%s'", c.Token)
	}
	if c.Account != "account456" {
		t.Errorf("expected Account 'account456', got '%s'", c.Account)
	}
	if c.HTTPClient == nil {
		t.Error("expected HTTPClient to be set")
	}
}

func TestNew_TrimsTrailingSlash(t *testing.T) {
	c := New("https://api.example.com/", "token", "account")

	if c.BaseURL != "https://api.example.com" {
		t.Errorf("expected BaseURL without trailing slash, got '%s'", c.BaseURL)
	}
}

func TestBuildURL(t *testing.T) {
	c := New("https://api.example.com", "token", "account123")

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "simple path",
			path:     "/boards.json",
			expected: "https://api.example.com/account123/boards.json",
		},
		{
			name:     "path without leading slash",
			path:     "boards.json",
			expected: "https://api.example.com/account123/boards.json",
		},
		{
			name:     "full URL",
			path:     "https://other.api.com/resource",
			expected: "https://other.api.com/resource",
		},
		{
			name:     "path with account already",
			path:     "/account123/boards.json",
			expected: "https://api.example.com/account123/boards.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.buildURL(tt.path)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestBuildURL_NoAccount(t *testing.T) {
	c := New("https://api.example.com", "token", "")

	result := c.buildURL("/boards.json")
	if result != "https://api.example.com/boards.json" {
		t.Errorf("expected path without account, got '%s'", result)
	}
}

func TestGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Authorization header, got '%s'", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("expected Accept header, got '%s'", r.Header.Get("Accept"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": "123", "name": "Test"})
	}))
	defer server.Close()

	c := New(server.URL, "test-token", "")
	resp, err := c.Get("/resource.json")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected map response data")
	}
	if data["id"] != "123" {
		t.Errorf("expected id '123', got '%v'", data["id"])
	}
}

func TestPost(t *testing.T) {
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type header, got '%s'", r.Header.Get("Content-Type"))
		}

		body, _ := io.ReadAll(r.Body)
		var data map[string]string
		json.Unmarshal(body, &data)

		if data["name"] != "New Resource" {
			t.Errorf("expected name 'New Resource', got '%s'", data["name"])
		}

		w.Header().Set("Location", serverURL+"/resource/456")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"id": "456"})
	}))
	defer server.Close()
	serverURL = server.URL

	c := New(server.URL, "test-token", "")
	resp, err := c.Post("/resources.json", map[string]string{"name": "New Resource"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}
	if resp.Location == "" {
		t.Error("expected Location header to be set")
	}
}

func TestPatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("expected PATCH, got %s", r.Method)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"id": "123", "name": "Updated"})
	}))
	defer server.Close()

	c := New(server.URL, "test-token", "")
	resp, err := c.Patch("/resources/123.json", map[string]string{"name": "Updated"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestPut(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got %s", r.Method)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
	}))
	defer server.Close()

	c := New(server.URL, "test-token", "")
	resp, err := c.Put("/resources/123.json", map[string]string{"status": "active"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	c := New(server.URL, "test-token", "")
	resp, err := c.Delete("/resources/123.json")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", resp.StatusCode)
	}
}

func TestErrorResponses(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		responseBody string
		expectedCode string
		expectedExit int
	}{
		{
			name:         "401 Unauthorized",
			statusCode:   401,
			responseBody: `{"error": "Invalid token"}`,
			expectedCode: "auth_required",
			expectedExit: errors.ExitAuth,
		},
		{
			name:         "403 Forbidden",
			statusCode:   403,
			responseBody: `{"error": "Access denied"}`,
			expectedCode: "forbidden",
			expectedExit: errors.ExitForbidden,
		},
		{
			name:         "404 Not Found",
			statusCode:   404,
			responseBody: `{"error": "Resource not found"}`,
			expectedCode: "not_found",
			expectedExit: errors.ExitNotFound,
		},
		{
			name:         "422 Validation Error",
			statusCode:   422,
			responseBody: `{"error": "Validation failed"}`,
			expectedCode: "api_error",
			expectedExit: errors.ExitAPI,
		},
		{
			name:         "500 Server Error",
			statusCode:   500,
			responseBody: `{"error": "Internal server error"}`,
			expectedCode: "api_error",
			expectedExit: errors.ExitAPI,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			c := New(server.URL, "test-token", "")
			c.Sleeper = func(d time.Duration) {} // no-op for fast tests
			_, err := c.Get("/resource.json")

			if err == nil {
				t.Fatal("expected error, got nil")
			}

			cliErr, ok := err.(*errors.CLIError)
			if !ok {
				t.Fatalf("expected CLIError, got %T", err)
			}

			if cliErr.Code != tt.expectedCode {
				t.Errorf("expected code '%s', got '%s'", tt.expectedCode, cliErr.Code)
			}
			if cliErr.ExitCode() != tt.expectedExit {
				t.Errorf("expected exit code %d, got %d", tt.expectedExit, cliErr.ExitCode())
			}
		})
	}
}

func TestErrorResponse_NoBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer server.Close()

	c := New(server.URL, "test-token", "")
	_, err := c.Get("/resource.json")

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	cliErr, ok := err.(*errors.CLIError)
	if !ok {
		t.Fatalf("expected CLIError, got %T", err)
	}

	// Should use HTTP status text as message
	if cliErr.Message != "Not Found" {
		t.Errorf("expected message 'Not Found', got '%s'", cliErr.Message)
	}
}

func TestParseLinkNext(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected string
	}{
		{
			name:     "empty header",
			header:   "",
			expected: "",
		},
		{
			name:     "next link",
			header:   `<https://api.example.com/page2>; rel="next"`,
			expected: "https://api.example.com/page2",
		},
		{
			name:     "next link with other links",
			header:   `<https://api.example.com/page1>; rel="prev", <https://api.example.com/page3>; rel="next"`,
			expected: "https://api.example.com/page3",
		},
		{
			name:     "no next link",
			header:   `<https://api.example.com/page1>; rel="prev"`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseLinkNext(tt.header)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestGetWithPagination(t *testing.T) {
	t.Run("single page", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode([]map[string]string{{"id": "1"}, {"id": "2"}})
		}))
		defer server.Close()

		c := New(server.URL, "test-token", "")
		resp, err := c.GetWithPagination("/resources.json", false)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		data, ok := resp.Data.([]any)
		if !ok {
			t.Fatalf("expected array response data")
		}
		if len(data) != 2 {
			t.Errorf("expected 2 items, got %d", len(data))
		}
	})

	t.Run("fetch all pages", func(t *testing.T) {
		page := 1
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if page == 1 {
				w.Header().Set("Link", `<`+r.Host+`/resources.json?page=2>; rel="next"`)
				json.NewEncoder(w).Encode([]map[string]string{{"id": "1"}})
				page++
			} else {
				json.NewEncoder(w).Encode([]map[string]string{{"id": "2"}})
			}
		}))
		defer server.Close()

		c := New(server.URL, "test-token", "")
		resp, err := c.GetWithPagination("/resources.json", true)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		data, ok := resp.Data.([]any)
		if !ok {
			t.Fatalf("expected array response data")
		}
		if len(data) != 2 {
			t.Errorf("expected 2 items from both pages, got %d", len(data))
		}
	})
}

func TestFollowLocation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"id": "123", "name": "Created Resource"})
	}))
	defer server.Close()

	c := New(server.URL, "test-token", "")

	t.Run("with location", func(t *testing.T) {
		resp, err := c.FollowLocation(server.URL + "/resource/123.json")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected response, got nil")
		}
	})

	t.Run("empty location", func(t *testing.T) {
		resp, err := c.FollowLocation("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp != nil {
			t.Error("expected nil response for empty location")
		}
	})
}

func TestDetectContentType(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"image.png", "image/png"},
		{"image.PNG", "image/png"},
		{"photo.jpg", "image/jpeg"},
		{"photo.jpeg", "image/jpeg"},
		{"animation.gif", "image/gif"},
		{"image.webp", "image/webp"},
		{"icon.svg", "image/svg+xml"},
		{"document.pdf", "application/pdf"},
		{"readme.txt", "text/plain"},
		{"page.html", "text/html"},
		{"data.json", "application/json"},
		{"config.xml", "application/xml"},
		{"archive.zip", "application/zip"},
		{"unknown.xyz", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := detectContentType(tt.filename)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestComputeChecksum(t *testing.T) {
	content := []byte("test content")
	checksum := computeChecksum(content)

	if checksum == "" {
		t.Error("expected non-empty checksum")
	}

	// Checksum should be base64 encoded
	if len(checksum) < 20 {
		t.Errorf("checksum seems too short: '%s'", checksum)
	}

	// Same content should produce same checksum
	checksum2 := computeChecksum(content)
	if checksum != checksum2 {
		t.Error("expected same checksum for same content")
	}

	// Different content should produce different checksum
	checksum3 := computeChecksum([]byte("different content"))
	if checksum == checksum3 {
		t.Error("expected different checksum for different content")
	}
}

func TestParsePage(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "empty url",
			url:      "",
			expected: "",
		},
		{
			name:     "url with page",
			url:      "https://api.example.com/resources?page=2",
			expected: "2",
		},
		{
			name:     "url without page",
			url:      "https://api.example.com/resources",
			expected: "",
		},
		{
			name:     "url with multiple params",
			url:      "https://api.example.com/resources?status=active&page=5",
			expected: "5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParsePage(tt.url)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestUploadFile(t *testing.T) {
	// Create a temp file
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test.txt")
	os.WriteFile(tempFile, []byte("test file content"), 0644)

	uploadCalled := false
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/rails/active_storage/direct_uploads" {
			// Blob creation request
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"signed_id": "test-signed-id-123",
				"direct_upload": map[string]any{
					"url": serverURL + "/upload",
					"headers": map[string]string{
						"Content-Type": "text/plain",
					},
				},
			})
		} else if r.Method == "PUT" && r.URL.Path == "/upload" {
			// Direct upload request
			uploadCalled = true
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	c := New(server.URL, "test-token", "")
	resp, err := c.UploadFile(tempFile)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !uploadCalled {
		t.Error("expected upload endpoint to be called")
	}

	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatal("expected map response data")
	}

	if data["signed_id"] != "test-signed-id-123" {
		t.Errorf("expected signed_id 'test-signed-id-123', got '%v'", data["signed_id"])
	}
}

func TestUploadFile_FileNotFound(t *testing.T) {
	c := New("https://api.example.com", "token", "account")
	_, err := c.UploadFile("/nonexistent/file.txt")

	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestNetworkError(t *testing.T) {
	c := New("http://localhost:1", "token", "") // Invalid port
	c.Sleeper = func(d time.Duration) {}        // no-op for fast tests
	_, err := c.Get("/resource.json")

	if err == nil {
		t.Fatal("expected network error")
	}

	cliErr, ok := err.(*errors.CLIError)
	if !ok {
		t.Fatalf("expected CLIError, got %T", err)
	}

	if cliErr.Code != "network" {
		t.Errorf("expected code 'NETWORK_ERROR', got '%s'", cliErr.Code)
	}
	if cliErr.ExitCode() != errors.ExitNetwork {
		t.Errorf("expected exit code %d, got %d", errors.ExitNetwork, cliErr.ExitCode())
	}
}

func TestVerboseMode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	c := New(server.URL, "test-token", "")
	c.Verbose = true

	// Capture stderr - this is just to verify it doesn't panic
	// In a real test we'd capture and verify the output
	_, err := c.Get("/resource.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUserAgentHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "fizzy-cli/1.0" {
			t.Errorf("expected User-Agent 'fizzy-cli/1.0', got '%s'", r.Header.Get("User-Agent"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := New(server.URL, "test-token", "")
	c.Get("/resource.json")
}

func TestRetryOn429(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts <= 2 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(429)
			w.Write([]byte(`{"error":"rate limited"}`))
			return
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]string{"id": "1"})
	}))
	defer server.Close()

	var sleepCalls []time.Duration
	c := New(server.URL, "test-token", "")
	c.Sleeper = func(d time.Duration) { sleepCalls = append(sleepCalls, d) }

	resp, err := c.Get("/resource.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
	if len(sleepCalls) != 2 {
		t.Errorf("expected 2 sleep calls, got %d", len(sleepCalls))
	}
	if sleepCalls[0] != time.Second {
		t.Errorf("expected 1s retry delay from Retry-After header, got %v", sleepCalls[0])
	}
}

func TestRetryOn5xx(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts <= 1 {
			w.WriteHeader(503)
			w.Write([]byte(`{"error":"unavailable"}`))
			return
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]string{"id": "1"})
	}))
	defer server.Close()

	var sleepCalls []time.Duration
	c := New(server.URL, "test-token", "")
	c.Sleeper = func(d time.Duration) { sleepCalls = append(sleepCalls, d) }

	resp, err := c.Get("/resource.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
	if len(sleepCalls) != 1 {
		t.Errorf("expected 1 sleep call, got %d", len(sleepCalls))
	}
}

func TestNoRetryPostOn5xx(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(500)
		w.Write([]byte(`{"error":"server error"}`))
	}))
	defer server.Close()

	c := New(server.URL, "test-token", "")
	c.Sleeper = func(d time.Duration) {}

	_, err := c.Post("/resource.json", map[string]string{"name": "test"})
	if err == nil {
		t.Fatal("expected error")
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt (no retry for POST on 5xx), got %d", attempts)
	}
}

func TestRetryPostOn429(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts <= 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(429)
			w.Write([]byte(`{"error":"rate limited"}`))
			return
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]string{"id": "1"})
	}))
	defer server.Close()

	c := New(server.URL, "test-token", "")
	c.Sleeper = func(d time.Duration) {}

	resp, err := c.Post("/resource.json", map[string]string{"name": "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts (retry POST on 429), got %d", attempts)
	}
}

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected time.Duration
	}{
		{"empty", "", time.Second},
		{"seconds", "5", 5 * time.Second},
		{"invalid", "not-a-number", time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := parseRetryAfter(tt.value)
			if d != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, d)
			}
		})
	}
}

func TestDownloadFile(t *testing.T) {
	t.Run("successful download", func(t *testing.T) {
		fileContent := []byte("test file content for download")
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				t.Errorf("expected GET, got %s", r.Method)
			}
			if r.Header.Get("Authorization") != "Bearer test-token" {
				t.Errorf("expected Authorization header, got '%s'", r.Header.Get("Authorization"))
			}
			if r.Header.Get("User-Agent") != "fizzy-cli/1.0" {
				t.Errorf("expected User-Agent header, got '%s'", r.Header.Get("User-Agent"))
			}
			w.WriteHeader(http.StatusOK)
			w.Write(fileContent)
		}))
		defer server.Close()

		tempDir := t.TempDir()
		destPath := filepath.Join(tempDir, "downloaded.txt")

		c := New(server.URL, "test-token", "")
		err := c.DownloadFile("/files/test.txt", destPath)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify file was created with correct content
		content, err := os.ReadFile(destPath)
		if err != nil {
			t.Fatalf("failed to read downloaded file: %v", err)
		}
		if string(content) != string(fileContent) {
			t.Errorf("expected content '%s', got '%s'", fileContent, content)
		}
	})

	t.Run("download with redirect", func(t *testing.T) {
		fileContent := []byte("redirected content")
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/redirect" {
				http.Redirect(w, r, "/final", http.StatusFound)
				return
			}
			w.Write(fileContent)
		}))
		defer server.Close()

		tempDir := t.TempDir()
		destPath := filepath.Join(tempDir, "redirected.txt")

		c := New(server.URL, "test-token", "")
		err := c.DownloadFile("/redirect", destPath)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := os.ReadFile(destPath)
		if err != nil {
			t.Fatalf("failed to read downloaded file: %v", err)
		}
		if string(content) != string(fileContent) {
			t.Errorf("expected content '%s', got '%s'", fileContent, content)
		}
	})

	t.Run("404 error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("file not found"))
		}))
		defer server.Close()

		tempDir := t.TempDir()
		destPath := filepath.Join(tempDir, "notfound.txt")

		c := New(server.URL, "test-token", "")
		err := c.DownloadFile("/missing.txt", destPath)

		if err == nil {
			t.Fatal("expected error for 404 response")
		}

		cliErr, ok := err.(*errors.CLIError)
		if !ok {
			t.Fatalf("expected CLIError, got %T", err)
		}
		if cliErr.Code != "api_error" {
			t.Errorf("expected code 'ERROR', got '%s'", cliErr.Code)
		}

		// Verify file was not created
		if _, err := os.Stat(destPath); !os.IsNotExist(err) {
			t.Error("expected file to not exist after 404 error")
		}
	})

	t.Run("network error", func(t *testing.T) {
		tempDir := t.TempDir()
		destPath := filepath.Join(tempDir, "network-error.txt")

		c := New("http://localhost:1", "test-token", "") // Invalid port
		c.Sleeper = func(d time.Duration) {}             // no-op for fast tests
		err := c.DownloadFile("/file.txt", destPath)

		if err == nil {
			t.Fatal("expected network error")
		}

		cliErr, ok := err.(*errors.CLIError)
		if !ok {
			t.Fatalf("expected CLIError, got %T", err)
		}
		if cliErr.Code != "network" {
			t.Errorf("expected code 'NETWORK_ERROR', got '%s'", cliErr.Code)
		}
	})

	t.Run("invalid destination path", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("content"))
		}))
		defer server.Close()

		c := New(server.URL, "test-token", "")
		err := c.DownloadFile("/file.txt", "/nonexistent/directory/file.txt")

		if err == nil {
			t.Fatal("expected error for invalid destination path")
		}

		cliErr, ok := err.(*errors.CLIError)
		if !ok {
			t.Fatalf("expected CLIError, got %T", err)
		}
		if cliErr.Code != "api_error" {
			t.Errorf("expected code 'ERROR', got '%s'", cliErr.Code)
		}
	})

	t.Run("verbose mode", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("content"))
		}))
		defer server.Close()

		tempDir := t.TempDir()
		destPath := filepath.Join(tempDir, "verbose.txt")

		c := New(server.URL, "test-token", "")
		c.Verbose = true

		// Just verify it doesn't panic with verbose mode
		err := c.DownloadFile("/file.txt", destPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
