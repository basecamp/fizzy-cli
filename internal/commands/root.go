// Package commands implements CLI commands for the Fizzy CLI.
package commands

import (
	stderrors "errors"
	"os"

	"github.com/basecamp/fizzy-cli/internal/client"
	"github.com/basecamp/fizzy-cli/internal/config"
	"github.com/basecamp/fizzy-cli/internal/errors"
	"github.com/basecamp/fizzy-cli/internal/response"
	"github.com/spf13/cobra"
)

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
)

// rootCmd represents the base command.
var rootCmd = &cobra.Command{
	Use:   "fizzy",
	Short: "Fizzy CLI - Command-line interface for the Fizzy API",
	Long: `A command-line interface for the Fizzy API.

Use fizzy to manage boards, cards, comments, and more from your terminal.`,
	Version: "dev",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
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

		// Set response formatting
		response.SetPrettyPrint(cfgPretty)
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
	if err := rootCmd.Execute(); err != nil {
		var cliErr *errors.CLIError
		if stderrors.As(err, &cliErr) {
			response.Error(cliErr).PrintAndExit()
		}
		response.Error(errors.NewError(err.Error())).PrintAndExit()
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
	Response *response.Response
	ExitCode int
}

// lastResult stores the last command result (for testing)
var lastResult *CommandResult

// testExitSignal is used to stop command execution in test mode
type testExitSignal struct{}

// exitWithError prints an error response and exits.
func exitWithError(err error) {
	var resp *response.Response
	var cliErr *errors.CLIError
	if stderrors.As(err, &cliErr) {
		resp = response.Error(cliErr)
	} else {
		resp = response.Error(errors.NewError(err.Error()))
	}

	if lastResult != nil {
		lastResult.Response = resp
		lastResult.ExitCode = resp.ExitCode()
		panic(testExitSignal{}) // Signal to stop execution in test mode
	}
	resp.PrintAndExit()
}

// printSuccess prints a success response.
func printSuccess(data interface{}) {
	resp := response.Success(data)
	if lastResult != nil {
		lastResult.Response = resp
		lastResult.ExitCode = errors.ExitSuccess
		panic(testExitSignal{}) // Signal to stop execution in test mode
	}
	resp.Print()
	os.Exit(errors.ExitSuccess)
}

// printSuccessWithLocation prints a success response with location.
func printSuccessWithLocation(location string) {
	resp := response.SuccessWithLocation(nil, location)
	if lastResult != nil {
		lastResult.Response = resp
		lastResult.ExitCode = errors.ExitSuccess
		panic(testExitSignal{}) // Signal to stop execution in test mode
	}
	resp.Print()
	os.Exit(errors.ExitSuccess)
}

// breadcrumb creates a single breadcrumb.
func breadcrumb(action, cmd, description string) response.Breadcrumb {
	return response.NewBreadcrumb(action, cmd, description)
}

// printSuccessWithBreadcrumbs prints a success response with breadcrumbs.
func printSuccessWithBreadcrumbs(data interface{}, summary string, breadcrumbs []response.Breadcrumb) {
	resp := response.SuccessWithBreadcrumbs(data, summary, breadcrumbs)
	if lastResult != nil {
		lastResult.Response = resp
		lastResult.ExitCode = errors.ExitSuccess
		panic(testExitSignal{}) // Signal to stop execution in test mode
	}
	resp.Print()
	os.Exit(errors.ExitSuccess)
}

// printSuccessWithPaginationAndBreadcrumbs prints a success response with pagination and breadcrumbs.
func printSuccessWithPaginationAndBreadcrumbs(data interface{}, hasNext bool, nextURL string, summary string, breadcrumbs []response.Breadcrumb) {
	resp := response.SuccessWithPaginationAndBreadcrumbs(data, hasNext, nextURL, summary, breadcrumbs)
	if lastResult != nil {
		lastResult.Response = resp
		lastResult.ExitCode = errors.ExitSuccess
		panic(testExitSignal{}) // Signal to stop execution in test mode
	}
	resp.Print()
	os.Exit(errors.ExitSuccess)
}

// SetTestMode configures the commands package for testing.
// It sets a mock client factory and captures results instead of exiting.
func SetTestMode(mockClient client.API) *CommandResult {
	clientFactory = func() client.API {
		return mockClient
	}
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

// RunTestCommand executes a command in test mode, recovering from panics.
// This is used to safely run commands that would normally call os.Exit.
func RunTestCommand(fn func()) {
	defer func() {
		if r := recover(); r != nil {
			// Check if it's our test exit signal
			if _, ok := r.(testExitSignal); !ok {
				// Re-panic if it's a real error
				panic(r)
			}
			// Otherwise, the command exited normally via printSuccess/exitWithError
		}
	}()
	fn()
}
