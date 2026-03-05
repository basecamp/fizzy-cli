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

	t.Run("default is JSON", func(t *testing.T) {
		ResetTestMode()
		f, err := resolveFormat()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if f != output.FormatJSON {
			t.Errorf("expected FormatJSON, got %v", f)
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
