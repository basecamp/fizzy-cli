package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var activityCmd = &cobra.Command{
	Use:   "activity",
	Short: "Manage activities",
	Long:  "Commands for listing Fizzy activities.",
}

var activityListBoard string
var activityListCreator string
var activityListPage int
var activityListAll bool

var activityListCmd = &cobra.Command{
	Use:   "list",
	Short: "List activities",
	Long:  "Lists activities with optional board and creator filters.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}
		if err := checkLimitAll(activityListAll); err != nil {
			return err
		}

		ac := getSDK()
		path := "/activities.json"

		var params []string
		if activityListBoard != "" {
			params = append(params, "board_ids[]="+activityListBoard)
		}
		if activityListCreator != "" {
			params = append(params, "creator_ids[]="+activityListCreator)
		}
		if activityListPage > 0 {
			params = append(params, "page="+strconv.Itoa(activityListPage))
		}
		if len(params) > 0 {
			path += "?" + strings.Join(params, "&")
		}

		var items any
		var linkNext string

		if activityListAll {
			pages, err := ac.GetAll(cmd.Context(), path)
			if err != nil {
				return convertSDKError(err)
			}
			items = jsonAnySlice(pages)
		} else {
			data, resp, err := ac.Cards().ListActivities(cmd.Context(), path)
			if err != nil {
				return convertSDKError(err)
			}
			items = normalizeAny(data)
			linkNext = parseSDKLinkNext(resp)
		}

		count := dataCount(items)
		summary := fmt.Sprintf("%d activities", count)
		if activityListAll {
			summary += " (all)"
		} else if activityListPage > 0 {
			summary += fmt.Sprintf(" (page %d)", activityListPage)
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("cards", "fizzy card show <number>", "View related card"),
			breadcrumb("board", "fizzy board show <id>", "View related board"),
		}
		if activityListBoard != "" {
			breadcrumbs = append(breadcrumbs, breadcrumb("board", fmt.Sprintf("fizzy board show %s", activityListBoard), "View board"))
		}

		hasNext := linkNext != ""
		if hasNext {
			nextPage := activityListPage + 1
			if activityListPage == 0 {
				nextPage = 2
			}
			breadcrumbs = append(breadcrumbs, breadcrumb("next", fmt.Sprintf("fizzy activity list --page %d", nextPage), "Next page"))
		}

		printListPaginated(items, activityColumns, hasNext, linkNext, activityListAll, summary, breadcrumbs)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(activityCmd)

	activityListCmd.Flags().StringVar(&activityListBoard, "board", "", "Filter by board ID")
	activityListCmd.Flags().StringVar(&activityListCreator, "creator", "", "Filter by creator user ID")
	activityListCmd.Flags().IntVar(&activityListPage, "page", 0, "Page number")
	activityListCmd.Flags().BoolVar(&activityListAll, "all", false, "Fetch all pages")
	activityCmd.AddCommand(activityListCmd)
}
