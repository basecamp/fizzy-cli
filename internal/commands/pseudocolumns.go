package commands

import "strings"

type pseudoColumn struct {
	ID   string
	Name string
	Kind string
}

var (
	// "Not Now" contains postponed cards (indexed_by=not_now)
	pseudoColumnNotNow = pseudoColumn{ID: "not-now", Name: "Not Now", Kind: "not_now"}
	// "Maybe?" contains triage/backlog cards. Kind remains "triage" for triage endpoints/aliases;
	// card listing maps this pseudo-column to indexed_by=maybe server-side.
	pseudoColumnMaybe = pseudoColumn{ID: "maybe", Name: "Maybe?", Kind: "triage"}
	// "Done" contains closed cards (indexed_by=closed)
	pseudoColumnDone = pseudoColumn{ID: "done", Name: "Done", Kind: "closed"}
)

func pseudoColumnObject(c pseudoColumn) map[string]any {
	return map[string]any{
		"id":     c.ID,
		"name":   c.Name,
		"kind":   c.Kind,
		"pseudo": true,
	}
}

func parsePseudoColumnID(id string) (pseudoColumn, bool) {
	switch strings.ToLower(strings.TrimSpace(id)) {
	case "not-now", "not_now", "notnow", "not-yet", "not_yet", "notyet":
		return pseudoColumnNotNow, true
	case "maybe", "maybe?", "triage":
		return pseudoColumnMaybe, true
	case "done", "closed", "close":
		return pseudoColumnDone, true
	default:
		return pseudoColumn{}, false
	}
}

// inferPseudoColumn determines which pseudo column an open card without a real
// column belongs to (cards in Not Now and Maybe have no real column; closed
// cards are handled by hydrateCardColumn). Postponing a card clears its column,
// and closing it destroys its not_now record. Drafts belong to no lane, so they
// are left untouched.
func inferPseudoColumn(card map[string]any) (pseudoColumn, bool) {
	if jsonBool(card["postponed"]) {
		return pseudoColumnNotNow, true
	}
	if status, _ := card["status"].(string); status == "published" {
		return pseudoColumnMaybe, true
	}
	return pseudoColumn{}, false
}

// hydrateCardColumns applies hydrateCardColumn to a card or a list of cards.
func hydrateCardColumns(items any) {
	switch d := items.(type) {
	case []map[string]any:
		for _, m := range d {
			hydrateCardColumn(m)
		}
	case map[string]any:
		hydrateCardColumn(d)
	}
}

// hydrateCardColumn sets the column a card is shown in so consumers always see
// a usable column id/name. Precedence:
//
//  1. A closed card is in Done, whatever its column. Closing a card doesn't
//     clear its column, so the API still returns the column it was closed
//     from, while Fizzy shows the card in Done.
//  2. A card with a real column keeps it.
//  3. A card without one is inferred by inferPseudoColumn: Not Now when
//     postponed, Maybe? when published; drafts are left untouched.
func hydrateCardColumn(card map[string]any) {
	if jsonBool(card["closed"]) {
		card["column"] = pseudoColumnObject(pseudoColumnDone)
		return
	}
	column, _ := card["column"].(map[string]any)
	if id, _ := column["id"].(string); id != "" {
		return
	}
	pseudo, ok := inferPseudoColumn(card)
	if !ok {
		return
	}
	card["column"] = pseudoColumnObject(pseudo)
}

func jsonBool(v any) bool {
	b, ok := v.(bool)
	return ok && b
}
