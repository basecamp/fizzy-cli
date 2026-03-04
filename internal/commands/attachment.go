package commands

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/basecamp/fizzy-cli/internal/errors"
	"github.com/spf13/cobra"
)

// Attachment represents a parsed attachment from description_html
type Attachment struct {
	Index       int    `json:"index"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Filesize    int64  `json:"filesize"`
	Width       int    `json:"width,omitempty"`
	Height      int    `json:"height,omitempty"`
	DownloadURL string `json:"download_url"`
	SGID        string `json:"sgid"`
}

var attachmentsCmd = &cobra.Command{
	Use:   "attachments",
	Short: "Manage card attachments",
	Long:  "Commands for viewing and downloading card attachments.",
}

// Attachment show flags
var attachmentsShowIncludeComments bool

var attachmentsShowCmd = &cobra.Command{
	Use:   "show CARD_NUMBER",
	Short: "List attachments on a card",
	Long: `Lists all attachments embedded in a card's description.

Use --include-comments to also include attachments from comments on the card.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := requireAuthAndAccount(); err != nil {
			exitWithError(err)
		}

		client := getClient()
		resp, err := client.Get("/cards/" + args[0] + ".json")
		if err != nil {
			exitWithError(err)
		}

		cardData, ok := resp.Data.(map[string]interface{})
		if !ok {
			exitWithError(errors.NewError("Invalid card response"))
		}

		descriptionHTML, _ := cardData["description_html"].(string)
		attachments := parseAttachments(descriptionHTML)

		if attachmentsShowIncludeComments {
			commentsResp, err := client.GetWithPagination("/cards/"+args[0]+"/comments.json", true)
			if err == nil {
				if comments, ok := commentsResp.Data.([]interface{}); ok {
					commentAttachments := extractCommentAttachments(comments)
					// Re-index and append
					for _, ca := range commentAttachments {
						ca.Index = len(attachments) + 1
						attachments = append(attachments, ca.Attachment)
					}
				}
			}
		}

		printSuccess(attachments)
	},
}

// Attachment download flags
var attachmentDownloadOutput string
var attachmentsDownloadIncludeComments bool

var attachmentsDownloadCmd = &cobra.Command{
	Use:   "download CARD_NUMBER [ATTACHMENT_INDEX]",
	Short: "Download attachments from a card",
	Long: `Downloads attachments from a card.

If ATTACHMENT_INDEX is provided, downloads only that attachment (1-based index).
If ATTACHMENT_INDEX is omitted, downloads all attachments.

Use --include-comments to also download attachments from comments on the card.

Use 'fizzy card attachments show CARD_NUMBER' to see available attachments and their indices.`,
	Args: cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		if err := requireAuthAndAccount(); err != nil {
			exitWithError(err)
		}

		cardNumber := args[0]

		client := getClient()
		resp, err := client.Get("/cards/" + cardNumber + ".json")
		if err != nil {
			exitWithError(err)
		}

		cardData, ok := resp.Data.(map[string]interface{})
		if !ok {
			exitWithError(errors.NewError("Invalid card response"))
		}

		descriptionHTML, _ := cardData["description_html"].(string)
		attachments := parseAttachments(descriptionHTML)

		if attachmentsDownloadIncludeComments {
			commentsResp, err := client.GetWithPagination("/cards/"+cardNumber+"/comments.json", true)
			if err == nil {
				if comments, ok := commentsResp.Data.([]interface{}); ok {
					commentAttachments := extractCommentAttachments(comments)
					for _, ca := range commentAttachments {
						ca.Index = len(attachments) + 1
						attachments = append(attachments, ca.Attachment)
					}
				}
			}
		}

		if len(attachments) == 0 {
			exitWithError(errors.NewNotFoundError("No attachments found on this card"))
		}

		// Determine which attachments to download
		var toDownload []Attachment
		if len(args) == 2 {
			// Download specific attachment
			attachmentIndex, err := strconv.Atoi(args[1])
			if err != nil {
				exitWithError(errors.NewInvalidArgsError("attachment index must be a number"))
			}
			if attachmentIndex < 1 || attachmentIndex > len(attachments) {
				exitWithError(errors.NewInvalidArgsError("attachment index must be between 1 and " + strconv.Itoa(len(attachments))))
			}
			toDownload = []Attachment{attachments[attachmentIndex-1]}
		} else {
			// Download all attachments
			toDownload = attachments
		}

		// Download the files
		results := make([]map[string]interface{}, 0, len(toDownload))
		for i, attachment := range toDownload {
			outputPath := buildOutputPath(attachmentDownloadOutput, attachment.Filename, i+1, len(toDownload))

			if err := client.DownloadFile(attachment.DownloadURL, outputPath); err != nil {
				exitWithError(err)
			}

			results = append(results, map[string]interface{}{
				"filename": attachment.Filename,
				"saved_to": outputPath,
				"filesize": attachment.Filesize,
			})
		}

		printSuccess(map[string]interface{}{
			"downloaded": len(results),
			"files":      results,
		})
	},
}

// parseAttachments extracts attachment information from description_html
func parseAttachments(html string) []Attachment {
	var attachments []Attachment

	// Match action-text-attachment elements with their inner content and any trailing
	// <figure> block. The figure may appear inside or outside the attachment tags depending
	// on how the server renders the HTML (e.g. when wrapped in <p> tags).
	attachmentRegex := regexp.MustCompile(`(?s)<action-text-attachment\s+([^>]+)>(.*?)</action-text-attachment>(?:\s*</p>)?\s*(<figure[^>]*>.*?</figure>)?`)
	matches := attachmentRegex.FindAllStringSubmatch(html, -1)

	for i, match := range matches {
		if len(match) < 3 {
			continue
		}

		attrs := match[1]
		content := match[2]
		if len(match) > 3 && match[3] != "" {
			content += match[3]
		}
		attachment := Attachment{
			Index: i + 1,
		}

		// Parse attributes
		attachment.SGID = extractAttr(attrs, "sgid")
		attachment.ContentType = extractAttr(attrs, "content-type")
		attachment.Filename = extractAttr(attrs, "filename")

		if filesize := extractAttr(attrs, "filesize"); filesize != "" {
			if size, err := strconv.ParseInt(filesize, 10, 64); err == nil {
				attachment.Filesize = size
			}
		}

		if width := extractAttr(attrs, "width"); width != "" {
			if w, err := strconv.Atoi(width); err == nil {
				attachment.Width = w
			}
		}

		if height := extractAttr(attrs, "height"); height != "" {
			if h, err := strconv.Atoi(height); err == nil {
				attachment.Height = h
			}
		}

		// Extract download URL from within this attachment's content
		downloadURLRegex := regexp.MustCompile(`href="([^"]+\?disposition=attachment)"`)
		if downloadMatch := downloadURLRegex.FindStringSubmatch(content); len(downloadMatch) > 1 {
			attachment.DownloadURL = downloadMatch[1]
		}

		// If no download URL with disposition found, try blob URL pattern within content
		if attachment.DownloadURL == "" {
			blobURLRegex := regexp.MustCompile(`href="(/[^"]+/rails/active_storage/blobs/redirect/[^"]+)"`)
			if blobMatch := blobURLRegex.FindStringSubmatch(content); len(blobMatch) > 1 {
				url := blobMatch[1]
				// Add disposition=attachment if not present
				if !regexp.MustCompile(`\?`).MatchString(url) {
					attachment.DownloadURL = url + "?disposition=attachment"
				} else {
					attachment.DownloadURL = url
				}
			}
		}

		attachments = append(attachments, attachment)
	}

	// Filter out non-downloadable entries (e.g. mentions) that have no filename or download URL
	filtered := attachments[:0]
	for _, a := range attachments {
		if a.Filename == "" && a.DownloadURL == "" {
			continue
		}
		filtered = append(filtered, a)
	}

	// Re-index after filtering
	for i := range filtered {
		filtered[i].Index = i + 1
	}

	return filtered
}

// extractAttr extracts an attribute value from an HTML attribute string
func extractAttr(attrs, name string) string {
	re := regexp.MustCompile(name + `="([^"]*)"`)
	match := re.FindStringSubmatch(attrs)
	if len(match) > 1 {
		return match[1]
	}
	return ""
}

// buildOutputPath determines the output filename for a download.
// For a single file, outputFlag is used as the exact filename.
// For multiple files, outputFlag is used as a prefix: prefix_1.ext, prefix_2.ext, etc.
// If outputFlag is empty, the original filename is used (sanitized).
func buildOutputPath(outputFlag, originalFilename string, index, total int) string {
	safeName := filepath.Base(originalFilename)
	if outputFlag == "" {
		return safeName
	}
	if total == 1 {
		return outputFlag
	}
	// Use as prefix: prefix_1.ext
	ext := filepath.Ext(safeName)
	prefix := strings.TrimSuffix(outputFlag, filepath.Ext(outputFlag))
	return fmt.Sprintf("%s_%d%s", prefix, index, ext)
}

func init() {
	cardCmd.AddCommand(attachmentsCmd)

	attachmentsShowCmd.Flags().BoolVar(&attachmentsShowIncludeComments, "include-comments", false, "Also include attachments from comments")
	attachmentsCmd.AddCommand(attachmentsShowCmd)

	attachmentsDownloadCmd.Flags().StringVarP(&attachmentDownloadOutput, "output", "o", "", "Output filename (single file) or prefix (multiple files, e.g. -o test produces test_1.png)")
	attachmentsDownloadCmd.Flags().BoolVar(&attachmentsDownloadIncludeComments, "include-comments", false, "Also include attachments from comments")
	attachmentsCmd.AddCommand(attachmentsDownloadCmd)
}
