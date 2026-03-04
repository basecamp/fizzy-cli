package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/basecamp/fizzy-cli/internal/client"
	"github.com/basecamp/fizzy-cli/internal/config"
	"github.com/basecamp/fizzy-cli/internal/errors"
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
	Long:  "Configure Fizzy CLI with your API token, account, and default board.",
	Run:   runSetup,
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

func runSetup(cmd *cobra.Command, args []string) {
	fmt.Println()
	fmt.Println("Welcome to Fizzy CLI setup!")
	fmt.Println()

	// Check for existing config
	globalExists := config.Exists()
	localPath := config.LocalConfigPath()

	if globalExists || localPath != "" {
		var reconfigure bool
		configLocation := "global config"
		if localPath != "" {
			configLocation = "local config (" + localPath + ")"
		}

		err := huh.NewConfirm().
			Title(fmt.Sprintf("Existing %s found. Reconfigure?", configLocation)).
			Value(&reconfigure).
			Run()

		if err != nil {
			fmt.Println("Setup cancelled.")
			os.Exit(0)
		}

		if !reconfigure {
			fmt.Println("Setup cancelled. Existing configuration unchanged.")
			os.Exit(0)
		}
	}

	// Ask hosted vs self-hosted
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
		fmt.Println("Setup cancelled.")
		os.Exit(0)
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
				if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
					return fmt.Errorf("URL must start with http:// or https://")
				}
				return nil
			}).
			Run()

		if err != nil {
			fmt.Println("Setup cancelled.")
			os.Exit(0)
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
			os.Exit(0)
		}

		// Validate token
		fmt.Print("Validating token... ")
		accounts, err = validateToken(apiURL, token)
		if err != nil {
			fmt.Println("✗")

			var retry bool
			_ = huh.NewConfirm().
				Title("Invalid token. Would you like to try again?").
				Value(&retry).
				Run()

			if !retry {
				fmt.Println("Setup cancelled.")
				os.Exit(0)
			}
			continue
		}

		fmt.Println("✓")
		break
	}

	if len(accounts) == 0 {
		exitWithError(errors.NewError("No accounts found for this token"))
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
			os.Exit(0)
		}
	}

	// Fetch boards for selected account
	fmt.Print("Fetching boards... ")
	boards, err := fetchBoards(apiURL, token, selectedAccountSlug)
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
			os.Exit(0)
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
		os.Exit(0)
	}

	// Build and save config
	newConfig := &config.Config{
		Token:   token,
		Account: selectedAccountSlug,
		Board:   selectedBoardID,
		APIURL:  apiURL,
	}

	if saveGlobal {
		// Load existing global config to preserve any other settings
		existingConfig := config.LoadGlobal()
		existingConfig.Token = newConfig.Token
		existingConfig.Account = newConfig.Account
		existingConfig.Board = newConfig.Board
		if newConfig.APIURL != "" {
			existingConfig.APIURL = newConfig.APIURL
		}

		if err := existingConfig.Save(); err != nil {
			exitWithError(err)
		}
		fmt.Println()
		fmt.Println("✓ Configuration saved to ~/.config/fizzy/config.yaml")
	} else {
		if err := newConfig.SaveLocal(); err != nil {
			exitWithError(err)
		}
		fmt.Println()
		fmt.Println("✓ Configuration saved to .fizzy.yaml")
		fmt.Println()
		fmt.Println("⚠ Remember to add .fizzy.yaml to your .gitignore to avoid committing your token!")
	}

	fmt.Println()
	fmt.Println("You're all set! Try: fizzy board list")
}

// validateToken validates the token by calling the identity endpoint.
// Returns the list of accounts on success.
func validateToken(apiURL, token string) ([]Account, error) {
	c := client.New(apiURL, token, "")
	resp, err := c.Get(apiURL + "/my/identity.json")
	if err != nil {
		return nil, err
	}

	return parseAccounts(resp.Data)
}

// parseAccounts extracts account information from the identity response.
func parseAccounts(data interface{}) ([]Account, error) {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}

	accountsRaw, ok := dataMap["accounts"]
	if !ok {
		return nil, fmt.Errorf("no accounts in response")
	}

	accountsList, ok := accountsRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected accounts format")
	}

	accounts := make([]Account, 0, len(accountsList))
	for _, acc := range accountsList {
		accMap, ok := acc.(map[string]interface{})
		if !ok {
			continue
		}

		id, _ := accMap["id"].(string)
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

// fetchBoards fetches the list of boards for the given account.
func fetchBoards(apiURL, token, accountSlug string) ([]Board, error) {
	c := client.New(apiURL, token, accountSlug)
	resp, err := c.GetWithPagination("/boards.json", true)
	if err != nil {
		return nil, err
	}

	return parseBoards(resp.Data)
}

// parseBoards extracts board information from the boards response.
func parseBoards(data interface{}) ([]Board, error) {
	boardsList, ok := data.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected boards format")
	}

	boards := make([]Board, 0, len(boardsList))
	for _, b := range boardsList {
		boardMap, ok := b.(map[string]interface{})
		if !ok {
			continue
		}

		id, _ := boardMap["id"].(string)
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
