package commands

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/basecamp/cli/output"
	"github.com/basecamp/fizzy-cli/internal/config"
	"github.com/basecamp/fizzy-cli/internal/errors"
	"github.com/basecamp/fizzy-cli/internal/tui"
	fizzy "github.com/basecamp/fizzy-sdk/go/pkg/fizzy"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

// Account represents an account from the identity response.
type Account struct {
	ID   string
	Name string
	Slug string
}

// Board represents a board from the boards response.
type Board struct {
	ID   string
	Name string
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive setup wizard",
	Long:  "Configure Fizzy CLI with your API token, account, and default board.\nNew users without an account will be guided through signup.",
	RunE:  runSetup,
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

func runSetup(cmd *cobra.Command, args []string) error {
	if cfgJQ != "" {
		return errors.ErrJQNotSupported("the setup command")
	}
	if IsMachineOutput() {
		return output.ErrUsageHint("setup requires an interactive terminal", "Run without --agent/--json/--quiet or in a TTY")
	}

	aw, wait := tui.AnimateBannerAsync(cmd.ErrOrStderr())
	fmt.Fprintln(aw)
	fmt.Fprintln(aw, "Welcome to Fizzy CLI setup!")
	fmt.Fprintln(aw)
	wait()

	// Ask if user has an account before checking existing config
	var hasAccount string
	err := huh.NewSelect[string]().
		Title("Do you have a Fizzy account?").
		Options(
			huh.NewOption("Yes, I have an account", "yes"),
			huh.NewOption("No, I'd like to sign up", "no"),
		).
		Value(&hasAccount).
		Run()

	if err != nil {
		fmt.Println("Setup cancelled.")
		return nil //nolint:nilerr // user cancelled prompt
	}

	if hasAccount == "no" {
		return signupWizard()
	}

	// Check for existing config
	globalExists := config.Exists()
	localPath := config.LocalConfigPath()

	if globalExists || localPath != "" {
		var reconfigure bool
		configLocation := "global config"
		if localPath != "" {
			configLocation = "local config (" + localPath + ")"
		}

		err = huh.NewConfirm().
			Title(fmt.Sprintf("Existing %s found. Reconfigure?", configLocation)).
			Value(&reconfigure).
			Run()

		if err != nil {
			fmt.Println("Setup cancelled.")
			return nil //nolint:nilerr // user cancelled prompt
		}

		if !reconfigure {
			fmt.Println("Setup cancelled. Existing configuration unchanged.")
			return nil
		}
	}

	// Ask hosted vs self-hosted
	var hostingType string
	err = huh.NewSelect[string]().
		Title("Are you using the hosted or self-hosted version?").
		Options(
			huh.NewOption("Hosted (app.fizzy.do)", "hosted"),
			huh.NewOption("Self-hosted", "selfhosted"),
		).
		Value(&hostingType).
		Run()

	if err != nil {
		fmt.Println("Setup cancelled.")
		return nil //nolint:nilerr // user cancelled prompt
	}

	apiURL := config.DefaultAPIURL
	if hostingType == "selfhosted" {
		err = huh.NewInput().
			Title("Enter your Fizzy URL").
			Placeholder("https://fizzy.example.com").
			Value(&apiURL).
			Validate(validateAPIURL).
			Run()

		if err != nil {
			fmt.Println("Setup cancelled.")
			return nil //nolint:nilerr // user cancelled prompt
		}
	}

	// Token input loop with retry
	var token string
	var accounts []Account

	for {
		err = huh.NewInput().
			Title("Enter your API token").
			Description("Visit My Profile → Personal Access Tokens").
			Placeholder("fizzy_...").
			Value(&token).
			EchoMode(huh.EchoModePassword).
			Validate(func(s string) error {
				if s == "" {
					return fmt.Errorf("token is required")
				}
				return nil
			}).
			Run()

		if err != nil {
			fmt.Println("Setup cancelled.")
			return nil //nolint:nilerr // user cancelled prompt
		}

		// Validate token
		fmt.Print("Validating token... ")
		accounts, err = validateToken(cmd, apiURL, token)
		if err != nil {
			fmt.Println("✗")

			var retry bool
			_ = huh.NewConfirm().
				Title("Invalid token. Would you like to try again?").
				Value(&retry).
				Run()

			if !retry {
				fmt.Println("Setup cancelled.")
				return nil
			}
			continue
		}

		fmt.Println("✓")
		break
	}

	if len(accounts) == 0 {
		return errors.NewError("No accounts found for this token")
	}

	// Account selection
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
			fmt.Println("Setup cancelled.")
			return nil //nolint:nilerr // user cancelled prompt
		}
	}

	// Fetch boards for selected account
	fmt.Print("Fetching boards... ")
	boards, err := fetchBoards(cmd, apiURL, token, selectedAccountSlug)
	if err != nil {
		fmt.Println("✗")
		// Non-fatal, just skip board selection
		fmt.Println("Could not fetch boards. Skipping board selection.")
		boards = nil
	} else {
		fmt.Println("✓")
	}

	// Board selection (optional)
	var selectedBoardID string
	if len(boards) > 0 {
		boardOptions := make([]huh.Option[string], len(boards)+1)
		boardOptions[0] = huh.NewOption("None (skip)", "")
		for i, board := range boards {
			boardOptions[i+1] = huh.NewOption(board.Name, board.ID)
		}

		err = huh.NewSelect[string]().
			Title("Select default board (optional)").
			Options(boardOptions...).
			Value(&selectedBoardID).
			Run()

		if err != nil {
			fmt.Println("Setup cancelled.")
			return nil //nolint:nilerr // user cancelled prompt
		}
	}

	// Ask where to save
	var saveGlobal bool
	err = huh.NewSelect[bool]().
		Title("Where should we save the configuration?").
		Options(
			huh.NewOption("Global (~/.config/fizzy/config.yaml)", true),
			huh.NewOption("Local (.fizzy.yaml in current directory)", false),
		).
		Value(&saveGlobal).
		Run()

	if err != nil {
		fmt.Println("Setup cancelled.")
		return nil //nolint:nilerr // user cancelled prompt
	}

	// Build and save config
	newConfig := &config.Config{
		Token:   token,
		Account: selectedAccountSlug,
		Board:   selectedBoardID,
		APIURL:  apiURL,
	}

	if saveGlobal {
		// Save token to credstore when available
		credstoreSaved := false
		if creds != nil {
			if err := credsSaveProfileToken(selectedAccountSlug, token); err != nil {
				fmt.Printf("Warning: could not save token to credential store: %v\n", err)
			} else {
				credstoreSaved = true
			}
		}

		// Create/update profile
		ensureProfile(selectedAccountSlug, apiURL, selectedBoardID)
		// If user chose "None (skip)", clear any previously saved board
		if selectedBoardID == "" && profiles != nil {
			if p, err := profiles.Get(selectedAccountSlug); err == nil {
				delete(p.Extra, "board")
				_ = profiles.Delete(selectedAccountSlug)
				_ = profiles.Create(p)
			}
		}
		if profiles != nil {
			_ = profiles.SetDefault(selectedAccountSlug)
		}

		// Load existing global config to preserve any other settings
		existingConfig := config.LoadGlobal()
		// Only clear YAML token when credstore save actually succeeded
		if credstoreSaved {
			existingConfig.Token = ""
		} else {
			existingConfig.Token = newConfig.Token
		}
		existingConfig.Account = newConfig.Account
		existingConfig.Board = newConfig.Board
		if newConfig.APIURL != "" {
			existingConfig.APIURL = newConfig.APIURL
		}

		if err := existingConfig.Save(); err != nil {
			return err
		}
		fmt.Println()
		fmt.Println("✓ Configuration saved to ~/.config/fizzy/config.yaml")
	} else {
		if err := newConfig.SaveLocal(); err != nil {
			return err
		}
		fmt.Println()
		fmt.Println("✓ Configuration saved to .fizzy.yaml")
		fmt.Println()
		fmt.Println("⚠ Remember to add .fizzy.yaml to your .gitignore to avoid committing your token!")
	}

	// Coding agent integration
	if err := setupAgents(cmd); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("You're all set! Try: fizzy board list")

	return nil
}

// validateToken validates the token by calling the identity endpoint via the SDK.
// Returns the list of accounts on success.
func validateToken(cmd *cobra.Command, apiURL, token string) ([]Account, error) {
	sdkCfg := &fizzy.Config{BaseURL: apiURL}
	testClient := fizzy.NewClient(sdkCfg, &fizzy.StaticTokenProvider{Token: token})
	_, resp, err := testClient.Identity().GetMyIdentity(cmd.Context())
	if err != nil {
		return nil, err
	}

	return parseAccounts(resp.Data)
}

// parseAccounts extracts account information from the raw identity response JSON.
func parseAccounts(data json.RawMessage) ([]Account, error) {
	var dataMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &dataMap); err != nil {
		return nil, fmt.Errorf("unexpected response format")
	}

	accountsRaw, ok := dataMap["accounts"]
	if !ok {
		return nil, fmt.Errorf("no accounts in response")
	}

	var accountsList []map[string]any
	if err := json.Unmarshal(accountsRaw, &accountsList); err != nil {
		return nil, fmt.Errorf("unexpected accounts format")
	}

	accounts := make([]Account, 0, len(accountsList))
	for _, accMap := range accountsList {
		idVal, ok := accMap["id"]
		if !ok || idVal == nil {
			continue
		}
		id := fmt.Sprintf("%v", idVal)
		name, _ := accMap["name"].(string)
		slug, _ := accMap["slug"].(string)

		// Remove leading slash from slug if present
		slug = strings.TrimPrefix(slug, "/")

		if slug != "" {
			accounts = append(accounts, Account{
				ID:   id,
				Name: name,
				Slug: slug,
			})
		}
	}

	return accounts, nil
}

// fetchBoards fetches the list of boards for the given account via the SDK.
func fetchBoards(cmd *cobra.Command, apiURL, token, accountSlug string) ([]Board, error) {
	sdkCfg := &fizzy.Config{BaseURL: apiURL}
	testClient := fizzy.NewClient(sdkCfg, &fizzy.StaticTokenProvider{Token: token})
	ac := testClient.ForAccount(accountSlug)

	pages, err := ac.GetAll(cmd.Context(), "/boards.json")
	if err != nil {
		return nil, err
	}

	return parseBoards(pages)
}

// parseBoards extracts board information from raw JSON board pages.
func parseBoards(pages []json.RawMessage) ([]Board, error) {
	boards := make([]Board, 0, len(pages))
	for _, raw := range pages {
		var boardMap map[string]any
		if err := json.Unmarshal(raw, &boardMap); err != nil {
			continue
		}

		var id string
		if v, ok := boardMap["id"]; ok && v != nil {
			id = fmt.Sprintf("%v", v)
		}
		name, _ := boardMap["name"].(string)

		if id != "" && name != "" {
			boards = append(boards, Board{
				ID:   id,
				Name: name,
			})
		}
	}

	return boards, nil
}

// validateAPIURL checks that the URL is well-formed and enforces HTTPS for non-localhost URLs.
func validateAPIURL(s string) error {
	if s == "" {
		return fmt.Errorf("URL is required")
	}
	if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
		return fmt.Errorf("URL must start with http:// or https://")
	}
	u, err := url.Parse(s)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Hostname() == "" {
		return fmt.Errorf("URL must include a hostname")
	}
	if u.Scheme == "http" {
		host := u.Hostname()
		if host != "localhost" && host != "127.0.0.1" && host != "::1" && !strings.HasSuffix(host, ".localhost") {
			return fmt.Errorf("non-localhost URLs must use https://")
		}
	}
	return nil
}
