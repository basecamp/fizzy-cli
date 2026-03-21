---
name: fizzy
description: |
  Interact with Fizzy via the Fizzy CLI. Manage boards, cards, columns, comments,
  steps, reactions, tags, users, notifications, pins, webhooks, and account settings.
  Use this skill for ANY Fizzy question or action — including creating, searching, updating,
  or organizing cards, even if the user doesn't explicitly mention "fizzy."
triggers:
  # Direct invocations
  - fizzy
  - /fizzy
  # Resource actions
  - fizzy board
  - fizzy card
  - fizzy column
  - fizzy comment
  - fizzy step
  - fizzy reaction
  - fizzy tag
  - fizzy notification
  - fizzy webhook
  - fizzy account
  # Common actions
  - link to fizzy
  - track in fizzy
  - create card
  - close card
  - move card
  - assign card
  - add comment
  - add step
  - search cards
  # Search and discovery
  - search fizzy
  - find in fizzy
  - check fizzy
  - list fizzy
  - show fizzy
  - get from fizzy
  # Questions
  - what's in fizzy
  - what fizzy
  - how do I fizzy
  # My work
  - my cards
  - my tasks
  - my board
  - assigned to me
  - pinned cards
  # URLs
  - fizzy.do
  - app.fizzy.do
invocable: true
argument-hint: "[action] [args...]"
---

# /fizzy - Fizzy Workflow Command

Full CLI coverage: boards, cards, columns, comments, steps, reactions, tags, users, notifications, pins, webhooks, account settings, search, and board migration.

**When you need exact flags/args for a command**, read `references/commands.md`.
**When parsing response fields or writing jq queries**, read `references/schemas.md`.

## Key Behaviors

1. **Cards use NUMBER, not ID** — Card CLI commands take the human-readable `number` (e.g., `fizzy card show 42`), not the internal `id`. The API routes by number, so using the internal ID will 404. Other resources (boards, columns, comments, etc.) use their `id` field.

2. **Parse JSON with jq** — CLI output can be verbose. Pipe through jq to extract what you need and keep token usage low: `fizzy card list | jq '[.data[] | {number, title}]'`

3. **Use breadcrumbs** — Responses include a `breadcrumbs` array with ready-to-run commands for logical next actions. Check them before guessing at command syntax.

4. **Check for board context** — Look for `.fizzy.yaml` or `--board` flag before listing cards. Without board context, card list returns cards across all boards.

5. **Rich text fields accept HTML** — Use `<p>` tags for paragraphs, `<action-text-attachment>` for inline images.

6. **Card description vs comment body** — Card `.description` is a plain string. Comment `.body` is a nested object with `.body.plain_text` and `.body.html`. This asymmetry causes bugs if you treat them the same way.

7. **Welcome message for new signups** — When `signup complete --name` returns `is_new_user: true`, display the `welcome_message` field prominently to the user. It's a one-time personal note from the CEO that won't be shown again.

## Decision Trees

### Finding Content

```
Need to find something?
├── Know the board? → fizzy card list --board <id>
├── Full-text search? → fizzy search "query"
├── Filter by status? → fizzy card list --indexed-by closed|not_now|golden|stalled
├── Filter by person? → fizzy card list --assignee <id>
├── Filter by time? → fizzy card list --created today|thisweek|thismonth
└── Cross-board? → fizzy search "query" (searches all boards)
```

### Modifying Content

```
Want to change something?
├── Move to column? → fizzy card column <number> --column <id>
├── Change status? → fizzy card close|reopen|postpone <number>
├── Assign? → fizzy card assign <number> --user <id>
├── Comment? → fizzy comment create --card <number> --body "text"
├── Add step? → fizzy step create --card <number> --content "text"
└── Move to board? → fizzy card move <number> --to <board_id>
```

## Quick Reference

| Resource | List | Show | Create | Update | Delete | Other |
|----------|------|------|--------|--------|--------|-------|
| account | - | `account show` | - | `account settings-update` | - | `account entropy`, `account export-create/show`, `account join-code-show/reset/update` |
| board | `board list` | `board show ID` | `board create` | `board update ID` | `board delete ID` | `board publish/unpublish`, `board entropy`, `board closed/postponed/stream`, `board involvement`, `migrate board` |
| card | `card list` | `card show NUMBER` | `card create` | `card update NUMBER` | `card delete NUMBER` | `card move`, `card close/reopen/postpone/untriage`, `card assign/self-assign`, `card tag`, `card pin/unpin`, `card golden/ungolden`, `card watch/unwatch`, `card publish`, `card mark-read/mark-unread`, `card image-remove`, `card attachments` |
| search | `search QUERY` | - | - | - | - | - |
| column | `column list --board ID` | `column show ID --board ID` | `column create` | `column update ID` | `column delete ID` | `column move-left/right` |
| comment | `comment list --card NUMBER` | `comment show ID --card NUMBER` | `comment create` | `comment update ID` | `comment delete ID` | `comment attachments` |
| step | `step list --card NUMBER` | `step show ID --card NUMBER` | `step create` | `step update ID` | `step delete ID` | - |
| reaction | `reaction list` | - | `reaction create` | - | `reaction delete ID` | - |
| tag | `tag list` | - | - | - | - | - |
| user | `user list` | `user show ID` | - | `user update ID` | - | `user deactivate`, `user role`, `user avatar-remove`, `user push-subscription-create/delete` |
| notification | `notification list` | - | - | - | - | `notification tray`, `notification read/read-all/unread`, `notification settings-show/update` |
| pin | `pin list` | - | - | - | - | `card pin/unpin` |
| webhook | `webhook list --board ID` | `webhook show ID --board ID` | `webhook create` | `webhook update ID` | `webhook delete ID` | `webhook reactivate` |

For full command syntax with all flags, see `references/commands.md`.

---

## Global Flags

All commands support:

| Flag | Description |
|------|-------------|
| `--token TOKEN` | API access token |
| `--profile NAME` | Named profile (for multi-account users) |
| `--api-url URL` | API base URL (default: https://app.fizzy.do) |
| `--json` | JSON envelope output |
| `--quiet` | Raw JSON data without envelope |
| `--styled` | Human-readable styled output (tables, colors) |
| `--markdown` | GFM markdown output (for agents) |
| `--agent` | Agent mode (defaults to quiet; combinable with --json/--markdown) |
| `--ids-only` | Print one ID per line |
| `--count` | Print count of results |
| `--limit N` | Client-side truncation of list results |
| `--verbose` | Show request/response details |

Output format defaults to auto-detection: styled for TTY, JSON for pipes/non-TTY.

## Pagination

List commands use `--page N` for pagination and `--limit N` for client-side truncation. Use `--all` to fetch all pages at once. `--limit` and `--all` cannot be combined.

The `--all` flag controls pagination only — it fetches all pages but does NOT change which cards are included. By default, `card list` returns only open cards. See Card Statuses below.

Commands supporting `--all` and `--page`: `board list`, `board closed`, `board postponed`, `board stream`, `card list`, `search`, `comment list`, `tag list`, `user list`, `notification list`, `webhook list`.

---

## Card Statuses

By default, `fizzy card list` returns **open cards only** (in triage or columns). To fetch cards in other states:

| Status | How to fetch |
|--------|--------------|
| Open (default) | `fizzy card list` |
| Closed/Done | `fizzy card list --indexed-by closed` |
| Not Now | `fizzy card list --indexed-by not_now` |
| Golden | `fizzy card list --indexed-by golden` |
| Stalled | `fizzy card list --indexed-by stalled` |

Pseudo-columns also work: `--column done`, `--column not-now`, `--column maybe` (triage).

To get **all cards** regardless of status, make separate queries for open, closed, and not_now.

## ID Formats

Cards have TWO identifiers:

| Field | Format | Use For |
|-------|--------|---------|
| `id` | `03fe4rug9kt1mpgyy51lq8i5i` | Internal ID (in JSON responses) |
| `number` | `579` | CLI commands (`card show`, `card update`, etc.) |

All card CLI commands use the card NUMBER. Other resources use their `id` field.

---

## Response Structure

```json
{
  "ok": true,
  "data": { ... },
  "summary": "4 boards",
  "breadcrumbs": [ ... ],
  "context": { ... }
}
```

**Breadcrumbs** suggest next actions with pre-filled values:
```json
[
  {"action": "comment", "cmd": "fizzy comment create --card 42 --body \"text\"", "description": "Add comment"},
  {"action": "close", "cmd": "fizzy card close 42", "description": "Close card"}
]
```

Values like card numbers and board IDs are pre-filled; placeholders like `<column_id>` need replacement.

**List responses** include `context.pagination.has_next` for pagination.
**Create/update responses** include `context.location`.

---

## Configuration

```yaml
# .fizzy.yaml (per-repo, committed to git)
account: 123456789
board: 03foq1hqmyy91tuyz3ghugg6c
```

Check context: `cat .fizzy.yaml 2>/dev/null || echo "No project configured"`

## Setup & Auth

```bash
fizzy setup                    # Interactive wizard
fizzy auth login TOKEN         # Save token
fizzy auth status              # Check auth
fizzy auth list                # List profiles
fizzy auth switch PROFILE      # Switch profile
fizzy auth logout              # Log out (--all for all profiles)
fizzy identity show            # Show identity and accounts
```

### Signup (for agents)

```bash
# 1. Request magic link
fizzy signup start --email user@example.com
# Returns: {"pending_authentication_token": "eyJ..."}

# 2. User checks email for 6-digit code, then verify
fizzy signup verify --code ABC123 --pending-token eyJ...
# Returns: {"session_token": "eyJ...", "requires_signup_completion": true/false}

# 3. Write session token to temp file (keep out of agent context)
echo "eyJ..." > /tmp/fizzy-session && chmod 600 /tmp/fizzy-session

# 4a. New user → fizzy signup complete --name "Full Name" < /tmp/fizzy-session
# 4b. Existing user → fizzy signup complete --account SLUG < /tmp/fizzy-session
# Returns: {"token": "fizzy_...", "account": "slug"}

# 5. Clean up
rm /tmp/fizzy-session
```

The user must check their email between steps 1 and 2. Token is saved to the system credential store when available.

---

## Common Workflows

### Create Card with Steps

```bash
CARD=$(fizzy card create --board BOARD_ID --title "New Feature" \
  --description "<p>Feature description</p>" | jq -r '.data.number')
fizzy step create --card $CARD --content "Design the feature"
fizzy step create --card $CARD --content "Implement backend"
fizzy step create --card $CARD --content "Write tests"
```

### Link Code to Card

```bash
fizzy comment create --card 42 --body "<p>Commit $(git rev-parse --short HEAD): $(git log -1 --format=%s)</p>"
fizzy card close 42
```

### Create Card with Inline Image

```bash
SGID=$(fizzy upload file screenshot.png | jq -r '.data.attachable_sgid')
cat > desc.html << EOF
<p>See the screenshot below:</p>
<action-text-attachment sgid="$SGID"></action-text-attachment>
EOF
fizzy card create --board BOARD_ID --title "Bug Report" --description_file desc.html
```

### Move Card Through Workflow

```bash
fizzy card column 579 --column maybe       # Move to column
fizzy card self-assign 579                  # Assign to yourself
fizzy card golden 579                       # Mark as important
fizzy card close 579                        # Close when done
```

### Search and Filter

```bash
fizzy search "bug" | jq '[.data[] | {number, title}]'
fizzy card list --created today --sort newest
fizzy card list --indexed-by closed --closed thisweek
fizzy card list --unassigned --board BOARD_ID
```

---

## Rich Text

Card descriptions and comments support HTML. For spacing: `<p>First</p><p><br></p><p>Second</p>`.

Each `attachable_sgid` can only be used once — re-upload for multiple uses.

**Defaults:** Use inline images (via `attachable_sgid`) by default. Only use background/header (`signed_id` with `--image`) when user explicitly says "background" or "header".

## Error Handling

| Exit Code | Meaning | Action |
|-----------|---------|--------|
| 0 | Success | — |
| 1 | Usage / invalid args | Check command syntax in `references/commands.md` |
| 2 | Not found | Verify card NUMBER or resource ID |
| 3 | Auth failure | Run `fizzy auth status` then `fizzy auth login TOKEN` |
| 4 | Permission denied | Requires admin/owner role |
| 5 | Rate limited | Wait and retry |
| 6 | Network error | Check `fizzy auth status` for API URL |
| 7 | API / server error | Retry or report |
| 8 | Ambiguous match | Be more specific |

## Learn More

- API documentation: https://github.com/basecamp/fizzy/blob/main/docs/API.md
- CLI repository: https://github.com/basecamp/fizzy-cli
