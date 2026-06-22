package commands

import "testing"

func TestTokenListRejectsUnexpectedArgs(t *testing.T) {
	if err := tokenListCmd.Args(tokenListCmd, []string{"extra"}); err == nil {
		t.Fatal("expected token list to reject unexpected positional args")
	}
	if err := tokenListCmd.Args(tokenListCmd, []string{}); err != nil {
		t.Fatalf("expected token list to allow no positional args, got %v", err)
	}
}

func TestTokenCreateRejectsUnexpectedArgs(t *testing.T) {
	if err := tokenCreateCmd.Args(tokenCreateCmd, []string{"extra"}); err == nil {
		t.Fatal("expected token create to reject unexpected positional args")
	}
	if err := tokenCreateCmd.Args(tokenCreateCmd, []string{}); err != nil {
		t.Fatalf("expected token create to allow no positional args, got %v", err)
	}
}
