package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/basecamp/fizzy-cli/internal/errors"
	"github.com/basecamp/fizzy-sdk/go/pkg/generated"
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
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}
		if err := checkLimitAll(cardListAll); err != nil {
			return err
		}

		boardID := defaultBoard(cardListBoard)
		columnFilter := strings.TrimSpace(cardListColumn)
		indexedByFilter := strings.TrimSpace(cardListIndexedBy)
		effectiveIndexedBy := indexedByFilter

		ac := getSDK()
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
						return errors.NewInvalidArgsError("cannot combine --indexed-by with --column maybe")
					}
					effectiveIndexedBy = "not_now"
				case "closed":
					if effectiveIndexedBy != "" && effectiveIndexedBy != "closed" {
						return errors.NewInvalidArgsError("cannot combine --indexed-by with --column done")
					}
					effectiveIndexedBy = "closed"
				case "triage":
					if effectiveIndexedBy != "" {
						return errors.NewInvalidArgsError("cannot combine --indexed-by with --column not-yet")
					}
					clientSideTriage = true
				default:
					clientSideColumnFilter = columnFilter
				}
			} else {
				if effectiveIndexedBy != "" {
					return errors.NewInvalidArgsError("cannot combine --indexed-by with --column")
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
			for term := range strings.FieldsSeq(cardListSearch) {
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
			return errors.NewInvalidArgsError("Filtering by column requires --all (or --page) because it is applied client-side")
		}

		var items any
		var linkNext string

		if cardListAll {
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

		if clientSideTriage || clientSideColumnFilter != "" {
			arr := toSliceAny(items)
			if arr == nil {
				return errors.NewError("Unexpected cards list response")
			}

			filtered := make([]any, 0, len(arr))
			for _, item := range arr {
				card, ok := item.(map[string]any)
				if !ok {
					continue
				}

				columnID := ""
				if v, ok := card["column_id"].(string); ok {
					columnID = v
				}
				if columnID == "" {
					if col, ok := card["column"].(map[string]any); ok {
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

			items = filtered
		}

		// Build summary
		count := dataCount(items)
		summary := fmt.Sprintf("%d cards", count)
		if cardListAll {
			summary += " (all)"
		} else if cardListPage > 0 {
			summary += fmt.Sprintf(" (page %d)", cardListPage)
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("show", "fizzy card show <number>", "View card details"),
			breadcrumb("create", "fizzy card create --board <id> --title \"title\"", "Create new card"),
			breadcrumb("search", "fizzy search \"query\"", "Search cards"),
		}

		hasNext := linkNext != ""
		if hasNext {
			nextPage := cardListPage + 1
			if cardListPage == 0 {
				nextPage = 2
			}
			breadcrumbs = append(breadcrumbs, breadcrumb("next", fmt.Sprintf("fizzy card list --page %d", nextPage), "Next page"))
		}

		printListPaginated(items, cardColumns, hasNext, linkNext, cardListAll, summary, breadcrumbs)
		return nil
	},
}

var cardShowCmd = &cobra.Command{
	Use:   "show CARD_NUMBER",
	Short: "Show a card",
	Long:  "Shows details of a specific card.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		cardNumber := args[0]

		data, _, err := getSDK().Cards().Get(cmd.Context(), cardNumber)
		if err != nil {
			return convertSDKError(err)
		}

		items := normalizeAny(data)

		// Build summary
		summary := fmt.Sprintf("Card #%s", cardNumber)
		if card, ok := items.(map[string]any); ok {
			if title, ok := card["title"].(string); ok {
				summary = fmt.Sprintf("Card #%s: %s", cardNumber, title)
			}
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("comment", fmt.Sprintf("fizzy comment create --card %s --body \"text\"", cardNumber), "Add comment"),
			breadcrumb("triage", fmt.Sprintf("fizzy card column %s --column <column_id>", cardNumber), "Move to column"),
			breadcrumb("close", fmt.Sprintf("fizzy card close %s", cardNumber), "Close card"),
			breadcrumb("assign", fmt.Sprintf("fizzy card assign %s --user <user_id>", cardNumber), "Assign user"),
		}

		printDetail(items, summary, breadcrumbs)
		return nil
	},
}

// Card create flags
var cardCreateBoard string
var cardCreateTitle string
var cardCreateDescription string
var cardCreateDescriptionFile string
var cardCreateAttach []string
var cardCreateImage string
var cardCreateCreatedAt string

var cardCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a card",
	Long:  "Creates a new card in a board. Use --attach for simple end-appended inline attachments. For precise placement, upload files first and embed <action-text-attachment> tags manually in --description or --description_file.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		boardID, err := requireBoard(cardCreateBoard)
		if err != nil {
			return err
		}
		if cardCreateTitle == "" {
			return newRequiredFlagError("title")
		}

		description, err := resolveRichTextContent(cardCreateDescription, cardCreateDescriptionFile)
		if err != nil {
			return err
		}
		description, err = appendInlineAttachmentsToContent(description, cardCreateAttach)
		if err != nil {
			return err
		}

		ac := getSDK()

		req := &generated.CreateCardRequest{
			BoardId: boardID,
			Title:   cardCreateTitle,
		}
		if description != "" {
			req.Description = description
		}
		if cardCreateImage != "" {
			req.Image = cardCreateImage
		}
		if cardCreateCreatedAt != "" {
			req.CreatedAt = cardCreateCreatedAt
		}

		data, resp, err := ac.Cards().Create(cmd.Context(), req)
		if err != nil {
			return convertSDKError(err)
		}

		items := normalizeAny(data)
		location := resp.Headers.Get("Location")

		// If the API returned an empty body with a Location header (201 Created),
		// follow the Location to fetch the created resource.
		if items == nil && location != "" {
			followData, _, followErr := ac.Cards().Get(cmd.Context(), locationCardNumber(location))
			if followErr == nil {
				items = normalizeAny(followData)
			}
		}

		// Extract card number from response
		cardNumber := ""
		if card, ok := items.(map[string]any); ok {
			if num, ok := card["number"].(float64); ok {
				cardNumber = fmt.Sprintf("%d", int(num))
			}
		}

		// Build breadcrumbs
		var breadcrumbs []Breadcrumb
		if cardNumber != "" {
			breadcrumbs = []Breadcrumb{
				breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card details"),
				breadcrumb("triage", fmt.Sprintf("fizzy card column %s --column <column_id>", cardNumber), "Move to column"),
				breadcrumb("comment", fmt.Sprintf("fizzy comment create --card %s --body \"text\"", cardNumber), "Add comment"),
			}
		}

		if location != "" {
			printMutationWithLocation(items, location, breadcrumbs)
		} else {
			printMutation(items, "", breadcrumbs)
		}
		return nil
	},
}

// Card update flags
var cardUpdateTitle string
var cardUpdateDescription string
var cardUpdateDescriptionFile string
var cardUpdateAttach []string
var cardUpdateImage string
var cardUpdateCreatedAt string

var cardUpdateCmd = &cobra.Command{
	Use:   "update CARD_NUMBER",
	Short: "Update a card",
	Long:  "Updates an existing card. Use --attach for simple end-appended inline attachments. For precise placement, upload files first and embed <action-text-attachment> tags manually in --description or --description_file.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		cardNumber := args[0]

		hasDescriptionInput := cardUpdateDescription != "" || cardUpdateDescriptionFile != ""
		description, err := resolveRichTextContent(cardUpdateDescription, cardUpdateDescriptionFile)
		if err != nil {
			return err
		}
		if len(cardUpdateAttach) > 0 && !hasDescriptionInput {
			currentData, _, getErr := getSDK().Cards().Get(cmd.Context(), cardNumber)
			if getErr != nil {
				return convertSDKError(getErr)
			}
			if current, ok := normalizeAny(currentData).(map[string]any); ok {
				if currentDescription, ok := current["description_html"].(string); ok {
					description = currentDescription
				}
			}
		}
		description, err = appendInlineAttachmentsToContent(description, cardUpdateAttach)
		if err != nil {
			return err
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card details"),
			breadcrumb("triage", fmt.Sprintf("fizzy card column %s --column <column_id>", cardNumber), "Move to column"),
			breadcrumb("comment", fmt.Sprintf("fizzy comment create --card %s --body \"text\"", cardNumber), "Add comment"),
		}

		req := &generated.UpdateCardRequest{}
		if cardUpdateTitle != "" {
			req.Title = cardUpdateTitle
		}
		if description != "" {
			req.Description = description
		}
		if cardUpdateImage != "" {
			req.Image = cardUpdateImage
		}
		if cardUpdateCreatedAt != "" {
			req.CreatedAt = cardUpdateCreatedAt
		}

		data, _, err := getSDK().Cards().Update(cmd.Context(), cardNumber, req)
		if err != nil {
			return convertSDKError(err)
		}
		printMutation(normalizeAny(data), "", breadcrumbs)
		return nil
	},
}

var cardDeleteCmd = &cobra.Command{
	Use:   "delete CARD_NUMBER",
	Short: "Delete a card",
	Long:  "Deletes a card.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		_, err := getSDK().Cards().Delete(cmd.Context(), args[0])
		if err != nil {
			return convertSDKError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("cards", "fizzy card list", "List cards"),
			breadcrumb("create", "fizzy card create --board <id> --title \"title\"", "Create new card"),
		}

		printMutation(map[string]any{
			"deleted": true,
		}, "", breadcrumbs)
		return nil
	},
}

var cardCloseCmd = &cobra.Command{
	Use:   "close CARD_NUMBER",
	Short: "Close a card",
	Long:  "Closes a card.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		cardNumber := args[0]

		_, err := getSDK().Cards().Close(cmd.Context(), cardNumber)
		if err != nil {
			return convertSDKError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("reopen", fmt.Sprintf("fizzy card reopen %s", cardNumber), "Reopen card"),
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
		}

		printMutation(map[string]any{}, "", breadcrumbs)
		return nil
	},
}

var cardReopenCmd = &cobra.Command{
	Use:   "reopen CARD_NUMBER",
	Short: "Reopen a card",
	Long:  "Reopens a closed card.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		cardNumber := args[0]

		_, err := getSDK().Cards().Reopen(cmd.Context(), cardNumber)
		if err != nil {
			return convertSDKError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("close", fmt.Sprintf("fizzy card close %s", cardNumber), "Close card"),
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
			breadcrumb("triage", fmt.Sprintf("fizzy card column %s --column <column_id>", cardNumber), "Move to column"),
		}

		printMutation(map[string]any{}, "", breadcrumbs)
		return nil
	},
}

var cardPostponeCmd = &cobra.Command{
	Use:   "postpone CARD_NUMBER",
	Short: "Postpone a card",
	Long:  "Moves a card to 'Not Now'.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		cardNumber := args[0]

		_, err := getSDK().Cards().Postpone(cmd.Context(), cardNumber)
		if err != nil {
			return convertSDKError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
			breadcrumb("triage", fmt.Sprintf("fizzy card column %s --column <column_id>", cardNumber), "Move to column"),
		}

		printMutation(map[string]any{}, "", breadcrumbs)
		return nil
	},
}

// Card move flags
var cardMoveBoard string

var cardMoveCmd = &cobra.Command{
	Use:   "move CARD_NUMBER",
	Short: "Move card to a different board",
	Long:  "Moves a card to a different board.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		if cardMoveBoard == "" {
			return newRequiredFlagError("to")
		}

		cardNumber := args[0]

		moveData, _, err := getSDK().Cards().Move(cmd.Context(), cardNumber, &generated.MoveCardRequest{
			BoardId: cardMoveBoard,
		})
		if err != nil {
			return convertSDKError(err)
		}

		items := normalizeAny(moveData)

		// Build summary with card title if available
		summary := fmt.Sprintf("Card #%s moved to board %s", cardNumber, cardMoveBoard)
		if card, ok := items.(map[string]any); ok {
			if title, ok := card["title"].(string); ok {
				summary = fmt.Sprintf("Card #%s \"%s\" moved to board %s", cardNumber, title, cardMoveBoard)
			}
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
			breadcrumb("triage", fmt.Sprintf("fizzy card column %s --column <column_id>", cardNumber), "Move to column"),
		}

		printMutation(items, summary, breadcrumbs)
		return nil
	},
}

// Card column flags
var cardColumnColumn string

var cardColumnCmd = &cobra.Command{
	Use:   "column CARD_NUMBER",
	Short: "Move card to column",
	Long:  "Moves a card to a specific column.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		if cardColumnColumn == "" {
			return newRequiredFlagError("column")
		}

		cardNumber := args[0]

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("untriage", fmt.Sprintf("fizzy card untriage %s", cardNumber), "Send back to triage"),
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
			breadcrumb("close", fmt.Sprintf("fizzy card close %s", cardNumber), "Close card"),
		}

		ac := getSDK()
		if pseudo, ok := parsePseudoColumnID(cardColumnColumn); ok {
			switch pseudo.Kind {
			case "triage":
				_, err := ac.Cards().UnTriage(cmd.Context(), cardNumber)
				if err != nil {
					return convertSDKError(err)
				}
				printMutation(map[string]any{}, "", breadcrumbs)
				return nil
			case "not_now":
				_, err := ac.Cards().Postpone(cmd.Context(), cardNumber)
				if err != nil {
					return convertSDKError(err)
				}
				printMutation(map[string]any{}, "", breadcrumbs)
				return nil
			case "closed":
				_, err := ac.Cards().Close(cmd.Context(), cardNumber)
				if err != nil {
					return convertSDKError(err)
				}
				printMutation(map[string]any{}, "", breadcrumbs)
				return nil
			}
		}

		_, err := ac.Cards().Triage(cmd.Context(), cardNumber, &generated.TriageCardRequest{
			ColumnId: cardColumnColumn,
		})
		if err != nil {
			return convertSDKError(err)
		}

		printMutation(map[string]any{}, "", breadcrumbs)
		return nil
	},
}

var cardUntriageCmd = &cobra.Command{
	Use:   "untriage CARD_NUMBER",
	Short: "Send card back to triage",
	Long:  "Removes a card from its column and sends it back to triage.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		cardNumber := args[0]

		_, err := getSDK().Cards().UnTriage(cmd.Context(), cardNumber)
		if err != nil {
			return convertSDKError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("triage", fmt.Sprintf("fizzy card column %s --column <column_id>", cardNumber), "Move to column"),
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
		}

		printMutation(map[string]any{
			"untriaged": true,
		}, "", breadcrumbs)
		return nil
	},
}

// Card assign flags
var cardAssignUser string

var cardAssignCmd = &cobra.Command{
	Use:   "assign CARD_NUMBER",
	Short: "Toggle assignment on a card",
	Long:  "Toggles a user's assignment on a card.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		if cardAssignUser == "" {
			return newRequiredFlagError("user")
		}

		cardNumber := args[0]

		_, err := getSDK().Cards().Assign(cmd.Context(), cardNumber, &generated.AssignCardRequest{
			AssigneeId: cardAssignUser,
		})
		if err != nil {
			return convertSDKError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
			breadcrumb("people", "fizzy user list", "List users"),
		}

		printMutation(map[string]any{}, "", breadcrumbs)
		return nil
	},
}

var cardSelfAssignCmd = &cobra.Command{
	Use:   "self-assign CARD_NUMBER",
	Short: "Toggle self-assignment on a card",
	Long:  "Toggles the current user's assignment on a card.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		cardNumber := args[0]

		_, err := getSDK().Cards().SelfAssign(cmd.Context(), cardNumber)
		if err != nil {
			return convertSDKError(err)
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
		}

		printMutation(map[string]any{}, "", breadcrumbs)
		return nil
	},
}

// Card tag flags
var cardTagTag string

var cardTagCmd = &cobra.Command{
	Use:   "tag CARD_NUMBER",
	Short: "Toggle tag on a card",
	Long:  "Toggles a tag on a card. Creates the tag if it doesn't exist.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		if cardTagTag == "" {
			return newRequiredFlagError("tag")
		}

		cardNumber := args[0]

		resp, err := getSDK().Cards().Tag(cmd.Context(), cardNumber, &generated.TagCardRequest{TagTitle: cardTagTag})
		if err != nil {
			return convertSDKError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
			breadcrumb("tags", "fizzy tag list", "List tags"),
		}

		data := normalizeAny(resp.Data)
		if data == nil {
			data = map[string]any{}
		}
		printMutation(data, "", breadcrumbs)
		return nil
	},
}

var cardWatchCmd = &cobra.Command{
	Use:   "watch CARD_NUMBER",
	Short: "Watch a card",
	Long:  "Subscribes to notifications for a card.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		cardNumber := args[0]

		_, err := getSDK().Cards().Watch(cmd.Context(), cardNumber)
		if err != nil {
			return convertSDKError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
			breadcrumb("notifications", "fizzy notification list", "View notifications"),
		}

		printMutation(map[string]any{}, "", breadcrumbs)
		return nil
	},
}

var cardUnwatchCmd = &cobra.Command{
	Use:   "unwatch CARD_NUMBER",
	Short: "Unwatch a card",
	Long:  "Unsubscribes from notifications for a card.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		cardNumber := args[0]

		_, err := getSDK().Cards().Unwatch(cmd.Context(), cardNumber)
		if err != nil {
			return convertSDKError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
			breadcrumb("notifications", "fizzy notification list", "View notifications"),
		}

		printMutation(map[string]any{}, "", breadcrumbs)
		return nil
	},
}

var cardImageRemoveCmd = &cobra.Command{
	Use:   "image-remove CARD_NUMBER",
	Short: "Remove card header image",
	Long:  "Removes the header image from a card.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		cardNumber := args[0]

		_, err := getSDK().Cards().DeleteImage(cmd.Context(), cardNumber)
		if err != nil {
			return convertSDKError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
			breadcrumb("update", fmt.Sprintf("fizzy card update %s", cardNumber), "Update card"),
		}

		printMutation(map[string]any{}, "", breadcrumbs)
		return nil
	},
}

var cardPinCmd = &cobra.Command{
	Use:   "pin CARD_NUMBER",
	Short: "Pin a card",
	Long:  "Pins a card for quick access.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		cardNumber := args[0]

		_, err := getSDK().Cards().Pin(cmd.Context(), cardNumber)
		if err != nil {
			return convertSDKError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
			breadcrumb("pins", "fizzy pin list", "List pinned cards"),
			breadcrumb("unpin", fmt.Sprintf("fizzy card unpin %s", cardNumber), "Unpin card"),
		}

		printMutation(map[string]any{}, "", breadcrumbs)
		return nil
	},
}

var cardUnpinCmd = &cobra.Command{
	Use:   "unpin CARD_NUMBER",
	Short: "Unpin a card",
	Long:  "Unpins a card, removing it from your pinned list.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		cardNumber := args[0]

		_, err := getSDK().Cards().Unpin(cmd.Context(), cardNumber)
		if err != nil {
			return convertSDKError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
			breadcrumb("pins", "fizzy pin list", "List pinned cards"),
			breadcrumb("pin", fmt.Sprintf("fizzy card pin %s", cardNumber), "Pin card"),
		}

		printMutation(map[string]any{}, "", breadcrumbs)
		return nil
	},
}

var cardGoldenCmd = &cobra.Command{
	Use:   "golden CARD_NUMBER",
	Short: "Mark card as golden",
	Long:  "Marks a card as golden (starred/important).",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		cardNumber := args[0]

		_, err := getSDK().Cards().Gold(cmd.Context(), cardNumber)
		if err != nil {
			return convertSDKError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
			breadcrumb("golden", "fizzy card list --indexed-by golden", "List golden cards"),
		}

		printMutation(map[string]any{}, "", breadcrumbs)
		return nil
	},
}

var cardUngoldenCmd = &cobra.Command{
	Use:   "ungolden CARD_NUMBER",
	Short: "Remove golden status from card",
	Long:  "Removes the golden (starred/important) status from a card.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		cardNumber := args[0]

		_, err := getSDK().Cards().Ungold(cmd.Context(), cardNumber)
		if err != nil {
			return convertSDKError(err)
		}

		// Build breadcrumbs
		breadcrumbs := []Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
			breadcrumb("golden", "fizzy card list --indexed-by golden", "List golden cards"),
		}

		printMutation(map[string]any{}, "", breadcrumbs)
		return nil
	},
}

var cardPublishCmd = &cobra.Command{
	Use:   "publish CARD_NUMBER",
	Short: "Publish a card",
	Long:  "Publishes a card.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		cardNumber := args[0]

		_, err := getSDK().Cards().Publish(cmd.Context(), cardNumber)
		if err != nil {
			return convertSDKError(err)
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
		}

		printMutation(map[string]any{}, "", breadcrumbs)
		return nil
	},
}

var cardMarkReadCmd = &cobra.Command{
	Use:   "mark-read CARD_NUMBER",
	Short: "Mark a card as read",
	Long:  "Marks a card as read.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		cardNumber := args[0]

		_, err := getSDK().Cards().MarkRead(cmd.Context(), cardNumber)
		if err != nil {
			return convertSDKError(err)
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
			breadcrumb("mark-unread", fmt.Sprintf("fizzy card mark-unread %s", cardNumber), "Mark as unread"),
		}

		printMutation(map[string]any{}, "", breadcrumbs)
		return nil
	},
}

var cardMarkUnreadCmd = &cobra.Command{
	Use:   "mark-unread CARD_NUMBER",
	Short: "Mark a card as unread",
	Long:  "Marks a card as unread.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuthAndAccount(); err != nil {
			return err
		}

		cardNumber := args[0]

		_, err := getSDK().Cards().MarkUnread(cmd.Context(), cardNumber)
		if err != nil {
			return convertSDKError(err)
		}

		breadcrumbs := []Breadcrumb{
			breadcrumb("show", fmt.Sprintf("fizzy card show %s", cardNumber), "View card"),
			breadcrumb("mark-read", fmt.Sprintf("fizzy card mark-read %s", cardNumber), "Mark as read"),
		}

		printMutation(map[string]any{}, "", breadcrumbs)
		return nil
	},
}

// locationCardNumber extracts a card number from a Location header path.
// Example: "/account/cards/42.json" → "42"
func locationCardNumber(location string) string {
	// Strip query string and fragment
	if i := strings.IndexAny(location, "?#"); i >= 0 {
		location = location[:i]
	}
	// Strip trailing .json
	location = strings.TrimSuffix(location, ".json")
	// Take the last path segment
	if i := strings.LastIndex(location, "/"); i >= 0 {
		return location[i+1:]
	}
	return location
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
	cardCreateCmd.Flags().StringVar(&cardCreateDescription, "description", "", "Card description (markdown or HTML)")
	cardCreateCmd.Flags().StringVar(&cardCreateDescriptionFile, "description_file", "", "Read description from file (markdown or HTML)")
	cardCreateCmd.Flags().StringArrayVar(&cardCreateAttach, "attach", nil, "Upload and append inline attachment at the end of the description. Repeatable.")
	cardCreateCmd.Flags().StringVar(&cardCreateImage, "image", "", "Header image signed ID")
	cardCreateCmd.Flags().StringVar(&cardCreateCreatedAt, "created-at", "", "Custom created_at timestamp")
	cardCmd.AddCommand(cardCreateCmd)

	// Update
	cardUpdateCmd.Flags().StringVar(&cardUpdateTitle, "title", "", "Card title")
	cardUpdateCmd.Flags().StringVar(&cardUpdateDescription, "description", "", "Card description (markdown or HTML)")
	cardUpdateCmd.Flags().StringVar(&cardUpdateDescriptionFile, "description_file", "", "Read description from file (markdown or HTML)")
	cardUpdateCmd.Flags().StringArrayVar(&cardUpdateAttach, "attach", nil, "Upload and append inline attachment at the end of the description. Repeatable.")
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

	// Publish
	cardCmd.AddCommand(cardPublishCmd)

	// Read state
	cardCmd.AddCommand(cardMarkReadCmd)
	cardCmd.AddCommand(cardMarkUnreadCmd)
}
