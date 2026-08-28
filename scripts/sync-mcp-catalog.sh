#!/bin/sh
# Sync the Fizzy MCP catalog from its sibling in fizzy-mcp-server.
#
# The hand-written catalog (domain specs, operations, rendering) is
# deliberately duplicated between fizzy-mcp-server (the hosted/HTTP story)
# and this CLI (the local stdio story), per the basecamp/mcp toolkit's
# two-instance-by-duplication convention: machinery proven in both moves
# to the toolkit, and until then each repo carries a full copy. This
# script copies the sibling's catalog package and snapshot verbatim and
# records provenance, so drift shows up as a reviewed diff here rather
# than as silently divergent tool surfaces.
#
# fizzy-mcp-server is a private repo, so this script is for maintainers
# with a checkout; the vendored copy keeps CI and outside builds hermetic.
#
# Usage: scripts/sync-mcp-catalog.sh [path-to-fizzy-mcp-server-checkout]
set -eu

sibling="${1:-../fizzy-mcp-server}"
src="$sibling/internal/catalog"
dest="$(dirname "$0")/../internal/mcpserver/catalog"

if [ ! -f "$src/catalog.go" ]; then
	echo "error: $src/catalog.go not found (pass a fizzy-mcp-server checkout)" >&2
	exit 1
fi

commit=$(git -C "$sibling" rev-parse HEAD)
if ! git -C "$sibling" diff --quiet HEAD -- internal/catalog 2>/dev/null; then
	commit="$commit-dirty"
fi

cp "$src/catalog.go" "$src/domains.go" "$dest/"
cp "$src/testdata/catalog_snapshot.txt" "$dest/testdata/"

cat > "$dest/PROVENANCE.json" <<JSON
{
  "source": "github.com/basecamp/fizzy-mcp-server",
  "commit": "$commit",
  "path": "internal/catalog",
  "files": ["catalog.go", "domains.go", "testdata/catalog_snapshot.txt"],
  "synced_by": "scripts/sync-mcp-catalog.sh"
}
JSON

echo "Synced catalog from fizzy-mcp-server@$commit"
