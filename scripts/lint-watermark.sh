#!/bin/sh
# Human-Accountability Attribution gate: no AI watermarks in tracked files or the
# branch's commit messages (calibration 2026-08-19). Patterns checked as literals.
set -eu
PAT='Co-Authored-By|Generated with \[?Claude|Claude-Session|noreply@anthropic'
scan() { grep -rInE "$PAT" "$@" 2>/dev/null | grep -v -e '_a2dh/' -e 'scripts/lint-watermark.sh' -e '06_docs/.*red-team' -e '06_docs/.*calibration' || true; }
if [ "${1:-}" = "--self-test" ]; then
  tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT
  printf '// Co-Authored-By: some-agent\n' > "$tmp/bad.go"
  found=$(scan "$tmp")
  if [ -z "$found" ]; then echo "lint-watermark SELF-TEST FAILED: control not detected"; exit 1; fi
  echo "lint-watermark self-test: control fired (OK)"
  exit 0
fi
bad=$(git ls-files -z | grep -zv -e '^06_docs/' -e '^_a2dh/' -e '^scripts/lint-watermark.sh$' | xargs -0 grep -lInE "$PAT" 2>/dev/null || true)
# Branch range when it exists; last 20 commits as the post-merge fallback (H-4).
range=$(git log --format=%B main..HEAD 2>/dev/null || true) # a tag checkout has no 'main' ref: fall through (set -e would otherwise exit 128 here)
[ -z "$range" ] && range=$(git log --format=%B -n 20 2>/dev/null || true)
msgs=$(echo "$range" | grep -nE "$PAT" || true)
if [ -n "$bad" ] || [ -n "$msgs" ]; then
  echo "lint-watermark: AI watermark found:"; [ -n "$bad" ] && echo "$bad"; [ -n "$msgs" ] && echo "commit messages: $msgs"
  exit 1
fi
echo "lint-watermark: OK"
