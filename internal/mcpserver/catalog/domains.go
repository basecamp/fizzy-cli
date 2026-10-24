package catalog

// Schema shorthands for the hand-written specs below.
func str(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

func strEnum(desc string, values ...any) map[string]any {
	return map[string]any{"type": "string", "description": desc, "enum": values}
}

func boolean(desc string) map[string]any {
	return map[string]any{"type": "boolean", "description": desc}
}

func integer(desc string) map[string]any {
	return map[string]any{"type": "integer", "description": desc}
}

func strArray(desc string) map[string]any {
	return map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": desc}
}

func object(properties map[string]any, required ...any) map[string]any {
	schema := map[string]any{"type": "object", "properties": properties}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func pathParam(name, desc string) Param {
	return Param{Name: name, In: "path", Required: true, Description: desc, Schema: str(desc)}
}

func queryParam(name, desc string, schema map[string]any) Param {
	return Param{Name: name, In: "query", Description: desc, Schema: schema}
}

func pageParam() Param {
	return queryParam("page", "Page number; responses include next_page when more results exist.", integer("Page number"))
}

var cardNumber = pathParam("card_number", "The card number (the number in the card's URL, not its ID)")

// Domains is the hand-written v1 catalog, in tool display order. Each
// operation mirrors one endpoint in Fizzy's public API docs (docs/api in
// the fizzy repo); operations within a domain are sorted by action name.
var Domains = []*Domain{
	{
		Key:   "identity",
		Tool:  "fizzy_identity",
		Blurb: "Who you are on Fizzy: the accounts your token can reach and your user in each. Call this first when the account slug is unknown.",
		Operations: []*Operation{
			{
				Action: "get_identity", Method: "GET", Path: "/my/identity", ReadOnly: true, Unscoped: true,
				Summary: "List the accounts the token can access, with your user record in each",
				Doc:     "Each account carries a slug naming it in API paths. Account-scoped actions run against the account the server is configured for; this action shows every account the token can reach.",
			},
		},
	},
	{
		Key:   "boards",
		Tool:  "fizzy_boards",
		Blurb: "Fizzy boards: the kanban boards cards live on, who can access them, and public publication.",
		Operations: []*Operation{
			{
				Action: "create_board", Method: "POST", Path: "/boards", BodyKey: "board",
				Summary: "Create a board",
				Body: object(map[string]any{
					"name":                         str("The name of the board"),
					"all_access":                   boolean("Whether any user in the account can access this board (default true)"),
					"auto_postpone_period_in_days": integer("Days of inactivity before cards are automatically postponed"),
					"public_description":           str("Rich text description shown on the public board page"),
				}, "name"),
			},
			{
				Action: "delete_board", Method: "DELETE", Path: "/boards/{board_id}",
				Summary: "Delete a board (board administrators only)",
				Params:  []Param{pathParam("board_id", "The board ID")},
			},
			{
				Action: "get_board", Method: "GET", Path: "/boards/{board_id}", ReadOnly: true,
				Summary: "Get one board",
				Params:  []Param{pathParam("board_id", "The board ID")},
			},
			{
				Action: "list_accesses", Method: "GET", Path: "/boards/{board_id}/accesses", ReadOnly: true, Paginated: true,
				Summary: "List account users with their access and involvement for a board",
				Params:  []Param{pathParam("board_id", "The board ID"), pageParam()},
			},
			{
				Action: "list_boards", Method: "GET", Path: "/boards", ReadOnly: true, Paginated: true,
				Summary: "List the boards you have access to",
				Params:  []Param{pageParam()},
			},
			{
				Action: "publish_board", Method: "POST", Path: "/boards/{board_id}/publication",
				Summary: "Publish a board to a shareable public link (administrators only)",
				Params:  []Param{pathParam("board_id", "The board ID")},
			},
			{
				Action: "unpublish_board", Method: "DELETE", Path: "/boards/{board_id}/publication",
				Summary: "Unpublish a board, removing public access (administrators only)",
				Params:  []Param{pathParam("board_id", "The board ID")},
			},
			{
				Action: "update_board", Method: "PUT", Path: "/boards/{board_id}", BodyKey: "board",
				Summary: "Update a board (administrators only)",
				Params:  []Param{pathParam("board_id", "The board ID")},
				Body: object(map[string]any{
					"name":                         str("The name of the board"),
					"all_access":                   boolean("Whether any user in the account can access this board"),
					"auto_postpone_period_in_days": integer("Days of inactivity before cards are automatically postponed"),
					"public_description":           str("Rich text description shown on the public board page"),
					"user_ids":                     strArray("All user IDs who should have access (only when all_access is false; replaces the whole list)"),
				}),
			},
		},
	},
	{
		Key:   "columns",
		Tool:  "fizzy_columns",
		Blurb: "Workflow columns on a Fizzy board. Cards in Maybe?, Not Now, or Done live outside columns; move cards between columns with the cards tool's triage_card.",
		Operations: []*Operation{
			{
				Action: "create_column", Method: "POST", Path: "/boards/{board_id}/columns", BodyKey: "column",
				Summary: "Create a column on a board",
				Params:  []Param{pathParam("board_id", "The board ID")},
				Body: object(map[string]any{
					"name":  str("The name of the column"),
					"color": str("Column color, e.g. var(--color-card-default) (Blue), var(--color-card-1) (Gray) through var(--color-card-8) (Pink)"),
				}, "name"),
			},
			{
				Action: "delete_column", Method: "DELETE", Path: "/boards/{board_id}/columns/{column_id}",
				Summary: "Delete a column",
				Params:  []Param{pathParam("board_id", "The board ID"), pathParam("column_id", "The column ID")},
			},
			{
				Action: "get_column", Method: "GET", Path: "/boards/{board_id}/columns/{column_id}", ReadOnly: true,
				Summary: "Get one column",
				Params:  []Param{pathParam("board_id", "The board ID"), pathParam("column_id", "The column ID")},
			},
			{
				Action: "list_cards", Method: "GET", Path: "/boards/{board_id}/columns/{column_id}/cards", ReadOnly: true, Paginated: true,
				Summary: "List the open cards in a column",
				Params:  []Param{pathParam("board_id", "The board ID"), pathParam("column_id", "The column ID"), pageParam()},
			},
			{
				Action: "list_columns", Method: "GET", Path: "/boards/{board_id}/columns", ReadOnly: true,
				Summary: "List a board's columns",
				Params:  []Param{pathParam("board_id", "The board ID")},
			},
			{
				Action: "update_column", Method: "PUT", Path: "/boards/{board_id}/columns/{column_id}", BodyKey: "column",
				Summary: "Rename or recolor a column",
				Params:  []Param{pathParam("board_id", "The board ID"), pathParam("column_id", "The column ID")},
				Body: object(map[string]any{
					"name":  str("The name of the column"),
					"color": str("The column color"),
				}),
			},
		},
	},
	{
		Key:   "cards",
		Tool:  "fizzy_cards",
		Blurb: "Fizzy cards: create, find, and update cards; close and reopen; move between columns (triage), boards, and Not Now; tag, assign, watch, and mark golden.",
		Operations: []*Operation{
			{
				Action: "close_card", Method: "POST", Path: "/cards/{card_number}/closure",
				Summary: "Close a card (move it to Done)",
				Params:  []Param{cardNumber},
			},
			{
				Action: "create_card", Method: "POST", Path: "/boards/{board_id}/cards", BodyKey: "card",
				Summary: "Create a card on a board (new cards start in Maybe? triage)",
				Params:  []Param{pathParam("board_id", "The board ID")},
				Body: object(map[string]any{
					"title":       str("The title of the card"),
					"description": str("Rich text description of the card"),
					"status":      strEnum("Initial status (default published)", "published", "drafted"),
					"tag_ids":     strArray("Tag IDs to apply to the card"),
				}, "title"),
			},
			{
				Action: "delete_card", Method: "DELETE", Path: "/cards/{card_number}",
				Summary: "Delete a card (creator or board administrators only)",
				Params:  []Param{cardNumber},
			},
			{
				Action: "get_card", Method: "GET", Path: "/cards/{card_number}", ReadOnly: true,
				Summary: "Get one card with its board, column, assignees, tags, and steps",
				Params:  []Param{cardNumber},
			},
			{
				Action: "list_cards", Method: "GET", Path: "/cards", ReadOnly: true, Paginated: true,
				Summary: "List cards you have access to, filtered by board, column, tag, assignee, state, or search terms",
				Params: []Param{
					queryParam("assignee_ids", "Filter by assignee user ID(s)", strArray("Assignee user IDs")),
					queryParam("assignment_status", "Filter by assignment status", strEnum("Assignment status", "unassigned")),
					queryParam("board_ids", "Filter by board ID(s)", strArray("Board IDs")),
					queryParam("card_ids", "Filter to specific card ID(s)", strArray("Card IDs")),
					queryParam("closer_ids", "Filter by user ID(s) who closed the cards", strArray("Closer user IDs")),
					queryParam("closure", "Filter by closure date", strEnum("Closure date range", "today", "yesterday", "thisweek", "lastweek", "thismonth", "lastmonth", "thisyear", "lastyear")),
					queryParam("column_ids", "Filter by workflow column ID(s); repeated values are ORed", strArray("Column IDs")),
					queryParam("creation", "Filter by creation date", strEnum("Creation date range", "today", "yesterday", "thisweek", "lastweek", "thismonth", "lastmonth", "thisyear", "lastyear")),
					queryParam("creator_ids", "Filter by card creator ID(s)", strArray("Creator user IDs")),
					queryParam("indexed_by", "Filter by card state", strEnum("Card state", "all", "maybe", "closed", "not_now", "stalled", "postponing_soon", "golden")),
					pageParam(),
					queryParam("sorted_by", "Sort order", strEnum("Sort order", "latest", "newest", "oldest")),
					queryParam("tag_ids", "Filter by tag ID(s)", strArray("Tag IDs")),
					queryParam("terms", "Search terms to filter cards", strArray("Search terms")),
				},
			},
			{
				Action: "mark_golden", Method: "POST", Path: "/cards/{card_number}/goldness",
				Summary: "Mark a card as golden",
				Params:  []Param{cardNumber},
			},
			{
				Action: "move_card", Method: "PUT", Path: "/cards/{card_number}/board",
				Summary: "Move a card to a different board",
				Params:  []Param{cardNumber},
				Body: object(map[string]any{
					"board_id": str("The ID of the board to move the card to"),
				}, "board_id"),
			},
			{
				Action: "postpone_card", Method: "POST", Path: "/cards/{card_number}/not_now",
				Summary: "Move a card to Not Now",
				Params:  []Param{cardNumber},
			},
			{
				Action: "reopen_card", Method: "DELETE", Path: "/cards/{card_number}/closure",
				Summary: "Reopen a closed card",
				Params:  []Param{cardNumber},
			},
			{
				Action: "toggle_assignment", Method: "POST", Path: "/cards/{card_number}/assignments",
				Summary: "Toggle assignment of a user to/from a card",
				Params:  []Param{cardNumber},
				Body: object(map[string]any{
					"assignee_id": str("The ID of the user to assign or unassign"),
				}, "assignee_id"),
			},
			{
				Action: "toggle_tag", Method: "POST", Path: "/cards/{card_number}/taggings",
				Summary: "Toggle a tag on or off for a card, creating the tag if needed",
				Params:  []Param{cardNumber},
				Body: object(map[string]any{
					"tag_title": str("The title of the tag (leading # is stripped)"),
				}, "tag_title"),
			},
			{
				Action: "triage_card", Method: "POST", Path: "/cards/{card_number}/triage",
				Summary: "Move a card into a workflow column",
				Params:  []Param{cardNumber},
				Body: object(map[string]any{
					"column_id": str("The ID of the column to move the card into"),
				}, "column_id"),
			},
			{
				Action: "unmark_golden", Method: "DELETE", Path: "/cards/{card_number}/goldness",
				Summary: "Remove golden status from a card",
				Params:  []Param{cardNumber},
			},
			{
				Action: "untriage_card", Method: "DELETE", Path: "/cards/{card_number}/triage",
				Summary: "Send a card back to triage (Maybe?)",
				Params:  []Param{cardNumber},
			},
			{
				Action: "unwatch_card", Method: "DELETE", Path: "/cards/{card_number}/watch",
				Summary: "Unsubscribe from notifications for a card",
				Params:  []Param{cardNumber},
			},
			{
				Action: "update_card", Method: "PUT", Path: "/cards/{card_number}", BodyKey: "card",
				Summary: "Update a card's title, description, status, or tags",
				Params:  []Param{cardNumber},
				Body: object(map[string]any{
					"title":       str("The title of the card"),
					"description": str("Rich text description of the card"),
					"status":      strEnum("Card status", "drafted", "published"),
					"tag_ids":     strArray("Tag IDs to apply to the card"),
				}),
			},
			{
				Action: "watch_card", Method: "POST", Path: "/cards/{card_number}/watch",
				Summary: "Subscribe to notifications for a card",
				Params:  []Param{cardNumber},
			},
		},
	},
	{
		Key:   "comments",
		Tool:  "fizzy_comments",
		Blurb: "Comments on Fizzy cards, chronological. Bodies support rich text.",
		Operations: []*Operation{
			{
				Action: "create_comment", Method: "POST", Path: "/cards/{card_number}/comments", BodyKey: "comment",
				Summary: "Comment on a card",
				Params:  []Param{cardNumber},
				Body: object(map[string]any{
					"body": str("The comment body (supports rich text)"),
				}, "body"),
			},
			{
				Action: "delete_comment", Method: "DELETE", Path: "/cards/{card_number}/comments/{comment_id}",
				Summary: "Delete a comment (comment creator only)",
				Params:  []Param{cardNumber, pathParam("comment_id", "The comment ID")},
			},
			{
				Action: "get_comment", Method: "GET", Path: "/cards/{card_number}/comments/{comment_id}", ReadOnly: true,
				Summary: "Get one comment",
				Params:  []Param{cardNumber, pathParam("comment_id", "The comment ID")},
			},
			{
				Action: "list_comments", Method: "GET", Path: "/cards/{card_number}/comments", ReadOnly: true, Paginated: true,
				Summary: "List a card's comments, oldest first",
				Params:  []Param{cardNumber, pageParam()},
			},
			{
				Action: "update_comment", Method: "PUT", Path: "/cards/{card_number}/comments/{comment_id}", BodyKey: "comment",
				Summary: "Update a comment (comment creator only)",
				Params:  []Param{cardNumber, pathParam("comment_id", "The comment ID")},
				Body: object(map[string]any{
					"body": str("The updated comment body"),
				}, "body"),
			},
		},
	},
	{
		Key:   "steps",
		Tool:  "fizzy_steps",
		Blurb: "Steps: the checklist items on a Fizzy card.",
		Operations: []*Operation{
			{
				Action: "create_step", Method: "POST", Path: "/cards/{card_number}/steps", BodyKey: "step",
				Summary: "Add a step to a card",
				Params:  []Param{cardNumber},
				Body: object(map[string]any{
					"content":   str("The step text"),
					"completed": boolean("Whether the step is completed (default false)"),
				}, "content"),
			},
			{
				Action: "delete_step", Method: "DELETE", Path: "/cards/{card_number}/steps/{step_id}",
				Summary: "Delete a step",
				Params:  []Param{cardNumber, pathParam("step_id", "The step ID")},
			},
			{
				Action: "get_step", Method: "GET", Path: "/cards/{card_number}/steps/{step_id}", ReadOnly: true,
				Summary: "Get one step",
				Params:  []Param{cardNumber, pathParam("step_id", "The step ID")},
			},
			{
				Action: "list_steps", Method: "GET", Path: "/cards/{card_number}/steps", ReadOnly: true,
				Summary: "List a card's steps",
				Params:  []Param{cardNumber},
			},
			{
				Action: "update_step", Method: "PUT", Path: "/cards/{card_number}/steps/{step_id}", BodyKey: "step",
				Summary: "Edit a step or toggle its completion",
				Params:  []Param{cardNumber, pathParam("step_id", "The step ID")},
				Body: object(map[string]any{
					"content":   str("The step text"),
					"completed": boolean("Whether the step is completed"),
				}),
			},
		},
	},
	{
		Key:   "tags",
		Tool:  "fizzy_tags",
		Blurb: "Tags: the labels applied to cards, account-wide. Apply or remove them with the cards tool's toggle_tag.",
		Operations: []*Operation{
			{
				Action: "list_tags", Method: "GET", Path: "/tags", ReadOnly: true, Paginated: true,
				Summary: "List the account's tags, alphabetically",
				Params:  []Param{pageParam()},
			},
		},
	},
	{
		Key:   "users",
		Tool:  "fizzy_users",
		Blurb: "Users: the people in the Fizzy account. Assign them to cards with the cards tool's toggle_assignment.",
		Operations: []*Operation{
			{
				Action: "get_user", Method: "GET", Path: "/users/{user_id}", ReadOnly: true,
				Summary: "Get one user",
				Params:  []Param{pathParam("user_id", "The user ID")},
			},
			{
				Action: "list_users", Method: "GET", Path: "/users", ReadOnly: true, Paginated: true,
				Summary: "List the account's active users",
				Params:  []Param{pageParam()},
			},
		},
	},
}
