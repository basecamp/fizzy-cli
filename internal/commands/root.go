// Package commands implements CLI commands for the Fizzy CLI.
package commands

import (
	"bytes"
	"encoding/json"
	stderrors "errors"
	"os"

	"github.com/basecamp/cli/output"
	"github.com/basecamp/fizzy-cli/internal/client"
	"github.com/basecamp/fizzy-cli/internal/config"
	"github.com/basecamp/fizzy-cli/internal/errors"
	"github.com/spf13/cobra"
)

// Breadcrumb is a type alias for output.Breadcrumb.
type Breadcrumb = output.Breadcrumb

var (
	// Global flags
	cfgToken   string
	cfgAccount string
	cfgAPIURL  string
	cfgVerbose bool
	cfgPretty  bool

	// Loaded config
	cfg *config.Config

	// Client factory (can be overridden for testing)
	clientFactory func() client.API

	// Output writer
	out *output.Writer
)

// rootCmd represents the base command.
var rootCmd = &cobra.Command{
	Use:   "fizzy",
	Short: "Fizzy CLI - Command-line interface for the Fizzy API",
	Long: `A command-line interface for the Fizzy API.

Use fizzy to manage boards, cards, comments, and more from your terminal.`,
	Version: "dev",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// In test mode, cfg is already set by SetTestConfig - don't overwrite
		if cfg == nil {
			// Load config from file/env
			cfg = config.Load()
		}

		// Override with command-line flags
		if cfgToken != "" {
			cfg.Token = cfgToken
		}
		if cfgAccount != "" {
			cfg.Account = cfgAccount
		}
		if cfgAPIURL != "" {
			cfg.APIURL = cfgAPIURL
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
	out = output.New(output.Options{Format: output.FormatJSON, Writer: os.Stdout})
	if err := rootCmd.Execute(); err != nil {
		var e *output.Error
		if !stderrors.As(err, &e) {
			// Cobra-level errors (arg count, unknown flag) → usage
			e = &output.Error{Code: output.CodeUsage, Message: err.Error()}
		}
		out.Err(e)
		os.Exit(e.ExitCode())
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgToken, "token", "", "API access token")
	rootCmd.PersistentFlags().StringVar(&cfgAccount, "account", "", "Account slug")
	rootCmd.PersistentFlags().StringVar(&cfgAPIURL, "api-url", "", "API base URL")
	rootCmd.PersistentFlags().BoolVar(&cfgVerbose, "verbose", false, "Show request/response details")
	rootCmd.PersistentFlags().BoolVar(&cfgPretty, "pretty", false, "Pretty-print JSON output with indentation")
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

// captureResponse parses the writer buffer into lastResult after each shim call.
func captureResponse() {
	if lastResult == nil {
		return
	}
	var resp output.Response
	json.Unmarshal(testBuf.Bytes(), &resp)
	lastResult.Response = &resp
	testBuf.Reset()
}

// printSuccess prints a success response.
func printSuccess(data interface{}) {
	out.OK(data)
	captureResponse()
}

// printSuccessWithLocation prints a success response with location.
func printSuccessWithLocation(location string) {
	out.OK(nil, output.WithContext("location", location))
	captureResponse()
}

// breadcrumb creates a single breadcrumb.
func breadcrumb(action, cmd, description string) Breadcrumb {
	return Breadcrumb{Action: action, Cmd: cmd, Description: description}
}

// printSuccessWithBreadcrumbs prints a success response with breadcrumbs.
func printSuccessWithBreadcrumbs(data interface{}, summary string, breadcrumbs []Breadcrumb) {
	opts := []output.ResponseOption{output.WithBreadcrumbs(breadcrumbs...)}
	if summary != "" {
		opts = append(opts, output.WithSummary(summary))
	}
	out.OK(data, opts...)
	captureResponse()
}

// printSuccessWithPaginationAndBreadcrumbs prints a success response with pagination and breadcrumbs.
func printSuccessWithPaginationAndBreadcrumbs(data interface{}, hasNext bool, nextURL string, summary string, breadcrumbs []Breadcrumb) {
	opts := []output.ResponseOption{output.WithBreadcrumbs(breadcrumbs...)}
	if summary != "" {
		opts = append(opts, output.WithSummary(summary))
	}
	if hasNext || nextURL != "" {
		opts = append(opts, output.WithContext("pagination", map[string]interface{}{
			"has_next": hasNext,
			"next_url": nextURL,
		}))
	}
	out.OK(data, opts...)
	captureResponse()
}

// printSuccessWithLocationAndBreadcrumbs prints a success response with both location and breadcrumbs.
func printSuccessWithLocationAndBreadcrumbs(data interface{}, location string, breadcrumbs []Breadcrumb) {
	out.OK(data,
		output.WithBreadcrumbs(breadcrumbs...),
		output.WithContext("location", location),
	)
	captureResponse()
}

// SetTestMode configures the commands package for testing.
// It sets a mock client factory and captures results instead of exiting.
func SetTestMode(mockClient client.API) *CommandResult {
	clientFactory = func() client.API {
		return mockClient
	}
	testBuf.Reset()
	out = output.New(output.Options{Format: output.FormatJSON, Writer: &testBuf})
	lastResult = &CommandResult{}
	return lastResult
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
	cfg = nil
}

// GetRootCmd returns the root command for testing.
func GetRootCmd() *cobra.Command {
	return rootCmd
}

// Helper function for required flag errors
func newRequiredFlagError(flag string) error {
	return errors.NewInvalidArgsError("required flag --" + flag + " not provided")
}
