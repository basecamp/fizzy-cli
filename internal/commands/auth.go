package commands

import (
	"encoding/json"
	"fmt"

	"github.com/basecamp/cli/output"
	"github.com/basecamp/fizzy-cli/internal/config"
	"github.com/basecamp/fizzy-cli/internal/errors"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
	Long:  "Commands for managing API authentication.",
}

var authLoginCmd = &cobra.Command{
	Use:   "login TOKEN",
	Short: "Save API token",
	Long:  "Saves the provided API token to the system keyring (or fallback file).",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		token := args[0]
		profileName := cfg.Account

		if profileName == "" {
			return errors.NewInvalidArgsError("No profile configured. Set --profile flag, FIZZY_PROFILE, or run 'fizzy setup'")
		}

		if creds != nil {
			if err := credsSaveProfileToken(profileName, token); err != nil {
				return &output.Error{Code: output.CodeAPI, Message: err.Error()}
			}

			// Ensure profile exists, set as default, clear YAML token
			ensureProfile(profileName, cfg.APIURL, "")
			if profiles != nil {
				_ = profiles.SetDefault(profileName)
			}
			globalCfg := config.LoadGlobal()
			globalCfg.Account = profileName
			if globalCfg.Token != "" {
				globalCfg.Token = ""
			}
			if err := globalCfg.Save(); err != nil {
				return &output.Error{Code: output.CodeAPI, Message: err.Error()}
			}
		} else {
			// Fallback: save to config file (test mode or credstore unavailable)
			globalCfg := config.LoadGlobal()
			globalCfg.Token = token
			globalCfg.Account = profileName
			if err := globalCfg.Save(); err != nil {
				return &output.Error{Code: output.CodeAPI, Message: err.Error()}
			}
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("status", "fizzy auth status", "Check auth status"),
			breadcrumb("identity", "fizzy identity show", "View identity"),
			breadcrumb("boards", "fizzy board list", "List boards"),
		}

		result := map[string]any{
			"authenticated": true,
			"profile":       profileName,
			"message":       "Token saved",
		}
		if creds != nil {
			if creds.UsingKeyring() {
				result["storage"] = "keyring"
			} else {
				result["storage"] = "file"
			}
		}

		printMutation(result, "", breadcrumbs)
		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove saved credentials",
	Long:  "Removes saved credentials for the current profile (or all profiles with --all).",
	RunE: func(cmd *cobra.Command, args []string) error {
		all, _ := cmd.Flags().GetBool("all")

		if all {
			return authLogoutAll()
		}

		profileName := cfg.Account
		if profileName == "" {
			return errors.NewInvalidArgsError("No profile configured. Use --profile to specify which profile to log out, or --all to log out of all profiles")
		}

		// Delete profile-scoped token from credstore.
		// Preserve legacy keys for downgrade compatibility.
		if creds != nil {
			_ = credsDeleteProfileToken(profileName)
		}

		// Remove profile from store
		if profiles != nil {
			_ = profiles.Delete(profileName)
		}

		// Clear active account if logging out of it
		globalCfg := config.LoadGlobal()
		if globalCfg.Account == profileName {
			globalCfg.Account = ""
			globalCfg.Token = ""
		}
		_ = globalCfg.Save()

		breadcrumbs := []Breadcrumb{
			breadcrumb("login", "fizzy auth login <token>", "Log in again"),
		}

		printMutation(map[string]any{
			"authenticated": false,
			"profile":       profileName,
			"message":       "Logged out successfully",
		}, "", breadcrumbs)
		return nil
	},
}

func authLogoutAll() error {
	// Collect all known profile/account names to clean up every key format
	names := map[string]bool{}

	if profiles != nil {
		allProfiles, _, _ := profiles.List()
		for name := range allProfiles {
			names[name] = true
		}
	}

	// Also include the YAML config's Account in case it's not in the profile store
	globalCfg := config.LoadGlobal()
	if globalCfg.Account != "" {
		names[globalCfg.Account] = true
	}

	for name := range names {
		if creds != nil {
			_ = credsDeleteProfileToken(name) // "profile:<name>"
			_ = creds.Delete("token:" + name) // legacy "token:<account>"
		}
		if profiles != nil {
			_ = profiles.Delete(name)
		}
	}
	if creds != nil {
		// Legacy bare key
		_ = creds.Delete("token")
	}

	// Clear config
	if err := config.Delete(); err != nil {
		return &output.Error{Code: output.CodeAPI, Message: err.Error()}
	}

	breadcrumbs := []Breadcrumb{
		breadcrumb("login", "fizzy auth login <token>", "Log in again"),
		breadcrumb("signup", "fizzy signup", "Sign up"),
	}

	printMutation(map[string]any{
		"authenticated": false,
		"message":       "Logged out of all profiles",
	}, "", breadcrumbs)
	return nil
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
				status["profile"] = effectiveCfg.Account
			}
			if effectiveCfg.APIURL != "" && effectiveCfg.APIURL != config.DefaultAPIURL {
				status["api_url"] = effectiveCfg.APIURL
			}
		}

		if creds != nil {
			status["using_keyring"] = creds.UsingKeyring()
			if w := creds.FallbackWarning(); w != "" {
				status["credential_warning"] = w
			}
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("identity", "fizzy identity show", "View identity"),
			breadcrumb("logout", "fizzy auth logout", "Log out"),
		}

		if profiles != nil {
			allProfiles, _, _ := profiles.List()
			if len(allProfiles) > 1 {
				breadcrumbs = append(breadcrumbs, breadcrumb("list", "fizzy auth list", "List profiles"))
			}
		}

		printDetail(status, "", breadcrumbs)
		return nil
	},
}

var authListCmd = &cobra.Command{
	Use:   "list",
	Short: "List authenticated profiles",
	Long:  "Shows all profiles with saved credentials.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if profiles == nil {
			breadcrumbs := []Breadcrumb{
				breadcrumb("login", "fizzy auth login <token>", "Log in"),
				breadcrumb("signup", "fizzy signup", "Sign up"),
			}
			printList([]any{}, authProfileColumns, "No profiles configured", breadcrumbs)
			return nil
		}

		allProfiles, defaultName, err := profiles.List()
		if err != nil || len(allProfiles) == 0 {
			breadcrumbs := []Breadcrumb{
				breadcrumb("login", "fizzy auth login <token>", "Log in"),
				breadcrumb("signup", "fizzy signup", "Sign up"),
			}
			printList([]any{}, authProfileColumns, "No profiles configured", breadcrumbs)
			return nil
		}

		entries := make([]any, 0, len(allProfiles))
		for name, p := range allProfiles {
			entry := map[string]any{
				"profile":  name,
				"base_url": p.BaseURL,
				"active":   name == defaultName,
			}

			// Check if token exists
			if creds != nil {
				_, err := credsLoadProfileToken(name)
				entry["has_token"] = err == nil
			} else {
				entry["has_token"] = false
			}

			// Include board from Extra if present
			if boardRaw, ok := p.Extra["board"]; ok {
				var board string
				if json.Unmarshal(boardRaw, &board) == nil {
					entry["board"] = board
				}
			}

			entries = append(entries, entry)
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("switch", "fizzy auth switch <profile>", "Switch profile"),
		}

		printList(entries, authProfileColumns, fmt.Sprintf("%d profile(s)", len(entries)), breadcrumbs)
		return nil
	},
}

var authSwitchCmd = &cobra.Command{
	Use:   "switch PROFILE",
	Short: "Switch active profile",
	Long:  "Sets the active profile for subsequent commands.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profileName := args[0]

		// Check if we have a token for this profile
		hasToken := false
		if creds != nil {
			if _, err := credsLoadProfileToken(profileName); err == nil {
				hasToken = true
			}
		}
		if !hasToken {
			// Also check legacy keys
			if creds != nil {
				if _, err := credsLoadLegacyToken(profileName); err == nil {
					hasToken = true
				}
			}
		}

		if !hasToken {
			return errors.NewError(fmt.Sprintf("No credentials found for profile %q. Run 'fizzy auth login <token> --profile %s' or 'fizzy signup'", profileName, profileName))
		}

		// Ensure profile exists in store
		if profiles != nil {
			ensureProfile(profileName, cfg.APIURL, "")
			if err := profiles.SetDefault(profileName); err != nil {
				return &output.Error{Code: output.CodeAPI, Message: err.Error()}
			}
		}

		// Read the target profile's board from Extra
		var profileBoard string
		if profiles != nil {
			if p, err := profiles.Get(profileName); err == nil {
				if boardRaw, ok := p.Extra["board"]; ok {
					_ = json.Unmarshal(boardRaw, &profileBoard)
				}
			}
		}

		// Update YAML config for backward compat
		globalCfg := config.LoadGlobal()
		globalCfg.Account = profileName
		globalCfg.Board = profileBoard
		if err := globalCfg.Save(); err != nil {
			return &output.Error{Code: output.CodeAPI, Message: err.Error()}
		}

		// Update in-memory config
		if cfg != nil {
			cfg.Account = profileName
			cfg.Board = profileBoard
			if creds != nil {
				if t, err := credsLoadProfileToken(profileName); err == nil {
					cfg.Token = t
				} else if t, err := credsLoadLegacyToken(profileName); err == nil {
					cfg.Token = t
				}
			}

			// Apply profile's BaseURL
			if profiles != nil {
				if p, err := profiles.Get(profileName); err == nil && p.BaseURL != "" {
					cfg.APIURL = p.BaseURL
				}
			}
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("boards", "fizzy board list", "List boards"),
			breadcrumb("status", "fizzy auth status", "Check auth status"),
		}

		printMutation(map[string]any{
			"profile": profileName,
			"message": fmt.Sprintf("Switched to profile %s", profileName),
		}, "", breadcrumbs)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authListCmd)
	authCmd.AddCommand(authSwitchCmd)

	authLogoutCmd.Flags().Bool("all", false, "Log out of all profiles")
}
