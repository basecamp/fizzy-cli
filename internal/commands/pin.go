package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var pinCmd = &cobra.Command{
	Use:   "pin",
	Short: "Manage pins",
	Long:  "Commands for managing your pinned cards.",
}

var pinListCmd = &cobra.Command{
	Use:   "list",
	Short: "List pinned cards",
	Long:  "Lists your pinned cards (up to 100).",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		client := getClient()
		resp, err := client.Get("/my/pins.json")
		if err != nil {
			return err
		}

		// Build summary
		count := 0
		if arr, ok := resp.Data.([]any); ok {
			count = len(arr)
		}
		summary := fmt.Sprintf("%d pinned cards", count)

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("show", "fizzy card show <number>", "View card details"),
			breadcrumb("unpin", "fizzy card unpin <number>", "Unpin a card"),
			breadcrumb("pin", "fizzy card pin <number>", "Pin a card"),
		}

		printList(resp.Data, pinColumns, summary, breadcrumbs)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pinCmd)
	pinCmd.AddCommand(pinListCmd)
}
