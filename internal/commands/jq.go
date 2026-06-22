package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/basecamp/fizzy-cli/internal/errors"
	"github.com/itchyny/gojq"
)

// jqWriter wraps an io.Writer and applies a compiled jq filter to JSON output.
// Non-JSON writes pass through unchanged.
type jqWriter struct {
	dest io.Writer
	code *gojq.Code
}

// newJQWriter parses and compiles the jq expression and returns a filtering writer.
// Delegates to compileJQ for compilation so options are maintained in one place.
func newJQWriter(dest io.Writer, filter string) (*jqWriter, error) {
	code, err := compileJQ(filter)
	if err != nil {
		return nil, err
	}
	return &jqWriter{dest: dest, code: code}, nil
}

// newJQWriterWithCode creates a jqWriter using a pre-compiled *gojq.Code.
// Used when the expression has already been validated and compiled (e.g. in PersistentPreRunE).
func newJQWriterWithCode(dest io.Writer, code *gojq.Code) *jqWriter {
	return &jqWriter{dest: dest, code: code}
}

// compileJQ parses and compiles a jq expression, returning the compiled code.
func compileJQ(filter string) (*gojq.Code, error) {
	query, err := gojq.Parse(filter)
	if err != nil {
		return nil, errors.ErrJQValidation(err)
	}
	code, err := gojq.Compile(query, gojq.WithEnvironLoader(os.Environ))
	if err != nil {
		return nil, errors.ErrJQValidation(err)
	}
	return code, nil
}

// Write intercepts JSON output, applies the jq filter, and writes filtered results.
// String results print as plain text; everything else prints as compact single-line JSON.
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

	iter := w.code.Run(input)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, isErr := v.(error); isErr {
			return 0, errors.ErrJQRuntime(err)
		}
		if s, isStr := v.(string); isStr {
			if _, err := fmt.Fprintln(w.dest, s); err != nil {
				return 0, err
			}
		} else {
			raw, err := json.Marshal(v)
			if err != nil {
				return 0, errors.ErrJQRuntime(fmt.Errorf("result not serializable: %w", err))
			}
			if _, err := fmt.Fprintln(w.dest, string(raw)); err != nil {
				return 0, err
			}
		}
	}
	return len(p), nil
}
