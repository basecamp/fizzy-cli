// Package catalog hand-writes the Fizzy MCP tool catalog.
//
// Where hey-mcp-server derives its catalog from hey-sdk's model exports,
// Fizzy has no SDK model to generate from yet — so this package follows
// basecamp-mcp-server's path instead: curated, hand-maintained domain
// specs, mirroring Fizzy's public API docs (docs/api in the fizzy repo).
// The rendering contract — tool description, input schema, the describe
// payload — matches what the toolkit's catalog package generates, so the
// wire surface stays consistent across the three servers, and generation
// can replace this file the day a fizzy-sdk ships model exports.
package catalog

import (
	"fmt"
	"sort"
	"strings"

	"github.com/basecamp/mcp/gateway"
)

// Param is one path or query parameter of an operation.
type Param struct {
	Name        string         `json:"name"`
	In          string         `json:"in"` // "path" or "query"
	Required    bool           `json:"required,omitempty"`
	Description string         `json:"description,omitempty"`
	Schema      map[string]any `json:"schema"`
}

// Operation is one Fizzy API endpoint exposed as a gateway action:
// everything a gateway tool needs to list, describe, and dispatch it.
type Operation struct {
	Action   string `json:"action"`
	Method   string `json:"method"`
	Path     string `json:"path"` // account-scoped unless Unscoped; {name} tokens match path Params
	Summary  string `json:"summary"`
	Doc      string `json:"doc,omitempty"`
	ReadOnly bool   `json:"readonly"`
	// Paginated marks endpoints that page via the Link header; responses
	// carry next_page when more results exist, fetched by passing page.
	Paginated bool    `json:"paginated,omitempty"`
	Params    []Param `json:"params,omitempty"`
	// Body is the JSON Schema of the request body's properties. The handler
	// nests supplied fields under BodyKey when set ({"card": {...}}).
	Body map[string]any `json:"body,omitempty"`
	// BodyKey is the wrapper key Rails wraps parameters under, "" for flat.
	BodyKey string `json:"body_key,omitempty"`
	// Unscoped marks the few endpoints outside the /:account_slug prefix.
	Unscoped bool `json:"unscoped,omitempty"`
}

// Domain is one gateway tool: a curated group of operations exposed as a
// single MCP tool with action dispatch.
type Domain struct {
	Key        string       // short name, e.g. "cards"
	Tool       string       // MCP tool name, e.g. "fizzy_cards"
	Blurb      string       // first line of the tool description
	Operations []*Operation // sorted by action name
}

var _ gateway.Domain = (*Domain)(nil)

// Name returns the short domain key, e.g. "cards".
func (d *Domain) Name() string { return d.Key }

// ToolName returns the MCP tool name, e.g. "fizzy_cards".
func (d *Domain) ToolName() string { return d.Tool }

// Find returns the dispatch surface of the operation registered under the
// given action name.
func (d *Domain) Find(action string) (gateway.Operation, bool) {
	op, ok := d.Operation(action)
	if !ok {
		return gateway.Operation{}, false
	}
	return gateway.Operation{Action: op.Action, ReadOnly: op.ReadOnly}, true
}

// FilterReadOnly returns a copy of the domain containing only read-only
// operations, reporting false when none remain.
func (d *Domain) FilterReadOnly() (gateway.Domain, bool) {
	filtered := &Domain{Key: d.Key, Tool: d.Tool, Blurb: d.Blurb}
	for _, op := range d.Operations {
		if op.ReadOnly {
			filtered.Operations = append(filtered.Operations, op)
		}
	}
	if len(filtered.Operations) == 0 {
		return nil, false
	}
	return filtered, true
}

// Operation returns the operation registered under the given action name.
func (d *Domain) Operation(action string) (*Operation, bool) {
	for _, op := range d.Operations {
		if op.Action == action {
			return op, true
		}
	}
	return nil, false
}

// AllReadOnly reports whether every operation in the domain is read-only.
func (d *Domain) AllReadOnly() bool {
	for _, op := range d.Operations {
		if !op.ReadOnly {
			return false
		}
	}
	return true
}

// Description renders the tool description: the domain blurb, the gateway
// calling convention, and a one-line summary per action. Matches the
// toolkit catalog's generated rendering.
func (d *Domain) Description() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", d.Blurb)
	b.WriteString("Gateway tool: call with {\"action\": \"...\", \"params\": {...}}.\n")
	fmt.Fprintf(&b, "Call {\"action\": %q, \"params\": {\"action\": \"NAME\"}} for an action's full parameter schema.\n\n", gateway.DescribeAction)
	b.WriteString("ACTIONS (RO = read-only):\n")
	for _, op := range d.Operations {
		var notes []string
		if op.ReadOnly {
			notes = append(notes, "RO")
		}
		if op.Paginated {
			notes = append(notes, "paginated")
		}
		suffix := ""
		if len(notes) > 0 {
			suffix = " (" + strings.Join(notes, ", ") + ")"
		}
		fmt.Fprintf(&b, "- %s%s: %s\n", op.Action, suffix, op.Summary)
	}
	return b.String()
}

// InputSchema renders the JSON Schema for the gateway tool's arguments.
// Per-action parameter and body schemas are served on demand via the
// describe action rather than inlined here, keeping tools/list small.
func (d *Domain) InputSchema() map[string]any {
	actions := make([]any, 0, len(d.Operations)+1)
	for _, op := range d.Operations {
		actions = append(actions, op.Action)
	}
	actions = append(actions, gateway.DescribeAction)
	return map[string]any{
		"type":                 "object",
		"required":             []any{"action"},
		"additionalProperties": false,
		"properties": map[string]any{
			"action": map[string]any{
				"type": "string",
				"enum": actions,
			},
			"params": map[string]any{
				"type":        "object",
				"description": "Parameters for the action. Call describe for the action's schema.",
			},
		},
	}
}

// Describe returns the describe payload for one action, or for the whole
// domain when action is empty.
func (d *Domain) Describe(action string) (any, error) {
	if action == "" {
		summaries := make([]map[string]any, 0, len(d.Operations))
		for _, op := range d.Operations {
			summaries = append(summaries, map[string]any{
				"action":   op.Action,
				"summary":  op.Summary,
				"readonly": op.ReadOnly,
			})
		}
		return map[string]any{"domain": d.Key, "actions": summaries}, nil
	}
	op, ok := d.Operation(action)
	if !ok {
		return nil, fmt.Errorf("unknown action %q in domain %q (actions: %s)", action, d.Key, strings.Join(d.ActionNames(), ", "))
	}
	return op, nil
}

// ActionNames returns the domain's action names in sorted order.
func (d *Domain) ActionNames() []string {
	names := make([]string, 0, len(d.Operations))
	for _, op := range d.Operations {
		names = append(names, op.Action)
	}
	sort.Strings(names)
	return names
}

// Load validates the hand-written domains and returns them ready for the
// gateway. Curation mistakes — duplicate actions, path tokens without a
// matching param, a query param named like a path token — fail here, at
// startup, not at dispatch time.
func Load() ([]*Domain, error) {
	seenKeys := map[string]bool{}
	for _, d := range Domains {
		if seenKeys[d.Key] {
			return nil, fmt.Errorf("duplicate domain key %q", d.Key)
		}
		seenKeys[d.Key] = true
		if err := validate(d); err != nil {
			return nil, fmt.Errorf("domain %q: %w", d.Key, err)
		}
	}
	return Domains, nil
}

// GatewayDomains adapts the catalog for gateway.New.
func GatewayDomains(domains []*Domain) []gateway.Domain {
	out := make([]gateway.Domain, len(domains))
	for i, d := range domains {
		out[i] = d
	}
	return out
}

func validate(d *Domain) error {
	seen := map[string]bool{gateway.DescribeAction: true}
	sorted := sort.SliceIsSorted(d.Operations, func(i, j int) bool {
		return d.Operations[i].Action < d.Operations[j].Action
	})
	if !sorted {
		return fmt.Errorf("operations must be sorted by action name")
	}
	for _, op := range d.Operations {
		if seen[op.Action] {
			return fmt.Errorf("duplicate or reserved action %q", op.Action)
		}
		seen[op.Action] = true
		if op.ReadOnly != (op.Method == "GET") {
			return fmt.Errorf("action %q: readonly must mirror the method (GET and only GET reads)", op.Action)
		}
		if err := validateParams(op); err != nil {
			return fmt.Errorf("action %q: %w", op.Action, err)
		}
	}
	return nil
}

func validateParams(op *Operation) error {
	pathParams := map[string]bool{}
	for _, p := range op.Params {
		switch p.In {
		case "path":
			if !p.Required {
				return fmt.Errorf("path param %q must be required", p.Name)
			}
			pathParams[p.Name] = true
		case "query":
		default:
			return fmt.Errorf("param %q: in must be \"path\" or \"query\", got %q", p.Name, p.In)
		}
		if p.Schema == nil {
			return fmt.Errorf("param %q has no schema", p.Name)
		}
	}
	for _, token := range PathTokens(op.Path) {
		if !pathParams[token] {
			return fmt.Errorf("path token {%s} has no matching path param", token)
		}
		delete(pathParams, token)
	}
	for name := range pathParams {
		return fmt.Errorf("path param %q does not appear in path %q", name, op.Path)
	}
	if op.Body != nil && op.Body["type"] != "object" {
		return fmt.Errorf("body schema must be an object schema")
	}
	if op.Body == nil && op.BodyKey != "" {
		return fmt.Errorf("body_key %q without a body schema", op.BodyKey)
	}
	return nil
}

// PathTokens returns the {name} tokens in a path template, in order.
func PathTokens(path string) []string {
	var tokens []string
	rest := path
	for {
		i := strings.IndexByte(rest, '{')
		if i < 0 {
			return tokens
		}
		rest = rest[i+1:]
		j := strings.IndexByte(rest, '}')
		if j < 0 {
			return tokens
		}
		tokens = append(tokens, rest[:j])
		rest = rest[j+1:]
	}
}
