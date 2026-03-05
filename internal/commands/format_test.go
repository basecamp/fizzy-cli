package commands

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/basecamp/cli/output"
	"github.com/basecamp/fizzy-cli/internal/client"
)

func TestResolveFormat(t *testing.T) {
	defer ResetTestMode()

	t.Run("default is Auto", func(t *testing.T) {
		ResetTestMode()
		f, err := resolveFormat()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if f != output.FormatAuto {
			t.Errorf("expected FormatAuto, got %v", f)
		}
	})

	t.Run("--json resolves to JSON", func(t *testing.T) {
		ResetTestMode()
		cfgJSON = true
		f, err := resolveFormat()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if f != output.FormatJSON {
			t.Errorf("expected FormatJSON, got %v", f)
		}
	})

	t.Run("--quiet resolves to Quiet", func(t *testing.T) {
		ResetTestMode()
		cfgQuiet = true
		f, err := resolveFormat()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if f != output.FormatQuiet {
			t.Errorf("expected FormatQuiet, got %v", f)
		}
	})

	t.Run("--ids-only resolves to IDs", func(t *testing.T) {
		ResetTestMode()
		cfgIDsOnly = true
		f, err := resolveFormat()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if f != output.FormatIDs {
			t.Errorf("expected FormatIDs, got %v", f)
		}
	})

	t.Run("--count resolves to Count", func(t *testing.T) {
		ResetTestMode()
		cfgCount = true
		f, err := resolveFormat()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if f != output.FormatCount {
			t.Errorf("expected FormatCount, got %v", f)
		}
	})

	t.Run("multiple flags is an error", func(t *testing.T) {
		ResetTestMode()
		cfgQuiet = true
		cfgCount = true
		_, err := resolveFormat()
		if err == nil {
			t.Fatal("expected error for multiple format flags")
		}
		if !strings.Contains(err.Error(), "only one") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("--json with another flag is an error", func(t *testing.T) {
		ResetTestMode()
		cfgJSON = true
		cfgIDsOnly = true
		_, err := resolveFormat()
		if err == nil {
			t.Fatal("expected error for multiple format flags")
		}
	})
}

func TestFormatQuietOutput(t *testing.T) {
	mock := NewMockClient()
	mock.GetWithPaginationResponse = &client.APIResponse{
		StatusCode: 200,
		Data: []interface{}{
			map[string]interface{}{"id": "1", "name": "Board 1"},
			map[string]interface{}{"id": "2", "name": "Board 2"},
		},
	}

	SetTestMode(mock)
	SetTestConfig("token", "account", "https://api.example.com")
	SetTestFormat(output.FormatQuiet)
	defer ResetTestMode()

	err := boardListCmd.RunE(boardListCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw := TestOutput()

	// Quiet mode should produce raw data without envelope
	var data []map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		t.Fatalf("expected raw JSON array, got parse error: %v\noutput: %s", err, raw)
	}
	if len(data) != 2 {
		t.Errorf("expected 2 items, got %d", len(data))
	}

	// Should NOT have "ok" envelope key
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &envelope); err == nil {
		if _, hasOK := envelope["ok"]; hasOK {
			t.Error("quiet format should not include envelope 'ok' key")
		}
	}
}

func TestFormatIDsOutput(t *testing.T) {
	mock := NewMockClient()
	mock.GetWithPaginationResponse = &client.APIResponse{
		StatusCode: 200,
		Data: []map[string]interface{}{
			{"id": "101", "name": "Board 1"},
			{"id": "202", "name": "Board 2"},
			{"id": "303", "name": "Board 3"},
		},
	}

	SetTestMode(mock)
	SetTestConfig("token", "account", "https://api.example.com")
	SetTestFormat(output.FormatIDs)
	defer ResetTestMode()

	err := boardListCmd.RunE(boardListCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw := strings.TrimSpace(TestOutput())
	lines := strings.Split(raw, "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %q", len(lines), raw)
	}
	expected := []string{"101", "202", "303"}
	for i, want := range expected {
		if strings.TrimSpace(lines[i]) != want {
			t.Errorf("line %d: expected %q, got %q", i, want, lines[i])
		}
	}
}

func TestFormatCountOutput(t *testing.T) {
	mock := NewMockClient()
	mock.GetWithPaginationResponse = &client.APIResponse{
		StatusCode: 200,
		Data: []interface{}{
			map[string]interface{}{"id": "1"},
			map[string]interface{}{"id": "2"},
			map[string]interface{}{"id": "3"},
			map[string]interface{}{"id": "4"},
			map[string]interface{}{"id": "5"},
		},
	}

	SetTestMode(mock)
	SetTestConfig("token", "account", "https://api.example.com")
	SetTestFormat(output.FormatCount)
	defer ResetTestMode()

	err := boardListCmd.RunE(boardListCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw := strings.TrimSpace(TestOutput())
	if raw != "5" {
		t.Errorf("expected count '5', got %q", raw)
	}
}

func TestFormatCountSingleObject(t *testing.T) {
	mock := NewMockClient()
	mock.GetResponse = &client.APIResponse{
		StatusCode: 200,
		Data:       map[string]interface{}{"id": "1", "name": "Board 1"},
	}

	SetTestMode(mock)
	SetTestConfig("token", "account", "https://api.example.com")
	SetTestFormat(output.FormatCount)
	defer ResetTestMode()

	err := boardShowCmd.RunE(boardShowCmd, []string{"1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw := strings.TrimSpace(TestOutput())
	if raw != "1" {
		t.Errorf("expected count '1' for single object, got %q", raw)
	}
}

func TestFormatJSONEnvelope(t *testing.T) {
	mock := NewMockClient()
	mock.GetWithPaginationResponse = &client.APIResponse{
		StatusCode: 200,
		Data: []interface{}{
			map[string]interface{}{"id": "1", "name": "Board 1"},
		},
	}

	SetTestMode(mock)
	SetTestConfig("token", "account", "https://api.example.com")
	// SetTestMode already uses FormatJSON — verify the envelope
	defer ResetTestMode()

	err := boardListCmd.RunE(boardListCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw := TestOutput()
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		t.Fatalf("expected JSON object, got parse error: %v", err)
	}
	if envelope["ok"] != true {
		t.Error("expected ok: true in JSON envelope")
	}
	if _, hasData := envelope["data"]; !hasData {
		t.Error("expected data key in JSON envelope")
	}
}

func TestPrettyFlagRemoved(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("pretty")
	if flag != nil {
		t.Error("--pretty flag should be removed")
	}
}

func TestFormatFlagsRegistered(t *testing.T) {
	flags := []string{"json", "quiet", "ids-only", "count"}
	for _, name := range flags {
		if rootCmd.PersistentFlags().Lookup(name) == nil {
			t.Errorf("expected --%s flag to be registered", name)
		}
	}
}

// Integration tests: exercise full Cobra arg parsing → PersistentPreRunE → command output.

func runCobraWithArgs(args ...string) (string, error) {
	// Reset format flags to prevent leaking between calls.
	cfgJSON = false
	cfgQuiet = false
	cfgIDsOnly = false
	cfgCount = false
	cfgAgent = false
	cfgStyled = false
	cfgMarkdown = false
	testBuf.Reset()
	lastRawOutput = ""
	out = output.New(output.Options{Format: output.FormatJSON, Writer: &testBuf})
	lastResult = &CommandResult{}
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	// Capture any remaining output not yet captured by captureResponse.
	if lastRawOutput == "" {
		lastRawOutput = testBuf.String()
	}
	return lastRawOutput, err
}

func TestCobraFormatCount(t *testing.T) {
	mock := NewMockClient()
	mock.GetWithPaginationResponse = &client.APIResponse{
		StatusCode: 200,
		Data: []map[string]interface{}{
			{"id": "1", "name": "A"},
			{"id": "2", "name": "B"},
		},
	}
	clientFactory = func() client.API { return mock }
	SetTestConfig("token", "account", "https://api.example.com")
	defer ResetTestMode()

	raw, err := runCobraWithArgs("board", "list", "--count")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(raw) != "2" {
		t.Errorf("expected '2', got %q", raw)
	}
}

func TestCobraFormatQuiet(t *testing.T) {
	mock := NewMockClient()
	mock.GetResponse = &client.APIResponse{
		StatusCode: 200,
		Data:       map[string]interface{}{"id": "42", "name": "Test Board"},
	}
	clientFactory = func() client.API { return mock }
	SetTestConfig("token", "account", "https://api.example.com")
	defer ResetTestMode()

	raw, err := runCobraWithArgs("board", "show", "42", "--quiet")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		t.Fatalf("expected JSON object, got parse error: %v\noutput: %s", err, raw)
	}
	if _, hasOK := data["ok"]; hasOK {
		t.Error("quiet format should not include envelope 'ok' key")
	}
	if data["name"] != "Test Board" {
		t.Errorf("expected name 'Test Board', got %v", data["name"])
	}
}

func TestCobraFormatIDsOnly(t *testing.T) {
	mock := NewMockClient()
	mock.GetWithPaginationResponse = &client.APIResponse{
		StatusCode: 200,
		Data: []map[string]interface{}{
			{"id": "10", "name": "A"},
			{"id": "20", "name": "B"},
		},
	}
	clientFactory = func() client.API { return mock }
	SetTestConfig("token", "account", "https://api.example.com")
	defer ResetTestMode()

	raw, err := runCobraWithArgs("board", "list", "--ids-only")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(raw), "\n")
	if len(lines) != 2 || lines[0] != "10" || lines[1] != "20" {
		t.Errorf("expected '10\\n20', got %q", raw)
	}
}

func TestCobraMutualExclusion(t *testing.T) {
	mock := NewMockClient()
	clientFactory = func() client.API { return mock }
	SetTestConfig("token", "account", "https://api.example.com")
	defer ResetTestMode()

	_, err := runCobraWithArgs("board", "list", "--quiet", "--count")
	if err == nil {
		t.Fatal("expected error for conflicting format flags")
	}
	if !strings.Contains(err.Error(), "only one") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestResolveFormatAgent(t *testing.T) {
	defer ResetTestMode()

	t.Run("--agent defaults to Quiet", func(t *testing.T) {
		ResetTestMode()
		cfgAgent = true
		f, err := resolveFormat()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if f != output.FormatQuiet {
			t.Errorf("expected FormatQuiet, got %v", f)
		}
	})

	t.Run("--agent --json resolves to JSON", func(t *testing.T) {
		ResetTestMode()
		cfgAgent = true
		cfgJSON = true
		f, err := resolveFormat()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if f != output.FormatJSON {
			t.Errorf("expected FormatJSON, got %v", f)
		}
	})

	t.Run("--agent --styled is an error", func(t *testing.T) {
		ResetTestMode()
		cfgAgent = true
		cfgStyled = true
		_, err := resolveFormat()
		if err == nil {
			t.Fatal("expected error for --agent --styled")
		}
		if !strings.Contains(err.Error(), "cannot be used together") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("--agent --markdown resolves to Markdown", func(t *testing.T) {
		ResetTestMode()
		cfgAgent = true
		cfgMarkdown = true
		f, err := resolveFormat()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if f != output.FormatMarkdown {
			t.Errorf("expected FormatMarkdown, got %v", f)
		}
	})

	t.Run("--agent --ids-only resolves to IDs", func(t *testing.T) {
		ResetTestMode()
		cfgAgent = true
		cfgIDsOnly = true
		f, err := resolveFormat()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if f != output.FormatIDs {
			t.Errorf("expected FormatIDs, got %v", f)
		}
	})

	t.Run("--agent --count resolves to Count", func(t *testing.T) {
		ResetTestMode()
		cfgAgent = true
		cfgCount = true
		f, err := resolveFormat()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if f != output.FormatCount {
			t.Errorf("expected FormatCount, got %v", f)
		}
	})
}

func TestResolveFormatStyledMarkdown(t *testing.T) {
	defer ResetTestMode()

	t.Run("--styled resolves to Styled", func(t *testing.T) {
		ResetTestMode()
		cfgStyled = true
		f, err := resolveFormat()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if f != output.FormatStyled {
			t.Errorf("expected FormatStyled, got %v", f)
		}
	})

	t.Run("--markdown resolves to Markdown", func(t *testing.T) {
		ResetTestMode()
		cfgMarkdown = true
		f, err := resolveFormat()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if f != output.FormatMarkdown {
			t.Errorf("expected FormatMarkdown, got %v", f)
		}
	})

	t.Run("--styled --json is an error", func(t *testing.T) {
		ResetTestMode()
		cfgStyled = true
		cfgJSON = true
		_, err := resolveFormat()
		if err == nil {
			t.Fatal("expected error for --styled --json")
		}
	})

	t.Run("--markdown --quiet is an error", func(t *testing.T) {
		ResetTestMode()
		cfgMarkdown = true
		cfgQuiet = true
		_, err := resolveFormat()
		if err == nil {
			t.Fatal("expected error for --markdown --quiet")
		}
	})
}

func TestFormatFlagsNewRegistered(t *testing.T) {
	flags := []string{"agent", "styled", "markdown"}
	for _, name := range flags {
		if rootCmd.PersistentFlags().Lookup(name) == nil {
			t.Errorf("expected --%s flag to be registered", name)
		}
	}
}

func TestSetupBlockedInMachineMode(t *testing.T) {
	defer ResetTestMode()
	ResetTestMode()
	cfgAgent = true
	err := runSetup(setupCmd, nil)
	if err == nil {
		t.Fatal("expected error when running setup with --agent")
	}
	if !strings.Contains(err.Error(), "interactive terminal") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSkillBlockedInMachineMode(t *testing.T) {
	defer ResetTestMode()
	ResetTestMode()
	cfgAgent = true
	err := runSkill(skillCmd, nil)
	if err == nil {
		t.Fatal("expected error when running skill with --agent")
	}
	if !strings.Contains(err.Error(), "interactive terminal") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLimitFlag(t *testing.T) {
	t.Run("--limit truncates paginated list", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data: []any{
				map[string]any{"id": "1", "name": "Board 1"},
				map[string]any{"id": "2", "name": "Board 2"},
				map[string]any{"id": "3", "name": "Board 3"},
			},
		}

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		cfgLimit = 2
		err := boardListCmd.RunE(boardListCmd, []string{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Response == nil {
			t.Fatal("expected response")
		}
		// Data should be truncated to 2 items
		data, ok := result.Response.Data.([]any)
		if !ok {
			t.Fatalf("expected []any data, got %T", result.Response.Data)
		}
		if len(data) != 2 {
			t.Errorf("expected 2 items after --limit, got %d", len(data))
		}
	})

	t.Run("--limit truncates non-paginated list", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetResponse = &client.APIResponse{
			StatusCode: 200,
			Data: []any{
				map[string]any{"id": "1", "card_number": "10"},
				map[string]any{"id": "2", "card_number": "20"},
				map[string]any{"id": "3", "card_number": "30"},
			},
		}

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		cfgLimit = 1
		err := pinListCmd.RunE(pinListCmd, []string{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, ok := result.Response.Data.([]any)
		if !ok {
			t.Fatalf("expected []any data, got %T", result.Response.Data)
		}
		if len(data) != 1 {
			t.Errorf("expected 1 item after --limit, got %d", len(data))
		}
	})

	t.Run("--limit with notice for non-paginated", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetResponse = &client.APIResponse{
			StatusCode: 200,
			Data: []any{
				map[string]any{"id": "1"},
				map[string]any{"id": "2"},
				map[string]any{"id": "3"},
			},
		}

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		cfgLimit = 2
		err := pinListCmd.RunE(pinListCmd, []string{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Response.Notice != "Showing 2 of 3 results" {
			t.Errorf("expected truncation notice, got %q", result.Response.Notice)
		}
	})

	t.Run("--limit with notice for paginated", func(t *testing.T) {
		mock := NewMockClient()
		items := make([]any, 20)
		for i := range items {
			items[i] = map[string]any{"id": string(rune('A' + i)), "name": "Board"}
		}
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data:       items,
		}

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		cfgLimit = 5
		err := boardListCmd.RunE(boardListCmd, []string{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// With cfgLimit=5, TruncationNotice(5, 20, false, 5) → "Showing 5 results (use --all for complete list)"
		if result.Response.Notice == "" {
			t.Error("expected truncation notice for paginated list with --limit")
		}
		if !strings.Contains(result.Response.Notice, "5 results") {
			t.Errorf("expected notice mentioning 5 results, got %q", result.Response.Notice)
		}
	})

	t.Run("no notice when limit not reached", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetResponse = &client.APIResponse{
			StatusCode: 200,
			Data: []any{
				map[string]any{"id": "1"},
			},
		}

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		cfgLimit = 10
		err := pinListCmd.RunE(pinListCmd, []string{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Response.Notice != "" {
			t.Errorf("expected no notice when data fits within limit, got %q", result.Response.Notice)
		}
	})

	t.Run("--limit and --all is an error", func(t *testing.T) {
		mock := NewMockClient()
		SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		cfgLimit = 5
		boardListAll = true
		err := boardListCmd.RunE(boardListCmd, []string{})
		boardListAll = false
		if err == nil {
			t.Fatal("expected error for --limit with --all")
		}
		if !strings.Contains(err.Error(), "cannot be used together") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("no limit means no truncation", func(t *testing.T) {
		mock := NewMockClient()
		mock.GetWithPaginationResponse = &client.APIResponse{
			StatusCode: 200,
			Data: []any{
				map[string]any{"id": "1"},
				map[string]any{"id": "2"},
				map[string]any{"id": "3"},
			},
		}

		result := SetTestMode(mock)
		SetTestConfig("token", "account", "https://api.example.com")
		defer ResetTestMode()

		err := boardListCmd.RunE(boardListCmd, []string{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, ok := result.Response.Data.([]any)
		if !ok {
			t.Fatalf("expected []any data, got %T", result.Response.Data)
		}
		if len(data) != 3 {
			t.Errorf("expected 3 items with no limit, got %d", len(data))
		}
	})
}

func TestLimitFlagRegistered(t *testing.T) {
	if rootCmd.PersistentFlags().Lookup("limit") == nil {
		t.Error("expected --limit flag to be registered")
	}
}

func TestCheckLimitAll(t *testing.T) {
	t.Run("no conflict when limit is 0", func(t *testing.T) {
		cfgLimit = 0
		if err := checkLimitAll(true); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("no conflict when all is false", func(t *testing.T) {
		cfgLimit = 5
		if err := checkLimitAll(false); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		cfgLimit = 0
	})

	t.Run("conflict when both set", func(t *testing.T) {
		cfgLimit = 5
		err := checkLimitAll(true)
		cfgLimit = 0
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
