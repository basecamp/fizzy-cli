package commands

import (
	"fmt"
	"os"

	"github.com/basecamp/fizzy-cli/internal/response"
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
	Run: func(cmd *cobra.Command, args []string) {
		if err := requireAuthAndAccount(); err != nil {
			exitWithError(err)
		}

		if commentListCard == "" {
			exitWithError(newRequiredFlagError("card"))
		}

		client := getClient()
		path := "/cards/" + commentListCard + "/comments.json"
		if commentListPage > 0 {
			path += "?page=" + string(rune(commentListPage+'0'))
		}

		resp, err := client.GetWithPagination(path, commentListAll)
		if err != nil {
			exitWithError(err)
		}

		// Build summary
		count := 0
		if arr, ok := resp.Data.([]interface{}); ok {
			count = len(arr)
		}
		summary := fmt.Sprintf("%d comments on card #%s", count, commentListCard)
		if commentListAll {
			summary += " (all)"
		} else if commentListPage > 0 {
			summary += fmt.Sprintf(" (page %d)", commentListPage)
		}

		// Build breadcrumbs
		breadcrumbs := []response.Breadcrumb{
			breadcrumb("add", fmt.Sprintf("fizzy comment create --card %s --body \"text\"", commentListCard), "Add comment"),
			breadcrumb("react", fmt.Sprintf("fizzy reaction create --card %s --comment <id> --content \"👍\"", commentListCard), "Add reaction"),
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", commentListCard), "View card"),
		}

		hasNext := resp.LinkNext != ""
		printSuccessWithPaginationAndBreadcrumbs(resp.Data, hasNext, resp.LinkNext, summary, breadcrumbs)
	},
}

// Comment show flags
var commentShowCard string

var commentShowCmd = &cobra.Command{
	Use:   "show COMMENT_ID",
	Short: "Show a comment",
	Long:  "Shows details of a specific comment.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := requireAuthAndAccount(); err != nil {
			exitWithError(err)
		}

		if commentShowCard == "" {
			exitWithError(newRequiredFlagError("card"))
		}

		commentID := args[0]
		cardNumber := commentShowCard

		client := getClient()
		resp, err := client.Get("/cards/" + cardNumber + "/comments/" + commentID + ".json")
		if err != nil {
			exitWithError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []response.Breadcrumb{
			breadcrumb("update", fmt.Sprintf("fizzy comment update %s --card %s", commentID, cardNumber), "Edit comment"),
			breadcrumb("react", fmt.Sprintf("fizzy reaction create --card %s --comment %s --content \"👍\"", cardNumber, commentID), "Add reaction"),
			breadcrumb("comments", fmt.Sprintf("fizzy comment list --card %s", cardNumber), "List comments"),
		}

		printSuccessWithBreadcrumbs(resp.Data, "", breadcrumbs)
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
	Run: func(cmd *cobra.Command, args []string) {
		if err := requireAuthAndAccount(); err != nil {
			exitWithError(err)
		}

		if commentCreateCard == "" {
			exitWithError(newRequiredFlagError("card"))
		}

		// Determine body content
		var body string
		if commentCreateBodyFile != "" {
			content, err := os.ReadFile(commentCreateBodyFile)
			if err != nil {
				exitWithError(err)
			}
			body = markdownToHTML(string(content))
		} else if commentCreateBody != "" {
			body = markdownToHTML(commentCreateBody)
		} else {
			exitWithError(newRequiredFlagError("body or body_file"))
		}

		commentParams := map[string]interface{}{
			"body": body,
		}
		if commentCreateCreatedAt != "" {
			commentParams["created_at"] = commentCreateCreatedAt
		}

		reqBody := map[string]interface{}{
			"comment": commentParams,
		}

		cardNumber := commentCreateCard

		client := getClient()
		resp, err := client.Post("/cards/"+cardNumber+"/comments.json", reqBody)
		if err != nil {
			exitWithError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []response.Breadcrumb{
			breadcrumb("comments", fmt.Sprintf("fizzy comment list --card %s", cardNumber), "List comments"),
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
		}

		// Create returns location header - follow it to get the created resource
		if resp.Location != "" {
			followResp, err := client.FollowLocation(resp.Location)
			if err == nil && followResp != nil {
				respObj := response.SuccessWithBreadcrumbs(followResp.Data, "", breadcrumbs)
				respObj.Location = resp.Location
				if lastResult != nil {
					lastResult.Response = respObj
					lastResult.ExitCode = 0
					panic(testExitSignal{})
				}
				respObj.Print()
				os.Exit(0)
				return
			}
			printSuccessWithLocation(resp.Location)
			return
		}

		printSuccessWithBreadcrumbs(resp.Data, "", breadcrumbs)
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
	Run: func(cmd *cobra.Command, args []string) {
		if err := requireAuthAndAccount(); err != nil {
			exitWithError(err)
		}

		if commentUpdateCard == "" {
			exitWithError(newRequiredFlagError("card"))
		}

		commentParams := make(map[string]interface{})

		if commentUpdateBodyFile != "" {
			content, err := os.ReadFile(commentUpdateBodyFile)
			if err != nil {
				exitWithError(err)
			}
			commentParams["body"] = markdownToHTML(string(content))
		} else if commentUpdateBody != "" {
			commentParams["body"] = markdownToHTML(commentUpdateBody)
		}

		reqBody := map[string]interface{}{
			"comment": commentParams,
		}

		commentID := args[0]
		cardNumber := commentUpdateCard

		client := getClient()
		resp, err := client.Patch("/cards/"+cardNumber+"/comments/"+commentID+".json", reqBody)
		if err != nil {
			exitWithError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []response.Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy comment show %s --card %s", commentID, cardNumber), "View comment"),
			breadcrumb("comments", fmt.Sprintf("fizzy comment list --card %s", cardNumber), "List comments"),
		}

		printSuccessWithBreadcrumbs(resp.Data, "", breadcrumbs)
	},
}

// Comment delete flags
var commentDeleteCard string

var commentDeleteCmd = &cobra.Command{
	Use:   "delete COMMENT_ID",
	Short: "Delete a comment",
	Long:  "Deletes a comment from a card.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := requireAuthAndAccount(); err != nil {
			exitWithError(err)
		}

		if commentDeleteCard == "" {
			exitWithError(newRequiredFlagError("card"))
		}

		cardNumber := commentDeleteCard

		client := getClient()
		_, err := client.Delete("/cards/" + cardNumber + "/comments/" + args[0] + ".json")
		if err != nil {
			exitWithError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []response.Breadcrumb{
			breadcrumb("comments", fmt.Sprintf("fizzy comment list --card %s", cardNumber), "List comments"),
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
		}

		printSuccessWithBreadcrumbs(map[string]interface{}{
			"deleted": true,
		}, "", breadcrumbs)
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
