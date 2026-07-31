package commands

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"os"

	"github.com/basecamp/cli/output"
	"github.com/basecamp/cli/profile"
	"github.com/basecamp/fizzy-cli/internal/config"
	"github.com/basecamp/fizzy-cli/internal/errors"
	"github.com/spf13/cobra"
)

var authLoginAccount string

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
		profileName := currentProfileName()
		account := firstNonEmpty(authLoginAccount, cfg.Account)

		if profileName == "" {
			return errors.NewInvalidArgsError("No profile configured. Set --profile flag, FIZZY_PROFILE, or run 'fizzy setup'")
		}
		if account == "" {
			return errors.NewInvalidArgsError("No account configured. Set --account to the Fizzy account slug or ID")
		}
		if err := profile.ValidateName(profileName); err != nil {
			return errors.NewInvalidArgsError(err.Error())
		}
		if err := validateAccountIdentifier(account); err != nil {
			return errors.NewInvalidArgsError(err.Error())
		}
		activeProfile = profileName
		cfg.Account = account

		if creds != nil {
			_, err := saveProfileCredentialState(profileCredentialSaveOptions{
				ProfileName: profileName,
				Account:     account,
				BaseURL:     cfg.APIURL,
				Token:       token,
				UpdateGlobal: func(globalCfg *config.Config, credentialStored bool) {
					globalCfg.Account = account
					if credentialStored {
						globalCfg.Token = ""
					}
				},
			})
			if err != nil {
				return &output.Error{Code: output.CodeAPI, Message: err.Error()}
			}
		} else {
			// Fallback: save to config file (test mode or credstore unavailable)
			globalCfg := config.LoadGlobal()
			globalCfg.Token = token
			globalCfg.Account = account
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
			"account":       account,
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

		profileName := currentProfileName()
		if profileName == "" {
			return errors.NewInvalidArgsError("No profile configured. Use --profile to specify which profile to log out, or --all to log out of all profiles")
		}

		// Read profile state before cleanup so failures cannot silently change
		// which identity remains active.
		wasDefault := false
		profileExists := false
		if profiles != nil {
			allProfiles, defaultName, err := profiles.List()
			if err != nil {
				return &output.Error{Code: output.CodeAPI, Message: err.Error()}
			}
			_, profileExists = allProfiles[profileName]
			wasDefault = defaultName == profileName
		}

		var cleanupErrors []error
		// Preserve legacy keys for downgrade compatibility. Explicit aliases do
		// not consume those keys in the current CLI.
		if creds != nil {
			if err := credsDeleteProfileToken(profileName); err != nil && !isCredentialNotFound(err) {
				cleanupErrors = append(cleanupErrors, fmt.Errorf("delete credential: %w", err))
			}
		}
		if profiles != nil && profileExists {
			if err := profiles.Delete(profileName); err != nil {
				cleanupErrors = append(cleanupErrors, fmt.Errorf("delete profile: %w", err))
			}
		}

		globalCfg := config.LoadGlobal()
		if wasDefault || globalCfg.Account == profileName {
			globalCfg.Account = ""
			globalCfg.Token = ""
		}
		if err := globalCfg.Save(); err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("save global config: %w", err))
		}
		if err := stderrors.Join(cleanupErrors...); err != nil {
			return &output.Error{Code: output.CodeAPI, Message: fmt.Sprintf("logout incomplete: %v", err)}
		}

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
	profileNames := map[string]bool{}
	credentialNames := map[string]bool{}
	var cleanupErrors []error

	if profiles != nil {
		allProfiles, _, err := profiles.List()
		if err != nil {
			return &output.Error{Code: output.CodeAPI, Message: err.Error()}
		}
		for name, p := range allProfiles {
			profileNames[name] = true
			credentialNames[name] = true
			binding, err := resolveProfileAccountBinding(name, p)
			if err == nil {
				credentialNames[binding.Account] = true
			}
		}
	}

	globalCfg := config.LoadGlobal()
	if globalCfg.Account != "" {
		credentialNames[globalCfg.Account] = true
	}

	if creds != nil {
		for name := range credentialNames {
			for _, key := range []string{profile.CredentialKey(name, ""), "token:" + name} {
				if err := creds.Delete(key); err != nil && !isCredentialNotFound(err) {
					cleanupErrors = append(cleanupErrors, fmt.Errorf("delete credential %q: %w", key, err))
				}
			}
		}
		if err := creds.Delete("token"); err != nil && !isCredentialNotFound(err) {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("delete legacy credential: %w", err))
		}
	}
	if profiles != nil {
		for name := range profileNames {
			if err := profiles.Delete(name); err != nil {
				cleanupErrors = append(cleanupErrors, fmt.Errorf("delete profile %q: %w", name, err))
			}
		}
	}
	if err := config.Delete(); err != nil {
		cleanupErrors = append(cleanupErrors, fmt.Errorf("delete global config: %w", err))
	}
	if err := stderrors.Join(cleanupErrors...); err != nil {
		return &output.Error{Code: output.CodeAPI, Message: fmt.Sprintf("logout incomplete: %v", err)}
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
			profileName := activeProfile
			if profileName == "" {
				profileName = effectiveCfg.Account
			}
			if profileName != "" {
				status["profile"] = profileName
			}
			if effectiveCfg.Account != "" {
				status["account"] = effectiveCfg.Account
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
		if err != nil {
			return &output.Error{Code: output.CodeAPI, Message: err.Error()}
		}
		if len(allProfiles) == 0 {
			breadcrumbs := []Breadcrumb{
				breadcrumb("login", "fizzy auth login <token>", "Log in"),
				breadcrumb("signup", "fizzy signup", "Sign up"),
			}
			printList([]any{}, authProfileColumns, "No profiles configured", breadcrumbs)
			return nil
		}

		entries := make([]any, 0, len(allProfiles))
		for name, p := range allProfiles {
			binding, err := resolveProfileAccountBinding(name, p)
			if err != nil {
				return &output.Error{Code: output.CodeUsage, Message: err.Error()}
			}
			entry := map[string]any{
				"profile":  name,
				"account":  binding.Account,
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
		if err := profile.ValidateName(profileName); err != nil {
			return errors.NewInvalidArgsError(err.Error())
		}

		profileSnapshot, err := snapshotProfileState(profileName)
		if err != nil {
			return &output.Error{Code: output.CodeAPI, Message: err.Error()}
		}
		targetExplicitAccount := false
		if profileSnapshot.previous != nil {
			binding, err := resolveProfileAccountBinding(profileName, profileSnapshot.previous)
			if err != nil {
				return &output.Error{Code: output.CodeUsage, Message: err.Error()}
			}
			targetExplicitAccount = binding.Explicit
		}

		hasToken := false
		if creds != nil {
			if token, err := credsLoadProfileToken(profileName); err == nil && token != "" {
				hasToken = true
			}
			if !hasToken && !targetExplicitAccount {
				if token, err := credsLoadLegacyToken(profileName); err == nil && token != "" {
					hasToken = true
				}
			}
		}

		if !hasToken {
			return errors.NewError(fmt.Sprintf("No credentials found for profile %q. Run 'fizzy auth login <token> --profile %s' or 'fizzy signup'", profileName, profileName))
		}

		globalSnapshot := snapshotGlobalConfig()
		// Preserve a saved deployment URL and seed reconstructed profiles from
		// the current effective URL.
		profileBaseURL := ""
		if profileSnapshot.previous == nil && cfg != nil {
			profileBaseURL = cfg.APIURL
		}
		if profiles != nil {
			if err := ensureProfile(profileName, profileBaseURL, ""); err != nil {
				return &output.Error{Code: output.CodeAPI, Message: err.Error()}
			}
			if err := profiles.SetDefault(profileName); err != nil {
				restoreErr := restoreProfileState(profileSnapshot)
				return &output.Error{Code: output.CodeAPI, Message: stderrors.Join(err, restoreErr).Error()}
			}
		}

		// Read the target profile's account, board, and deployment URL.
		profileAccountID := profileName
		profileAPIURL := config.DefaultAPIURL
		if cfg != nil && cfg.APIURL != "" {
			profileAPIURL = cfg.APIURL
		}
		var profileBoard string
		if profiles != nil {
			p, err := profiles.Get(profileName)
			if err != nil {
				restoreErr := restoreProfileState(profileSnapshot)
				return &output.Error{Code: output.CodeAPI, Message: stderrors.Join(err, restoreErr).Error()}
			}
			binding, bindingErr := resolveProfileAccountBinding(profileName, p)
			if bindingErr != nil {
				restoreErr := restoreProfileState(profileSnapshot)
				return &output.Error{Code: output.CodeUsage, Message: stderrors.Join(bindingErr, restoreErr).Error()}
			}
			profileAccountID = binding.Account
			targetExplicitAccount = binding.Explicit
			if p.BaseURL != "" {
				profileAPIURL = p.BaseURL
			}
			if boardRaw, ok := p.Extra["board"]; ok {
				_ = json.Unmarshal(boardRaw, &profileBoard)
			}
		}

		// Update YAML config for backward compatibility.
		globalCfg := config.LoadGlobal()
		globalCfg.Account = profileAccountID
		globalCfg.APIURL = profileAPIURL
		globalCfg.Board = profileBoard
		if err := globalCfg.Save(); err != nil {
			restoreErr := restoreProfileState(profileSnapshot)
			globalRestoreErr := restoreGlobalConfig(globalSnapshot)
			return &output.Error{Code: output.CodeAPI, Message: stderrors.Join(err, restoreErr, globalRestoreErr).Error()}
		}

		// Update in-memory config
		if cfg != nil {
			activeProfile = profileName
			activeProfileExplicitAccount = targetExplicitAccount
			cfg.Account = profileAccountID
			cfg.Board = profileBoard
			if creds != nil {
				if t, err := credsLoadProfileToken(profileName); err == nil {
					cfg.Token = t
				} else if !targetExplicitAccount {
					if t, err := credsLoadLegacyToken(profileName); err == nil {
						cfg.Token = t
					}
				}
			}

			if cfgAPIURL == "" && os.Getenv("FIZZY_API_URL") == "" {
				cfg.APIURL = profileAPIURL
			}
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("boards", "fizzy board list", "List boards"),
			breadcrumb("status", "fizzy auth status", "Check auth status"),
		}

		printMutation(map[string]any{
			"profile": profileName,
			"account": profileAccountID,
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

	authLoginCmd.Flags().StringVar(&authLoginAccount, "account", "", "Fizzy account slug or ID for this profile")
	authLogoutCmd.Flags().Bool("all", false, "Log out of all profiles")
}
