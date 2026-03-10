package commands

import (
	"fmt"
	"os"
	"strconv"

	"github.com/basecamp/fizzy-sdk/go/pkg/generated"
	"github.com/spf13/cobra"
)

var commentCmd = &cobra.Command{
	Use:   "comment",
	Short: "Manage comments",
	Long:  "Commands for managing card comments.",
}

// Comment list flags
var commentListCard string
var commentListPage int
var commentListAll bool

var commentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List comments for a card",
	Long:  "Lists all comments for a specific card.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}
		if err := checkLimitAll(commentListAll); err != nil {
			return err
		}

		if commentListCard == "" {
			return newRequiredFlagError("card")
		}

		ac := getSDK()
		var items any
		var linkNext string

		path := "/cards/" + commentListCard + "/comments.json"
		if commentListPage > 0 {
			path += "?page=" + strconv.Itoa(commentListPage)
		}

		if commentListAll {
			pages, err := ac.GetAll(cmd.Context(), path)
			if err != nil {
				return convertSDKError(err)
			}
			items = jsonAnySlice(pages)
		} else {
			listPath := ""
			if commentListPage > 0 {
				listPath = path
			}
			data, resp, err := ac.Comments().List(cmd.Context(), commentListCard, listPath)
			if err != nil {
				return convertSDKError(err)
			}
			items = normalizeAny(data)
			linkNext = parseSDKLinkNext(resp)
		}

		// Build summary
		count := dataCount(items)
		summary := fmt.Sprintf("%d comments on card #%s", count, commentListCard)
		if commentListAll {
			summary += " (all)"
		} else if commentListPage > 0 {
			summary += fmt.Sprintf(" (page %d)", commentListPage)
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("add", fmt.Sprintf("fizzy comment create --card %s --body \"text\"", commentListCard), "Add comment"),
			breadcrumb("react", fmt.Sprintf("fizzy reaction create --card %s --comment <id> --content \"👍\"", commentListCard), "Add reaction"),
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", commentListCard), "View card"),
		}

		hasNext := linkNext != ""
		printListPaginated(items, commentColumns, hasNext, linkNext, commentListAll, summary, breadcrumbs)
		return nil
	},
}

// Comment show flags
var commentShowCard string

var commentShowCmd = &cobra.Command{
	Use:   "show COMMENT_ID",
	Short: "Show a comment",
	Long:  "Shows details of a specific comment.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		if commentShowCard == "" {
			return newRequiredFlagError("card")
		}

		commentID := args[0]
		cardNumber := commentShowCard

		data, _, err := getSDK().Comments().Get(cmd.Context(), cardNumber, commentID)
		if err != nil {
			return convertSDKError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("update", fmt.Sprintf("fizzy comment update %s --card %s", commentID, cardNumber), "Edit comment"),
			breadcrumb("react", fmt.Sprintf("fizzy reaction create --card %s --comment %s --content \"👍\"", cardNumber, commentID), "Add reaction"),
			breadcrumb("comments", fmt.Sprintf("fizzy comment list --card %s", cardNumber), "List comments"),
		}

		printDetail(normalizeAny(data), "", breadcrumbs)
		return nil
	},
}

// Comment create flags
var commentCreateCard string
var commentCreateBody string
var commentCreateBodyFile string
var commentCreateCreatedAt string

var commentCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a comment",
	Long:  "Creates a new comment on a card.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		if commentCreateCard == "" {
			return newRequiredFlagError("card")
		}

		// Determine body content
		var body string
		if commentCreateBodyFile != "" {
			content, err := os.ReadFile(commentCreateBodyFile)
			if err != nil {
				return err
			}
			body = markdownToHTML(string(content))
		} else if commentCreateBody != "" {
			body = markdownToHTML(commentCreateBody)
		} else {
			return newRequiredFlagError("body or body_file")
		}

		cardNumber := commentCreateCard
		ac := getSDK()

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("comments", fmt.Sprintf("fizzy comment list --card %s", cardNumber), "List comments"),
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
		}

		req := &generated.CreateCommentRequest{Body: body}
		if commentCreateCreatedAt != "" {
			req.CreatedAt = commentCreateCreatedAt
		}
		data, resp, err := ac.Comments().Create(cmd.Context(), cardNumber, req)
		if err != nil {
			return convertSDKError(err)
		}

		items := normalizeAny(data)
		if location := resp.Headers.Get("Location"); location != "" {
			printMutationWithLocation(items, location, breadcrumbs)
		} else {
			printMutation(items, "", breadcrumbs)
		}
		return nil
	},
}

// Comment update flags
var commentUpdateCard string
var commentUpdateBody string
var commentUpdateBodyFile string

var commentUpdateCmd = &cobra.Command{
	Use:   "update COMMENT_ID",
	Short: "Update a comment",
	Long:  "Updates an existing comment.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		if commentUpdateCard == "" {
			return newRequiredFlagError("card")
		}

		var body string
		if commentUpdateBodyFile != "" {
			content, err := os.ReadFile(commentUpdateBodyFile)
			if err != nil {
				return err
			}
			body = markdownToHTML(string(content))
		} else if commentUpdateBody != "" {
			body = markdownToHTML(commentUpdateBody)
		}

		commentID := args[0]
		cardNumber := commentUpdateCard

		req := &generated.UpdateCommentRequest{}
		if body != "" {
			req.Body = body
		}
		data, _, err := getSDK().Comments().Update(cmd.Context(), cardNumber, commentID, req)
		if err != nil {
			return convertSDKError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy comment show %s --card %s", commentID, cardNumber), "View comment"),
			breadcrumb("comments", fmt.Sprintf("fizzy comment list --card %s", cardNumber), "List comments"),
		}

		printMutation(normalizeAny(data), "", breadcrumbs)
		return nil
	},
}

// Comment delete flags
var commentDeleteCard string

var commentDeleteCmd = &cobra.Command{
	Use:   "delete COMMENT_ID",
	Short: "Delete a comment",
	Long:  "Deletes a comment from a card.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		if commentDeleteCard == "" {
			return newRequiredFlagError("card")
		}

		cardNumber := commentDeleteCard

		_, err := getSDK().Comments().Delete(cmd.Context(), cardNumber, args[0])
		if err != nil {
			return convertSDKError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("comments", fmt.Sprintf("fizzy comment list --card %s", cardNumber), "List comments"),
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
		}

		printMutation(map[string]any{
			"deleted": true,
		}, "", breadcrumbs)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(commentCmd)

	// List
	commentListCmd.Flags().StringVar(&commentListCard, "card", "", "Card number (required)")
	commentListCmd.Flags().IntVar(&commentListPage, "page", 0, "Page number")
	commentListCmd.Flags().BoolVar(&commentListAll, "all", false, "Fetch all pages")
	commentCmd.AddCommand(commentListCmd)

	// Show
	commentShowCmd.Flags().StringVar(&commentShowCard, "card", "", "Card number (required)")
	commentCmd.AddCommand(commentShowCmd)

	// Create
	commentCreateCmd.Flags().StringVar(&commentCreateCard, "card", "", "Card number (required)")
	commentCreateCmd.Flags().StringVar(&commentCreateBody, "body", "", "Comment body (HTML)")
	commentCreateCmd.Flags().StringVar(&commentCreateBodyFile, "body_file", "", "Read body from file")
	commentCreateCmd.Flags().StringVar(&commentCreateCreatedAt, "created-at", "", "Custom created_at timestamp")
	commentCmd.AddCommand(commentCreateCmd)

	// Update
	commentUpdateCmd.Flags().StringVar(&commentUpdateCard, "card", "", "Card number (required)")
	commentUpdateCmd.Flags().StringVar(&commentUpdateBody, "body", "", "Comment body (HTML)")
	commentUpdateCmd.Flags().StringVar(&commentUpdateBodyFile, "body_file", "", "Read body from file")
	commentCmd.AddCommand(commentUpdateCmd)

	// Delete
	commentDeleteCmd.Flags().StringVar(&commentDeleteCard, "card", "", "Card number (required)")
	commentCmd.AddCommand(commentDeleteCmd)
}
