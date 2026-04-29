package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search QUERY",
	Short: "Search cards",
	Long: `Searches cards using the dedicated full-text search endpoint.

The query is sent as a single string. If the query exactly matches a card ID,
that card is returned directly.

To filter cards by structured criteria (board, tag, assignee, status, etc.),
use 'fizzy card list' with --search and the relevant filter flags.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		query := strings.Join(args, " ")

		ac := getSDK()
		raw, _, err := ac.Search().Search(cmd.Context(), &query)
		if err != nil {
			return convertSDKError(err)
		}

		items := normalizeAny(raw)
		count := dataCount(items)
		summary := fmt.Sprintf("%d results for \"%s\"", count, query)

		breadcrumbs := []Breadcrumb{
			breadcrumb("show", "fizzy card show <number>", "View card details"),
			breadcrumb("filter", fmt.Sprintf("fizzy card list --search \"%s\" --board <id>", query), "Filter cards by criteria"),
		}

		printList(items, searchColumns, summary, breadcrumbs)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
}
