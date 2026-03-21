# Fizzy CLI — Full Command Reference

Detailed syntax and flags for every command. See the main SKILL.md for decision trees, key behaviors, and common workflows.

## Table of Contents

- [Identity](#identity)
- [Account](#account)
- [Search](#search)
- [Boards](#boards)
- [Board Migration](#board-migration)
- [Cards](#cards)
- [Columns](#columns)
- [Comments](#comments)
- [Steps](#steps)
- [Reactions](#reactions)
- [Tags](#tags)
- [Users](#users)
- [Pins](#pins)
- [Notifications](#notifications)
- [Webhooks](#webhooks)
- [File Uploads](#file-uploads)

---

## Identity

```bash
fizzy identity show                    # Show your identity and accessible accounts
```

## Account

```bash
fizzy account show                     # Show account settings (name, auto-postpone period)
fizzy account settings-update --name "Name"            # Update account name
fizzy account entropy --auto_postpone_period_in_days N  # Update account default auto-postpone period (admin only, N: 3, 7, 11, 30, 90, 365)
fizzy account export-create                            # Create data export
fizzy account export-show EXPORT_ID                    # Check export status
fizzy account join-code-show                           # Show join code
fizzy account join-code-reset                          # Reset join code
fizzy account join-code-update --usage-limit N         # Update join code limit
```

The `auto_postpone_period_in_days` is the account-level default. Cards are automatically moved to "Not Now" after this period of inactivity. Each board can override this with `board entropy`.

## Search

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
fizzy search "login error"             # Cards containing both "login" AND "error"
fizzy search "bug" --board BOARD_ID    # Search within a specific board
fizzy search "bug" --indexed-by closed # Include closed cards
fizzy search "feature" --sort newest   # Sort by newest first
```

## Boards

```bash
fizzy board list [--page N] [--all]
fizzy board show BOARD_ID
fizzy board create --name "Name" [--all_access true/false] [--auto_postpone_period_in_days N]
fizzy board update BOARD_ID [--name "Name"] [--all_access true/false] [--auto_postpone_period_in_days N]
fizzy board publish BOARD_ID
fizzy board unpublish BOARD_ID
fizzy board delete BOARD_ID
fizzy board entropy BOARD_ID --auto_postpone_period_in_days N  # N: 3, 7, 11, 30, 90, 365
fizzy board closed --board ID [--page N] [--all]       # List closed cards
fizzy board postponed --board ID [--page N] [--all]    # List postponed cards
fizzy board stream --board ID [--page N] [--all]       # List stream cards
fizzy board involvement BOARD_ID --involvement LEVEL   # Update your involvement
```

`board show` includes `public_url` only when the board is published.
`board entropy` updates the auto-postpone period for a specific board (overrides account default). Requires board admin.

## Board Migration

Migrate boards between accounts (e.g., from personal to team account).

```bash
fizzy migrate board BOARD_ID --from SOURCE_SLUG --to TARGET_SLUG [flags]
  --include-images                       # Migrate card header images and inline attachments
  --include-comments                     # Migrate card comments
  --include-steps                        # Migrate card steps (to-do items)
  --dry-run                              # Preview migration without making changes
```

**What gets migrated:**
- Board with same name, all columns (preserving order and colors)
- All cards with titles, descriptions, timestamps, and tags
- Card states (closed, golden, column placement)
- Optional: header images, inline attachments, comments, and steps

**What cannot be migrated:**
- Card creators/comment authors (become the migrating user)
- Card numbers (new sequential numbers in target)
- User assignments (team must reassign manually)

**Requirements:** You must have API access to both source and target accounts. Verify with `fizzy identity show`.

```bash
# Preview first
fizzy migrate board BOARD_ID --from personal --to team-account --dry-run

# Full migration with all content
fizzy migrate board BOARD_ID --from personal --to team-account \
  --include-images --include-comments --include-steps
```

## Cards

### Listing & Viewing

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

### Creating & Updating

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

### Status Changes

```bash
fizzy card close CARD_NUMBER           # Close card (sets closed: true)
fizzy card reopen CARD_NUMBER          # Reopen closed card
fizzy card postpone CARD_NUMBER        # Move to Not Now lane
fizzy card untriage CARD_NUMBER        # Remove from column, back to triage
```

Card `status` field stays "published" for active cards. Use `closed: true/false` to check if closed, and `--indexed-by` flags to filter by state.

### Actions

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
fizzy card publish CARD_NUMBER               # Publish a card
fizzy card mark-read CARD_NUMBER             # Mark card as read
fizzy card mark-unread CARD_NUMBER           # Mark card as unread
```

### Attachments

```bash
fizzy card attachments show CARD_NUMBER [--include-comments]           # List attachments
fizzy card attachments download CARD_NUMBER [INDEX] [--include-comments]  # Download (1-based index)
  -o, --output FILENAME                                    # Exact name (single) or prefix (multiple: test_1.png, test_2.png)
```

## Columns

Boards have pseudo columns by default: `not-now`, `maybe`, `done`

```bash
fizzy column list --board ID
fizzy column show COLUMN_ID --board ID
fizzy column create --board ID --name "Name" [--color HEX]
fizzy column update COLUMN_ID --board ID [--name "Name"] [--color HEX]
fizzy column delete COLUMN_ID --board ID
fizzy column move-left COLUMN_ID             # Move column one position left
fizzy column move-right COLUMN_ID            # Move column one position right
```

## Comments

```bash
fizzy comment list --card NUMBER [--page N] [--all]
fizzy comment show COMMENT_ID --card NUMBER
fizzy comment create --card NUMBER --body "HTML" [--body_file PATH] [--created-at TIMESTAMP]
fizzy comment update COMMENT_ID --card NUMBER [--body "HTML"] [--body_file PATH]
fizzy comment delete COMMENT_ID --card NUMBER
```

### Comment Attachments

```bash
fizzy comment attachments show --card NUMBER                  # List attachments in comments
fizzy comment attachments download --card NUMBER [INDEX]      # Download (1-based index)
  -o, --output FILENAME                                       # Exact name (single) or prefix (multiple)
```

## Steps

Steps are returned in `card show` response but can also be listed separately.

```bash
fizzy step list --card NUMBER
fizzy step show STEP_ID --card NUMBER
fizzy step create --card NUMBER --content "Text" [--completed]
fizzy step update STEP_ID --card NUMBER [--content "Text"] [--completed] [--not_completed]
fizzy step delete STEP_ID --card NUMBER
```

## Reactions

Reactions can be added to cards directly or to comments on cards.

```bash
# Card reactions
fizzy reaction list --card NUMBER
fizzy reaction create --card NUMBER --content "emoji"
fizzy reaction delete REACTION_ID --card NUMBER

# Comment reactions
fizzy reaction list --card NUMBER --comment COMMENT_ID
fizzy reaction create --card NUMBER --comment COMMENT_ID --content "emoji"
fizzy reaction delete REACTION_ID --card NUMBER --comment COMMENT_ID
```

| Flag | Required | Description |
|------|----------|-------------|
| `--card` | Yes | Card number (always required) |
| `--comment` | No | Comment ID (omit for card reactions) |
| `--content` | Yes (create) | Emoji or text, max 16 characters |

## Tags

Tags are created automatically when using `card tag`. List shows all existing tags.

```bash
fizzy tag list [--page N] [--all]
```

## Users

```bash
fizzy user list [--page N] [--all]
fizzy user show USER_ID
fizzy user update USER_ID --name "Name"       # Update user name (requires admin/owner)
fizzy user update USER_ID --avatar /path.jpg  # Update user avatar
fizzy user deactivate USER_ID                  # Deactivate user (requires admin/owner)
fizzy user role USER_ID --role ROLE            # Update user role (requires admin/owner)
fizzy user avatar-remove USER_ID               # Remove user avatar
fizzy user push-subscription-create --user ID --endpoint URL --p256dh-key KEY --auth-key KEY
fizzy user push-subscription-delete SUB_ID --user ID
```

## Pins

```bash
fizzy pin list                                 # List your pinned cards (up to 100)
```

## Notifications

```bash
fizzy notification list [--page N] [--all]
fizzy notification tray                    # Unread notifications (up to 100)
fizzy notification tray --include-read     # Include read notifications
fizzy notification read NOTIFICATION_ID
fizzy notification read-all
fizzy notification unread NOTIFICATION_ID
fizzy notification settings-show              # Show notification settings
fizzy notification settings-update --bundle-email-frequency FREQ  # Update settings
```

## Webhooks

Webhooks notify external services when events occur on a board. Requires account admin access.

```bash
fizzy webhook list --board ID [--page N] [--all]
fizzy webhook show WEBHOOK_ID --board ID
fizzy webhook create --board ID --name "Name" --url "https://..." [--actions card_published,card_closed,...]
fizzy webhook update WEBHOOK_ID --board ID [--name "Name"] [--actions card_closed,...]
fizzy webhook delete WEBHOOK_ID --board ID
fizzy webhook reactivate WEBHOOK_ID --board ID    # Reactivate a deactivated webhook
```

**Supported actions:** `card_assigned`, `card_closed`, `card_postponed`, `card_auto_postponed`, `card_board_changed`, `card_published`, `card_reopened`, `card_sent_back_to_triage`, `card_triaged`, `card_unassigned`, `comment_created`

Webhook URL is immutable after creation. Use `--actions` with comma-separated values.

## File Uploads

```bash
fizzy upload file PATH
# Returns: { "signed_id": "...", "attachable_sgid": "..." }
```

| ID | Use For |
|---|---|
| `signed_id` | Card header/background images (`--image` flag) |
| `attachable_sgid` | Inline images in rich text (descriptions, comments) |

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

### React to a Card or Comment

```bash
# React to card
fizzy reaction create --card 579 --content "👍"

# React to comment
COMMENT=$(fizzy comment create --card 579 --body "<p>Looks good!</p>" | jq -r '.data.id')
fizzy reaction create --card 579 --comment $COMMENT --content "👍"
```
