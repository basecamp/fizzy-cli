package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/basecamp/fizzy-cli/internal/client"
	"github.com/basecamp/fizzy-cli/internal/errors"
	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migration tools",
	Long:  "Commands for migrating data between accounts.",
}

// Migrate board flags
var migrateBoardFrom string
var migrateBoardTo string
var migrateBoardIncludeComments bool
var migrateBoardIncludeSteps bool
var migrateBoardIncludeImages bool
var migrateBoardDryRun bool

var migrateBoardCmd = &cobra.Command{
	Use:   "board BOARD_ID",
	Short: "Migrate a board to another account",
	Long: `Migrates a board and all its cards from one account to another.

This command copies:
- The board with the same name
- All columns (preserving order)
- All cards with their titles, descriptions, timestamps, tags, and state
- Optionally: comments and steps

What cannot be migrated:
- Card creators (will become the migrating user)
- User assignments (team will need to reassign)
- Card numbers (will be new sequential numbers)
- Comment authors (will become the migrating user)

Example:
  fizzy migrate board 12345 --from personal --to team-acme
  fizzy migrate board 12345 --from personal --to team-acme --include-comments --include-steps`,
	Args: cobra.ExactArgs(1),
	Run:  runMigrateBoard,
}

type migrationStats struct {
	boardCreated    bool
	targetBoardID   string
	targetBoardName string
	columnsCreated  int
	cardsCreated    int
	tagsApplied     int
	commentsCreated int
	stepsCreated    int
	imagesMigrated  int
	cardMapping     map[int]int // source card number -> target card number
}

func runMigrateBoard(cmd *cobra.Command, args []string) {
	if err := requireAuth(); err != nil {
		exitWithError(err)
	}

	// Validate flags
	if migrateBoardFrom == "" {
		exitWithError(errors.NewInvalidArgsError("--from flag is required"))
	}
	if migrateBoardTo == "" {
		exitWithError(errors.NewInvalidArgsError("--to flag is required"))
	}
	if migrateBoardFrom == migrateBoardTo {
		exitWithError(errors.NewInvalidArgsError("--from and --to accounts must be different"))
	}

	sourceBoardID := args[0]
	stats := &migrationStats{
		cardMapping: make(map[int]int),
	}

	// Create clients for both accounts
	sourceClient := createClientForAccount(migrateBoardFrom)
	targetClient := createClientForAccount(migrateBoardTo)

	// 1. Verify access to both accounts
	fmt.Fprintf(os.Stderr, "Verifying access to accounts...\n")
	if err := verifyAccountAccess(migrateBoardFrom, migrateBoardTo); err != nil {
		exitWithError(err)
	}

	// 2. Get source board
	fmt.Fprintf(os.Stderr, "Fetching source board...\n")
	sourceBoard, err := getBoard(sourceClient, sourceBoardID)
	if err != nil {
		exitWithError(errors.NewError(fmt.Sprintf("Failed to fetch source board: %v", err)))
	}

	boardName := getStringField(sourceBoard, "name")
	fmt.Fprintf(os.Stderr, "Source board: %s\n", boardName)

	// 3. Get source columns
	fmt.Fprintf(os.Stderr, "Fetching source columns...\n")
	sourceColumns, err := getColumns(sourceClient, sourceBoardID)
	if err != nil {
		exitWithError(errors.NewError(fmt.Sprintf("Failed to fetch source columns: %v", err)))
	}

	// 4. Get all cards from source board
	fmt.Fprintf(os.Stderr, "Fetching source cards...\n")
	sourceCards, err := getAllCards(sourceClient, sourceBoardID)
	if err != nil {
		exitWithError(errors.NewError(fmt.Sprintf("Failed to fetch source cards: %v", err)))
	}

	fmt.Fprintf(os.Stderr, "Found %d cards to migrate\n", len(sourceCards))

	// Dry run: just show what would be done
	if migrateBoardDryRun {
		printDryRunSummary(boardName, sourceColumns, sourceCards)
		printSuccess(map[string]interface{}{
			"dry_run":      true,
			"board":        boardName,
			"columns":      len(sourceColumns),
			"cards":        len(sourceCards),
			"from_account": migrateBoardFrom,
			"to_account":   migrateBoardTo,
		})
		return
	}

	// 5. Create target board
	fmt.Fprintf(os.Stderr, "Creating target board...\n")
	targetBoardID, err := createBoard(targetClient, boardName)
	if err != nil {
		exitWithError(errors.NewError(fmt.Sprintf("Failed to create target board: %v", err)))
	}
	stats.boardCreated = true
	stats.targetBoardID = targetBoardID
	stats.targetBoardName = boardName

	// 6. Create columns in target (preserve order)
	fmt.Fprintf(os.Stderr, "Creating columns...\n")
	columnMapping := make(map[string]string) // source column ID -> target column ID
	for _, col := range sourceColumns {
		colMap, ok := col.(map[string]interface{})
		if !ok {
			continue
		}

		// Skip pseudo-columns (not_now, triage, done)
		if kind, ok := colMap["kind"].(string); ok && kind != "real" {
			continue
		}
		if pseudo, ok := colMap["pseudo"].(bool); ok && pseudo {
			continue
		}

		colName := getStringField(colMap, "name")
		colColor := getStringField(colMap, "color")
		sourceColID := getStringField(colMap, "id")

		targetColID, err := createColumn(targetClient, targetBoardID, colName, colColor)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to create column '%s': %v\n", colName, err)
			continue
		}
		columnMapping[sourceColID] = targetColID
		stats.columnsCreated++
	}

	// 7. Migrate cards
	fmt.Fprintf(os.Stderr, "Migrating cards...\n")
	for i, card := range sourceCards {
		cardMap, ok := card.(map[string]interface{})
		if !ok {
			continue
		}

		sourceCardNum := getIntField(cardMap, "number")
		fmt.Fprintf(os.Stderr, "  [%d/%d] Card #%d: %s\n", i+1, len(sourceCards), sourceCardNum, getStringField(cardMap, "title"))

		targetCardNum, err := migrateCard(sourceClient, targetClient, cardMap, targetBoardID, columnMapping, stats)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Warning: Failed to migrate card #%d: %v\n", sourceCardNum, err)
			continue
		}

		stats.cardMapping[sourceCardNum] = targetCardNum
		stats.cardsCreated++
	}

	// Print summary
	printMigrationSummary(stats)

	printSuccess(map[string]interface{}{
		"migrated":         true,
		"board_id":         stats.targetBoardID,
		"board_name":       stats.targetBoardName,
		"columns_created":  stats.columnsCreated,
		"cards_created":    stats.cardsCreated,
		"tags_applied":     stats.tagsApplied,
		"comments_created": stats.commentsCreated,
		"steps_created":    stats.stepsCreated,
		"images_migrated":  stats.imagesMigrated,
		"card_mapping":     stats.cardMapping,
	})
}

func createClientForAccount(account string) client.API {
	c := client.New(cfg.APIURL, cfg.Token, account)
	c.Verbose = cfgVerbose
	return c
}

func verifyAccountAccess(sourceAccount, targetAccount string) error {
	// Get identity to verify access to both accounts
	c := client.New(cfg.APIURL, cfg.Token, "")
	resp, err := c.Get(cfg.APIURL + "/my/identity.json")
	if err != nil {
		return errors.NewError(fmt.Sprintf("Failed to fetch identity: %v", err))
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		return errors.NewError("Invalid identity response")
	}

	accounts, ok := data["accounts"].([]interface{})
	if !ok {
		return errors.NewError("No accounts found in identity response")
	}

	foundSource := false
	foundTarget := false

	for _, acc := range accounts {
		accMap, ok := acc.(map[string]interface{})
		if !ok {
			continue
		}
		slug := getStringField(accMap, "slug")
		// Slug from API includes leading slash, e.g., "/6086023"
		// Normalize for comparison
		normalizedSlug := strings.TrimPrefix(slug, "/")
		if normalizedSlug == sourceAccount || slug == sourceAccount {
			foundSource = true
		}
		if normalizedSlug == targetAccount || slug == targetAccount {
			foundTarget = true
		}
	}

	if !foundSource {
		return errors.NewError(fmt.Sprintf("You don't have access to source account '%s'", sourceAccount))
	}
	if !foundTarget {
		return errors.NewError(fmt.Sprintf("You don't have access to target account '%s'", targetAccount))
	}

	return nil
}

func getBoard(c client.API, boardID string) (map[string]interface{}, error) {
	resp, err := c.Get("/boards/" + boardID + ".json")
	if err != nil {
		return nil, err
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		return nil, errors.NewError("Invalid board response")
	}

	return data, nil
}

func getColumns(c client.API, boardID string) ([]interface{}, error) {
	resp, err := c.Get("/boards/" + boardID + "/columns.json")
	if err != nil {
		return nil, err
	}

	data, ok := resp.Data.([]interface{})
	if !ok {
		return nil, errors.NewError("Invalid columns response")
	}

	return data, nil
}

func getAllCards(c client.API, boardID string) ([]interface{}, error) {
	resp, err := c.GetWithPagination("/cards.json?board_ids[]="+boardID, true)
	if err != nil {
		return nil, err
	}

	data, ok := resp.Data.([]interface{})
	if !ok {
		return nil, errors.NewError("Invalid cards response")
	}

	return data, nil
}

func createBoard(c client.API, name string) (string, error) {
	body := map[string]interface{}{
		"board": map[string]interface{}{
			"name": name,
		},
	}

	resp, err := c.Post("/boards.json", body)
	if err != nil {
		return "", err
	}

	// Follow location to get the created board
	if resp.Location != "" {
		followResp, err := c.FollowLocation(resp.Location)
		if err == nil && followResp != nil {
			if data, ok := followResp.Data.(map[string]interface{}); ok {
				return getStringField(data, "id"), nil
			}
		}
	}

	// Try to get ID from response data
	if data, ok := resp.Data.(map[string]interface{}); ok {
		return getStringField(data, "id"), nil
	}

	return "", errors.NewError("Failed to get board ID from response")
}

func createColumn(c client.API, boardID, name, color string) (string, error) {
	colParams := map[string]interface{}{
		"name": name,
	}
	if color != "" {
		colParams["color"] = color
	}

	body := map[string]interface{}{
		"column": colParams,
	}

	resp, err := c.Post("/boards/"+boardID+"/columns.json", body)
	if err != nil {
		return "", err
	}

	// Follow location to get the created column
	if resp.Location != "" {
		followResp, err := c.FollowLocation(resp.Location)
		if err == nil && followResp != nil {
			if data, ok := followResp.Data.(map[string]interface{}); ok {
				return getStringField(data, "id"), nil
			}
		}
	}

	// Try to get ID from response data
	if data, ok := resp.Data.(map[string]interface{}); ok {
		return getStringField(data, "id"), nil
	}

	return "", errors.NewError("Failed to get column ID from response")
}

func migrateCard(sourceClient, targetClient client.API, sourceCard map[string]interface{}, targetBoardID string, columnMapping map[string]string, stats *migrationStats) (int, error) {
	// Extract card data
	title := getStringField(sourceCard, "title")
	description := getStringField(sourceCard, "description")
	descriptionHTML := getStringField(sourceCard, "description_html")
	createdAt := getStringField(sourceCard, "created_at")
	sourceCardNum := getIntField(sourceCard, "number")

	// Migrate inline attachments in description if requested
	if migrateBoardIncludeImages && descriptionHTML != "" {
		migratedDesc, attachmentCount := migrateInlineAttachments(sourceClient, targetClient, descriptionHTML)
		if attachmentCount > 0 {
			// Use the migrated HTML as the description
			description = migratedDesc
			stats.imagesMigrated += attachmentCount
		}
	}

	// Create card in target
	cardParams := map[string]interface{}{
		"title": title,
	}
	if description != "" {
		cardParams["description"] = description
	}
	if createdAt != "" {
		cardParams["created_at"] = createdAt
	}

	body := map[string]interface{}{
		"board_id": targetBoardID,
		"card":     cardParams,
	}

	resp, err := targetClient.Post("/cards.json", body)
	if err != nil {
		return 0, err
	}

	// Get the new card number
	var newCardNum int
	var newCardData map[string]interface{}

	if resp.Location != "" {
		followResp, err := targetClient.FollowLocation(resp.Location)
		if err == nil && followResp != nil {
			if data, ok := followResp.Data.(map[string]interface{}); ok {
				newCardData = data
				newCardNum = getIntField(data, "number")
			}
		}
	}
	if newCardNum == 0 {
		if data, ok := resp.Data.(map[string]interface{}); ok {
			newCardData = data
			newCardNum = getIntField(data, "number")
		}
	}
	if newCardNum == 0 {
		return 0, errors.NewError("Failed to get new card number")
	}

	newCardNumStr := strconv.Itoa(newCardNum)
	_ = newCardData // might use later for additional operations

	// Apply tags
	if tags, ok := sourceCard["tags"].([]interface{}); ok {
		for _, tag := range tags {
			tagName, ok := tag.(string)
			if !ok {
				continue
			}
			err := applyTag(targetClient, newCardNumStr, tagName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "    Warning: Failed to apply tag '%s': %v\n", tagName, err)
			} else {
				stats.tagsApplied++
			}
		}
	}

	// Move to correct column
	sourceColumnID := getCardColumnID(sourceCard)
	if sourceColumnID != "" {
		if targetColumnID, ok := columnMapping[sourceColumnID]; ok {
			err := moveToColumn(targetClient, newCardNumStr, targetColumnID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "    Warning: Failed to move card to column: %v\n", err)
			}
		}
	}

	// Apply card state
	status := getStringField(sourceCard, "status")
	golden := getBoolField(sourceCard, "golden")

	// Check if card is closed
	if status == "closed" {
		err := closeCard(targetClient, newCardNumStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "    Warning: Failed to close card: %v\n", err)
		}
	}

	// Check if card is in not_now (need to check indexed_by or another indicator)
	// Cards in not_now would have been fetched with indexed_by=not_now, but we're fetching all
	// The column might be indicated differently - for now, skip this as it's complex to detect

	// Apply golden status
	if golden {
		err := markGolden(targetClient, newCardNumStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "    Warning: Failed to mark card as golden: %v\n", err)
		}
	}

	// Migrate comments if requested
	if migrateBoardIncludeComments {
		commentsCreated, err := migrateComments(sourceClient, targetClient, strconv.Itoa(sourceCardNum), newCardNumStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "    Warning: Failed to migrate comments: %v\n", err)
		}
		stats.commentsCreated += commentsCreated
	}

	// Migrate steps if requested
	if migrateBoardIncludeSteps {
		stepsCreated, err := migrateSteps(sourceClient, targetClient, sourceCard, newCardNumStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "    Warning: Failed to migrate steps: %v\n", err)
		}
		stats.stepsCreated += stepsCreated
	}

	// Migrate card image if requested
	if migrateBoardIncludeImages {
		imageURL := getStringField(sourceCard, "image_url")
		if imageURL != "" {
			err := migrateCardImage(sourceClient, targetClient, imageURL, newCardNumStr)
			if err != nil {
				fmt.Fprintf(os.Stderr, "    Warning: Failed to migrate image: %v\n", err)
			} else {
				stats.imagesMigrated++
			}
		}
	}

	return newCardNum, nil
}

func getCardColumnID(card map[string]interface{}) string {
	// Try column_id directly
	if colID, ok := card["column_id"].(string); ok && colID != "" {
		return colID
	}

	// Try nested column object
	if col, ok := card["column"].(map[string]interface{}); ok {
		if colID, ok := col["id"].(string); ok {
			return colID
		}
	}

	return ""
}

func applyTag(c client.API, cardNum, tagName string) error {
	body := map[string]interface{}{
		"tag_title": tagName,
	}
	_, err := c.Post("/cards/"+cardNum+"/taggings.json", body)
	return err
}

func moveToColumn(c client.API, cardNum, columnID string) error {
	body := map[string]interface{}{
		"column_id": columnID,
	}
	_, err := c.Post("/cards/"+cardNum+"/triage.json", body)
	return err
}

func closeCard(c client.API, cardNum string) error {
	_, err := c.Post("/cards/"+cardNum+"/closure.json", nil)
	return err
}

func markGolden(c client.API, cardNum string) error {
	_, err := c.Post("/cards/"+cardNum+"/goldness.json", nil)
	return err
}

func migrateComments(sourceClient, targetClient client.API, sourceCardNum, targetCardNum string) (int, error) {
	// Get all comments from source card
	resp, err := sourceClient.GetWithPagination("/cards/"+sourceCardNum+"/comments.json", true)
	if err != nil {
		return 0, err
	}

	comments, ok := resp.Data.([]interface{})
	if !ok {
		return 0, nil // No comments or invalid response
	}

	created := 0
	for _, comment := range comments {
		commentMap, ok := comment.(map[string]interface{})
		if !ok {
			continue
		}

		// Get comment body - it might be a string or an object with html/plain_text
		var bodyContent string
		var bodyHTML string
		if body, ok := commentMap["body"].(map[string]interface{}); ok {
			// Try to get HTML content first, then plain_text
			if html, ok := body["html"].(string); ok {
				bodyHTML = html
				bodyContent = html
			} else if plain, ok := body["plain_text"].(string); ok {
				bodyContent = plain
			}
		} else if body, ok := commentMap["body"].(string); ok {
			bodyContent = body
		}

		if bodyContent == "" {
			continue
		}

		// Migrate inline attachments in comment if we have HTML and images are enabled
		if migrateBoardIncludeImages && bodyHTML != "" {
			migratedBody, _ := migrateInlineAttachments(sourceClient, targetClient, bodyHTML)
			bodyContent = migratedBody
		}

		createdAt := getStringField(commentMap, "created_at")

		commentParams := map[string]interface{}{
			"body": bodyContent,
		}
		if createdAt != "" {
			commentParams["created_at"] = createdAt
		}

		reqBody := map[string]interface{}{
			"comment": commentParams,
		}

		_, err := targetClient.Post("/cards/"+targetCardNum+"/comments.json", reqBody)
		if err != nil {
			fmt.Fprintf(os.Stderr, "      Warning: Failed to create comment: %v\n", err)
			continue
		}
		created++
	}

	return created, nil
}

func migrateSteps(sourceClient, targetClient client.API, sourceCard map[string]interface{}, targetCardNum string) (int, error) {
	// Steps are not included in card list response, need to fetch card details
	steps, ok := sourceCard["steps"].([]interface{})
	if !ok || len(steps) == 0 {
		// Fetch card details to get steps
		sourceCardNum := getIntField(sourceCard, "number")
		if sourceCardNum == 0 {
			return 0, nil
		}

		cardResp, err := sourceClient.Get("/cards/" + strconv.Itoa(sourceCardNum) + ".json")
		if err != nil {
			return 0, fmt.Errorf("failed to fetch card details: %w", err)
		}

		cardData, ok := cardResp.Data.(map[string]interface{})
		if !ok {
			return 0, nil
		}

		steps, ok = cardData["steps"].([]interface{})
		if !ok || len(steps) == 0 {
			return 0, nil
		}
	}

	created := 0
	for _, step := range steps {
		stepMap, ok := step.(map[string]interface{})
		if !ok {
			continue
		}

		content := getStringField(stepMap, "content")
		if content == "" {
			continue
		}

		completed := getBoolField(stepMap, "completed")

		stepParams := map[string]interface{}{
			"content": content,
		}
		if completed {
			stepParams["completed"] = true
		}

		reqBody := map[string]interface{}{
			"step": stepParams,
		}

		_, err := targetClient.Post("/cards/"+targetCardNum+"/steps.json", reqBody)
		if err != nil {
			fmt.Fprintf(os.Stderr, "      Warning: Failed to create step: %v\n", err)
			continue
		}
		created++
	}

	return created, nil
}

func migrateCardImage(sourceClient, targetClient client.API, imageURL, targetCardNum string) error {
	// Create a temp file to download the image
	tempDir := os.TempDir()
	tempFile := filepath.Join(tempDir, fmt.Sprintf("fizzy-migrate-image-%s.jpg", targetCardNum))
	defer func() { _ = os.Remove(tempFile) }()

	// Download the image from source
	// The imageURL is a relative path like /6086023/rails/active_storage/blobs/redirect/...
	err := sourceClient.DownloadFile(imageURL, tempFile)
	if err != nil {
		return fmt.Errorf("failed to download image: %w", err)
	}

	// Upload to target account
	uploadResp, err := targetClient.UploadFile(tempFile)
	if err != nil {
		return fmt.Errorf("failed to upload image: %w", err)
	}

	// Get the signed_id from upload response
	uploadData, ok := uploadResp.Data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid upload response")
	}

	signedID := getStringField(uploadData, "signed_id")
	if signedID == "" {
		return fmt.Errorf("no signed_id in upload response")
	}

	// Update the card with the new image
	cardParams := map[string]interface{}{
		"image": signedID,
	}
	body := map[string]interface{}{
		"card": cardParams,
	}

	_, err = targetClient.Patch("/cards/"+targetCardNum+".json", body)
	if err != nil {
		return fmt.Errorf("failed to set image on card: %w", err)
	}

	return nil
}

// migrateInlineAttachments finds all <action-text-attachment> elements in HTML,
// downloads and re-uploads them to the target account, and returns the modified HTML
// with updated attachment references.
func migrateInlineAttachments(sourceClient, targetClient client.API, html string) (string, int) {
	attachments := parseAttachments(html)
	if len(attachments) == 0 {
		return html, 0
	}

	migratedCount := 0
	result := html

	for _, attachment := range attachments {
		if attachment.DownloadURL == "" {
			continue
		}

		// Create temp file for download
		tempDir := os.TempDir()
		tempFile := filepath.Join(tempDir, fmt.Sprintf("fizzy-migrate-attachment-%d-%s", attachment.Index, attachment.Filename))

		// Download the attachment
		err := sourceClient.DownloadFile(attachment.DownloadURL, tempFile)
		if err != nil {
			_ = os.Remove(tempFile)
			fmt.Fprintf(os.Stderr, "      Warning: Failed to download attachment '%s': %v\n", attachment.Filename, err)
			continue
		}

		// Upload to target account
		uploadResp, err := targetClient.UploadFile(tempFile)
		_ = os.Remove(tempFile) // Clean up temp file
		if err != nil {
			fmt.Fprintf(os.Stderr, "      Warning: Failed to upload attachment '%s': %v\n", attachment.Filename, err)
			continue
		}

		// Get the new SGID from upload response
		uploadData, ok := uploadResp.Data.(map[string]interface{})
		if !ok {
			fmt.Fprintf(os.Stderr, "      Warning: Invalid upload response for '%s'\n", attachment.Filename)
			continue
		}

		newSGID := getStringField(uploadData, "attachable_sgid")
		if newSGID == "" {
			// Try signed_id as fallback
			newSGID = getStringField(uploadData, "signed_id")
		}
		if newSGID == "" {
			fmt.Fprintf(os.Stderr, "      Warning: No SGID in upload response for '%s'\n", attachment.Filename)
			continue
		}

		// Replace the old SGID with the new one in the HTML
		if attachment.SGID != "" {
			result = strings.Replace(result, attachment.SGID, newSGID, 1)
			migratedCount++
		}
	}

	return result, migratedCount
}

func printDryRunSummary(boardName string, columns, cards []interface{}) {
	fmt.Fprintf(os.Stderr, "\n=== DRY RUN SUMMARY ===\n")
	fmt.Fprintf(os.Stderr, "Would migrate board: %s\n", boardName)
	fmt.Fprintf(os.Stderr, "Columns to create: %d\n", countRealColumns(columns))
	fmt.Fprintf(os.Stderr, "Cards to migrate: %d\n", len(cards))

	if migrateBoardIncludeComments {
		fmt.Fprintf(os.Stderr, "Comments: will be included\n")
	}
	if migrateBoardIncludeSteps {
		fmt.Fprintf(os.Stderr, "Steps: will be included\n")
	}
	if migrateBoardIncludeImages {
		fmt.Fprintf(os.Stderr, "Images: will be included\n")
	}

	fmt.Fprintf(os.Stderr, "\nNo changes were made.\n")
}

func printMigrationSummary(stats *migrationStats) {
	fmt.Fprintf(os.Stderr, "\n=== MIGRATION COMPLETE ===\n")
	fmt.Fprintf(os.Stderr, "Board created: %s (ID: %s)\n", stats.targetBoardName, stats.targetBoardID)
	fmt.Fprintf(os.Stderr, "Columns created: %d\n", stats.columnsCreated)
	fmt.Fprintf(os.Stderr, "Cards migrated: %d\n", stats.cardsCreated)
	fmt.Fprintf(os.Stderr, "Tags applied: %d\n", stats.tagsApplied)

	if migrateBoardIncludeComments {
		fmt.Fprintf(os.Stderr, "Comments created: %d\n", stats.commentsCreated)
	}
	if migrateBoardIncludeSteps {
		fmt.Fprintf(os.Stderr, "Steps created: %d\n", stats.stepsCreated)
	}
	if migrateBoardIncludeImages {
		fmt.Fprintf(os.Stderr, "Images migrated: %d\n", stats.imagesMigrated)
	}

	fmt.Fprintf(os.Stderr, "\nNote: Card creators and comment authors are now you (the migrating user).\n")
	fmt.Fprintf(os.Stderr, "      User assignments were not migrated - reassign as needed.\n")
}

func countRealColumns(columns []interface{}) int {
	count := 0
	for _, col := range columns {
		colMap, ok := col.(map[string]interface{})
		if !ok {
			continue
		}
		if kind, ok := colMap["kind"].(string); ok && kind != "real" {
			continue
		}
		if pseudo, ok := colMap["pseudo"].(bool); ok && pseudo {
			continue
		}
		count++
	}
	return count
}

// Helper functions for safe type assertions
func getStringField(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getIntField(m map[string]interface{}, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	if v, ok := m[key].(int); ok {
		return v
	}
	return 0
}

func getBoolField(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

func init() {
	rootCmd.AddCommand(migrateCmd)

	// Migrate board subcommand
	migrateBoardCmd.Flags().StringVar(&migrateBoardFrom, "from", "", "Source account slug (required)")
	migrateBoardCmd.Flags().StringVar(&migrateBoardTo, "to", "", "Target account slug (required)")
	migrateBoardCmd.Flags().BoolVar(&migrateBoardIncludeComments, "include-comments", false, "Also migrate card comments")
	migrateBoardCmd.Flags().BoolVar(&migrateBoardIncludeSteps, "include-steps", false, "Also migrate card steps (to-do items)")
	migrateBoardCmd.Flags().BoolVar(&migrateBoardIncludeImages, "include-images", false, "Also migrate card header images")
	migrateBoardCmd.Flags().BoolVar(&migrateBoardDryRun, "dry-run", false, "Show what would be migrated without making changes")
	migrateCmd.AddCommand(migrateBoardCmd)
}
