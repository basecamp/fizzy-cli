#!/usr/bin/env bash
set -euo pipefail

# Sync this CLI's skills to the basecamp/skills distribution repo.
#
# basecamp/skills is shared: every CLI (basecamp-cli, hey-cli, ...) publishes
# its skills as skills/<name>/ at the target root. Each publisher owns one
# manifest there, .managed-skills.<source>, listing the names it published. On
# a sync it removes a skills/<name> only when its own manifest lists the name,
# its current skills tree no longer has it, and no other publisher's manifest
# claims it. A publisher with no manifest yet removes nothing. That is what
# stops one CLI's release from deleting another's skills (basecamp/skills#5:
# the CLIs used to share a single .managed-skills and each pruned the other).
#
# The legacy .managed-skills is rewritten on every run as a comment-only
# tombstone. A CLI still running the pre-fix script skips every line it cannot
# parse as a skill name, so it deletes nothing — whereas if the file were
# removed, that script would fall back to claiming every skills/*/ as its own.
#
# Env vars:
#   SKILLS_TOKEN  - GitHub token with push access to basecamp/skills (required
#                   when cloning, i.e. unless DRY_RUN=local or SKILLS_TARGET is set)
#   RELEASE_TAG   - Release tag, e.g. v1.2.3 (required)
#   SOURCE_SHA    - Source commit SHA (required)
#   SKILLS_SOURCE - Optional: directory holding the skills tree (default: skills)
#   SKILLS_TARGET - Optional: existing basecamp/skills checkout to sync into
#                   instead of cloning (tests). The remote and branch asserts
#                   still apply to it.
#   SYNC_SOURCE   - Optional: the publishing repo's name (default: fizzy-cli).
#                   Names the manifest, the bot and the commit; tests use it to
#                   play another CLI.
#   DRY_RUN       - Optional: "local" (no network) or "remote" (clone but skip push).
#                   With SKILLS_TARGET, "local" applies and commits but skips the push.

RELEASE_TAG="${RELEASE_TAG:?RELEASE_TAG is required}"
SOURCE_SHA="${SOURCE_SHA:?SOURCE_SHA is required}"
DRY_RUN="${DRY_RUN:-}"

# The skills tree to mirror. The manual recovery workflow (sync-skills.yml)
# points this at a separate checkout of the release tag, so the sync logic can
# come from a newer ref (with fixes) than the content it mirrors.
SKILLS_SOURCE="${SKILLS_SOURCE:-skills}"
SKILLS_TARGET="${SKILLS_TARGET:-}"
SYNC_SOURCE="${SYNC_SOURCE:-fizzy-cli}"
TARGET_REPO="basecamp/skills"
TARGET_BRANCH="main"
SKILLS_SUBDIR="skills"
LEGACY_MANIFEST=".managed-skills"
MANIFEST="$LEGACY_MANIFEST.$SYNC_SOURCE"

# --- Helpers ---

die() { echo "ERROR: $*" >&2; exit 1; }

assert_remote_url() {
  local url
  url=$(git -C "$1" remote get-url origin)
  local stripped
  stripped=$(echo "$url" | sed -E 's/\.git$//')
  # Validate host + owner/repo for both HTTPS and SSH forms
  case "$stripped" in
    https://github.com/"$TARGET_REPO") ;;
    https://x-access-token:*@github.com/"$TARGET_REPO") ;;
    git@github.com:"$TARGET_REPO") ;;
    *) die "origin remote '$(echo "$url" | sed -E 's#(https://[^:@]+:)[^@]*@#\1***@#')' does not point to github.com/$TARGET_REPO" ;;
  esac
}

assert_branch() {
  local branch
  branch=$(git -C "$1" rev-parse --abbrev-ref HEAD)
  [[ "$branch" == "$TARGET_BRANCH" ]] || die "checked-out branch is '$branch', expected '$TARGET_BRANCH'"
}

# Prints the skill names listed in a manifest, one per line, skipping (with a
# warning) any line that is not a plain name — the tombstone's comment lines
# included. The pre-fix script parses .managed-skills the same way, which is
# what makes the tombstone inert for it.
manifest_names() {
  local file="$1" entry label
  label=$(basename "$file")
  while IFS= read -r entry; do
    [[ -z "$entry" ]] && continue
    if [[ "$entry" == "." || "$entry" == ".." || ! "$entry" =~ ^[a-zA-Z0-9._-]+$ ]]; then
      echo "WARNING: skipping invalid entry in $label: $entry" >&2
      continue
    fi
    echo "$entry"
  done < "$file"
}

# Prints the source names whose manifests (other than this source's) list $2.
other_claimants() {
  local target="$1" name="$2" file names
  for file in "$target/$LEGACY_MANIFEST".*; do
    [[ -f "$file" ]] || continue
    [[ "$(basename "$file")" == "$MANIFEST" ]] && continue
    # Read the whole manifest first: under pipefail, `grep -q` closing the pipe
    # early could fail the pipeline on a match and hide the claimant.
    names=$(manifest_names "$file" 2>/dev/null)
    if grep -qxF -- "$name" <<< "$names"; then
      echo "${file##*/"$LEGACY_MANIFEST".}"
    fi
  done
}

# --- Discover skills ---

skill_dirs=()
for skill_md in "$SKILLS_SOURCE"/*/SKILL.md; do
  [[ -f "$skill_md" ]] || continue
  skill_dirs+=("$(dirname "$skill_md")")
done

[[ ${#skill_dirs[@]} -gt 0 ]] || die "no skills found under $SKILLS_SOURCE/*/SKILL.md"
echo "Found ${#skill_dirs[@]} skill(s): ${skill_dirs[*]}"

source_skill_names=()
for skill_dir in "${skill_dirs[@]}"; do
  source_skill_names+=("$(basename "$skill_dir")")
done

in_source_set() {
  local name
  for name in "${source_skill_names[@]}"; do
    [[ "$name" == "$1" ]] && return 0
  done
  return 1
}

# --- Copy skills into target, excluding *.go and dotfiles ---

copy_skills() {
  local target_dir="$1"
  for skill_dir in "${skill_dirs[@]}"; do
    local name
    name=$(basename "$skill_dir")
    rm -rf "${target_dir:?}/${name}"
    mkdir -p "$target_dir/$name"
    # Copy files, excluding *.go and dotfiles
    find "$skill_dir" -mindepth 1 \
      ! -name '*.go' \
      ! -name '.*' \
      ! -path '*/.*' \
      -type f \
      -exec bash -c '
        src="$1"; skill_dir="$2"; target_dir="$3"
        rel="${src#"$skill_dir"/}"
        mkdir -p "$(dirname "$target_dir/$rel")"
        cp "$src" "$target_dir/$rel"
      ' _ {} "$skill_dir" "$target_dir/$name" \;
  done
}

# --- DRY_RUN=local without a target: copy into tmpdir, diff against empty baseline ---

if [[ "$DRY_RUN" == "local" && -z "$SKILLS_TARGET" ]]; then
  tmpdir=$(mktemp -d)
  trap 'rm -rf "$tmpdir"' EXIT
  echo "DRY_RUN=local: copying skills into $tmpdir"
  copy_skills "$tmpdir/$SKILLS_SUBDIR"
  echo ""
  echo "=== Skills copied ==="
  find "$tmpdir" -type f | sort | while read -r f; do
    echo "  ${f#"$tmpdir/"}"
  done
  echo ""
  echo "=== Diff (against empty baseline) ==="
  # Initialize as empty git repo to get a clean diff
  git -C "$tmpdir" init -q
  git -C "$tmpdir" add -A
  git -C "$tmpdir" diff --cached --stat
  echo ""
  echo "DRY_RUN=local complete. No network operations performed."
  exit 0
fi

# --- Target: an existing checkout, or a fresh clone ---

if [[ -n "$SKILLS_TARGET" ]]; then
  [[ -e "$SKILLS_TARGET/.git" ]] || die "SKILLS_TARGET '$SKILLS_TARGET' is not a git checkout"
  target="$SKILLS_TARGET"
  echo "Syncing into existing checkout $target"
else
  [[ -n "${SKILLS_TOKEN:-}" ]] || die "SKILLS_TOKEN is required (set DRY_RUN=local for offline testing)"

  tmpdir=$(mktemp -d)
  trap 'rm -rf "$tmpdir"' EXIT

  echo "Cloning $TARGET_REPO into $tmpdir/skills..."
  git clone --depth 1 --branch "$TARGET_BRANCH" \
    "https://x-access-token:${SKILLS_TOKEN}@github.com/${TARGET_REPO}.git" \
    "$tmpdir/skills"

  target="$tmpdir/skills"
fi

# --- Safety checks ---

assert_remote_url "$target"
assert_branch "$target"

# --- Copy skills ---

echo "Copying skills into target..."
copy_skills "$target/$SKILLS_SUBDIR"

# --- Remove skills this source no longer ships ---
# Only names in this source's own manifest are candidates. No manifest means a
# first run since the per-source manifests arrived (or a brand-new publisher):
# remove nothing, and let the manifest written below claim the current set.

if [[ -f "$target/$MANIFEST" ]]; then
  previously_managed_names=()
  while IFS= read -r entry; do
    previously_managed_names+=("$entry")
  done < <(manifest_names "$target/$MANIFEST")

  for previously_managed in "${previously_managed_names[@]}"; do
    in_source_set "$previously_managed" && continue
    [[ -d "$target/$SKILLS_SUBDIR/$previously_managed" ]] || continue
    claimants=$(other_claimants "$target" "$previously_managed")
    if [[ -n "$claimants" ]]; then
      # Two publishers claiming one name is a collision to settle upstream, not
      # something a release should resolve by deletion.
      echo "WARNING: not removing skill '$previously_managed': also listed by $(echo "$claimants" | paste -sd ' ' -)" >&2
      continue
    fi
    echo "Removing stale skill: $previously_managed"
    rm -rf "${target:?}/$SKILLS_SUBDIR/$previously_managed"
  done
else
  echo "No $MANIFEST in target (first run for $SYNC_SOURCE): removing nothing."
fi

# --- Write this source's manifest and the legacy tombstone ---

printf '%s\n' "${source_skill_names[@]}" | sort > "$target/$MANIFEST"

cat > "$target/$LEGACY_MANIFEST" <<'EOF'
# Superseded by the per-source manifests (.managed-skills.<cli>), one per publishing CLI.
# Each CLI deletes only the skill directories listed in its own manifest.
# Kept so a CLI still running the pre-fix sync script deletes nothing: that script skips
# every line it cannot parse as a skill name and only deletes names it can.
EOF

# --- Commit ---

git -C "$target" add -A

if git -C "$target" diff --cached --quiet; then
  echo "No changes to commit. Skills are already up to date."
  exit 0
fi

echo ""
echo "=== Changes ==="
git -C "$target" diff --cached --stat
echo ""

if [[ "$DRY_RUN" == "remote" ]]; then
  echo "DRY_RUN=remote: skipping commit and push."
  echo ""
  echo "=== Full diff ==="
  git -C "$target" diff --cached
  exit 0
fi

git -C "$target" \
  -c user.name="${SYNC_SOURCE}[bot]" \
  -c user.email="${SYNC_SOURCE}[bot]@users.noreply.github.com" \
  commit -m "$(cat <<EOF
Sync skills from ${SYNC_SOURCE} ${RELEASE_TAG}

Source: basecamp/${SYNC_SOURCE}@${SOURCE_SHA}
EOF
)"

if [[ "$DRY_RUN" == "local" ]]; then
  echo "DRY_RUN=local: committed to $target, skipping push."
  exit 0
fi

# --- Push (with one retry on non-fast-forward) ---

push_target() {
  git -C "$target" push origin "$TARGET_BRANCH" 2>&1
}

if ! output=$(push_target); then
  if echo "$output" | grep -qi "non-fast-forward"; then
    echo "Push rejected (non-fast-forward). Pulling with rebase and retrying..."
    git -C "$target" pull --rebase origin "$TARGET_BRANCH"
    if ! retry_output=$(push_target); then
      echo "$retry_output" >&2
      die "Push failed after retry"
    fi
  else
    echo "$output" >&2
    die "Push failed"
  fi
fi

echo ""
echo "Skills synced to $TARGET_REPO ($TARGET_BRANCH) from $RELEASE_TAG"
