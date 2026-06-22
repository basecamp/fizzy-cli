package commands

import (
	"github.com/basecamp/fizzy-sdk/go/pkg/generated"
	"github.com/spf13/cobra"
)

var identityCmd = &cobra.Command{
	Use:   "identity",
	Short: "Manage identity",
	Long:  "Commands for viewing your identity and accessible accounts.",
}

var identityShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show your identity and accessible accounts",
	Long:  "Displays your user identity and all accounts you have access to.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		if err := requireSDK(); err != nil {
			return err
		}

		data, _, err := getSDKClient().Identity().GetMyIdentity(cmd.Context())
		if err != nil {
			return convertSDKError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("status", "fizzy auth status", "Auth status"),
		}

		printDetail(normalizeAny(data), "", breadcrumbs)
		return nil
	},
}

var identityTimezoneUpdateTimezone string

var identityTimezoneUpdateCmd = &cobra.Command{
	Use:   "timezone-update",
	Short: "Update your timezone",
	Long:  "Updates your timezone for the current account.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		if identityTimezoneUpdateTimezone == "" {
			return newRequiredFlagError("timezone")
		}

		resp, err := getSDKClient().Identity().UpdateMyTimezone(cmd.Context(), cfg.Account, &generated.UpdateMyTimezoneRequest{
			TimezoneName: identityTimezoneUpdateTimezone,
		})
		if err != nil {
			return convertSDKError(err)
		}

		data := any(map[string]any{"timezone_name": identityTimezoneUpdateTimezone})
		if resp != nil && len(resp.Data) > 0 {
			if normalized := normalizeAny(resp.Data); normalized != nil {
				data = normalized
			}
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("show", "fizzy identity show", "View identity"),
		}

		printMutation(data, "Timezone updated", breadcrumbs)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(identityCmd)
	identityCmd.AddCommand(identityShowCmd)
	identityTimezoneUpdateCmd.Flags().StringVar(&identityTimezoneUpdateTimezone, "timezone", "", "Timezone name, for example America/New_York (required)")
	identityCmd.AddCommand(identityTimezoneUpdateCmd)
}
