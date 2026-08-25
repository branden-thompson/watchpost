#!/bin/sh
# Refresh the in-repo copy of go-studs (third_party/go-studs) from a local
# checkout of the author's MIT-licensed terminal UI kit. Carrying the
# packages Watchpost uses inside this module keeps `go build` and
# `go install` working for anyone. Copies only the packages Watchpost
# imports (and what they import), without tests, docs or coverage files,
# rewrites their import paths to this module, and records the upstream
# commit in NOTICE.md.
#
#   scripts/sync-go-studs.sh /path/to/go-studs     (or set GO_STUDS_SRC)
set -eu
src="${1:-${GO_STUDS_SRC:-}}"
[ -n "$src" ] && [ -f "$src/go.mod" ] || { echo "sync-go-studs: pass the go-studs checkout path (has go.mod)" >&2; exit 1; }
cd "$(dirname "$0")/.."
dst=third_party/go-studs
upstream=$(sed -n 's/^module //p' "$src/go.mod")
here=$(sed -n 's/^module //p' go.mod)/third_party/go-studs
rm -rf "$dst"
mkdir -p "$dst"
for pkg in rendering components theme tokens; do
  mkdir -p "$dst/$pkg"
  find "$src/$pkg" -maxdepth 1 -name '*.go' ! -name '*_test.go' -exec cp {} "$dst/$pkg/" \;
done
cp "$src/LICENSE" "$dst/LICENSE"
# The copied packages import each other under this module's path.
find "$dst" -name '*.go' -exec sed -i.bak "s#$upstream#$here#g" {} \; -exec rm {}.bak \;
rev=$(git -C "$src" rev-parse --short HEAD 2>/dev/null || echo unknown)
cat > "$dst/NOTICE.md" <<NOTE
# go-studs (in-repo copy)

go-studs is the author's MIT-licensed terminal UI kit (LICENSE alongside; upstream commit
\`$rev\`). Only the packages Watchpost imports are carried — rendering, components, theme,
tokens — without their tests or docs, as part of this module under \`third_party/go-studs\`.
Do not edit here: change upstream and rerun \`scripts/sync-go-studs.sh <path to the upstream
checkout>\`, which copies the packages and rewrites their import paths to this module.
NOTE
echo "sync-go-studs: copied rendering components theme tokens @ $rev"
