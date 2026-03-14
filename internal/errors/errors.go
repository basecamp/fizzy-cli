// Package errors defines error types and exit codes for the Fizzy CLI.
// Types and constants are thin re-exports from the shared output package.
package errors

import (
	"errors"
	"fmt"

	"github.com/basecamp/cli/output"
)

// CLIError is a type alias for output.Error.
// Breaking change: ExitCode is now a method, not a field.
type CLIError = output.Error

// Exit codes aligned to the shared rubric.
const (
	ExitSuccess   = output.ExitOK        // 0
	ExitUsage     = output.ExitUsage     // 1
	ExitNotFound  = output.ExitNotFound  // 2
	ExitAuth      = output.ExitAuth      // 3
	ExitForbidden = output.ExitForbidden // 4
	ExitRateLimit = output.ExitRateLimit // 5
	ExitNetwork   = output.ExitNetwork   // 6
	ExitAPI       = output.ExitAPI       // 7
	ExitAmbiguous = output.ExitAmbiguous // 8

	// Deprecated aliases — kept for compilation, values change.
	ExitError       = output.ExitAPI   // was 1, now 7
	ExitInvalidArgs = output.ExitUsage // was 2, now 1
	ExitAuthFailure = output.ExitAuth  // was 3, stays 3
	ExitValidation  = output.ExitAPI   // was 6, now 7
)

// NewError creates a general API error.
func NewError(message string) *CLIError {
	return &output.Error{Code: output.CodeAPI, Message: message}
}

// NewInvalidArgsError creates a usage/invalid-arguments error.
func NewInvalidArgsError(message string) *CLIError {
	return &output.Error{Code: output.CodeUsage, Message: message}
}

// NewAuthError creates an authentication error with a fizzy-specific hint.
func NewAuthError(message string) *CLIError {
	e := output.ErrAuth(message)
	e.Hint = "Run 'fizzy auth login TOKEN' or set FIZZY_TOKEN"
	return e
}

// NewForbiddenError creates a permission denied error.
func NewForbiddenError(message string) *CLIError {
	return &output.Error{Code: output.CodeForbidden, Message: message, HTTPStatus: 403}
}

// NewNotFoundError creates a not found error.
func NewNotFoundError(message string) *CLIError {
	return &output.Error{Code: output.CodeNotFound, Message: message, HTTPStatus: 404}
}

// NewValidationError creates a validation error (mapped to API error).
func NewValidationError(message string) *CLIError {
	return &output.Error{Code: output.CodeAPI, Message: message, HTTPStatus: 422}
}

// NewNetworkError creates a network error with retryable hint.
func NewNetworkError(message string) *CLIError {
	e := output.ErrNetwork(fmt.Errorf("%s", message))
	return e
}

// errJQ is a sentinel cause for all jq-related errors (validation,
// unsupported command, flag conflict, and runtime failures).
// root.go uses IsJQError() to detect these and bypass jq filtering
// when rendering the error itself.
var errJQ = errors.New("jq error")

// ErrJQValidation returns a usage error for invalid --jq expressions.
func ErrJQValidation(cause error) *CLIError {
	return &output.Error{
		Code:    output.CodeUsage,
		Message: fmt.Sprintf("invalid --jq expression: %s", cause),
		Cause:   errJQ,
	}
}

// ErrJQNotSupported returns a usage error for commands that don't support --jq.
func ErrJQNotSupported(command string) *CLIError {
	return &output.Error{
		Code:    output.CodeUsage,
		Message: fmt.Sprintf("--jq is not supported by %s", command),
		Cause:   errJQ,
	}
}

// ErrJQConflict returns a usage error for flags that conflict with --jq.
func ErrJQConflict(flag string) *CLIError {
	return &output.Error{
		Code:    output.CodeUsage,
		Message: fmt.Sprintf("cannot use --jq with %s", flag),
		Cause:   errJQ,
	}
}

// ErrJQRuntime returns a usage error for jq runtime failures
// (e.g. type errors, non-serializable results).
func ErrJQRuntime(cause error) *CLIError {
	return &output.Error{
		Code:    output.CodeUsage,
		Message: fmt.Sprintf("jq filter error: %s", cause),
		Cause:   errJQ,
	}
}

// IsJQError returns true if the error is a jq-related error
// (validation failure, unsupported command, flag conflict, or runtime failure).
func IsJQError(err error) bool {
	return errors.Is(err, errJQ)
}

// FromHTTPStatus creates an appropriate error from an HTTP status code.
func FromHTTPStatus(status int, message string) *CLIError {
	switch status {
	case 401:
		e := output.ErrAuth(message)
		e.Hint = "Run 'fizzy auth login TOKEN' or set FIZZY_TOKEN"
		return e
	case 403:
		return NewForbiddenError(message)
	case 404:
		return NewNotFoundError(message)
	case 422:
		return NewValidationError(message)
	case 429:
		e := output.ErrRateLimit(0)
		if message != "" {
			e.Message = message
		}
		return e
	case 502, 503, 504:
		return &output.Error{
			Code:       output.CodeAPI,
			Message:    fmt.Sprintf("Request failed: %d %s", status, message),
			HTTPStatus: status,
			Retryable:  true,
			Hint:       "The server returned a temporary error. Try again in a moment.",
		}
	default:
		return &output.Error{
			Code:       output.CodeAPI,
			Message:    fmt.Sprintf("Request failed: %d %s", status, message),
			HTTPStatus: status,
		}
	}
}
