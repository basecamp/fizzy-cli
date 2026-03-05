package commands

import (
	"github.com/basecamp/fizzy-cli/internal/config"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
	Long:  "Commands for managing API authentication.",
}

var authLoginCmd = &cobra.Command{
	Use:   "login TOKEN",
	Short: "Save API token to config file",
	Long:  "Saves the provided API token to ~/.config/fizzy/config.yaml for future use.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		token := args[0]

		// Load existing config or create new
		globalCfg := config.LoadGlobal()
		globalCfg.Token = token

		if err := globalCfg.Save(); err != nil {
			return err
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("status", "fizzy auth status", "Check auth status"),
			breadcrumb("identity", "fizzy identity show", "View identity"),
			breadcrumb("boards", "fizzy board list", "List boards"),
		}

		printSuccessWithBreadcrumbs(map[string]any{
			"authenticated": true,
			"message":       "Token saved to config file",
		}, "", breadcrumbs)
		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove saved credentials",
	Long:  "Removes the config file containing saved credentials.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.Delete(); err != nil {
			return err
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("login", "fizzy auth login <token>", "Log in again"),
		}

		printSuccessWithBreadcrumbs(map[string]any{
			"authenticated": false,
			"message":       "Logged out successfully",
		}, "", breadcrumbs)
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	Long:  "Shows whether you are currently authenticated.",
	RunE: func(cmd *cobra.Command, args []string) error {
		effectiveCfg := cfg
		if effectiveCfg == nil {
			effectiveCfg = config.Load()
		}

		status := map[string]any{
			"authenticated": effectiveCfg.Token != "",
		}

		if effectiveCfg.Token != "" {
			status["token_configured"] = true
			if effectiveCfg.Account != "" {
				status["account"] = effectiveCfg.Account
			}
			if effectiveCfg.APIURL != "" && effectiveCfg.APIURL != config.DefaultAPIURL {
				status["api_url"] = effectiveCfg.APIURL
			}
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("identity", "fizzy identity show", "View identity"),
			breadcrumb("logout", "fizzy auth logout", "Log out"),
		}

		printSuccessWithBreadcrumbs(status, "", breadcrumbs)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
}
