#!/usr/bin/env sh
# Import-direction gate (architecture.md §1): no file under modes/ may import
# github.com/branden-thompson/watchpost/domains/... . Discovers consumers by walk,
# never a hardcoded list (calibration: Discover Consumers, Don't Enumerate Them).
# And go-studs stays behind platform/render (0.19.5 D-10): no other package's
# non-test file imports it, so the table kit can be swapped in one place.
#
# AND THE LAYERS 0.19.6 SPLITS ACROSS (TG-1, TG-14, DQ-6, D-15, A-24). Today the
# rule `modes` never imports `app` is held only by Go's ban on import cycles,
# and a split removes that: a package below app is a new package a mode could
# import. So the direction is a rule here: modes, platform and domains never
# import app; platform never imports modes; the deck and the map data never
# import each other; and the two test-support packages are imported by test
# files alone, so nothing in them reaches the binary.
set -eu
MOD="github.com/branden-thompson/watchpost"
check_tree() {
  root="$1"
  [ -d "$root" ] || return 0
  violations=$(grep -rn --include='*.go' "\"$MOD/domains/" "$root" || true)
  if [ -n "$violations" ]; then
    echo "lint-imports: FORBIDDEN import of domains/* from $root:"
    echo "$violations"
    return 1
  fi
}
# check_studs refuses a go-studs import from a non-test file outside
# platform/render, under the tree at $1. Tests may import go-studs (its colour
# switch, for one); go-studs itself, in third_party, and the agents' worktrees
# under .claude are not ours to judge.
check_studs() {
  root="$1"
  violations=$(cd "$root" && grep -rln --include='*.go' "\"$MOD/third_party/go-studs" . 2>/dev/null |
    sed 's|^\./||' | grep -Ev '^(third_party|platform/render|\.claude|dist)/|_test\.go$' || true)
  if [ -n "$violations" ]; then
    echo "lint-imports: FORBIDDEN import of go-studs outside platform/render (D-10):"
    echo "$violations"
    echo "  add a render function that does what this file needs, and call it; render is the one package that imports go-studs"
    return 1
  fi
}

# check_none refuses any Go file under $1 importing a path that starts with
# $MOD/$2, printing $3 as the reason; with $4 = nontest, test files may import it.
check_none() {
  root="$1" prefix="$2" why="$3" scope="${4:-all}"
  [ -d "$root" ] || return 0
  hits=$(grep -rln --include='*.go' --exclude-dir=.git --exclude-dir=.claude --exclude-dir=dist --exclude-dir=third_party -E "\"$MOD/$prefix(/|\")" "$root" 2>/dev/null || true)
  if [ "$scope" = nontest ]; then
    hits=$(printf '%s\n' "$hits" | grep -v '_test\.go$' || true)
  fi
  hits=$(printf '%s\n' "$hits" | grep -v '^$' || true)
  if [ -n "$hits" ]; then
    echo "lint-imports: FORBIDDEN import of $prefix from $root ($why):"
    echo "$hits"
    return 1
  fi
}

# check_layers is every 0.19.6 direction rule, under the tree at $1.
check_layers() {
  r="$1"
  for from in modes platform domains; do
    check_none "$r/$from" app "nothing below app imports it; the composition root is the top" || return 1
  done
  check_none "$r/platform" modes "platform is a leaf of the modes" || return 1
  check_none "$r/app/internal/deck" app/internal/mapdata "the deck and the map data never import each other" || return 1
  check_none "$r/app/internal/mapdata" app/internal/deck "the deck and the map data never import each other" || return 1
  check_none "$r" modes/tty/internal/ttytest "test support, imported by test files alone" nontest || return 1
  check_none "$r" app/internal/apptest "test support, imported by test files alone" nontest || return 1
}

# AN ARGUMENT THIS SCRIPT DOES NOT KNOW IS A FAILURE, NOT A NO-OP (B6).
#
# gate-controls exists to prove the other gates still fire, and it invokes them
# by flag. A script that ignored an unrecognised argument would run its NORMAL
# path to a green exit, so a renamed or mistyped flag in the Makefile would leave
# the controls gate passing having run no control at all — the one gate whose
# whole job is to notice that, unable to notice it about itself.
case "${1:-}" in
  ""|--self-test) ;;
  *) echo "lint-imports: unknown argument '$1'" >&2; exit 2 ;;
esac

if [ "${1:-}" = "--self-test" ]; then
  tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT
  mkdir -p "$tmp/modes/report" "$tmp/platform/lineup"
  printf 'package report\nimport _ "%s/domains/weather"\n' "$MOD" > "$tmp/modes/report/bad.go"
  if check_tree "$tmp/modes" >/dev/null 2>&1; then
    echo "lint-imports SELF-TEST FAILED: known-bad modes/ fixture not detected"; exit 1
  fi
  printf 'package lineup\nimport _ "%s/domains/radio/cast"\n' "$MOD" > "$tmp/platform/lineup/bad.go"
  if check_tree "$tmp/platform" >/dev/null 2>&1; then
    echo "lint-imports SELF-TEST FAILED: known-bad platform/ fixture not detected"; exit 1
  fi
  # AND GO-STUDS: refused from a mode's file, allowed in render and in a test.
  mkdir -p "$tmp/studs/modes/tty" "$tmp/studs/platform/render"
  printf 'package render\nimport _ "%s/third_party/go-studs/components"\n' "$MOD" > "$tmp/studs/platform/render/ok.go"
  printf 'package tty\nimport _ "%s/third_party/go-studs/rendering"\n' "$MOD" > "$tmp/studs/modes/tty/ok_test.go"
  if ! check_studs "$tmp/studs" >/dev/null 2>&1; then
    echo "lint-imports SELF-TEST FAILED: go-studs in render or in a test was refused"; exit 1
  fi
  printf 'package tty\nimport _ "%s/third_party/go-studs/components"\n' "$MOD" > "$tmp/studs/modes/tty/bad.go"
  if check_studs "$tmp/studs" >/dev/null 2>&1; then
    echo "lint-imports SELF-TEST FAILED: go-studs in a mode's file was not detected"; exit 1
  fi
  # AND THE LAYERS: each 0.19.6 rule refuses its planted file, and a test
  # file's import of the test support passes.
  n=4
  for planted in "modes/tty/bad.go:app" "platform/x/bad.go:app/internal/deck" "domains/x/bad.go:app" \
    "platform/x/bad2.go:modes/tty" "app/internal/deck/bad.go:app/internal/mapdata" \
    "app/internal/mapdata/bad.go:app/internal/deck" "modes/tty/console/bad.go:modes/tty/internal/ttytest" \
    "app/bad.go:app/internal/apptest"; do
    lay="$tmp/layers$n"
    file=${planted%%:*} imp=${planted#*:}
    mkdir -p "$lay/$(dirname "$file")"
    printf 'package x\nimport _ "%s/%s"\n' "$MOD" "$imp" > "$lay/$file"
    if check_layers "$lay" >/dev/null 2>&1; then
      echo "lint-imports SELF-TEST FAILED: $file importing $imp was not refused"; exit 1
    fi
    n=$((n + 1))
  done
  mkdir -p "$tmp/layersok/modes/tty"
  printf 'package tty\nimport _ "%s/modes/tty/internal/ttytest"\n' "$MOD" > "$tmp/layersok/modes/tty/ok_test.go"
  if ! check_layers "$tmp/layersok" >/dev/null 2>&1; then
    echo "lint-imports SELF-TEST FAILED: a test file's import of the test support was refused"; exit 1
  fi
  n=$((n + 1))
  echo "lint-imports self-test: all $n controls fired (OK)"
  exit 0
fi
check_tree modes
# platform/ IS A LEAF, and this closes a one-hop hole in the rule above.
#
# The modes/ check is a textual grep, not a transitive one, so
# modes/broadcaster -> platform/lineup -> domains/radio/cast would PASS while
# violating exactly what the modes/ rule exists to enforce. platform/ was
# already clean of domains/* when this was added; nothing was enforcing it, and
# platform/lineup is the first package where the temptation is constant — a
# card's role and subject are domain-shaped unless something says otherwise.
check_tree platform
check_studs .
check_layers .
echo "lint-imports: OK"
