package clitests

import (
	"testing"

	"github.com/basecamp/fizzy-cli/e2e/harness"
)

func TestSearch(t *testing.T) {
	h := newHarness(t)
	// Single-token query
	assertOK(t, h.Run("search", "test"))
	// Multi-arg joined into a single q string
	assertOK(t, h.Run("search", "login", "error"))
}

func TestCardListWithSearch(t *testing.T) {
	// Filter use cases that used to live on `fizzy search` now belong here.
	h := newHarness(t)
	assertOK(t, h.Run("card", "list", "--search", "test", "--board", fixture.BoardID))
	assertOK(t, h.Run("card", "list", "--search", "test", "--all"))
}

func TestAuthInvalidToken(t *testing.T) {
	badCfg := *cfg
	badCfg.Token = "fizzy_invalid_token"
	h := harness.NewWithConfig(t, &badCfg)
	assertResult(t, h.Run("board", "list"), harness.ExitAuthFailure)
}

func TestAuthMissingToken(t *testing.T) {
	missingCfg := *cfg
	missingCfg.Token = ""
	h := harness.NewWithConfig(t, &missingCfg)
	assertResult(t, h.RunWithEnv(map[string]string{
		"FIZZY_TOKEN":   "",
		"FIZZY_ACCOUNT": "",
	}, "board", "list"), harness.ExitAuthFailure)
}
