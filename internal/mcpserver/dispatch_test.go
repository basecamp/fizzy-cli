package mcpserver

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/basecamp/fizzy-cli/internal/mcpserver/catalog"
)

func operation(t *testing.T, domainKey, action string) *catalog.Operation {
	t.Helper()
	domains, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range domains {
		if d.Key != domainKey {
			continue
		}
		op, ok := d.Operation(action)
		if !ok {
			t.Fatalf("action %q not in domain %q", action, domainKey)
		}
		return op
	}
	t.Fatalf("domain %q not in catalog", domainKey)
	return nil
}

// params round-trips arguments through JSON so values arrive exactly as
// the MCP layer delivers them (numbers as float64, arrays as []any).
func params(t *testing.T, jsonText string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(jsonText), &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestBuildRequest(t *testing.T) {
	cases := []struct {
		name     string
		domain   string
		action   string
		params   string
		wantPath string
		wantBody string // JSON, "" for no body
	}{
		{
			name:   "no params",
			domain: "boards", action: "list_boards",
			params:   `{}`,
			wantPath: "/boards",
		},
		{
			name:   "path param substituted and escaped",
			domain: "cards", action: "get_card",
			params:   `{"card_number": "4/2"}`,
			wantPath: "/cards/4%2F2",
		},
		{
			name:   "numeric path param renders without exponent",
			domain: "cards", action: "get_card",
			params:   `{"card_number": 42}`,
			wantPath: "/cards/42",
		},
		{
			name:   "query params encoded rails style",
			domain: "cards", action: "list_cards",
			params:   `{"board_ids": ["b1", "b2"], "indexed_by": "closed", "page": 2}`,
			wantPath: "/cards?board_ids%5B%5D=b1&board_ids%5B%5D=b2&indexed_by=closed&page=2",
		},
		{
			name:   "body wrapped under body key",
			domain: "cards", action: "create_card",
			params:   `{"board_id": "b1", "title": "Add dark mode", "description": "Please"}`,
			wantPath: "/boards/b1/cards",
			wantBody: `{"card": {"title": "Add dark mode", "description": "Please"}}`,
		},
		{
			name:   "flat body without body key",
			domain: "cards", action: "triage_card",
			params:   `{"card_number": "42", "column_id": "c9"}`,
			wantPath: "/cards/42/triage",
			wantBody: `{"column_id": "c9"}`,
		},
		{
			name:   "bodiless write",
			domain: "cards", action: "close_card",
			params:   `{"card_number": "42"}`,
			wantPath: "/cards/42/closure",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path, body, err := buildRequest(operation(t, tc.domain, tc.action), params(t, tc.params))
			if err != nil {
				t.Fatal(err)
			}
			if path != tc.wantPath {
				t.Errorf("path = %q, want %q", path, tc.wantPath)
			}
			if tc.wantBody == "" {
				if body != nil {
					t.Errorf("body = %v, want none", body)
				}
			} else {
				got, err := json.Marshal(body)
				if err != nil {
					t.Fatal(err)
				}
				assertJSONEq(t, tc.wantBody, string(got))
			}
		})
	}
}

func TestBuildRequestErrors(t *testing.T) {
	cases := []struct {
		name   string
		domain string
		action string
		params string
		want   string
	}{
		{
			name:   "missing path param",
			domain: "cards", action: "get_card",
			params: `{}`,
			want:   `missing required param "card_number"`,
		},
		{
			name:   "missing required body field",
			domain: "cards", action: "triage_card",
			params: `{"card_number": "42"}`,
			want:   `missing required param "column_id"`,
		},
		{
			name:   "unknown param names what the action accepts",
			domain: "cards", action: "get_card",
			params: `{"card_number": "1", "bogus": true}`,
			want:   `unknown param "bogus" for action "get_card" (accepts: card_number)`,
		},
		{
			name:   "non-scalar path param",
			domain: "cards", action: "get_card",
			params: `{"card_number": {"nested": 1}}`,
			want:   "must be a string, number, or boolean",
		},
		{
			name:   "non-scalar query param",
			domain: "cards", action: "list_cards",
			params: `{"page": {"nested": 1}}`,
			want:   "must be a scalar",
		},
		{
			name:   "array of objects in query param",
			domain: "cards", action: "list_cards",
			params: `{"board_ids": [{"nested": 1}]}`,
			want:   "must be a scalar",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := buildRequest(operation(t, tc.domain, tc.action), params(t, tc.params))
			if err == nil {
				t.Fatal("buildRequest accepted the params")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want it to mention %q", err, tc.want)
			}
		})
	}
}

func TestNextPage(t *testing.T) {
	cases := []struct {
		name string
		link string
		want int
	}{
		{"absent", "", 0},
		{"next with page", `<https://app.fizzy.do/123/cards?page=2>; rel="next"`, 2},
		{"prev only", `<https://app.fizzy.do/123/cards?page=1>; rel="prev"`, 0},
		{"prev and next", `<https://app.fizzy.do/123/cards?page=1>; rel="prev", <https://app.fizzy.do/123/cards?page=3>; rel="next"`, 3},
		{"next without page", `<https://app.fizzy.do/123/cards>; rel="next"`, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := nextPage(tc.link); got != tc.want {
				t.Errorf("nextPage(%q) = %d, want %d", tc.link, got, tc.want)
			}
		})
	}
}

func assertJSONEq(t *testing.T, want, got string) {
	t.Helper()
	var w, g any
	if err := json.Unmarshal([]byte(want), &w); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(got), &g); err != nil {
		t.Fatal(err)
	}
	wc, _ := json.Marshal(w)
	gc, _ := json.Marshal(g)
	// Canonical re-marshal sorts map keys, so equal JSON compares equal.
	if string(wc) != string(gc) {
		t.Errorf("JSON = %s, want %s", got, want)
	}
}
