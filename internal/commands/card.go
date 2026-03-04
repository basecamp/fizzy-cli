package commands

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/basecamp/fizzy-cli/internal/errors"
	"github.com/basecamp/fizzy-cli/internal/response"
	"github.com/spf13/cobra"
)

var cardCmd = &cobra.Command{
	Use:   "card",
	Short: "Manage cards",
	Long:  "Commands for managing Fizzy cards.",
}

// Card list flags
var cardListBoard string
var cardListColumn string
var cardListTag string
var cardListIndexedBy string
var cardListAssignee string
var cardListSearch string
var cardListSort string
var cardListCreator string
var cardListCloser string
var cardListUnassigned bool
var cardListCreated string
var cardListClosed string
var cardListPage int
var cardListAll bool

var cardListCmd = &cobra.Command{
	Use:   "list",
	Short: "List cards",
	Long:  "Lists cards with optional filters.",
	Run: func(cmd *cobra.Command, args []string) {
		if err := requireAuthAndAccount(); err != nil {
			exitWithError(err)
		}

		boardID := defaultBoard(cardListBoard)
		columnFilter := strings.TrimSpace(cardListColumn)
		indexedByFilter := strings.TrimSpace(cardListIndexedBy)
		effectiveIndexedBy := indexedByFilter

		client := getClient()
		path := "/cards.json"

		var params []string
		if boardID != "" {
			params = append(params, "board_ids[]="+boardID)
		}

		clientSideColumnFilter := ""
		clientSideTriage := false
		if columnFilter != "" {
			if pseudo, ok := parsePseudoColumnID(columnFilter); ok {
				switch pseudo.Kind {
				case "not_now":
					if effectiveIndexedBy != "" && effectiveIndexedBy != "not_now" {
						exitWithError(errors.NewInvalidArgsError("cannot combine --indexed-by with --column maybe"))
					}
					effectiveIndexedBy = "not_now"
				case "closed":
					if effectiveIndexedBy != "" && effectiveIndexedBy != "closed" {
						exitWithError(errors.NewInvalidArgsError("cannot combine --indexed-by with --column done"))
					}
					effectiveIndexedBy = "closed"
				case "triage":
					if effectiveIndexedBy != "" {
						exitWithError(errors.NewInvalidArgsError("cannot combine --indexed-by with --column not-yet"))
					}
					clientSideTriage = true
				default:
					clientSideColumnFilter = columnFilter
				}
			} else {
				if effectiveIndexedBy != "" {
					exitWithError(errors.NewInvalidArgsError("cannot combine --indexed-by with --column"))
				}
				clientSideColumnFilter = columnFilter
			}
		}

		if effectiveIndexedBy != "" {
			params = append(params, "indexed_by="+effectiveIndexedBy)
		}

		if cardListTag != "" {
			params = append(params, "tag_ids[]="+cardListTag)
		}
		if cardListAssignee != "" {
			params = append(params, "assignee_ids[]="+cardListAssignee)
		}
		if cardListSearch != "" {
			for _, term := range strings.Fields(cardListSearch) {
				params = append(params, "terms[]="+term)
			}
		}
		if cardListSort != "" {
			params = append(params, "sorted_by="+cardListSort)
		}
		if cardListCreator != "" {
			params = append(params, "creator_ids[]="+cardListCreator)
		}
		if cardListCloser != "" {
			params = append(params, "closer_ids[]="+cardListCloser)
		}
		if cardListUnassigned {
			params = append(params, "assignment_status=unassigned")
		}
		if cardListCreated != "" {
			params = append(params, "creation="+cardListCreated)
		}
		if cardListClosed != "" {
			params = append(params, "closure="+cardListClosed)
		}
		if cardListPage > 0 {
			params = append(params, "page="+strconv.Itoa(cardListPage))
		}
		if len(params) > 0 {
			path += "?" + strings.Join(params, "&")
		}

		if (clientSideTriage || clientSideColumnFilter != "") && !cardListAll && cardListPage == 0 {
			exitWithError(errors.NewInvalidArgsError("Filtering by column requires --all (or --page) because it is applied client-side"))
		}

		resp, err := client.GetWithPagination(path, cardListAll)
		if err != nil {
			exitWithError(err)
		}

		if clientSideTriage || clientSideColumnFilter != "" {
			arr, ok := resp.Data.([]interface{})
			if !ok {
				exitWithError(errors.NewError("Unexpected cards list response"))
			}

			filtered := make([]interface{}, 0, len(arr))
			for _, item := range arr {
				card, ok := item.(map[string]interface{})
				if !ok {
					continue
				}

				columnID := ""
				if v, ok := card["column_id"].(string); ok {
					columnID = v
				}
				if columnID == "" {
					if col, ok := card["column"].(map[string]interface{}); ok {
						if id, ok := col["id"].(string); ok {
							columnID = id
						}
					}
				}

				if clientSideTriage {
					if columnID == "" {
						filtered = append(filtered, item)
					}
					continue
				}

				if clientSideColumnFilter != "" && columnID == clientSideColumnFilter {
					filtered = append(filtered, item)
				}
			}

			resp.Data = filtered
		}

		// Build summary
		count := 0
		if arr, ok := resp.Data.([]interface{}); ok {
			count = len(arr)
		}
		summary := fmt.Sprintf("%d cards", count)
		if cardListAll {
			summary += " (all)"
		} else if cardListPage > 0 {
			summary += fmt.Sprintf(" (page %d)", cardListPage)
		}

		// Build breadcrumbs
		breadcrumbs := []response.Breadcrumb{
			breadcrumb("show", "fizzy card show <number>", "View card details"),
			breadcrumb("create", "fizzy card create --board <id> --title \"title\"", "Create new card"),
			breadcrumb("search", "fizzy search \"query\"", "Search cards"),
		}

		hasNext := resp.LinkNext != ""
		if hasNext {
			nextPage := cardListPage + 1
			if nextPage == 0 {
				nextPage = 2
			}
			breadcrumbs = append(breadcrumbs, breadcrumb("next", fmt.Sprintf("fizzy card list --page %d", nextPage), "Next page"))
		}

		printSuccessWithPaginationAndBreadcrumbs(resp.Data, hasNext, resp.LinkNext, summary, breadcrumbs)
	},
}

var cardShowCmd = &cobra.Command{
	Use:   "show CARD_NUMBER",
	Short: "Show a card",
	Long:  "Shows details of a specific card.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := requireAuthAndAccount(); err != nil {
			exitWithError(err)
		}

		client := getClient()
		resp, err := client.Get("/cards/" + args[0] + ".json")
		if err != nil {
			exitWithError(err)
		}

		cardNumber := args[0]

		// Build summary
		summary := fmt.Sprintf("Card #%s", cardNumber)
		if card, ok := resp.Data.(map[string]interface{}); ok {
			if title, ok := card["title"].(string); ok {
				summary = fmt.Sprintf("Card #%s: %s", cardNumber, title)
			}
		}

		// Build breadcrumbs
		breadcrumbs := []response.Breadcrumb{
			breadcrumb("comment", fmt.Sprintf("fizzy comment create --card %s --body \"text\"", cardNumber), "Add comment"),
			breadcrumb("triage", fmt.Sprintf("fizzy card column %s --column <column_id>", cardNumber), "Move to column"),
			breadcrumb("close", fmt.Sprintf("fizzy card close %s", cardNumber), "Close card"),
			breadcrumb("assign", fmt.Sprintf("fizzy card assign %s --user <user_id>", cardNumber), "Assign user"),
		}

		printSuccessWithBreadcrumbs(resp.Data, summary, breadcrumbs)
	},
}

// Card create flags
var cardCreateBoard string
var cardCreateTitle string
var cardCreateDescription string
var cardCreateDescriptionFile string
var cardCreateTagIDs string
var cardCreateImage string
var cardCreateCreatedAt string

var cardCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a card",
	Long:  "Creates a new card in a board.",
	Run: func(cmd *cobra.Command, args []string) {
		if err := requireAuthAndAccount(); err != nil {
			exitWithError(err)
		}

		boardID, err := requireBoard(cardCreateBoard)
		if err != nil {
			exitWithError(err)
		}
		if cardCreateTitle == "" {
			exitWithError(newRequiredFlagError("title"))
		}

		cardParams := map[string]interface{}{
			"title": cardCreateTitle,
		}

		// Handle description
		if cardCreateDescriptionFile != "" {
			descContent, descErr := os.ReadFile(cardCreateDescriptionFile)
			if descErr != nil {
				exitWithError(descErr)
			}
			cardParams["description"] = markdownToHTML(string(descContent))
		} else if cardCreateDescription != "" {
			cardParams["description"] = markdownToHTML(cardCreateDescription)
		}

		if cardCreateTagIDs != "" {
			cardParams["tag_ids"] = cardCreateTagIDs
		}
		if cardCreateImage != "" {
			cardParams["image"] = cardCreateImage
		}
		if cardCreateCreatedAt != "" {
			cardParams["created_at"] = cardCreateCreatedAt
		}

		body := map[string]interface{}{
			"board_id": boardID,
			"card":     cardParams,
		}

		client := getClient()
		resp, err := client.Post("/cards.json", body)
		if err != nil {
			exitWithError(err)
		}

		// Create returns location header - follow it to get the created resource
		if resp.Location != "" {
			followResp, err := client.FollowLocation(resp.Location)
			if err == nil && followResp != nil {
				// Extract card number from response
				cardNumber := ""
				if card, ok := followResp.Data.(map[string]interface{}); ok {
					if num, ok := card["number"].(float64); ok {
						cardNumber = fmt.Sprintf("%d", int(num))
					}
				}

				// Build breadcrumbs
				var breadcrumbs []response.Breadcrumb
				if cardNumber != "" {
					breadcrumbs = []response.Breadcrumb{
						breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card details"),
						breadcrumb("triage", fmt.Sprintf("fizzy card column %s --column <column_id>", cardNumber), "Move to column"),
						breadcrumb("comment", fmt.Sprintf("fizzy comment create --card %s --body \"text\"", cardNumber), "Add comment"),
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
			printSuccessWithLocation(resp.Location)
			return
		}

		printSuccess(resp.Data)
	},
}

// Card update flags
var cardUpdateTitle string
var cardUpdateDescription string
var cardUpdateDescriptionFile string
var cardUpdateImage string
var cardUpdateCreatedAt string

var cardUpdateCmd = &cobra.Command{
	Use:   "update CARD_NUMBER",
	Short: "Update a card",
	Long:  "Updates an existing card.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := requireAuthAndAccount(); err != nil {
			exitWithError(err)
		}

		cardParams := make(map[string]interface{})

		if cardUpdateTitle != "" {
			cardParams["title"] = cardUpdateTitle
		}
		if cardUpdateDescriptionFile != "" {
			content, err := os.ReadFile(cardUpdateDescriptionFile)
			if err != nil {
				exitWithError(err)
			}
			cardParams["description"] = markdownToHTML(string(content))
		} else if cardUpdateDescription != "" {
			cardParams["description"] = markdownToHTML(cardUpdateDescription)
		}
		if cardUpdateImage != "" {
			cardParams["image"] = cardUpdateImage
		}
		if cardUpdateCreatedAt != "" {
			cardParams["created_at"] = cardUpdateCreatedAt
		}

		body := map[string]interface{}{
			"card": cardParams,
		}

		client := getClient()
		resp, err := client.Patch("/cards/"+args[0]+".json", body)
		if err != nil {
			exitWithError(err)
		}

		cardNumber := args[0]

		// Build breadcrumbs
		breadcrumbs := []response.Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card details"),
			breadcrumb("triage", fmt.Sprintf("fizzy card column %s --column <column_id>", cardNumber), "Move to column"),
			breadcrumb("comment", fmt.Sprintf("fizzy comment create --card %s --body \"text\"", cardNumber), "Add comment"),
		}

		printSuccessWithBreadcrumbs(resp.Data, "", breadcrumbs)
	},
}

var cardDeleteCmd = &cobra.Command{
	Use:   "delete CARD_NUMBER",
	Short: "Delete a card",
	Long:  "Deletes a card.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := requireAuthAndAccount(); err != nil {
			exitWithError(err)
		}

		client := getClient()
		_, err := client.Delete("/cards/" + args[0] + ".json")
		if err != nil {
			exitWithError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []response.Breadcrumb{
			breadcrumb("cards", "fizzy card list", "List cards"),
			breadcrumb("create", "fizzy card create --board <id> --title \"title\"", "Create new card"),
		}

		printSuccessWithBreadcrumbs(map[string]interface{}{
			"deleted": true,
		}, "", breadcrumbs)
	},
}

var cardCloseCmd = &cobra.Command{
	Use:   "close CARD_NUMBER",
	Short: "Close a card",
	Long:  "Closes a card.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := requireAuthAndAccount(); err != nil {
			exitWithError(err)
		}

		cardNumber := args[0]

		client := getClient()
		resp, err := client.Post("/cards/"+cardNumber+"/closure.json", nil)
		if err != nil {
			exitWithError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []response.Breadcrumb{
			breadcrumb("reopen", fmt.Sprintf("fizzy card reopen %s", cardNumber), "Reopen card"),
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
		}

		data := resp.Data
		if data == nil {
			data = map[string]interface{}{}
		}
		printSuccessWithBreadcrumbs(data, "", breadcrumbs)
	},
}

var cardReopenCmd = &cobra.Command{
	Use:   "reopen CARD_NUMBER",
	Short: "Reopen a card",
	Long:  "Reopens a closed card.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := requireAuthAndAccount(); err != nil {
			exitWithError(err)
		}

		cardNumber := args[0]

		client := getClient()
		resp, err := client.Delete("/cards/" + cardNumber + "/closure.json")
		if err != nil {
			exitWithError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []response.Breadcrumb{
			breadcrumb("close", fmt.Sprintf("fizzy card close %s", cardNumber), "Close card"),
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
			breadcrumb("triage", fmt.Sprintf("fizzy card column %s --column <column_id>", cardNumber), "Move to column"),
		}

		data := resp.Data
		if data == nil {
			data = map[string]interface{}{}
		}
		printSuccessWithBreadcrumbs(data, "", breadcrumbs)
	},
}

var cardPostponeCmd = &cobra.Command{
	Use:   "postpone CARD_NUMBER",
	Short: "Postpone a card",
	Long:  "Moves a card to 'Not Now'.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := requireAuthAndAccount(); err != nil {
			exitWithError(err)
		}

		cardNumber := args[0]

		client := getClient()
		resp, err := client.Post("/cards/"+cardNumber+"/not_now.json", nil)
		if err != nil {
			exitWithError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []response.Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
			breadcrumb("triage", fmt.Sprintf("fizzy card column %s --column <column_id>", cardNumber), "Move to column"),
		}

		data := resp.Data
		if data == nil {
			data = map[string]interface{}{}
		}
		printSuccessWithBreadcrumbs(data, "", breadcrumbs)
	},
}

// Card move flags
var cardMoveBoard string

var cardMoveCmd = &cobra.Command{
	Use:   "move CARD_NUMBER",
	Short: "Move card to a different board",
	Long:  "Moves a card to a different board.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := requireAuthAndAccount(); err != nil {
			exitWithError(err)
		}

		if cardMoveBoard == "" {
			exitWithError(newRequiredFlagError("to"))
		}

		cardNumber := args[0]

		body := map[string]interface{}{
			"board_id": cardMoveBoard,
		}

		client := getClient()
		_, err := client.Patch("/cards/"+cardNumber+"/board.json", body)
		if err != nil {
			exitWithError(err)
		}

		// Fetch the updated card to show confirmation with title
		resp, err := client.Get("/cards/" + cardNumber + ".json")
		if err != nil {
			exitWithError(err)
		}

		// Build summary with card title if available
		summary := fmt.Sprintf("Card #%s moved to board %s", cardNumber, cardMoveBoard)
		if card, ok := resp.Data.(map[string]interface{}); ok {
			if title, ok := card["title"].(string); ok {
				summary = fmt.Sprintf("Card #%s \"%s\" moved to board %s", cardNumber, title, cardMoveBoard)
			}
		}

		// Build breadcrumbs
		breadcrumbs := []response.Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
			breadcrumb("triage", fmt.Sprintf("fizzy card column %s --column <column_id>", cardNumber), "Move to column"),
		}

		printSuccessWithBreadcrumbs(resp.Data, summary, breadcrumbs)
	},
}

// Card column flags
var cardColumnColumn string

var cardColumnCmd = &cobra.Command{
	Use:   "column CARD_NUMBER",
	Short: "Move card to column",
	Long:  "Moves a card to a specific column.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := requireAuthAndAccount(); err != nil {
			exitWithError(err)
		}

		if cardColumnColumn == "" {
			exitWithError(newRequiredFlagError("column"))
		}

		cardNumber := args[0]

		// Build breadcrumbs
		breadcrumbs := []response.Breadcrumb{
			breadcrumb("untriage", fmt.Sprintf("fizzy card untriage %s", cardNumber), "Send back to triage"),
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
			breadcrumb("close", fmt.Sprintf("fizzy card close %s", cardNumber), "Close card"),
		}

		client := getClient()
		if pseudo, ok := parsePseudoColumnID(cardColumnColumn); ok {
			switch pseudo.Kind {
			case "triage":
				resp, err := client.Delete("/cards/" + cardNumber + "/triage.json")
				if err != nil {
					exitWithError(err)
				}
				data := resp.Data
				if data == nil {
					data = map[string]interface{}{}
				}
				printSuccessWithBreadcrumbs(data, "", breadcrumbs)
				return
			case "not_now":
				resp, err := client.Post("/cards/"+cardNumber+"/not_now.json", nil)
				if err != nil {
					exitWithError(err)
				}
				data := resp.Data
				if data == nil {
					data = map[string]interface{}{}
				}
				printSuccessWithBreadcrumbs(data, "", breadcrumbs)
				return
			case "closed":
				resp, err := client.Post("/cards/"+cardNumber+"/closure.json", nil)
				if err != nil {
					exitWithError(err)
				}
				data := resp.Data
				if data == nil {
					data = map[string]interface{}{}
				}
				printSuccessWithBreadcrumbs(data, "", breadcrumbs)
				return
			}
		}

		body := map[string]interface{}{
			"column_id": cardColumnColumn,
		}

		resp, err := client.Post("/cards/"+cardNumber+"/triage.json", body)
		if err != nil {
			exitWithError(err)
		}

		data := resp.Data
		if data == nil {
			data = map[string]interface{}{}
		}
		printSuccessWithBreadcrumbs(data, "", breadcrumbs)
	},
}

var cardUntriageCmd = &cobra.Command{
	Use:   "untriage CARD_NUMBER",
	Short: "Send card back to triage",
	Long:  "Removes a card from its column and sends it back to triage.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := requireAuthAndAccount(); err != nil {
			exitWithError(err)
		}

		cardNumber := args[0]

		client := getClient()
		resp, err := client.Delete("/cards/" + cardNumber + "/triage.json")
		if err != nil {
			exitWithError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []response.Breadcrumb{
			breadcrumb("triage", fmt.Sprintf("fizzy card column %s --column <column_id>", cardNumber), "Move to column"),
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
		}

		data := resp.Data
		if data == nil {
			data = map[string]interface{}{
				"untriaged": true,
			}
		}
		printSuccessWithBreadcrumbs(data, "", breadcrumbs)
	},
}

// Card assign flags
var cardAssignUser string

var cardAssignCmd = &cobra.Command{
	Use:   "assign CARD_NUMBER",
	Short: "Toggle assignment on a card",
	Long:  "Toggles a user's assignment on a card.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := requireAuthAndAccount(); err != nil {
			exitWithError(err)
		}

		if cardAssignUser == "" {
			exitWithError(newRequiredFlagError("user"))
		}

		cardNumber := args[0]

		body := map[string]interface{}{
			"assignee_id": cardAssignUser,
		}

		client := getClient()
		resp, err := client.Post("/cards/"+cardNumber+"/assignments.json", body)
		if err != nil {
			exitWithError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []response.Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
			breadcrumb("people", "fizzy user list", "List users"),
		}

		data := resp.Data
		if data == nil {
			data = map[string]interface{}{}
		}
		printSuccessWithBreadcrumbs(data, "", breadcrumbs)
	},
}

var cardSelfAssignCmd = &cobra.Command{
	Use:   "self-assign CARD_NUMBER",
	Short: "Toggle self-assignment on a card",
	Long:  "Toggles the current user's assignment on a card.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := requireAuthAndAccount(); err != nil {
			exitWithError(err)
		}

		cardNumber := args[0]

		client := getClient()
		resp, err := client.Post("/cards/"+cardNumber+"/self_assignment.json", nil)
		if err != nil {
			exitWithError(err)
		}

		breadcrumbs := []response.Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
		}

		data := resp.Data
		if data == nil {
			data = map[string]interface{}{}
		}
		printSuccessWithBreadcrumbs(data, "", breadcrumbs)
	},
}

// Card tag flags
var cardTagTag string

var cardTagCmd = &cobra.Command{
	Use:   "tag CARD_NUMBER",
	Short: "Toggle tag on a card",
	Long:  "Toggles a tag on a card. Creates the tag if it doesn't exist.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := requireAuthAndAccount(); err != nil {
			exitWithError(err)
		}

		if cardTagTag == "" {
			exitWithError(newRequiredFlagError("tag"))
		}

		cardNumber := args[0]

		body := map[string]interface{}{
			"tag_title": cardTagTag,
		}

		client := getClient()
		resp, err := client.Post("/cards/"+cardNumber+"/taggings.json", body)
		if err != nil {
			exitWithError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []response.Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
			breadcrumb("tags", "fizzy tag list", "List tags"),
		}

		data := resp.Data
		if data == nil {
			data = map[string]interface{}{}
		}
		printSuccessWithBreadcrumbs(data, "", breadcrumbs)
	},
}

var cardWatchCmd = &cobra.Command{
	Use:   "watch CARD_NUMBER",
	Short: "Watch a card",
	Long:  "Subscribes to notifications for a card.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := requireAuthAndAccount(); err != nil {
			exitWithError(err)
		}

		cardNumber := args[0]

		client := getClient()
		resp, err := client.Post("/cards/"+cardNumber+"/watch.json", nil)
		if err != nil {
			exitWithError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []response.Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
			breadcrumb("notifications", "fizzy notification list", "View notifications"),
		}

		data := resp.Data
		if data == nil {
			data = map[string]interface{}{}
		}
		printSuccessWithBreadcrumbs(data, "", breadcrumbs)
	},
}

var cardUnwatchCmd = &cobra.Command{
	Use:   "unwatch CARD_NUMBER",
	Short: "Unwatch a card",
	Long:  "Unsubscribes from notifications for a card.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := requireAuthAndAccount(); err != nil {
			exitWithError(err)
		}

		cardNumber := args[0]

		client := getClient()
		resp, err := client.Delete("/cards/" + cardNumber + "/watch.json")
		if err != nil {
			exitWithError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []response.Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
			breadcrumb("notifications", "fizzy notification list", "View notifications"),
		}

		data := resp.Data
		if data == nil {
			data = map[string]interface{}{}
		}
		printSuccessWithBreadcrumbs(data, "", breadcrumbs)
	},
}

var cardImageRemoveCmd = &cobra.Command{
	Use:   "image-remove CARD_NUMBER",
	Short: "Remove card header image",
	Long:  "Removes the header image from a card.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := requireAuthAndAccount(); err != nil {
			exitWithError(err)
		}

		cardNumber := args[0]

		client := getClient()
		resp, err := client.Delete("/cards/" + cardNumber + "/image.json")
		if err != nil {
			exitWithError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []response.Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
			breadcrumb("update", fmt.Sprintf("fizzy card update %s", cardNumber), "Update card"),
		}

		data := resp.Data
		if data == nil {
			data = map[string]interface{}{}
		}
		printSuccessWithBreadcrumbs(data, "", breadcrumbs)
	},
}

var cardPinCmd = &cobra.Command{
	Use:   "pin CARD_NUMBER",
	Short: "Pin a card",
	Long:  "Pins a card for quick access.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := requireAuthAndAccount(); err != nil {
			exitWithError(err)
		}

		cardNumber := args[0]

		client := getClient()
		resp, err := client.Post("/cards/"+cardNumber+"/pin.json", nil)
		if err != nil {
			exitWithError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []response.Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
			breadcrumb("pins", "fizzy pin list", "List pinned cards"),
			breadcrumb("unpin", fmt.Sprintf("fizzy card unpin %s", cardNumber), "Unpin card"),
		}

		data := resp.Data
		if data == nil {
			data = map[string]interface{}{}
		}
		printSuccessWithBreadcrumbs(data, "", breadcrumbs)
	},
}

var cardUnpinCmd = &cobra.Command{
	Use:   "unpin CARD_NUMBER",
	Short: "Unpin a card",
	Long:  "Unpins a card, removing it from your pinned list.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := requireAuthAndAccount(); err != nil {
			exitWithError(err)
		}

		cardNumber := args[0]

		client := getClient()
		resp, err := client.Delete("/cards/" + cardNumber + "/pin.json")
		if err != nil {
			exitWithError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []response.Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
			breadcrumb("pins", "fizzy pin list", "List pinned cards"),
			breadcrumb("pin", fmt.Sprintf("fizzy card pin %s", cardNumber), "Pin card"),
		}

		data := resp.Data
		if data == nil {
			data = map[string]interface{}{}
		}
		printSuccessWithBreadcrumbs(data, "", breadcrumbs)
	},
}

var cardGoldenCmd = &cobra.Command{
	Use:   "golden CARD_NUMBER",
	Short: "Mark card as golden",
	Long:  "Marks a card as golden (starred/important).",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := requireAuthAndAccount(); err != nil {
			exitWithError(err)
		}

		cardNumber := args[0]

		client := getClient()
		resp, err := client.Post("/cards/"+cardNumber+"/goldness.json", nil)
		if err != nil {
			exitWithError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []response.Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
			breadcrumb("golden", "fizzy card list --indexed-by golden", "List golden cards"),
		}

		data := resp.Data
		if data == nil {
			data = map[string]interface{}{}
		}
		printSuccessWithBreadcrumbs(data, "", breadcrumbs)
	},
}

var cardUngoldenCmd = &cobra.Command{
	Use:   "ungolden CARD_NUMBER",
	Short: "Remove golden status from card",
	Long:  "Removes the golden (starred/important) status from a card.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := requireAuthAndAccount(); err != nil {
			exitWithError(err)
		}

		cardNumber := args[0]

		client := getClient()
		resp, err := client.Delete("/cards/" + cardNumber + "/goldness.json")
		if err != nil {
			exitWithError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []response.Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
			breadcrumb("golden", "fizzy card list --indexed-by golden", "List golden cards"),
		}

		data := resp.Data
		if data == nil {
			data = map[string]interface{}{}
		}
		printSuccessWithBreadcrumbs(data, "", breadcrumbs)
	},
}

func init() {
	rootCmd.AddCommand(cardCmd)

	// List
	cardListCmd.Flags().StringVar(&cardListBoard, "board", "", "Filter by board ID")
	cardListCmd.Flags().StringVar(&cardListColumn, "column", "", "Filter by column ID or pseudo column (not-yet, maybe, done)")
	cardListCmd.Flags().StringVar(&cardListTag, "tag", "", "Filter by tag ID")
	cardListCmd.Flags().StringVar(&cardListIndexedBy, "indexed-by", "", "Filter by lane/index (all, closed, not_now, stalled, postponing_soon, golden)")
	cardListCmd.Flags().StringVar(&cardListIndexedBy, "status", "", "Alias for --indexed-by")
	_ = cardListCmd.Flags().MarkDeprecated("status", "use --indexed-by")
	cardListCmd.Flags().StringVar(&cardListAssignee, "assignee", "", "Filter by assignee ID")
	cardListCmd.Flags().StringVar(&cardListSearch, "search", "", "Search terms (space-separated for multiple)")
	cardListCmd.Flags().StringVar(&cardListSort, "sort", "", "Sort order: newest, oldest, or latest (default)")
	cardListCmd.Flags().StringVar(&cardListCreator, "creator", "", "Filter by creator user ID")
	cardListCmd.Flags().StringVar(&cardListCloser, "closer", "", "Filter by closer user ID")
	cardListCmd.Flags().BoolVar(&cardListUnassigned, "unassigned", false, "Only show unassigned cards")
	cardListCmd.Flags().StringVar(&cardListCreated, "created", "", "Filter by creation time (today, yesterday, thisweek, lastweek, thismonth, lastmonth)")
	cardListCmd.Flags().StringVar(&cardListClosed, "closed", "", "Filter by closure time (today, yesterday, thisweek, lastweek, thismonth, lastmonth)")
	cardListCmd.Flags().IntVar(&cardListPage, "page", 0, "Page number")
	cardListCmd.Flags().BoolVar(&cardListAll, "all", false, "Fetch all pages")
	cardCmd.AddCommand(cardListCmd)

	// Show
	cardCmd.AddCommand(cardShowCmd)

	// Create
	cardCreateCmd.Flags().StringVar(&cardCreateBoard, "board", "", "Board ID (required)")
	cardCreateCmd.Flags().StringVar(&cardCreateTitle, "title", "", "Card title (required)")
	cardCreateCmd.Flags().StringVar(&cardCreateDescription, "description", "", "Card description (HTML)")
	cardCreateCmd.Flags().StringVar(&cardCreateDescriptionFile, "description_file", "", "Read description from file")
	cardCreateCmd.Flags().StringVar(&cardCreateTagIDs, "tag-ids", "", "Comma-separated tag IDs")
	cardCreateCmd.Flags().StringVar(&cardCreateImage, "image", "", "Header image signed ID")
	cardCreateCmd.Flags().StringVar(&cardCreateCreatedAt, "created-at", "", "Custom created_at timestamp")
	cardCmd.AddCommand(cardCreateCmd)

	// Update
	cardUpdateCmd.Flags().StringVar(&cardUpdateTitle, "title", "", "Card title")
	cardUpdateCmd.Flags().StringVar(&cardUpdateDescription, "description", "", "Card description (HTML)")
	cardUpdateCmd.Flags().StringVar(&cardUpdateDescriptionFile, "description_file", "", "Read description from file")
	cardUpdateCmd.Flags().StringVar(&cardUpdateImage, "image", "", "Header image signed ID")
	cardUpdateCmd.Flags().StringVar(&cardUpdateCreatedAt, "created-at", "", "Custom created_at timestamp")
	cardCmd.AddCommand(cardUpdateCmd)

	// Delete
	cardCmd.AddCommand(cardDeleteCmd)

	// Actions
	cardCmd.AddCommand(cardCloseCmd)
	cardCmd.AddCommand(cardReopenCmd)
	cardCmd.AddCommand(cardPostponeCmd)

	// Move to different board
	cardMoveCmd.Flags().StringVarP(&cardMoveBoard, "to", "t", "", "Target board ID (required)")
	cardCmd.AddCommand(cardMoveCmd)

	// Column
	cardColumnCmd.Flags().StringVar(&cardColumnColumn, "column", "", "Column ID (required)")
	cardCmd.AddCommand(cardColumnCmd)

	// Untriage
	cardCmd.AddCommand(cardUntriageCmd)

	// Assign
	cardAssignCmd.Flags().StringVar(&cardAssignUser, "user", "", "User ID (required)")
	cardCmd.AddCommand(cardAssignCmd)

	// Self-assign
	cardCmd.AddCommand(cardSelfAssignCmd)

	// Tag
	cardTagCmd.Flags().StringVar(&cardTagTag, "tag", "", "Tag name (required)")
	cardCmd.AddCommand(cardTagCmd)

	// Watch/Unwatch
	cardCmd.AddCommand(cardWatchCmd)
	cardCmd.AddCommand(cardUnwatchCmd)

	// Image removal
	cardCmd.AddCommand(cardImageRemoveCmd)

	// Golden
	cardCmd.AddCommand(cardGoldenCmd)
	cardCmd.AddCommand(cardUngoldenCmd)

	// Pin/Unpin
	cardCmd.AddCommand(cardPinCmd)
	cardCmd.AddCommand(cardUnpinCmd)
}
