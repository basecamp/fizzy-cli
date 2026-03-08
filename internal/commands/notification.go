package commands

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var notificationCmd = &cobra.Command{
	Use:   "notification",
	Short: "Manage notifications",
	Long:  "Commands for managing your notifications.",
}

// Notification list flags
var notificationListPage int
var notificationListAll bool

var notificationListCmd = &cobra.Command{
	Use:   "list",
	Short: "List notifications",
	Long:  "Lists your notifications.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}
		if err := checkLimitAll(notificationListAll); err != nil {
			return err
		}

		ac := getSDK()
		var items any
		var linkNext string

		path := "/notifications.json"
		if notificationListPage > 0 {
			path += "?page=" + strconv.Itoa(notificationListPage)
		}

		if notificationListAll {
			pages, err := ac.GetAll(cmd.Context(), path)
			if err != nil {
				return convertSDKError(err)
			}
			items = jsonAnySlice(pages)
		} else {
			data, resp, err := ac.Notifications().List(cmd.Context(), path)
			if err != nil {
				return convertSDKError(err)
			}
			items = normalizeAny(data)
			linkNext = parseSDKLinkNext(resp)
		}

		// Build summary with unread count
		count := dataCount(items)
		unreadCount := 0
		for _, item := range toSliceAny(items) {
			if notif, ok := item.(map[string]any); ok {
				if read, ok := notif["read"].(bool); ok && !read {
					unreadCount++
				}
			}
		}
		summary := fmt.Sprintf("%d notifications (%d unread)", count, unreadCount)
		if notificationListAll {
			summary = fmt.Sprintf("%d notifications (%d unread, all)", count, unreadCount)
		} else if notificationListPage > 0 {
			summary = fmt.Sprintf("%d notifications (%d unread, page %d)", count, unreadCount, notificationListPage)
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("read", "fizzy notification read <id>", "Mark as read"),
			breadcrumb("read-all", "fizzy notification read-all", "Mark all as read"),
			breadcrumb("show", "fizzy card show <card_number>", "View card"),
		}

		hasNext := linkNext != ""
		if hasNext {
			nextPage := notificationListPage + 1
			if notificationListPage == 0 {
				nextPage = 2
			}
			breadcrumbs = append(breadcrumbs, breadcrumb("next", fmt.Sprintf("fizzy notification list --page %d", nextPage), "Next page"))
		}

		printListPaginated(items, notificationColumns, hasNext, linkNext, notificationListAll, summary, breadcrumbs)
		return nil
	},
}

var notificationReadCmd = &cobra.Command{
	Use:   "read NOTIFICATION_ID",
	Short: "Mark notification as read",
	Long:  "Marks a notification as read.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		_, err := getSDK().Notifications().Read(cmd.Context(), args[0])
		if err != nil {
			return convertSDKError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("notifications", "fizzy notification list", "List notifications"),
		}

		printMutation(map[string]any{}, "", breadcrumbs)
		return nil
	},
}

var notificationUnreadCmd = &cobra.Command{
	Use:   "unread NOTIFICATION_ID",
	Short: "Mark notification as unread",
	Long:  "Marks a notification as unread.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		_, err := getSDK().Notifications().Unread(cmd.Context(), args[0])
		if err != nil {
			return convertSDKError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("notifications", "fizzy notification list", "List notifications"),
		}

		printMutation(map[string]any{}, "", breadcrumbs)
		return nil
	},
}

var notificationReadAllCmd = &cobra.Command{
	Use:   "read-all",
	Short: "Mark all notifications as read",
	Long:  "Marks all notifications as read.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		_, err := getSDK().Notifications().BulkRead(cmd.Context(), nil)
		if err != nil {
			return convertSDKError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("notifications", "fizzy notification list", "List notifications"),
		}

		printMutation(map[string]any{}, "", breadcrumbs)
		return nil
	},
}

// Notification tray flags
var notificationTrayIncludeRead bool

var notificationTrayCmd = &cobra.Command{
	Use:   "tray",
	Short: "Show notification tray",
	Long:  "Shows your notification tray (up to 100 unread notifications). Use --include-read to also include read notifications.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		var includeRead *bool
		if notificationTrayIncludeRead {
			t := true
			includeRead = &t
		}
		data, _, err := getSDK().Notifications().GetTray(cmd.Context(), includeRead)
		if err != nil {
			return convertSDKError(err)
		}
		items := normalizeAny(data)

		// Build summary
		count := dataCount(items)
		unreadCount := 0
		for _, item := range toSliceAny(items) {
			if notif, ok := item.(map[string]any); ok {
				if read, ok := notif["read"].(bool); ok && !read {
					unreadCount++
				}
			}
		}
		summary := fmt.Sprintf("%d notifications (%d unread)", count, unreadCount)

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("read", "fizzy notification read <id>", "Mark as read"),
			breadcrumb("read-all", "fizzy notification read-all", "Mark all as read"),
			breadcrumb("list", "fizzy notification list", "List all notifications"),
		}

		printList(items, notificationColumns, summary, breadcrumbs)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(notificationCmd)

	// List
	notificationListCmd.Flags().IntVar(&notificationListPage, "page", 0, "Page number")
	notificationListCmd.Flags().BoolVar(&notificationListAll, "all", false, "Fetch all pages")
	notificationCmd.AddCommand(notificationListCmd)

	// Tray
	notificationTrayCmd.Flags().BoolVar(&notificationTrayIncludeRead, "include-read", false, "Include read notifications")
	notificationCmd.AddCommand(notificationTrayCmd)

	// Read/Unread
	notificationCmd.AddCommand(notificationReadCmd)
	notificationCmd.AddCommand(notificationUnreadCmd)
	notificationCmd.AddCommand(notificationReadAllCmd)
}
