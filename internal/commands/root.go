// Package commands implements CLI commands for the Fizzy CLI.
package commands

import (
	"bytes"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"

	"github.com/basecamp/cli/credstore"
	"github.com/basecamp/cli/output"
	"github.com/basecamp/fizzy-cli/internal/client"
	"github.com/basecamp/fizzy-cli/internal/config"
	"github.com/basecamp/fizzy-cli/internal/errors"
	"github.com/basecamp/fizzy-cli/internal/render"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

// Breadcrumb is a type alias for output.Breadcrumb.
type Breadcrumb = output.Breadcrumb

var (
	// Global flags
	cfgToken    string
	cfgAccount  string
	cfgAPIURL   string
	cfgVerbose  bool
	cfgJSON     bool
	cfgQuiet    bool
	cfgIDsOnly  bool
	cfgCount    bool
	cfgAgent    bool
	cfgStyled   bool
	cfgMarkdown bool
	cfgLimit    int

	// Loaded config
	cfg *config.Config

	// Client factory (can be overridden for testing)
	clientFactory func() client.API

	// Credential store
	creds *credstore.Store

	// Output writer
	out       *output.Writer
	outWriter io.Writer // raw writer for styled/markdown rendering
)

// rootCmd represents the base command.
var rootCmd = &cobra.Command{
	Use:   "fizzy",
	Short: "Fizzy CLI - Command-line interface for the Fizzy API",
	Long: `A command-line interface for the Fizzy API.

Use fizzy to manage boards, cards, comments, and more from your terminal.`,
	Version: "dev",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Resolve output format from parsed flags (must happen post-parse).
		format, err := resolveFormat()
		if err != nil {
			return &output.Error{Code: output.CodeUsage, Message: err.Error()}
		}
		if lastResult != nil {
			// Test mode — preserve test buffer as writer.
			outWriter = &testBuf
			out = output.New(output.Options{Format: format, Writer: &testBuf})
		} else {
			outWriter = os.Stdout
			out = output.New(output.Options{Format: format, Writer: os.Stdout})
		}

		// In test mode, cfg is already set by SetTestConfig - don't overwrite
		if cfg == nil {
			// Load config from file/env
			cfg = config.Load()
		}

		// Initialize credential store (skip in test mode)
		if creds == nil && lastResult == nil {
			fallbackDir := ""
			if cfgPath, err := config.ConfigPath(); err == nil {
				fallbackDir = filepath.Join(filepath.Dir(cfgPath), "credentials")
			} else if home, err := os.UserHomeDir(); err == nil {
				fallbackDir = filepath.Join(home, ".config", "fizzy", "credentials")
			}
			creds = credstore.NewStore(credstore.StoreOptions{
				ServiceName:   "fizzy",
				DisableEnvVar: "FIZZY_NO_KEYRING",
				FallbackDir:   fallbackDir,
			})
		}

		resolveToken()

		if cfgAccount != "" {
			cfg.Account = cfgAccount
		}
		if cfgAPIURL != "" {
			cfg.APIURL = cfgAPIURL
		}

		// FIZZY_DEBUG enables verbose output
		if os.Getenv("FIZZY_DEBUG") != "" {
			cfgVerbose = true
		}

		return nil
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

// SetVersion sets the CLI version used for `--version` and `version`.
func SetVersion(v string) {
	if v == "" {
		return
	}
	rootCmd.Version = v
}

// Execute runs the root command.
func Execute() {
	// Default to Auto — PersistentPreRunE will re-resolve from parsed flags.
	outWriter = os.Stdout
	out = output.New(output.Options{Format: output.FormatAuto, Writer: os.Stdout})
	if err := rootCmd.Execute(); err != nil {
		var e *output.Error
		if !stderrors.As(err, &e) {
			// Cobra-level errors (arg count, unknown flag) → usage
			e = &output.Error{Code: output.CodeUsage, Message: err.Error()}
		}
		_ = out.Err(e)
		os.Exit(e.ExitCode())
	}
}

// resolveFormat returns the output format from flags.
// Default is FormatAuto (TTY→Styled, pipe→JSON). At most one format flag may be set.
func resolveFormat() (output.Format, error) {
	// Count mutually exclusive format flags
	n := 0
	if cfgJSON {
		n++
	}
	if cfgQuiet {
		n++
	}
	if cfgIDsOnly {
		n++
	}
	if cfgCount {
		n++
	}
	if cfgStyled {
		n++
	}
	if cfgMarkdown {
		n++
	}
	if n > 1 {
		return 0, fmt.Errorf("only one output format flag may be used at a time (--json, --quiet, --ids-only, --count, --styled, --markdown)")
	}

	// --agent is orthogonal to format flags but --agent --styled is an error
	if cfgAgent && cfgStyled {
		return 0, fmt.Errorf("--agent and --styled cannot be used together")
	}

	// Explicit format flag wins
	switch {
	case cfgQuiet:
		return output.FormatQuiet, nil
	case cfgIDsOnly:
		return output.FormatIDs, nil
	case cfgCount:
		return output.FormatCount, nil
	case cfgJSON:
		return output.FormatJSON, nil
	case cfgStyled:
		return output.FormatStyled, nil
	case cfgMarkdown:
		return output.FormatMarkdown, nil
	}

	// --agent alone defaults to FormatQuiet
	if cfgAgent {
		return output.FormatQuiet, nil
	}

	return output.FormatAuto, nil
}

// IsMachineOutput returns true when output should be treated as machine-consumable.
// True when any machine format flag is set, --agent is set, or stdout/stdin is not a TTY.
func IsMachineOutput() bool {
	if cfgAgent || cfgJSON || cfgQuiet || cfgIDsOnly || cfgCount {
		return true
	}
	if !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		return true
	}
	if !isatty.IsTerminal(os.Stdin.Fd()) && !isatty.IsCygwinTerminal(os.Stdin.Fd()) {
		return true
	}
	return false
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgToken, "token", "", "API access token")
	rootCmd.PersistentFlags().StringVar(&cfgAccount, "account", "", "Account slug")
	rootCmd.PersistentFlags().StringVar(&cfgAPIURL, "api-url", "", "API base URL")
	rootCmd.PersistentFlags().BoolVar(&cfgVerbose, "verbose", false, "Show request/response details")
	rootCmd.PersistentFlags().BoolVar(&cfgJSON, "json", false, "JSON envelope output")
	rootCmd.PersistentFlags().BoolVar(&cfgQuiet, "quiet", false, "Raw JSON data without envelope")
	rootCmd.PersistentFlags().BoolVar(&cfgIDsOnly, "ids-only", false, "Print one ID per line")
	rootCmd.PersistentFlags().BoolVar(&cfgCount, "count", false, "Print count of results")
	rootCmd.PersistentFlags().BoolVar(&cfgAgent, "agent", false, "Agent mode (default: quiet format, no interactive prompts)")
	rootCmd.PersistentFlags().BoolVar(&cfgStyled, "styled", false, "Styled terminal output with colors")
	rootCmd.PersistentFlags().BoolVar(&cfgMarkdown, "markdown", false, "Markdown formatted output")
	rootCmd.PersistentFlags().IntVar(&cfgLimit, "limit", 0, "Maximum number of results to display")

	installAgentHelp()
}

// getClient returns an API client configured from global settings.
func getClient() client.API {
	if clientFactory != nil {
		return clientFactory()
	}
	c := client.New(cfg.APIURL, cfg.Token, cfg.Account)
	c.Verbose = cfgVerbose
	return c
}

// requireAuth checks that we have authentication configured.
func requireAuth() error {
	if cfg.Token == "" {
		return errors.NewAuthError("No API token configured. Run 'fizzy auth login TOKEN' or set FIZZY_TOKEN")
	}
	return nil
}

// requireAccount checks that we have an account configured.
func requireAccount() error {
	if cfg.Account == "" {
		return errors.NewInvalidArgsError("No account configured. Set --account flag or FIZZY_ACCOUNT")
	}
	return nil
}

// requireAuthAndAccount checks both auth and account.
func requireAuthAndAccount() error {
	if err := requireAuth(); err != nil {
		return err
	}
	return requireAccount()
}

func effectiveConfig() *config.Config {
	if cfg != nil {
		return cfg
	}
	return config.Load()
}

func defaultBoard(board string) string {
	if board != "" {
		return board
	}
	return effectiveConfig().Board
}

func requireBoard(board string) (string, error) {
	board = defaultBoard(board)
	if board == "" {
		return "", errors.NewInvalidArgsError("No board configured. Set --board, FIZZY_BOARD, or add 'board' to your config file")
	}
	return board, nil
}

// CommandResult holds the result of a command execution for testing.
type CommandResult struct {
	Response *output.Response
}

// lastResult stores the last command result (for testing)
var lastResult *CommandResult

// testBuf captures output for test mode
var testBuf bytes.Buffer

// lastRawOutput holds the raw output from the last command (before buffer reset).
var lastRawOutput string

// captureResponse parses the writer buffer into lastResult after each shim call.
func captureResponse() {
	if lastResult == nil {
		return
	}
	lastRawOutput = testBuf.String()
	lastResult.Response = nil
	var resp output.Response
	if json.Unmarshal(testBuf.Bytes(), &resp) == nil {
		lastResult.Response = &resp
	}
	testBuf.Reset()
}

// printSuccess prints a success response.
func printSuccess(data any) {
	_ = out.OK(data)
	captureResponse()
}

// printSuccessWithLocation prints a success response with location.
func printSuccessWithLocation(location string) {
	_ = out.OK(nil, output.WithContext("location", location))
	captureResponse()
}

// breadcrumb creates a single breadcrumb.
func breadcrumb(action, cmd, description string) Breadcrumb {
	return Breadcrumb{Action: action, Cmd: cmd, Description: description}
}

// printSuccessWithBreadcrumbs prints a success response with breadcrumbs.
func printSuccessWithBreadcrumbs(data any, summary string, breadcrumbs []Breadcrumb) {
	opts := []output.ResponseOption{output.WithBreadcrumbs(breadcrumbs...)}
	if summary != "" {
		opts = append(opts, output.WithSummary(summary))
	}
	_ = out.OK(data, opts...)
	captureResponse()
}

// printSuccessWithLocationAndBreadcrumbs prints a success response with both location and breadcrumbs.
func printSuccessWithLocationAndBreadcrumbs(data any, location string, breadcrumbs []Breadcrumb) {
	_ = out.OK(data,
		output.WithBreadcrumbs(breadcrumbs...),
		output.WithContext("location", location),
	)
	captureResponse()
}

// defaultPageSize is the Fizzy API's default page size.
const defaultPageSize = 20

// checkLimitAll validates that --limit and --all are not both set.
func checkLimitAll(all bool) error {
	if cfgLimit > 0 && all {
		return errors.NewInvalidArgsError("--limit and --all cannot be used together")
	}
	return nil
}

// truncateData applies --limit client-side truncation to a slice.
// Returns the (possibly truncated) data and the original count.
// Handles both []any and typed slices (e.g. []Attachment).
func truncateData(data any) (any, int) {
	if arr, ok := data.([]any); ok {
		originalCount := len(arr)
		if cfgLimit > 0 && originalCount > cfgLimit {
			return arr[:cfgLimit], originalCount
		}
		return data, originalCount
	}
	// Handle typed slices via reflect
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Slice {
		originalCount := v.Len()
		if cfgLimit > 0 && originalCount > cfgLimit {
			return v.Slice(0, cfgLimit).Interface(), originalCount
		}
		return data, originalCount
	}
	return data, 0
}

// dataCount returns the length of data if it's a slice.
func dataCount(data any) int {
	if arr, ok := data.([]any); ok {
		return len(arr)
	}
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Slice {
		return v.Len()
	}
	return 0
}

// printList renders list data with format-aware dispatch.
// For non-paginated lists (no --all flag). Applies --limit truncation.
func printList(data any, cols render.Columns, summary string, breadcrumbs []Breadcrumb) {
	data, originalCount := truncateData(data)

	// For non-paginated lists, generate a simple limit notice (no --all to suggest)
	notice := ""
	if cfgLimit > 0 && originalCount > cfgLimit {
		notice = fmt.Sprintf("Showing %d of %d results", cfgLimit, originalCount)
	}

	switch out.EffectiveFormat() {
	case output.FormatStyled:
		fmt.Fprint(outWriter, render.StyledList(toMaps(data), cols, summary))
		if notice != "" {
			fmt.Fprintf(outWriter, "\n%s\n", notice)
		}
		captureResponse()
	case output.FormatMarkdown:
		fmt.Fprint(outWriter, render.MarkdownList(toMaps(data), cols, summary))
		if notice != "" {
			fmt.Fprintf(outWriter, "\n%s\n", notice)
		}
		captureResponse()
	default:
		opts := []output.ResponseOption{output.WithBreadcrumbs(breadcrumbs...)}
		if summary != "" {
			opts = append(opts, output.WithSummary(summary))
		}
		if notice != "" {
			opts = append(opts, output.WithNotice(notice))
		}
		_ = out.OK(data, opts...)
		captureResponse()
	}
}

// printListPaginated renders paginated list data with format-aware dispatch.
// For paginated lists (commands with --all flag). Applies --limit truncation and truncation notices.
func printListPaginated(data any, cols render.Columns, hasNext bool, nextURL string, all bool, summary string, breadcrumbs []Breadcrumb) {
	data, _ = truncateData(data)
	notice := output.TruncationNotice(dataCount(data), defaultPageSize, all, cfgLimit)

	switch out.EffectiveFormat() {
	case output.FormatStyled:
		fmt.Fprint(outWriter, render.StyledList(toMaps(data), cols, summary))
		if notice != "" {
			fmt.Fprintf(outWriter, "\n%s\n", notice)
		}
		captureResponse()
	case output.FormatMarkdown:
		fmt.Fprint(outWriter, render.MarkdownList(toMaps(data), cols, summary))
		if notice != "" {
			fmt.Fprintf(outWriter, "\n%s\n", notice)
		}
		captureResponse()
	default:
		opts := []output.ResponseOption{output.WithBreadcrumbs(breadcrumbs...)}
		if summary != "" {
			opts = append(opts, output.WithSummary(summary))
		}
		if notice != "" {
			opts = append(opts, output.WithNotice(notice))
		}
		if hasNext || nextURL != "" {
			opts = append(opts, output.WithContext("pagination", map[string]any{
				"has_next": hasNext,
				"next_url": nextURL,
			}))
		}
		_ = out.OK(data, opts...)
		captureResponse()
	}
}

// printDetail renders a single object with format-aware dispatch.
func printDetail(data any, summary string, breadcrumbs []Breadcrumb) {
	switch out.EffectiveFormat() {
	case output.FormatStyled:
		fmt.Fprint(outWriter, render.StyledDetail(toMap(data), summary))
		captureResponse()
	case output.FormatMarkdown:
		fmt.Fprint(outWriter, render.MarkdownDetail(toMap(data), summary))
		captureResponse()
	default:
		printSuccessWithBreadcrumbs(data, summary, breadcrumbs)
	}
}

// printMutationWithLocation renders a mutation result that includes a location URL.
func printMutationWithLocation(data any, location string, breadcrumbs []Breadcrumb) {
	switch out.EffectiveFormat() {
	case output.FormatStyled:
		fmt.Fprint(outWriter, render.StyledDetail(toMap(data), ""))
		captureResponse()
	case output.FormatMarkdown:
		fmt.Fprint(outWriter, render.MarkdownDetail(toMap(data), ""))
		captureResponse()
	default:
		printSuccessWithLocationAndBreadcrumbs(data, location, breadcrumbs)
	}
}

// printMutation renders a mutation result with format-aware dispatch.
// For styled/markdown, uses summary rendering for simple confirmations.
func printMutation(data any, summary string, breadcrumbs []Breadcrumb) {
	switch out.EffectiveFormat() {
	case output.FormatStyled:
		fmt.Fprint(outWriter, render.StyledSummary(toMap(data), summary))
		captureResponse()
	case output.FormatMarkdown:
		fmt.Fprint(outWriter, render.MarkdownSummary(toMap(data), summary))
		captureResponse()
	default:
		printSuccessWithBreadcrumbs(data, summary, breadcrumbs)
	}
}

// toMaps converts any (expected []any of map[string]any) to []map[string]any.
// Falls back to JSON round-trip for typed slices (e.g., []Attachment).
func toMaps(data any) []map[string]any {
	if data == nil {
		return nil
	}
	if slice, ok := data.([]any); ok {
		result := make([]map[string]any, 0, len(slice))
		for _, item := range slice {
			if m, ok := item.(map[string]any); ok {
				result = append(result, m)
			}
		}
		return result
	}
	// JSON round-trip for typed structs
	b, err := json.Marshal(data)
	if err != nil {
		return nil
	}
	var result []map[string]any
	if json.Unmarshal(b, &result) == nil {
		return result
	}
	return nil
}

// toMap converts any (expected map[string]any) to map[string]any.
func toMap(data any) map[string]any {
	if m, ok := data.(map[string]any); ok {
		return m
	}
	return nil
}

// SetTestMode configures the commands package for testing.
// It sets a mock client factory and captures results instead of exiting.
func SetTestMode(mockClient client.API) *CommandResult {
	clientFactory = func() client.API {
		return mockClient
	}
	testBuf.Reset()
	outWriter = &testBuf
	out = output.New(output.Options{Format: output.FormatJSON, Writer: &testBuf})
	lastResult = &CommandResult{}
	return lastResult
}

// SetTestFormat reconfigures the output writer with the given format.
// Must be called after SetTestMode.
func SetTestFormat(format output.Format) {
	testBuf.Reset()
	outWriter = &testBuf
	out = output.New(output.Options{Format: format, Writer: &testBuf})
}

// TestOutput returns the raw output from the last command execution.
// Useful for verifying non-JSON format output.
func TestOutput() string {
	return lastRawOutput
}

// credsSaveToken JSON-encodes a token and saves it to the credential store.
// The file backend requires valid JSON (values are stored as json.RawMessage).
func credsSaveToken(token string) error {
	data, err := json.Marshal(token)
	if err != nil {
		return err
	}
	return creds.Save("token", data)
}

// credsLoadToken loads and JSON-decodes a token from the credential store.
// Falls back to treating the payload as a raw string for backward compatibility
// with pre-JSON credstore entries.
func credsLoadToken() (string, error) {
	data, err := creds.Load("token")
	if err != nil {
		return "", err
	}
	var token string
	if json.Unmarshal(data, &token) == nil {
		return token, nil
	}
	// Legacy: raw string stored before JSON encoding was adopted
	return string(data), nil
}

// resolveToken applies token precedence: YAML → credstore (with migration) → env → flag.
func resolveToken() {
	// 1. YAML file (global + local, already in cfg.Token from config.Load())
	// 2. credstore (overrides YAML — credstore is the "new" storage)
	if creds != nil {
		if t, err := credsLoadToken(); err == nil && t != "" {
			cfg.Token = t
		} else {
			// Auto-migrate: if the global YAML config has a token but credstore
			// doesn't, migrate it. Only read from the global file directly to
			// avoid persisting env-var or local-config tokens.
			globalCfg := config.LoadGlobal()
			if globalCfg.Token != "" {
				if err := credsSaveToken(globalCfg.Token); err == nil {
					globalCfg.Token = ""
					_ = globalCfg.Save()
				}
			}
		}
	}
	// 3. env var (overrides credstore)
	if t := os.Getenv("FIZZY_TOKEN"); t != "" {
		cfg.Token = t
	}
	// 4. CLI flag (overrides everything)
	if cfgToken != "" {
		cfg.Token = cfgToken
	}
}

// SetTestCreds sets the credential store for testing.
func SetTestCreds(store *credstore.Store) {
	creds = store
}

// SetTestConfig sets the config for testing.
func SetTestConfig(token, account, apiURL string) {
	cfg = &config.Config{
		Token:   token,
		Account: account,
		APIURL:  apiURL,
	}
}

// ResetTestMode resets the test mode configuration.
func ResetTestMode() {
	clientFactory = nil
	lastResult = nil
	lastRawOutput = ""
	cfg = nil
	creds = nil
	cfgJSON = false
	cfgQuiet = false
	cfgIDsOnly = false
	cfgCount = false
	cfgAgent = false
	cfgStyled = false
	cfgMarkdown = false
	cfgLimit = 0
}

// GetRootCmd returns the root command for testing.
func GetRootCmd() *cobra.Command {
	return rootCmd
}

// Helper function for required flag errors
func newRequiredFlagError(flag string) error {
	return errors.NewInvalidArgsError("required flag --" + flag + " not provided")
}
