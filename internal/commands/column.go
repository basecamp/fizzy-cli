package commands

import (
	"fmt"

	"github.com/basecamp/fizzy-cli/internal/errors"
	"github.com/spf13/cobra"
)

var columnCmd = &cobra.Command{
	Use:   "column",
	Short: "Manage columns",
	Long:  "Commands for managing board columns.",
}

// Column list flags
var columnListBoard string

var columnListCmd = &cobra.Command{
	Use:   "list",
	Short: "List columns for a board",
	Long:  "Lists all columns for a specific board.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		boardID, err := requireBoard(columnListBoard)
		if err != nil {
			return err
		}

		client := getClient()
		resp, err := client.Get("/boards/" + boardID + "/columns.json")
		if err != nil {
			return err
		}

		data, ok := resp.Data.([]any)
		if !ok {
			printSuccess(resp.Data)
			return nil
		}

		cols := make([]any, 0, len(data)+3)
		cols = append(cols, pseudoColumnObject(pseudoColumnNotNow), pseudoColumnObject(pseudoColumnMaybe))
		cols = append(cols, data...)
		cols = append(cols, pseudoColumnObject(pseudoColumnDone))

		// Build summary
		summary := fmt.Sprintf("%d columns", len(cols))

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("create", fmt.Sprintf("fizzy column create --board %s --name \"name\"", boardID), "Create column"),
			breadcrumb("cards", fmt.Sprintf("fizzy card list --board %s", boardID), "List cards"),
		}

		printSuccessWithBreadcrumbs(cols, summary, breadcrumbs)
		return nil
	},
}

// Column show flags
var columnShowBoard string

var columnShowCmd = &cobra.Command{
	Use:   "show COLUMN_ID",
	Short: "Show a column",
	Long:  "Shows details of a specific column.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		columnID := args[0]

		if pseudo, ok := parsePseudoColumnID(columnID); ok {
			// For pseudo columns, we don't have a board ID context
			breadcrumbs := []Breadcrumb{
				breadcrumb("columns", "fizzy column list --board <board_id>", "List columns"),
			}
			printSuccessWithBreadcrumbs(pseudoColumnObject(pseudo), "", breadcrumbs)
			return nil
		}

		boardID, err := requireBoard(columnShowBoard)
		if err != nil {
			return err
		}

		client := getClient()
		resp, err := client.Get("/boards/" + boardID + "/columns/" + columnID + ".json")
		if err != nil {
			return err
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("columns", fmt.Sprintf("fizzy column list --board %s", boardID), "List columns"),
			breadcrumb("update", fmt.Sprintf("fizzy column update %s --board %s", columnID, boardID), "Update column"),
		}

		printSuccessWithBreadcrumbs(resp.Data, "", breadcrumbs)
		return nil
	},
}

// Column create flags
var columnCreateBoard string
var columnCreateName string
var columnCreateColor string

var columnCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a column",
	Long:  "Creates a new column in a board.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		boardID, err := requireBoard(columnCreateBoard)
		if err != nil {
			return err
		}
		if columnCreateName == "" {
			return newRequiredFlagError("name")
		}

		columnParams := map[string]any{
			"name": columnCreateName,
		}
		if columnCreateColor != "" {
			columnParams["color"] = columnCreateColor
		}

		body := map[string]any{
			"column": columnParams,
		}

		client := getClient()
		resp, err := client.Post("/boards/"+boardID+"/columns.json", body)
		if err != nil {
			return err
		}

		// Create returns location header - follow it to get the created resource
		if resp.Location != "" {
			followResp, err := client.FollowLocation(resp.Location)
			if err == nil && followResp != nil {
				// Extract column ID from response
				columnID := ""
				if col, ok := followResp.Data.(map[string]any); ok {
					if id, ok := col["id"].(string); ok {
						columnID = id
					}
				}

				// Build breadcrumbs
				var breadcrumbs []Breadcrumb
				if columnID != "" {
					breadcrumbs = []Breadcrumb{
						breadcrumb("columns", fmt.Sprintf("fizzy column list --board %s", boardID), "List columns"),
						breadcrumb("show", fmt.Sprintf("fizzy column show %s --board %s", columnID, boardID), "View column"),
					}
				}

				printSuccessWithLocationAndBreadcrumbs(followResp.Data, resp.Location, breadcrumbs)
				return nil
			}
			printSuccessWithLocation(resp.Location)
			return nil
		}

		printSuccess(resp.Data)
		return nil
	},
}

// Column update flags
var columnUpdateBoard string
var columnUpdateName string
var columnUpdateColor string

var columnUpdateCmd = &cobra.Command{
	Use:   "update COLUMN_ID",
	Short: "Update a column",
	Long:  "Updates an existing column.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		if _, ok := parsePseudoColumnID(args[0]); ok {
			return errors.NewInvalidArgsError("cannot update pseudo columns (Not Yet, Maybe?, Done)")
		}

		boardID, err := requireBoard(columnUpdateBoard)
		if err != nil {
			return err
		}

		columnParams := make(map[string]any)
		if columnUpdateName != "" {
			columnParams["name"] = columnUpdateName
		}
		if columnUpdateColor != "" {
			columnParams["color"] = columnUpdateColor
		}

		body := map[string]any{
			"column": columnParams,
		}

		columnID := args[0]

		client := getClient()
		resp, err := client.Patch("/boards/"+boardID+"/columns/"+columnID+".json", body)
		if err != nil {
			return err
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("columns", fmt.Sprintf("fizzy column list --board %s", boardID), "List columns"),
			breadcrumb("show", fmt.Sprintf("fizzy column show %s --board %s", columnID, boardID), "View column"),
		}

		printSuccessWithBreadcrumbs(resp.Data, "", breadcrumbs)
		return nil
	},
}

// Column delete flags
var columnDeleteBoard string

var columnDeleteCmd = &cobra.Command{
	Use:   "delete COLUMN_ID",
	Short: "Delete a column",
	Long:  "Deletes a column from a board.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		if _, ok := parsePseudoColumnID(args[0]); ok {
			return errors.NewInvalidArgsError("cannot delete pseudo columns (Not Yet, Maybe?, Done)")
		}

		boardID, err := requireBoard(columnDeleteBoard)
		if err != nil {
			return err
		}

		client := getClient()
		_, err = client.Delete("/boards/" + boardID + "/columns/" + args[0] + ".json")
		if err != nil {
			return err
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("columns", fmt.Sprintf("fizzy column list --board %s", boardID), "List columns"),
			breadcrumb("create", fmt.Sprintf("fizzy column create --board %s --name \"name\"", boardID), "Create column"),
		}

		printSuccessWithBreadcrumbs(map[string]any{
			"deleted": true,
		}, "", breadcrumbs)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(columnCmd)

	// List
	columnListCmd.Flags().StringVar(&columnListBoard, "board", "", "Board ID (required)")
	columnCmd.AddCommand(columnListCmd)

	// Show
	columnShowCmd.Flags().StringVar(&columnShowBoard, "board", "", "Board ID (required)")
	columnCmd.AddCommand(columnShowCmd)

	// Create
	columnCreateCmd.Flags().StringVar(&columnCreateBoard, "board", "", "Board ID (required)")
	columnCreateCmd.Flags().StringVar(&columnCreateName, "name", "", "Column name (required)")
	columnCreateCmd.Flags().StringVar(&columnCreateColor, "color", "", "Column color")
	columnCmd.AddCommand(columnCreateCmd)

	// Update
	columnUpdateCmd.Flags().StringVar(&columnUpdateBoard, "board", "", "Board ID (required)")
	columnUpdateCmd.Flags().StringVar(&columnUpdateName, "name", "", "Column name")
	columnUpdateCmd.Flags().StringVar(&columnUpdateColor, "color", "", "Column color")
	columnCmd.AddCommand(columnUpdateCmd)

	// Delete
	columnDeleteCmd.Flags().StringVar(&columnDeleteBoard, "board", "", "Board ID (required)")
	columnCmd.AddCommand(columnDeleteCmd)
}
