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

# THE WHOLE TREE, NOT THE DIFF. An anti-pattern is not excused by predating the
# session that finds it: walking into a room that is already a mess does not make
# the mess someone else's. A contributor leaves the place better than they found
# it, so the check asks "does this repository contain one", never "did I write
# one" — and a check scoped to authorship is a check that can be made quiet by
# narrowing it, which is exactly what happened to this file's first version.
#
# EXCLUDES `third_party/` ONLY, because that tree is vendored and not ours to
# rewrite; its gaps go upstream instead.
files=$(git ls-files '*.go' | grep -v '^third_party/' || true)
[ -n "$files" ] || { echo "lint-authoring: no Go files"; exit 0; }

fail=0

# --- AP-HIST-01: comments describe the code, not the history that led to it ---
#
# THE PHRASES ARE THE CATALOGUE'S OWN ANTI-SPECIMENS plus the shapes a
# remediation produces: an account of its own correction. It is the NARRATION
# that is banned, not the reasoning — "the boundary errs towards telling the
# listener" is the code as it stands and passes; "corrected at D-160" is a past
# the reader does not have and does not.
HIST='used to (be|do|have|return|call)|(^|[^a-z])legacy( compatibility|:)|for backward compat|backwards compat|the old (API|behaviour|behavior|field|name|way)|an earlier version|previously (it|this|we)|corrected at D-[0-9]|(found|caught) by (a |the )?(blind|red[- ]team)|red[- ]?team.s (round|second|third|final)|said the opposite|the first fix|for a fortnight|until D-[0-9]|stayed in place after|was [0-9]+, bumped'

hist=$(grep -nEi "^[[:space:]]*(//|\*)" $files 2>/dev/null | grep -Ei "$HIST" || true)
if [ -n "$hist" ]; then
  n=$(printf '%s\n' "$hist" | wc -l | tr -d ' ')
  echo "lint-authoring: AP-HIST-01 — $n comment(s) narrate history"
  printf '%s\n' "$hist" | head -12 | sed 's/^/    /'
  [ "$n" -gt 12 ] && echo "    … and $((n - 12)) more"
  fail=1
fi

# --- AP-DEAD-01: a declaration kept alive by a blank assignment ---
#
# `_ = x` on its own line is how unused code survives the compiler. The P10
# checker reads it as a USE, which is how a dead closure sat in the request
# window with a gate watching the file.
dead=$(grep -nE '^[[:space:]]*_ = [a-zA-Z][a-zA-Z0-9_]*[[:space:]]*$' $files 2>/dev/null || true)
if [ -n "$dead" ]; then
  n=$(printf '%s\n' "$dead" | wc -l | tr -d ' ')
  echo "lint-authoring: AP-DEAD-01 — $n declaration(s) kept alive by a blank assignment"
  printf '%s\n' "$dead" | head -6 | sed 's/^/    /'
  fail=1
fi

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
