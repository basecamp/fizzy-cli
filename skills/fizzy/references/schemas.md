# Fizzy CLI — Resource Schemas & jq Patterns

Complete field reference for all resources and common jq patterns for parsing CLI output.

## Table of Contents

- [Card Schema](#card-schema)
- [Board Schema](#board-schema)
- [Account Settings Schema](#account-settings-schema)
- [User Schema](#user-schema)
- [Comment Schema](#comment-schema)
- [Step Schema](#step-schema)
- [Column Schema](#column-schema)
- [Tag Schema](#tag-schema)
- [Reaction Schema](#reaction-schema)
- [Identity Schema](#identity-schema)
- [Webhook Schema](#webhook-schema)
- [Key Schema Differences](#key-schema-differences)
- [jq Patterns](#jq-patterns)

---

## Card Schema

`card list` and `card show` return different fields. `steps` only appears in `card show`.

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

## Board Schema

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Board ID (use for CLI commands) |
| `name` | string | Board name |
| `all_access` | boolean | All users have access |
| `auto_postpone_period_in_days` | integer | Days of inactivity before cards are auto-postponed |
| `created_at` | timestamp | ISO 8601 |
| `url` | string | Web URL |
| `creator` | object | Nested User |

## Account Settings Schema

From `account show`:

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Account ID |
| `name` | string | Account name |
| `cards_count` | integer | Total cards in account |
| `auto_postpone_period_in_days` | integer | Account-level default auto-postpone period |
| `created_at` | timestamp | ISO 8601 |

## User Schema

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | User ID (use for CLI commands) |
| `name` | string | Display name |
| `email_address` | string | Email |
| `role` | string | "owner", "admin", or "member" |
| `active` | boolean | Account is active |
| `created_at` | timestamp | ISO 8601 |
| `url` | string | Web URL |

## Comment Schema

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

## Step Schema

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Step ID (use for CLI commands) |
| `content` | string | Step text |
| `completed` | boolean | Completion status |

## Column Schema

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Column ID or pseudo ID ("not-now", "maybe", "done") |
| `name` | string | Display name |
| `kind` | string | "not_now", "triage", "closed", or custom |
| `pseudo` | boolean | true = built-in column |

## Tag Schema

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Tag ID |
| `title` | string | Tag name |
| `created_at` | timestamp | ISO 8601 |
| `url` | string | Web URL |

## Reaction Schema

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Reaction ID (use for CLI commands) |
| `content` | string | Emoji |
| `url` | string | Web URL |
| `reacter` | object | Nested User |

## Identity Schema

From `identity show`:

| Field | Type | Description |
|-------|------|-------------|
| `accounts` | array | Array of Account objects |
| `accounts[].id` | string | Account ID |
| `accounts[].name` | string | Account name |
| `accounts[].slug` | string | Account slug (use with `signup complete --account` or as profile name) |
| `accounts[].user` | object | Your User in this account |

## Webhook Schema

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Webhook ID (use for CLI commands) |
| `name` | string | Webhook name |
| `payload_url` | string | Destination URL |
| `active` | boolean | Whether webhook is active |
| `signing_secret` | string | Secret for verifying payloads |
| `subscribed_actions` | array | List of subscribed event actions |
| `created_at` | timestamp | ISO 8601 |
| `url` | string | API URL |
| `board` | object | Nested Board |

---

## Key Schema Differences

| Resource | Text Field | HTML Field |
|----------|------------|------------|
| Card | `.description` (string) | `.description_html` (string) |
| Comment | `.body.plain_text` (nested) | `.body.html` (nested) |

---

## jq Patterns

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

# Card with description length
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

# Steps for a card (must use card show, steps not in list)
fizzy card show 579 | jq '.data.steps'
```

### Extracting Nested Data

```bash
# Comment text only (body.plain_text for comments)
fizzy comment list --card 579 | jq '[.data[].body.plain_text]'

# Card description (just .description — it's a string, not an object)
fizzy card show 579 | jq '.data.description'

# Step completion status
fizzy card show 579 | jq '[.data.steps[] | {content, completed}]'
```

### Activity Analysis

```bash
# Steps count for a card (requires card show)
fizzy card show 579 | jq '.data | {number, title, steps_count: (.steps | length)}'

# Comments count
fizzy comment list --card 579 | jq '.data | length'

# Breadcrumbs (available next actions)
fizzy card show 42 | jq '.breadcrumbs'
```
