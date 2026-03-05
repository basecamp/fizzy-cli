package commands

import (
	"fmt"
	"strconv"

	"github.com/basecamp/fizzy-cli/internal/errors"
	"github.com/spf13/cobra"
)

// CommentAttachment extends Attachment with comment context
type CommentAttachment struct {
	Attachment
	CommentID string `json:"comment_id"`
}

var commentAttachmentsCmd = &cobra.Command{
	Use:   "attachments",
	Short: "Manage comment attachments",
	Long:  "Commands for viewing and downloading attachments embedded in comments.",
}

// Comment attachments show flags
var commentAttachmentsShowCard string

var commentAttachmentsShowCmd = &cobra.Command{
	Use:   "show",
	Short: "List attachments in comments",
	Long:  "Lists all attachments embedded in comment bodies for a card.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		if commentAttachmentsShowCard == "" {
			return newRequiredFlagError("card")
		}

		client := getClient()
		path := "/cards/" + commentAttachmentsShowCard + "/comments.json"
		resp, err := client.GetWithPagination(path, true)
		if err != nil {
			return err
		}

		comments, ok := resp.Data.([]any)
		if !ok {
			return errors.NewError("Invalid comments response")
		}

		attachments := extractCommentAttachments(comments)

		summary := fmt.Sprintf("%d attachments across %d comments on card #%s", len(attachments), len(comments), commentAttachmentsShowCard)

		breadcrumbs := []Breadcrumb{
			breadcrumb("download", fmt.Sprintf("fizzy comment attachments download --card %s", commentAttachmentsShowCard), "Download attachments"),
			breadcrumb("comments", fmt.Sprintf("fizzy comment list --card %s", commentAttachmentsShowCard), "List comments"),
			breadcrumb("card-attachments", fmt.Sprintf("fizzy card attachments show %s", commentAttachmentsShowCard), "Card attachments"),
		}

		printSuccessWithBreadcrumbs(attachments, summary, breadcrumbs)
		return nil
	},
}

// Comment attachments download flags
var commentAttachmentsDownloadCard string
var commentAttachmentsDownloadOutput string

var commentAttachmentsDownloadCmd = &cobra.Command{
	Use:   "download [ATTACHMENT_INDEX]",
	Short: "Download attachments from comments",
	Long: `Downloads attachments embedded in comment bodies for a card.

If ATTACHMENT_INDEX is provided, downloads only that attachment (1-based index).
If ATTACHMENT_INDEX is omitted, downloads all comment attachments.

When downloading a single attachment, -o sets the exact output filename.
When downloading multiple attachments, -o sets a prefix (e.g. -o test produces test_1.png, test_2.png).

Use 'fizzy comment attachments show --card CARD_NUMBER' to see available attachments and their indices.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		if commentAttachmentsDownloadCard == "" {
			return newRequiredFlagError("card")
		}

		client := getClient()
		path := "/cards/" + commentAttachmentsDownloadCard + "/comments.json"
		resp, err := client.GetWithPagination(path, true)
		if err != nil {
			return err
		}

		comments, ok := resp.Data.([]any)
		if !ok {
			return errors.NewError("Invalid comments response")
		}

		attachments := extractCommentAttachments(comments)

		if len(attachments) == 0 {
			return errors.NewNotFoundError("No attachments found in comments on this card")
		}

		// Determine which attachments to download
		var toDownload []CommentAttachment
		if len(args) == 1 {
			attachmentIndex, err := strconv.Atoi(args[0])
			if err != nil {
				return errors.NewInvalidArgsError("attachment index must be a number")
			}
			if attachmentIndex < 1 || attachmentIndex > len(attachments) {
				return errors.NewInvalidArgsError("attachment index must be between 1 and " + strconv.Itoa(len(attachments)))
			}
			toDownload = []CommentAttachment{attachments[attachmentIndex-1]}
		} else {
			toDownload = attachments
		}

		// Download the files
		results := make([]map[string]any, 0, len(toDownload))
		for i, attachment := range toDownload {
			outputPath := buildOutputPath(commentAttachmentsDownloadOutput, attachment.Filename, i+1, len(toDownload))

			if err := client.DownloadFile(attachment.DownloadURL, outputPath); err != nil {
				return err
			}

			results = append(results, map[string]any{
				"filename":   attachment.Filename,
				"saved_to":   outputPath,
				"filesize":   attachment.Filesize,
				"comment_id": attachment.CommentID,
			})
		}

		printSuccess(map[string]any{
			"downloaded": len(results),
			"files":      results,
		})
		return nil
	},
}

// extractCommentAttachments parses all comments and returns attachments with comment context
func extractCommentAttachments(comments []any) []CommentAttachment {
	var allAttachments []CommentAttachment
	globalIndex := 1

	for _, c := range comments {
		comment, ok := c.(map[string]any)
		if !ok {
			continue
		}

		commentID, _ := comment["id"].(string)

		// Comment body is an object with html and plain_text fields
		bodyObj, ok := comment["body"].(map[string]any)
		if !ok {
			continue
		}

		bodyHTML, _ := bodyObj["html"].(string)
		if bodyHTML == "" {
			continue
		}

		attachments := parseAttachments(bodyHTML)
		for _, a := range attachments {
			a.Index = globalIndex
			globalIndex++
			allAttachments = append(allAttachments, CommentAttachment{
				Attachment: a,
				CommentID:  commentID,
			})
		}
	}

	return allAttachments
}

func init() {
	commentCmd.AddCommand(commentAttachmentsCmd)

	// Show
	commentAttachmentsShowCmd.Flags().StringVar(&commentAttachmentsShowCard, "card", "", "Card number (required)")
	commentAttachmentsCmd.AddCommand(commentAttachmentsShowCmd)

	// Download
	commentAttachmentsDownloadCmd.Flags().StringVar(&commentAttachmentsDownloadCard, "card", "", "Card number (required)")
	commentAttachmentsDownloadCmd.Flags().StringVarP(&commentAttachmentsDownloadOutput, "output", "o", "", "Output filename (single file) or prefix (multiple files, e.g. -o test produces test_1.png)")
	commentAttachmentsCmd.AddCommand(commentAttachmentsDownloadCmd)
}
