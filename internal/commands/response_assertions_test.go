package commands

import "testing"

func responseDataMap(t *testing.T, result *CommandResult) map[string]any {
	t.Helper()
	if result == nil || result.Response == nil {
		t.Fatal("expected command response")
	}
	data, ok := result.Response.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected response data map, got %#v", result.Response.Data)
	}
	return data
}
