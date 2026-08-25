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
  mkdir -p "$tmp/modes/report"
  printf 'package report\nimport _ "%s/domains/weather"\n' "$MOD" > "$tmp/modes/report/bad.go"
  if check_tree "$tmp/modes" >/dev/null 2>&1; then
    echo "lint-imports SELF-TEST FAILED: known-bad fixture not detected"; exit 1
  fi
  echo "lint-imports self-test: control fired (OK)"
  exit 0
fi
check_tree modes
echo "lint-imports: OK"
