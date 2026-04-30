package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// Search flags
var searchBoard string
var searchTag string
var searchAssignee string
var searchIndexedBy string
var searchSort string
var searchPage int
var searchAll bool

var searchCmd = &cobra.Command{
	Use:   "search QUERY",
	Short: "Search cards",
	Long:  "Searches cards by text. Multiple words are treated as separate terms (AND).",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}
		if err := checkLimitAll(searchAll); err != nil {
			return err
		}

		query := strings.Join(args, " ")

		ac := getSDK()
		path := "/cards.json"

		var params []string

		// Add search terms
		for term := range strings.FieldsSeq(query) {
			params = append(params, "terms[]="+term)
		}

		// Add optional filters (search is cross-board by default;
		// only scope to a board when explicitly requested via --board)
		if searchBoard != "" {
			params = append(params, "board_ids[]="+searchBoard)
		}
		if searchTag != "" {
			params = append(params, "tag_ids[]="+searchTag)
		}
		if searchAssignee != "" {
			params = append(params, "assignee_ids[]="+searchAssignee)
		}
		if searchIndexedBy != "" {
			params = append(params, "indexed_by="+searchIndexedBy)
		}
		if searchSort != "" {
			params = append(params, "sorted_by="+searchSort)
		}
		if searchPage > 0 {
			params = append(params, "page="+strconv.Itoa(searchPage))
		}

		if len(params) > 0 {
			path += "?" + strings.Join(params, "&")
		}

		var items any
		var linkNext string

		if searchAll {
			pages, err := ac.GetAll(cmd.Context(), path)
			if err != nil {
				return convertSDKError(err)
			}
			items = jsonAnySlice(pages)
		} else {
			data, resp, err := ac.Cards().List(cmd.Context(), path)
			if err != nil {
				return convertSDKError(err)
			}
			items = normalizeAny(data)
			linkNext = parseSDKLinkNext(resp)
		}

		// Build summary
		count := dataCount(items)
		summary := fmt.Sprintf("%d results for \"%s\"", count, query)
		if searchAll {
			summary = fmt.Sprintf("%d results for \"%s\" (all)", count, query)
		} else if searchPage > 0 {
			summary = fmt.Sprintf("%d results for \"%s\" (page %d)", count, query, searchPage)
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("show", "fizzy card show <number>", "View card details"),
			breadcrumb("narrow", fmt.Sprintf("fizzy search \"%s\" --board <id>", query), "Filter by board"),
		}

		hasNext := linkNext != ""
		printListPaginated(items, searchColumns, hasNext, linkNext, searchAll, summary, breadcrumbs)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)

	searchCmd.Flags().StringVar(&searchBoard, "board", "", "Filter by board ID")
	searchCmd.Flags().StringVar(&searchTag, "tag", "", "Filter by tag ID")
	searchCmd.Flags().StringVar(&searchAssignee, "assignee", "", "Filter by assignee ID")
	searchCmd.Flags().StringVar(&searchIndexedBy, "indexed-by", "", "Filter by status (all, closed, maybe, not_now, golden)")
	searchCmd.Flags().StringVar(&searchSort, "sort", "", "Sort order: newest, oldest, or latest (default)")
	searchCmd.Flags().IntVar(&searchPage, "page", 0, "Page number")
	searchCmd.Flags().BoolVar(&searchAll, "all", false, "Fetch all pages")
}
