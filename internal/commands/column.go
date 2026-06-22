package commands

import (
	"fmt"

	"github.com/basecamp/fizzy-cli/internal/errors"
	"github.com/basecamp/fizzy-sdk/go/pkg/generated"
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

		ac := getSDK()
		data, _, err := ac.Columns().List(cmd.Context(), boardID)
		if err != nil {
			return convertSDKError(err)
		}

		items := normalizeAny(data)

		dataSlice := toSliceAny(items)
		if dataSlice == nil {
			printDetail(items, "", nil)
			return nil
		}

		cols := make([]any, 0, len(dataSlice)+3) //nolint:gosec // len is non-negative; +3 cannot overflow
		cols = append(cols, pseudoColumnObject(pseudoColumnNotNow), pseudoColumnObject(pseudoColumnMaybe))
		cols = append(cols, dataSlice...)
		cols = append(cols, pseudoColumnObject(pseudoColumnDone))

		// Build summary
		summary := fmt.Sprintf("%d columns", len(cols))

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("create", fmt.Sprintf("fizzy column create --board %s --name \"name\"", boardID), "Create column"),
			breadcrumb("cards", fmt.Sprintf("fizzy card list --board %s", boardID), "List cards"),
		}

		printList(cols, columnColumns, summary, breadcrumbs)
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
			printDetail(pseudoColumnObject(pseudo), "", breadcrumbs)
			return nil
		}

		boardID, err := requireBoard(columnShowBoard)
		if err != nil {
			return err
		}

		data, _, err := getSDK().Columns().Get(cmd.Context(), boardID, columnID)
		if err != nil {
			return convertSDKError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("columns", fmt.Sprintf("fizzy column list --board %s", boardID), "List columns"),
			breadcrumb("update", fmt.Sprintf("fizzy column update %s --board %s", columnID, boardID), "Update column"),
		}

		printDetail(normalizeAny(data), "", breadcrumbs)
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

		ac := getSDK()
		req := &generated.CreateColumnRequest{Name: columnCreateName}
		if columnCreateColor != "" {
			req.Color = columnCreateColor
		}

		data, resp, err := ac.Columns().Create(cmd.Context(), boardID, req)
		if err != nil {
			return convertSDKError(err)
		}

		items := normalizeAny(data)
		if items == nil {
			items = map[string]any{}
		}

		// Extract column ID from response
		columnID := ""
		if col, ok := items.(map[string]any); ok {
			if id, ok := col["id"]; ok {
				columnID = fmt.Sprintf("%v", id)
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

		if location := resp.Headers.Get("Location"); location != "" {
			printMutationWithLocation(items, location, breadcrumbs)
		} else {
			printMutation(items, "", breadcrumbs)
		}
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

		columnID := args[0]

		req := &generated.UpdateColumnRequest{}
		if columnUpdateName != "" {
			req.Name = columnUpdateName
		}
		if columnUpdateColor != "" {
			req.Color = columnUpdateColor
		}

		data, _, err := getSDK().Columns().Update(cmd.Context(), boardID, columnID, req)
		if err != nil {
			return convertSDKError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("columns", fmt.Sprintf("fizzy column list --board %s", boardID), "List columns"),
			breadcrumb("show", fmt.Sprintf("fizzy column show %s --board %s", columnID, boardID), "View column"),
		}

		result := normalizeAny(data)
		if result == nil {
			result = map[string]any{}
		}
		printMutation(result, "", breadcrumbs)
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

		_, err = getSDK().Columns().Delete(cmd.Context(), boardID, args[0])
		if err != nil {
			return convertSDKError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("columns", fmt.Sprintf("fizzy column list --board %s", boardID), "List columns"),
			breadcrumb("create", fmt.Sprintf("fizzy column create --board %s --name \"name\"", boardID), "Create column"),
		}

		printMutation(map[string]any{
			"deleted": true,
		}, "", breadcrumbs)
		return nil
	},
}

var columnMoveLeftCmd = &cobra.Command{
	Use:   "move-left COLUMN_ID",
	Short: "Move a column left",
	Long:  "Moves a column one position to the left.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		if _, ok := parsePseudoColumnID(args[0]); ok {
			return errors.NewInvalidArgsError("cannot move pseudo columns (Not Yet, Maybe?, Done)")
		}

		columnID := args[0]

		_, err := getSDK().Columns().MoveLeft(cmd.Context(), columnID)
		if err != nil {
			return convertSDKError(err)
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("move-right", fmt.Sprintf("fizzy column move-right %s", columnID), "Move column right"),
			breadcrumb("columns", "fizzy column list --board <board_id>", "List columns"),
		}

		printMutation(map[string]any{}, "", breadcrumbs)
		return nil
	},
}

var columnMoveRightCmd = &cobra.Command{
	Use:   "move-right COLUMN_ID",
	Short: "Move a column right",
	Long:  "Moves a column one position to the right.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		if _, ok := parsePseudoColumnID(args[0]); ok {
			return errors.NewInvalidArgsError("cannot move pseudo columns (Not Yet, Maybe?, Done)")
		}

		columnID := args[0]

		_, err := getSDK().Columns().MoveRight(cmd.Context(), columnID)
		if err != nil {
			return convertSDKError(err)
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("move-left", fmt.Sprintf("fizzy column move-left %s", columnID), "Move column left"),
			breadcrumb("columns", "fizzy column list --board <board_id>", "List columns"),
		}

		printMutation(map[string]any{}, "", breadcrumbs)
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

	// Move
	columnCmd.AddCommand(columnMoveLeftCmd)
	columnCmd.AddCommand(columnMoveRightCmd)
}
