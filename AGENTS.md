# Fizzy CLI Development Context

## Repository Structure

```
fizzy-cli/
├── cmd/fizzy/           # Main entrypoint
├── internal/
│   ├── client/          # Legacy HTTP client (upload, download, multipart, migrate)
│   ├── commands/        # Command implementations
│   ├── config/          # Configuration management
│   ├── errors/          # Error handling and types
│   └── render/          # Output rendering (styled, markdown, columns)
├── e2e/                 # Go integration tests
├── skills/              # Agent skills
└── .claude-plugin/      # Claude Code integration
```

## SDK Architecture

Commands use the fizzy-sdk (`github.com/basecamp/fizzy-sdk/go/pkg/fizzy`) for API access:

- **`getSDK()`** returns `*fizzy.AccountClient` — account-scoped SDK client for most commands
- **`getSDKClient()`** returns `*fizzy.Client` — root client for account-independent operations (e.g. identity)
- **`getClient()`** (deprecated) returns `client.API` — legacy client, used only for file upload, download, multipart PATCH, and board migration

SDK service methods return typed structs (e.g. `*generated.Board`, `[]generated.Card`). Helper functions convert to the `any` types the output layer expects:
- **`normalizeAny(data)`** — JSON-round-trips any value (typed structs, `json.RawMessage`, maps) to `map[string]any` / `[]map[string]any`
- **`jsonAnySlice(pages)`** — converts `[]json.RawMessage` from `GetAll()` pagination to `[]map[string]any`
- **`convertSDKError(err)`** — converts SDK errors to `*output.Error`
- **`toSliceAny(v)`** — normalizes `[]map[string]any` or `[]any` to `[]any` for iteration

Account can be a slug or numeric ID.

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

### Unit Test Patterns

Tests use `SetTestModeWithSDK(mock)` which creates an httptest server backed by `MockClient`:

```go
mock := NewMockClient()
mock.GetResponse = &client.APIResponse{Data: map[string]any{"id": "1"}}
SetTestModeWithSDK(mock)
SetTestConfig("token", "account", "https://api.example.com")
defer resetTest()
```

- `mock.OnGet(path, resp)` — route-specific GET responses
- `mock.WithGetData(data)` — set default GET response data
- `mock.WithListData(data)` — set GetWithPagination response (used as fallback for GET when GetResponse is default empty map)
- JSON round-trip through httptest converts `int` → `float64` — use `float64(n)` in assertions

E2E environment variables:
- `FIZZY_TEST_TOKEN` - API token (required)
- `FIZZY_TEST_ACCOUNT` - Account slug (required)
- `FIZZY_TEST_USER_ID` - User ID for user tests (optional)

## Configuration

The CLI reads config from multiple sources with this priority:
1. CLI flags (`--token`, `--profile`, `--api-url`, `--board`)
2. Environment variables (`FIZZY_TOKEN`, `FIZZY_PROFILE`, `FIZZY_API_URL`, `FIZZY_BOARD`)
3. Named profile settings (base URL, board from `~/.config/fizzy/config.json`)
4. Local project config (`.fizzy.yaml`)
5. Global config (`~/.config/fizzy/config.yaml` or `~/.fizzy/config.yaml`)

`FIZZY_ACCOUNT` is accepted as a deprecated alias for `FIZZY_PROFILE`.

## Authentication

Token-based via personal access tokens. Run `fizzy setup` for interactive configuration or `fizzy auth login` to save a token directly.
