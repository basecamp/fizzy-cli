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

		client := getClient()
		path := "/notifications.json"
		if notificationListPage > 0 {
			path += "?page=" + strconv.Itoa(notificationListPage)
		}

		resp, err := client.GetWithPagination(path, notificationListAll)
		if err != nil {
			return err
		}

		// Build summary with unread count
		count := 0
		unreadCount := 0
		if arr, ok := resp.Data.([]any); ok {
			count = len(arr)
			for _, item := range arr {
				if notif, ok := item.(map[string]any); ok {
					if read, ok := notif["read"].(bool); ok && !read {
						unreadCount++
					}
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

		hasNext := resp.LinkNext != ""
		if hasNext {
			nextPage := notificationListPage + 1
			if nextPage == 0 {
				nextPage = 2
			}
			breadcrumbs = append(breadcrumbs, breadcrumb("next", fmt.Sprintf("fizzy notification list --page %d", nextPage), "Next page"))
		}

		printListPaginated(resp.Data, notificationColumns, hasNext, resp.LinkNext, notificationListAll, summary, breadcrumbs)
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

		client := getClient()
		resp, err := client.Post("/notifications/"+args[0]+"/reading.json", nil)
		if err != nil {
			return err
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("notifications", "fizzy notification list", "List notifications"),
		}

		data := resp.Data
		if data == nil {
			data = map[string]any{}
		}
		printMutation(data, "", breadcrumbs)
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

		client := getClient()
		resp, err := client.Delete("/notifications/" + args[0] + "/reading.json")
		if err != nil {
			return err
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("notifications", "fizzy notification list", "List notifications"),
		}

		data := resp.Data
		if data == nil {
			data = map[string]any{}
		}
		printMutation(data, "", breadcrumbs)
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

		client := getClient()
		resp, err := client.Post("/notifications/bulk_reading.json", nil)
		if err != nil {
			return err
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("notifications", "fizzy notification list", "List notifications"),
		}

		printMutation(resp.Data, "", breadcrumbs)
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

		client := getClient()
		path := "/notifications/tray.json"
		if notificationTrayIncludeRead {
			path += "?include_read=true"
		}

		resp, err := client.Get(path)
		if err != nil {
			return err
		}

		// Build summary
		count := 0
		unreadCount := 0
		if arr, ok := resp.Data.([]any); ok {
			count = len(arr)
			for _, item := range arr {
				if notif, ok := item.(map[string]any); ok {
					if read, ok := notif["read"].(bool); ok && !read {
						unreadCount++
					}
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

		printList(resp.Data, notificationColumns, summary, breadcrumbs)
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
