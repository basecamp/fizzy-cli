package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var validWebhookActions = []string{
	"card_assigned",
	"card_closed",
	"card_postponed",
	"card_auto_postponed",
	"card_board_changed",
	"card_published",
	"card_reopened",
	"card_sent_back_to_triage",
	"card_triaged",
	"card_unassigned",
	"comment_created",
}

var webhookCmd = &cobra.Command{
	Use:   "webhook",
	Short: "Manage webhooks",
	Long:  "Commands for managing webhooks on a board. Requires account admin access.",
}

// Webhook list flags
var webhookListBoard string
var webhookListPage int
var webhookListAll bool

var webhookListCmd = &cobra.Command{
	Use:   "list",
	Short: "List webhooks for a board",
	Long:  "Lists all webhooks configured on a board.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}
		if err := checkLimitAll(webhookListAll); err != nil {
			return err
		}

		boardID, err := requireBoard(webhookListBoard)
		if err != nil {
			return err
		}

		client := getClient()
		path := fmt.Sprintf("/boards/%s/webhooks.json", boardID)
		if webhookListPage > 0 {
			path += fmt.Sprintf("?page=%d", webhookListPage)
		}

		resp, err := client.GetWithPagination(path, webhookListAll)
		if err != nil {
			return err
		}

		count := 0
		if arr, ok := resp.Data.([]any); ok {
			count = len(arr)
		}
		summary := fmt.Sprintf("%d webhooks", count)
		if webhookListAll {
			summary += " (all)"
		} else if webhookListPage > 0 {
			summary += fmt.Sprintf(" (page %d)", webhookListPage)
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("show", "fizzy webhook show --board <id> WEBHOOK_ID", "View webhook details"),
			breadcrumb("create", fmt.Sprintf("fizzy webhook create --board %s --name \"name\" --url \"url\"", boardID), "Create webhook"),
		}

		hasNext := resp.LinkNext != ""
		if hasNext {
			nextPage := webhookListPage + 1
			if webhookListPage == 0 {
				nextPage = 2
			}
			breadcrumbs = append(breadcrumbs, breadcrumb("next", fmt.Sprintf("fizzy webhook list --board %s --page %d", boardID, nextPage), "Next page"))
		}

		printListPaginated(resp.Data, webhookColumns, hasNext, resp.LinkNext, webhookListAll, summary, breadcrumbs)
		return nil
	},
}

// Webhook deliveries flags
var webhookDeliveriesBoard string
var webhookDeliveriesPage int
var webhookDeliveriesAll bool

var webhookDeliveriesCmd = &cobra.Command{
	Use:   "deliveries WEBHOOK_ID",
	Short: "List webhook deliveries",
	Long:  "Lists deliveries for a webhook.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}
		if err := checkLimitAll(webhookDeliveriesAll); err != nil {
			return err
		}

		boardID, err := requireBoard(webhookDeliveriesBoard)
		if err != nil {
			return err
		}

		webhookID := args[0]
		ac := getSDK()
		path := fmt.Sprintf("/boards/%s/webhooks/%s/deliveries.json", boardID, webhookID)
		if webhookDeliveriesPage > 0 {
			path += fmt.Sprintf("?page=%d", webhookDeliveriesPage)
		}

		var items any
		var linkNext string

		if webhookDeliveriesAll {
			pages, err := ac.GetAll(cmd.Context(), path)
			if err != nil {
				return convertSDKError(err)
			}
			items = jsonAnySlice(pages)
		} else {
			data, resp, err := ac.Webhooks().ListWebhookDeliveries(cmd.Context(), boardID, webhookID, path)
			if err != nil {
				return convertSDKError(err)
			}
			items = normalizeAny(data)
			linkNext = parseSDKLinkNext(resp)
		}

		count := dataCount(items)
		summary := fmt.Sprintf("%d webhook deliveries", count)
		if webhookDeliveriesAll {
			summary += " (all)"
		} else if webhookDeliveriesPage > 0 {
			summary += fmt.Sprintf(" (page %d)", webhookDeliveriesPage)
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("webhook", fmt.Sprintf("fizzy webhook show --board %s %s", boardID, webhookID), "View webhook"),
			breadcrumb("webhooks", fmt.Sprintf("fizzy webhook list --board %s", boardID), "List webhooks"),
		}

		hasNext := linkNext != ""
		if hasNext {
			nextPage := webhookDeliveriesPage + 1
			if webhookDeliveriesPage == 0 {
				nextPage = 2
			}
			breadcrumbs = append(breadcrumbs, breadcrumb("next", fmt.Sprintf("fizzy webhook deliveries --board %s %s --page %d", boardID, webhookID, nextPage), "Next page"))
		}

		printListPaginated(items, webhookDeliveryColumns, hasNext, linkNext, webhookDeliveriesAll, summary, breadcrumbs)
		return nil
	},
}

// Webhook show
var webhookShowBoard string

var webhookShowCmd = &cobra.Command{
	Use:   "show WEBHOOK_ID",
	Short: "Show a webhook",
	Long:  "Shows details of a specific webhook.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		boardID, err := requireBoard(webhookShowBoard)
		if err != nil {
			return err
		}

		webhookID := args[0]

		client := getClient()
		resp, err := client.Get(fmt.Sprintf("/boards/%s/webhooks/%s.json", boardID, webhookID))
		if err != nil {
			return err
		}

		summary := "Webhook"
		if wh, ok := resp.Data.(map[string]any); ok {
			if name, ok := wh["name"].(string); ok {
				summary = fmt.Sprintf("Webhook: %s", name)
			}
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("update", fmt.Sprintf("fizzy webhook update --board %s %s --name \"name\"", boardID, webhookID), "Update webhook"),
			breadcrumb("delete", fmt.Sprintf("fizzy webhook delete --board %s %s", boardID, webhookID), "Delete webhook"),
			breadcrumb("reactivate", fmt.Sprintf("fizzy webhook reactivate --board %s %s", boardID, webhookID), "Reactivate webhook"),
		}

		printDetail(resp.Data, summary, breadcrumbs)
		return nil
	},
}

// Webhook create flags
var webhookCreateBoard string
var webhookCreateName string
var webhookCreateURL string
var webhookCreateActions []string

var webhookCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a webhook",
	Long:  "Creates a new webhook on a board.\n\nSupported actions: " + strings.Join(validWebhookActions, ", "),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		boardID, err := requireBoard(webhookCreateBoard)
		if err != nil {
			return err
		}

		if webhookCreateName == "" {
			return newRequiredFlagError("name")
		}
		if webhookCreateURL == "" {
			return newRequiredFlagError("url")
		}

		webhookParams := map[string]any{
			"name": webhookCreateName,
			"url":  webhookCreateURL,
		}

		if len(webhookCreateActions) > 0 {
			webhookParams["subscribed_actions"] = webhookCreateActions
		}

		body := map[string]any{
			"webhook": webhookParams,
		}

		client := getClient()
		resp, err := client.Post(fmt.Sprintf("/boards/%s/webhooks.json", boardID), body)
		if err != nil {
			return err
		}

		if resp.Location != "" {
			followResp, err := client.FollowLocation(resp.Location)
			if err == nil && followResp != nil {
				webhookID := ""
				if wh, ok := followResp.Data.(map[string]any); ok {
					if id, ok := wh["id"].(string); ok {
						webhookID = id
					}
				}

				var breadcrumbs []Breadcrumb
				if webhookID != "" {
					breadcrumbs = []Breadcrumb{
						breadcrumb("show", fmt.Sprintf("fizzy webhook show --board %s %s", boardID, webhookID), "View webhook"),
						breadcrumb("update", fmt.Sprintf("fizzy webhook update --board %s %s --name \"name\"", boardID, webhookID), "Update webhook"),
					}
				}

				printMutationWithLocation(followResp.Data, resp.Location, breadcrumbs)
				return nil
			}
			printSuccessWithLocation(resp.Location)
			return nil
		}

		printSuccess(resp.Data)
		return nil
	},
}

// Webhook update flags
var webhookUpdateBoard string
var webhookUpdateName string
var webhookUpdateActions []string

var webhookUpdateCmd = &cobra.Command{
	Use:   "update WEBHOOK_ID",
	Short: "Update a webhook",
	Long:  "Updates an existing webhook. Note: URL is immutable after creation.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		boardID, err := requireBoard(webhookUpdateBoard)
		if err != nil {
			return err
		}

		webhookID := args[0]

		webhookParams := make(map[string]any)

		if webhookUpdateName != "" {
			webhookParams["name"] = webhookUpdateName
		}
		if len(webhookUpdateActions) > 0 {
			webhookParams["subscribed_actions"] = webhookUpdateActions
		}

		body := map[string]any{
			"webhook": webhookParams,
		}

		client := getClient()
		resp, err := client.Patch(fmt.Sprintf("/boards/%s/webhooks/%s.json", boardID, webhookID), body)
		if err != nil {
			return err
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy webhook show --board %s %s", boardID, webhookID), "View webhook"),
			breadcrumb("delete", fmt.Sprintf("fizzy webhook delete --board %s %s", boardID, webhookID), "Delete webhook"),
		}

		printMutation(resp.Data, "", breadcrumbs)
		return nil
	},
}

// Webhook delete
var webhookDeleteBoard string

var webhookDeleteCmd = &cobra.Command{
	Use:   "delete WEBHOOK_ID",
	Short: "Delete a webhook",
	Long:  "Deletes a webhook from a board.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		boardID, err := requireBoard(webhookDeleteBoard)
		if err != nil {
			return err
		}

		client := getClient()
		_, err = client.Delete(fmt.Sprintf("/boards/%s/webhooks/%s.json", boardID, args[0]))
		if err != nil {
			return err
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("webhooks", fmt.Sprintf("fizzy webhook list --board %s", boardID), "List webhooks"),
			breadcrumb("create", fmt.Sprintf("fizzy webhook create --board %s --name \"name\" --url \"url\"", boardID), "Create new webhook"),
		}

		printMutation(map[string]any{
			"deleted": true,
		}, "", breadcrumbs)
		return nil
	},
}

// Webhook reactivate
var webhookReactivateBoard string

var webhookReactivateCmd = &cobra.Command{
	Use:   "reactivate WEBHOOK_ID",
	Short: "Reactivate a webhook",
	Long:  "Reactivates a deactivated webhook.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		boardID, err := requireBoard(webhookReactivateBoard)
		if err != nil {
			return err
		}

		webhookID := args[0]

		client := getClient()
		resp, err := client.Post(fmt.Sprintf("/boards/%s/webhooks/%s/activation.json", boardID, webhookID), nil)
		if err != nil {
			return err
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy webhook show --board %s %s", boardID, webhookID), "View webhook"),
			breadcrumb("webhooks", fmt.Sprintf("fizzy webhook list --board %s", boardID), "List webhooks"),
		}

		printMutation(resp.Data, "", breadcrumbs)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(webhookCmd)

	// List
	webhookListCmd.Flags().StringVar(&webhookListBoard, "board", "", "Board ID (required)")
	webhookListCmd.Flags().IntVar(&webhookListPage, "page", 0, "Page number")
	webhookListCmd.Flags().BoolVar(&webhookListAll, "all", false, "Fetch all pages")
	webhookCmd.AddCommand(webhookListCmd)

	// Deliveries
	webhookDeliveriesCmd.Flags().StringVar(&webhookDeliveriesBoard, "board", "", "Board ID (required)")
	webhookDeliveriesCmd.Flags().IntVar(&webhookDeliveriesPage, "page", 0, "Page number")
	webhookDeliveriesCmd.Flags().BoolVar(&webhookDeliveriesAll, "all", false, "Fetch all pages")
	webhookCmd.AddCommand(webhookDeliveriesCmd)

	// Show
	webhookShowCmd.Flags().StringVar(&webhookShowBoard, "board", "", "Board ID (required)")
	webhookCmd.AddCommand(webhookShowCmd)

	// Create
	webhookCreateCmd.Flags().StringVar(&webhookCreateBoard, "board", "", "Board ID (required)")
	webhookCreateCmd.Flags().StringVar(&webhookCreateName, "name", "", "Webhook name (required)")
	webhookCreateCmd.Flags().StringVar(&webhookCreateURL, "url", "", "Payload URL (required)")
	webhookCreateCmd.Flags().StringSliceVar(&webhookCreateActions, "actions", nil, "Subscribed actions (comma-separated: "+strings.Join(validWebhookActions, ", ")+")")
	webhookCmd.AddCommand(webhookCreateCmd)

	// Update
	webhookUpdateCmd.Flags().StringVar(&webhookUpdateBoard, "board", "", "Board ID (required)")
	webhookUpdateCmd.Flags().StringVar(&webhookUpdateName, "name", "", "Webhook name")
	webhookUpdateCmd.Flags().StringSliceVar(&webhookUpdateActions, "actions", nil, "Subscribed actions (comma-separated: "+strings.Join(validWebhookActions, ", ")+")")
	webhookCmd.AddCommand(webhookUpdateCmd)

	// Delete
	webhookDeleteCmd.Flags().StringVar(&webhookDeleteBoard, "board", "", "Board ID (required)")
	webhookCmd.AddCommand(webhookDeleteCmd)

	// Reactivate
	webhookReactivateCmd.Flags().StringVar(&webhookReactivateBoard, "board", "", "Board ID (required)")
	webhookCmd.AddCommand(webhookReactivateCmd)
}
