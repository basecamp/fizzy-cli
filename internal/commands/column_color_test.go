package commands

import (
	"strings"
	"testing"

	"github.com/basecamp/fizzy-cli/internal/client"
	"github.com/basecamp/fizzy-cli/internal/errors"
)

func TestNormalizeColumnColor(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "blue alias", in: "blue", want: "var(--color-card-default)"},
		{name: "gray alias", in: "gray", want: "var(--color-card-1)"},
		{name: "tan alias", in: "tan", want: "var(--color-card-2)"},
		{name: "yellow alias", in: "yellow", want: "var(--color-card-3)"},
		{name: "lime alias", in: "lime", want: "var(--color-card-4)"},
		{name: "aqua alias", in: "aqua", want: "var(--color-card-5)"},
		{name: "violet alias", in: "violet", want: "var(--color-card-6)"},
		{name: "purple alias", in: "purple", want: "var(--color-card-7)"},
		{name: "pink alias", in: "pink", want: "var(--color-card-8)"},
		{name: "mixed-case alias", in: "AqUa", want: "var(--color-card-5)"},
		{name: "trimmed alias", in: " aqua ", want: "var(--color-card-5)"},
		{name: "API value", in: "var(--color-card-5)", want: "var(--color-card-5)"},
		{name: "blank", in: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeColumnColor(tt.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("normalizeColumnColor(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestNormalizeColumnColorRejectsUnknownColor(t *testing.T) {
	_, err := normalizeColumnColor("Teal")
	assertExitCode(t, err, errors.ExitInvalidArgs)

	if err == nil {
		t.Fatal("expected error")
	}
	for _, name := range []string{"blue", "gray", "tan", "yellow", "lime", "aqua", "violet", "purple", "pink"} {
		if !strings.Contains(err.Error(), name) {
			t.Fatalf("expected error to include %s, got %v", name, err)
		}
	}
	if !strings.Contains(err.Error(), "supported API color value") {
		t.Fatalf("expected error to clarify supported API color values, got %v", err)
	}
}

func TestColumnCreateRejectsUnknownColor(t *testing.T) {
	mock := NewMockClient()
	SetTestModeWithSDK(mock)
	SetTestConfig("token", "account", "https://api.example.com")
	defer resetTest()

	columnCreateBoard = "123"
	columnCreateName = "Test"
	columnCreateColor = "Teal"
	err := columnCreateCmd.RunE(columnCreateCmd, []string{})
	columnCreateBoard = ""
	columnCreateName = ""
	columnCreateColor = ""

	assertExitCode(t, err, errors.ExitInvalidArgs)
	if len(mock.PostCalls) != 0 {
		t.Fatalf("expected no POST calls, got %d", len(mock.PostCalls))
	}
}

func TestColumnUpdateNormalizesColorAlias(t *testing.T) {
	mock := NewMockClient()
	mock.PatchResponse = &client.APIResponse{
		StatusCode: 200,
		Data: map[string]any{
			"id":   "col-1",
			"name": "Updated Column",
			"color": map[string]any{
				"name":  "Aqua",
				"value": "var(--color-card-5)",
			},
		},
	}

	SetTestModeWithSDK(mock)
	SetTestConfig("token", "account", "https://api.example.com")
	defer resetTest()

	columnUpdateBoard = "123"
	columnUpdateColor = "AQUA"
	err := columnUpdateCmd.RunE(columnUpdateCmd, []string{"col-1"})
	columnUpdateBoard = ""
	columnUpdateColor = ""

	assertExitCode(t, err, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.PatchCalls) != 1 {
		t.Fatalf("expected 1 PATCH call, got %d", len(mock.PatchCalls))
	}
	body := mock.PatchCalls[0].Body.(map[string]any)
	if body["color"] != "var(--color-card-5)" {
		t.Fatalf("expected color 'var(--color-card-5)', got %v", body["color"])
	}
}

func TestColumnUpdateAcceptsAPIColorValue(t *testing.T) {
	mock := NewMockClient()
	mock.PatchResponse = &client.APIResponse{
		StatusCode: 200,
		Data:       map[string]any{"id": "col-1", "name": "Updated Column"},
	}

	SetTestModeWithSDK(mock)
	SetTestConfig("token", "account", "https://api.example.com")
	defer resetTest()

	columnUpdateBoard = "123"
	columnUpdateColor = "var(--color-card-8)"
	err := columnUpdateCmd.RunE(columnUpdateCmd, []string{"col-1"})
	columnUpdateBoard = ""
	columnUpdateColor = ""

	assertExitCode(t, err, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	body := mock.PatchCalls[0].Body.(map[string]any)
	if body["color"] != "var(--color-card-8)" {
		t.Fatalf("expected color 'var(--color-card-8)', got %v", body["color"])
	}
}

func TestColumnUpdateRejectsUnknownColor(t *testing.T) {
	mock := NewMockClient()
	SetTestModeWithSDK(mock)
	SetTestConfig("token", "account", "https://api.example.com")
	defer resetTest()

	columnUpdateBoard = "123"
	columnUpdateColor = "Teal"
	err := columnUpdateCmd.RunE(columnUpdateCmd, []string{"col-1"})
	columnUpdateBoard = ""
	columnUpdateColor = ""

	assertExitCode(t, err, errors.ExitInvalidArgs)
	if len(mock.PatchCalls) != 0 {
		t.Fatalf("expected no PATCH calls, got %d", len(mock.PatchCalls))
	}
}

func TestColumnColorHelpIncludesLowercaseFriendlyNames(t *testing.T) {
	for _, cmd := range []string{"create", "update"} {
		t.Run(cmd, func(t *testing.T) {
			var flagUsage string
			switch cmd {
			case "create":
				flagUsage = columnCreateCmd.Flags().Lookup("color").Usage
			case "update":
				flagUsage = columnUpdateCmd.Flags().Lookup("color").Usage
			}

			for _, name := range []string{"blue", "gray", "tan", "yellow", "lime", "aqua", "violet", "purple", "pink"} {
				if !strings.Contains(flagUsage, name) {
					t.Fatalf("expected --color help to include %s, got %q", name, flagUsage)
				}
			}
			for _, name := range []string{"Blue", "Gray", "Tan", "Yellow", "Lime", "Aqua", "Violet", "Purple", "Pink"} {
				if strings.Contains(flagUsage, name) {
					t.Fatalf("expected --color help to use lowercase aliases, got %q", flagUsage)
				}
			}
			if !strings.Contains(flagUsage, "supported API color value") {
				t.Fatalf("expected --color help to clarify supported API color values, got %q", flagUsage)
			}
		})
	}
}
