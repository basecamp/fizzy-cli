#!/usr/bin/env bash
# session-start.sh - Fizzy plugin liveness check
#
# Lightweight: one subprocess call. Context priming happens on first
# use via the /fizzy skill, not here.

set -euo pipefail

if ! command -v fizzy &>/dev/null; then
  cat << 'EOF'
<hook-output>
Fizzy plugin active — CLI not found on PATH.
Install: https://github.com/basecamp/fizzy-cli#installation
</hook-output>
EOF
  exit 0
fi

auth_json=$(fizzy auth status --json 2>/dev/null || echo '{}')

if ! command -v jq &>/dev/null; then
  cat << 'EOF'
<hook-output>
Fizzy plugin active.
</hook-output>
EOF
  exit 0
fi

is_auth=false
if parsed_auth=$(echo "$auth_json" | jq -er '.data.authenticated' 2>/dev/null); then
  is_auth="$parsed_auth"
fi

if [[ "$is_auth" == "true" ]]; then
  cat << 'EOF'
<hook-output>
Fizzy plugin active.
</hook-output>
EOF
else
  cat << 'EOF'
<hook-output>
Fizzy plugin active — not authenticated.
Run: fizzy setup
</hook-output>
EOF
fi
