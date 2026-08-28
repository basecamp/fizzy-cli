// Wire tests: a real MCP client drives the server over in-memory
// transports (the toolkit's mcptest harness), and dispatched actions land
// on a fake Fizzy (httptest) through a real fizzy-sdk client — so the
// asserted HTTP surface (bearer auth, account scoping, query encoding,
// body wrapping) is exactly what `fizzy mcp` sends, and the played-back
// response shapes (JSON, 204s, 201 Locations, Link pagination) are
// rendered the way the API produces them.
package mcpserver

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	fizzy "github.com/basecamp/fizzy-sdk/go/pkg/fizzy"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/basecamp/mcp/mcptest"
)

const testAccount = "897362094"

// fakeFizzy runs an httptest server that checks auth on every request and
// serves the registered handlers.
func fakeFizzy(t *testing.T, mux *http.ServeMux) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q, want the CLI's token", got)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept = %q", got)
		}
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(ts.Close)
	return ts
}

// connect builds a server against the fake Fizzy — through a real SDK
// client bound to the test account — and connects a real MCP client to
// it. Writes are enabled unless the config says otherwise, so dispatch
// tests reach write actions; read-only is the served default and has its
// own tests.
func connect(t *testing.T, cfg Config, upstream *httptest.Server) (*Server, *mcp.ClientSession) {
	t.Helper()

	baseURL := "http://127.0.0.1:0" // no upstream: any dispatch attempt fails loudly
	if upstream != nil {
		baseURL = upstream.URL
	}
	sdk := fizzy.NewClient(&fizzy.Config{BaseURL: baseURL}, &fizzy.StaticTokenProvider{Token: "test-token"})

	srv, err := New(sdk.ForAccount(testAccount), sdk, cfg)
	if err != nil {
		t.Fatal(err)
	}
	return srv, mcptest.Connect(t, srv.BuildMCPServer(slog.New(slog.DiscardHandler)))
}

func callJSON(t *testing.T, session *mcp.ClientSession, tool string, args map[string]any) map[string]any {
	t.Helper()
	text, isError := mcptest.CallText(t, session, tool, args)
	if isError {
		t.Fatalf("call failed: %s", text)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		t.Fatalf("result %q: %v", text, err)
	}
	return payload
}

func TestListToolsServesCatalog(t *testing.T) {
	_, session := connect(t, Config{}, nil)
	tools := mcptest.ListTools(t, session)

	for _, name := range []string{"fizzy_identity", "fizzy_boards", "fizzy_columns", "fizzy_cards", "fizzy_comments", "fizzy_steps", "fizzy_tags", "fizzy_users"} {
		tool, ok := tools[name]
		if !ok {
			t.Errorf("missing tool %q", name)
			continue
		}
		if !strings.Contains(tool.Description, "ACTIONS") {
			t.Errorf("tool %q description lacks its action list", name)
		}
	}
	if len(tools) != 8 {
		t.Errorf("tools/list returned %d tools, want 8", len(tools))
	}

	// identity is all reads; cards is not.
	if !tools["fizzy_identity"].Annotations.ReadOnlyHint {
		t.Error("fizzy_identity must hint read-only")
	}
	if tools["fizzy_cards"].Annotations.ReadOnlyHint {
		t.Error("fizzy_cards must not hint read-only with writes served")
	}
}

func TestReadOnlyFiltersWriteActions(t *testing.T) {
	srv, session := connect(t, Config{ReadOnly: true}, nil)

	// Every domain keeps at least one read, so all eight tools survive,
	// and every survivor is all-read.
	tools := mcptest.ListTools(t, session)
	if len(tools) != 8 {
		t.Fatalf("tools/list returned %d tools, want 8", len(tools))
	}
	for name, tool := range tools {
		if !tool.Annotations.ReadOnlyHint {
			t.Errorf("tool %q must be read-only", name)
		}
	}
	for _, d := range srv.Domains() {
		if slices.Contains(d.ActionNames(), "create_card") {
			t.Error("read-only server still serves create_card")
		}
	}

	// A filtered write action is gone from the catalog, so dispatch
	// refuses it in-band even when a client ignores the schema.
	text, isError := mcptest.CallText(t, session, "fizzy_cards", map[string]any{"action": "create_card"})
	if !isError {
		t.Fatalf("write action succeeded on read-only server: %s", text)
	}
	if !strings.Contains(text, "unknown action") {
		t.Errorf("refusal = %q", text)
	}
}

func TestDescribeServesOperationSchema(t *testing.T) {
	_, session := connect(t, Config{}, nil)

	op := callJSON(t, session, "fizzy_cards", map[string]any{
		"action": "describe",
		"params": map[string]any{"action": "triage_card"},
	})
	if op["method"] != "POST" || op["path"] != "/cards/{card_number}/triage" {
		t.Errorf("describe = %v", op)
	}
}

func TestUnknownDomainFailsClosed(t *testing.T) {
	sdk := fizzy.NewClient(&fizzy.Config{BaseURL: "http://127.0.0.1:0"}, &fizzy.StaticTokenProvider{Token: "test-token"})
	_, err := New(sdk.ForAccount(testAccount), sdk, Config{Domains: []string{"cards", "nope"}})
	if err == nil || !strings.Contains(err.Error(), `unknown domain "nope"`) {
		t.Errorf("err = %v, want unknown domain failure", err)
	}
}

func TestNarrowedDomainsServeOnlyThose(t *testing.T) {
	_, session := connect(t, Config{Domains: []string{"cards"}}, nil)
	tools := mcptest.ListTools(t, session)
	if len(tools) != 1 {
		t.Fatalf("tools = %d, want just fizzy_cards", len(tools))
	}
	if _, ok := tools["fizzy_cards"]; !ok {
		t.Fatal("fizzy_cards missing")
	}
}

func TestServerRequiresClients(t *testing.T) {
	if _, err := New(nil, nil, Config{}); err == nil {
		t.Error("nil clients did not error")
	}
}

func TestDispatchGetIsAccountScoped(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /"+testAccount+"/cards/42", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"number": 42, "title": "First!"}`))
	})
	_, session := connect(t, Config{}, fakeFizzy(t, mux))

	card := callJSON(t, session, "fizzy_cards", map[string]any{
		"action": "get_card",
		"params": map[string]any{"card_number": 42},
	})
	if card["title"] != "First!" {
		t.Errorf("card = %v", card)
	}
}

func TestDispatchEncodesQueryRailsStyle(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /"+testAccount+"/cards", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if got := query["board_ids[]"]; !slices.Equal(got, []string{"b1", "b2"}) {
			t.Errorf("board_ids[] = %v", got)
		}
		if query.Get("indexed_by") != "closed" || query.Get("page") != "2" {
			t.Errorf("query = %v", query)
		}
		_, _ = w.Write([]byte(`[]`))
	})
	_, session := connect(t, Config{}, fakeFizzy(t, mux))

	text, isError := mcptest.CallText(t, session, "fizzy_cards", map[string]any{
		"action": "list_cards",
		"params": map[string]any{
			"board_ids":  []any{"b1", "b2"},
			"indexed_by": "closed",
			"page":       2,
		},
	})
	if isError || text != `[]` {
		t.Errorf("result = %q (isError=%v)", text, isError)
	}
}

func TestDispatchWrapsBodyAndFollowsCreatedLocation(t *testing.T) {
	location := "/" + testAccount + "/cards/7.json"
	mux := http.NewServeMux()
	mux.HandleFunc("POST /"+testAccount+"/boards/b1/cards", func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		var body map[string]any
		if err := json.Unmarshal(data, &body); err != nil {
			t.Fatalf("body %q: %v", data, err)
		}
		card, _ := body["card"].(map[string]any)
		if card["title"] != "Add dark mode" || card["description"] != "Please" {
			t.Errorf("body = %s", data)
		}
		w.Header().Set("Location", location)
		w.WriteHeader(http.StatusCreated)
	})
	mux.HandleFunc("GET "+location, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"number": 7, "title": "Add dark mode"}`))
	})
	_, session := connect(t, Config{}, fakeFizzy(t, mux))

	card := callJSON(t, session, "fizzy_cards", map[string]any{
		"action": "create_card",
		"params": map[string]any{"board_id": "b1", "title": "Add dark mode", "description": "Please"},
	})
	if card["number"] != float64(7) {
		t.Errorf("create must return the created resource via its Location, got %v", card)
	}
}

func TestDispatchReportsBodilessSuccess(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /"+testAccount+"/cards/42/triage", func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		var body map[string]any
		if err := json.Unmarshal(data, &body); err != nil || body["column_id"] != "c9" {
			t.Errorf("body = %s", data)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	_, session := connect(t, Config{}, fakeFizzy(t, mux))

	result := callJSON(t, session, "fizzy_cards", map[string]any{
		"action": "triage_card",
		"params": map[string]any{"card_number": "42", "column_id": "c9"},
	})
	if result["ok"] != true || result["status"] != float64(204) {
		t.Errorf("result = %v", result)
	}
}

func TestDispatchSurfacesPagination(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /"+testAccount+"/cards", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Link", `<http://example.test/`+testAccount+`/cards?page=2>; rel="next"`)
		_, _ = w.Write([]byte(`[{"number": 1}]`))
	})
	_, session := connect(t, Config{}, fakeFizzy(t, mux))

	page := callJSON(t, session, "fizzy_cards", map[string]any{"action": "list_cards"})
	if page["next_page"] != float64(2) {
		t.Errorf("next_page = %v", page["next_page"])
	}
	if data, ok := page["data"].([]any); !ok || len(data) != 1 {
		t.Errorf("data = %v", page["data"])
	}
}

func TestDispatchSurfacesAPIErrorsInBand(t *testing.T) {
	mux := http.NewServeMux() // no routes: everything 404s
	_, session := connect(t, Config{}, fakeFizzy(t, mux))

	text, isError := mcptest.CallText(t, session, "fizzy_cards", map[string]any{
		"action": "get_card",
		"params": map[string]any{"card_number": "999"},
	})
	if !isError {
		t.Fatalf("missing card did not error: %s", text)
	}
	if !strings.Contains(text, "404") {
		t.Errorf("error = %q, want the HTTP status surfaced", text)
	}
}

func TestIdentityIsUnscoped(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /my/identity", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"accounts": [{"name": "37signals", "slug": "/` + testAccount + `"}]}`))
	})
	_, session := connect(t, Config{}, fakeFizzy(t, mux))

	payload := callJSON(t, session, "fizzy_identity", map[string]any{"action": "get_identity"})
	if _, ok := payload["accounts"]; !ok {
		t.Errorf("payload = %v", payload)
	}
}
