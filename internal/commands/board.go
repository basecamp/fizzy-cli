package commands

import (
	"fmt"
	"strconv"

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

		client := getClient()
		path := "/boards.json"
		if boardListPage > 0 {
			path += "?page=" + strconv.Itoa(boardListPage)
		}

		resp, err := client.GetWithPagination(path, boardListAll)
		if err != nil {
			return err
		}

		// Build summary
		count := 0
		if arr, ok := resp.Data.([]any); ok {
			count = len(arr)
		}
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

		hasNext := resp.LinkNext != ""
		if hasNext {
			nextPage := boardListPage + 1
			if nextPage == 0 {
				nextPage = 2
			}
			breadcrumbs = append(breadcrumbs, breadcrumb("next", fmt.Sprintf("fizzy board list --page %d", nextPage), "Next page"))
		}

		printSuccessWithPaginationAndBreadcrumbs(resp.Data, hasNext, resp.LinkNext, summary, breadcrumbs)
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

		client := getClient()
		resp, err := client.Get("/boards/" + boardID + ".json")
		if err != nil {
			return err
		}

		// Build summary
		summary := "Board"
		if board, ok := resp.Data.(map[string]any); ok {
			if name, ok := board["name"].(string); ok {
				summary = fmt.Sprintf("Board: %s", name)
			}
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("cards", fmt.Sprintf("fizzy card list --board %s", boardID), "List cards"),
			breadcrumb("columns", fmt.Sprintf("fizzy column list --board %s", boardID), "List columns"),
			breadcrumb("create-card", fmt.Sprintf("fizzy card create --board %s --title \"title\"", boardID), "Create card"),
		}

		printSuccessWithBreadcrumbs(resp.Data, summary, breadcrumbs)
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

		boardParams := map[string]any{
			"name": boardCreateName,
		}

		if boardCreateAllAccess != "" {
			boardParams["all_access"] = boardCreateAllAccess == "true"
		}
		if boardCreateAutoPostponePeriod > 0 {
			boardParams["auto_postpone_period"] = boardCreateAutoPostponePeriod
		}

		body := map[string]any{
			"board": boardParams,
		}

		client := getClient()
		resp, err := client.Post("/boards.json", body)
		if err != nil {
			return err
		}

		// Create returns location header - follow it to get the created resource
		if resp.Location != "" {
			followResp, err := client.FollowLocation(resp.Location)
			if err == nil && followResp != nil {
				// Extract board ID from response
				boardID := ""
				if board, ok := followResp.Data.(map[string]any); ok {
					if id, ok := board["id"].(string); ok {
						boardID = id
					}
				}

				// Build breadcrumbs
				var breadcrumbs []Breadcrumb
				if boardID != "" {
					breadcrumbs = []Breadcrumb{
						breadcrumb("show", fmt.Sprintf("fizzy board show %s", boardID), "View board details"),
						breadcrumb("cards", fmt.Sprintf("fizzy card list --board %s", boardID), "List cards"),
						breadcrumb("columns", fmt.Sprintf("fizzy column list --board %s", boardID), "List columns"),
					}
				}

				printSuccessWithLocationAndBreadcrumbs(followResp.Data, resp.Location, breadcrumbs)
				return nil
			}
			// If follow fails, just return success with location
			printSuccessWithLocation(resp.Location)
			return nil
		}

		printSuccess(resp.Data)
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

		boardParams := make(map[string]any)

		if boardUpdateName != "" {
			boardParams["name"] = boardUpdateName
		}
		if boardUpdateAllAccess != "" {
			boardParams["all_access"] = boardUpdateAllAccess == "true"
		}
		if boardUpdateAutoPostponePeriod > 0 {
			boardParams["auto_postpone_period"] = boardUpdateAutoPostponePeriod
		}

		body := map[string]any{
			"board": boardParams,
		}

		client := getClient()
		resp, err := client.Patch("/boards/"+boardID+".json", body)
		if err != nil {
			return err
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy board show %s", boardID), "View board"),
			breadcrumb("cards", fmt.Sprintf("fizzy card list --board %s", boardID), "List cards"),
		}

		printSuccessWithBreadcrumbs(resp.Data, "", breadcrumbs)
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

		client := getClient()
		_, err := client.Delete("/boards/" + args[0] + ".json")
		if err != nil {
			return err
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("boards", "fizzy board list", "List boards"),
			breadcrumb("create", "fizzy board create --name \"name\"", "Create new board"),
		}

		printSuccessWithBreadcrumbs(map[string]any{
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
