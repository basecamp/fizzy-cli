package commands

import (
	"fmt"
	"strings"

	"github.com/basecamp/fizzy-sdk/go/pkg/generated"
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

		ac := getSDK()
		var items any
		var linkNext string

		switch {
		case webhookListAll:
			path := fmt.Sprintf("/boards/%s/webhooks.json", boardID)
			if webhookListPage > 0 {
				path += fmt.Sprintf("?page=%d", webhookListPage)
			}
			pages, err := ac.GetAll(cmd.Context(), path)
			if err != nil {
				return convertSDKError(err)
			}
			items = jsonAnySlice(pages)
		case webhookListPage > 0:
			path := fmt.Sprintf("/boards/%s/webhooks.json?page=%d", boardID, webhookListPage)
			resp, err := ac.Get(cmd.Context(), path)
			if err != nil {
				return convertSDKError(err)
			}
			var list []map[string]any
			if err := resp.UnmarshalData(&list); err != nil {
				return convertSDKError(err)
			}
			items = toSliceAny(list)
			linkNext = parseSDKLinkNext(resp)
		default:
			data, resp, err := ac.Webhooks().List(cmd.Context(), boardID)
			if err != nil {
				return convertSDKError(err)
			}
			items = normalizeAny(data)
			linkNext = parseSDKLinkNext(resp)
		}

		count := dataCount(items)
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

		hasNext := linkNext != ""
		if hasNext {
			nextPage := webhookListPage + 1
			if webhookListPage == 0 {
				nextPage = 2
			}
			breadcrumbs = append(breadcrumbs, breadcrumb("next", fmt.Sprintf("fizzy webhook list --board %s --page %d", boardID, nextPage), "Next page"))
		}

		printListPaginated(items, webhookColumns, hasNext, linkNext, webhookListAll, summary, breadcrumbs)
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

		ac := getSDK()
		raw, _, err := ac.Webhooks().Get(cmd.Context(), boardID, webhookID)
		if err != nil {
			return convertSDKError(err)
		}

		data := normalizeAny(raw)

		summary := "Webhook"
		if wh, ok := data.(map[string]any); ok {
			if name, ok := wh["name"].(string); ok && name != "" {
				summary = fmt.Sprintf("Webhook: %s", name)
			}
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("update", fmt.Sprintf("fizzy webhook update --board %s %s --name \"name\"", boardID, webhookID), "Update webhook"),
			breadcrumb("delete", fmt.Sprintf("fizzy webhook delete --board %s %s", boardID, webhookID), "Delete webhook"),
			breadcrumb("reactivate", fmt.Sprintf("fizzy webhook reactivate --board %s %s", boardID, webhookID), "Reactivate webhook"),
		}

		printDetail(data, summary, breadcrumbs)
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

		ac := getSDK()
		req := &generated.CreateWebhookRequest{
			Name:              webhookCreateName,
			Url:               webhookCreateURL,
			SubscribedActions: webhookCreateActions,
		}

		raw, resp, err := ac.Webhooks().Create(cmd.Context(), boardID, req)
		if err != nil {
			return convertSDKError(err)
		}

		data := normalizeAny(raw)
		webhookID := ""
		if wh, ok := data.(map[string]any); ok {
			webhookID = getStringField(wh, "id")
		}

		var breadcrumbs []Breadcrumb
		if webhookID != "" {
			breadcrumbs = []Breadcrumb{
				breadcrumb("show", fmt.Sprintf("fizzy webhook show --board %s %s", boardID, webhookID), "View webhook"),
				breadcrumb("update", fmt.Sprintf("fizzy webhook update --board %s %s --name \"name\"", boardID, webhookID), "Update webhook"),
			}
		}

		if location := resp.Headers.Get("Location"); location != "" {
			printMutationWithLocation(data, location, breadcrumbs)
		} else {
			printMutation(data, "", breadcrumbs)
		}
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

		req := &generated.UpdateWebhookRequest{}
		if webhookUpdateName != "" {
			req.Name = webhookUpdateName
		}
		if len(webhookUpdateActions) > 0 {
			req.SubscribedActions = webhookUpdateActions
		}

		ac := getSDK()
		raw, _, err := ac.Webhooks().Update(cmd.Context(), boardID, webhookID, req)
		if err != nil {
			return convertSDKError(err)
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy webhook show --board %s %s", boardID, webhookID), "View webhook"),
			breadcrumb("delete", fmt.Sprintf("fizzy webhook delete --board %s %s", boardID, webhookID), "Delete webhook"),
		}

		printMutation(normalizeAny(raw), "", breadcrumbs)
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

		ac := getSDK()
		if _, err := ac.Webhooks().Delete(cmd.Context(), boardID, args[0]); err != nil {
			return convertSDKError(err)
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

		ac := getSDK()
		resp, err := ac.Webhooks().Activate(cmd.Context(), boardID, webhookID)
		if err != nil {
			return convertSDKError(err)
		}

		data := normalizeAny(resp.Data)
		if data == nil {
			data = map[string]any{"id": webhookID, "active": true}
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy webhook show --board %s %s", boardID, webhookID), "View webhook"),
			breadcrumb("webhooks", fmt.Sprintf("fizzy webhook list --board %s", boardID), "List webhooks"),
		}

		printMutation(data, "", breadcrumbs)
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
