#!/usr/bin/env bash
# session-start.sh - Load Fizzy context at session start
#
# This hook runs when Claude Code starts a session and outputs
# relevant Fizzy board context if configured.

set -euo pipefail

# Require jq for JSON parsing
if ! command -v jq &>/dev/null; then
  exit 0
fi

# Find fizzy - prefer PATH, fall back to plugin's bin directory
if command -v fizzy &>/dev/null; then
  FIZZY_BIN="fizzy"
else
  SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  FIZZY_BIN="${SCRIPT_DIR}/../../bin/fizzy"
  if [[ ! -x "$FIZZY_BIN" ]]; then
    cat << 'EOF'
<hook-output>
Fizzy plugin: fizzy CLI not found.
Install: https://github.com/robzolkos/fizzy-cli#quick-start
</hook-output>
EOF
    exit 0
  fi
fi

# Get CLI version
cli_version=$("$FIZZY_BIN" version 2>/dev/null | jq -r '.data.version // empty' 2>/dev/null || true)

# Check if authenticated
auth_output=$("$FIZZY_BIN" auth status 2>/dev/null || echo '{}')
is_auth=$(echo "$auth_output" | jq -r '.data.authenticated // false')
account=$(echo "$auth_output" | jq -r '.data.account // empty')

if [[ "$is_auth" != "true" ]]; then
  cat << 'EOF'
<hook-output>
Fizzy plugin: Not authenticated.
Run: fizzy setup
</hook-output>
EOF
  exit 0
fi

# Build context message
context="Fizzy context loaded:"

if [[ -n "$cli_version" ]]; then
  context+="\n  CLI: ${cli_version}"
fi

if [[ -n "$account" ]]; then
  context+="\n  Account: $account"
fi

# Check for local .fizzy.yaml board config
if command -v yq &>/dev/null; then
  local_board=$(yq -r '.board // empty' .fizzy.yaml 2>/dev/null || true)
elif command -v python3 &>/dev/null; then
  local_board=$(python3 -c "import yaml,sys; d=yaml.safe_load(open('.fizzy.yaml')); print(d.get('board',''))" 2>/dev/null || true)
fi

if [[ -n "${local_board:-}" ]]; then
  context+="\n  Board: $local_board (from .fizzy.yaml)"
fi

cat << EOF
<hook-output>
$(echo -e "$context")

Use \`fizzy\` commands to interact with Fizzy:
  fizzy board list               # List boards
  fizzy card list                # List cards on default board
  fizzy search "query"           # Search across cards
  fizzy card show <number>       # Show card details
</hook-output>
EOF
