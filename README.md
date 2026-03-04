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

## Quick Start

```bash
brew install basecamp/tap/fizzy
fizzy setup
```

That's it. The setup wizard walks you through configuring your token, selecting your account, and optionally setting a default board. Try `fizzy board list` to verify everything is working.

<details>
<summary>Other installation methods</summary>

**Scoop (Windows):**
```bash
scoop bucket add basecamp https://github.com/basecamp/homebrew-tap
scoop install fizzy
```

**curl installer:**
```bash
curl -fsSL https://raw.githubusercontent.com/basecamp/fizzy-cli/master/scripts/install.sh | bash
```

**Go install:**
```bash
go install github.com/basecamp/fizzy-cli/cmd/fizzy@latest
```

**Arch Linux (AUR):**
```bash
yay -S fizzy-cli
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
fizzy board list               # JSON output
fizzy board list | jq '.data'  # Pipe through jq for raw data
```

### JSON Envelope

Every command returns structured JSON:

```json
{
  "success": true,
  "data": [...],
  "summary": "5 boards",
  "breadcrumbs": [{"action": "show", "cmd": "fizzy board show <id>"}]
}
```

Breadcrumbs suggest next commands, making it easy for humans and agents to navigate.

## AI Agent Integration

`fizzy` works with any AI agent that can run shell commands.

```bash
fizzy skill
```

This interactive command installs the [SKILL.md](skills/fizzy/SKILL.md) file to your preferred AI assistant (Claude Code, OpenCode, Codex, or a custom path).

## Configuration

```
~/.config/fizzy/              # Global config
└── config.yaml               #   Token, account, API URL, default board

.fizzy.yaml                   # Per-repo (local config overrides global)
```

Configuration priority (highest to lowest):
1. CLI flags (`--token`, `--account`, `--api-url`, `--board`)
2. Environment variables (`FIZZY_TOKEN`, `FIZZY_ACCOUNT`, `FIZZY_API_URL`, `FIZZY_BOARD`)
3. Local project config (`.fizzy.yaml`)
4. Global config (`~/.config/fizzy/config.yaml` or `~/.fizzy/config.yaml`)

## Development

```bash
make build            # Build binary
make test-unit        # Run unit tests (no API required)
make test-e2e         # Run e2e tests (requires FIZZY_TEST_TOKEN, FIZZY_TEST_ACCOUNT)
```

## License

[MIT](LICENSE)
