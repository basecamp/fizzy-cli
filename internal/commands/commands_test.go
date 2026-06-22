package commands

import (
	"strings"
	"testing"

	"github.com/basecamp/cli/output"
)

func TestCommandsStyledOutputRendersHumanCatalog(t *testing.T) {
	mock := NewMockClient()
	SetTestModeWithSDK(mock)
	SetTestFormat(output.FormatStyled)
	defer resetTest()

	if err := commandsCmd.RunE(commandsCmd, []string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw := TestOutput()
	if !strings.Contains(raw, "CORE COMMANDS") {
		t.Fatalf("expected styled catalog heading, got:\n%s", raw)
	}
	if !strings.Contains(raw, "auth") || !strings.Contains(raw, "activity") || !strings.Contains(raw, "board") {
		t.Fatalf("expected styled catalog to include commands, got:\n%s", raw)
	}
	if strings.Contains(raw, "list, show") {
		t.Fatalf("expected unfiltered styled catalog to omit action lists, got:\n%s", raw)
	}
}

func TestCommandsFilterRendersMatchingHumanCatalog(t *testing.T) {
	mock := NewMockClient()
	SetTestModeWithSDK(mock)
	SetTestFormat(output.FormatStyled)
	defer resetTest()

	if err := commandsCmd.RunE(commandsCmd, []string{"auth"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw := TestOutput()
	if !strings.Contains(raw, "auth") {
		t.Fatalf("expected filtered catalog to include auth, got:\n%s", raw)
	}
	if strings.Contains(raw, "board") {
		t.Fatalf("expected filtered catalog to omit non-matching board command, got:\n%s", raw)
	}
	if !strings.Contains(raw, "list, login, logout, status, switch") {
		t.Fatalf("expected filtered catalog to include action list, got:\n%s", raw)
	}
}

func TestCommandsFilterFindsActivity(t *testing.T) {
	mock := NewMockClient()
	SetTestModeWithSDK(mock)
	SetTestFormat(output.FormatStyled)
	defer resetTest()

	if err := commandsCmd.RunE(commandsCmd, []string{"activity"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw := TestOutput()
	if !strings.Contains(raw, "activity") || !strings.Contains(raw, "list") {
		t.Fatalf("expected filtered catalog to include activity list, got:\n%s", raw)
	}
	if strings.Contains(raw, "No commands match") {
		t.Fatalf("expected activity to be discoverable, got:\n%s", raw)
	}
}

func TestCommandsFilterFindsToken(t *testing.T) {
	mock := NewMockClient()
	SetTestModeWithSDK(mock)
	SetTestFormat(output.FormatStyled)
	defer resetTest()

	if err := commandsCmd.RunE(commandsCmd, []string{"token"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw := TestOutput()
	if !strings.Contains(raw, "token") || !strings.Contains(raw, "create, delete, list") {
		t.Fatalf("expected filtered catalog to include token actions, got:\n%s", raw)
	}
	if strings.Contains(raw, "No commands match") {
		t.Fatalf("expected token to be discoverable, got:\n%s", raw)
	}
}

func TestCommandsJSONOutputReturnsStructuredCatalog(t *testing.T) {
	mock := NewMockClient()
	result := SetTestModeWithSDK(mock)
	defer resetTest()

	if err := commandsCmd.RunE(commandsCmd, []string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Response == nil || !result.Response.OK {
		t.Fatalf("expected OK JSON response, got %#v", result.Response)
	}

	items, ok := result.Response.Data.([]any)
	if !ok {
		t.Fatalf("expected command catalog slice, got %#v", result.Response.Data)
	}
	if len(items) == 0 {
		t.Fatal("expected command catalog entries")
	}

	found := false
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if entry["name"] == "fizzy commands" {
			if entry["category"] != "utilities" {
				t.Fatalf("expected fizzy commands category utilities, got %#v", entry["category"])
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected command catalog to include fizzy commands, got %#v", items)
	}
}
