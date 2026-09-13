#!/usr/bin/env bash
#
# Ownership contract for scripts/sync-skills.sh against the shared
# basecamp/skills distribution repo (basecamp/skills#5).
#
# Several CLIs publish into one target. Each keeps its own manifest there
# (.managed-skills.<source>) and may delete only the skill directories that
# manifest lists — never a sibling's. The pre-fix script kept one shared
# .managed-skills, so every release pruned whatever the previous publisher had
# put there: hey-cli's v0.1.1 sync deleted skills/basecamp and
# skills/basecamp-doctor (08ef7ea), and basecamp-cli's next two releases
# deleted skills/hey (728a916, 42716d7).
#
# The fixture reproduces basecamp/skills as that history left it: the basecamp
# skills present and the legacy manifest listing the basecamp names. fizzy-cli
# has never published there, so its first sync is the first-run path. The
# script's default SYNC_SOURCE plays this repo; the sibling CLI is played by
# setting SYNC_SOURCE explicitly (the two sync_* helpers below are the only
# lines that differ between this file and its twins in the sibling repos).
# SKILLS_TARGET points the script at the fixture checkout and DRY_RUN=local
# applies and commits without pushing, so nothing here touches the network.
#
# This is the BATS file the sibling repos carry, run without BATS: this repo's
# toolchain has none, so a small harness at the bottom plays its part. Each
# test_* function runs in a subshell under `set -e` with an ERR trap, so the
# first failing command ends the test and names its line; `run` captures a
# command's merged output and status the way BATS' does.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SYNC="$REPO_ROOT/scripts/sync-skills.sh"

setup() {
  WORK="$(mktemp -d)"
  TARGET="$WORK/target"

  # Keep the operator's git identity, signing and hooks out of the fixture.
  export GIT_CONFIG_NOSYSTEM=1
  export GIT_CONFIG_GLOBAL=/dev/null
  export RELEASE_TAG=v4.1.0
  export SOURCE_SHA=0123456789abcdef0123456789abcdef01234567
  export DRY_RUN=local

  # basecamp/skills today. README.md and skills/orphan stand in for content no
  # publisher owns; both must survive every run, orphan in particular because it
  # is exactly what the dropped first-run fallback used to claim and delete.
  mkdir -p "$TARGET/skills/basecamp" "$TARGET/skills/basecamp-doctor" "$TARGET/skills/orphan"
  printf '# basecamp (stale)\n' > "$TARGET/skills/basecamp/SKILL.md"
  printf '# basecamp-doctor (stale)\n' > "$TARGET/skills/basecamp-doctor/SKILL.md"
  printf '# orphan\n' > "$TARGET/skills/orphan/SKILL.md"
  printf '# basecamp/skills\n' > "$TARGET/README.md"
  printf 'basecamp\nbasecamp-doctor\n' > "$TARGET/.managed-skills"
  git -C "$TARGET" init -q -b main
  git -C "$TARGET" remote add origin https://github.com/basecamp/skills.git
  git -C "$TARGET" -c user.name=t -c user.email=t@t add -A
  git -C "$TARGET" -c user.name=t -c user.email=t@t commit -q -m "Sync skills from basecamp-cli v0.11.0"

  # basecamp-cli's tree. One skill carries a nested file plus a *.go and a
  # dotfile that the copy filter must drop.
  BASECAMP="$WORK/basecamp/skills"
  mkdir -p "$BASECAMP/basecamp/reference" "$BASECAMP/basecamp-doctor"
  printf '# basecamp\n' > "$BASECAMP/basecamp/SKILL.md"
  printf 'reference\n' > "$BASECAMP/basecamp/reference/api.md"
  printf 'package skills\n' > "$BASECAMP/basecamp/embed.go"
  printf 'secret\n' > "$BASECAMP/basecamp/.hidden"
  printf '# basecamp-doctor\n' > "$BASECAMP/basecamp-doctor/SKILL.md"

  # fizzy-cli's tree.
  FIZZY="$WORK/fizzy/skills"
  mkdir -p "$FIZZY/fizzy"
  printf '# fizzy\n' > "$FIZZY/fizzy/SKILL.md"

  TOMBSTONE='# Superseded by the per-source manifests (.managed-skills.<cli>), one per publishing CLI.
# Each CLI deletes only the skill directories listed in its own manifest.
# Kept so a CLI still running the pre-fix sync script deletes nothing: that script skips
# every line it cannot parse as a skill name and only deletes names it can.'
}

teardown() {
  rm -rf "$WORK"
}

sync_basecamp() {
  run env SYNC_SOURCE=basecamp-cli SKILLS_SOURCE="$BASECAMP" SKILLS_TARGET="$TARGET" "$SYNC"
  [ "$status" -eq 0 ]
}

sync_fizzy() {
  run env SKILLS_SOURCE="$FIZZY" SKILLS_TARGET="$TARGET" "$SYNC"
  [ "$status" -eq 0 ]
}

# Both CLIs' skills present, each manifest listing exactly its own names, the
# legacy file a tombstone, and the unowned content untouched.
assert_shared_state() {
  [ -f "$TARGET/skills/basecamp/SKILL.md" ]
  [ -f "$TARGET/skills/basecamp-doctor/SKILL.md" ]
  [ -f "$TARGET/skills/fizzy/SKILL.md" ]
  [ "$(cat "$TARGET/.managed-skills.basecamp-cli")" = $'basecamp\nbasecamp-doctor' ]
  [ "$(cat "$TARGET/.managed-skills.fizzy-cli")" = "fizzy" ]
  [ "$(cat "$TARGET/.managed-skills")" = "$TOMBSTONE" ]
  assert_unowned_intact
}

assert_unowned_intact() {
  [ -f "$TARGET/README.md" ]
  [ -f "$TARGET/skills/orphan/SKILL.md" ]
}

# Every run leaves the fixture committed: the next run must see a clean tree,
# as the real clone does.
assert_clean_tree() {
  [ -z "$(git -C "$TARGET" status --porcelain)" ]
}

test_interleaved_releases_keep_both_clis_skills() {
  # basecamp-cli's first run: no .managed-skills.basecamp-cli yet, so nothing
  # is removed and the manifest claims the current set.
  sync_basecamp
  [[ "$output" == *"first run for basecamp-cli"* ]]
  [[ "$output" != *"Removing stale skill"* ]]
  [ "$(cat "$TARGET/skills/basecamp/SKILL.md")" = "# basecamp" ]
  [ -f "$TARGET/skills/basecamp/reference/api.md" ]
  [ ! -e "$TARGET/skills/basecamp/embed.go" ]
  [ ! -e "$TARGET/skills/basecamp/.hidden" ]
  [ "$(cat "$TARGET/.managed-skills.basecamp-cli")" = $'basecamp\nbasecamp-doctor' ]
  [ ! -e "$TARGET/.managed-skills.fizzy-cli" ]
  [ "$(cat "$TARGET/.managed-skills")" = "$TOMBSTONE" ]
  assert_unowned_intact
  assert_clean_tree

  sync_fizzy
  [[ "$output" == *"first run for fizzy-cli"* ]]
  assert_shared_state
  assert_clean_tree

  sync_basecamp
  [[ "$output" != *"Removing stale skill"* ]]
  assert_shared_state
  assert_clean_tree

  sync_fizzy
  [[ "$output" != *"Removing stale skill"* ]]
  assert_shared_state
  assert_clean_tree
}

test_a_rerun_with_nothing_changed_commits_nothing() {
  sync_basecamp
  sync_fizzy
  sync_basecamp
  [[ "$output" == *"No changes to commit"* ]]
  assert_shared_state
  assert_clean_tree
}

test_removes_only_a_skill_dropped_from_its_own_tree() {
  sync_basecamp
  sync_fizzy
  rm -rf "$BASECAMP/basecamp-doctor"

  sync_basecamp
  [[ "$output" == *"Removing stale skill: basecamp-doctor"* ]]
  [ ! -e "$TARGET/skills/basecamp-doctor" ]
  [ -f "$TARGET/skills/basecamp/SKILL.md" ]
  [ -f "$TARGET/skills/fizzy/SKILL.md" ]
  [ "$(cat "$TARGET/.managed-skills.basecamp-cli")" = "basecamp" ]
  [ "$(cat "$TARGET/.managed-skills.fizzy-cli")" = "fizzy" ]
  assert_unowned_intact
  assert_clean_tree
}

test_a_stale_legacy_manifest_from_a_pre_fix_sibling_deletes_nothing() {
  sync_basecamp
  sync_fizzy
  # The pre-fix script rewrites .managed-skills with its own names after every
  # run. The new script must read only its own manifest and put the tombstone
  # back.
  printf 'basecamp\nbasecamp-doctor\n' > "$TARGET/.managed-skills"
  git -C "$TARGET" -c user.name=t -c user.email=t@t commit -q -am "Sync skills from basecamp-cli v0.11.1 (pre-fix)"

  sync_fizzy
  [[ "$output" != *"Removing stale skill"* ]]
  assert_shared_state
  assert_clean_tree
}

test_a_name_another_manifest_claims_survives_with_a_warning() {
  sync_basecamp
  sync_fizzy
  # Collision: basecamp-cli's manifest lists fizzy, which its tree does not
  # ship and fizzy-cli's manifest also lists. Deleting it would take fizzy-cli's
  # skill; leave it and say so.
  printf 'basecamp\nbasecamp-doctor\nfizzy\n' > "$TARGET/.managed-skills.basecamp-cli"
  git -C "$TARGET" -c user.name=t -c user.email=t@t commit -q -am "collide"

  sync_basecamp
  [[ "$output" == *"WARNING: not removing skill 'fizzy': also listed by fizzy-cli"* ]]
  [[ "$output" != *"Removing stale skill"* ]]
  assert_shared_state
  assert_clean_tree
}

test_a_first_run_against_a_legacy_only_target_removes_nothing() {
  # fizzy-cli's first release with a sync: the target has never seen a
  # .managed-skills.fizzy-cli, and the legacy manifest lists only basecamp's
  # names. The pre-fix script would have claimed and deleted both basecamp
  # skills here, as hey-cli's did (08ef7ea).
  sync_fizzy
  [[ "$output" != *"Removing stale skill"* ]]
  [ -f "$TARGET/skills/basecamp/SKILL.md" ]
  [ -f "$TARGET/skills/basecamp-doctor/SKILL.md" ]
  [ -f "$TARGET/skills/fizzy/SKILL.md" ]
  [ "$(cat "$TARGET/.managed-skills.fizzy-cli")" = "fizzy" ]
  [ ! -e "$TARGET/.managed-skills.basecamp-cli" ]
  [ "$(cat "$TARGET/.managed-skills")" = "$TOMBSTONE" ]
  assert_unowned_intact
  assert_clean_tree
}

test_commits_as_source_bot_with_the_source_in_the_message() {
  sync_basecamp
  [ "$(git -C "$TARGET" log -1 --format=%an)" = "basecamp-cli[bot]" ]
  [ "$(git -C "$TARGET" log -1 --format=%ae)" = "basecamp-cli[bot]@users.noreply.github.com" ]
  [ "$(git -C "$TARGET" log -1 --format=%s)" = "Sync skills from basecamp-cli v4.1.0" ]
  [[ "$(git -C "$TARGET" log -1 --format=%b)" == *"basecamp/basecamp-cli@${SOURCE_SHA}"* ]]

  sync_fizzy
  [ "$(git -C "$TARGET" log -1 --format=%an)" = "fizzy-cli[bot]" ]
  [ "$(git -C "$TARGET" log -1 --format=%ae)" = "fizzy-cli[bot]@users.noreply.github.com" ]
  [ "$(git -C "$TARGET" log -1 --format=%s)" = "Sync skills from fizzy-cli v4.1.0" ]
  [[ "$(git -C "$TARGET" log -1 --format=%b)" == *"basecamp/fizzy-cli@${SOURCE_SHA}"* ]]
}

test_still_refuses_a_target_that_is_not_basecamp_skills_on_main() {
  git -C "$TARGET" remote set-url origin https://github.com/basecamp/other.git
  run env SKILLS_SOURCE="$FIZZY" SKILLS_TARGET="$TARGET" "$SYNC"
  [ "$status" -ne 0 ]
  [[ "$output" == *"does not point to github.com/basecamp/skills"* ]]

  git -C "$TARGET" remote set-url origin https://github.com/basecamp/skills.git
  git -C "$TARGET" checkout -q -b topic
  run env SKILLS_SOURCE="$FIZZY" SKILLS_TARGET="$TARGET" "$SYNC"
  [ "$status" -ne 0 ]
  [[ "$output" == *"checked-out branch is 'topic'"* ]]
}

test_dry_run_local_without_a_target_still_previews_offline() {
  cd "$WORK/basecamp"
  run "$SYNC"
  [ "$status" -eq 0 ]
  [[ "$output" == *"Found 2 skill(s):"* ]]
  [[ "$output" == *"skills/basecamp/SKILL.md"* ]]
  [[ "$output" == *"skills/basecamp-doctor/SKILL.md"* ]]
  [[ "$output" == *"skills/basecamp/reference/api.md"* ]]
  [[ "$output" != *"embed.go"* ]]
  [[ "$output" != *".hidden"* ]]
  [[ "$output" == *"No network operations performed"* ]]
}

test_skills_source_points_the_sync_at_another_checkout() {
  cd "$WORK/basecamp"
  run env SKILLS_SOURCE="$FIZZY" "$SYNC"
  [ "$status" -eq 0 ]
  [[ "$output" == *"Found 1 skill(s): $FIZZY/fizzy"* ]]
  [[ "$output" != *"basecamp"* ]]
}

test_dies_when_the_source_tree_has_no_skills() {
  run env SKILLS_SOURCE="$WORK/empty" "$SYNC"
  [ "$status" -ne 0 ]
  [[ "$output" == *"no skills found under $WORK/empty/*/SKILL.md"* ]]
}

# --- Harness ---

# Captures a command's merged stdout+stderr in $output and its exit status in
# $status without tripping `set -e`, as BATS' run does. The ERR trap is
# suspended too: it fires on any nonzero status, `set +e` or not, and a
# refusal under test is not a failure of the test.
run() {
  local saved_trap
  saved_trap=$(trap -p ERR)
  set +e
  trap - ERR
  output="$("$@" 2>&1)"
  status=$?
  eval "$saved_trap"
  set -e
}

# ERR trap for a test's subshell: names the failing line and command, and
# shows the last captured output so a failed assertion on it is diagnosable.
fail_report() {
  # BASH_LINENO[0] is the line that tripped the trap; LINENO here would be
  # this function's own.
  echo "    failed at line ${BASH_LINENO[0]}: $BASH_COMMAND"
  if [[ -n "${output:-}" ]]; then
    printf '    --- last output ---\n%s\n' "$output" | sed 's/^/    /'
  fi
}

main() {
  local passed=0 failed=0 test result
  for test in $(declare -F | awk '$3 ~ /^test_/ { print $3 }'); do
    setup
    # A subshell inside an `if` would have `set -e` ignored, so run it plainly
    # and read the status afterwards.
    set +e
    (
      set -eE
      trap fail_report ERR
      "$test"
    )
    result=$?
    set -e
    teardown
    if [[ $result -eq 0 ]]; then
      echo "ok   ${test#test_}"
      passed=$((passed + 1))
    else
      echo "FAIL ${test#test_}"
      failed=$((failed + 1))
    fi
  done
  echo
  echo "$passed passed, $failed failed"
  [[ $failed -eq 0 ]]
}

main "$@"
