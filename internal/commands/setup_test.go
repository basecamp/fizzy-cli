package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/basecamp/fizzy-cli/internal/config"
	"gopkg.in/yaml.v3"
)

func TestParseAccounts(t *testing.T) {
	t.Run("parses accounts from identity response", func(t *testing.T) {
		data := map[string]any{
			"accounts": []any{
				map[string]any{
					"id":   "abc123",
					"name": "Acme Corp",
					"slug": "/897362094",
				},
				map[string]any{
					"id":   "def456",
					"name": "Personal",
					"slug": "/123456789",
				},
			},
		}

		accounts, err := parseAccounts(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(accounts) != 2 {
			t.Fatalf("expected 2 accounts, got %d", len(accounts))
		}

		if accounts[0].Name != "Acme Corp" {
			t.Errorf("expected first account name 'Acme Corp', got '%s'", accounts[0].Name)
		}
		if accounts[0].Slug != "897362094" {
			t.Errorf("expected first account slug '897362094', got '%s'", accounts[0].Slug)
		}
		if accounts[1].Name != "Personal" {
			t.Errorf("expected second account name 'Personal', got '%s'", accounts[1].Name)
		}
	})

	t.Run("returns error for invalid data format", func(t *testing.T) {
		data := "invalid"
		_, err := parseAccounts(data)
		if err == nil {
			t.Error("expected error for invalid data format")
		}
	})

	t.Run("returns error when accounts key missing", func(t *testing.T) {
		data := map[string]any{
			"other": "data",
		}
		_, err := parseAccounts(data)
		if err == nil {
			t.Error("expected error when accounts key missing")
		}
	})

	t.Run("handles accounts without slug", func(t *testing.T) {
		data := map[string]any{
			"accounts": []any{
				map[string]any{
					"id":   "abc123",
					"name": "No Slug Account",
				},
				map[string]any{
					"id":   "def456",
					"name": "Has Slug",
					"slug": "/123",
				},
			},
		}

		accounts, err := parseAccounts(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Only account with slug should be included
		if len(accounts) != 1 {
			t.Fatalf("expected 1 account (with slug), got %d", len(accounts))
		}
		if accounts[0].Name != "Has Slug" {
			t.Errorf("expected account name 'Has Slug', got '%s'", accounts[0].Name)
		}
	})
}

func TestParseBoards(t *testing.T) {
	t.Run("parses boards from boards response", func(t *testing.T) {
		data := []any{
			map[string]any{
				"id":   "board1",
				"name": "Engineering",
			},
			map[string]any{
				"id":   "board2",
				"name": "Marketing",
			},
		}

		boards, err := parseBoards(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(boards) != 2 {
			t.Fatalf("expected 2 boards, got %d", len(boards))
		}

		if boards[0].Name != "Engineering" {
			t.Errorf("expected first board name 'Engineering', got '%s'", boards[0].Name)
		}
		if boards[0].ID != "board1" {
			t.Errorf("expected first board ID 'board1', got '%s'", boards[0].ID)
		}
	})

	t.Run("returns error for invalid data format", func(t *testing.T) {
		data := "invalid"
		_, err := parseBoards(data)
		if err == nil {
			t.Error("expected error for invalid data format")
		}
	})

	t.Run("handles empty boards list", func(t *testing.T) {
		data := []any{}

		boards, err := parseBoards(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(boards) != 0 {
			t.Errorf("expected 0 boards, got %d", len(boards))
		}
	})

	t.Run("skips boards without id or name", func(t *testing.T) {
		data := []any{
			map[string]any{
				"id": "board1",
				// missing name
			},
			map[string]any{
				"name": "No ID Board",
				// missing id
			},
			map[string]any{
				"id":   "board3",
				"name": "Valid Board",
			},
		}

		boards, err := parseBoards(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(boards) != 1 {
			t.Fatalf("expected 1 valid board, got %d", len(boards))
		}
		if boards[0].Name != "Valid Board" {
			t.Errorf("expected board name 'Valid Board', got '%s'", boards[0].Name)
		}
	})
}

func TestValidateToken(t *testing.T) {
	t.Run("returns accounts on successful validation", func(t *testing.T) {
		mock := NewMockClient().WithGetData(map[string]any{
			"accounts": []any{
				map[string]any{
					"id":   "abc123",
					"name": "Test Account",
					"slug": "/123456",
				},
			},
		})

		// We need to use the mock client, but validateToken creates its own client
		// This test verifies the parsing logic works when given valid data
		// The actual API call is tested via e2e tests

		// For now, test that parseAccounts works correctly
		accounts, err := parseAccounts(mock.GetResponse.Data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(accounts) != 1 {
			t.Fatalf("expected 1 account, got %d", len(accounts))
		}
	})
}

func TestSaveLocal(t *testing.T) {
	t.Run("saves config to local .fizzy.yaml file", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "fizzy-test-*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tempDir)

		config.SetTestWorkingDir(tempDir)
		defer config.ResetTestWorkingDir()

		cfg := &config.Config{
			Token:   "test-token",
			Account: "123456",
			Board:   "board-id",
			APIURL:  "https://custom.fizzy.do",
		}

		err = cfg.SaveLocal()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify file was created
		configPath := filepath.Join(tempDir, ".fizzy.yaml")
		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("config file not created: %v", err)
		}

		var savedConfig config.Config
		if err := yaml.Unmarshal(data, &savedConfig); err != nil {
			t.Fatalf("failed to parse config: %v", err)
		}

		if savedConfig.Token != "test-token" {
			t.Errorf("expected token 'test-token', got '%s'", savedConfig.Token)
		}
		if savedConfig.Account != "123456" {
			t.Errorf("expected account '123456', got '%s'", savedConfig.Account)
		}
		if savedConfig.Board != "board-id" {
			t.Errorf("expected board 'board-id', got '%s'", savedConfig.Board)
		}
		if savedConfig.APIURL != "https://custom.fizzy.do" {
			t.Errorf("expected api_url 'https://custom.fizzy.do', got '%s'", savedConfig.APIURL)
		}
	})

	t.Run("file has correct permissions", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "fizzy-test-*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tempDir)

		config.SetTestWorkingDir(tempDir)
		defer config.ResetTestWorkingDir()

		cfg := &config.Config{
			Token: "test-token",
		}

		err = cfg.SaveLocal()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		configPath := filepath.Join(tempDir, ".fizzy.yaml")
		info, err := os.Stat(configPath)
		if err != nil {
			t.Fatalf("failed to stat config file: %v", err)
		}

		// Check that file is not world-readable (0600)
		perm := info.Mode().Perm()
		if perm != 0600 {
			t.Errorf("expected file permissions 0600, got %o", perm)
		}
	})
}
