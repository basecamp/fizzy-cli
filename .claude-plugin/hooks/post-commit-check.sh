#!/usr/bin/env bash
# post-commit-check.sh - Check for Fizzy card references after git commits
#
# This hook runs after Bash tool use and checks if a git commit was made
# that references a Fizzy card (FIZZY-123, card-123, etc.)

set -euo pipefail

# Read tool input from stdin (JSON with tool_name, tool_input, tool_output)
input=$(cat)

# Extract tool input (the bash command that was run)
tool_input=$(echo "$input" | jq -r '.tool_input.command // empty' 2>/dev/null)

# Only process git commit commands
if [[ ! "$tool_input" =~ ^git\ commit ]]; then
  exit 0
fi

# Check if commit succeeded by looking for output patterns
tool_output=$(echo "$input" | jq -r '.tool_output // empty' 2>/dev/null)

# Skip if commit failed — detect error indicators before checking for success.
# Strip the "[branch hash] subject" success line before scanning for errors.
filtered_output=$(echo "$tool_output" | grep -v '^\[.*[a-f0-9]\{7,\}\]' || true)
if echo "$filtered_output" | grep -qiE '(^|[[:space:]])(error|fatal|aborted|rejected)[[:space:]:]|hook[[:space:]].*[[:space:]]failed|pre-commit[[:space:]].*[[:space:]]failed|^error:'; then
  exit 0
fi

# Verify commit actually succeeded - look for commit hash pattern or "create mode"
if [[ ! "$tool_output" =~ \[.*[a-f0-9]{7,}\] ]] && [[ ! "$tool_output" =~ "create mode" ]]; then
  exit 0
fi

# Look for card references in the commit message or branch name
branch=$(git branch --show-current 2>/dev/null || true)
last_commit_msg=$(git log -1 --format=%s 2>/dev/null || true)

# Patterns: FIZZY-123, card-123, fizzy-123
todo_patterns='FIZZY-[0-9]+|card-[0-9]+|fizzy-[0-9]+'

found_in_branch=$(echo "$branch" | grep -oEi "$todo_patterns" | head -1 || true)
found_in_msg=$(echo "$last_commit_msg" | grep -oEi "$todo_patterns" | head -1 || true)

if [[ -n "$found_in_branch" ]] || [[ -n "$found_in_msg" ]]; then
  ref="${found_in_msg:-$found_in_branch}"
  # Extract just the number
  card_number=$(echo "$ref" | grep -oE '[0-9]+')

  cat << EOF
<hook-output>
Detected Fizzy card reference: $ref

To link this commit to Fizzy:
  fizzy comment create --card $card_number --body "Commit $(git rev-parse --short HEAD 2>/dev/null): $last_commit_msg"

Or close the card:
  fizzy card close $card_number
</hook-output>
EOF
fi
