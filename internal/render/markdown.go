package render

import (
	"fmt"
	"strings"
)

// MarkdownList renders a slice of maps as a GFM table.
func MarkdownList(data []map[string]any, cols Columns, summary string) string {
	if len(data) == 0 {
		if summary != "" {
			return summary + "\n"
		}
		return "No results.\n"
	}

	var sb strings.Builder
	if summary != "" {
		sb.WriteString(summary)
		sb.WriteString("\n\n")
	}

	// Header row
	headers := make([]string, len(cols))
	seps := make([]string, len(cols))
	for i, c := range cols {
		headers[i] = c.Header
		seps[i] = "---"
	}
	sb.WriteString("| ")
	sb.WriteString(strings.Join(headers, " | "))
	sb.WriteString(" |\n")
	sb.WriteString("| ")
	sb.WriteString(strings.Join(seps, " | "))
	sb.WriteString(" |\n")

	// Data rows
	for _, item := range data {
		vals := make([]string, len(cols))
		for i, c := range cols {
			vals[i] = escapeMarkdown(extractString(item, c.Field))
		}
		sb.WriteString("| ")
		sb.WriteString(strings.Join(vals, " | "))
		sb.WriteString(" |\n")
	}
	return sb.String()
}

// MarkdownDetail renders a single map as bold-label: value pairs.
func MarkdownDetail(data map[string]any, summary string) string {
	if data == nil {
		return "No data.\n"
	}

	keys := sortedKeys(data)

	var sb strings.Builder
	if summary != "" {
		sb.WriteString(summary)
		sb.WriteString("\n\n")
	}

	for _, k := range keys {
		val := formatValue(data[k])
		fmt.Fprintf(&sb, "**%s:** %s\n", k, escapeMarkdown(val))
	}
	return sb.String()
}

// MarkdownSummary renders a single-line summary message for mutations.
func MarkdownSummary(data map[string]any, summary string) string {
	if summary != "" {
		return fmt.Sprintf("> %s\n", summary)
	}
	if data == nil {
		return "> Done\n"
	}
	return MarkdownDetail(data, "")
}

// escapeMarkdown escapes pipe characters in markdown table cells.
func escapeMarkdown(s string) string {
	return strings.ReplaceAll(s, "|", "\\|")
}
