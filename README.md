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

That's it. The installer detects your platform and architecture, downloads the right binary, and verifies checksums. The setup wizard then walks you through configuring your token, selecting your account, and optionally setting a default board. Try `fizzy board list` to verify everything is working.

<details>
<summary>Other installation methods</summary>

**Omarchy/Arch Linux (AUR):**
```bash
yay -S fizzy-cli
```

**Homebrew (macOS):**
```bash
brew install basecamp/tap/fizzy
```

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

## Usage

```bash
fizzy board list                          # List boards
fizzy card list                           # List cards on default board
fizzy card show 42                        # Show card details
fizzy card create --board ID --title "Fix login bug"  # Create card
fizzy card close 42                       # Close card
fizzy search "authentication"             # Search across cards
fizzy comment create --card 42 --body "Looks good!"   # Add comment
```

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
3. Named profile settings (base URL, board from `config.json`)
4. Local project config (`.fizzy.yaml`)
5. Global config (`~/.config/fizzy/config.yaml` or `~/.fizzy/config.yaml`)

`FIZZY_ACCOUNT` is accepted as a deprecated alias for `FIZZY_PROFILE`.

## Development

```bash
make build            # Build binary
make test-unit        # Run unit tests (no API required)
make test-e2e         # Run e2e tests (requires FIZZY_TEST_TOKEN, FIZZY_TEST_ACCOUNT)
```

## License

[MIT](LICENSE)
