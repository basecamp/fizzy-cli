package commands

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/basecamp/cli/output"
	"github.com/basecamp/fizzy-cli/internal/config"
	"github.com/basecamp/fizzy-cli/internal/errors"
	"github.com/basecamp/fizzy-cli/internal/tui"
	"github.com/charmbracelet/huh"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

var signupCmd = &cobra.Command{
	Use:   "signup",
	Short: "Sign up or generate an access token",
	Long:  "Interactive signup wizard. Use subcommands (start, verify, complete) for programmatic access.",
	RunE:  runSignup,
}

var signupStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Request a magic link code via email",
	RunE:  runSignupStart,
}

var signupVerifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Verify the magic link code",
	RunE:  runSignupVerify,
}

var signupCompleteCmd = &cobra.Command{
	Use:   "complete",
	Short: "Complete signup and generate an access token",
	RunE:  runSignupComplete,
}

func init() {
	rootCmd.AddCommand(signupCmd)
	signupCmd.AddCommand(signupStartCmd)
	signupCmd.AddCommand(signupVerifyCmd)
	signupCmd.AddCommand(signupCompleteCmd)

	signupStartCmd.Flags().String("email", "", "Email address")
	_ = signupStartCmd.MarkFlagRequired("email")

	signupVerifyCmd.Flags().String("code", "", "Magic link code from email")
	signupVerifyCmd.Flags().String("pending-token", "", "Pending authentication token from start step")
	_ = signupVerifyCmd.MarkFlagRequired("code")
	_ = signupVerifyCmd.MarkFlagRequired("pending-token")

	signupCompleteCmd.Flags().String("name", "", "Full name (required for new users)")
	signupCompleteCmd.Flags().String("account", "", "Account slug (required for existing users)")
}

const welcomeSignature = `
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢠⠏⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣠⠏⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣀⡤⠖⠚⠉⠉
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⡼⠁⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⡤⠖⠋⠁
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢀⡴⠋⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢠⠞⠉
⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣠⠖⠋⠀⢀⠀⠀⠀⠉⠉⠉⠉⠉⢉⡽⠋⢉⣉⠉⠉⠉⠉⠉⡉⠉⠉⠉⠉
⠀⠀⠀⠀⠀⠀⠀⠀⢀⣠⠴⠋⢁⡴⠯⠞⠙⣦⠞⠙⠦⠤⠤⠤⢠⠞⣀⡴⠋⠈⠋⠙⠒⠒⠚⠁
⠀⠀⠀⠀⢀⣠⠤⠖⠉⠀⠀⠀⠈⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⢠⠏⠞⠁
⠤⠤⠖⠚⠉⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⡞
`

const welcomeMessage = `Welcome, and thanks for signing up for Fizzy.

To get you started, try "fizzy board list" to see your boards. You
should find a Playground board with a few cards designed to help you
learn Fizzy itself — open each card, go through the simple steps, and
you'll be an expert in no time.

If you ever need a hand, please contact me directly at
jason@37signals.com. I'm here for you, we're all here for you.

Thanks again and all the best,`

const welcomeSignoff = `Jason Fried, jason@37signals.com
CEO & co-founder of 37signals, makers of Fizzy, Basecamp, and HEY`

// printWelcomeMessage prints the CEO welcome message for new users.
func printWelcomeMessage() {
	fmt.Println()
	fmt.Println(welcomeMessage)
	fmt.Print(welcomeSignature)
	fmt.Println(welcomeSignoff)
	fmt.Println()
}

// runSignup is the interactive wizard that walks through the entire signup flow.
func runSignup(cmd *cobra.Command, args []string) error {
	if IsMachineOutput() {
		return output.ErrUsageHint("signup requires an interactive terminal — use subcommands (start, verify, complete) for programmatic access", "Run without --agent/--json/--quiet or in a TTY")
	}

	aw, wait := tui.AnimateBannerAsync(cmd.ErrOrStderr())
	fmt.Fprintln(aw)
	fmt.Fprintln(aw, "Welcome to Fizzy CLI signup!")
	fmt.Fprintln(aw)
	wait()

	return signupWizard()
}

// signupWizard contains the core interactive signup flow (steps 1-9).
// Called by both runSignup and runSetup to avoid duplicate preamble.
func signupWizard() error {
	// Step 1: Hosting type
	var hostingType string
	err := huh.NewSelect[string]().
		Title("Are you using the hosted or self-hosted version?").
		Options(
			huh.NewOption("Hosted (app.fizzy.do)", "hosted"),
			huh.NewOption("Self-hosted", "selfhosted"),
		).
		Value(&hostingType).
		Run()

	if err != nil {
		fmt.Println("Signup cancelled.")
		return nil //nolint:nilerr // user cancelled prompt
	}

	apiURL := config.DefaultAPIURL
	if hostingType == "selfhosted" {
		err = huh.NewInput().
			Title("Enter your Fizzy URL").
			Placeholder("https://fizzy.example.com").
			Value(&apiURL).
			Validate(func(s string) error {
				if s == "" {
					return fmt.Errorf("URL is required")
				}
				return validateSignupURL(s)
			}).
			Run()

		if err != nil {
			fmt.Println("Signup cancelled.")
			return nil //nolint:nilerr // user cancelled prompt
		}

		apiURL = strings.TrimSuffix(apiURL, "/")
	}

	httpClient := newSignupHTTPClient()

	// Step 2: Email
	var email string
	err = huh.NewInput().
		Title("Enter your email address").
		Placeholder("you@example.com").
		Value(&email).
		Validate(func(s string) error {
			if s == "" {
				return fmt.Errorf("email is required")
			}
			if !strings.Contains(s, "@") {
				return fmt.Errorf("invalid email address")
			}
			return nil
		}).
		Run()

	if err != nil {
		fmt.Println("Signup cancelled.")
		return nil //nolint:nilerr // user cancelled prompt
	}

	// Step 3: Request magic link
	fmt.Print("Sending magic link... ")
	_, respHeader, err := signupPost(httpClient, apiURL+"/session.json", map[string]any{
		"email_address": email,
	})
	if err != nil {
		fmt.Println("✗")
		return errors.NewError(fmt.Sprintf("Failed to send magic link: %v", err))
	}
	fmt.Println("✓")

	// Development servers include the magic link code in a response header
	if devCode := respHeader.Get("X-Magic-Link-Code"); devCode != "" {
		fmt.Printf("Development code: %s\n", devCode)
	} else {
		fmt.Println("Check your email for a 6-digit code.")
	}
	fmt.Println()

	// Step 4: Code verification loop
	var requiresCompletion bool

	for {
		var code string
		err = huh.NewInput().
			Title("Enter the 6-digit code from your email").
			Placeholder("ABC123").
			Value(&code).
			Validate(func(s string) error {
				if s == "" {
					return fmt.Errorf("code is required")
				}
				return nil
			}).
			Run()

		if err != nil {
			fmt.Println("Signup cancelled.")
			return nil //nolint:nilerr // user cancelled prompt
		}

		fmt.Print("Verifying code... ")
		data, _, verifyErr := signupPost(httpClient, apiURL+"/session/magic_link.json", map[string]any{
			"code": code,
		})
		if verifyErr != nil {
			fmt.Println("✗")

			// Only offer retry for authentication errors (wrong code)
			var he *signupHTTPError
			if stderrors.As(verifyErr, &he) && he.statusCode == http.StatusUnauthorized {
				var retry bool
				if confirmErr := huh.NewConfirm().
					Title("Invalid code. Would you like to try again?").
					Value(&retry).
					Run(); confirmErr != nil {
					retry = false
				}

				if !retry {
					fmt.Println("Signup cancelled.")
					return nil
				}
				continue
			}

			return errors.NewError(fmt.Sprintf("Failed to verify code: %v", verifyErr))
		}

		fmt.Println("✓")
		requiresCompletion, _ = data["requires_signup_completion"].(bool)
		break
	}

	// The cookie jar already has the signed session_token cookie from the verify response.

	// Step 5: Complete signup if new user
	if requiresCompletion {
		var fullName string
		err = huh.NewInput().
			Title("Enter your full name").
			Placeholder("Jane Smith").
			Value(&fullName).
			Validate(func(s string) error {
				if s == "" {
					return fmt.Errorf("name is required")
				}
				return nil
			}).
			Run()

		if err != nil {
			fmt.Println("Signup cancelled.")
			return nil //nolint:nilerr // user cancelled prompt
		}

		fmt.Print("Completing signup... ")
		_, _, err = signupPost(httpClient, apiURL+"/signup/completion.json", map[string]any{
			"signup": map[string]any{
				"full_name": fullName,
			},
		})
		if err != nil {
			fmt.Println("✗")
			return errors.NewError(fmt.Sprintf("Failed to complete signup: %v", err))
		}
		fmt.Println("✓")
	}

	// Step 6: Fetch accounts
	fmt.Print("Fetching accounts... ")
	identityData, err := signupGet(httpClient, apiURL+"/my/identity.json")
	if err != nil {
		fmt.Println("✗")
		return errors.NewError(fmt.Sprintf("Failed to fetch accounts: %v", err))
	}
	fmt.Println("✓")

	identityJSON, marshalErr := json.Marshal(identityData)
	if marshalErr != nil {
		return errors.NewError(fmt.Sprintf("Failed to process identity data: %v", marshalErr))
	}
	accounts, err := parseAccounts(json.RawMessage(identityJSON))
	if err != nil || len(accounts) == 0 {
		return errors.NewError("No accounts found")
	}

	// Step 7: Account selection
	var selectedAccountSlug string
	if len(accounts) == 1 {
		selectedAccountSlug = accounts[0].Slug
		fmt.Printf("Using account: %s (%s)\n", accounts[0].Name, accounts[0].Slug)
	} else {
		accountOptions := make([]huh.Option[string], len(accounts))
		for i, acc := range accounts {
			accountOptions[i] = huh.NewOption(fmt.Sprintf("%s (%s)", acc.Name, acc.Slug), acc.Slug)
		}

		err = huh.NewSelect[string]().
			Title("Select your account").
			Options(accountOptions...).
			Value(&selectedAccountSlug).
			Run()

		if err != nil {
			fmt.Println("Signup cancelled.")
			return nil //nolint:nilerr // user cancelled prompt
		}
	}

	// Step 8: Generate access token
	fmt.Print("Generating access token... ")
	tokenData, _, err := signupPost(httpClient, apiURL+"/"+url.PathEscape(selectedAccountSlug)+"/my/access_tokens.json", map[string]any{
		"access_token": map[string]any{
			"description": "Fizzy CLI",
			"permission":  "write",
		},
	})
	if err != nil {
		fmt.Println("✗")
		return errors.NewError(fmt.Sprintf("Failed to generate access token: %v", err))
	}
	fmt.Println("✓")

	token, _ := tokenData["token"].(string)
	if token == "" {
		return errors.NewError("Server did not return an access token")
	}

	// Step 9: Save config
	if err := saveSignupConfig(token, selectedAccountSlug, apiURL); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("✓ Configuration saved.")

	if requiresCompletion {
		printWelcomeMessage()
	} else {
		fmt.Println()
		fmt.Println("You're all set! Try: fizzy board list")
	}
	return nil
}

// runSignupStart handles `fizzy signup start --email user@example.com`.
func runSignupStart(cmd *cobra.Command, args []string) error {
	email, _ := cmd.Flags().GetString("email")
	apiURL := signupAPIURL()

	httpClient := newSignupHTTPClient()
	_, respHeader, err := signupPost(httpClient, apiURL+"/session.json", map[string]any{
		"email_address": email,
	})
	if err != nil {
		return errors.NewError(fmt.Sprintf("Failed to request magic link: %v", err))
	}

	// Extract the signed cookie value (not the JSON body value)
	pendingToken := getCookieValue(httpClient, apiURL, "pending_authentication_token")
	if pendingToken == "" {
		return errors.NewError("Server did not return a pending_authentication_token cookie")
	}

	breadcrumbs := []Breadcrumb{
		breadcrumb("verify", "fizzy signup verify --code <CODE> --pending-token <TOKEN>", "Verify magic link code"),
	}

	data := map[string]any{
		"pending_authentication_token": pendingToken,
	}

	// Development servers include the magic link code in a response header
	if code := respHeader.Get("X-Magic-Link-Code"); code != "" {
		data["code"] = code
	}

	printMutation(signupHumanData(data, "code"), "Magic link sent. Check your email for a 6-digit code.", breadcrumbs)
	return nil
}

// runSignupVerify handles `fizzy signup verify --code ABC123 --pending-token eyJ...`.
func runSignupVerify(cmd *cobra.Command, args []string) error {
	code, _ := cmd.Flags().GetString("code")
	pendingToken, _ := cmd.Flags().GetString("pending-token")
	apiURL := signupAPIURL()

	httpClient := newSignupHTTPClient()

	// Set the pending_authentication_token as a cookie (already signed value from start step)
	setSignedCookie(httpClient, apiURL, "pending_authentication_token", pendingToken)

	data, _, err := signupPost(httpClient, apiURL+"/session/magic_link.json", map[string]any{
		"code": code,
	})
	if err != nil {
		return errors.NewError(fmt.Sprintf("Failed to verify code: %v", err))
	}

	// Extract the signed session_token cookie (not the JSON body value)
	sessionCookie := getCookieValue(httpClient, apiURL, "session_token")
	if sessionCookie == "" {
		return errors.NewError("Server did not return a session_token cookie")
	}

	result := map[string]any{
		"session_token":              sessionCookie,
		"requires_signup_completion": data["requires_signup_completion"],
	}

	// For existing users, fetch accounts using the same client (jar has session cookie)
	requiresCompletion, _ := data["requires_signup_completion"].(bool)
	if !requiresCompletion {
		identityData, err := signupGet(httpClient, apiURL+"/my/identity.json")
		if err != nil {
			return errors.NewError(fmt.Sprintf("Failed to fetch accounts: %v", err))
		}
		if accountsList, ok := identityData["accounts"].([]any); ok {
			result["accounts"] = normalizeAccountSlugs(accountsList)
		}
	}

	printDetail(signupHumanData(result, "requires_signup_completion", "accounts"), "", nil)
	return nil
}

// runSignupComplete handles `fizzy signup complete`. The session token is read
// from stdin (piped or interactive prompt) to avoid exposing it in shell history.
func runSignupComplete(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	account, _ := cmd.Flags().GetString("account")
	apiURL := signupAPIURL()

	sessionToken, err := readSessionToken()
	if err != nil {
		return errors.NewError(fmt.Sprintf("Failed to read session token: %v", err))
	}
	if sessionToken == "" {
		return errors.NewInvalidArgsError("Session token is required (pipe via stdin or enter when prompted)")
	}

	if name == "" && account == "" {
		return errors.NewInvalidArgsError("Either --name (new user) or --account (existing user) is required")
	}

	if name != "" && account != "" {
		return errors.NewInvalidArgsError("Use --name for new users or --account for existing users, not both")
	}

	// Normalize account slug (strip leading "/" if present)
	account = strings.TrimPrefix(account, "/")

	httpClient := newSignupHTTPClient()
	setSessionCookie(httpClient, apiURL, sessionToken)

	var accountSlug string

	if name != "" {
		// New user: complete signup first
		_, _, err = signupPost(httpClient, apiURL+"/signup/completion.json", map[string]any{
			"signup": map[string]any{
				"full_name": name,
			},
		})
		if err != nil {
			return errors.NewError(fmt.Sprintf("Failed to complete signup: %v", err))
		}

		// Fetch the new account
		var identityData map[string]any
		identityData, err = signupGet(httpClient, apiURL+"/my/identity.json")
		if err != nil {
			return errors.NewError(fmt.Sprintf("Failed to fetch account: %v", err))
		}

		var accounts []Account
		identityJSON, marshalErr := json.Marshal(identityData)
		if marshalErr != nil {
			return errors.NewError(fmt.Sprintf("Failed to process identity data: %v", marshalErr))
		}
		accounts, err = parseAccounts(json.RawMessage(identityJSON))
		if err != nil || len(accounts) == 0 {
			return errors.NewError("No accounts found after signup")
		}
		accountSlug = accounts[0].Slug
	} else {
		// Existing user: use provided account
		accountSlug = account
	}

	// Generate access token
	tokenData, _, err := signupPost(httpClient, apiURL+"/"+url.PathEscape(accountSlug)+"/my/access_tokens.json", map[string]any{
		"access_token": map[string]any{
			"description": "Fizzy CLI",
			"permission":  "write",
		},
	})
	if err != nil {
		return errors.NewError(fmt.Sprintf("Failed to generate access token: %v", err))
	}

	token, _ := tokenData["token"].(string)
	if token == "" {
		return errors.NewError("Server did not return an access token")
	}

	// Save config
	if err := saveSignupConfig(token, accountSlug, apiURL); err != nil {
		return err
	}

	result := map[string]any{
		"token":   token,
		"account": accountSlug,
	}

	summary := "Configuration saved."

	if name != "" {
		result["welcome_message"] = welcomeMessage + welcomeSignature + welcomeSignoff
		result["is_new_user"] = true
	}

	breadcrumbs := []Breadcrumb{
		breadcrumb("boards", "fizzy board list", "List boards"),
		breadcrumb("setup", "fizzy setup", "Full interactive setup"),
	}

	printMutation(signupHumanData(result, "account", "is_new_user"), summary, breadcrumbs)
	return nil
}

// signupHumanData removes machine-only secrets from human-facing output while
// preserving the full payload for JSON/quiet/agent workflows.
func signupHumanData(data map[string]any, fields ...string) map[string]any {
	if !isHumanOutput() {
		return data
	}

	result := make(map[string]any, len(fields))
	for _, field := range fields {
		if value, ok := data[field]; ok {
			result[field] = value
		}
	}
	return result
}

// signupAPIURL returns the effective API URL for signup commands, normalized without trailing slash.
func signupAPIURL() string {
	if cfg != nil && cfg.APIURL != "" {
		return strings.TrimSuffix(cfg.APIURL, "/")
	}
	return strings.TrimSuffix(config.DefaultAPIURL, "/")
}

// newSignupHTTPClient creates an HTTP client with a cookie jar for session-based auth.
// Redirects are validated to prevent SSRF via open redirectors.
func newSignupHTTPClient() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{
		Timeout: 30 * time.Second,
		Jar:     jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return validateSignupURL(req.URL.String())
		},
	}
}

// signupHTTPError is returned by signupPost for HTTP error responses (status >= 400),
// allowing callers to distinguish auth errors from operational failures.
type signupHTTPError struct {
	statusCode int
	message    string
}

func (e *signupHTTPError) Error() string { return e.message }

// signupMaxResponseBytes is the maximum response body size for signup HTTP requests (1MB).
const signupMaxResponseBytes = 1 << 20

// signupPost makes a JSON POST request and returns the parsed response body and headers.
func signupPost(client *http.Client, reqURL string, body any) (map[string]any, http.Header, error) {
	if err := validateSignupURL(reqURL); err != nil {
		return nil, nil, err
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal body: %w", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), "POST", reqURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("fizzy-cli/%s", rootCmd.Version))

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, signupMaxResponseBytes))
	if err != nil {
		return nil, resp.Header, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errData struct {
			Error   string   `json:"error"`
			Message string   `json:"message"`
			Errors  []string `json:"errors"`
		}
		if json.Unmarshal(respBody, &errData) == nil {
			if errData.Error != "" {
				return nil, resp.Header, &signupHTTPError{statusCode: resp.StatusCode, message: errData.Error}
			}
			if errData.Message != "" {
				return nil, resp.Header, &signupHTTPError{statusCode: resp.StatusCode, message: errData.Message}
			}
			if len(errData.Errors) > 0 {
				return nil, resp.Header, &signupHTTPError{statusCode: resp.StatusCode, message: strings.Join(errData.Errors, ", ")}
			}
		}
		return nil, resp.Header, &signupHTTPError{statusCode: resp.StatusCode, message: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody))}
	}

	if len(respBody) == 0 {
		return map[string]any{}, resp.Header, nil
	}

	var data map[string]any
	if err := json.Unmarshal(respBody, &data); err != nil {
		return nil, resp.Header, fmt.Errorf("failed to parse response: %w", err)
	}

	return data, resp.Header, nil
}

// signupGet makes a GET request and returns the parsed response body.
func signupGet(client *http.Client, reqURL string) (map[string]any, error) {
	if err := validateSignupURL(reqURL); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(context.Background(), "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("fizzy-cli/%s", rootCmd.Version))

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, signupMaxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, &signupHTTPError{statusCode: resp.StatusCode, message: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody))}
	}

	var data map[string]any
	if err := json.Unmarshal(respBody, &data); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return data, nil
}

// setSessionCookie sets the session_token cookie on the HTTP client's cookie jar.
func setSessionCookie(client *http.Client, apiURL, sessionToken string) {
	u, err := url.Parse(apiURL)
	if err != nil {
		return
	}
	client.Jar.SetCookies(u, []*http.Cookie{
		{Name: "session_token", Value: sessionToken, Secure: u.Scheme == "https", HttpOnly: true},
	})
}

// setSignedCookie sets a named cookie on the HTTP client's cookie jar.
func setSignedCookie(client *http.Client, apiURL, name, value string) {
	u, err := url.Parse(apiURL)
	if err != nil {
		return
	}
	client.Jar.SetCookies(u, []*http.Cookie{
		{Name: name, Value: value, Secure: u.Scheme == "https", HttpOnly: true},
	})
}

// validateSignupURL checks that a URL is safe to use for signup HTTP requests.
// Only http:// and https:// schemes are allowed. Plain http:// is restricted to
// literal loopback addresses (localhost, 127.0.0.1, [::1]) to prevent SSRF
// against internal network services. No DNS resolution is performed to avoid
// TOCTOU races between validation and connection time.
func validateSignupURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	switch u.Scheme {
	case "https":
		return nil
	case "http":
		host := u.Hostname()
		if host == "localhost" || host == "127.0.0.1" || host == "::1" {
			return nil
		}
		return fmt.Errorf("http:// is only allowed for localhost; use https:// for remote hosts")
	default:
		return fmt.Errorf("unsupported URL scheme %q; use https://", u.Scheme)
	}
}

// normalizeAccountSlugs strips leading "/" from account slugs in the identity response
// so they match the format expected by --account flags and URL construction.
func normalizeAccountSlugs(accounts []any) []any {
	for _, acc := range accounts {
		if m, ok := acc.(map[string]any); ok {
			if slug, ok := m["slug"].(string); ok {
				m["slug"] = strings.TrimPrefix(slug, "/")
			}
		}
	}
	return accounts
}

// getCookieValue extracts a cookie value from the HTTP client's cookie jar.
func getCookieValue(client *http.Client, apiURL, name string) string {
	u, err := url.Parse(apiURL)
	if err != nil {
		return ""
	}
	for _, c := range client.Jar.Cookies(u) {
		if c.Name == name {
			return c.Value
		}
	}
	return ""
}

// readSessionToken reads a session token from stdin. If stdin is a terminal,
// it prompts interactively with masked input. If stdin is a pipe, it reads
// a line silently.
func readSessionToken() (string, error) {
	if isatty.IsTerminal(os.Stdin.Fd()) || isatty.IsCygwinTerminal(os.Stdin.Fd()) {
		var token string
		err := huh.NewInput().
			Title("Enter session token").
			EchoMode(huh.EchoModePassword).
			Value(&token).
			Run()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(token), nil
	}

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text()), nil
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", nil
}

// saveSignupConfig saves the token (to credstore if available, else YAML) and
// account/API URL to the global config file, matching the auth login behavior.
func saveSignupConfig(token, account, apiURL string) error {
	globalCfg := config.LoadGlobal()

	if creds != nil {
		if err := credsSaveProfileToken(account, token); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not save token to credential store: %v\n", err)
			globalCfg.Token = token
		} else {
			globalCfg.Token = ""
		}
	} else {
		globalCfg.Token = token
	}

	// Create/update profile
	ensureProfile(account, apiURL, "")
	if profiles != nil {
		_ = profiles.SetDefault(account)
	}

	globalCfg.Account = account
	if apiURL != config.DefaultAPIURL {
		globalCfg.APIURL = apiURL
	} else {
		globalCfg.APIURL = ""
	}

	return globalCfg.Save()
}
