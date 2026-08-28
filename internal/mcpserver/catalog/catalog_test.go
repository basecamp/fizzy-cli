// These tests pin the duplicated catalog: the curation validates cleanly,
// the invariants the dispatcher relies on hold, and the full rendered
// surface is snapshotted so any change — including a sync from the sibling
// in fizzy-mcp-server — shows its effect as a reviewable diff. The generic
// gateway machinery is covered by the toolkit's own suite in
// github.com/basecamp/mcp/gateway.
package catalog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/basecamp/mcp/mcptest"
)

func load(t *testing.T) []*Domain {
	t.Helper()
	domains, err := Load()
	if err != nil {
		t.Fatalf("catalog must validate cleanly: %v", err)
	}
	return domains
}

func TestCatalogShape(t *testing.T) {
	domains := load(t)

	keys := make([]string, 0, len(domains))
	total := 0
	for _, d := range domains {
		keys = append(keys, d.Key)
		total += len(d.Operations)
		if d.Tool != "fizzy_"+d.Key {
			t.Errorf("domain %q tool = %q", d.Key, d.Tool)
		}
		if d.Blurb == "" {
			t.Errorf("domain %q has no blurb", d.Key)
		}
	}
	want := []string{"identity", "boards", "columns", "cards", "comments", "steps", "tags", "users"}
	if got, expected := strings.Join(keys, ","), strings.Join(want, ","); got != expected {
		t.Errorf("domains = %s, want %s", got, expected)
	}
	if total != 45 {
		t.Errorf("operations = %d, want 45 (the count is deliberate; update alongside the snapshot)", total)
	}
}

func TestOnlyIdentityIsUnscoped(t *testing.T) {
	for _, d := range load(t) {
		for _, op := range d.Operations {
			if op.Unscoped && op.Action != "get_identity" {
				t.Errorf("%s/%s is unscoped; unscoped operations are the deliberate exception", d.Key, op.Action)
			}
		}
	}
}

func TestPaginatedOperationsTakePage(t *testing.T) {
	for _, d := range load(t) {
		for _, op := range d.Operations {
			hasPage := false
			for _, p := range op.Params {
				if p.Name == "page" && p.In == "query" {
					hasPage = true
				}
			}
			if op.Paginated != hasPage {
				t.Errorf("%s/%s: paginated and the page param travel together", d.Key, op.Action)
			}
		}
	}
}

func TestBodyRequiredFieldsExist(t *testing.T) {
	for _, d := range load(t) {
		for _, op := range d.Operations {
			if op.Body == nil {
				continue
			}
			props, ok := op.Body["properties"].(map[string]any)
			if !ok {
				t.Errorf("%s/%s: body must declare properties", d.Key, op.Action)
				continue
			}
			if required, ok := op.Body["required"].([]any); ok {
				for _, name := range required {
					field, _ := name.(string)
					if _, present := props[field]; !present {
						t.Errorf("%s/%s: required field %q must be a property", d.Key, op.Action, field)
					}
				}
			}
		}
	}
}

func TestValidateRejectsCurationMistakes(t *testing.T) {
	cases := []struct {
		name   string
		domain *Domain
		want   string
	}{
		{
			"unsorted operations",
			&Domain{Key: "x", Tool: "fizzy_x", Operations: []*Operation{
				{Action: "b", Method: "GET", Path: "/b", ReadOnly: true},
				{Action: "a", Method: "GET", Path: "/a", ReadOnly: true},
			}},
			"sorted",
		},
		{
			"reserved describe action",
			&Domain{Key: "x", Tool: "fizzy_x", Operations: []*Operation{
				{Action: "describe", Method: "GET", Path: "/x", ReadOnly: true},
			}},
			"reserved",
		},
		{
			"write marked read-only",
			&Domain{Key: "x", Tool: "fizzy_x", Operations: []*Operation{
				{Action: "a", Method: "POST", Path: "/a", ReadOnly: true},
			}},
			"readonly must mirror the method",
		},
		{
			"path token without param",
			&Domain{Key: "x", Tool: "fizzy_x", Operations: []*Operation{
				{Action: "a", Method: "GET", Path: "/a/{id}", ReadOnly: true},
			}},
			"no matching path param",
		},
		{
			"path param not in path",
			&Domain{Key: "x", Tool: "fizzy_x", Operations: []*Operation{
				{Action: "a", Method: "GET", Path: "/a", ReadOnly: true,
					Params: []Param{{Name: "id", In: "path", Required: true, Schema: str("x")}}},
			}},
			"does not appear in path",
		},
		{
			"body_key without body",
			&Domain{Key: "x", Tool: "fizzy_x", Operations: []*Operation{
				{Action: "a", Method: "POST", Path: "/a", BodyKey: "thing"},
			}},
			"without a body schema",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validate(tc.domain)
			if err == nil {
				t.Fatal("validate accepted the mistake")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want it to mention %q", err, tc.want)
			}
		})
	}
}

// TestProvenanceRecordsTheSibling keeps the duplication honest: the
// catalog is synced from fizzy-mcp-server by scripts/sync-mcp-catalog.sh,
// and the provenance names the source commit so drift is traceable.
func TestProvenanceRecordsTheSibling(t *testing.T) {
	data, err := os.ReadFile("PROVENANCE.json")
	if err != nil {
		t.Fatalf("PROVENANCE.json: %v", err)
	}
	var provenance struct {
		Source string `json:"source"`
		Commit string `json:"commit"`
	}
	if err := json.Unmarshal(data, &provenance); err != nil {
		t.Fatalf("PROVENANCE.json: %v", err)
	}
	if provenance.Source != "github.com/basecamp/fizzy-mcp-server" {
		t.Errorf("provenance source = %q", provenance.Source)
	}
	if provenance.Commit == "" {
		t.Error("provenance commit is empty")
	}
}

// TestSnapshot renders the entire served surface — tool names,
// descriptions, input schemas, and every action's describe payload — so a
// catalog change shows its full effect as a reviewable diff. Regenerate
// with `go test ./internal/mcpserver/catalog -update`.
func TestSnapshot(t *testing.T) {
	var b strings.Builder
	for _, d := range load(t) {
		b.WriteString("==== TOOL " + d.ToolName() + " ====\n")
		b.WriteString(d.Description())
		b.WriteString("---- input schema ----\n")
		writeJSON(t, &b, d.InputSchema())
		for _, op := range d.Operations {
			b.WriteString("---- describe " + op.Action + " ----\n")
			payload, err := d.Describe(op.Action)
			if err != nil {
				t.Fatal(err)
			}
			writeJSON(t, &b, payload)
		}
	}
	mcptest.Snapshot(t, filepath.Join("testdata", "catalog_snapshot.txt"), []byte(b.String()))
}

func writeJSON(t *testing.T, b *strings.Builder, v any) {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	b.Write(data)
	b.WriteString("\n")
}
