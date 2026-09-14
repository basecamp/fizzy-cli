package commands

import (
	"testing"

	"github.com/basecamp/fizzy-cli/internal/client"
)

func cardWithEmptyColumn(number int, extra map[string]any) map[string]any {
	card := map[string]any{
		"id":     "card-id",
		"number": float64(number),
		"title":  "Test Card",
		"status": "published",
		"column": map[string]any{"id": "", "name": "", "created_at": ""},
	}
	for k, v := range extra {
		card[k] = v
	}
	return card
}

func TestHydrateCardColumn(t *testing.T) {
	t.Run("hydrates published card without column as maybe", func(t *testing.T) {
		card := cardWithEmptyColumn(1, nil)
		hydrateCardColumn(card)

		column, ok := card["column"].(map[string]any)
		if !ok {
			t.Fatalf("expected column map, got %T", card["column"])
		}
		if column["id"] != "maybe" || column["name"] != "Maybe?" {
			t.Errorf("expected maybe pseudo column, got %v", column)
		}
		if column["kind"] != "triage" || column["pseudo"] != true {
			t.Errorf("expected pseudo triage marker, got %v", column)
		}
	})

	t.Run("hydrates closed card as done", func(t *testing.T) {
		card := cardWithEmptyColumn(2, map[string]any{"closed": true})
		hydrateCardColumn(card)

		column := card["column"].(map[string]any)
		if column["id"] != "done" || column["name"] != "Done" {
			t.Errorf("expected done pseudo column, got %v", column)
		}
	})

	t.Run("hydrates postponed card as not-now", func(t *testing.T) {
		card := cardWithEmptyColumn(3, map[string]any{"postponed": true})
		hydrateCardColumn(card)

		column := card["column"].(map[string]any)
		if column["id"] != "not-now" || column["name"] != "Not Now" {
			t.Errorf("expected not-now pseudo column, got %v", column)
		}
	})

	t.Run("prefers closed over other flags", func(t *testing.T) {
		card := cardWithEmptyColumn(4, map[string]any{"closed": true, "postponed": true})
		hydrateCardColumn(card)

		column := card["column"].(map[string]any)
		if column["id"] != "done" {
			t.Errorf("expected done pseudo column, got %v", column)
		}
	})

	t.Run("leaves drafts untouched", func(t *testing.T) {
		card := cardWithEmptyColumn(5, map[string]any{"status": "drafted"})
		hydrateCardColumn(card)

		column := card["column"].(map[string]any)
		if column["id"] != "" {
			t.Errorf("expected draft column to remain empty, got %v", column)
		}
	})

	t.Run("leaves cards with a real column untouched", func(t *testing.T) {
		card := cardWithEmptyColumn(6, map[string]any{
			"column": map[string]any{"id": "col-123", "name": "Development"},
		})
		hydrateCardColumn(card)

		column := card["column"].(map[string]any)
		if column["id"] != "col-123" || column["name"] != "Development" {
			t.Errorf("expected real column to remain untouched, got %v", column)
		}
	})

	t.Run("leaves unknown payloads untouched", func(t *testing.T) {
		card := map[string]any{"id": "x", "column": map[string]any{"id": ""}}
		hydrateCardColumn(card)

		column := card["column"].(map[string]any)
		if column["id"] != "" {
			t.Errorf("expected column to remain empty, got %v", column)
		}
	})
}

func TestHydrateCardColumnsList(t *testing.T) {
	items := []map[string]any{
		cardWithEmptyColumn(1, nil),
		cardWithEmptyColumn(2, map[string]any{"closed": true}),
		cardWithEmptyColumn(3, map[string]any{"postponed": true}),
	}
	hydrateCardColumns(items)

	expected := []string{"maybe", "done", "not-now"}
	for i, want := range expected {
		column := items[i]["column"].(map[string]any)
		if column["id"] != want {
			t.Errorf("card %d: expected column id %q, got %v", i+1, want, column["id"])
		}
	}

	single := cardWithEmptyColumn(7, nil)
	hydrateCardColumns(single)
	if single["column"].(map[string]any)["id"] != "maybe" {
		t.Errorf("expected single card hydration, got %v", single["column"])
	}
}

func TestCardShowHydratesPseudoColumn(t *testing.T) {
	mock := NewMockClient()
	mock.GetResponse = &client.APIResponse{
		StatusCode: 200,
		Data:       cardWithEmptyColumn(216, nil),
	}

	result := SetTestModeWithSDK(mock)
	SetTestConfig("token", "account", "https://api.example.com")
	defer resetTest()

	err := cardShowCmd.RunE(cardShowCmd, []string{"216"})
	assertExitCode(t, err, 0)

	data, ok := result.Response.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected map response data, got %T", result.Response.Data)
	}
	column, ok := data["column"].(map[string]any)
	if !ok {
		t.Fatalf("expected column map, got %T", data["column"])
	}
	if column["id"] != "maybe" || column["name"] != "Maybe?" {
		t.Errorf("expected hydrated maybe column, got %v", column)
	}
}

func TestCardListHydratesPseudoColumns(t *testing.T) {
	mock := NewMockClient()
	mock.GetWithPaginationResponse = &client.APIResponse{
		StatusCode: 200,
		Data: []any{
			cardWithEmptyColumn(1, nil),
			cardWithEmptyColumn(2, map[string]any{"closed": true}),
			cardWithEmptyColumn(3, map[string]any{"postponed": true}),
		},
	}

	result := SetTestModeWithSDK(mock)
	SetTestConfig("token", "account", "https://api.example.com")
	defer resetTest()

	err := cardListCmd.RunE(cardListCmd, []string{})
	assertExitCode(t, err, 0)

	arr, ok := result.Response.Data.([]any)
	if !ok {
		t.Fatalf("expected array response data, got %T", result.Response.Data)
	}
	expected := []string{"maybe", "done", "not-now"}
	for i, want := range expected {
		card := arr[i].(map[string]any)
		column := card["column"].(map[string]any)
		if column["id"] != want {
			t.Errorf("card %d: expected column id %q, got %v", i+1, want, column["id"])
		}
	}
}
