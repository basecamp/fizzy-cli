package commands

import (
	"fmt"

	"github.com/basecamp/fizzy-sdk/go/pkg/generated"
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

		ac := getSDK()
		var data any

		if reactionListComment != "" {
			raw, _, err := ac.Reactions().ListComment(cmd.Context(), reactionListCard, reactionListComment)
			if err != nil {
				return convertSDKError(err)
			}
			data = normalizeAny(raw)
		} else {
			raw, _, err := ac.Reactions().ListCard(cmd.Context(), reactionListCard)
			if err != nil {
				return convertSDKError(err)
			}
			data = normalizeAny(raw)
		}

		// Build summary
		count := dataCount(data)
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

		printList(data, reactionColumns, summary, breadcrumbs)
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

		ac := getSDK()

		var result any
		if reactionCreateComment != "" {
			req := &generated.CreateCommentReactionRequest{Content: reactionCreateContent}
			raw, _, err := ac.Reactions().CreateComment(cmd.Context(), reactionCreateCard, reactionCreateComment, req)
			if err != nil {
				return convertSDKError(err)
			}
			result = normalizeAny(raw)
		} else {
			req := &generated.CreateCardReactionRequest{Content: reactionCreateContent}
			raw, _, err := ac.Reactions().CreateCard(cmd.Context(), reactionCreateCard, req)
			if err != nil {
				return convertSDKError(err)
			}
			result = normalizeAny(raw)
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

		if result == nil {
			result = map[string]any{}
		}
		printMutation(result, "", breadcrumbs)
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

		ac := getSDK()

		if reactionDeleteComment != "" {
			_, err := ac.Reactions().DeleteComment(cmd.Context(), reactionDeleteCard, reactionDeleteComment, args[0])
			if err != nil {
				return convertSDKError(err)
			}
		} else {
			_, err := ac.Reactions().DeleteCard(cmd.Context(), reactionDeleteCard, args[0])
			if err != nil {
				return convertSDKError(err)
			}
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

		printMutation(map[string]any{
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
