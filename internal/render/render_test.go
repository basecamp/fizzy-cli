package render

import (
	"strings"
	"testing"
)

func TestStyledListEmpty(t *testing.T) {
	result := StyledList(nil, Columns{{Header: "ID", Field: "id"}}, "")
	if result != "No results.\n" {
		t.Errorf("expected 'No results.', got %q", result)
	}
}

func TestStyledListEmptyWithSummary(t *testing.T) {
	result := StyledList(nil, Columns{{Header: "ID", Field: "id"}}, "0 boards")
	if result != "0 boards\n" {
		t.Errorf("expected '0 boards', got %q", result)
	}
}

func TestStyledListRendersTable(t *testing.T) {
	data := []map[string]any{
		{"id": "1", "name": "Alpha"},
		{"id": "2", "name": "Beta"},
	}
	cols := Columns{
		{Header: "ID", Field: "id"},
		{Header: "Name", Field: "name"},
	}
	result := StyledList(data, cols, "2 boards")
	if !strings.Contains(result, "Alpha") {
		t.Error("expected output to contain 'Alpha'")
	}
	if !strings.Contains(result, "Beta") {
		t.Error("expected output to contain 'Beta'")
	}
	if !strings.Contains(result, "2 boards") {
		t.Error("expected output to contain summary")
	}
}

func TestStyledListNestedField(t *testing.T) {
	data := []map[string]any{
		{"id": "1", "column": map[string]any{"name": "Inbox"}},
	}
	cols := Columns{{Header: "Column", Field: "column.name"}}
	result := StyledList(data, cols, "")
	if !strings.Contains(result, "Inbox") {
		t.Error("expected nested field 'Inbox' in output")
	}
}

func TestStyledDetailNil(t *testing.T) {
	result := StyledDetail(nil, "")
	if result != "No data.\n" {
		t.Errorf("expected 'No data.', got %q", result)
	}
}

func TestStyledDetailRendersKV(t *testing.T) {
	data := map[string]any{"id": "42", "name": "Test Board"}
	result := StyledDetail(data, "Board: Test Board")
	if !strings.Contains(result, "42") {
		t.Error("expected output to contain '42'")
	}
	if !strings.Contains(result, "Test Board") {
		t.Error("expected output to contain 'Test Board'")
	}
}

func TestFormatValue(t *testing.T) {
	tests := []struct {
		input    any
		expected string
	}{
		{nil, ""},
		{"hello", "hello"},
		{true, "yes"},
		{false, "no"},
		{float64(42), "42"},
		{float64(3.14), "3.14"},
		{[]any{1, 2, 3}, "[3 items]"},
		{map[string]any{"name": "Foo"}, "Foo"},
		{map[string]any{"id": "123"}, "123"},
	}
	for _, tt := range tests {
		got := formatValue(tt.input)
		if got != tt.expected {
			t.Errorf("formatValue(%v) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestSortedKeys(t *testing.T) {
	m := map[string]any{
		"zzz":    1,
		"id":     2,
		"name":   3,
		"aaa":    4,
		"title":  5,
		"number": 6,
	}
	keys := sortedKeys(m)
	// Priority keys should come first: id, number, name, title, then alphabetical
	expected := []string{"id", "number", "name", "title", "aaa", "zzz"}
	for i, k := range expected {
		if keys[i] != k {
			t.Errorf("position %d: expected %q, got %q (full: %v)", i, k, keys[i], keys)
			break
		}
	}
}
