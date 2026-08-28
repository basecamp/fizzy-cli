# <img src="assets/fizzy-badge.png" height="28" alt="Fizzy"> Fizzy CLI

```
⠀⡀⠄⠀⣤⣤⡄⠠⢠⣤⣤⠀⠄⣠⣤⣤⠀⠠⣠⣤⣄⠠⠀⣤⣤⡄⠠⠀⠠⠀⠠⠀⠠⠀⠠⠀⠠⠀⠠⠀⠠⠀⠠⠀⠠⠀⠠⠀⠠⠀⠠⠀⠠⠀⠠⠀⠠⠀⠠⠀⠠⠀⠠⠀⠠⠀⠠⠀⠠⠀⠠⠀⠠⠀⠠⠀⠠⠀⠠⠀
⠀⠀⠄⢸⣿⣿⡇⠠⢸⣿⣿⡇⠀⣿⣿⡿⡇⢈⣿⣿⣿⠀⢸⣿⣿⡇⢀⠈⡀⢈⠀⡈⢀⠈⡀⢈⠀⡈⢀⠈⡀⢈⠀⡈⢀⠈⡀⢈⠀⡈⢀⠈⡀⢈⠀⡈⢀⠈⡀⢈⠀⡈⢀⠈⠠⠈⢀⠈⡀⢈⠀⡈⢀⠈⡀⢈⠀⡈⠠⠀
⠀⢁⠠⢸⣿⡿⡇⠀⢸⣿⡿⡇⠀⣿⣿⢿⡃⠠⣿⣿⣾⠀⢸⣿⣽⡇⠀⠠⠀⠄⠠⠀⣦⣦⣦⣦⣦⣦⣦⣦⣦⠀⢰⣾⣷⡦⠀⠄⠠⠀⠄⠠⠀⠄⠠⠀⠄⠠⠀⠄⠠⠀⠄⠐⢀⠈⡀⠠⠀⠄⠠⠀⠄⠠⠀⠄⠠⠀⠐⠀
⠀⠠⠀⢸⣿⣿⡇⠈⢸⣿⣿⡇⠀⣿⣿⣿⠇⢈⣿⣷⣿⠀⢸⣿⣽⡇⠀⠂⠐⠀⠂⡀⣿⣿⡟⠛⠛⠛⠛⠛⠛⠀⠄⠉⠉⢁⠀⠂⠐⠀⠂⠐⠀⠂⠐⠀⠂⠐⠀⠂⠐⠀⠂⠁⠠⠀⠄⠐⠀⠂⠐⠀⠂⠐⠀⠂⠐⠈⡀⠁
⠀⠂⠈⢸⣿⣯⡇⠠⢸⣿⣯⡇⠀⣿⣿⣾⡇⠐⣿⣯⣿⠀⢸⣿⣽⡇⠀⡁⢈⠀⡁⠀⣿⣿⡇⠀⠄⠂⠀⠂⠐⢀⢸⣿⣿⡇⠀⡁⣿⣿⣿⣿⣿⣿⣿⣿⡃⢈⠀⣿⣿⣿⣿⣿⣿⣿⣿⠅⠈⢿⣿⣿⡀⢁⠈⡀⣽⣿⣿⠃
⠀⡈⠠⢸⣿⡿⡇⠀⢸⣿⡿⡇⠀⣿⣿⣽⡆⠨⣿⣟⣿⠀⢸⣯⣿⡇⠀⠄⠠⠀⠄⠂⣿⣿⢷⣶⣶⣶⣷⣾⡆⢀⢸⣿⣻⡇⠀⡀⠉⡈⠁⣁⣵⣿⣿⠋⠀⠠⠀⢉⠈⢁⢁⣵⣿⡿⠋⠀⠐⠘⣿⣿⣧⠀⠄⢠⣿⣿⠎⠀
⠀⠠⠀⢸⣿⣿⡇⠈⢸⣿⣿⡇⠀⣿⣿⣻⡅⠐⠿⣿⠟⠀⢸⣿⣽⡇⠀⠂⠐⠀⠂⡀⣿⣿⡟⠛⠛⠋⠛⠛⠃⠀⢸⣟⣿⡇⠀⠄⠂⠠⣰⣾⣿⠟⠁⢀⠈⠠⠈⢀⢠⣰⣿⡿⠟⠀⡀⢁⠈⡀⠸⣿⣾⡇⢀⣿⣿⡟⠀⠄
⠀⠂⠈⠸⣿⣯⡇⠠⢸⣿⣯⡇⠀⣿⣿⢿⡃⠠⠀⡢⠀⡁⠘⣿⣽⡇⠀⡁⢈⠀⢁⠀⣿⣿⡇⠀⠐⠀⠂⠐⠀⡁⢸⣿⣟⡇⢀⠐⣠⣶⣿⡟⠃⢀⠐⠀⡐⢀⠈⣠⣾⣿⡗⠋⠀⠄⠠⠀⠄⠀⠂⢹⣿⣷⣼⣿⡿⠀⠐⠀
⠀⡈⢀⠁⠉⡏⠀⠠⢸⣿⡿⡇⠀⣿⣿⣿⠇⠀⠂⢜⠀⡀⠂⠉⡏⠀⠄⠠⠀⠐⢀⠀⣿⣿⡇⠀⡁⢈⠀⡁⠄⠠⢸⣿⣽⡇⠀⡀⣿⣿⣻⣿⣿⣿⣿⣿⡇⠀⢐⣿⣿⣷⣿⣿⣿⣿⣿⡇⢀⠁⡈⠀⢹⣿⣻⣽⠁⢀⠁⠄
⠀⠠⠀⠐⠀⡇⠈⡀⢸⣿⣿⡇⠀⠙⢻⠊⠠⠈⡀⢕⠀⠠⠐⠀⡇⠐⠀⠂⢈⠠⠀⠄⢀⠀⡀⠄⠠⠀⠄⠠⠀⠂⢀⠀⡀⠠⠀⠄⢀⠀⡀⠀⡀⠀⡀⠀⡀⠐⢀⠀⡀⠀⡀⠀⡀⠀⡀⠀⠄⠠⣀⣈⣸⣿⣿⠃⠀⠄⠐⠀
⠀⠐⠈⢀⠁⡇⠠⠀⠸⢿⡯⠃⢀⠁⢸⠀⠂⠠⠀⠪⠀⠐⠀⠁⡇⢀⠁⡈⢀⠠⠐⠀⠄⠠⠀⠄⠂⠐⠀⠂⡀⢁⠠⠀⠄⠐⠀⠂⠠⠀⠄⠂⠀⠂⠀⠂⡀⢈⠀⠠⠀⠂⠀⠂⠀⠂⠀⠂⠐⠰⣿⣿⠿⠟⠁⡀⠂⡀⠁⠄
⠀⢁⠈⢀⠠⠐⠀⡈⠠⠀⡀⠄⠠⠐⠀⠂⠁⠐⠀⠂⡈⢀⠁⠐⡀⠄⠠⠀⠄⢀⠐⠀⠂⠐⠀⠂⡀⢁⠈⡀⠄⠠⠀⠐⢀⠈⡀⢁⠐⢀⠐⠀⡁⢈⠀⡁⠀⠄⠐⠀⠂⠁⡈⢀⠁⡈⢀⠁⡈⢀⠀⡀⠄⠂⠁⠀⠄⠠⠈⠀
```

`fizzy` is a command-line interface for [Fizzy](https://fizzy.do). Manage boards, cards, comments, and more from your terminal or through AI agents.

- Works standalone or with any AI agent (Claude, Codex, Copilot, Gemini)
- JSON output with breadcrumbs for easy navigation
- Token authentication via personal access tokens
- Includes agent skill and Claude plugin setup

## Quick Start

```bash
curl -fsSL https://raw.githubusercontent.com/basecamp/fizzy-cli/master/scripts/install.sh | bash
fizzy setup
```

That's it. The installer detects your platform and architecture, downloads the right binary, and verifies checksums. The setup wizard then walks you through configuring your token, selecting your account, and optionally setting a default board.

Recommended first checks:

```bash
fizzy doctor
fizzy board list
```

Use `fizzy doctor` any time you want a full health check of your install, config, auth, API connectivity, board context, and agent setup.

<details>
<summary>Other installation methods</summary>

**Omarchy/Arch Linux (AUR):**
```bash
yay -S fizzy-cli
```

**Homebrew (macOS and Linux):**

```bash
brew install --cask basecamp/tap/fizzy
```

Release candidates are not published to Homebrew; they are available from [Releases](https://github.com/basecamp/fizzy-cli/releases).

*Upgrading from Fizzy 3.x:* 3.x was the `fizzy-cli` formula in the `robzolkos/fizzy-cli` tap. `brew update` points that tap at this cask, but Homebrew never taps a third-party tap for you — not during migration, and not from a fully-qualified `brew install` — so run `brew tap` yourself first:

```bash
brew tap basecamp/tap
brew unlink fizzy-cli
brew install --cask basecamp/tap/fizzy
brew uninstall --formula --force fizzy-cli
brew untap robzolkos/fizzy-cli
```

`brew unlink` first is required: a cask will not overwrite a `bin/fizzy` symlink owned by the formula, so installing the cask while the formula is still linked skips the binary and the later uninstall leaves no `fizzy` on your PATH. Recover with `brew reinstall --cask fizzy`. Note also that `brew migrate fizzy-cli`, which Homebrew suggests, is a no-op for a formula-to-cask move.

**Scoop (Windows):**
```bash
scoop bucket add basecamp https://github.com/basecamp/homebrew-tap
scoop install fizzy
```

**Go install:**
```bash
go install github.com/basecamp/fizzy-cli/cmd/fizzy@latest
```

**Debian/Ubuntu:**
```bash
curl -LO https://github.com/basecamp/fizzy-cli/releases/latest/download/fizzy-cli_VERSION_linux_amd64.deb
sudo dpkg -i fizzy-cli_VERSION_linux_amd64.deb
```

**Fedora/RHEL:**
```bash
curl -LO https://github.com/basecamp/fizzy-cli/releases/latest/download/fizzy-cli_VERSION_linux_amd64.rpm
sudo rpm -i fizzy-cli_VERSION_linux_amd64.rpm
```

**Windows:** download `fizzy_VERSION_windows_amd64.zip` from [Releases](https://github.com/basecamp/fizzy-cli/releases), extract, and add `fizzy.exe` to your PATH.

**GitHub Release:** download from [Releases](https://github.com/basecamp/fizzy-cli/releases).

</details>

## Next Steps

Start with a few common commands:

```bash
fizzy board list
fizzy card list
fizzy card show 42
fizzy search "authentication"
fizzy comment create --card 42 --body "Looks good!"
```

Then branch out as needed:

```bash
fizzy board accesses --board ID           # Show board access settings and users
fizzy activity list --board ID            # List recent board activity
fizzy webhook deliveries --board ID WEBHOOK_ID
fizzy user export-create USER_ID
```

For the full command surface, run `fizzy commands --json` or read [`skills/fizzy/SKILL.md`](skills/fizzy/SKILL.md).

### Attachments

Simple mode uses repeatable `--attach` and appends inline attachments to the end of card descriptions or comment bodies:

```bash
fizzy card create --board ID --title "Bug report" --description "See attached" --attach screenshot.png
fizzy comment create --card 42 --attach logs.txt
fizzy comment create --card 42 --body_file comment.md --attach screenshot.png --attach trace.txt
```

Advanced mode still works when exact placement matters:

```bash
SGID=$(fizzy upload file screenshot.png --jq '.data.attachable_sgid')
fizzy card create --board ID --title "Bug report" \
  --description "<p>See screenshot</p><action-text-attachment sgid=\"$SGID\"></action-text-attachment>"
```

Use `signed_id` from `fizzy upload file` only for card header images via `--image`.

### Output Formats

```bash
fizzy board list                                 # JSON output
fizzy board list --jq '.data[0].name'            # Filter the JSON envelope (built-in, no external jq required)
fizzy board list --quiet --jq '.[0].name'        # Filter raw data without the envelope
fizzy board list --jq '[.data[] | {id, name}]'   # Extract specific fields
```

`--jq` is for machine-readable JSON output. It implies `--json` and cannot be combined with `--styled`, `--markdown`, `--ids-only`, or `--count`.

### JSON Envelope

Every command returns structured JSON:

```json
{
  "ok": true,
  "data": [...],
  "summary": "5 boards",
  "breadcrumbs": [{"action": "show", "cmd": "fizzy board show <id>"}]
}
```

Breadcrumbs suggest next commands, making it easy for humans and agents to navigate.

## AI Agent Integration

`fizzy` works with any AI agent that can run shell commands.

**Claude Code:** `fizzy setup claude` — installs the Claude plugin from the marketplace and links the embedded Fizzy skill into Claude's skills directory.

**Other agents:** Point your agent at [`skills/fizzy/SKILL.md`](skills/fizzy/SKILL.md) for Fizzy workflow coverage. `fizzy skill` launches the interactive installer by default, while `fizzy skill install` installs the embedded skill directly.

**Agent discovery:** Every command supports `--help --agent` for structured help output. Use `fizzy commands --json` for the full command catalog.

**Troubleshooting:** Run `fizzy doctor` for a read-only health check with remediation hints and next steps.

## Configuration

```
~/.config/fizzy/              # Global config
├── config.json               #   Named profiles (account, base URL, board)
├── config.yaml               #   Legacy/fallback settings
└── credentials/              #   Fallback token storage (when keyring unavailable)

.fizzy.yaml                   # Per-repo (local config overrides global)
```

Configuration priority (highest to lowest):
1. CLI flags (`--token`, `--profile`, `--api-url`, `--board`)
2. Environment variables (`FIZZY_TOKEN`, `FIZZY_PROFILE`, `FIZZY_API_URL`, `FIZZY_BOARD`)
3. Named profile settings (account, base URL, board from `config.json`)
4. Local project config (`.fizzy.yaml`)
5. Global config (`~/.config/fizzy/config.yaml` or `~/.fizzy/config.yaml`)

`FIZZY_ACCOUNT` is accepted as a deprecated alias for `FIZZY_PROFILE`.

Profiles created by `fizzy setup` use the account slug as their name and continue to work without changes. A profile can also use a distinct name by storing its account in the profile's `extra` settings. This supports separate credentials for multiple users of the same account:

```bash
fizzy auth login "$WALTER_TOKEN" --profile walter --account 1
fizzy auth login "$AGENT_TOKEN" --profile walter-agent --account 1
```

The commands store credentials under each profile name while routing both profiles to account `1`:

```json
{
  "profiles": {
    "walter": {
      "base_url": "https://app.fizzy.do",
      "extra": { "account": "1" }
    },
    "walter-agent": {
      "base_url": "https://app.fizzy.do",
      "extra": { "account": "1" }
    }
  },
  "default_profile": "walter"
}
```

`FIZZY_NO_UPDATE_NOTIFIER=1` runs commands without update notifications.

Inspect the effective config and precedence:

```bash
fizzy config show
fizzy config explain
fizzy config explain --profile acme
```

## MCP server

`fizzy mcp` runs an MCP (Model Context Protocol) server on stdin/stdout, serving Fizzy
boards, columns, cards, comments, steps, tags, users, and your identity as tools
backed by your signed-in account — the same credentials every other command uses. Register it with any MCP
client as a stdio server:

```bash
claude mcp add fizzy -- fizzy mcp        # Claude Code
fizzy mcp --writes                       # serve write actions too (pair with a Read+Write token)
fizzy mcp --domains boards,cards         # narrow the served tool surface
```

Each domain is one gateway tool (`fizzy_identity`, `fizzy_boards`, `fizzy_columns`,
`fizzy_cards`, `fizzy_comments`, `fizzy_steps`, `fizzy_tags`, `fizzy_users`); call an
action named `describe` for any action's parameter schema. Read-only by default —
`--writes` opts in. Listings with more pages come back as
`{"data": ..., "next_page": N}` — pass the number back as the action's `page`
parameter. Logs go to stderr — stdout carries the MCP wire protocol.

## Troubleshooting

```bash
fizzy doctor                 # Full install/config/auth/API/agent health check
fizzy doctor --profile acme  # Check one saved profile explicitly
fizzy doctor --all-profiles  # Sweep every saved profile
fizzy doctor --verbose       # Include effective config details and timings
fizzy doctor --json          # Structured output for scripts and support
```

Common follow-up commands:

```bash
fizzy auth status
fizzy config show
fizzy config explain
fizzy identity show
fizzy board list
fizzy setup
fizzy setup claude
fizzy skill install
```

## Development

```bash
make build            # Build binary
make test-unit        # Run unit tests (no API required)
make e2e              # Run owner-only CLI contract e2e suite
make e2e-run NAME=TestBoardList
```

E2E requirements:
- `FIZZY_TEST_TOKEN`
- `FIZZY_TEST_ACCOUNT`
- optional: `FIZZY_TEST_API_URL`
- optional: `FIZZY_TEST_BINARY`

Useful local inspection modes:
- `FIZZY_E2E_KEEP_FIXTURE=1 make e2e`
- `FIZZY_E2E_TEARDOWN_DELAY=120 make e2e`

## License

[MIT](LICENSE)
