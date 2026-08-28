package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	fizzy "github.com/basecamp/fizzy-sdk/go/pkg/fizzy"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/basecamp/mcp/gateway"

	"github.com/basecamp/fizzy-cli/internal/mcpserver/catalog"
)

// API is the slice of the fizzy-sdk client the dispatcher drives. Both
// *fizzy.Client and *fizzy.AccountClient satisfy it; the SDK carries auth,
// retry, and base URL resolution — the AccountClient adds account-slug
// scoping — so the dispatcher only assembles paths and bodies.
type API interface {
	Get(ctx context.Context, path string) (*fizzy.Response, error)
	Post(ctx context.Context, path string, body any) (*fizzy.Response, error)
	Put(ctx context.Context, path string, body any) (*fizzy.Response, error)
	Patch(ctx context.Context, path string, body any) (*fizzy.Response, error)
	Delete(ctx context.Context, path string) (*fizzy.Response, error)
}

// The CLI hands its SDK clients straight to New.
var (
	_ API = (*fizzy.Client)(nil)
	_ API = (*fizzy.AccountClient)(nil)
)

// dispatcher turns catalog operations into fizzy-sdk requests.
//
// Calling convention (matching fizzy-mcp-server): the tool call's params
// object carries the operation's path and query parameters by name, and
// every remaining entry becomes a request body property, wrapped under the
// operation's body key. The describe action serves the schema for all
// three. Failures are in-band isError results per MCP convention.
type dispatcher struct {
	account API // account-scoped operations (the catalog default)
	root    API // the few Unscoped operations, e.g. get_identity
}

// handle dispatches one gateway action as a Fizzy API call. The gateway
// hands over only the dispatch surface (action, read-only); the full
// operation — method, path, param routing — is looked up on the concrete
// catalog domain.
func (d dispatcher) handle(ctx context.Context, dom gateway.Domain, op gateway.Operation, params map[string]any) (*mcp.CallToolResult, error) {
	domain, ok := dom.(*catalog.Domain)
	if !ok {
		return gateway.ErrorResult("internal error: domain %q is not a catalog domain", dom.Name()), nil
	}
	full, ok := domain.Operation(op.Action)
	if !ok {
		return gateway.ErrorResult("internal error: action %q not in domain %q", op.Action, dom.Name()), nil
	}

	path, body, err := buildRequest(full, params)
	if err != nil {
		return gateway.ErrorResult("%v", err), nil
	}

	api := d.account
	if full.Unscoped {
		api = d.root
	}

	res, err := call(ctx, api, full.Method, path, body)
	if err != nil {
		return errorResult(full.Method, path, err), nil
	}

	return d.render(ctx, res)
}

// errorResult renders a failed API call in-band. SDK errors carry the HTTP
// status; surface it so callers can tell a 404 from a 422.
func errorResult(method, path string, err error) *mcp.CallToolResult {
	var apiErr *fizzy.Error
	if errors.As(err, &apiErr) && apiErr.HTTPStatus != 0 {
		return gateway.ErrorResult("fizzy returned %d %s: %v", apiErr.HTTPStatus, http.StatusText(apiErr.HTTPStatus), err)
	}
	return gateway.ErrorResult("%s %s failed: %v", method, path, err)
}

// render translates a Fizzy response into an MCP result: JSON passes
// through, a bodiless success reports its status, a 201 Location is
// followed so create actions return the created resource, and a Link
// rel="next" page number rides along as next_page.
func (d dispatcher) render(ctx context.Context, res *fizzy.Response) (*mcp.CallToolResult, error) {
	if bodiless(res.Data) && res.StatusCode == http.StatusCreated {
		if location := res.Headers.Get("Location"); location != "" {
			created, err := d.followLocation(ctx, location)
			if err != nil {
				// The write succeeded; report it rather than masking it as
				// a failure because the follow-up read did not.
				return gateway.JSONResult(map[string]any{
					"ok": true, "status": res.StatusCode, "location": location,
					"note": fmt.Sprintf("created, but fetching %s failed: %v", location, err),
				})
			}
			res = created
		}
	}

	if bodiless(res.Data) {
		return gateway.JSONResult(map[string]any{"ok": true, "status": res.StatusCode})
	}

	if next := nextPage(res.Headers.Get("Link")); next > 0 {
		wrapped, err := json.Marshal(struct {
			Data     json.RawMessage `json:"data"`
			NextPage int             `json:"next_page"`
		}{res.Data, next})
		if err == nil {
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(wrapped)}}}, nil
		}
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(res.Data)}}}, nil
}

// followLocation fetches a 201 Location so create actions answer with the
// created resource. Locations arrive account-prefixed (relative or
// absolute); an absolute URL is reduced to its path and query, keeping the
// request on the configured instance, and the root client is used so no
// second account prefix is added.
func (d dispatcher) followLocation(ctx context.Context, location string) (*fizzy.Response, error) {
	if strings.HasPrefix(location, "http://") || strings.HasPrefix(location, "https://") {
		u, err := url.Parse(location)
		if err != nil {
			return nil, fmt.Errorf("invalid location %q: %w", location, err)
		}
		location = u.RequestURI()
	}
	return d.root.Get(ctx, location)
}

// bodiless reports whether a response carries no JSON payload. The SDK
// normalizes 204s to a literal null body.
func bodiless(data json.RawMessage) bool {
	trimmed := bytes.TrimSpace(data)
	return len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null"))
}

// call maps the operation's method onto the SDK verbs. The SDK's GET and
// DELETE take no body, so a catalog operation declaring one is refused
// loudly rather than silently dropped — today no such operation exists,
// and this guard keeps a future catalog sync from introducing one
// unnoticed.
func call(ctx context.Context, api API, method, path string, body any) (*fizzy.Response, error) {
	if body != nil && (method == http.MethodGet || method == http.MethodDelete) {
		return nil, fmt.Errorf("internal error: %s %s declares a request body the SDK cannot send", method, path)
	}
	switch method {
	case http.MethodGet:
		return api.Get(ctx, path)
	case http.MethodPost:
		return api.Post(ctx, path, body)
	case http.MethodPut:
		return api.Put(ctx, path, body)
	case http.MethodPatch:
		return api.Patch(ctx, path, body)
	case http.MethodDelete:
		return api.Delete(ctx, path)
	default:
		return nil, fmt.Errorf("internal error: unsupported method %s", method)
	}
}

// buildRequest places each supplied param where the operation declares it:
// path tokens substituted and escaped, query params encoded (arrays
// Rails-style), body fields collected and wrapped under the operation's
// body key. Unknown params and missing required ones are in-band errors
// naming what the action accepts.
func buildRequest(op *catalog.Operation, params map[string]any) (path string, body any, err error) {
	pathParams := map[string]bool{}
	queryParams := map[string]bool{}
	for _, p := range op.Params {
		if p.In == "path" {
			pathParams[p.Name] = true
		} else {
			queryParams[p.Name] = true
		}
	}
	bodyFields := map[string]bool{}
	if op.Body != nil {
		props, _ := op.Body["properties"].(map[string]any)
		for name := range props {
			bodyFields[name] = true
		}
	}

	query := url.Values{}
	bodyValues := map[string]any{}
	substituted := map[string]string{}
	for name, value := range params {
		switch {
		case pathParams[name]:
			str, ok := scalar(value)
			if !ok {
				return "", nil, fmt.Errorf("param %q for action %q must be a string, number, or boolean, got %T", name, op.Action, value)
			}
			substituted[name] = url.PathEscape(str)
		case queryParams[name]:
			if !scalarOrScalarArray(value) {
				return "", nil, fmt.Errorf("param %q for action %q must be a scalar or an array of scalars, got %T", name, op.Action, value)
			}
			addQuery(query, name, value)
		case bodyFields[name]:
			bodyValues[name] = value
		default:
			return "", nil, fmt.Errorf("unknown param %q for action %q (accepts: %s)",
				name, op.Action, strings.Join(accepted(pathParams, queryParams, bodyFields), ", "))
		}
	}

	path = op.Path
	for _, token := range catalog.PathTokens(op.Path) {
		value, ok := substituted[token]
		if !ok || value == "" {
			return "", nil, fmt.Errorf("missing required param %q for action %q", token, op.Action)
		}
		path = strings.ReplaceAll(path, "{"+token+"}", value)
	}

	if op.Body != nil {
		if required, ok := op.Body["required"].([]any); ok {
			for _, name := range required {
				field, _ := name.(string)
				if _, present := bodyValues[field]; !present {
					return "", nil, fmt.Errorf("missing required param %q for action %q", field, op.Action)
				}
			}
		}
	}
	if len(bodyValues) > 0 {
		if op.BodyKey != "" {
			body = map[string]any{op.BodyKey: bodyValues}
		} else {
			body = bodyValues
		}
	}

	if len(query) > 0 {
		path += "?" + query.Encode()
	}
	return path, body, nil
}

func accepted(sets ...map[string]bool) []string {
	var names []string
	for _, set := range sets {
		for name := range set {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// scalar renders a path or query param value, refusing objects and arrays
// — stringified composites would concatenate garbage into request paths.
// json.Unmarshal delivers numbers as float64; integral values render
// without an exponent.
func scalar(v any) (string, bool) {
	switch t := v.(type) {
	case string:
		return t, true
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64), true
	case bool:
		return strconv.FormatBool(t), true
	default:
		return "", false
	}
}

// scalarOrScalarArray reports whether v is a JSON scalar or an array of
// scalars — the only shapes the query encoder renders faithfully.
func scalarOrScalarArray(v any) bool {
	if items, ok := v.([]any); ok {
		for _, item := range items {
			if _, ok := scalar(item); !ok {
				return false
			}
		}
		return true
	}
	_, ok := scalar(v)
	return ok
}

// addQuery appends one parameter value to query, arrays in Rails style:
// repeated keys with a [] suffix (board_ids[]=a&board_ids[]=b).
func addQuery(query url.Values, name string, value any) {
	switch v := value.(type) {
	case []any:
		for _, item := range v {
			s, _ := scalar(item)
			query.Add(name+"[]", s)
		}
	default:
		s, _ := scalar(v)
		query.Add(name, s)
	}
}

// nextPage extracts the page number of the rel="next" target from a Link
// header, 0 when absent or unparseable. Fizzy pages by number (Rails
// geared_pagination), so the caller passes it back as the action's page
// parameter.
func nextPage(link string) int {
	for part := range strings.SplitSeq(link, ",") {
		section := strings.Split(part, ";")
		if len(section) < 2 {
			continue
		}
		target := strings.Trim(strings.TrimSpace(section[0]), "<>")
		rel := ""
		for _, param := range section[1:] {
			param = strings.TrimSpace(param)
			if value, ok := strings.CutPrefix(param, "rel="); ok {
				rel = strings.Trim(value, `"`)
			}
		}
		if rel != "next" {
			continue
		}
		u, err := url.Parse(target)
		if err != nil {
			return 0
		}
		page, err := strconv.Atoi(u.Query().Get("page"))
		if err != nil {
			return 0
		}
		return page
	}
	return 0
}
