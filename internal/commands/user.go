package commands

import (
	"fmt"
	"strconv"

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

		client := getClient()
		path := "/users.json"
		if userListPage > 0 {
			path += "?page=" + strconv.Itoa(userListPage)
		}

		resp, err := client.GetWithPagination(path, userListAll)
		if err != nil {
			return err
		}

		// Build summary
		count := 0
		if arr, ok := resp.Data.([]any); ok {
			count = len(arr)
		}
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

		hasNext := resp.LinkNext != ""
		if hasNext {
			nextPage := userListPage + 1
			if nextPage == 0 {
				nextPage = 2
			}
			breadcrumbs = append(breadcrumbs, breadcrumb("next", fmt.Sprintf("fizzy user list --page %d", nextPage), "Next page"))
		}

		printSuccessWithPaginationAndBreadcrumbs(resp.Data, hasNext, resp.LinkNext, summary, breadcrumbs)
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

		client := getClient()
		resp, err := client.Get("/users/" + userID + ".json")
		if err != nil {
			return err
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("people", "fizzy user list", "List users"),
			breadcrumb("assign", fmt.Sprintf("fizzy card assign <number> --user %s", userID), "Assign to card"),
		}

		printSuccessWithBreadcrumbs(resp.Data, "", breadcrumbs)
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
		path := "/users/" + userID + ".json"

		if userUpdateName == "" && userUpdateAvatar == "" {
			return newRequiredFlagError("name or --avatar")
		}

		apiClient := getClient()

		if userUpdateAvatar != "" {
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
			printSuccessWithBreadcrumbs(data, "", breadcrumbs)
			return nil
		}

		body := map[string]any{
			"user": map[string]any{
				"name": userUpdateName,
			},
		}
		resp, err := apiClient.Patch(path, body)
		if err != nil {
			return err
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy user show %s", userID), "View user"),
			breadcrumb("people", "fizzy user list", "List users"),
		}

		data := resp.Data
		if data == nil {
			data = map[string]any{}
		}
		printSuccessWithBreadcrumbs(data, "", breadcrumbs)
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

		client := getClient()
		_, err := client.Delete("/users/" + userID + ".json")
		if err != nil {
			return err
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("people", "fizzy user list", "List users"),
		}

		printSuccessWithBreadcrumbs(map[string]any{
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
