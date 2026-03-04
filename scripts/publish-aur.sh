#!/usr/bin/env bash
set -euo pipefail

# Publish fizzy-cli to AUR
# Requires: AUR_KEY environment variable

if [ -z "${GITHUB_REF_NAME:-}" ]; then
  echo "ERROR: GITHUB_REF_NAME is not set (must run from GitHub Actions release workflow)"
  exit 1
fi
VERSION="${GITHUB_REF_NAME#v}"
REPO="basecamp/fizzy-cli"

echo "Publishing fizzy-cli $VERSION to AUR..."

# Get source tarball checksum
SOURCE_URL="https://github.com/$REPO/archive/v${VERSION}.tar.gz"
curl -fsSL "$SOURCE_URL" -o source.tar.gz
SHA256=$(sha256sum source.tar.gz | cut -d' ' -f1)
rm source.tar.gz

# Generate PKGBUILD
cat > PKGBUILD << EOF
# Maintainer: 37signals <support@37signals.com>
pkgname=fizzy-cli
pkgver=$VERSION
pkgrel=1
pkgdesc="CLI for managing Fizzy boards, cards, and tasks"
arch=('x86_64' 'aarch64')
url="https://github.com/$REPO"
license=('MIT')
depends=('glibc')
makedepends=('go')
provides=('fizzy')
conflicts=('fizzy' 'fizzy-bin')
source=("\$pkgname-\$pkgver.tar.gz::https://github.com/$REPO/archive/v\$pkgver.tar.gz")
sha256sums=('$SHA256')
options=('!debug')

build() {
    cd "\$pkgname-\$pkgver"
    export CGO_CPPFLAGS="\${CPPFLAGS}"
    export CGO_CFLAGS="\${CFLAGS}"
    export CGO_CXXFLAGS="\${CXXFLAGS}"
    export CGO_LDFLAGS="\${LDFLAGS}"
    export GOFLAGS="-buildmode=pie -trimpath -mod=readonly -modcacherw"
    go build -ldflags "-s -w -X main.version=\${pkgver}" -o fizzy ./cmd/fizzy

    # Generate completions
    ./fizzy completion bash > fizzy.bash
    ./fizzy completion zsh > fizzy.zsh
    ./fizzy completion fish > fizzy.fish
}

package() {
    cd "\$pkgname-\$pkgver"
    install -Dm755 fizzy "\$pkgdir/usr/bin/fizzy"
    install -Dm644 MIT-LICENSE "\$pkgdir/usr/share/licenses/\$pkgname/MIT-LICENSE"
    install -Dm644 fizzy.bash "\$pkgdir/usr/share/bash-completion/completions/fizzy"
    install -Dm644 fizzy.zsh "\$pkgdir/usr/share/zsh/site-functions/_fizzy"
    install -Dm644 fizzy.fish "\$pkgdir/usr/share/fish/vendor_completions.d/fizzy.fish"
}
EOF

# Generate .SRCINFO
cat > .SRCINFO << EOF
pkgbase = fizzy-cli
	pkgdesc = CLI for managing Fizzy boards, cards, and tasks
	pkgver = $VERSION
	pkgrel = 1
	url = https://github.com/$REPO
	arch = x86_64
	arch = aarch64
	license = MIT
	makedepends = go
	depends = glibc
	provides = fizzy
	conflicts = fizzy
	conflicts = fizzy-bin
	options = !debug
	source = fizzy-cli-$VERSION.tar.gz::https://github.com/$REPO/archive/v$VERSION.tar.gz
	sha256sums = $SHA256

pkgname = fizzy-cli
EOF

# Clone AUR repo and push
mkdir -p ~/.ssh
echo "$AUR_KEY" > ~/.ssh/aur
chmod 600 ~/.ssh/aur
cat >> ~/.ssh/config << SSHEOF
Host aur.archlinux.org
    IdentityFile ~/.ssh/aur
    User aur
    StrictHostKeyChecking accept-new
SSHEOF

git clone ssh://aur@aur.archlinux.org/fizzy-cli.git aur-repo
cp PKGBUILD .SRCINFO aur-repo/
cd aur-repo
git config user.name "cli-release-bot"
git config user.email "cli-release-bot@users.noreply.github.com"
git add PKGBUILD .SRCINFO
if git diff --cached --quiet; then
  echo "AUR package already up to date for $VERSION"
else
  git commit -m "Update to $VERSION"
  git push
fi

echo "Published fizzy-cli $VERSION to AUR"
