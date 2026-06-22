package commands

import (
	"fmt"

	"github.com/basecamp/fizzy-sdk/go/pkg/generated"
	"github.com/spf13/cobra"
)

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Manage personal access tokens",
	Long:  "Commands for managing your personal access tokens.",
}

var tokenListCmd = &cobra.Command{
	Use:   "list",
	Short: "List personal access tokens",
	Long:  "Lists your personal access tokens.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		if err := requireSDK(); err != nil {
			return err
		}

		ac := getSDKClient()
		data, _, err := ac.AccessTokens().List(cmd.Context())
		if err != nil {
			return convertSDKError(err)
		}

		items := normalizeAny(data)

		count := dataCount(items)
		summary := fmt.Sprintf("%d access tokens", count)

		breadcrumbs := []Breadcrumb{
			breadcrumb("create", "fizzy token create --description <desc> --permission <perm>", "Create a token"),
			breadcrumb("delete", "fizzy token delete <id>", "Delete a token"),
		}

		printList(items, tokenColumns, summary, breadcrumbs)
		return nil
	},
}

var (
	tokenCreateDescription string
	tokenCreatePermission  string
)

var tokenCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a personal access token",
	Long:  "Creates a new personal access token. The token value is shown once at creation and cannot be retrieved later.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		if err := requireSDK(); err != nil {
			return err
		}

		if tokenCreateDescription == "" {
			return newRequiredFlagError("description")
		}
		if tokenCreatePermission == "" {
			return newRequiredFlagError("permission")
		}

		ac := getSDKClient()
		req := &generated.CreateAccessTokenRequest{
			Description: tokenCreateDescription,
			Permission:  tokenCreatePermission,
		}
		raw, _, err := ac.AccessTokens().Create(cmd.Context(), req)
		if err != nil {
			return convertSDKError(err)
		}

		result := normalizeAny(raw)
		id := ""
		if m, ok := result.(map[string]any); ok {
			id = getStringField(m, "id")
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("list", "fizzy token list", "List tokens"),
		}
		if id != "" {
			breadcrumbs = append(breadcrumbs, breadcrumb("delete", fmt.Sprintf("fizzy token delete %s", id), "Delete this token"))
		}

		notice := "Save the token now — it will not be shown again."
		printMutation(result, notice, breadcrumbs)
		return nil
	},
}

var tokenDeleteCmd = &cobra.Command{
	Use:   "delete TOKEN_ID",
	Short: "Delete a personal access token",
	Long:  "Deletes a personal access token by ID.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}
		if err := requireSDK(); err != nil {
			return err
		}

		ac := getSDKClient()
		if _, err := ac.AccessTokens().Delete(cmd.Context(), args[0]); err != nil {
			return convertSDKError(err)
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("list", "fizzy token list", "List remaining tokens"),
		}

		printMutation(map[string]any{"deleted": true, "id": args[0]}, "", breadcrumbs)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(tokenCmd)

	tokenCmd.AddCommand(tokenListCmd)

	tokenCreateCmd.Flags().StringVar(&tokenCreateDescription, "description", "", "Token description (required)")
	tokenCreateCmd.Flags().StringVar(&tokenCreatePermission, "permission", "", "Token permission (required)")
	tokenCmd.AddCommand(tokenCreateCmd)

	tokenCmd.AddCommand(tokenDeleteCmd)
}
