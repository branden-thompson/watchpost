#!/bin/sh
# NFR-2: no shipped artifact may be able to fabricate a hazard.
#
# THE PREDICATE IS A STRING, AND THAT IS NOT A COMPROMISE — IT IS THE ONLY THING
# THAT WORKS. Release artifacts are linked with `-s -w`, which strips the symbol
# table, so `go tool nm` reports "no symbol section" on linux and "no symbols" on
# windows. A symbol-based gate therefore finds zero in a DEBUG build and zero in
# a clean one, and passes everything. Measured, after a symbol gate was written
# and shipped blind for ten minutes.
#
#                                    debug matrix   clean matrix
#   watchpost-injected-  (id prefix)    1 1 1          0 0 0     <- the anchor
#   Injected Test Location             1 1 1          0 0 0
#   INJECT AN ALERT                    1 1 1          1 1 1      <- useless
#   (*tickerDeck).Inject  symbol       0 0 0          0 0 0      <- stripped
#
# `INJECT AN ALERT` is the window's own copy in modes/tty/debug.go, which has NO
# build tag, so it ships in every clean binary. A gate on it fails good builds.
#
# THE ANCHOR IS NOT COPY. `watchpost-injected-` is the id prefix minted by
# Inject; the seen store and the debug log both depend on it, so FR-4.4 can
# rewrite every user-facing marking string without retiring this gate. F-38
# proposed anchoring on "Injected Test Location" — right about strings, and that
# one is exactly the copy FR-4.4 rewrites.
#
#   ./lint-injector.sh <artifact…>
set -eu
ANCHOR='watchpost-injected-'

check() {
  f=$1
  [ -f "$f" ] || { echo "lint-injector: $f does not exist"; return 1; }
  # LIVENESS: a file we cannot read strings from is not a pass. Without this a
  # truncated or unreadable artifact looks identical to a clean one.
  if [ "$(strings "$f" | head -c 1 | wc -c)" -eq 0 ]; then
    echo "lint-injector: $f yields no strings at all — unreadable, so a pass would prove nothing"
    return 1
  fi
  if strings "$f" | grep -q "$ANCHOR"; then
    echo "lint-injector: FORBIDDEN — $f carries the injector ($ANCHOR)."
    echo "  A release artifact that can fabricate a hazard must never ship (F-21b, NFR-2)."
    return 1
  fi
}

if [ "${1:-}" = "--self-test" ]; then
  tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT
  root=$(git rev-parse --show-toplevel)
  # BUILT THE WAY THE MATRIX BUILDS, -s -w included. The first version of this
  # self-test omitted the ldflags, so it exercised an UNSTRIPPED binary — a
  # shape this project never ships — and passed while the real gate was blind.
  LD='-s -w'
  go build -tags watchpost_debug -ldflags "$LD" -o "$tmp/debug" "$root/cmd/watchpost"
  if check "$tmp/debug" >/dev/null 2>&1; then
    echo "lint-injector SELF-TEST FAILED: a stripped DEBUG build was not detected"; exit 1
  fi
  go build -ldflags "$LD" -o "$tmp/clean" "$root/cmd/watchpost"
  if ! check "$tmp/clean" >/dev/null 2>&1; then
    echo "lint-injector SELF-TEST FAILED: a stripped CLEAN build was rejected"; exit 1
  fi
  echo "lint-injector self-test: both controls fired against STRIPPED builds (OK)"
  exit 0
fi

[ $# -gt 0 ] || { echo "usage: lint-injector.sh <artifact…>"; exit 2; }
rc=0
for f in "$@"; do check "$f" || rc=1; done
[ $rc -eq 0 ] && echo "lint-injector: OK ($# artifact(s))"
exit $rc
