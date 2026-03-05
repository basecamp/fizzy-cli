# Fizzy CLI Development Context

## Repository Structure

```
fizzy-cli/
├── cmd/fizzy/           # Main entrypoint
├── internal/
│   ├── client/          # HTTP client wrapper
│   ├── commands/        # Command implementations
│   ├── config/          # Configuration management
│   ├── errors/          # Error handling and types
│   └── response/        # Response formatting
├── e2e/                 # Go integration tests
├── skills/              # Agent skills
└── .claude-plugin/      # Claude Code integration
```

## Fizzy API Reference

API documentation: https://github.com/basecamp/fizzy/blob/main/docs/API.md

Key endpoints used by the CLI:
- `/boards.json` - List boards
- `/cards.json?board_ids[]=<id>` - Cards on a board
- `/cards/{number}.json` - Show card by number
- `/cards.json?terms[]=<query>` - Search cards by text

**Important:** Cards use NUMBER for CLI commands, not internal ID. `fizzy card show 42` uses the card number.

## Testing

```bash
make build            # Build binary to ./bin/fizzy
make test-unit        # Run Go unit tests (no API required)
make test-e2e         # Run e2e tests (requires credentials)
make test-run NAME=TestBoardCRUD  # Run a specific test
```

Requirements: Go 1.26+, API credentials for e2e tests.

E2E environment variables:
- `FIZZY_TEST_TOKEN` - API token (required)
- `FIZZY_TEST_ACCOUNT` - Account slug (required)
- `FIZZY_TEST_USER_ID` - User ID for user tests (optional)

## Configuration

The CLI reads config from multiple sources with this priority:
1. CLI flags (`--token`, `--account`, `--api-url`, `--board`)
2. Environment variables (`FIZZY_TOKEN`, `FIZZY_ACCOUNT`, `FIZZY_API_URL`, `FIZZY_BOARD`)
3. Local project config (`.fizzy.yaml`)
4. Global config (`~/.config/fizzy/config.yaml` or `~/.fizzy/config.yaml`)

## Authentication

Token-based via personal access tokens. Run `fizzy setup` for interactive configuration or `fizzy auth login` to save a token directly.
