package commands

import (
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

		client := getClient()
		// Identity endpoint doesn't use account prefix
		resp, err := client.Get(cfg.APIURL + "/my/identity.json")
		if err != nil {
			return err
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("status", "fizzy auth status", "Auth status"),
		}

		printDetail(resp.Data, "", breadcrumbs)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(identityCmd)
	identityCmd.AddCommand(identityShowCmd)
}
