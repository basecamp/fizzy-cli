package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

type quickStartResponse struct {
	Version  string                 `json:"version"`
	Auth     quickStartAuthInfo     `json:"auth"`
	Context  quickStartContextInfo  `json:"context"`
	Commands quickStartCommandsInfo `json:"commands"`
}

type quickStartAuthInfo struct {
	Status  string `json:"status"`
	Profile string `json:"profile,omitempty"`
	Account string `json:"account,omitempty"`
}

type quickStartContextInfo struct {
	Board string `json:"board,omitempty"`
}

type quickStartCommandsInfo struct {
	QuickStart []string `json:"quick_start"`
	Common     []string `json:"common"`
}

func runRootDefault(cmd *cobra.Command, args []string) error {
	if isHumanOutput() {
		return cmd.Help()
	}

	auth := quickStartAuthInfo{Status: "unauthenticated"}
	if cfgProfile != "" {
		auth.Profile = cfgProfile
	}
	if cfg != nil {
		if cfg.Account != "" {
			auth.Account = cfg.Account
		}
		if cfg.Token != "" {
			auth.Status = "authenticated"
		}
	}

	context := quickStartContextInfo{}
	if cfg != nil && cfg.Board != "" {
		context.Board = cfg.Board
	}

	resp := quickStartResponse{
		Version: currentVersion(),
		Auth:    auth,
		Context: context,
		Commands: quickStartCommandsInfo{
			QuickStart: []string{"fizzy doctor", "fizzy config show", "fizzy board list", "fizzy card list", `fizzy search "query"`},
			Common:     []string{"fizzy auth status", "fizzy config explain", "fizzy doctor", "fizzy board list", "fizzy card show 42"},
		},
	}

	summary := fmt.Sprintf("fizzy %s - not logged in", currentVersion())
	if auth.Status == "authenticated" {
		summary = fmt.Sprintf("fizzy %s - logged in", currentVersion())
		if auth.Account != "" {
			summary += " @ " + auth.Account
		}
	}
	if auth.Profile != "" {
		summary += fmt.Sprintf(" (profile: %s)", auth.Profile)
	}

	breadcrumbs := []Breadcrumb{
		breadcrumb("doctor", "fizzy doctor", "Check CLI health"),
		breadcrumb("config", "fizzy config show", "Show the effective config"),
		breadcrumb("list_boards", "fizzy board list", "List boards"),
		breadcrumb("list_cards", "fizzy card list", "List cards"),
		breadcrumb("search_cards", `fizzy search "query"`, "Search cards"),
	}
	if auth.Status == "unauthenticated" {
		breadcrumbs = append(breadcrumbs, breadcrumb("authenticate", authLoginHint(), "Authenticate"))
	}

	printSuccessWithBreadcrumbs(resp, summary, breadcrumbs)
	return nil
}

func authLoginHint() string {
	parts := []string{"fizzy", "auth", "login", "<token>"}
	if strings.TrimSpace(cfgProfile) != "" {
		parts = append(parts, "--profile", cfgProfile)
	}
	return strings.Join(parts, " ")
}
