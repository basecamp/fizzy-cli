package commands

import (
	"fmt"
	"strconv"

	"github.com/basecamp/fizzy-sdk/go/pkg/generated"
	"github.com/spf13/cobra"
)

var boardCmd = &cobra.Command{
	Use:   "board",
	Short: "Manage boards",
	Long:  "Commands for managing Fizzy boards.",
}

// Board list flags
var boardListPage int
var boardListAll bool

var boardListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all boards",
	Long:  "Lists all boards you have access to.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}
		if err := checkLimitAll(boardListAll); err != nil {
			return err
		}

		ac := getSDK()
		var items any
		var linkNext string

		path := "/boards.json"
		if boardListPage > 0 {
			path += "?page=" + strconv.Itoa(boardListPage)
		}

		if boardListAll {
			pages, err := ac.GetAll(cmd.Context(), path)
			if err != nil {
				return convertSDKError(err)
			}
			items = jsonAnySlice(pages)
		} else {
			data, resp, err := ac.Boards().List(cmd.Context(), path)
			if err != nil {
				return convertSDKError(err)
			}
			items = normalizeAny(data)
			linkNext = parseSDKLinkNext(resp)
		}

		// Build summary
		count := dataCount(items)
		summary := fmt.Sprintf("%d boards", count)
		if boardListAll {
			summary += " (all)"
		} else if boardListPage > 0 {
			summary += fmt.Sprintf(" (page %d)", boardListPage)
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("show", "fizzy board show <id>", "View board details"),
			breadcrumb("cards", "fizzy card list --board <id>", "List cards on board"),
			breadcrumb("columns", "fizzy column list --board <id>", "List board columns"),
		}

		hasNext := linkNext != ""
		if hasNext {
			nextPage := boardListPage + 1
			if boardListPage == 0 {
				nextPage = 2
			}
			breadcrumbs = append(breadcrumbs, breadcrumb("next", fmt.Sprintf("fizzy board list --page %d", nextPage), "Next page"))
		}

		printListPaginated(items, boardColumns, hasNext, linkNext, boardListAll, summary, breadcrumbs)
		return nil
	},
}

var boardShowCmd = &cobra.Command{
	Use:   "show BOARD_ID",
	Short: "Show a board",
	Long:  "Shows details of a specific board.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		boardID := args[0]

		data, _, err := getSDK().Boards().Get(cmd.Context(), boardID)
		if err != nil {
			return convertSDKError(err)
		}

		items := normalizeAny(data)

		summary := "Board"
		if board, ok := items.(map[string]any); ok {
			if name, ok := board["name"].(string); ok && name != "" {
				summary = fmt.Sprintf("Board: %s", name)
			}
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("cards", fmt.Sprintf("fizzy card list --board %s", boardID), "List cards"),
			breadcrumb("columns", fmt.Sprintf("fizzy column list --board %s", boardID), "List columns"),
			breadcrumb("create-card", fmt.Sprintf("fizzy card create --board %s --title \"title\"", boardID), "Create card"),
		}

		printDetail(items, summary, breadcrumbs)
		return nil
	},
}

// Board create flags
var boardCreateName string
var boardCreateAllAccess string
var boardCreateAutoPostponePeriod int

var boardCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a board",
	Long:  "Creates a new board.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		if boardCreateName == "" {
			return newRequiredFlagError("name")
		}

		req := &generated.CreateBoardRequest{
			Name: boardCreateName,
		}
		if boardCreateAllAccess != "" {
			req.AllAccess = boardCreateAllAccess == "true"
		}
		if boardCreateAutoPostponePeriod > 0 {
			req.AutoPostponePeriod = int32(boardCreateAutoPostponePeriod)
		}

		ac := getSDK()
		data, resp, err := ac.Boards().Create(cmd.Context(), req)
		if err != nil {
			return convertSDKError(err)
		}

		items := normalizeAny(data)
		boardID := ""
		if board, ok := items.(map[string]any); ok {
			if id, ok := board["id"]; ok {
				boardID = fmt.Sprintf("%v", id)
			}
		}

		var breadcrumbs []Breadcrumb
		if boardID != "" {
			breadcrumbs = []Breadcrumb{
				breadcrumb("show", fmt.Sprintf("fizzy board show %s", boardID), "View board details"),
				breadcrumb("cards", fmt.Sprintf("fizzy card list --board %s", boardID), "List cards"),
				breadcrumb("columns", fmt.Sprintf("fizzy column list --board %s", boardID), "List columns"),
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

// Board update flags
var boardUpdateName string
var boardUpdateAllAccess string
var boardUpdateAutoPostponePeriod int

var boardUpdateCmd = &cobra.Command{
	Use:   "update BOARD_ID",
	Short: "Update a board",
	Long:  "Updates an existing board.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		boardID := args[0]

		// When --all_access false is set, we must send `"all_access": false`
		// explicitly. The SDK's UpdateBoardRequest uses `omitempty` on the
		// AllAccess bool, which silently drops false values. Use raw Patch
		// when all_access is being set to false.
		ac := getSDK()
		var data any
		if boardUpdateAllAccess == "false" {
			body := map[string]any{"all_access": false}
			if boardUpdateName != "" {
				body["name"] = boardUpdateName
			}
			if boardUpdateAutoPostponePeriod > 0 {
				body["auto_postpone_period"] = boardUpdateAutoPostponePeriod
			}
			resp, patchErr := ac.Patch(cmd.Context(), "/boards/"+boardID+".json", body)
			if patchErr != nil {
				return convertSDKError(patchErr)
			}
			data = resp.Data
		} else {
			req := &generated.UpdateBoardRequest{}
			if boardUpdateName != "" {
				req.Name = boardUpdateName
			}
			if boardUpdateAllAccess == "true" {
				req.AllAccess = true
			}
			if boardUpdateAutoPostponePeriod > 0 {
				req.AutoPostponePeriod = int32(boardUpdateAutoPostponePeriod)
			}
			var updateErr error
			data, _, updateErr = ac.Boards().Update(cmd.Context(), boardID, req)
			if updateErr != nil {
				return convertSDKError(updateErr)
			}
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy board show %s", boardID), "View board"),
			breadcrumb("cards", fmt.Sprintf("fizzy card list --board %s", boardID), "List cards"),
		}

		printMutation(normalizeAny(data), "", breadcrumbs)
		return nil
	},
}

var boardDeleteCmd = &cobra.Command{
	Use:   "delete BOARD_ID",
	Short: "Delete a board",
	Long:  "Deletes a board.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		_, err := getSDK().Boards().Delete(cmd.Context(), args[0])
		if err != nil {
			return convertSDKError(err)
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("boards", "fizzy board list", "List boards"),
			breadcrumb("create", "fizzy board create --name \"name\"", "Create new board"),
		}

		printMutation(map[string]any{
			"deleted": true,
		}, "", breadcrumbs)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(boardCmd)

	// List
	boardListCmd.Flags().IntVar(&boardListPage, "page", 0, "Page number")
	boardListCmd.Flags().BoolVar(&boardListAll, "all", false, "Fetch all pages")
	boardCmd.AddCommand(boardListCmd)

	// Show
	boardCmd.AddCommand(boardShowCmd)

	// Create
	boardCreateCmd.Flags().StringVar(&boardCreateName, "name", "", "Board name (required)")
	boardCreateCmd.Flags().StringVar(&boardCreateAllAccess, "all_access", "", "Allow all team members access (true/false)")
	boardCreateCmd.Flags().IntVar(&boardCreateAutoPostponePeriod, "auto_postpone_period", 0, "Auto postpone period in days")
	boardCmd.AddCommand(boardCreateCmd)

	// Update
	boardUpdateCmd.Flags().StringVar(&boardUpdateName, "name", "", "Board name")
	boardUpdateCmd.Flags().StringVar(&boardUpdateAllAccess, "all_access", "", "Allow all team members access (true/false)")
	boardUpdateCmd.Flags().IntVar(&boardUpdateAutoPostponePeriod, "auto_postpone_period", 0, "Auto postpone period in days")
	boardCmd.AddCommand(boardUpdateCmd)

	// Delete
	boardCmd.AddCommand(boardDeleteCmd)
}
