package commands

import (
	"fmt"
	"os"

	"github.com/basecamp/fizzy-cli/internal/errors"
	"github.com/basecamp/fizzy-cli/internal/response"
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
	Run: func(cmd *cobra.Command, args []string) {
		if err := requireAuthAndAccount(); err != nil {
			exitWithError(err)
		}

		client := getClient()
		path := "/boards.json"
		if boardListPage > 0 {
			path += "?page=" + string(rune(boardListPage+'0'))
		}

		resp, err := client.GetWithPagination(path, boardListAll)
		if err != nil {
			exitWithError(err)
		}

		// Build summary
		count := 0
		if arr, ok := resp.Data.([]interface{}); ok {
			count = len(arr)
		}
		summary := fmt.Sprintf("%d boards", count)
		if boardListAll {
			summary += " (all)"
		} else if boardListPage > 0 {
			summary += fmt.Sprintf(" (page %d)", boardListPage)
		}

		// Build breadcrumbs
		breadcrumbs := []response.Breadcrumb{
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
	},
}

var boardShowCmd = &cobra.Command{
	Use:   "show BOARD_ID",
	Short: "Show a board",
	Long:  "Shows details of a specific board.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := requireAuthAndAccount(); err != nil {
			exitWithError(err)
		}

		boardID := args[0]

		client := getClient()
		resp, err := client.Get("/boards/" + boardID + ".json")
		if err != nil {
			exitWithError(err)
		}

		// Build summary
		summary := "Board"
		if board, ok := resp.Data.(map[string]interface{}); ok {
			if name, ok := board["name"].(string); ok {
				summary = fmt.Sprintf("Board: %s", name)
			}
		}

		// Build breadcrumbs
		breadcrumbs := []response.Breadcrumb{
			breadcrumb("cards", fmt.Sprintf("fizzy card list --board %s", boardID), "List cards"),
			breadcrumb("columns", fmt.Sprintf("fizzy column list --board %s", boardID), "List columns"),
			breadcrumb("create-card", fmt.Sprintf("fizzy card create --board %s --title \"title\"", boardID), "Create card"),
		}

		printSuccessWithBreadcrumbs(resp.Data, summary, breadcrumbs)
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
	Run: func(cmd *cobra.Command, args []string) {
		if err := requireAuthAndAccount(); err != nil {
			exitWithError(err)
		}

		if boardCreateName == "" {
			exitWithError(newRequiredFlagError("name"))
		}

		boardParams := map[string]interface{}{
			"name": boardCreateName,
		}

		if boardCreateAllAccess != "" {
			boardParams["all_access"] = boardCreateAllAccess == "true"
		}
		if boardCreateAutoPostponePeriod > 0 {
			boardParams["auto_postpone_period"] = boardCreateAutoPostponePeriod
		}

		body := map[string]interface{}{
			"board": boardParams,
		}

		client := getClient()
		resp, err := client.Post("/boards.json", body)
		if err != nil {
			exitWithError(err)
		}

		// Create returns location header - follow it to get the created resource
		if resp.Location != "" {
			followResp, err := client.FollowLocation(resp.Location)
			if err == nil && followResp != nil {
				// Extract board ID from response
				boardID := ""
				if board, ok := followResp.Data.(map[string]interface{}); ok {
					if id, ok := board["id"].(string); ok {
						boardID = id
					}
				}

				// Build breadcrumbs
				var breadcrumbs []response.Breadcrumb
				if boardID != "" {
					breadcrumbs = []response.Breadcrumb{
						breadcrumb("show", fmt.Sprintf("fizzy board show %s", boardID), "View board details"),
						breadcrumb("cards", fmt.Sprintf("fizzy card list --board %s", boardID), "List cards"),
						breadcrumb("columns", fmt.Sprintf("fizzy column list --board %s", boardID), "List columns"),
					}
				}

				respObj := response.SuccessWithBreadcrumbs(followResp.Data, "", breadcrumbs)
				respObj.Location = resp.Location
				if lastResult != nil {
					lastResult.Response = respObj
					lastResult.ExitCode = 0
					panic(testExitSignal{})
				}
				respObj.Print()
				os.Exit(0)
				return
			}
			// If follow fails, just return success with location
			printSuccessWithLocation(resp.Location)
			return
		}

		printSuccess(resp.Data)
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
	Run: func(cmd *cobra.Command, args []string) {
		if err := requireAuthAndAccount(); err != nil {
			exitWithError(err)
		}

		boardID := args[0]

		boardParams := make(map[string]interface{})

		if boardUpdateName != "" {
			boardParams["name"] = boardUpdateName
		}
		if boardUpdateAllAccess != "" {
			boardParams["all_access"] = boardUpdateAllAccess == "true"
		}
		if boardUpdateAutoPostponePeriod > 0 {
			boardParams["auto_postpone_period"] = boardUpdateAutoPostponePeriod
		}

		body := map[string]interface{}{
			"board": boardParams,
		}

		client := getClient()
		resp, err := client.Patch("/boards/"+boardID+".json", body)
		if err != nil {
			exitWithError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []response.Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy board show %s", boardID), "View board"),
			breadcrumb("cards", fmt.Sprintf("fizzy card list --board %s", boardID), "List cards"),
		}

		printSuccessWithBreadcrumbs(resp.Data, "", breadcrumbs)
	},
}

var boardDeleteCmd = &cobra.Command{
	Use:   "delete BOARD_ID",
	Short: "Delete a board",
	Long:  "Deletes a board.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := requireAuthAndAccount(); err != nil {
			exitWithError(err)
		}

		client := getClient()
		_, err := client.Delete("/boards/" + args[0] + ".json")
		if err != nil {
			exitWithError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []response.Breadcrumb{
			breadcrumb("boards", "fizzy board list", "List boards"),
			breadcrumb("create", "fizzy board create --name \"name\"", "Create new board"),
		}

		printSuccessWithBreadcrumbs(map[string]interface{}{
			"deleted": true,
		}, "", breadcrumbs)
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

// Helper function for required flag errors
func newRequiredFlagError(flag string) error {
	return errors.NewInvalidArgsError("required flag --" + flag + " not provided")
}
