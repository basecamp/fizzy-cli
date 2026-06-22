package commands

import (
	"fmt"
	"strconv"

	"github.com/basecamp/fizzy-sdk/go/pkg/generated"
	"github.com/spf13/cobra"
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage users",
	Long:  "Commands for viewing users in your account.",
}

// User list flags
var userListPage int
var userListAll bool

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List users",
	Long:  "Lists all users in your account.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}
		if err := checkLimitAll(userListAll); err != nil {
			return err
		}

		ac := getSDK()
		var items any
		var linkNext string

		path := "/users.json"
		if userListPage > 0 {
			path += "?page=" + strconv.Itoa(userListPage)
		}

		if userListAll {
			pages, err := ac.GetAll(cmd.Context(), path)
			if err != nil {
				return convertSDKError(err)
			}
			items = jsonAnySlice(pages)
		} else {
			listPath := ""
			if userListPage > 0 {
				listPath = path
			}
			data, resp, err := ac.Users().List(cmd.Context(), listPath)
			if err != nil {
				return convertSDKError(err)
			}
			items = normalizeAny(data)
			linkNext = parseSDKLinkNext(resp)
		}

		// Build summary
		count := dataCount(items)
		summary := fmt.Sprintf("%d users", count)
		if userListAll {
			summary += " (all)"
		} else if userListPage > 0 {
			summary += fmt.Sprintf(" (page %d)", userListPage)
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("show", "fizzy user show <id>", "View user details"),
			breadcrumb("assign", "fizzy card assign <number> --user <user_id>", "Assign user to card"),
		}

		hasNext := linkNext != ""
		if hasNext {
			nextPage := userListPage + 1
			if userListPage == 0 {
				nextPage = 2
			}
			breadcrumbs = append(breadcrumbs, breadcrumb("next", fmt.Sprintf("fizzy user list --page %d", nextPage), "Next page"))
		}

		printListPaginated(items, userColumns, hasNext, linkNext, userListAll, summary, breadcrumbs)
		return nil
	},
}

var userShowCmd = &cobra.Command{
	Use:   "show USER_ID",
	Short: "Show a user",
	Long:  "Shows details of a specific user.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		userID := args[0]

		data, _, err := getSDK().Users().Get(cmd.Context(), userID)
		if err != nil {
			return convertSDKError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("people", "fizzy user list", "List users"),
			breadcrumb("assign", fmt.Sprintf("fizzy card assign <number> --user %s", userID), "Assign to card"),
		}

		printDetail(normalizeAny(data), "", breadcrumbs)
		return nil
	},
}

// User update flags
var userUpdateName string
var userUpdateAvatar string

var userUpdateCmd = &cobra.Command{
	Use:   "update USER_ID",
	Short: "Update a user",
	Long:  "Updates a user's details. Requires admin or owner permissions.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		userID := args[0]

		if userUpdateName == "" && userUpdateAvatar == "" {
			return newRequiredFlagError("name or --avatar")
		}

		// Avatar upload requires multipart — keep using old client for this case
		if userUpdateAvatar != "" {
			apiClient := getClient()
			path := "/users/" + userID + ".json"
			fields := make(map[string]string)
			if userUpdateName != "" {
				fields["user[name]"] = userUpdateName
			}
			resp, err := apiClient.PatchMultipart(path, "user[avatar]", userUpdateAvatar, fields)
			if err != nil {
				return err
			}

			breadcrumbs := []Breadcrumb{
				breadcrumb("show", fmt.Sprintf("fizzy user show %s", userID), "View user"),
				breadcrumb("people", "fizzy user list", "List users"),
			}

			data := resp.Data
			if data == nil {
				data = map[string]any{}
			}
			printMutation(data, "", breadcrumbs)
			return nil
		}

		respData, _, err := getSDK().Users().Update(cmd.Context(), userID, &generated.UpdateUserRequest{Name: userUpdateName})
		if err != nil {
			return convertSDKError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy user show %s", userID), "View user"),
			breadcrumb("people", "fizzy user list", "List users"),
		}

		data := normalizeAny(respData)
		if data == nil {
			data = map[string]any{}
		}
		printMutation(data, "", breadcrumbs)
		return nil
	},
}

var userDeactivateCmd = &cobra.Command{
	Use:   "deactivate USER_ID",
	Short: "Deactivate a user",
	Long:  "Deactivates a user, removing their access to the account. Requires admin or owner permissions.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		userID := args[0]

		_, err := getSDK().Users().Deactivate(cmd.Context(), userID)
		if err != nil {
			return convertSDKError(err)
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("people", "fizzy user list", "List users"),
		}

		printMutation(map[string]any{
			"deactivated": true,
		}, "", breadcrumbs)
		return nil
	},
}

// User role flags
var userRoleRole string

var userRoleCmd = &cobra.Command{
	Use:   "role USER_ID",
	Short: "Update a user's role",
	Long:  "Updates a user's role. Requires admin or owner permissions.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		if userRoleRole == "" {
			return newRequiredFlagError("role")
		}

		userID := args[0]

		_, err := getSDK().Users().UpdateRole(cmd.Context(), userID, &generated.UpdateUserRoleRequest{
			Role: userRoleRole,
		})
		if err != nil {
			return convertSDKError(err)
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy user show %s", userID), "View user"),
			breadcrumb("people", "fizzy user list", "List users"),
		}

		printMutation(map[string]any{}, "", breadcrumbs)
		return nil
	},
}

var userAvatarRemoveCmd = &cobra.Command{
	Use:   "avatar-remove USER_ID",
	Short: "Remove a user's avatar",
	Long:  "Removes a user's avatar image.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		userID := args[0]

		_, err := getSDK().Users().DeleteAvatar(cmd.Context(), userID)
		if err != nil {
			return convertSDKError(err)
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy user show %s", userID), "View user"),
			breadcrumb("people", "fizzy user list", "List users"),
		}

		printMutation(map[string]any{}, "", breadcrumbs)
		return nil
	},
}

var userExportCreateCmd = &cobra.Command{
	Use:   "export-create USER_ID",
	Short: "Create a user export",
	Long:  "Creates a new user data export.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		userID := args[0]

		data, _, err := getSDK().Users().CreateUserDataExport(cmd.Context(), userID)
		if err != nil {
			return convertSDKError(err)
		}

		items := normalizeAny(data)
		exportID := ""
		if export, ok := items.(map[string]any); ok {
			if id, ok := export["id"]; ok {
				exportID = fmt.Sprintf("%v", id)
			}
		}

		var breadcrumbs []Breadcrumb
		if exportID != "" {
			breadcrumbs = []Breadcrumb{
				breadcrumb("show", fmt.Sprintf("fizzy user export-show %s %s", userID, exportID), "View export status"),
				breadcrumb("user", fmt.Sprintf("fizzy user show %s", userID), "View user"),
			}
		}

		printMutation(items, "", breadcrumbs)
		return nil
	},
}

var userExportShowCmd = &cobra.Command{
	Use:   "export-show USER_ID EXPORT_ID",
	Short: "Show a user export",
	Long:  "Shows the status of a user data export.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		userID := args[0]
		exportID := args[1]

		data, _, err := getSDK().Users().GetUserDataExport(cmd.Context(), userID, exportID)
		if err != nil {
			return convertSDKError(err)
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("user", fmt.Sprintf("fizzy user show %s", userID), "View user"),
			breadcrumb("export-create", fmt.Sprintf("fizzy user export-create %s", userID), "Create another export"),
		}

		printDetail(normalizeAny(data), "", breadcrumbs)
		return nil
	},
}

var userEmailChangeRequestEmail string

var userEmailChangeRequestCmd = &cobra.Command{
	Use:   "email-change-request USER_ID",
	Short: "Request a user email address change",
	Long:  "Requests an email address change for a user.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}
		if userEmailChangeRequestEmail == "" {
			return newRequiredFlagError("email")
		}

		userID := args[0]
		resp, err := getSDK().Users().RequestEmailAddressChange(cmd.Context(), userID, &generated.RequestEmailAddressChangeRequest{
			EmailAddress: userEmailChangeRequestEmail,
		})
		if err != nil {
			return convertSDKError(err)
		}

		data := normalizeAny(resp.Data)
		if data == nil {
			data = map[string]any{"requested": true}
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("user", fmt.Sprintf("fizzy user show %s", userID), "View user"),
		}

		printMutation(data, "", breadcrumbs)
		return nil
	},
}

var userEmailChangeConfirmCmd = &cobra.Command{
	Use:   "email-change-confirm USER_ID TOKEN",
	Short: "Confirm a user email address change",
	Long:  "Confirms an email address change for a user.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		userID := args[0]
		token := args[1]

		resp, err := getSDK().Users().ConfirmEmailAddressChange(cmd.Context(), userID, token)
		if err != nil {
			return convertSDKError(err)
		}

		data := normalizeAny(resp.Data)
		if data == nil {
			data = map[string]any{"confirmed": true}
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("user", fmt.Sprintf("fizzy user show %s", userID), "View user"),
		}

		printMutation(data, "", breadcrumbs)
		return nil
	},
}

// Push subscription create flags
var pushSubCreateUser string
var pushSubCreateEndpoint string
var pushSubCreateP256dhKey string
var pushSubCreateAuthKey string

var userPushSubscriptionCreateCmd = &cobra.Command{
	Use:   "push-subscription-create",
	Short: "Create a push subscription",
	Long:  "Creates a web push subscription for a user.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		if pushSubCreateUser == "" {
			return newRequiredFlagError("user")
		}
		if pushSubCreateEndpoint == "" {
			return newRequiredFlagError("endpoint")
		}
		if pushSubCreateP256dhKey == "" {
			return newRequiredFlagError("p256dh-key")
		}
		if pushSubCreateAuthKey == "" {
			return newRequiredFlagError("auth-key")
		}

		_, err := getSDK().Users().CreatePushSubscription(cmd.Context(), pushSubCreateUser, &generated.CreatePushSubscriptionRequest{
			Endpoint:  pushSubCreateEndpoint,
			P256dhKey: pushSubCreateP256dhKey,
			AuthKey:   pushSubCreateAuthKey,
		})
		if err != nil {
			return convertSDKError(err)
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy user show %s", pushSubCreateUser), "View user"),
		}

		printMutation(map[string]any{}, "", breadcrumbs)
		return nil
	},
}

// Push subscription delete flags
var pushSubDeleteUser string

var userPushSubscriptionDeleteCmd = &cobra.Command{
	Use:   "push-subscription-delete SUBSCRIPTION_ID",
	Short: "Delete a push subscription",
	Long:  "Deletes a web push subscription for a user.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		if pushSubDeleteUser == "" {
			return newRequiredFlagError("user")
		}

		_, err := getSDK().Users().DeletePushSubscription(cmd.Context(), pushSubDeleteUser, args[0])
		if err != nil {
			return convertSDKError(err)
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy user show %s", pushSubDeleteUser), "View user"),
		}

		printMutation(map[string]any{
			"deleted": true,
		}, "", breadcrumbs)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(userCmd)

	// List
	userListCmd.Flags().IntVar(&userListPage, "page", 0, "Page number")
	userListCmd.Flags().BoolVar(&userListAll, "all", false, "Fetch all pages")
	userCmd.AddCommand(userListCmd)

	// Show
	userCmd.AddCommand(userShowCmd)

	// Update
	userUpdateCmd.Flags().StringVar(&userUpdateName, "name", "", "User's display name")
	userUpdateCmd.Flags().StringVar(&userUpdateAvatar, "avatar", "", "Path to avatar image file")
	userCmd.AddCommand(userUpdateCmd)

	// Deactivate
	userCmd.AddCommand(userDeactivateCmd)

	// Role
	userRoleCmd.Flags().StringVar(&userRoleRole, "role", "", "Role to assign (required)")
	userCmd.AddCommand(userRoleCmd)

	// Avatar remove
	userCmd.AddCommand(userAvatarRemoveCmd)

	// Exports
	userCmd.AddCommand(userExportCreateCmd)
	userCmd.AddCommand(userExportShowCmd)

	// Email change
	userEmailChangeRequestCmd.Flags().StringVar(&userEmailChangeRequestEmail, "email", "", "New email address (required)")
	userCmd.AddCommand(userEmailChangeRequestCmd)
	userCmd.AddCommand(userEmailChangeConfirmCmd)

	// Push subscriptions
	userPushSubscriptionCreateCmd.Flags().StringVar(&pushSubCreateUser, "user", "", "User ID (required)")
	userPushSubscriptionCreateCmd.Flags().StringVar(&pushSubCreateEndpoint, "endpoint", "", "Push endpoint URL (required)")
	userPushSubscriptionCreateCmd.Flags().StringVar(&pushSubCreateP256dhKey, "p256dh-key", "", "P256dh key (required)")
	userPushSubscriptionCreateCmd.Flags().StringVar(&pushSubCreateAuthKey, "auth-key", "", "Auth key (required)")
	userCmd.AddCommand(userPushSubscriptionCreateCmd)

	userPushSubscriptionDeleteCmd.Flags().StringVar(&pushSubDeleteUser, "user", "", "User ID (required)")
	userCmd.AddCommand(userPushSubscriptionDeleteCmd)
}
