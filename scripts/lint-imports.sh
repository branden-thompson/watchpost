#!/bin/sh
# Import-direction gate (architecture.md §1): no file under modes/ may import
# github.com/branden-thompson/watchpost/domains/... . Discovers consumers by walk,
# never a hardcoded list (calibration: Discover Consumers, Don't Enumerate Them).
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
  echo "lint-imports self-test: both controls fired (OK)"
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
echo "lint-imports: OK"
