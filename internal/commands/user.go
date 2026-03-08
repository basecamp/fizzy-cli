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
}
