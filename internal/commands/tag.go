package commands

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Manage tags",
	Long:  "Commands for viewing tags in your account.",
}

// Tag list flags
var tagListPage int
var tagListAll bool

var tagListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tags",
	Long:  "Lists all tags in your account.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}
		if err := checkLimitAll(tagListAll); err != nil {
			return err
		}

		ac := getSDK()
		var items any
		var linkNext string

		path := "/tags.json"
		if tagListPage > 0 {
			path += "?page=" + strconv.Itoa(tagListPage)
		}

		if tagListAll {
			pages, err := ac.GetAll(cmd.Context(), path)
			if err != nil {
				return convertSDKError(err)
			}
			items = jsonAnySlice(pages)
		} else {
			listPath := ""
			if tagListPage > 0 {
				listPath = path
			}
			data, resp, err := ac.Tags().List(cmd.Context(), listPath)
			if err != nil {
				return convertSDKError(err)
			}
			items = normalizeAny(data)
			linkNext = parseSDKLinkNext(resp)
		}

		// Build summary
		count := dataCount(items)
		summary := fmt.Sprintf("%d tags", count)
		if tagListAll {
			summary += " (all)"
		} else if tagListPage > 0 {
			summary += fmt.Sprintf(" (page %d)", tagListPage)
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("tag", "fizzy card tag <number> --tag <name>", "Tag a card"),
			breadcrumb("cards", "fizzy card list --tag <id>", "List cards with tag"),
		}

		hasNext := linkNext != ""
		if hasNext {
			nextPage := tagListPage + 1
			if tagListPage == 0 {
				nextPage = 2
			}
			breadcrumbs = append(breadcrumbs, breadcrumb("next", fmt.Sprintf("fizzy tag list --page %d", nextPage), "Next page"))
		}

		printListPaginated(items, tagColumns, hasNext, linkNext, tagListAll, summary, breadcrumbs)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(tagCmd)

	// List
	tagListCmd.Flags().IntVar(&tagListPage, "page", 0, "Page number")
	tagListCmd.Flags().BoolVar(&tagListAll, "all", false, "Fetch all pages")
	tagCmd.AddCommand(tagListCmd)
}
