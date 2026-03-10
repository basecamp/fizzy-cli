package commands

import (
	"errors"
	"net"

	"github.com/basecamp/cli/output"
	fizzy "github.com/basecamp/fizzy-sdk/go/pkg/fizzy"
)

// convertSDKError converts an SDK error to a CLI output.Error.
// Returns nil for nil errors. Non-SDK errors pass through unchanged.
func convertSDKError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, fizzy.ErrCircuitOpen) {
		return &output.Error{
			Code:      output.CodeAPI,
			Message:   "Service temporarily unavailable (circuit breaker open)",
			Hint:      "Try again in a moment",
			Retryable: true,
		}
	}
	if errors.Is(err, fizzy.ErrBulkheadFull) {
		return &output.Error{
			Code:      output.CodeAPI,
			Message:   "Too many concurrent requests",
			Hint:      "Try again in a moment",
			Retryable: true,
		}
	}
	if errors.Is(err, fizzy.ErrRateLimited) {
		return &output.Error{
			Code:      output.CodeRateLimit,
			Message:   "Rate limit exceeded",
			Hint:      "Try again later",
			Retryable: true,
		}
	}

	var sdkErr *fizzy.Error
	if errors.As(err, &sdkErr) {
		e := &output.Error{
			Code:       mapSDKCode(sdkErr.Code),
			Message:    sdkErr.Message,
			Hint:       sdkErr.Hint,
			HTTPStatus: sdkErr.HTTPStatus,
			Retryable:  sdkErr.Retryable,
		}
		// Add fizzy-specific hint for auth errors
		if sdkErr.Code == fizzy.CodeAuth && e.Hint == "" {
			e.Hint = "Run 'fizzy auth login TOKEN' or set FIZZY_TOKEN"
		}
		return e
	}

	// Catch raw network errors that weren't wrapped by the SDK
	var netErr net.Error
	if errors.As(err, &netErr) {
		return &output.Error{
			Code:      output.CodeNetwork,
			Message:   netErr.Error(),
			Hint:      "Check your internet connection",
			Retryable: true,
		}
	}

	return err
}

// mapSDKCode maps SDK error codes to output package codes.
// Codes that exist in both packages pass through; SDK-only codes
// (e.g. "validation") are mapped to the closest output equivalent.
func mapSDKCode(code string) string {
	switch code {
	case fizzy.CodeValidation:
		return output.CodeAPI // output package has no "validation" code
	default:
		return code
	}
}
