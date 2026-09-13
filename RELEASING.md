# Releasing fizzy-cli

## Quick Release

```bash
# Run preflight checks and tag
make release VERSION=v4.0.0

# Or dry-run first
make release VERSION=v4.0.0 DRY_RUN=1
```

Pushing the tag triggers the GitHub Actions release workflow, which:
1. Runs the full test suite
2. Builds binaries for all platforms (linux/darwin/windows/freebsd/openbsd x amd64/arm64)
3. Signs macOS binaries (Developer ID + notarization)
4. Signs checksums with a cosign keyless bundle (OIDC)
5. Generates SBOMs with Syft
6. Builds .deb and .rpm packages
7. For stable tags only, publishes the Homebrew cask to `basecamp/homebrew-tap`
8. For stable tags only, publishes the Scoop manifest to `basecamp/homebrew-tap`
9. For stable tags only, publishes to AUR (if `AUR_KEY` configured)
10. For stable tags only, mirrors `skills/` into [basecamp/skills](https://github.com/basecamp/skills)

## Versioning

Follow [semver](https://semver.org/). Use `v` prefix for tags: `v4.0.0`, `v4.0.0-beta1`, `v4.1.0-rc.1`.

Stable tags like `v4.0.0` publish to all normal distribution channels. Prerelease tags with a suffix like `-beta1`, `-beta.1`, or `-rc.1` are marked as GitHub prereleases and are not marked as the latest GitHub release.

## Beta / Prerelease Releases

Use a prerelease tag when technical testers need a build before the next stable version:

```bash
make release VERSION=v4.0.0-beta1
```

Prerelease behavior is intentionally conservative so existing package-manager users do not upgrade unless they explicitly opt in. Example behavior:

| Surface | Stable tag `v4.0.0` | Prerelease tag `v4.0.0-beta1` |
|---------|----------------------|--------------------------------|
| GitHub Releases | Published as a normal release and eligible to be GitHub's latest release. | Published as a GitHub prerelease and explicitly not marked latest. |
| Release assets | Binaries, archives, checksums, SBOMs, `.deb`, and `.rpm` artifacts are uploaded. | Same artifacts are uploaded for explicit tester download/install. |
| curl installer | Installs `v4.0.0` once GitHub marks it latest. | Does not install the prerelease via `releases/latest`; testers must download assets explicitly. |
| Homebrew | Updates the normal `basecamp/tap/fizzy` cask. `brew upgrade fizzy` can move users to `v4.0.0`. | Does not update the normal cask (`skip_upload: auto`). Existing `brew upgrade fizzy` users stay on the latest stable cask. |
| Scoop | Updates the normal `fizzy` manifest. `scoop update fizzy` can move users to `v4.0.0`. | Does not update the normal manifest (`skip_upload: auto`). Existing Scoop users stay on the latest stable manifest. |
| AUR | Updates the normal `fizzy-cli` package if `AUR_KEY` is configured. | Skips the AUR publish job. Existing AUR users stay on the latest stable package. |
| basecamp/skills | Mirrors `skills/fizzy` into the shared skills repo. | Skips the skills sync job. `npx skills add basecamp/skills` keeps serving the latest stable skill. |
| Go install | The git tag exists for users who explicitly request it. | The prerelease tag exists for users who explicitly request it; no package-manager manifest is updated. |

Technical testers can install prereleases explicitly from the GitHub release assets, for example by downloading the asset for their OS/architecture from `https://github.com/basecamp/fizzy-cli/releases/tag/v4.0.0-beta1`.

## CI Secrets

All release credentials live in the `release` environment (`Settings > Environments > release`), so they are only exposed to jobs that pass the environment's required-reviewer gate. There are no repository-level release secrets. `HOMEBREW_TAP_TOKEN` and the skills sync token do not exist as stored secrets — each is minted per-run from the `cli-release-bot` GitHub App credentials, scoped to the one repo that job pushes to (`homebrew-tap`, `skills`). The app must be installed on both repos with contents write access.

| Name | Type | Purpose |
|------|------|---------|
| `RELEASE_CLIENT_ID` | variable | GitHub App client ID for `cli-release-bot` |
| `RELEASE_APP_PRIVATE_KEY` | secret | GitHub App private key for tap and skills push |
| `AUR_KEY` | secret | ed25519 SSH private key for AUR (optional) |
| `MACOS_SIGN_P12` | secret | Base64-encoded Developer ID Application .p12 |
| `MACOS_SIGN_PASSWORD` | secret | Password for the .p12 certificate |
| `MACOS_NOTARY_KEY` | secret | Base64-encoded App Store Connect API key (.p8) |
| `MACOS_NOTARY_KEY_ID` | secret | App Store Connect API key ID (10 chars) |
| `MACOS_NOTARY_ISSUER_ID` | secret | App Store Connect issuer UUID |

Set a secret with `gh secret set <NAME> --env release -R basecamp/fizzy-cli`; the `RELEASE_CLIENT_ID` variable uses `gh variable set` instead. For multi-line secrets like SSH keys, pipe the file directly (`gh secret set AUR_KEY --env release -R basecamp/fizzy-cli < keyfile`) so newlines are preserved — pasting a flattened key produces an "invalid format" SSH failure at publish time.

## Distribution Channels

| Channel | Location | Updated by |
|---------|----------|------------|
| GitHub Releases | `basecamp/fizzy-cli/releases` | GoReleaser |
| Homebrew | `basecamp/homebrew-tap` Casks/fizzy.rb | GoReleaser (stable tags only) |
| Scoop | `basecamp/homebrew-tap` fizzy.json | GoReleaser (stable tags only) |
| AUR | `aur.archlinux.org/packages/fizzy-cli` | `publish-aur.sh` (stable tags only) |
| Skills | `basecamp/skills` skills/fizzy | `sync-skills.sh` (stable tags only) |
| Go install | `go install github.com/basecamp/fizzy-cli/cmd/fizzy@latest` | Go module proxy |
| curl installer | `scripts/install.sh` | Manual |

## Dry Run

```bash
# Full preflight without tagging
make release VERSION=v4.0.0 DRY_RUN=1

# GoReleaser snapshot (local build test — generate completions first)
go build -o fizzy-tmp ./cmd/fizzy
mkdir -p completions
./fizzy-tmp completion bash > completions/fizzy.bash
./fizzy-tmp completion zsh > completions/fizzy.zsh
./fizzy-tmp completion fish > completions/fizzy.fish
rm fizzy-tmp
goreleaser release --snapshot --clean
```

## Skills sync

Stable releases mirror `skills/` into [basecamp/skills](https://github.com/basecamp/skills),
which several CLIs share. `scripts/sync-skills.sh` owns only this CLI's skills there:
it records the names it published in `.managed-skills.fizzy-cli` at the target root and
removes a `skills/<name>` only when that manifest lists it, the release no longer ships
it, and no other CLI's `.managed-skills.*` claims it (a collision is warned about and
left alone). A target with no `.managed-skills.fizzy-cli` yet is a first run: nothing is
removed. The legacy shared `.managed-skills` is rewritten as a comment-only tombstone
so a CLI still on the pre-fix script — which deleted everything its own tree lacked —
deletes nothing (basecamp/skills#5). `e2e/sync_skills_test.sh` pins the contract; it
runs under `make test-scripts` (part of `make check`) and in CI.

The copy drops `*.go` and dotfiles, so `skills/embed.go` stays here and only
`skills/fizzy/**` is published. Preview what a release would publish, offline:

```bash
DRY_RUN=local RELEASE_TAG=v0.0.0 SOURCE_SHA=$(git rev-parse HEAD) scripts/sync-skills.sh
```

The job is `continue-on-error`, so a failed sync never fails a release that has
already shipped. If it fails, a `skills-sync`-labeled issue is filed; recover with
the `Sync skills` workflow (`workflow_dispatch`, stable tag, optional dry run). It
refuses anything but the latest stable release so it cannot roll the distribution
repo back, and it runs the sync script from the dispatching branch against the
tag's skills tree — so when the failure was a defect in `sync-skills.sh` itself,
merge the fix to master and dispatch; no new release needed.

## AUR Setup

1. Generate ed25519 SSH keypair: `ssh-keygen -t ed25519 -f aur_key`
2. Add public key to the AUR account that maintains `fizzy-cli`
3. Validate the private key parses and authenticates:
   `ssh-keygen -y -f aur_key > /dev/null && ssh -T -i aur_key aur@aur.archlinux.org`
   (expect "Interactive shell is disabled")
4. Store it in the `release` environment, preserving newlines:
   `gh secret set AUR_KEY --env release -R basecamp/fizzy-cli < aur_key`

## Tap Migration Ordering

When moving the Homebrew install path from another tap to `basecamp/tap` (learned during the v4.0.0 release, 2026-07-27):

1. **First** ship a stable release so GoReleaser publishes `Casks/fizzy.rb` to `basecamp/homebrew-tap`. The cask must exist before anything points at it.
2. **Then** land `tap_migrations.json` (mapping the old formula to `basecamp/tap/fizzy`) in the old tap and remove its formula. `brew update` surfaces the migration to existing users.
3. Do **not** transfer the old tap repo into the basecamp org — a repo named `homebrew-*` or matching `basecamp/fizzy-cli` would become a stray implicit tap. Archive it once users have migrated.
4. Update README install instructions last, once `brew install --cask basecamp/tap/fizzy` actually works.
