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
4. Signs checksums with cosign (keyless, OIDC)
5. Generates SBOMs with Syft
6. Builds .deb and .rpm packages
7. For stable tags only, publishes the Homebrew cask to `basecamp/homebrew-tap`
8. For stable tags only, publishes the Scoop manifest to `basecamp/homebrew-tap`
9. For stable tags only, publishes to AUR (if `AUR_KEY` configured)

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
| Go install | The git tag exists for users who explicitly request it. | The prerelease tag exists for users who explicitly request it; no package-manager manifest is updated. |

Technical testers can install prereleases explicitly from the GitHub release assets, for example by downloading the asset for their OS/architecture from `https://github.com/basecamp/fizzy-cli/releases/tag/v4.0.0-beta1`.

## CI Secrets

### Repository level (`Settings > Secrets and variables > Actions`)

| Name | Type | Purpose |
|------|------|---------|
| `RELEASE_CLIENT_ID` | variable | GitHub App client ID for `cli-release-bot` |
| `RELEASE_APP_PRIVATE_KEY` | secret | GitHub App private key for tap push |
| `AUR_KEY` | secret | ed25519 SSH private key for AUR (optional) |

### Environment level (`release` environment — `Settings > Environments`)

| Secret | Purpose |
|--------|---------|
| `MACOS_SIGN_P12` | Base64-encoded Developer ID Application .p12 |
| `MACOS_SIGN_PASSWORD` | Password for the .p12 certificate |
| `MACOS_NOTARY_KEY` | Base64-encoded App Store Connect API key (.p8) |
| `MACOS_NOTARY_KEY_ID` | App Store Connect API key ID (10 chars) |
| `MACOS_NOTARY_ISSUER_ID` | App Store Connect issuer UUID |

## Distribution Channels

| Channel | Location | Updated by |
|---------|----------|------------|
| GitHub Releases | `basecamp/fizzy-cli/releases` | GoReleaser |
| Homebrew | `basecamp/homebrew-tap` Casks/fizzy.rb | GoReleaser (stable tags only) |
| Scoop | `basecamp/homebrew-tap` fizzy.json | GoReleaser (stable tags only) |
| AUR | `aur.archlinux.org/packages/fizzy-cli` | `publish-aur.sh` (stable tags only) |
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

## AUR Setup

1. Generate ed25519 SSH keypair: `ssh-keygen -t ed25519 -f aur_key`
2. Add public key to your AUR account profile
3. Add private key as `AUR_KEY` secret on the fizzy-cli repo
