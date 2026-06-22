package errors

import (
	stderrors "errors"
	"fmt"
	"testing"

	"github.com/basecamp/cli/output"
)

func TestExitCodes(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		expected int
	}{
		{"ExitSuccess", ExitSuccess, 0},
		{"ExitUsage", ExitUsage, 1},
		{"ExitNotFound", ExitNotFound, 2},
		{"ExitAuth", ExitAuth, 3},
		{"ExitForbidden", ExitForbidden, 4},
		{"ExitRateLimit", ExitRateLimit, 5},
		{"ExitNetwork", ExitNetwork, 6},
		{"ExitAPI", ExitAPI, 7},
		{"ExitAmbiguous", ExitAmbiguous, 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.code != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, tt.code)
			}
		})
	}
}

func TestDeprecatedAliases(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		expected int
	}{
		{"ExitError", ExitError, output.ExitAPI},
		{"ExitInvalidArgs", ExitInvalidArgs, output.ExitUsage},
		{"ExitAuthFailure", ExitAuthFailure, output.ExitAuth},
		{"ExitValidation", ExitValidation, output.ExitAPI},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.code != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, tt.code)
			}
		})
	}
}

func TestCLIError_Error(t *testing.T) {
	err := &CLIError{
		Code:    "test_error",
		Message: "test message",
	}

	if err.Error() != "test message" {
		t.Errorf("expected 'test message', got '%s'", err.Error())
	}
}

func TestCLIError_ExitCode(t *testing.T) {
	err := &CLIError{Code: output.CodeAuth, Message: "unauthorized"}
	if err.ExitCode() != ExitAuth {
		t.Errorf("expected exit code %d, got %d", ExitAuth, err.ExitCode())
	}
}

func TestNewError(t *testing.T) {
	err := NewError("something went wrong")

	if err.Code != output.CodeAPI {
		t.Errorf("expected code %q, got %q", output.CodeAPI, err.Code)
	}
	if err.Message != "something went wrong" {
		t.Errorf("expected message 'something went wrong', got %q", err.Message)
	}
	if err.ExitCode() != ExitAPI {
		t.Errorf("expected exit code %d, got %d", ExitAPI, err.ExitCode())
	}
}

func TestNewAuthError(t *testing.T) {
	err := NewAuthError("invalid token")

	if err.Code != output.CodeAuth {
		t.Errorf("expected code %q, got %q", output.CodeAuth, err.Code)
	}
	if err.Message != "invalid token" {
		t.Errorf("expected message 'invalid token', got %q", err.Message)
	}
	if err.ExitCode() != ExitAuth {
		t.Errorf("expected exit code %d, got %d", ExitAuth, err.ExitCode())
	}
	if err.Hint == "" {
		t.Error("auth errors should have a hint")
	}
}

func TestNewForbiddenError(t *testing.T) {
	err := NewForbiddenError("access denied")

	if err.Code != output.CodeForbidden {
		t.Errorf("expected code %q, got %q", output.CodeForbidden, err.Code)
	}
	if err.Message != "access denied" {
		t.Errorf("expected message 'access denied', got %q", err.Message)
	}
	if err.HTTPStatus != 403 {
		t.Errorf("expected HTTP status 403, got %d", err.HTTPStatus)
	}
	if err.ExitCode() != ExitForbidden {
		t.Errorf("expected exit code %d, got %d", ExitForbidden, err.ExitCode())
	}
}

func TestNewNotFoundError(t *testing.T) {
	err := NewNotFoundError("resource not found")

	if err.Code != output.CodeNotFound {
		t.Errorf("expected code %q, got %q", output.CodeNotFound, err.Code)
	}
	if err.Message != "resource not found" {
		t.Errorf("expected message 'resource not found', got %q", err.Message)
	}
	if err.HTTPStatus != 404 {
		t.Errorf("expected HTTP status 404, got %d", err.HTTPStatus)
	}
	if err.ExitCode() != ExitNotFound {
		t.Errorf("expected exit code %d, got %d", ExitNotFound, err.ExitCode())
	}
}

func TestNewValidationError(t *testing.T) {
	err := NewValidationError("invalid input")

	if err.Code != output.CodeAPI {
		t.Errorf("expected code %q, got %q", output.CodeAPI, err.Code)
	}
	if err.Message != "invalid input" {
		t.Errorf("expected message 'invalid input', got %q", err.Message)
	}
	if err.HTTPStatus != 422 {
		t.Errorf("expected HTTP status 422, got %d", err.HTTPStatus)
	}
	if err.ExitCode() != ExitAPI {
		t.Errorf("expected exit code %d, got %d", ExitAPI, err.ExitCode())
	}
}

func TestNewNetworkError(t *testing.T) {
	err := NewNetworkError("connection failed")

	if err.Code != output.CodeNetwork {
		t.Errorf("expected code %q, got %q", output.CodeNetwork, err.Code)
	}
	if err.ExitCode() != ExitNetwork {
		t.Errorf("expected exit code %d, got %d", ExitNetwork, err.ExitCode())
	}
	if !err.Retryable {
		t.Error("network errors should be retryable")
	}
}

func TestNewInvalidArgsError(t *testing.T) {
	err := NewInvalidArgsError("missing required flag")

	if err.Code != output.CodeUsage {
		t.Errorf("expected code %q, got %q", output.CodeUsage, err.Code)
	}
	if err.Message != "missing required flag" {
		t.Errorf("expected message 'missing required flag', got %q", err.Message)
	}
	if err.ExitCode() != ExitUsage {
		t.Errorf("expected exit code %d, got %d", ExitUsage, err.ExitCode())
	}
}

// =============================================================================
// JQ Error Constructor Tests
// =============================================================================

func TestErrJQValidation(t *testing.T) {
	cause := fmt.Errorf("unexpected token")
	err := ErrJQValidation(cause)

	if err.Code != output.CodeUsage {
		t.Errorf("expected code %q, got %q", output.CodeUsage, err.Code)
	}
	if err.Message != "invalid --jq expression: unexpected token" {
		t.Errorf("unexpected message: %q", err.Message)
	}
	if !IsJQError(err) {
		t.Error("expected IsJQError to be true")
	}
}

func TestErrJQNotSupported(t *testing.T) {
	err := ErrJQNotSupported("the version command")

	if err.Code != output.CodeUsage {
		t.Errorf("expected code %q, got %q", output.CodeUsage, err.Code)
	}
	if err.Message != "--jq is not supported by the version command" {
		t.Errorf("unexpected message: %q", err.Message)
	}
	if !IsJQError(err) {
		t.Error("expected IsJQError to be true")
	}
}

func TestErrJQConflict(t *testing.T) {
	err := ErrJQConflict("--ids-only")

	if err.Code != output.CodeUsage {
		t.Errorf("expected code %q, got %q", output.CodeUsage, err.Code)
	}
	if err.Message != "cannot use --jq with --ids-only" {
		t.Errorf("unexpected message: %q", err.Message)
	}
	if !IsJQError(err) {
		t.Error("expected IsJQError to be true")
	}
}

func TestErrJQRuntime(t *testing.T) {
	cause := fmt.Errorf("division by zero")
	err := ErrJQRuntime(cause)

	if err.Code != output.CodeUsage {
		t.Errorf("expected code %q, got %q", output.CodeUsage, err.Code)
	}
	if err.Message != "jq filter error: division by zero" {
		t.Errorf("unexpected message: %q", err.Message)
	}
	if !IsJQError(err) {
		t.Error("expected IsJQError to be true")
	}
}

func TestIsJQErrorFalseForNonJQError(t *testing.T) {
	if IsJQError(NewInvalidArgsError("plain error")) {
		t.Error("expected false for non-jq usage error")
	}
	if IsJQError(stderrors.New("random error")) {
		t.Error("expected false for random error")
	}
	if IsJQError(NewNotFoundError("project not found")) {
		t.Error("expected false for not-found error")
	}
}

func TestJQErrorExitCodes(t *testing.T) {
	tests := []struct {
		name string
		err  *CLIError
	}{
		{"validation", ErrJQValidation(fmt.Errorf("test"))},
		{"not_supported", ErrJQNotSupported("test")},
		{"conflict", ErrJQConflict("--test")},
		{"runtime", ErrJQRuntime(fmt.Errorf("test"))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.ExitCode() != ExitUsage {
				t.Errorf("expected exit code %d, got %d", ExitUsage, tt.err.ExitCode())
			}
		})
	}
}

func TestFromHTTPStatus(t *testing.T) {
	tests := []struct {
		name         string
		status       int
		message      string
		expectedCode string
		expectedExit int
	}{
		{"401 Unauthorized", 401, "Unauthorized", output.CodeAuth, ExitAuth},
		{"403 Forbidden", 403, "Forbidden", output.CodeForbidden, ExitForbidden},
		{"404 Not Found", 404, "Not Found", output.CodeNotFound, ExitNotFound},
		{"422 Unprocessable", 422, "Validation failed", output.CodeAPI, ExitAPI},
		{"429 Rate Limited", 429, "Too Many Requests", output.CodeRateLimit, ExitRateLimit},
		{"500 Server Error", 500, "Internal Server Error", output.CodeAPI, ExitAPI},
		{"502 Bad Gateway", 502, "Bad Gateway", output.CodeAPI, ExitAPI},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := FromHTTPStatus(tt.status, tt.message)

			if err.Code != tt.expectedCode {
				t.Errorf("expected code %q, got %q", tt.expectedCode, err.Code)
			}
			if err.ExitCode() != tt.expectedExit {
				t.Errorf("expected exit code %d, got %d", tt.expectedExit, err.ExitCode())
			}
		})
	}
}

func TestFromHTTPStatus401HasHint(t *testing.T) {
	err := FromHTTPStatus(401, "Unauthorized")
	if err.Hint == "" {
		t.Error("401 error should have a hint")
	}
}

func TestFromHTTPStatus429IsRetryable(t *testing.T) {
	err := FromHTTPStatus(429, "Too Many Requests")
	if !err.Retryable {
		t.Error("429 error should be retryable")
	}
}
