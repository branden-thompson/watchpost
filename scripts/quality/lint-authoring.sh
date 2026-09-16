#!/bin/sh
# The A2DH code-authoring rules that a machine can check, checked by a machine.
#
# WHY THIS EXISTS. `AP-HIST-01` has been in the catalogue since before this
# release, is cited by `code-documentation`, by `semantic-code` and by two
# red-team axes, and is routed by name in the harness's own table. It was still
# violated 29 times across 20 files in a single remediation pass. The rule was
# never the problem; the rule had no carrier at the moment it applies.
#
# A PRECONDITION WITH NO MECHANICAL TRIGGER IS A COIN FLIP. An instruction to
# load a skill before authoring competes with whatever the author is already
# doing, and loses. A check that runs on the diff does not.
#
# SCOPED TO WHAT CHANGED, against the merge base, so it reports what this branch
# wrote rather than what it inherited. A rule applied retroactively to a whole
# tree is a rule nobody can act on.
set -eu

BASE=${1:-$(git merge-base origin/main HEAD 2>/dev/null || echo HEAD)}
files=$(git diff --name-only --diff-filter=ACM "$BASE"...HEAD -- '*.go' 2>/dev/null | grep -v '^third_party/' || true)
[ -n "$files" ] || { echo "lint-authoring: no changed Go files"; exit 0; }

fail=0

# --- AP-HIST-01: comments describe the code, not the history that led to it ---
#
# THE PHRASES ARE THE CATALOGUE'S OWN ANTI-SPECIMENS plus the shapes this
# repository produced: a remediation narrating its own correction. It is the
# narration that is banned, not the reasoning — "the boundary errs towards
# telling the listener" is the code as it stands and passes; "corrected at
# D-160" is a past the reader does not have and does not.
HIST='used to (be|do|have|return|call)|(^|[^a-z])legacy( compatibility|:)|for backward compat|backwards compat|the old (API|behaviour|behavior|field|name|way)|an earlier version|previously (it|this|we)|this (used|was) (to|previously)|corrected at D-[0-9]|(found|caught) by (a |the )?(blind|red[- ]team)|red[- ]?team.s (round|second|third|final)|said the opposite|the first fix|for a fortnight|until D-[0-9]|stayed in place after|was [0-9]+, bumped'

for f in $files; do
  hits=$(grep -nEi "^[[:space:]]*(//|\*)" "$f" 2>/dev/null | grep -nEi "$HIST" || true)
  if [ -n "$hits" ]; then
    echo "lint-authoring: AP-HIST-01 — comments narrating history in $f"
    printf '%s\n' "$hits" | head -6 | sed 's/^/    /'
    fail=1
  fi
done

# --- AP-DEAD-01: a declaration kept alive only by a blank assignment ---
#
# `_ = x` on its own line is how unused code survives the compiler. The P10
# checker reads it as a USE, which is exactly how a dead closure sat in the
# request window for two weeks with a gate watching the file.
for f in $files; do
  hits=$(grep -nE '^[[:space:]]*_ = [a-zA-Z][a-zA-Z0-9_]*[[:space:]]*$' "$f" 2>/dev/null || true)
  if [ -n "$hits" ]; then
    echo "lint-authoring: AP-DEAD-01 — declaration kept alive by a blank assignment in $f"
    printf '%s\n' "$hits" | head -4 | sed 's/^/    /'
    echo "    If it is genuinely needed, say why on the line; if not, delete the declaration."
    fail=1
  fi
done

if [ "$fail" -ne 0 ]; then
  echo
  echo "  AP-HIST-01: state what the code does and why NOW. The change belongs in git log,"
  echo "  06_docs/follow-ups.md and the gate roster — all three already record it."
  echo "  AP-DEAD-01: delete dead code rather than suppressing the compiler."
  exit 1
fi

echo "lint-authoring: OK — no history narration, no blank-assignment keep-alives in changed Go"
echo "  scope: CHANGED files only, and only the two rules a grep can decide."
echo "  It cannot see a comment that is merely WRONG, which is the larger class and still needs a reader."
