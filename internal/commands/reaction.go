package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var reactionCmd = &cobra.Command{
	Use:   "reaction",
	Short: "Manage reactions",
	Long:  "Commands for managing reactions on cards and comments.",
}

// Reaction list flags
var reactionListCard string
var reactionListComment string

var reactionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List reactions",
	Long:  "Lists reactions on a card, or on a comment if --comment is provided.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		if reactionListCard == "" {
			return newRequiredFlagError("card")
		}

		// Build URL based on whether --comment was provided
		var path string
		if reactionListComment != "" {
			path = "/cards/" + reactionListCard + "/comments/" + reactionListComment + "/reactions.json"
		} else {
			path = "/cards/" + reactionListCard + "/reactions.json"
		}

		client := getClient()
		resp, err := client.Get(path)
		if err != nil {
			return err
		}

		// Build summary
		count := 0
		if arr, ok := resp.Data.([]any); ok {
			count = len(arr)
		}
		var summary string
		if reactionListComment != "" {
			summary = fmt.Sprintf("%d reactions on comment", count)
		} else {
			summary = fmt.Sprintf("%d reactions on card #%s", count, reactionListCard)
		}

		// Build breadcrumbs
		var breadcrumbs []Breadcrumb
		if reactionListComment != "" {
			breadcrumbs = []Breadcrumb{
				breadcrumb("react", fmt.Sprintf("fizzy reaction create --card %s --comment %s --content \"👍\"", reactionListCard, reactionListComment), "Add reaction"),
				breadcrumb("comment", fmt.Sprintf("fizzy comment show %s --card %s", reactionListComment, reactionListCard), "View comment"),
				breadcrumb("show", fmt.Sprintf("fizzy card show %s", reactionListCard), "View card"),
			}
		} else {
			breadcrumbs = []Breadcrumb{
				breadcrumb("react", fmt.Sprintf("fizzy reaction create --card %s --content \"👍\"", reactionListCard), "Add reaction"),
				breadcrumb("comments", fmt.Sprintf("fizzy comment list --card %s", reactionListCard), "View comments"),
				breadcrumb("show", fmt.Sprintf("fizzy card show %s", reactionListCard), "View card"),
			}
		}

		printSuccessWithBreadcrumbs(resp.Data, summary, breadcrumbs)
		return nil
	},
}

// Reaction create flags
var reactionCreateCard string
var reactionCreateComment string
var reactionCreateContent string

var reactionCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Add a reaction",
	Long:  "Adds a reaction to a card, or to a comment if --comment is provided.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		if reactionCreateCard == "" {
			return newRequiredFlagError("card")
		}
		if reactionCreateContent == "" {
			return newRequiredFlagError("content")
		}

		body := map[string]any{
			"content": reactionCreateContent,
		}

		// Build URL based on whether --comment was provided
		var path string
		if reactionCreateComment != "" {
			path = "/cards/" + reactionCreateCard + "/comments/" + reactionCreateComment + "/reactions.json"
		} else {
			path = "/cards/" + reactionCreateCard + "/reactions.json"
		}

		client := getClient()
		resp, err := client.Post(path, body)
		if err != nil {
			return err
		}

		// Build breadcrumbs
		var breadcrumbs []Breadcrumb
		if reactionCreateComment != "" {
			breadcrumbs = []Breadcrumb{
				breadcrumb("reactions", fmt.Sprintf("fizzy reaction list --card %s --comment %s", reactionCreateCard, reactionCreateComment), "List reactions"),
				breadcrumb("comment", fmt.Sprintf("fizzy comment show %s --card %s", reactionCreateComment, reactionCreateCard), "View comment"),
			}
		} else {
			breadcrumbs = []Breadcrumb{
				breadcrumb("reactions", fmt.Sprintf("fizzy reaction list --card %s", reactionCreateCard), "List reactions"),
				breadcrumb("show", fmt.Sprintf("fizzy card show %s", reactionCreateCard), "View card"),
			}
		}

		data := resp.Data
		if data == nil {
			data = map[string]any{}
		}
		printSuccessWithBreadcrumbs(data, "", breadcrumbs)
		return nil
	},
}

// Reaction delete flags
var reactionDeleteCard string
var reactionDeleteComment string

var reactionDeleteCmd = &cobra.Command{
	Use:   "delete REACTION_ID",
	Short: "Remove a reaction",
	Long:  "Removes a reaction from a card, or from a comment if --comment is provided.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		if reactionDeleteCard == "" {
			return newRequiredFlagError("card")
		}

		// Build URL based on whether --comment was provided
		var path string
		if reactionDeleteComment != "" {
			path = "/cards/" + reactionDeleteCard + "/comments/" + reactionDeleteComment + "/reactions/" + args[0] + ".json"
		} else {
			path = "/cards/" + reactionDeleteCard + "/reactions/" + args[0] + ".json"
		}

		client := getClient()
		_, err := client.Delete(path)
		if err != nil {
			return err
		}

		// Build breadcrumbs
		var breadcrumbs []Breadcrumb
		if reactionDeleteComment != "" {
			breadcrumbs = []Breadcrumb{
				breadcrumb("reactions", fmt.Sprintf("fizzy reaction list --card %s --comment %s", reactionDeleteCard, reactionDeleteComment), "List reactions"),
				breadcrumb("comment", fmt.Sprintf("fizzy comment show %s --card %s", reactionDeleteComment, reactionDeleteCard), "View comment"),
			}
		} else {
			breadcrumbs = []Breadcrumb{
				breadcrumb("reactions", fmt.Sprintf("fizzy reaction list --card %s", reactionDeleteCard), "List reactions"),
				breadcrumb("show", fmt.Sprintf("fizzy card show %s", reactionDeleteCard), "View card"),
			}
		}

		printSuccessWithBreadcrumbs(map[string]any{
			"deleted": true,
		}, "", breadcrumbs)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(reactionCmd)

	// List
	reactionListCmd.Flags().StringVar(&reactionListCard, "card", "", "Card number (required)")
	reactionListCmd.Flags().StringVar(&reactionListComment, "comment", "", "Comment ID (optional, for comment reactions)")
	reactionCmd.AddCommand(reactionListCmd)

	// Create
	reactionCreateCmd.Flags().StringVar(&reactionCreateCard, "card", "", "Card number (required)")
	reactionCreateCmd.Flags().StringVar(&reactionCreateComment, "comment", "", "Comment ID (optional, for comment reactions)")
	reactionCreateCmd.Flags().StringVar(&reactionCreateContent, "content", "", "Reaction content (required)")
	reactionCmd.AddCommand(reactionCreateCmd)

	// Delete
	reactionDeleteCmd.Flags().StringVar(&reactionDeleteCard, "card", "", "Card number (required)")
	reactionDeleteCmd.Flags().StringVar(&reactionDeleteComment, "comment", "", "Comment ID (optional, for comment reactions)")
	reactionCmd.AddCommand(reactionDeleteCmd)
}
