// Package harness provides a test harness for end-to-end testing of the Fizzy CLI by
// executing the CLI binary and capturing stdout, stderr, and exit codes.
package harness

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Harness provides methods for executing CLI commands and capturing results.
type Harness struct {
	// BinaryPath is the path to the CLI binary (Ruby or Go)
	BinaryPath string

	// Token is the API access token
	Token string

	// Account is the account slug
	Account string

	// APIURL is the API base URL
	APIURL string

	// Cleanup tracks created resources for cleanup
	Cleanup *CleanupTracker

	// configHome is a temporary directory used as HOME to isolate from host config
	configHome string

	// t is the testing context
	t *testing.T
}

// Response represents the JSON response envelope from the CLI.
// Handles both success and error envelopes in a single struct.
type Response struct {
	OK          bool                   `json:"ok"`
	Data        any                    `json:"data,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Code        string                 `json:"code,omitempty"`
	Hint        string                 `json:"hint,omitempty"`
	Summary     string                 `json:"summary,omitempty"`
	Notice      string                 `json:"notice,omitempty"`
	Breadcrumbs []Breadcrumb           `json:"breadcrumbs,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Meta        map[string]any         `json:"meta,omitempty"`
}

// Breadcrumb represents a suggested next action.
type Breadcrumb struct {
	Action      string `json:"action"`
	Cmd         string `json:"cmd"`
	Description string `json:"description"`
}

// Result contains the output from a CLI command execution.
type Result struct {
	// Stdout is the standard output
	Stdout string

	// Stderr is the standard error output
	Stderr string

	// ExitCode is the process exit code
	ExitCode int

	// Response is the parsed JSON response (nil if parsing failed)
	Response *Response

	// ParseError is set if JSON parsing failed
	ParseError error
}

// Config holds test harness configuration from environment variables.
type Config struct {
	BinaryPath string
	Token      string
	Account    string
	APIURL     string
	UserID     string
}

// Exit codes aligned to the shared rubric.
const (
	ExitSuccess   = 0
	ExitUsage     = 1
	ExitNotFound  = 2
	ExitAuth      = 3
	ExitForbidden = 4
	ExitRateLimit = 5
	ExitNetwork   = 6
	ExitAPI       = 7
	ExitAmbiguous = 8

	// Deprecated aliases — kept for compilation.
	ExitError       = ExitAPI
	ExitInvalidArgs = ExitUsage
	ExitAuthFailure = ExitAuth
	ExitValidation  = ExitAPI
)

// LoadConfig loads test configuration from environment variables.
func LoadConfig() *Config {
	repoRoot, _ := RepoRoot()
	defaultBinary := "./bin/fizzy"
	if repoRoot != "" {
		defaultBinary = filepath.Join(repoRoot, "bin", "fizzy")
	}

	return &Config{
		BinaryPath: getEnvOrDefault("FIZZY_TEST_BINARY", defaultBinary),
		Token:      os.Getenv("FIZZY_TEST_TOKEN"),
		Account:    os.Getenv("FIZZY_TEST_ACCOUNT"),
		APIURL:     getEnvOrDefault("FIZZY_TEST_API_URL", "https://app.fizzy.do"),
		UserID:     os.Getenv("FIZZY_TEST_USER_ID"),
	}
}

// MissingVars returns names of any required env vars that are not set.
func (c *Config) MissingVars() []string {
	var missing []string
	if c.Token == "" {
		missing = append(missing, "FIZZY_TEST_TOKEN")
	}
	if c.Account == "" {
		missing = append(missing, "FIZZY_TEST_ACCOUNT")
	}
	return missing
}

func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// New creates a new test harness with configuration from environment variables.
func New(t *testing.T) *Harness {
	t.Helper()

	cfg := LoadConfig()

	if cfg.Token == "" {
		t.Skip("FIZZY_TEST_TOKEN not set, skipping integration tests")
	}
	if cfg.Account == "" {
		t.Skip("FIZZY_TEST_ACCOUNT not set, skipping integration tests")
	}

	tmpDir, err := os.MkdirTemp("", "fizzy-e2e-*")
	if err != nil {
		t.Fatalf("failed to create temp config dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	return &Harness{
		BinaryPath: cfg.BinaryPath,
		Token:      cfg.Token,
		Account:    cfg.Account,
		APIURL:     cfg.APIURL,
		Cleanup:    NewCleanupTracker(),
		configHome: tmpDir,
		t:          t,
	}
}

// NewWithConfig creates a new test harness with explicit configuration.
func NewWithConfig(t *testing.T, cfg *Config) *Harness {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "fizzy-e2e-*")
	if err != nil {
		t.Fatalf("failed to create temp config dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	return &Harness{
		BinaryPath: cfg.BinaryPath,
		Token:      cfg.Token,
		Account:    cfg.Account,
		APIURL:     cfg.APIURL,
		Cleanup:    NewCleanupTracker(),
		configHome: tmpDir,
		t:          t,
	}
}

// Run executes a CLI command and returns the result.
func (h *Harness) Run(args ...string) *Result {
	h.t.Helper()
	return h.RunWithEnv(nil, args...)
}

// RunWithEnv executes a CLI command with additional environment variables.
func (h *Harness) RunWithEnv(env map[string]string, args ...string) *Result {
	h.t.Helper()

	// Build full argument list with global options
	fullArgs := h.buildArgs(args...)

	// Merge global env (FIZZY_PROFILE) with caller-provided env
	mergedEnv := h.globalEnv()
	for k, v := range env {
		mergedEnv[k] = v
	}

	// Execute the command
	result := Execute(h.BinaryPath, fullArgs, mergedEnv)

	// Try to parse JSON response
	if result.Stdout != "" {
		var resp Response
		if err := json.Unmarshal([]byte(result.Stdout), &resp); err != nil {
			result.ParseError = err
		} else {
			result.Response = &resp
		}
	}

	return result
}

// RunWithoutAuth executes a CLI command without authentication.
func (h *Harness) RunWithoutAuth(args ...string) *Result {
	h.t.Helper()

	// Execute without global options (no token/account)
	result := Execute(h.BinaryPath, args, nil)

	// Try to parse JSON response
	if result.Stdout != "" {
		var resp Response
		if err := json.Unmarshal([]byte(result.Stdout), &resp); err != nil {
			result.ParseError = err
		} else {
			result.Response = &resp
		}
	}

	return result
}

// buildArgs builds the full argument list with global options.
func (h *Harness) buildArgs(args ...string) []string {
	globalArgs := []string{"--api-url", h.APIURL}
	if h.Token != "" {
		globalArgs = append(globalArgs, "--token", h.Token)
	}
	// Append global args after the command args
	return append(args, globalArgs...)
}

// globalEnv returns environment variables for the test harness.
// Uses a temporary HOME to isolate from host config/keyring.
func (h *Harness) globalEnv() map[string]string {
	return map[string]string{
		"FIZZY_PROFILE":    h.Account,
		"FIZZY_ACCOUNT":    "",
		"FIZZY_TOKEN":      "",
		"FIZZY_NO_KEYRING": "1",
		"HOME":             h.configHome,
		"XDG_CONFIG_HOME":  filepath.Join(h.configHome, "config"),
		"XDG_DATA_HOME":    filepath.Join(h.configHome, "data"),
		"XDG_STATE_HOME":   filepath.Join(h.configHome, "state"),
	}
}

// GetDataString extracts a string value from the response data.
func (r *Result) GetDataString(key string) string {
	if r.Response == nil || r.Response.Data == nil {
		return ""
	}
	data, ok := r.Response.Data.(map[string]any)
	if !ok {
		return ""
	}
	v, ok := data[key].(string)
	if !ok {
		return ""
	}
	return v
}

// GetDataInt extracts an integer value from the response data.
func (r *Result) GetDataInt(key string) int {
	if r.Response == nil || r.Response.Data == nil {
		return 0
	}
	data, ok := r.Response.Data.(map[string]any)
	if !ok {
		return 0
	}
	// JSON numbers are float64
	v, ok := data[key].(float64)
	if !ok {
		return 0
	}
	return int(v)
}

// GetDataBool extracts a boolean value from the response data.
func (r *Result) GetDataBool(key string) bool {
	if r.Response == nil || r.Response.Data == nil {
		return false
	}
	data, ok := r.Response.Data.(map[string]any)
	if !ok {
		return false
	}
	v, ok := data[key].(bool)
	if !ok {
		return false
	}
	return v
}

// GetDataArray extracts an array from the response data.
func (r *Result) GetDataArray() []any {
	if r.Response == nil || r.Response.Data == nil {
		return nil
	}
	arr, ok := r.Response.Data.([]any)
	if !ok {
		return nil
	}
	return arr
}

// GetDataMap extracts the data as a map.
func (r *Result) GetDataMap() map[string]any {
	if r.Response == nil || r.Response.Data == nil {
		return nil
	}
	data, ok := r.Response.Data.(map[string]any)
	if !ok {
		return nil
	}
	return data
}

// GetLocation returns the location URL from the response context.
func (r *Result) GetLocation() string {
	if r.Response == nil || r.Response.Context == nil {
		return ""
	}
	loc, ok := r.Response.Context["location"].(string)
	if !ok {
		return ""
	}
	return loc
}

// GetIDFromLocation extracts the resource ID from the location URL.
// Location format: /account/resource/ID.json
func (r *Result) GetIDFromLocation() string {
	loc := r.GetLocation()
	if loc == "" {
		return ""
	}
	// Remove .json suffix if present
	loc = strings.TrimSuffix(loc, ".json")
	// Get the last path segment
	parts := strings.Split(loc, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// GetNumberFromLocation extracts a numeric ID from the location URL.
// Used for cards which use numeric IDs.
func (r *Result) GetNumberFromLocation() int {
	idStr := r.GetIDFromLocation()
	if idStr == "" {
		return 0
	}
	// Try to parse as int
	var num int
	n, err := fmt.Sscanf(idStr, "%d", &num)
	if err != nil || n != 1 {
		return 0
	}
	return num
}
