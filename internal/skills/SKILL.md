---
name: fizzy
description: |
  Interact with Fizzy via the Fizzy CLI. Manage boards, cards, columns, comments,
  steps, reactions, tags, users, notifications, and pins. Use for ANY Fizzy question or action.
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

Full CLI coverage: boards, cards, columns, comments, steps, reactions, tags, users, notifications, pins, search, and board migration.

## Agent Invariants

**MUST follow these rules:**

1. **Cards use NUMBER, not ID** — `fizzy card show 42` uses the card number. Other resources use their `id` field.
2. **Parse JSON with jq** to reduce token output — `fizzy card list | jq '[.data[] | {number, title}]'`
3. **Check breadcrumbs** in responses for available next actions with pre-filled values
4. **Check for board context** via `.fizzy.yaml` or `--board` flag before listing cards
5. **Rich text fields accept HTML** — use `<p>` tags for paragraphs, `<action-text-attachment>` for inline images
6. **Card description is a string**, but comment body is a nested object — `.description` vs `.body.plain_text`
7. **Display the welcome message for new signups** — When `signup complete --name` returns `is_new_user: true`, you MUST immediately display the `welcome_message` field prominently to the user. This is a one-time personal note from the CEO — if you skip it, the user will never see it.

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
| board | `board list` | `board show ID` | `board create` | `board update ID` | `board delete ID` | `migrate board ID` |
| card | `card list` | `card show NUMBER` | `card create` | `card update NUMBER` | `card delete NUMBER` | `card move NUMBER` |
| search | `search QUERY` | - | - | - | - | - |
| column | `column list --board ID` | `column show ID --board ID` | `column create` | `column update ID` | `column delete ID` | - |
| comment | `comment list --card NUMBER` | `comment show ID --card NUMBER` | `comment create` | `comment update ID` | `comment delete ID` | `comment attachments show --card NUMBER` |
| step | - | `step show ID --card NUMBER` | `step create` | `step update ID` | `step delete ID` | - |
| reaction | `reaction list` | - | `reaction create` | - | `reaction delete ID` | - |
| tag | `tag list` | - | - | - | - | - |
| user | `user list` | `user show ID` | - | `user update ID` | - | `user deactivate ID` |
| notification | `notification list` | - | - | - | - | `notification tray`, `notification read-all` |
| pin | `pin list` | - | - | - | - | `card pin NUMBER`, `card unpin NUMBER` |

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

List commands use `--page` for pagination and `--limit` for client-side truncation.

```bash
# Get first page (default)
fizzy card list --page 1

# Limit to N results
fizzy card list --limit 5

# Fetch ALL pages at once
fizzy card list --all
```

Note: `--limit` and `--all` cannot be used together.

**IMPORTANT:** The `--all` flag controls pagination only - it fetches all pages of results for your current filter. It does NOT change which cards are included. By default, `card list` returns only open cards. See [Card Statuses](#card-statuses) for how to fetch closed or postponed cards.

Commands supporting `--all` and `--page`:
- `board list`
- `card list`
- `search`
- `comment list`
- `tag list`
- `user list`
- `notification list`

---

## Configuration

```
~/.config/fizzy/              # Global config (or ~/.fizzy/)
└── config.yaml               #   Token, account, API URL, default board

.fizzy.yaml                   # Per-repo config (committed to git)
```

**Per-repo config:** `.fizzy.yaml`
```yaml
account: 123456789
board: 03foq1hqmyy91tuyz3ghugg6c
```

**Priority (highest to lowest):**
1. CLI flags (`--token`, `--profile`, `--api-url`, `--board`)
2. Environment variables (`FIZZY_TOKEN`, `FIZZY_PROFILE`, `FIZZY_API_URL`, `FIZZY_BOARD`)
3. Named profile settings (base URL, board from `config.json`)
4. Local project config (`.fizzy.yaml`)
5. Global config (`~/.config/fizzy/config.yaml` or `~/.fizzy/config.yaml`)

**Check context:**
```bash
cat .fizzy.yaml 2>/dev/null || echo "No project configured"
```

**Setup:**
```bash
fizzy setup                              # Interactive wizard
fizzy auth login TOKEN                   # Save token for current profile
fizzy auth status                        # Check auth status
fizzy auth list                          # List all authenticated profiles
fizzy auth switch PROFILE                # Switch active profile
fizzy auth logout                        # Log out current profile
fizzy auth logout --all                  # Log out all profiles
fizzy identity show                      # Show profiles
```

### Signup (New User or Token Generation)

Interactive:
```bash
fizzy signup                                    # Full interactive wizard
```

Step-by-step (for agents):
```bash
# Step 1: Request magic link
fizzy signup start --email user@example.com
# Returns: {"pending_authentication_token": "eyJ..."}

# Step 2: User checks email for 6-digit code, then verify
fizzy signup verify --code ABC123 --pending-token eyJ...
# Returns: {"session_token": "eyJ...", "requires_signup_completion": true/false}
# For existing users (requires_signup_completion=false), also returns: "accounts": [{"name": "...", "slug": "..."}]

# Step 3: Write the session token to a temp file to keep it out of the agent session
echo "eyJ..." > /tmp/fizzy-session && chmod 600 /tmp/fizzy-session

# Step 4a: New user — complete signup (session token via stdin)
fizzy signup complete --name "Full Name" < /tmp/fizzy-session
# Returns: {"token": "fizzy_...", "account": "slug"}

# Step 4b: Existing user — generate token for an account
fizzy signup complete --account SLUG < /tmp/fizzy-session
# Returns: {"token": "fizzy_...", "account": "slug"}

# Step 5: Clean up the temp file
rm /tmp/fizzy-session
```

**Note:** The user must check their email for the 6-digit code between steps 1 and 2.
The session token is written to a temp file and piped via stdin to avoid exposing it in shell history or the agent's conversation context.
Token is saved to the system credential store when available, with a config file as fallback. Profile and API URL are saved to `~/.config/fizzy/` (preferred) or `~/.fizzy/`.

**Welcome message for new users:** When `signup complete --name` succeeds (new user), the response includes `is_new_user: true` and a `welcome_message` field. See Agent Invariant #7 — you MUST display it.

---

## Response Structure

All responses follow this structure:

```json
{
  "ok": true,
  "data": { ... },           // Single object or array
  "summary": "4 boards",     // Human-readable description
  "breadcrumbs": [ ... ],    // Contextual next actions (omitted when empty)
  "context": { ... },        // Location, pagination, and other context (omitted when empty)
  "meta": {
  }
}
```

**Summary field formats:**
| Command | Example Summary |
|---------|-----------------|
| `board list` | "5 boards" |
| `board show ID` | "Board: Engineering" |
| `card list` | "42 cards (page 1)" or "42 cards (all)" |
| `card show 123` | "Card #123: Fix login bug" |
| `search "bug"` | "7 results for \"bug\"" |
| `notification list` | "8 notifications (3 unread)" |

**List responses with pagination:**
```json
{
  "ok": true,
  "data": [ ... ],
  "summary": "10 cards (page 1)",
  "context": {
    "pagination": {
      "has_next": true,
      "next_url": "https://..."
    }
  }
}
```

**Breadcrumbs (contextual next actions):**

Responses include a `breadcrumbs` array suggesting what you can do next. Each breadcrumb has:
- `action`: Short action name (e.g., "comment", "close", "assign")
- `cmd`: Ready-to-run command with actual values interpolated
- `description`: Human-readable description

```bash
fizzy card show 42 | jq '.breadcrumbs'
```

```json
[
  {"action": "comment", "cmd": "fizzy comment create --card 42 --body \"text\"", "description": "Add comment"},
  {"action": "triage", "cmd": "fizzy card column 42 --column <column_id>", "description": "Move to column"},
  {"action": "close", "cmd": "fizzy card close 42", "description": "Close card"},
  {"action": "assign", "cmd": "fizzy card assign 42 --user <user_id>", "description": "Assign user"}
]
```

Use breadcrumbs to discover available actions without memorizing the full CLI. Values like card numbers and board IDs are pre-filled; placeholders like `<column_id>` need to be replaced.

**Create/update responses include location:**
```json
{
  "ok": true,
  "data": { ... },
  "context": {
    "location": "/6102600/cards/579.json"
  }
}
```

---

## ID Formats

**IMPORTANT:** Cards use TWO identifiers:

| Field | Format | Use For |
|-------|--------|---------|
| `id` | `03fe4rug9kt1mpgyy51lq8i5i` | Internal ID (in JSON responses) |
| `number` | `579` | CLI commands (`card show`, `card update`, etc.) |

**All card CLI commands use the card NUMBER, not the ID.**

Other resources (boards, columns, comments, steps, reactions, users) use their `id` field.

---

## Card Statuses

Cards exist in different states. By default, `fizzy card list` returns **open cards only** (cards in triage or columns). To fetch cards in other states, use the `--indexed-by` or `--column` flags:

| Status | How to fetch | Description |
|--------|--------------|-------------|
| Open (default) | `fizzy card list` | Cards in triage ("Maybe?") or any column |
| Closed/Done | `fizzy card list --indexed-by closed` | Completed cards |
| Not Now | `fizzy card list --indexed-by not_now` | Postponed cards |
| Golden | `fizzy card list --indexed-by golden` | Starred/important cards |
| Stalled | `fizzy card list --indexed-by stalled` | Cards with no recent activity |

You can also use pseudo-columns:

```bash
fizzy card list --column done --all     # Same as --indexed-by closed
fizzy card list --column not-now --all  # Same as --indexed-by not_now
fizzy card list --column maybe --all    # Cards in triage (no column assigned)
```

**Fetching all cards on a board:**

To get all cards regardless of status (for example, to build a complete board view), make separate queries:

```bash
# Open cards (triage + columns)
fizzy card list --board BOARD_ID --all

# Closed/Done cards
fizzy card list --board BOARD_ID --indexed-by closed --all

# Optionally, Not Now cards
fizzy card list --board BOARD_ID --indexed-by not_now --all
```

---

## Common jq Patterns

### Reducing Output

```bash
# Card summary (most useful)
fizzy card list | jq '[.data[] | {number, title, status, board: .board.name}]'

# First N items
fizzy card list | jq '.data[:5]'

# Just IDs
fizzy board list | jq '[.data[].id]'

# Specific fields from single item
fizzy card show 579 | jq '.data | {number, title, status, golden}'

# Card with description length (description is a string, not object)
fizzy card show 579 | jq '.data | {number, title, desc_length: (.description | length)}'
```

### Filtering

```bash
# Cards with a specific status
fizzy card list --all | jq '[.data[] | select(.status == "published")]'

# Golden cards only
fizzy card list --indexed-by golden | jq '[.data[] | {number, title}]'

# Cards with non-empty descriptions
fizzy card list | jq '[.data[] | select(.description | length > 0) | {number, title}]'

# Cards with steps (must use card show, steps not in list)
fizzy card show 579 | jq '.data.steps'
```

### Extracting Nested Data

```bash
# Comment text only (body.plain_text for comments)
fizzy comment list --card 579 | jq '[.data[].body.plain_text]'

# Card description (just .description for cards - it's a string)
fizzy card show 579 | jq '.data.description'

# Step completion status
fizzy card show 579 | jq '[.data.steps[] | {content, completed}]'
```

### Activity Analysis

```bash
# Cards with steps count (requires card show for each)
fizzy card show 579 | jq '.data | {number, title, steps_count: (.steps | length)}'

# Comments count for a card
fizzy comment list --card 579 | jq '.data | length'
```

---

## Command Reference

### Identity

```bash
fizzy identity show                    # Show your identity and accessible accounts
```

### Search

Quick text search across cards. Multiple words are treated as separate terms (AND).

```bash
fizzy search QUERY [flags]
  --board ID                           # Filter by board
  --assignee ID                        # Filter by assignee user ID
  --tag ID                             # Filter by tag ID
  --indexed-by LANE                    # Filter: all, closed, not_now, golden
  --sort ORDER                         # Sort: newest, oldest, or latest (default)
  --page N                             # Page number
  --all                                # Fetch all pages
```

**Examples:**
```bash
fizzy search "bug"                     # Search for "bug"
fizzy search "login error"             # Search for cards containing both "login" AND "error"
fizzy search "bug" --board BOARD_ID    # Search within a specific board
fizzy search "bug" --indexed-by closed # Include closed cards
fizzy search "feature" --sort newest   # Sort by newest first
```

### Boards

```bash
fizzy board list [--page N] [--all]
fizzy board show BOARD_ID
fizzy board create --name "Name" [--all_access true/false] [--auto_postpone_period N]
fizzy board update BOARD_ID [--name "Name"] [--all_access true/false] [--auto_postpone_period N]
fizzy board delete BOARD_ID
```

### Board Migration

Migrate boards between accounts (e.g., from personal to team account).

```bash
fizzy migrate board BOARD_ID --from SOURCE_SLUG --to TARGET_SLUG [flags]
  --include-images                       # Migrate card header images and inline attachments
  --include-comments                     # Migrate card comments
  --include-steps                        # Migrate card steps (to-do items)
  --dry-run                              # Preview migration without making changes
```

**What gets migrated:**
- Board with same name
- All columns (preserving order and colors)
- All cards with titles, descriptions, timestamps, and tags
- Card states (closed, golden, column placement)
- Optional: header images, inline attachments, comments, and steps

**What cannot be migrated:**
- Card creators (become the migrating user)
- Card numbers (new sequential numbers in target)
- Comment authors (become the migrating user)
- User assignments (team must reassign manually)

**Requirements:** You must have API access to both source and target accounts. Verify with `fizzy identity show`.

```bash
# Preview migration first
fizzy migrate board BOARD_ID --from personal --to team-account --dry-run

# Basic migration
fizzy migrate board BOARD_ID --from personal --to team-account

# Full migration with all content
fizzy migrate board BOARD_ID --from personal --to team-account \
  --include-images --include-comments --include-steps
```

### Cards

#### Listing & Viewing

```bash
fizzy card list [flags]
  --board ID                           # Filter by board
  --column ID                          # Filter by column ID or pseudo: not-now, maybe, done
  --assignee ID                        # Filter by assignee user ID
  --tag ID                             # Filter by tag ID
  --indexed-by LANE                    # Filter: all, closed, not_now, stalled, postponing_soon, golden
  --search "terms"                     # Search by text (space-separated for multiple terms)
  --sort ORDER                         # Sort: newest, oldest, or latest (default)
  --creator ID                         # Filter by creator user ID
  --closer ID                          # Filter by user who closed the card
  --unassigned                         # Only show unassigned cards
  --created PERIOD                     # Filter by creation: today, yesterday, thisweek, lastweek, thismonth, lastmonth
  --closed PERIOD                      # Filter by closure: today, yesterday, thisweek, lastweek, thismonth, lastmonth
  --page N                             # Page number
  --all                                # Fetch all pages

fizzy card show CARD_NUMBER            # Show card details (includes steps)
```

#### Creating & Updating

```bash
fizzy card create --board ID --title "Title" [flags]
  --description "HTML"                 # Card description (HTML)
  --description_file PATH              # Read description from file
  --image SIGNED_ID                    # Header image (use signed_id from upload)
  --tag-ids "id1,id2"                  # Comma-separated tag IDs
  --created-at TIMESTAMP               # Custom created_at

fizzy card update CARD_NUMBER [flags]
  --title "Title"
  --description "HTML"
  --description_file PATH
  --image SIGNED_ID
  --created-at TIMESTAMP

fizzy card delete CARD_NUMBER
```

#### Status Changes

```bash
fizzy card close CARD_NUMBER           # Close card (sets closed: true)
fizzy card reopen CARD_NUMBER          # Reopen closed card
fizzy card postpone CARD_NUMBER        # Move to Not Now lane
fizzy card untriage CARD_NUMBER        # Remove from column, back to triage
```

**Note:** Card `status` field stays "published" for active cards. Use:
- `closed: true/false` to check if closed
- `--indexed-by not_now` to find postponed cards
- `--indexed-by closed` to find closed cards

#### Actions

```bash
fizzy card column CARD_NUMBER --column ID     # Move to column (use column ID or: maybe, not-now, done)
fizzy card move CARD_NUMBER --to BOARD_ID     # Move card to a different board
fizzy card assign CARD_NUMBER --user ID       # Toggle user assignment
fizzy card self-assign CARD_NUMBER            # Toggle current user's assignment
fizzy card tag CARD_NUMBER --tag "name"       # Toggle tag (creates tag if needed)
fizzy card watch CARD_NUMBER                  # Subscribe to notifications
fizzy card unwatch CARD_NUMBER                # Unsubscribe
fizzy card pin CARD_NUMBER                    # Pin card for quick access
fizzy card unpin CARD_NUMBER                  # Unpin card
fizzy card golden CARD_NUMBER                 # Mark as golden/starred
fizzy card ungolden CARD_NUMBER               # Remove golden status
fizzy card image-remove CARD_NUMBER           # Remove header image
```

#### Attachments

```bash
fizzy card attachments show CARD_NUMBER [--include-comments]           # List attachments
fizzy card attachments download CARD_NUMBER [INDEX] [--include-comments]  # Download (1-based index)
  -o, --output FILENAME                                    # Exact name (single) or prefix (multiple: test_1.png, test_2.png)
```

### Columns

Boards have pseudo columns by default: `not-now`, `maybe`, `done`

```bash
fizzy column list --board ID
fizzy column show COLUMN_ID --board ID
fizzy column create --board ID --name "Name" [--color HEX]
fizzy column update COLUMN_ID --board ID [--name "Name"] [--color HEX]
fizzy column delete COLUMN_ID --board ID
```

### Comments

```bash
fizzy comment list --card NUMBER [--page N] [--all]
fizzy comment show COMMENT_ID --card NUMBER
fizzy comment create --card NUMBER --body "HTML" [--body_file PATH] [--created-at TIMESTAMP]
fizzy comment update COMMENT_ID --card NUMBER [--body "HTML"] [--body_file PATH]
fizzy comment delete COMMENT_ID --card NUMBER
```

#### Comment Attachments

```bash
fizzy comment attachments show --card NUMBER                  # List attachments in comments
fizzy comment attachments download --card NUMBER [INDEX]      # Download (1-based index)
  -o, --output FILENAME                                       # Exact name (single) or prefix (multiple: test_1.png, test_2.png)
```

### Steps (To-Do Items)

Steps are returned in `card show` response. No separate list command.

```bash
fizzy step show STEP_ID --card NUMBER
fizzy step create --card NUMBER --content "Text" [--completed]
fizzy step update STEP_ID --card NUMBER [--content "Text"] [--completed] [--not_completed]
fizzy step delete STEP_ID --card NUMBER
```

### Reactions

Reactions can be added to cards directly or to comments on cards.

```bash
# Card reactions (react directly to a card)
fizzy reaction list --card NUMBER
fizzy reaction create --card NUMBER --content "emoji"
fizzy reaction delete REACTION_ID --card NUMBER

# Comment reactions (react to a specific comment)
fizzy reaction list --card NUMBER --comment COMMENT_ID
fizzy reaction create --card NUMBER --comment COMMENT_ID --content "emoji"
fizzy reaction delete REACTION_ID --card NUMBER --comment COMMENT_ID
```

| Flag | Required | Description |
|------|----------|-------------|
| `--card` | Yes | Card number (always required) |
| `--comment` | No | Comment ID (omit for card reactions) |
| `--content` | Yes (create) | Emoji or text, max 16 characters |

### Tags

Tags are created automatically when using `card tag`. List shows all existing tags.

```bash
fizzy tag list [--page N] [--all]
```

### Users

```bash
fizzy user list [--page N] [--all]
fizzy user show USER_ID
fizzy user update USER_ID --name "Name"       # Update user name (requires admin/owner)
fizzy user update USER_ID --avatar /path.jpg  # Update user avatar
fizzy user deactivate USER_ID                  # Deactivate user (requires admin/owner)
```

### Pins

```bash
fizzy pin list                                 # List your pinned cards (up to 100)
```

### Notifications

```bash
fizzy notification list [--page N] [--all]
fizzy notification tray                    # Unread notifications (up to 100)
fizzy notification tray --include-read     # Include read notifications
fizzy notification read NOTIFICATION_ID
fizzy notification read-all
fizzy notification unread NOTIFICATION_ID
```

### File Uploads

```bash
fizzy upload file PATH
# Returns: { "signed_id": "...", "attachable_sgid": "..." }
```

| ID | Use For |
|---|---|
| `signed_id` | Card header/background images (`--image` flag) |
| `attachable_sgid` | Inline images in rich text (descriptions, comments) |

---

## Common Workflows

### Create Card with Steps

```bash
# Create the card
CARD=$(fizzy card create --board BOARD_ID --title "New Feature" \
  --description "<p>Feature description</p>" | jq -r '.data.number')

# Add steps
fizzy step create --card $CARD --content "Design the feature"
fizzy step create --card $CARD --content "Implement backend"
fizzy step create --card $CARD --content "Write tests"
```

### Link Code to Card

```bash
# Add a comment linking the commit
fizzy comment create --card 42 --body "<p>Commit $(git rev-parse --short HEAD): $(git log -1 --format=%s)</p>"

# Close the card when done
fizzy card close 42
```

### Create Card with Inline Image

```bash
# Upload image
SGID=$(fizzy upload file screenshot.png | jq -r '.data.attachable_sgid')

# Create description file with embedded image
cat > desc.html << EOF
<p>See the screenshot below:</p>
<action-text-attachment sgid="$SGID"></action-text-attachment>
EOF

# Create card
fizzy card create --board BOARD_ID --title "Bug Report" --description_file desc.html
```

### Create Card with Background Image (only when explicitly requested)

```bash
# Validate file is an image
MIME=$(file --mime-type -b /path/to/image.png)
if [[ ! "$MIME" =~ ^image/ ]]; then
  echo "Error: Not a valid image (detected: $MIME)"
  exit 1
fi

# Upload and get signed_id
SIGNED_ID=$(fizzy upload file /path/to/header.png | jq -r '.data.signed_id')

# Create card with background
fizzy card create --board BOARD_ID --title "Card" --image "$SIGNED_ID"
```

### Move Card Through Workflow

```bash
# Move to a column
fizzy card column 579 --column maybe

# Assign to yourself
fizzy card self-assign 579

# Or assign to another user
fizzy card assign 579 --user USER_ID

# Mark as golden (important)
fizzy card golden 579

# When done, close it
fizzy card close 579
```

### Move Card to Different Board

```bash
# Move card to another board
fizzy card move 579 --to TARGET_BOARD_ID
```

### Search and Filter Cards

```bash
# Quick search
fizzy search "bug" | jq '[.data[] | {number, title}]'

# Search with filters
fizzy search "login" --board BOARD_ID --sort newest

# Find recently created cards
fizzy card list --created today --sort newest

# Find cards closed this week
fizzy card list --indexed-by closed --closed thisweek

# Find unassigned cards
fizzy card list --unassigned --board BOARD_ID
```

### React to a Card

```bash
# Add reaction directly to a card
fizzy reaction create --card 579 --content "👍"

# List reactions on a card
fizzy reaction list --card 579 | jq '[.data[] | {id, content, reacter: .reacter.name}]'
```

### Add Comment with Reaction

```bash
# Add comment
COMMENT=$(fizzy comment create --card 579 --body "<p>Looks good!</p>" | jq -r '.data.id')

# Add reaction to the comment
fizzy reaction create --card 579 --comment $COMMENT --content "👍"
```

---

## Resource Schemas

Complete field reference for all resources. Use these exact field paths in jq queries.

### Card Schema

**IMPORTANT:** `card list` and `card show` return different fields. `steps` only in `card show`.

| Field | Type | Description |
|-------|------|-------------|
| `number` | integer | **Use this for CLI commands** |
| `id` | string | Internal ID (in responses only) |
| `title` | string | Card title |
| `description` | string | Plain text content (**NOT an object**) |
| `description_html` | string | HTML version with attachments |
| `status` | string | Usually "published" for active cards |
| `closed` | boolean | true = card is closed |
| `golden` | boolean | true = starred/important |
| `image_url` | string/null | Header/background image URL |
| `has_attachments` | boolean | true = card has file attachments |
| `has_more_assignees` | boolean | More assignees than shown |
| `created_at` | timestamp | ISO 8601 |
| `last_active_at` | timestamp | ISO 8601 |
| `url` | string | Web URL |
| `comments_url` | string | Comments endpoint URL |
| `board` | object | Nested Board (see below) |
| `creator` | object | Nested User (see below) |
| `assignees` | array | Array of User objects |
| `tags` | array | Array of Tag objects |
| `steps` | array | **Only in `card show`**, not in list |

### Board Schema

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Board ID (use for CLI commands) |
| `name` | string | Board name |
| `all_access` | boolean | All users have access |
| `created_at` | timestamp | ISO 8601 |
| `url` | string | Web URL |
| `creator` | object | Nested User |

### User Schema

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | User ID (use for CLI commands) |
| `name` | string | Display name |
| `email_address` | string | Email |
| `role` | string | "owner", "admin", or "member" |
| `active` | boolean | Account is active |
| `created_at` | timestamp | ISO 8601 |
| `url` | string | Web URL |

### Comment Schema

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Comment ID (use for CLI commands) |
| `body` | object | **Nested object with html and plain_text** |
| `body.html` | string | HTML content |
| `body.plain_text` | string | Plain text content |
| `created_at` | timestamp | ISO 8601 |
| `updated_at` | timestamp | ISO 8601 |
| `url` | string | Web URL |
| `reactions_url` | string | Reactions endpoint URL |
| `creator` | object | Nested User |
| `card` | object | Nested {id, url} |

### Step Schema

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Step ID (use for CLI commands) |
| `content` | string | Step text |
| `completed` | boolean | Completion status |

### Column Schema

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Column ID or pseudo ID ("not-now", "maybe", "done") |
| `name` | string | Display name |
| `kind` | string | "not_now", "triage", "closed", or custom |
| `pseudo` | boolean | true = built-in column |

### Tag Schema

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Tag ID |
| `title` | string | Tag name |
| `created_at` | timestamp | ISO 8601 |
| `url` | string | Web URL |

### Reaction Schema

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Reaction ID (use for CLI commands) |
| `content` | string | Emoji |
| `url` | string | Web URL |
| `reacter` | object | Nested User |

### Identity Schema (from `identity show`)

| Field | Type | Description |
|-------|------|-------------|
| `accounts` | array | Array of Account objects |
| `accounts[].id` | string | Account ID |
| `accounts[].name` | string | Account name |
| `accounts[].slug` | string | Account slug (use with `signup complete --account` or as profile name) |
| `accounts[].user` | object | Your User in this account |

### Key Schema Differences

| Resource | Text Field | HTML Field |
|----------|------------|------------|
| Card | `.description` (string) | `.description_html` (string) |
| Comment | `.body.plain_text` (nested) | `.body.html` (nested) |

---

## Rich Text Formatting

Card descriptions and comments support HTML. For multiple paragraphs with spacing:

```html
<p>First paragraph.</p>
<p><br></p>
<p>Second paragraph with spacing above.</p>
```

**Note:** Each `attachable_sgid` can only be used once. Upload the file again for multiple uses.

## Default Behaviors

- **Card images:** Use inline (via `attachable_sgid` in description) by default. Only use background/header (`signed_id` with `--image`) when user explicitly says "background" or "header".
- **Comment images:** Always inline. Comments do not support background images.

---

## Error Handling

**Error response format:**
```json
{
  "ok": false,
  "error": "Not Found",
  "code": "not_found",
  "hint": "optional context"
}
```

**Exit codes:**

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Usage / invalid arguments |
| 2 | Not found |
| 3 | Authentication failure |
| 4 | Permission denied |
| 5 | Rate limited |
| 6 | Network error |
| 7 | API / server error |
| 8 | Ambiguous match |

**Authentication errors (exit 3):**
```bash
fizzy auth status                        # Check auth
fizzy auth list                          # Check which profiles are configured
fizzy auth switch PROFILE                # Switch to correct profile
fizzy auth login TOKEN                   # Re-authenticate
fizzy setup                              # Full interactive setup
```

**Not found errors (exit 2):** Verify the card number or resource ID is correct. Cards use NUMBER, not ID.

**Permission denied (exit 4):** Some operations (user update, user deactivate) require admin/owner role.

**Network errors (exit 6):** Check API URL configuration:
```bash
fizzy auth status                        # Shows configured profile and API URL
```

## Learn More

- API documentation: https://github.com/basecamp/fizzy/blob/main/docs/API.md
- CLI repository: https://github.com/basecamp/fizzy-cli
