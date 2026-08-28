package commands

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCPCommandRegistration(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"mcp"})
	if err != nil || cmd.Name() != "mcp" {
		t.Fatalf("mcp command not registered: %v", err)
	}

	writes := cmd.Flags().Lookup("writes")
	if writes == nil || writes.DefValue != "false" {
		t.Errorf("writes flag = %#v, want present and defaulting off (read-only is the served default)", writes)
	}
	if cmd.Flags().Lookup("domains") == nil {
		t.Error("domains flag missing")
	}
}

func TestMCPCommandRequiresAuth(t *testing.T) {
	resetMCPFlags(t)
	mock := NewMockClient()
	SetTestModeWithSDK(mock)
	SetTestConfig("", "", "https://api.example.com")
	defer resetTest()

	rootCmd.SetArgs([]string{"mcp"})
	err := rootCmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "No API token configured") {
		t.Fatalf("err = %v, want auth error", err)
	}
}

// resetMCPFlags clears the mcp flag variables: cobra's Var flags
// accumulate across Execute calls within one process, so each test starts
// from the command's real defaults.
func resetMCPFlags(t *testing.T) {
	t.Helper()
	mcpWrites = false
	mcpDomains = nil
}

// stubMCPTransport swaps the stdio transport for the server side of an
// in-memory pipe and returns the client side.
func stubMCPTransport(t *testing.T) mcp.Transport {
	t.Helper()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	orig := mcpTransport
	mcpTransport = func() mcp.Transport { return serverTransport }
	t.Cleanup(func() { mcpTransport = orig })
	return clientTransport
}

// runMCPCommand runs `fizzy mcp` against a stub Fizzy upstream and
// connects a real MCP client to it over the transport seam. The command
// exits when the client session closes.
func runMCPCommand(t *testing.T, upstream *httptest.Server, args ...string) *mcp.ClientSession {
	t.Helper()

	resetMCPFlags(t)
	SetTestModeWithSDK(NewMockClient())
	SetTestSDK(upstream.URL) // repoint the SDK at the asserting upstream
	SetTestConfig("test-token", "test-account", upstream.URL)
	t.Cleanup(resetTest)

	clientTransport := stubMCPTransport(t)

	rootCmd.SetArgs(append([]string{"mcp"}, args...))
	done := make(chan error, 1)
	go func() { done <- rootCmd.Execute() }()
	t.Cleanup(func() {
		if err := <-done; err != nil {
			t.Errorf("fizzy mcp exited with error: %v", err)
		}
	})

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.0"}, nil)
	session, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatalf("MCP initialize failed: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func TestMCPCommandServesMCP(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/test-account/boards" {
			t.Errorf("unexpected HTTP request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
			return
		}
		// Tool calls must ride on the CLI's own credentials.
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q, want the CLI's token", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":"b1","name":"Launch list"}]`))
	}))
	t.Cleanup(upstream.Close)

	session := runMCPCommand(t, upstream)

	if got := session.InitializeResult().ServerInfo.Name; got != "fizzy-cli" {
		t.Errorf("server name = %q, want fizzy-cli", got)
	}

	names := make([]string, 0, 8)
	for tool, err := range session.Tools(context.Background(), nil) {
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, tool.Name)
	}
	if len(names) != 8 {
		t.Fatalf("tools = %v, want 8 fizzy_* tools", names)
	}

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "fizzy_boards",
		Arguments: map[string]any{"action": "list_boards", "params": map[string]any{}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("list_boards failed: %v", result.Content)
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("content = %T", result.Content[0])
	}
	var boards []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(text.Text), &boards); err != nil || len(boards) == 0 || boards[0].Name != "Launch list" {
		t.Fatalf("list_boards result = %q (%v)", text.Text, err)
	}
}

func TestMCPCommandDefaultsToReadOnlyAndPassesDomainsThrough(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected HTTP request: %s %s", r.Method, r.URL.Path)
		http.NotFound(w, r)
	}))
	t.Cleanup(upstream.Close)

	session := runMCPCommand(t, upstream, "--domains", "cards")

	tools := make([]*mcp.Tool, 0, 1)
	for tool, err := range session.Tools(context.Background(), nil) {
		if err != nil {
			t.Fatal(err)
		}
		tools = append(tools, tool)
	}
	if len(tools) != 1 || tools[0].Name != "fizzy_cards" {
		t.Fatalf("tools = %v, want just fizzy_cards", tools)
	}
	if !strings.Contains(tools[0].Description, "get_card") {
		t.Error("read-only fizzy_cards lost its read actions")
	}
	if strings.Contains(tools[0].Description, "create_card") {
		t.Error("fizzy_cards lists a write action without --writes")
	}
}

func TestMCPCommandWritesOptIn(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected HTTP request: %s %s", r.Method, r.URL.Path)
		http.NotFound(w, r)
	}))
	t.Cleanup(upstream.Close)

	session := runMCPCommand(t, upstream, "--writes", "--domains", "cards")

	tools := make([]*mcp.Tool, 0, 1)
	for tool, err := range session.Tools(context.Background(), nil) {
		if err != nil {
			t.Fatal(err)
		}
		tools = append(tools, tool)
	}
	if len(tools) != 1 || tools[0].Name != "fizzy_cards" {
		t.Fatalf("tools = %v, want just fizzy_cards", tools)
	}
	if !strings.Contains(tools[0].Description, "create_card") {
		t.Error("--writes did not serve write actions")
	}
}

func TestMCPCommandUnknownDomainFailsClosed(t *testing.T) {
	resetMCPFlags(t)
	mock := NewMockClient()
	SetTestModeWithSDK(mock)
	SetTestConfig("test-token", "test-account", "https://api.example.com")
	defer resetTest()

	rootCmd.SetArgs([]string{"mcp", "--domains", "bogus"})
	err := rootCmd.Execute()
	if err == nil || !strings.Contains(err.Error(), `unknown domain "bogus"`) {
		t.Fatalf("err = %v, want unknown domain failure", err)
	}
}
