package render

import (
	"strings"
	"testing"
)

func TestMarkdownListEmpty(t *testing.T) {
	result := MarkdownList(nil, Columns{{Header: "ID", Field: "id"}}, "")
	if result != "No results.\n" {
		t.Errorf("expected 'No results.', got %q", result)
	}
}

func TestMarkdownListRendersTable(t *testing.T) {
	data := []map[string]any{
		{"id": "1", "name": "Alpha"},
		{"id": "2", "name": "Beta"},
	}
	cols := Columns{
		{Header: "ID", Field: "id"},
		{Header: "Name", Field: "name"},
	}
	result := MarkdownList(data, cols, "2 boards")
	if !strings.Contains(result, "| ID | Name |") {
		t.Errorf("expected header row, got:\n%s", result)
	}
	if !strings.Contains(result, "| --- | --- |") {
		t.Errorf("expected separator row, got:\n%s", result)
	}
	if !strings.Contains(result, "| 1 | Alpha |") {
		t.Errorf("expected data row for Alpha, got:\n%s", result)
	}
	if !strings.Contains(result, "| 2 | Beta |") {
		t.Errorf("expected data row for Beta, got:\n%s", result)
	}
	if !strings.Contains(result, "2 boards") {
		t.Error("expected summary in output")
	}
}

func TestMarkdownListEscapesPipes(t *testing.T) {
	data := []map[string]any{
		{"name": "foo|bar"},
	}
	cols := Columns{{Header: "Name", Field: "name"}}
	result := MarkdownList(data, cols, "")
	if strings.Contains(result, "foo|bar") {
		t.Error("pipe character should be escaped")
	}
	if !strings.Contains(result, `foo\|bar`) {
		t.Errorf("expected escaped pipe, got:\n%s", result)
	}
}

func TestMarkdownDetailNil(t *testing.T) {
	result := MarkdownDetail(nil, "")
	if result != "No data.\n" {
		t.Errorf("expected 'No data.', got %q", result)
	}
}

func TestMarkdownDetailRendersKV(t *testing.T) {
	data := map[string]any{"id": "42", "name": "Test"}
	result := MarkdownDetail(data, "")
	if !strings.Contains(result, "**id:** 42") {
		t.Errorf("expected bold key, got:\n%s", result)
	}
	if !strings.Contains(result, "**name:** Test") {
		t.Errorf("expected bold key, got:\n%s", result)
	}
}
