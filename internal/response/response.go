// Package response handles JSON response formatting for the Fizzy CLI.
package response

import (
	"bytes"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"os"
	"time"

	"github.com/basecamp/fizzy-cli/internal/errors"
)

// prettyPrint controls whether JSON output is indented.
var prettyPrint bool

// SetPrettyPrint enables or disables pretty-printed JSON output.
func SetPrettyPrint(enabled bool) {
	prettyPrint = enabled
}

// Response represents the JSON response envelope.
type Response struct {
	Success     bool           `json:"success"`
	Data        any            `json:"data,omitempty"`
	Error       *ErrorDetail   `json:"error,omitempty"`
	Pagination  *Pagination    `json:"pagination,omitempty"`
	Breadcrumbs []Breadcrumb   `json:"breadcrumbs,omitempty"`
	Location    string         `json:"location,omitempty"`
	Summary     string         `json:"summary,omitempty"`
	Meta        map[string]any `json:"meta,omitempty"`
}

// ErrorDetail represents an error in the response.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"status,omitempty"`
	Details any    `json:"details,omitempty"`
}

// Pagination represents pagination info in the response.
type Pagination struct {
	HasNext bool   `json:"has_next"`
	NextURL string `json:"next_url,omitempty"`
}

// Breadcrumb represents a suggested next action.
type Breadcrumb struct {
	Action      string `json:"action"`
	Cmd         string `json:"cmd"`
	Description string `json:"description"`
}

// NewBreadcrumb creates a new breadcrumb.
func NewBreadcrumb(action, cmd, description string) Breadcrumb {
	return Breadcrumb{
		Action:      action,
		Cmd:         cmd,
		Description: description,
	}
}

// Success creates a successful response with data.
func Success(data any) *Response {
	return &Response{
		Success: true,
		Data:    data,
		Meta:    createMeta(),
	}
}

// SuccessWithLocation creates a successful response with location.
func SuccessWithLocation(data any, location string) *Response {
	return &Response{
		Success:  true,
		Data:     data,
		Location: location,
		Meta:     createMeta(),
	}
}

// SuccessWithPagination creates a successful response with pagination.
func SuccessWithPagination(data any, hasNext bool, nextURL string) *Response {
	resp := &Response{
		Success: true,
		Data:    data,
		Meta:    createMeta(),
	}
	if hasNext || nextURL != "" {
		resp.Pagination = &Pagination{
			HasNext: hasNext,
			NextURL: nextURL,
		}
	}
	return resp
}

// SuccessWithSummary creates a successful response with a summary.
func SuccessWithSummary(data any, summary string) *Response {
	return &Response{
		Success: true,
		Data:    data,
		Summary: summary,
		Meta:    createMeta(),
	}
}

// SuccessWithPaginationAndSummary creates a successful response with pagination and summary.
func SuccessWithPaginationAndSummary(data any, hasNext bool, nextURL string, summary string) *Response {
	resp := &Response{
		Success: true,
		Data:    data,
		Summary: summary,
		Meta:    createMeta(),
	}
	if hasNext || nextURL != "" {
		resp.Pagination = &Pagination{
			HasNext: hasNext,
			NextURL: nextURL,
		}
	}
	return resp
}

// SuccessWithBreadcrumbs creates a successful response with breadcrumbs.
func SuccessWithBreadcrumbs(data any, summary string, breadcrumbs []Breadcrumb) *Response {
	return &Response{
		Success:     true,
		Data:        data,
		Summary:     summary,
		Breadcrumbs: breadcrumbs,
		Meta:        createMeta(),
	}
}

// SuccessWithPaginationAndBreadcrumbs creates a successful response with pagination, summary, and breadcrumbs.
func SuccessWithPaginationAndBreadcrumbs(data any, hasNext bool, nextURL string, summary string, breadcrumbs []Breadcrumb) *Response {
	resp := &Response{
		Success:     true,
		Data:        data,
		Summary:     summary,
		Breadcrumbs: breadcrumbs,
		Meta:        createMeta(),
	}
	if hasNext || nextURL != "" {
		resp.Pagination = &Pagination{
			HasNext: hasNext,
			NextURL: nextURL,
		}
	}
	return resp
}

// Error creates an error response from a CLIError.
func Error(err *errors.CLIError) *Response {
	resp := &Response{
		Success: false,
		Error: &ErrorDetail{
			Code:    err.Code,
			Message: err.Message,
		},
		Meta: createMeta(),
	}
	if err.Status != 0 {
		resp.Error.Status = err.Status
	}
	return resp
}

// ErrorFromError creates an error response from a generic error.
func ErrorFromError(err error) *Response {
	if cliErr, ok := stderrors.AsType[*errors.CLIError](err); ok {
		return Error(cliErr)
	}
	return Error(errors.NewError(err.Error()))
}

func createMeta() map[string]any {
	return map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
}

// Print outputs the response as JSON to stdout.
func (r *Response) Print() {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	if prettyPrint {
		encoder.SetIndent("", "  ")
	}
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(r); err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling response: %v\n", err)
		return
	}
	fmt.Print(buf.String())
}

// PrintAndExit prints the response and exits with appropriate code.
func (r *Response) PrintAndExit() {
	r.Print()
	if r.Success {
		os.Exit(errors.ExitSuccess)
	}
	// Try to get exit code from the error
	if r.Error != nil {
		switch r.Error.Code {
		case "AUTH_ERROR":
			os.Exit(errors.ExitAuthFailure)
		case "FORBIDDEN":
			os.Exit(errors.ExitForbidden)
		case "NOT_FOUND":
			os.Exit(errors.ExitNotFound)
		case "VALIDATION_ERROR":
			os.Exit(errors.ExitValidation)
		case "NETWORK_ERROR":
			os.Exit(errors.ExitNetwork)
		case "INVALID_ARGS":
			os.Exit(errors.ExitInvalidArgs)
		default:
			os.Exit(errors.ExitError)
		}
	}
	os.Exit(errors.ExitError)
}

// ExitCode returns the appropriate exit code for this response.
func (r *Response) ExitCode() int {
	if r.Success {
		return errors.ExitSuccess
	}
	if r.Error != nil {
		switch r.Error.Code {
		case "AUTH_ERROR":
			return errors.ExitAuthFailure
		case "FORBIDDEN":
			return errors.ExitForbidden
		case "NOT_FOUND":
			return errors.ExitNotFound
		case "VALIDATION_ERROR":
			return errors.ExitValidation
		case "NETWORK_ERROR":
			return errors.ExitNetwork
		case "INVALID_ARGS":
			return errors.ExitInvalidArgs
		}
	}
	return errors.ExitError
}
