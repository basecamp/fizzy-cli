package commands

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/itchyny/gojq"
)

// jqWriter wraps an io.Writer and applies a compiled jq filter to JSON output.
// Non-JSON writes pass through unchanged.
type jqWriter struct {
	dest  io.Writer
	query *gojq.Query
}

// newJQWriter parses the jq expression and returns a filtering writer.
func newJQWriter(dest io.Writer, filter string) (*jqWriter, error) {
	query, err := gojq.Parse(filter)
	if err != nil {
		return nil, fmt.Errorf("invalid jq expression: %w", err)
	}
	return &jqWriter{dest: dest, query: query}, nil
}

// Write intercepts JSON output, applies the jq filter, and writes filtered results.
// String results print as plain text; everything else prints as indented JSON.
// Error envelopes (ok: false) pass through unfiltered so error messages are never hidden.
func (w *jqWriter) Write(p []byte) (int, error) {
	var input any
	if err := json.Unmarshal(p, &input); err != nil {
		// Not JSON — pass through unchanged.
		return w.dest.Write(p)
	}

	// Pass through error envelopes unfiltered so jq doesn't hide error messages.
	if m, ok := input.(map[string]any); ok {
		if okVal, exists := m["ok"]; exists {
			if okBool, isBool := okVal.(bool); isBool && !okBool {
				return w.dest.Write(p)
			}
		}
	}

	iter := w.query.Run(input)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, isErr := v.(error); isErr {
			return 0, fmt.Errorf("jq: %w", err)
		}
		if s, isStr := v.(string); isStr {
			if _, err := fmt.Fprintln(w.dest, s); err != nil {
				return 0, err
			}
		} else {
			enc := json.NewEncoder(w.dest)
			enc.SetIndent("", "  ")
			if err := enc.Encode(v); err != nil {
				return 0, err
			}
		}
	}
	return len(p), nil
}
