// Package render provides styled and markdown output rendering for the Fizzy CLI.
// The shared output.Writer falls back to JSON for FormatStyled/FormatMarkdown;
// this package supplies the app-specific rendering.
package render

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

// Column maps a display header to a JSON field name.
type Column struct {
	Header string
	Field  string
}

// Columns is a slice of Column specs for table rendering.
type Columns []Column

// headerStyle is the style for table headers in styled output.
var headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")).PaddingRight(1)

// cellStyle is the style for table data cells in styled output.
var cellStyle = lipgloss.NewStyle().PaddingRight(1)

// StyledList renders a slice of maps as a styled terminal table.
func StyledList(data []map[string]any, cols Columns, summary string) string {
	if len(data) == 0 {
		if summary != "" {
			return summary + "\n"
		}
		return "No results.\n"
	}

	headers := make([]string, len(cols))
	for i, c := range cols {
		headers[i] = c.Header
	}

	rows := make([][]string, 0, len(data))
	for _, item := range data {
		row := make([]string, len(cols))
		for i, c := range cols {
			row[i] = extractString(item, c.Field)
		}
		rows = append(rows, row)
	}

	t := table.New().
		Headers(headers...).
		Rows(rows...).
		BorderLeft(false).
		BorderRight(false).
		BorderTop(false).
		BorderBottom(false).
		BorderColumn(false).
		BorderHeader(true).
		Border(lipgloss.NormalBorder()).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			return cellStyle
		})

	var sb strings.Builder
	if summary != "" {
		sb.WriteString(summary)
		sb.WriteString("\n\n")
	}
	sb.WriteString(t.String())
	sb.WriteString("\n")
	return sb.String()
}

// StyledDetail renders a single map as styled key-value pairs.
func StyledDetail(data map[string]any, summary string) string {
	if data == nil {
		return "No data.\n"
	}

	keys := sortedKeys(data)

	var sb strings.Builder
	if summary != "" {
		sb.WriteString(summary)
		sb.WriteString("\n\n")
	}

	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	for _, k := range keys {
		label := labelStyle.Render(k + ":")
		val := formatValue(data[k])
		fmt.Fprintf(&sb, "%s %s\n", label, val)
	}
	return sb.String()
}

// extractString gets a field value from a map and converts it to a display string.
// Supports dot-separated paths for nested fields (e.g., "column.name").
func extractString(m map[string]any, field string) string {
	parts := strings.SplitN(field, ".", 2)
	val, ok := m[parts[0]]
	if !ok {
		return ""
	}

	if len(parts) == 2 {
		if nested, ok := val.(map[string]any); ok {
			return extractString(nested, parts[1])
		}
		return ""
	}

	return formatValue(val)
}

// formatValue converts any value to a display string.
func formatValue(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case bool:
		if val {
			return "yes"
		}
		return "no"
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	case map[string]any:
		// Try common display fields
		for _, key := range []string{"name", "title", "slug", "id"} {
			if s, ok := val[key]; ok {
				return formatValue(s)
			}
		}
		return fmt.Sprintf("%v", val)
	case []any:
		return fmt.Sprintf("[%d items]", len(val))
	default:
		return fmt.Sprintf("%v", val)
	}
}

// StyledSummary renders a summary message for mutations.
// If structured data is present, include it below the summary for human readability.
func StyledSummary(data map[string]any, summary string) string {
	if summary != "" {
		line := lipgloss.NewStyle().Bold(true).Render("✓ " + summary)
		if len(data) == 0 {
			return line + "\n"
		}
		return line + "\n\n" + StyledDetail(data, "")
	}
	if len(data) == 0 {
		return lipgloss.NewStyle().Bold(true).Render("✓ Done") + "\n"
	}
	return StyledDetail(data, "")
}

func sortedKeys(m map[string]any) []string {
	// Put common identifying fields first, then alphabetical
	priority := map[string]int{
		"id": 0, "number": 1, "name": 2, "title": 3,
		"email": 4, "status": 5,
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		pi, oki := priority[keys[i]]
		pj, okj := priority[keys[j]]
		if oki && okj {
			return pi < pj
		}
		if oki {
			return true
		}
		if okj {
			return false
		}
		return keys[i] < keys[j]
	})
	return keys
}
