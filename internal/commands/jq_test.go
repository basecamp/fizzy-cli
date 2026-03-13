package commands

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/basecamp/cli/output"
	"github.com/basecamp/fizzy-cli/internal/client"
)

func TestJQFlagRegistered(t *testing.T) {
	if rootCmd.PersistentFlags().Lookup("jq") == nil {
		t.Error("expected --jq flag to be registered")
	}
}

func TestResolveFormatJQImpliesJSON(t *testing.T) {
	defer resetTest()

	t.Run("--jq implies JSON", func(t *testing.T) {
		resetTest()
		cfgJQ = ".data"
		f, err := resolveFormat()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if f != output.FormatJSON {
			t.Errorf("expected FormatJSON, got %v", f)
		}
	})

	t.Run("--jq --agent implies Quiet", func(t *testing.T) {
		resetTest()
		cfgJQ = ".data"
		cfgAgent = true
		f, err := resolveFormat()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if f != output.FormatQuiet {
			t.Errorf("expected FormatQuiet, got %v", f)
		}
	})

	t.Run("--jq --styled is an error", func(t *testing.T) {
		resetTest()
		cfgJQ = ".data"
		cfgStyled = true
		_, err := resolveFormat()
		if err == nil {
			t.Fatal("expected error for --jq --styled")
		}
		if !strings.Contains(err.Error(), "--jq cannot be used with") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("--jq --markdown is an error", func(t *testing.T) {
		resetTest()
		cfgJQ = ".data"
		cfgMarkdown = true
		_, err := resolveFormat()
		if err == nil {
			t.Fatal("expected error for --jq --markdown")
		}
	})

	t.Run("--jq --ids-only is an error", func(t *testing.T) {
		resetTest()
		cfgJQ = ".data"
		cfgIDsOnly = true
		_, err := resolveFormat()
		if err == nil {
			t.Fatal("expected error for --jq --ids-only")
		}
	})

	t.Run("--jq --count is an error", func(t *testing.T) {
		resetTest()
		cfgJQ = ".data"
		cfgCount = true
		_, err := resolveFormat()
		if err == nil {
			t.Fatal("expected error for --jq --count")
		}
	})

	t.Run("--jq --json is allowed", func(t *testing.T) {
		resetTest()
		cfgJQ = ".data"
		cfgJSON = true
		f, err := resolveFormat()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if f != output.FormatJSON {
			t.Errorf("expected FormatJSON, got %v", f)
		}
	})

	t.Run("--jq --quiet is allowed", func(t *testing.T) {
		resetTest()
		cfgJQ = ".data"
		cfgQuiet = true
		f, err := resolveFormat()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if f != output.FormatQuiet {
			t.Errorf("expected FormatQuiet, got %v", f)
		}
	})
}

func TestJQIsMachineOutput(t *testing.T) {
	defer resetTest()
	resetTest()
	cfgJQ = ".data"
	if !IsMachineOutput() {
		t.Error("expected IsMachineOutput to be true when --jq is set")
	}
}

func TestJQWriterExtractsField(t *testing.T) {
	var buf strings.Builder
	w, err := newJQWriter(&buf, ".data")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	input := `{"ok":true,"data":[{"id":"1","name":"Board 1"},{"id":"2","name":"Board 2"}]}`
	if _, err := w.Write([]byte(input)); err != nil {
		t.Fatalf("write error: %v", err)
	}

	var result []map[string]any
	if err := json.Unmarshal([]byte(buf.String()), &result); err != nil {
		t.Fatalf("expected JSON array, got parse error: %v\noutput: %s", err, buf.String())
	}
	if len(result) != 2 {
		t.Errorf("expected 2 items, got %d", len(result))
	}
}

func TestJQWriterStringOutput(t *testing.T) {
	var buf strings.Builder
	w, err := newJQWriter(&buf, ".data[0].name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	input := `{"ok":true,"data":[{"id":"1","name":"Board 1"}]}`
	if _, err := w.Write([]byte(input)); err != nil {
		t.Fatalf("write error: %v", err)
	}

	got := strings.TrimSpace(buf.String())
	if got != "Board 1" {
		t.Errorf("expected plain string 'Board 1', got %q", got)
	}
}

func TestJQWriterSelect(t *testing.T) {
	var buf strings.Builder
	w, err := newJQWriter(&buf, `[.data[] | select(.completed == true)]`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	input := `{"ok":true,"data":[{"id":"1","completed":true},{"id":"2","completed":false},{"id":"3","completed":true}]}`
	if _, err := w.Write([]byte(input)); err != nil {
		t.Fatalf("write error: %v", err)
	}

	var result []map[string]any
	if err := json.Unmarshal([]byte(buf.String()), &result); err != nil {
		t.Fatalf("expected JSON array: %v\noutput: %s", err, buf.String())
	}
	if len(result) != 2 {
		t.Errorf("expected 2 completed items, got %d", len(result))
	}
}

func TestJQWriterLength(t *testing.T) {
	var buf strings.Builder
	w, err := newJQWriter(&buf, ".data | length")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	input := `{"ok":true,"data":[{"id":"1"},{"id":"2"},{"id":"3"}]}`
	if _, err := w.Write([]byte(input)); err != nil {
		t.Fatalf("write error: %v", err)
	}

	got := strings.TrimSpace(buf.String())
	if got != "3" {
		t.Errorf("expected '3', got %q", got)
	}
}

func TestJQWriterInvalidExpression(t *testing.T) {
	_, err := newJQWriter(nil, ".data[")
	if err == nil {
		t.Fatal("expected error for invalid jq expression")
	}
	if !strings.Contains(err.Error(), "invalid jq expression") {
		t.Errorf("expected 'invalid jq expression' error, got: %v", err)
	}
}

func TestJQWriterMap(t *testing.T) {
	var buf strings.Builder
	w, err := newJQWriter(&buf, "[.data[] | {id, name}]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	input := `{"ok":true,"data":[{"id":"1","name":"A","extra":"x"},{"id":"2","name":"B","extra":"y"}]}`
	if _, err := w.Write([]byte(input)); err != nil {
		t.Fatalf("write error: %v", err)
	}

	var result []map[string]any
	if err := json.Unmarshal([]byte(buf.String()), &result); err != nil {
		t.Fatalf("expected JSON array: %v\noutput: %s", err, buf.String())
	}
	if len(result) != 2 {
		t.Errorf("expected 2 items, got %d", len(result))
	}
	// Should not contain "extra" field
	if _, hasExtra := result[0]["extra"]; hasExtra {
		t.Error("expected 'extra' field to be excluded")
	}
}

func TestJQWriterPassthroughErrorEnvelope(t *testing.T) {
	var buf strings.Builder
	w, err := newJQWriter(&buf, ".data")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Error envelopes (ok: false) should pass through unfiltered
	errorJSON := `{"ok":false,"error":"not found","code":"not_found"}`
	if _, err := w.Write([]byte(errorJSON)); err != nil {
		t.Fatalf("write error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(buf.String()), &result); err != nil {
		t.Fatalf("expected JSON: %v\noutput: %s", err, buf.String())
	}
	if result["ok"] != false {
		t.Error("expected ok to be false")
	}
	if result["error"] != "not found" {
		t.Errorf("expected error message preserved, got %v", result["error"])
	}
}

func TestJQWriterPassthroughNonJSON(t *testing.T) {
	var buf strings.Builder
	w, err := newJQWriter(&buf, ".data")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nonJSON := "not json at all"
	if _, err := w.Write([]byte(nonJSON)); err != nil {
		t.Fatalf("write error: %v", err)
	}
	if buf.String() != nonJSON {
		t.Errorf("expected passthrough of non-JSON, got %q", buf.String())
	}
}

func TestJQWriterIdentity(t *testing.T) {
	var buf strings.Builder
	w, err := newJQWriter(&buf, ".")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	input := `{"ok":true,"data":"hello"}`
	if _, err := w.Write([]byte(input)); err != nil {
		t.Fatalf("write error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(buf.String()), &result); err != nil {
		t.Fatalf("expected JSON: %v\noutput: %s", err, buf.String())
	}
	if result["ok"] != true {
		t.Error("expected identity filter to pass through full object")
	}
}

func TestCobraJQExtractsData(t *testing.T) {
	mock := NewMockClient()
	mock.GetWithPaginationResponse = &client.APIResponse{
		StatusCode: 200,
		Data: []map[string]any{
			{"id": "1", "name": "Board 1"},
			{"id": "2", "name": "Board 2"},
		},
	}
	SetTestModeWithSDK(mock)
	SetTestConfig("token", "account", "https://api.example.com")
	defer resetTest()

	raw, err := runCobraWithArgs("board", "list", "--jq", ".data")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data []map[string]any
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		t.Fatalf("expected JSON array, got parse error: %v\noutput: %s", err, raw)
	}
	if len(data) != 2 {
		t.Errorf("expected 2 items, got %d", len(data))
	}
}

func TestCobraJQExtractsFieldAsString(t *testing.T) {
	mock := NewMockClient()
	mock.GetWithPaginationResponse = &client.APIResponse{
		StatusCode: 200,
		Data: []map[string]any{
			{"id": "1", "name": "Board 1"},
		},
	}
	SetTestModeWithSDK(mock)
	SetTestConfig("token", "account", "https://api.example.com")
	defer resetTest()

	raw, err := runCobraWithArgs("board", "list", "--jq", ".data[0].name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := strings.TrimSpace(raw)
	if got != "Board 1" {
		t.Errorf("expected 'Board 1', got %q", got)
	}
}

func TestCobraJQInvalidExpression(t *testing.T) {
	mock := NewMockClient()
	SetTestModeWithSDK(mock)
	SetTestConfig("token", "account", "https://api.example.com")
	defer resetTest()

	_, err := runCobraWithArgs("board", "list", "--jq", ".data[")
	if err == nil {
		t.Fatal("expected error for invalid jq expression")
	}
}
