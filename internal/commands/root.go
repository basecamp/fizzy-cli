// Package commands implements CLI commands for the Fizzy CLI.
package commands

import (
	"bytes"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/basecamp/cli/credstore"
	"github.com/basecamp/cli/output"
	"github.com/basecamp/cli/profile"
	"github.com/basecamp/fizzy-cli/internal/client"
	"github.com/basecamp/fizzy-cli/internal/config"
	"github.com/basecamp/fizzy-cli/internal/errors"
	"github.com/basecamp/fizzy-cli/internal/render"
	fizzy "github.com/basecamp/fizzy-sdk/go/pkg/fizzy"
	"github.com/itchyny/gojq"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

// Breadcrumb is a type alias for output.Breadcrumb.
type Breadcrumb = output.Breadcrumb

var (
	// Global flags
	cfgToken    string
	cfgProfile  string
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
	cfgJQ       string

	// Loaded config
	cfg *config.Config

	// Client factory (can be overridden for testing)
	clientFactory func() client.API

	// SDK client
	sdk        *fizzy.Client
	sdkAccount func() *fizzy.AccountClient

	// Credential store
	creds *credstore.Store

	// Profile store
	profiles *profile.Store

	// Output writer
	out       *output.Writer
	outWriter io.Writer // raw writer for styled/markdown rendering
)

var cliVersion = "dev"

// rootCmd represents the base command.
var rootCmd = &cobra.Command{
	Use:     "fizzy",
	Short:   "Fizzy CLI - Command-line interface for the Fizzy API",
	Long:    `Command-line interface for Fizzy`,
	Version: "dev",
	RunE:    runRootDefault,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		errOutputWrite = nil
		// Early jq validation: check flag conflicts first (actionable message),
		// then parse + compile before RunE so invalid expressions are rejected
		// with no side effects. The compiled code is reused below to avoid
		// parsing the expression twice.
		var jqCode *gojq.Code
		if cfgJQ != "" {
			if cfgIDsOnly {
				return errors.ErrJQConflict("--ids-only")
			}
			if cfgCount {
				return errors.ErrJQConflict("--count")
			}
			var err error
			jqCode, err = compileJQ(cfgJQ)
			if err != nil {
				return err
			}
		}

		// Resolve output format from parsed flags (must happen post-parse).
		format, err := resolveFormat()
		if err != nil {
			return &output.Error{Code: output.CodeUsage, Message: err.Error()}
		}
		if lastResult != nil {
			// Test mode — preserve test buffer as writer.
			outWriter = &testBuf
			var w io.Writer = &testBuf
			if jqCode != nil {
				w = newJQWriterWithCode(&testBuf, jqCode)
			}
			out = output.New(output.Options{Format: format, Writer: w})
		} else {
			outWriter = os.Stdout
			var w io.Writer = os.Stdout
			if jqCode != nil {
				w = newJQWriterWithCode(os.Stdout, jqCode)
			}
			out = output.New(output.Options{Format: format, Writer: w})
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

		// Initialize profile store (skip in test mode)
		if profiles == nil && lastResult == nil {
			if cfgPath, err := config.ConfigPath(); err == nil {
				profiles = profile.NewStore(filepath.Join(filepath.Dir(cfgPath), "config.json"))
			}
		}

		if err := resolveProfile(); err != nil {
			return &output.Error{Code: output.CodeUsage, Message: err.Error()}
		}
		resolveToken()

		// --api-url flag overrides everything (including profile BaseURL)
		if cfgAPIURL != "" {
			cfg.APIURL = cfgAPIURL
		}

		// FIZZY_DEBUG enables verbose output
		if os.Getenv("FIZZY_DEBUG") != "" {
			cfgVerbose = true
		}

		// Initialize SDK client (skip if already set by test mode)
		if sdk == nil {
			if err := initSDK(cmd, cfg.APIURL, cfg.Token, cfg.Account); err != nil {
				// Non-fatal: commands that don't need the SDK (e.g. help, version) can proceed.
				// Commands that call getSDK() will return the stored initialization error.
				errSDKInit = err
			}
		}

		return nil
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		if errOutputWrite != nil {
			err := errOutputWrite
			errOutputWrite = nil
			return err
		}
		if RefreshSkillsIfVersionChanged() && !IsMachineOutput() {
			fmt.Fprintf(os.Stderr, "Agent skill updated to match CLI %s\n", currentVersion())
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
	cliVersion = v
	rootCmd.Version = v
}

// Execute runs the root command.
func Execute() {
	configureCLIUX()

	// Default to Auto — PersistentPreRunE will re-resolve from parsed flags.
	outWriter = os.Stdout
	out = output.New(output.Options{Format: output.FormatAuto, Writer: os.Stdout})
	cmd, err := rootCmd.ExecuteC()
	if err != nil {
		if format, formatErr := resolveFormat(); formatErr == nil {
			out = output.New(output.Options{Format: format, Writer: os.Stdout})
		}

		var e *output.Error
		if !stderrors.As(err, &e) {
			// Cobra-level errors (arg count, unknown flag) → usage
			e = &output.Error{Code: output.CodeUsage, Message: err.Error()}
		}

		// jq-related errors (validation failures, unsupported commands, conflicts)
		// must never be fed through the jq filter. Rebuild the output writer
		// without jq so the error renders cleanly. Re-resolve the format to
		// honor explicit flags like --agent --json.
		if errors.IsJQError(err) && cfgJQ != "" {
			format, fmtErr := resolveFormat()
			if fmtErr != nil {
				// resolveFormat() can fail when --jq conflicts with another flag
				// (e.g. --jq --styled). Fall back to a sensible machine format.
				format = output.FormatJSON
				if cfgAgent || cfgQuiet {
					format = output.FormatQuiet
				}
			}
			out = output.New(output.Options{Format: format, Writer: outWriter})
		}
		if isHumanOutput() {
			printHumanError(cmd, e)
		} else {
			_ = out.Err(e)
		}
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

	// --jq is a JSON transform and is incompatible with human/count/id renderers.
	if cfgJQ != "" && (cfgStyled || cfgMarkdown || cfgIDsOnly || cfgCount) {
		return 0, fmt.Errorf("--jq filters JSON output; use it with default JSON output or --quiet, not with --styled, --markdown, --ids-only, or --count")
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

	// --jq implies JSON (or quiet for --agent)
	if cfgJQ != "" {
		if cfgAgent {
			return output.FormatQuiet, nil
		}
		return output.FormatJSON, nil
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
	if cfgAgent || cfgJSON || cfgQuiet || cfgIDsOnly || cfgCount || cfgJQ != "" {
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

func isHumanOutput() bool {
	if cfgStyled || cfgMarkdown || requestedHumanOutput() {
		return true
	}
	if out != nil {
		switch out.EffectiveFormat() {
		case output.FormatStyled, output.FormatMarkdown:
			return true
		default:
			return false
		}
	}
	return !IsMachineOutput()
}

func requestedHumanOutput() bool {
	for _, arg := range os.Args[1:] {
		if arg == "--styled" || arg == "--markdown" {
			return true
		}
	}
	return false
}

func printHumanError(cmd *cobra.Command, err error) {
	e := output.AsError(err)
	msg := strings.TrimSpace(e.Message)
	if msg != "" {
		fmt.Fprintln(os.Stderr, msg)
	}
	if e.Hint != "" && !strings.Contains(msg, e.Hint) {
		fmt.Fprintf(os.Stderr, "\nHint: %s\n", e.Hint)
	}
	if e.Code == output.CodeUsage && !strings.Contains(msg, "--help") {
		fmt.Fprintf(os.Stderr, "\nRun `%s` for usage.\n", usageHelpCommand(cmd))
	}
}

func usageHelpCommand(cmd *cobra.Command) string {
	if cmd == nil {
		return rootCmd.CommandPath() + " --help"
	}
	path := strings.TrimSpace(cmd.CommandPath())
	if path == "" {
		path = rootCmd.CommandPath()
	}
	return path + " --help"
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgToken, "token", "", "API access token")
	rootCmd.PersistentFlags().StringVar(&cfgProfile, "profile", "", "Named profile to use")
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
	rootCmd.PersistentFlags().StringVar(&cfgJQ, "jq", "", "Apply jq filter to JSON output (built-in, no external jq required; implies --json)")

	installAgentHelp()
}

// getClient returns an API client configured from global settings.
// Deprecated: Use getSDK() for new code.
func getClient() client.API {
	if clientFactory != nil {
		return clientFactory()
	}
	c := client.New(cfg.APIURL, cfg.Token, cfg.Account)
	c.Verbose = cfgVerbose
	return c
}

// errSDKInit stores any error from SDK initialization so commands can return it.
var errSDKInit error

// getSDK returns an SDK AccountClient bound to the configured account.
func getSDK() *fizzy.AccountClient {
	return sdkAccount()
}

// getSDKClient returns the root SDK Client (for account-independent operations).
func getSDKClient() *fizzy.Client {
	return sdk
}

// requireSDK returns the SDK init error if the SDK failed to initialize.
// Commands that use getSDK()/getSDKClient() should call this first.
func requireSDK() error {
	if errSDKInit != nil {
		return errSDKInit
	}
	return nil
}

// initSDK creates the SDK client, guarding against panics from HTTPS validation.
func initSDK(cmd *cobra.Command, apiURL, token, account string) (initErr error) {
	defer func() {
		if r := recover(); r != nil {
			initErr = &output.Error{
				Code:    output.CodeNetwork,
				Message: fmt.Sprintf("Cannot initialize SDK: %v", r),
				Hint:    "Non-localhost API URLs must use HTTPS. Update your config with 'fizzy setup'.",
			}
		}
	}()

	sdkCfg := &fizzy.Config{
		BaseURL: apiURL,
	}
	var opts []fizzy.ClientOption
	opts = append(opts, fizzy.WithUserAgent("fizzy-cli/"+cmd.Root().Version))
	if cfgVerbose {
		opts = append(opts, fizzy.WithHooks(fizzy.NewSlogHooks(slog.New(slog.NewTextHandler(os.Stderr, nil)))))
	}
	sdk = fizzy.NewClient(sdkCfg, &fizzy.StaticTokenProvider{Token: token}, opts...)
	sdkAccount = func() *fizzy.AccountClient {
		return sdk.ForAccount(account)
	}
	return nil
}

// normalizeAny converts any value to map[string]any or []map[string]any
// via JSON round-trip. Handles typed structs (e.g. *generated.Board),
// typed slices (e.g. []generated.Card), json.RawMessage, and plain
// map/slice types that are already in the right shape.
func normalizeAny(v any) any {
	if v == nil {
		return nil
	}
	switch d := v.(type) {
	case json.RawMessage:
		if len(d) == 0 {
			return nil
		}
		var parsed any
		if json.Unmarshal(d, &parsed) != nil {
			return nil
		}
		return normalizeAny(parsed)
	case map[string]any:
		return d
	case []map[string]any:
		return d
	case []any:
		maps := make([]map[string]any, 0, len(d))
		for _, item := range d {
			if m, ok := item.(map[string]any); ok {
				maps = append(maps, m)
			} else {
				return v // mixed types, return as-is
			}
		}
		return maps
	}
	// Typed struct or slice — JSON round-trip
	b, err := json.Marshal(v)
	if err != nil {
		return v
	}
	var result any
	if json.Unmarshal(b, &result) != nil {
		return v
	}
	return normalizeAny(result)
}

// jsonAnySlice converts []json.RawMessage (from GetAll pagination) to []any.
func jsonAnySlice(items []json.RawMessage) any {
	maps := make([]map[string]any, 0, len(items))
	for _, item := range items {
		var m map[string]any
		if json.Unmarshal(item, &m) == nil {
			maps = append(maps, m)
		}
	}
	return maps
}

// toSliceAny converts []map[string]any or []any to []any for iteration.
func toSliceAny(v any) []any {
	switch d := v.(type) {
	case []any:
		return d
	case []map[string]any:
		result := make([]any, len(d))
		for i, m := range d {
			result[i] = m
		}
		return result
	}
	return nil
}

// parseSDKLinkNext extracts the next page URL from SDK response Link headers.
func parseSDKLinkNext(resp *fizzy.Response) string {
	if resp == nil {
		return ""
	}
	linkHeader := resp.Headers.Get("Link")
	if linkHeader == "" {
		return ""
	}
	// Parse Link header for rel="next"
	for _, part := range strings.Split(linkHeader, ",") {
		part = strings.TrimSpace(part)
		if strings.Contains(part, `rel="next"`) {
			start := strings.Index(part, "<")
			end := strings.Index(part, ">")
			if start >= 0 && end > start {
				return part[start+1 : end]
			}
		}
	}
	return ""
}

// requireAuth checks that we have authentication configured.
// Does NOT require the SDK — legacy commands (upload, download, multipart)
// only need a valid token and account, not an initialized SDK client.
func requireAuth() error {
	if cfg.Token == "" {
		return errors.NewAuthError("No API token configured. Run 'fizzy auth login TOKEN' or set FIZZY_TOKEN")
	}
	return nil
}

// requireAccount checks that we have an account configured.
func requireAccount() error {
	if cfg.Account == "" {
		return errors.NewInvalidArgsError("No account configured. Set --profile flag, FIZZY_PROFILE, or run 'fizzy setup'")
	}
	return nil
}

// requireAuthAndAccount checks auth, account, and SDK initialization.
func requireAuthAndAccount() error {
	if err := requireAuth(); err != nil {
		return err
	}
	if err := requireAccount(); err != nil {
		return err
	}
	return requireSDK()
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

// errOutputWrite stores the first output rendering/writer error from the current command.
var errOutputWrite error

func recordOutputError(err error) {
	if err != nil && errOutputWrite == nil {
		errOutputWrite = err
	}
}

func writeOutputString(s string) {
	_, err := io.WriteString(outWriter, s)
	recordOutputError(err)
}

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
	switch out.EffectiveFormat() {
	case output.FormatStyled:
		writeOutputString(renderHumanData(data, "", false))
		captureResponse()
	case output.FormatMarkdown:
		writeOutputString(renderHumanData(data, "", true))
		captureResponse()
	default:
		recordOutputError(out.OK(data))
		captureResponse()
	}
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
	recordOutputError(out.OK(data, opts...))
	captureResponse()
}

// printSuccessWithLocationAndBreadcrumbs prints a success response with both location and breadcrumbs.
func printSuccessWithLocationAndBreadcrumbs(data any, location string, breadcrumbs []Breadcrumb) {
	recordOutputError(out.OK(data,
		output.WithBreadcrumbs(breadcrumbs...),
		output.WithContext("location", location),
	))
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
		body := render.StyledList(toMaps(data), cols, summary)
		writeOutputString(appendHumanSections(body, notice, "", breadcrumbs, false))
		captureResponse()
	case output.FormatMarkdown:
		body := render.MarkdownList(toMaps(data), cols, summary)
		writeOutputString(appendHumanSections(body, notice, "", breadcrumbs, true))
		captureResponse()
	default:
		opts := []output.ResponseOption{output.WithBreadcrumbs(breadcrumbs...)}
		if summary != "" {
			opts = append(opts, output.WithSummary(summary))
		}
		if notice != "" {
			opts = append(opts, output.WithNotice(notice))
		}
		recordOutputError(out.OK(data, opts...))
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
		body := render.StyledList(toMaps(data), cols, summary)
		writeOutputString(appendHumanSections(body, notice, "", breadcrumbs, false))
		captureResponse()
	case output.FormatMarkdown:
		body := render.MarkdownList(toMaps(data), cols, summary)
		writeOutputString(appendHumanSections(body, notice, "", breadcrumbs, true))
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
		recordOutputError(out.OK(data, opts...))
		captureResponse()
	}
}

// printDetail renders a single object with format-aware dispatch.
func printDetail(data any, summary string, breadcrumbs []Breadcrumb) {
	printDetailPaginated(data, summary, breadcrumbs, false, "")
}

// printDetailPaginated renders a single object and includes pagination context when present.
func printDetailPaginated(data any, summary string, breadcrumbs []Breadcrumb, hasNext bool, nextURL string) {
	switch out.EffectiveFormat() {
	case output.FormatStyled:
		body := render.StyledDetail(toMap(data), summary)
		writeOutputString(appendHumanSections(body, "", "", breadcrumbs, false))
		captureResponse()
	case output.FormatMarkdown:
		body := render.MarkdownDetail(toMap(data), summary)
		writeOutputString(appendHumanSections(body, "", "", breadcrumbs, true))
		captureResponse()
	default:
		opts := []output.ResponseOption{output.WithBreadcrumbs(breadcrumbs...)}
		if summary != "" {
			opts = append(opts, output.WithSummary(summary))
		}
		if hasNext || nextURL != "" {
			opts = append(opts, output.WithContext("pagination", map[string]any{
				"has_next": hasNext,
				"next_url": nextURL,
			}))
		}
		recordOutputError(out.OK(data, opts...))
		captureResponse()
	}
}

// printMutationWithLocation renders a mutation result that includes a location URL.
func printMutationWithLocation(data any, location string, breadcrumbs []Breadcrumb) {
	switch out.EffectiveFormat() {
	case output.FormatStyled:
		body := render.StyledDetail(toMap(data), "")
		writeOutputString(appendHumanSections(body, "", location, breadcrumbs, false))
		captureResponse()
	case output.FormatMarkdown:
		body := render.MarkdownDetail(toMap(data), "")
		writeOutputString(appendHumanSections(body, "", location, breadcrumbs, true))
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
		body := render.StyledSummary(toMap(data), summary)
		writeOutputString(appendHumanSections(body, "", "", breadcrumbs, false))
		captureResponse()
	case output.FormatMarkdown:
		body := render.MarkdownSummary(toMap(data), summary)
		writeOutputString(appendHumanSections(body, "", "", breadcrumbs, true))
		captureResponse()
	default:
		printSuccessWithBreadcrumbs(data, summary, breadcrumbs)
	}
}

func renderHumanData(data any, location string, markdown bool) string {
	switch v := data.(type) {
	case nil:
		if markdown {
			return appendHumanSections(render.MarkdownSummary(nil, ""), "", location, nil, true)
		}
		return appendHumanSections(render.StyledSummary(nil, ""), "", location, nil, false)
	case map[string]any:
		if markdown {
			return appendHumanSections(render.MarkdownDetail(v, ""), "", location, nil, true)
		}
		return appendHumanSections(render.StyledDetail(v, ""), "", location, nil, false)
	}

	if maps := toMaps(data); maps != nil {
		cols := inferColumns(maps)
		if markdown {
			return appendHumanSections(render.MarkdownList(maps, cols, ""), "", location, nil, true)
		}
		return appendHumanSections(render.StyledList(maps, cols, ""), "", location, nil, false)
	}

	if m := toMap(data); m != nil {
		if markdown {
			return appendHumanSections(render.MarkdownDetail(m, ""), "", location, nil, true)
		}
		return appendHumanSections(render.StyledDetail(m, ""), "", location, nil, false)
	}

	value := fmt.Sprintf("%v\n", data)
	return appendHumanSections(value, "", location, nil, markdown)
}

func appendHumanSections(body, notice, location string, breadcrumbs []Breadcrumb, markdown bool) string {
	body = strings.TrimRight(body, "\n")
	var sb strings.Builder
	if body != "" {
		sb.WriteString(body)
		sb.WriteString("\n")
	}
	if notice != "" {
		sb.WriteString("\n")
		if markdown {
			sb.WriteString("> ")
		}
		sb.WriteString(notice)
		sb.WriteString("\n")
	}
	if location != "" {
		sb.WriteString("\n")
		if markdown {
			sb.WriteString("**Location:** `")
			sb.WriteString(location)
			sb.WriteString("`\n")
		} else {
			sb.WriteString("Location: ")
			sb.WriteString(location)
			sb.WriteString("\n")
		}
	}
	if len(breadcrumbs) > 0 {
		sb.WriteString("\n")
		if markdown {
			sb.WriteString("### Next steps\n")
			for _, crumb := range breadcrumbs {
				sb.WriteString("- `")
				sb.WriteString(crumb.Cmd)
				sb.WriteString("`")
				if crumb.Description != "" {
					sb.WriteString(" — ")
					sb.WriteString(crumb.Description)
				}
				sb.WriteString("\n")
			}
		} else {
			sb.WriteString("Next steps:\n")
			for _, crumb := range breadcrumbs {
				sb.WriteString("  ")
				sb.WriteString(crumb.Cmd)
				if crumb.Description != "" {
					sb.WriteString("  # ")
					sb.WriteString(crumb.Description)
				}
				sb.WriteString("\n")
			}
		}
	}
	return sb.String()
}

func inferColumns(data []map[string]any) render.Columns {
	if len(data) == 0 {
		return render.Columns{{Header: "Value", Field: "id"}}
	}

	priority := map[string]int{
		"number":      0,
		"id":          1,
		"profile":     2,
		"name":        3,
		"title":       4,
		"description": 5,
		"active":      6,
		"board":       7,
		"base_url":    8,
	}

	keys := make([]string, 0, len(data[0]))
	for key := range data[0] {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		pi, okI := priority[keys[i]]
		pj, okJ := priority[keys[j]]
		if okI && okJ {
			return pi < pj
		}
		if okI {
			return true
		}
		if okJ {
			return false
		}
		return keys[i] < keys[j]
	})

	if len(keys) > 4 {
		keys = keys[:4]
	}

	cols := make(render.Columns, 0, len(keys))
	for _, key := range keys {
		cols = append(cols, render.Column{Header: humanizeFieldName(key), Field: key})
	}
	return cols
}

func humanizeFieldName(name string) string {
	parts := strings.Split(name, "_")
	for i, part := range parts {
		if part == "id" {
			parts[i] = "ID"
			continue
		}
		parts[i] = titleWord(part)
	}
	return strings.Join(parts, " ")
}

func titleWord(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(strings.ToLower(s))
	runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
	return string(runes)
}

// toMaps converts any (expected []any of map[string]any) to []map[string]any.
// Falls back to JSON round-trip for typed slices (e.g., []Attachment).
func toMaps(data any) []map[string]any {
	if data == nil {
		return nil
	}
	if maps, ok := data.([]map[string]any); ok {
		return maps
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
// Falls back to JSON round-trip for typed structs.
func toMap(data any) map[string]any {
	if m, ok := data.(map[string]any); ok {
		return m
	}
	// JSON round-trip for typed structs
	b, err := json.Marshal(data)
	if err != nil {
		return nil
	}
	var m map[string]any
	if json.Unmarshal(b, &m) == nil {
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

// credsSaveProfileToken JSON-encodes a token and saves it to the credential store
// under a profile-scoped key ("profile:<name>").
func credsSaveProfileToken(profileName, token string) error {
	data, err := json.Marshal(token)
	if err != nil {
		return err
	}
	return creds.Save(profile.CredentialKey(profileName, ""), data)
}

// credsLoadProfileToken loads and JSON-decodes a token for a profile.
func credsLoadProfileToken(profileName string) (string, error) {
	data, err := creds.Load(profile.CredentialKey(profileName, ""))
	if err != nil {
		return "", err
	}
	var token string
	if json.Unmarshal(data, &token) == nil {
		return token, nil
	}
	return string(data), nil
}

// credsDeleteProfileToken removes the token for a profile.
func credsDeleteProfileToken(profileName string) error {
	return creds.Delete(profile.CredentialKey(profileName, ""))
}

// credsLoadLegacyToken loads a token from a legacy credstore entry.
// Checks both the old single "token" key and the account-scoped "token:<account>" key.
func credsLoadLegacyToken(account string) (string, error) {
	// Try account-scoped key first (from earlier multi-account PR)
	if account != "" {
		if data, err := creds.Load("token:" + account); err == nil {
			var token string
			if json.Unmarshal(data, &token) == nil {
				return token, nil
			}
			return string(data), nil
		}
	}
	// Then try bare "token" key (original single-key format)
	data, err := creds.Load("token")
	if err != nil {
		return "", err
	}
	var token string
	if json.Unmarshal(data, &token) == nil {
		return token, nil
	}
	return string(data), nil
}

// resolveProfile uses profile.Resolve() to determine the active profile,
// then applies its BaseURL and board (from Extra) to cfg.
//
// Profile settings (layer 3) outrank local and global YAML config (layers
// 4–5) but yield to env vars (layer 2) and flags (layer 1). Because
// config.Load() has already run, cfg may contain values from YAML config.
// resolveProfile intentionally overwrites those with profile data:
//
//	profile BaseURL  → cfg.APIURL (unless FIZZY_API_URL env var is set)
//	profile board    → cfg.Board  (unless FIZZY_BOARD env var is set)
//
// Returns an error when the user explicitly selected a profile (via flag
// or env var) that doesn't exist — that must be a hard failure, not a
// silent fallback to whatever was in the YAML config.
func resolveProfile() error {
	if profiles == nil {
		// No profile store (test mode or init failure) — fall back to env var
		if p := os.Getenv("FIZZY_PROFILE"); p != "" {
			cfg.Account = p
		}
		return nil
	}

	allProfiles, defaultName, err := profiles.List()
	if err != nil || len(allProfiles) == 0 {
		// No profiles configured — fall back to env var for account
		if v := profileEnvVar(); v != "" {
			cfg.Account = v
		}
		return nil
	}

	envProfile := profileEnvVar()
	resolved, err := profile.Resolve(profile.ResolveOptions{
		FlagValue:      cfgProfile,
		EnvVar:         envProfile,
		DefaultProfile: defaultName,
		Profiles:       allProfiles,
	})
	if err != nil {
		// If the user explicitly specified a profile (flag or env), that's a
		// hard error — don't silently fall back to a different account.
		if cfgProfile != "" || envProfile != "" {
			return err
		}
		// Otherwise (ambiguous default, etc.) — not fatal, just skip profile
		return nil
	}
	if resolved == "" {
		return nil
	}

	p := allProfiles[resolved]
	if p == nil {
		return nil
	}

	// Apply profile settings to cfg — but only for fields that haven't
	// already been set by a higher-precedence source (env var).
	cfg.Account = resolved
	if p.BaseURL != "" && os.Getenv("FIZZY_API_URL") == "" {
		cfg.APIURL = p.BaseURL
	}
	if boardRaw, ok := p.Extra["board"]; ok {
		var board string
		if json.Unmarshal(boardRaw, &board) == nil && board != "" && os.Getenv("FIZZY_BOARD") == "" {
			cfg.Board = board
		}
	}
	return nil
}

// profileEnvVar returns the FIZZY_PROFILE env var, falling back to FIZZY_ACCOUNT
// for backward compatibility.
func profileEnvVar() string {
	if v := os.Getenv("FIZZY_PROFILE"); v != "" {
		return v
	}
	if v := os.Getenv("FIZZY_ACCOUNT"); v != "" {
		fmt.Fprintln(os.Stderr, "Warning: FIZZY_ACCOUNT is deprecated, use FIZZY_PROFILE instead")
		return v
	}
	return ""
}

// resolveToken applies token precedence: YAML → credstore (with migration) → env → flag.
func resolveToken() {
	// 1. YAML file (global + local, already in cfg.Token from config.Load())
	// 2. credstore (overrides YAML — credstore is the "new" storage)
	if creds != nil {
		profileName := cfg.Account // profile name = account slug

		if profileName != "" {
			// Try profile-scoped token first
			if t, err := credsLoadProfileToken(profileName); err == nil && t != "" {
				cfg.Token = t
			} else {
				// Legacy migration: old keys → profile-scoped key
				migrateLegacyToken(profileName)
			}
		} else {
			// No profile — try legacy single key
			if t, err := credsLoadLegacyToken(""); err == nil && t != "" {
				cfg.Token = t
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

// migrateLegacyToken moves a token from legacy storage to profile-scoped storage.
// Handles the old single-key credstore entry, account-scoped keys, and YAML config tokens.
func migrateLegacyToken(profileName string) {
	// Check legacy credstore keys — copy to profile-scoped key but keep the
	// legacy keys so older CLI versions still work after a downgrade.
	if t, err := credsLoadLegacyToken(profileName); err == nil && t != "" {
		// Always use the token, even if migration to profile-scoped key fails
		cfg.Token = t
		if err := credsSaveProfileToken(profileName, t); err == nil {
			ensureProfile(profileName, cfg.APIURL, "")
		}
		return
	}

	// Check YAML config token (pre-credstore migration)
	globalCfg := config.LoadGlobal()
	if globalCfg.Token != "" {
		// Always use the token, even if migration to profile-scoped key fails
		cfg.Token = globalCfg.Token
		if err := credsSaveProfileToken(profileName, globalCfg.Token); err == nil {
			globalCfg.Token = ""
			globalCfg.Account = profileName
			_ = globalCfg.Save()
			ensureProfile(profileName, cfg.APIURL, "")
		}
	}
}

// ensureProfile creates or updates a profile in the store.
// If the profile already exists, fields are merged: BaseURL is
// preserved only when the caller passes an empty string (meaning
// "keep whatever is there"), and Extra entries are preserved unless
// explicitly replaced.
func ensureProfile(name, baseURL, board string) {
	if profiles == nil {
		return
	}

	existing, _ := profiles.Get(name)

	newBaseURL := baseURL
	if newBaseURL == "" {
		if existing != nil && existing.BaseURL != "" {
			newBaseURL = existing.BaseURL
		} else {
			newBaseURL = config.DefaultAPIURL
		}
	}

	extra := map[string]json.RawMessage{}
	if existing != nil {
		for k, v := range existing.Extra {
			extra[k] = v
		}
	}
	if board != "" {
		extra["board"] = func() json.RawMessage { b, _ := json.Marshal(board); return b }()
	}

	p := &profile.Profile{
		Name:    name,
		BaseURL: newBaseURL,
	}
	if len(extra) > 0 {
		p.Extra = extra
	}

	if err := profiles.Create(p); err != nil {
		_ = profiles.Delete(name)
		_ = profiles.Create(p)
	}
}

// SetTestSDK configures the commands package for SDK-based testing.
// Pass an httptest.Server URL and the SDK will be created pointing at it.
func SetTestSDK(baseURL string) {
	sdkCfg := &fizzy.Config{BaseURL: baseURL}
	sdk = fizzy.NewClient(sdkCfg, &fizzy.StaticTokenProvider{Token: "test-token"})
	sdkAccount = func() *fizzy.AccountClient {
		return sdk.ForAccount("test-account")
	}
}

// SetTestCreds sets the credential store for testing.
func SetTestCreds(store *credstore.Store) {
	creds = store
}

// SetTestProfiles sets the profile store for testing.
func SetTestProfiles(store *profile.Store) {
	profiles = store
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
	sdk = nil
	sdkAccount = nil
	errSDKInit = nil
	lastResult = nil
	lastRawOutput = ""
	errOutputWrite = nil
	cfg = nil
	creds = nil
	profiles = nil
	cfgJSON = false
	cfgQuiet = false
	cfgIDsOnly = false
	cfgCount = false
	cfgAgent = false
	cfgStyled = false
	cfgMarkdown = false
	cfgLimit = 0
	cfgJQ = ""
	cfgProfile = ""
}

// GetRootCmd returns the root command for testing.
func GetRootCmd() *cobra.Command {
	return rootCmd
}

// Helper function for required flag errors
func newRequiredFlagError(flag string) error {
	return errors.NewInvalidArgsError("required flag --" + flag + " not provided")
}
