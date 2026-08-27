#!/bin/sh
# Refresh the in-repo copy of go-studs (third_party/go-studs) from a local
# checkout of the author's MIT-licensed terminal UI kit, then re-apply the
# approved local patches (quality pass plan §0.6, ADR-04):
#
#   1. refuse unless the checkout's HEAD is the commit pinned in
#      third_party/go-studs/LOCAL_CHANGES.md (or --allow-drift is given);
#   2. copy the packages Watchpost imports into a temp dir — no tests, no
#      docs, nothing listed in sync-exclude.txt — and rewrite their import
#      paths to this module;
#   3. `git apply --check` then apply every patch LOCAL_CHANGES.md lists,
#      in name order, from patches/ (tracked) or .local/ (machine-local,
#      untracked), stopping on the first missing or failing one and naming it;
#   4. swap the package directories in atomically (the old copy is kept
#      beside the tree until the swap completes, then removed);
#   5. regenerate THIRD_PARTY_LICENSES.md and re-run `a2dh p10 check` on
#      the kit (each skippable for a self-test: --no-licences, --no-p10).
#
#   scripts/sync-go-studs.sh /path/to/go-studs [--allow-drift] [--no-licences] [--no-p10]
#   scripts/sync-go-studs.sh --self-test        (a temp upstream + temp project; touches nothing here)
set -eu

usage() { echo "usage: scripts/sync-go-studs.sh <go-studs checkout> [--allow-drift] [--no-licences] [--no-p10] | --self-test" >&2; exit 2; }
fail() { echo "sync-go-studs: $*" >&2; exit 1; }

src=""; drift=0; licences=1; p10=1; selftest=0
for arg in "$@"; do
  case "$arg" in
    --allow-drift) drift=1 ;;
    --no-licences) licences=0 ;;
    --no-p10) p10=1; p10=0 ;;
    --self-test) selftest=1 ;;
    --*) usage ;;
    *) src="$arg" ;;
  esac
done

project="${SYNC_PROJECT:-$(cd "$(dirname "$0")/.." && pwd)}"
dst="$project/third_party/go-studs"
pkgs="rendering components theme tokens"

sync() {
  [ -n "$src" ] && [ -f "$src/go.mod" ] || usage
  [ -f "$dst/LOCAL_CHANGES.md" ] || fail "$dst/LOCAL_CHANGES.md is missing (it pins the upstream commit)"
  pinned=$(sed -n 's/^Upstream commit: `\([0-9a-f]*\)`.*/\1/p' "$dst/LOCAL_CHANGES.md" | head -1)
  [ -n "$pinned" ] || fail "no 'Upstream commit: \`<sha>\`' line in LOCAL_CHANGES.md"
  head=$(git -C "$src" rev-parse --short "$pinned" 2>/dev/null || echo unknown)
  actual=$(git -C "$src" rev-parse --short HEAD)
  if [ "$actual" != "$head" ]; then
    [ "$drift" -eq 1 ] || fail "checkout is at $actual, LOCAL_CHANGES.md pins $pinned — move the pin (and review the patches) or pass --allow-drift"
    echo "sync-go-studs: drift allowed: $pinned -> $actual"
  fi
  upstream=$(sed -n 's/^module //p' "$src/go.mod")
  here=$(sed -n 's/^module //p' "$project/go.mod")/third_party/go-studs
  tmp=$(mktemp -d "${TMPDIR:-/tmp}/go-studs-sync.XXXXXX")
  trap 'rm -rf "$tmp"' EXIT
  for pkg in $pkgs; do
    [ -d "$src/$pkg" ] || fail "upstream has no $pkg package"
    mkdir -p "$tmp/$pkg"
    for f in "$src/$pkg"/*.go; do
      case "$f" in *_test.go) continue ;; esac
      name=$(basename "$f")
      if [ -f "$dst/sync-exclude.txt" ] && grep -v '^#' "$dst/sync-exclude.txt" | grep -qx "$pkg/$name"; then
        continue
      fi
      cp "$f" "$tmp/$pkg/$name"
    done
  done
  cp "$src/LICENSE" "$tmp/LICENSE"
  find "$tmp" -name '*.go' -exec sed -i.bak "s#$upstream#$here#g" {} \; -exec rm {}.bak \;
  # 3. the patch stack: every patch LOCAL_CHANGES.md lists, in name order,
  #    from patches/ (tracked) or .local/ (machine-local, untracked — the
  #    public-tree wording scrub); a listed patch that is missing, or one
  #    that does not apply, stops the sync by name with nothing changed.
  for name in $(sed -n 's/^| `\([0-9][0-9][0-9]-[^`]*\.patch\)`.*/\1/p' "$dst/LOCAL_CHANGES.md" | sort); do
    patch="$dst/patches/$name"
    [ -f "$patch" ] || patch="$dst/.local/$name"
    [ -f "$patch" ] || fail "patch $name is listed in LOCAL_CHANGES.md but present in neither patches/ nor .local/ — nothing was changed"
    git -C "$tmp" apply --check "$patch" 2>/dev/null || fail "patch $name does not apply to upstream $actual — nothing was changed"
    git -C "$tmp" apply "$patch" || fail "patch $name failed to apply — nothing was changed"
    echo "sync-go-studs: applied $name"
  done
  # 4. atomic swap
  old=$(mktemp -d "$project/third_party/.go-studs-old.XXXXXX")
  for pkg in $pkgs; do
    [ -d "$dst/$pkg" ] && mv "$dst/$pkg" "$old/$pkg"
    mv "$tmp/$pkg" "$dst/$pkg"
  done
  mv "$tmp/LICENSE" "$dst/LICENSE"
  rm -rf "$old"
  cat > "$dst/NOTICE.md" <<NOTE
# go-studs (in-repo copy)

go-studs is the author's MIT-licensed terminal UI kit (LICENSE alongside; upstream commit
\`$actual\`, pinned in LOCAL_CHANGES.md). Only the packages Watchpost imports are carried —
rendering, components, theme, tokens — without their tests or docs, as part of this module under
\`third_party/go-studs\`. Local changes are the patches under \`patches/\`, listed in
LOCAL_CHANGES.md and re-applied by \`scripts/sync-go-studs.sh <path to the upstream checkout>\`;
do not edit the packages directly — change upstream, or add a patch with its row.
NOTE
  # 5. licences and P10 on the kit
  if [ "$licences" -eq 1 ]; then
    "$project/scripts/third-party-licenses.sh" || fail "THIRD_PARTY_LICENSES.md regeneration failed"
  fi
  if [ "$p10" -eq 1 ]; then
    a2dh="${A2DH:-a2dh}"
    command -v "$a2dh" >/dev/null 2>&1 || fail "a2dh CLI not found (set A2DH) — the kit's P10 check did not run; pass --no-p10 only for a self-test"
    (cd "$project" && "$a2dh" p10 check --json >/dev/null) || fail "P10 check failed after the sync — see dist/p10.json"
  fi
  echo "sync-go-studs: copied $pkgs @ $actual"
}

self_test() {
  work=$(mktemp -d "${TMPDIR:-/tmp}/go-studs-selftest.XXXXXX")
  trap 'rm -rf "$work"' EXIT
  up="$work/upstream"; proj="$work/project"
  mkdir -p "$up/rendering" "$up/components" "$up/theme" "$up/tokens" "$proj/scripts" "$proj/third_party/go-studs/patches"
  printf 'module example.com/kit\n\ngo 1.25\n' > "$up/go.mod"
  printf 'package rendering\n\nfunc Width(s string) int { return len(s) }\n' > "$up/rendering/width.go"
  printf 'package rendering\n\nfunc helper() {}\n' > "$up/rendering/width_test.go"
  printf 'package rendering\n\nfunc Skipped() {}\n' > "$up/rendering/skipped.go"
  printf 'package components\n\nimport "example.com/kit/rendering"\n\nfunc W(s string) int { return rendering.Width(s) }\n' > "$up/components/table.go"
  printf 'package theme\n' > "$up/theme/theme.go"
  printf 'package tokens\n' > "$up/tokens/tokens.go"
  printf 'MIT\n' > "$up/LICENSE"
  git -C "$up" init -q && git -C "$up" add -A && git -C "$up" -c user.name=t -c user.email=t@t -c commit.gpgsign=false commit -q -m upstream
  rev=$(git -C "$up" rev-parse --short HEAD)
  printf 'module example.com/app\n\ngo 1.25\n' > "$proj/go.mod"
  changes="$proj/third_party/go-studs/LOCAL_CHANGES.md"
  listing() { printf '# local\n\nUpstream commit: `%s`\n\n| Patch | Why |\n|---|---|\n' "$1"; for p in "$@"; do [ "$p" = "$1" ] && continue; printf '| `%s` | t |\n' "$p"; done; }
  listing "$rev" 000-local.patch 001-width.patch > "$changes"
  printf 'rendering/skipped.go\n' > "$proj/third_party/go-studs/sync-exclude.txt"
  cp "$0" "$proj/scripts/sync-go-studs.sh"
  mkdir -p "$proj/third_party/go-studs/.local"
  # a machine-local patch (comment wording), a tracked patch that applies
  # (against the rewritten copy), and later one that does not
  cat > "$proj/third_party/go-studs/.local/000-local.patch" <<'P'
--- a/theme/theme.go
+++ b/theme/theme.go
@@ -1 +1 @@
-package theme
+package theme // scrubbed
P
  cat > "$proj/third_party/go-studs/patches/001-width.patch" <<'P'
--- a/rendering/width.go
+++ b/rendering/width.go
@@ -1,3 +1,3 @@
 package rendering
 
-func Width(s string) int { return len(s) }
+func Width(s string) int { return len([]rune(s)) }
P
  run() { (cd "$proj" && SYNC_PROJECT="$proj" sh scripts/sync-go-studs.sh "$@" --no-licences --no-p10 2>&1); }
  # 1. the pin is enforced
  listing 0000000 000-local.patch 001-width.patch > "$changes"
  if run "$up" >/dev/null; then fail "self-test: a drifted checkout must be refused"; fi
  run "$up" --allow-drift >/dev/null || fail "self-test: --allow-drift must let a drifted checkout through"
  listing "$rev" 000-local.patch 001-width.patch > "$changes"
  # 2. copy, exclusion, rewrite, patch
  run "$up" >/dev/null || fail "self-test: a clean sync must succeed"
  k="$proj/third_party/go-studs"
  [ -f "$k/rendering/width.go" ] || fail "self-test: package not copied"
  [ ! -f "$k/rendering/width_test.go" ] || fail "self-test: tests must not be copied"
  [ ! -f "$k/rendering/skipped.go" ] || fail "self-test: sync-exclude.txt not honoured"
  grep -q 'example.com/app/third_party/go-studs/rendering' "$k/components/table.go" || fail "self-test: import path not rewritten"
  grep -q 'len(\[\]rune(s))' "$k/rendering/width.go" || fail "self-test: patch 001 not applied"
  grep -q 'scrubbed' "$k/theme/theme.go" || fail "self-test: the machine-local patch 000 not applied"
  grep -q "$rev" "$k/NOTICE.md" || fail "self-test: NOTICE.md not updated"
  # 3. a listed patch that is missing stops the sync by name
  listing "$rev" 000-local.patch 001-width.patch 003-missing.patch > "$changes"
  out=$(run "$up" || true)
  echo "$out" | grep -q '003-missing.patch' || fail "self-test: a listed-but-missing patch must be named, got: $out"
  # 4. a failing patch stops the sync, names itself, and leaves the tree as it was
  listing "$rev" 000-local.patch 001-width.patch 002-bad.patch > "$changes"
  cat > "$proj/third_party/go-studs/patches/002-bad.patch" <<'P'
--- a/rendering/width.go
+++ b/rendering/width.go
@@ -1,3 +1,3 @@
 package rendering
 
-func Width(s string) int { return 42 }
+func Width(s string) int { return 43 }
P
  before=$(cat "$k/rendering/width.go")
  out=$(run "$up" || true)
  echo "$out" | grep -q '002-bad.patch' || fail "self-test: a failing patch must be named, got: $out"
  [ "$(cat "$k/rendering/width.go")" = "$before" ] || fail "self-test: a failed sync must leave the kit untouched"
  echo "sync-go-studs self-test: control fired (OK)"
}

if [ "$selftest" -eq 1 ]; then self_test; else sync; fi
