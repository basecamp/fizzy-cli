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
	// "Maybe?" contains triage/backlog cards (indexed_by=maybe)
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
