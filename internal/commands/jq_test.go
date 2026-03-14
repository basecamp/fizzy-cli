package commands

import (
	"encoding/json"
	stderrors "errors"
	"strings"
	"testing"

	"github.com/basecamp/cli/output"
	"github.com/basecamp/fizzy-cli/internal/client"
	"github.com/basecamp/fizzy-cli/internal/errors"
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
	if !strings.Contains(err.Error(), "invalid --jq expression") {
		t.Errorf("expected 'invalid --jq expression' error, got: %v", err)
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

// =============================================================================
// Early validation tests (PersistentPreRunE)
// =============================================================================

func TestJQInvalidExpressionRejectedBeforeRunE(t *testing.T) {
	mock := NewMockClient()
	SetTestModeWithSDK(mock)
	SetTestConfig("token", "account", "https://api.example.com")
	defer resetTest()

	_, err := runCobraWithArgs("board", "list", "--jq", ".[invalid")
	if err == nil {
		t.Fatal("expected error for invalid jq expression")
	}
	if !strings.Contains(err.Error(), "invalid --jq expression") {
		t.Errorf("expected 'invalid --jq expression', got: %v", err)
	}
	if !errors.IsJQError(err) {
		t.Error("expected IsJQError to be true")
	}
}

func TestJQCompileErrorRejectedBeforeRunE(t *testing.T) {
	mock := NewMockClient()
	SetTestModeWithSDK(mock)
	SetTestConfig("token", "account", "https://api.example.com")
	defer resetTest()

	_, err := runCobraWithArgs("board", "list", "--jq", "$__loc__")
	if err == nil {
		t.Fatal("expected error for compile-time jq error")
	}
	if !strings.Contains(err.Error(), "invalid --jq expression") {
		t.Errorf("expected 'invalid --jq expression', got: %v", err)
	}
}

func TestJQWithIDsOnlyConflict(t *testing.T) {
	mock := NewMockClient()
	SetTestModeWithSDK(mock)
	SetTestConfig("token", "account", "https://api.example.com")
	defer resetTest()

	_, err := runCobraWithArgs("board", "list", "--jq", ".data", "--ids-only")
	if err == nil {
		t.Fatal("expected error for --jq --ids-only conflict")
	}
	if !strings.Contains(err.Error(), "cannot use --jq with --ids-only") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestJQWithCountConflict(t *testing.T) {
	mock := NewMockClient()
	SetTestModeWithSDK(mock)
	SetTestConfig("token", "account", "https://api.example.com")
	defer resetTest()

	_, err := runCobraWithArgs("board", "list", "--jq", ".data", "--count")
	if err == nil {
		t.Fatal("expected error for --jq --count conflict")
	}
	if !strings.Contains(err.Error(), "cannot use --jq with --count") {
		t.Errorf("unexpected error: %v", err)
	}
}

// =============================================================================
// env.VAR and $ENV.VAR access
// =============================================================================

func TestJQWriterEnvAccess(t *testing.T) {
	t.Setenv("FIZZY_TEST_JQ_VAR", "hello_from_env")

	var buf strings.Builder
	w, err := newJQWriter(&buf, `env.FIZZY_TEST_JQ_VAR`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	input := `{"ok":true,"data":{"id":1}}`
	if _, err := w.Write([]byte(input)); err != nil {
		t.Fatalf("write error: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "hello_from_env" {
		t.Errorf("expected 'hello_from_env', got %q", buf.String())
	}
}

func TestJQWriterEnvDollarAccess(t *testing.T) {
	t.Setenv("FIZZY_TEST_JQ_VAR2", "dollar_form")

	var buf strings.Builder
	w, err := newJQWriter(&buf, `$ENV.FIZZY_TEST_JQ_VAR2`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	input := `{"ok":true,"data":{"id":1}}`
	if _, err := w.Write([]byte(input)); err != nil {
		t.Fatalf("write error: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "dollar_form" {
		t.Errorf("expected 'dollar_form', got %q", buf.String())
	}
}

// =============================================================================
// Compact output
// =============================================================================

func TestJQWriterCompactOutput(t *testing.T) {
	var buf strings.Builder
	w, err := newJQWriter(&buf, `.data`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	input := `{"ok":true,"data":{"name":"test","value":42}}`
	if _, err := w.Write([]byte(input)); err != nil {
		t.Fatalf("write error: %v", err)
	}

	out := strings.TrimSpace(buf.String())
	// Must be single-line compact JSON
	if strings.Contains(out, "\n") {
		t.Error("expected compact single-line JSON, got multi-line")
	}
	if !strings.Contains(out, `"name":"test"`) {
		t.Errorf("expected compact JSON with name field, got %q", out)
	}
}

// =============================================================================
// Edge cases: empty, null, multi-result, non-serializable
// =============================================================================

func TestJQWriterEmptyResult(t *testing.T) {
	var buf strings.Builder
	w, err := newJQWriter(&buf, `empty`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	input := `{"ok":true,"data":{"id":1}}`
	if _, err := w.Write([]byte(input)); err != nil {
		t.Fatalf("write error: %v", err)
	}
	if buf.String() != "" {
		t.Errorf("expected no output for empty result, got %q", buf.String())
	}
}

func TestJQWriterNullResult(t *testing.T) {
	var buf strings.Builder
	w, err := newJQWriter(&buf, `null`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	input := `{"ok":true,"data":{"id":1}}`
	if _, err := w.Write([]byte(input)); err != nil {
		t.Fatalf("write error: %v", err)
	}
	if strings.TrimSpace(buf.String()) != "null" {
		t.Errorf("expected 'null', got %q", buf.String())
	}
}

func TestJQWriterMultiResult(t *testing.T) {
	var buf strings.Builder
	w, err := newJQWriter(&buf, `.data.a, .data.b, .data.c`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	input := `{"ok":true,"data":{"a":1,"b":"two","c":true}}`
	if _, err := w.Write([]byte(input)); err != nil {
		t.Fatalf("write error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "1" {
		t.Errorf("expected '1', got %q", lines[0])
	}
	if lines[1] != "two" {
		t.Errorf("expected 'two', got %q", lines[1])
	}
	if lines[2] != "true" {
		t.Errorf("expected 'true', got %q", lines[2])
	}
}

func TestJQWriterNonSerializableResultReturnsError(t *testing.T) {
	for _, expr := range []string{"nan", "infinite"} {
		t.Run(expr, func(t *testing.T) {
			var buf strings.Builder
			w, err := newJQWriter(&buf, expr)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			input := `{"ok":true,"data":{"id":1}}`
			_, err = w.Write([]byte(input))
			if err == nil {
				t.Fatal("expected error for non-serializable result")
			}
			if !strings.Contains(err.Error(), "not serializable") {
				t.Errorf("expected 'not serializable' error, got: %v", err)
			}
			if !errors.IsJQError(err) {
				t.Error("expected IsJQError to be true")
			}
		})
	}
}

func TestJQWriterRuntimeError(t *testing.T) {
	var buf strings.Builder
	w, err := newJQWriter(&buf, `.data / 0`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	input := `{"ok":true,"data":1}`
	_, err = w.Write([]byte(input))
	if err == nil {
		t.Fatal("expected runtime error")
	}
	if !errors.IsJQError(err) {
		t.Error("expected IsJQError to be true for runtime error")
	}
	var outputErr *output.Error
	if !stderrors.As(err, &outputErr) {
		t.Error("expected error to be *output.Error")
	} else if outputErr.Code != output.CodeUsage {
		t.Errorf("expected CodeUsage, got %v", outputErr.Code)
	}
}

// =============================================================================
// Command-level --jq rejection
// =============================================================================

func TestJQRejectedByCompletion(t *testing.T) {
	mock := NewMockClient()
	SetTestModeWithSDK(mock)
	SetTestConfig("token", "account", "https://api.example.com")
	defer resetTest()

	_, err := runCobraWithArgs("completion", "bash", "--jq", ".data")
	if err == nil {
		t.Fatal("expected error for --jq with completion")
	}
	if !strings.Contains(err.Error(), "--jq is not supported by") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestJQRejectedBySkillInMachineMode(t *testing.T) {
	defer resetTest()
	resetTest()
	cfgAgent = true
	cfgJQ = ".data"

	err := runSkill(skillCmd, nil)
	if err == nil {
		t.Fatal("expected error for --jq with skill in machine mode")
	}
	if !strings.Contains(err.Error(), "--jq is not supported by") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestJQRejectedByVersion(t *testing.T) {
	mock := NewMockClient()
	SetTestModeWithSDK(mock)
	SetTestConfig("token", "account", "https://api.example.com")
	defer resetTest()

	_, err := runCobraWithArgs("version", "--jq", ".version")
	if err == nil {
		t.Fatal("expected error for --jq with version")
	}
	if !strings.Contains(err.Error(), "--jq is not supported by the version command") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestVersionOutputPlainText(t *testing.T) {
	defer resetTest()
	resetTest()

	var buf strings.Builder
	versionCmd.SetOut(&buf)
	defer versionCmd.SetOut(nil)

	err := versionCmd.RunE(versionCmd, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "fizzy version") {
		t.Errorf("expected 'fizzy version' in output, got %q", buf.String())
	}
}

func TestJQSetupBlockedInMachineMode(t *testing.T) {
	defer resetTest()
	resetTest()
	cfgJQ = ".data"

	err := runSetup(setupCmd, nil)
	if err == nil {
		t.Fatal("expected error when running setup with --jq")
	}
	if !strings.Contains(err.Error(), "interactive terminal") {
		t.Errorf("unexpected error: %v", err)
	}
}
