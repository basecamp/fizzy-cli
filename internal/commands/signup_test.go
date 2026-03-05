package commands

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/basecamp/fizzy-cli/internal/config"
	"gopkg.in/yaml.v3"
)

// newTestSignupServer creates an httptest server that simulates the Fizzy signup API.
// It tracks state across requests via cookies, mirroring the Rails signed cookie flow.
func newTestSignupServer(t *testing.T, opts testSignupServerOpts) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	// POST /session.json — request magic link
	mux.HandleFunc("POST /session.json", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:  "pending_authentication_token",
			Value: "signed-pending-token-value",
			Path:  "/",
		})
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Magic-Link-Code", "TEST01")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"pending_authentication_token": "raw-pending-token-value",
		})
	})

	// POST /session/magic_link.json — verify code
	mux.HandleFunc("POST /session/magic_link.json", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)

		code, _ := body["code"].(string)
		if code != "VALID1" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]any{
				"message": "Try another code.",
			})
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:  "session_token",
			Value: "signed-session-token-value",
			Path:  "/",
		})
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"session_token":              "raw-session-token-value",
			"requires_signup_completion": opts.requiresCompletion,
		})
	})

	// POST /signup/completion.json — complete signup for new users
	mux.HandleFunc("POST /signup/completion.json", func(w http.ResponseWriter, r *http.Request) {
		if !hasSessionCookie(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusCreated)
	})

	// GET /my/identity.json — fetch accounts
	mux.HandleFunc("GET /my/identity.json", func(w http.ResponseWriter, r *http.Request) {
		if !hasSessionCookie(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"accounts": []any{
				map[string]any{
					"id":   "acct-1",
					"name": "Test Account",
					"slug": "/123456",
				},
				map[string]any{
					"id":   "acct-2",
					"name": "Other Account",
					"slug": "/789012",
				},
			},
		})
	})

	// POST /{account}/my/access_tokens.json — generate access token
	mux.HandleFunc("POST /", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/my/access_tokens.json") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if !hasSessionCookie(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Verify the URL doesn't have double slashes (slug normalization)
		if strings.Contains(r.URL.Path, "//") {
			t.Errorf("URL contains double slash: %s", r.URL.Path)
		}

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"token":       opts.accessToken,
			"description": "Fizzy CLI",
			"permission":  "write",
		})
	})

	return httptest.NewServer(mux)
}

type testSignupServerOpts struct {
	requiresCompletion bool
	accessToken        string
}

func hasSessionCookie(r *http.Request) bool {
	cookie, err := r.Cookie("session_token")
	return err == nil && cookie.Value != ""
}

// pipeSessionToken replaces os.Stdin with a pipe containing the session token.
// Returns a cleanup function that restores the original stdin.
func pipeSessionToken(token string) func() {
	r, w, _ := os.Pipe()
	w.WriteString(token + "\n")
	w.Close()
	origStdin := os.Stdin
	os.Stdin = r
	return func() { os.Stdin = origStdin }
}

// resetSignupFlags clears cobra flag values between test runs to prevent cross-contamination.
func resetSignupFlags() {
	signupStartCmd.Flags().Set("email", "")
	signupVerifyCmd.Flags().Set("code", "")
	signupVerifyCmd.Flags().Set("pending-token", "")
	signupCompleteCmd.Flags().Set("name", "")
	signupCompleteCmd.Flags().Set("account", "")
}

func TestSignupStart(t *testing.T) {
	server := newTestSignupServer(t, testSignupServerOpts{accessToken: "fizzy_test"})
	defer server.Close()

	t.Run("returns signed cookie value not JSON body value", func(t *testing.T) {
		resetSignupFlags()
		mock := NewMockClient()
		result := SetTestMode(mock)
		SetTestConfig("", "", server.URL)
		defer ResetTestMode()

		signupStartCmd.Flags().Set("email", "test@example.com")
		err := signupStartCmd.RunE(signupStartCmd, []string{})
		assertExitCode(t, err, 0)

		data := result.Response.Data.(map[string]any)
		pendingToken := data["pending_authentication_token"].(string)

		if pendingToken != "signed-pending-token-value" {
			t.Errorf("expected signed cookie value 'signed-pending-token-value', got '%s'", pendingToken)
		}
		if pendingToken == "raw-pending-token-value" {
			t.Error("got raw JSON value instead of signed cookie value")
		}

		// Development servers include the magic link code
		code, _ := data["code"].(string)
		if code != "TEST01" {
			t.Errorf("expected code 'TEST01', got '%s'", code)
		}
	})
}

func TestSignupVerify(t *testing.T) {
	t.Run("returns signed session cookie and normalized account slugs", func(t *testing.T) {
		resetSignupFlags()
		server := newTestSignupServer(t, testSignupServerOpts{
			requiresCompletion: false,
			accessToken:        "fizzy_test",
		})
		defer server.Close()

		mock := NewMockClient()
		result := SetTestMode(mock)
		SetTestConfig("", "", server.URL)
		defer ResetTestMode()

		signupVerifyCmd.Flags().Set("code", "VALID1")
		signupVerifyCmd.Flags().Set("pending-token", "signed-pending-token-value")
		err := signupVerifyCmd.RunE(signupVerifyCmd, []string{})
		assertExitCode(t, err, 0)

		data := result.Response.Data.(map[string]any)

		// Session token should be signed cookie value
		sessionToken := data["session_token"].(string)
		if sessionToken != "signed-session-token-value" {
			t.Errorf("expected signed cookie value 'signed-session-token-value', got '%s'", sessionToken)
		}
		if sessionToken == "raw-session-token-value" {
			t.Error("got raw JSON value instead of signed cookie value")
		}

		// Accounts should be present for existing users
		accounts, ok := data["accounts"].([]any)
		if !ok {
			t.Fatal("expected accounts in response for existing user")
		}
		if len(accounts) != 2 {
			t.Fatalf("expected 2 accounts, got %d", len(accounts))
		}

		// Account slugs should be normalized (no leading /)
		firstAccount := accounts[0].(map[string]any)
		slug := firstAccount["slug"].(string)
		if strings.HasPrefix(slug, "/") {
			t.Errorf("account slug should not have leading /, got '%s'", slug)
		}
		if slug != "123456" {
			t.Errorf("expected slug '123456', got '%s'", slug)
		}
	})

	t.Run("returns requires_signup_completion for new users without accounts", func(t *testing.T) {
		resetSignupFlags()
		server := newTestSignupServer(t, testSignupServerOpts{
			requiresCompletion: true,
			accessToken:        "fizzy_test",
		})
		defer server.Close()

		mock := NewMockClient()
		result := SetTestMode(mock)
		SetTestConfig("", "", server.URL)
		defer ResetTestMode()

		signupVerifyCmd.Flags().Set("code", "VALID1")
		signupVerifyCmd.Flags().Set("pending-token", "signed-pending-token-value")
		err := signupVerifyCmd.RunE(signupVerifyCmd, []string{})
		assertExitCode(t, err, 0)

		data := result.Response.Data.(map[string]any)

		requiresCompletion, _ := data["requires_signup_completion"].(bool)
		if !requiresCompletion {
			t.Error("expected requires_signup_completion=true for new user")
		}

		// Should NOT have accounts for new users
		if _, ok := data["accounts"]; ok {
			t.Error("expected no accounts for new user requiring signup completion")
		}
	})

	t.Run("fails with invalid code", func(t *testing.T) {
		resetSignupFlags()
		server := newTestSignupServer(t, testSignupServerOpts{accessToken: "fizzy_test"})
		defer server.Close()

		mock := NewMockClient()
		SetTestMode(mock)
		SetTestConfig("", "", server.URL)
		defer ResetTestMode()

		signupVerifyCmd.Flags().Set("code", "BADCODE")
		signupVerifyCmd.Flags().Set("pending-token", "signed-pending-token-value")
		err := signupVerifyCmd.RunE(signupVerifyCmd, []string{})

		if err == nil {
			t.Error("expected error for invalid code")
		}
	})
}

func TestSignupComplete(t *testing.T) {
	t.Run("existing user generates token and saves config", func(t *testing.T) {
		resetSignupFlags()
		server := newTestSignupServer(t, testSignupServerOpts{
			accessToken: "fizzy_generated_token",
		})
		defer server.Close()

		tempDir, _ := os.MkdirTemp("", "fizzy-test-*")
		defer os.RemoveAll(tempDir)
		config.SetTestConfigDir(tempDir)
		defer config.ResetTestConfigDir()

		mock := NewMockClient()
		result := SetTestMode(mock)
		SetTestConfig("", "", server.URL)
		defer ResetTestMode()

		restoreStdin := pipeSessionToken("signed-session-token-value")
		defer restoreStdin()

		signupCompleteCmd.Flags().Set("account", "123456")
		err := signupCompleteCmd.RunE(signupCompleteCmd, []string{})
		assertExitCode(t, err, 0)

		data := result.Response.Data.(map[string]any)

		token := data["token"].(string)
		if token != "fizzy_generated_token" {
			t.Errorf("expected token 'fizzy_generated_token', got '%s'", token)
		}

		account := data["account"].(string)
		if account != "123456" {
			t.Errorf("expected account '123456', got '%s'", account)
		}

		// Verify config was saved
		configPath := filepath.Join(tempDir, "config.yaml")
		configData, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("config file not created: %v", err)
		}

		var savedConfig config.Config
		yaml.Unmarshal(configData, &savedConfig)

		if savedConfig.Token != "fizzy_generated_token" {
			t.Errorf("expected saved token 'fizzy_generated_token', got '%s'", savedConfig.Token)
		}
		if savedConfig.Account != "123456" {
			t.Errorf("expected saved account '123456', got '%s'", savedConfig.Account)
		}
	})

	t.Run("strips leading slash from account flag", func(t *testing.T) {
		resetSignupFlags()
		server := newTestSignupServer(t, testSignupServerOpts{
			accessToken: "fizzy_test_token",
		})
		defer server.Close()

		tempDir, _ := os.MkdirTemp("", "fizzy-test-*")
		defer os.RemoveAll(tempDir)
		config.SetTestConfigDir(tempDir)
		defer config.ResetTestConfigDir()

		mock := NewMockClient()
		result := SetTestMode(mock)
		SetTestConfig("", "", server.URL)
		defer ResetTestMode()

		restoreStdin := pipeSessionToken("signed-session-token-value")
		defer restoreStdin()

		signupCompleteCmd.Flags().Set("account", "/123456")
		err := signupCompleteCmd.RunE(signupCompleteCmd, []string{})
		assertExitCode(t, err, 0)

		data := result.Response.Data.(map[string]any)
		if data["account"] != "123456" {
			t.Errorf("expected normalized account '123456', got '%s'", data["account"])
		}
	})

	t.Run("rejects both name and account flags", func(t *testing.T) {
		resetSignupFlags()
		server := newTestSignupServer(t, testSignupServerOpts{accessToken: "fizzy_test"})
		defer server.Close()

		mock := NewMockClient()
		SetTestMode(mock)
		SetTestConfig("", "", server.URL)
		defer ResetTestMode()

		restoreStdin := pipeSessionToken("signed-session-token-value")
		defer restoreStdin()

		signupCompleteCmd.Flags().Set("name", "Test User")
		signupCompleteCmd.Flags().Set("account", "123456")
		err := signupCompleteCmd.RunE(signupCompleteCmd, []string{})

		if err == nil {
			t.Error("expected error when both --name and --account provided")
		}
	})

	t.Run("rejects neither name nor account flags", func(t *testing.T) {
		resetSignupFlags()
		server := newTestSignupServer(t, testSignupServerOpts{accessToken: "fizzy_test"})
		defer server.Close()

		mock := NewMockClient()
		SetTestMode(mock)
		SetTestConfig("", "", server.URL)
		defer ResetTestMode()

		restoreStdin := pipeSessionToken("signed-session-token-value")
		defer restoreStdin()

		err := signupCompleteCmd.RunE(signupCompleteCmd, []string{})

		if err == nil {
			t.Error("expected error when neither --name nor --account provided")
		}
	})

	t.Run("new user completes signup and generates token", func(t *testing.T) {
		resetSignupFlags()
		server := newTestSignupServer(t, testSignupServerOpts{
			requiresCompletion: true,
			accessToken:        "fizzy_new_user_token",
		})
		defer server.Close()

		tempDir, _ := os.MkdirTemp("", "fizzy-test-*")
		defer os.RemoveAll(tempDir)
		config.SetTestConfigDir(tempDir)
		defer config.ResetTestConfigDir()

		mock := NewMockClient()
		result := SetTestMode(mock)
		SetTestConfig("", "", server.URL)
		defer ResetTestMode()

		restoreStdin := pipeSessionToken("signed-session-token-value")
		defer restoreStdin()

		signupCompleteCmd.Flags().Set("name", "New User")
		err := signupCompleteCmd.RunE(signupCompleteCmd, []string{})
		assertExitCode(t, err, 0)

		data := result.Response.Data.(map[string]any)

		if data["token"] != "fizzy_new_user_token" {
			t.Errorf("expected token 'fizzy_new_user_token', got '%s'", data["token"])
		}
		if data["account"] != "123456" {
			t.Errorf("expected account '123456', got '%s'", data["account"])
		}
	})
}

func TestSignupAPIURL(t *testing.T) {
	t.Run("trims trailing slash", func(t *testing.T) {
		SetTestConfig("", "", "https://example.com/")
		defer ResetTestMode()

		result := signupAPIURL()
		if result != "https://example.com" {
			t.Errorf("expected 'https://example.com', got '%s'", result)
		}
	})

	t.Run("returns default when no config", func(t *testing.T) {
		cfg = nil
		result := signupAPIURL()
		if result != strings.TrimSuffix(config.DefaultAPIURL, "/") {
			t.Errorf("expected default API URL, got '%s'", result)
		}
	})
}

func TestNormalizeAccountSlugs(t *testing.T) {
	t.Run("strips leading slash from slugs", func(t *testing.T) {
		input := []any{
			map[string]any{
				"name": "Account 1",
				"slug": "/123456",
			},
			map[string]any{
				"name": "Account 2",
				"slug": "/789012",
			},
		}

		result := normalizeAccountSlugs(input)
		accounts := result.([]any)

		slug1 := accounts[0].(map[string]any)["slug"].(string)
		slug2 := accounts[1].(map[string]any)["slug"].(string)

		if slug1 != "123456" {
			t.Errorf("expected '123456', got '%s'", slug1)
		}
		if slug2 != "789012" {
			t.Errorf("expected '789012', got '%s'", slug2)
		}
	})

	t.Run("handles slugs without leading slash", func(t *testing.T) {
		input := []any{
			map[string]any{
				"slug": "already-clean",
			},
		}

		result := normalizeAccountSlugs(input)
		accounts := result.([]any)
		slug := accounts[0].(map[string]any)["slug"].(string)

		if slug != "already-clean" {
			t.Errorf("expected 'already-clean', got '%s'", slug)
		}
	})

	t.Run("returns non-array input unchanged", func(t *testing.T) {
		input := "not-an-array"
		result := normalizeAccountSlugs(input)
		if result != "not-an-array" {
			t.Errorf("expected unchanged input, got '%v'", result)
		}
	})
}

func TestGetCookieValue(t *testing.T) {
	t.Run("returns empty string for missing cookie", func(t *testing.T) {
		client := newSignupHTTPClient()
		result := getCookieValue(client, "https://example.com", "nonexistent")
		if result != "" {
			t.Errorf("expected empty string, got '%s'", result)
		}
	})
}

func TestReadSessionTokenFromStdin(t *testing.T) {
	t.Run("reads token from piped stdin", func(t *testing.T) {
		restoreStdin := pipeSessionToken("eyJ-session-token-value")
		defer restoreStdin()

		token, err := readSessionToken()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token != "eyJ-session-token-value" {
			t.Errorf("expected 'eyJ-session-token-value', got '%s'", token)
		}
	})

	t.Run("trims whitespace from piped input", func(t *testing.T) {
		restoreStdin := pipeSessionToken("  eyJ-token  ")
		defer restoreStdin()

		token, err := readSessionToken()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token != "eyJ-token" {
			t.Errorf("expected 'eyJ-token', got '%s'", token)
		}
	})

	t.Run("returns empty string for empty stdin", func(t *testing.T) {
		r, w, _ := os.Pipe()
		w.Close()

		origStdin := os.Stdin
		os.Stdin = r
		defer func() { os.Stdin = origStdin }()

		token, err := readSessionToken()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token != "" {
			t.Errorf("expected empty string, got '%s'", token)
		}
	})
}
