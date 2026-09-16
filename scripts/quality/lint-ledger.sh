#!/bin/sh
# The public mirror of the P10 ledger must not name anything a public reader
# cannot reach.
#
# WHY A GATE AND NOT A CONVENTION. `06_docs/p10-ledger.md` is generated, and the
# generator already refuses a leaking row — but a generated file can be
# hand-edited, and the next worker to touch it will not have read the generator.
# A rule the reader has to remember is a rule the next session forgets, which is
# the same argument P10-02 makes about a loop bound: put it in the shape.
#
# WHAT IT REFUSES, and why each matters on a PUBLIC repository:
#   - absolute machine paths        name a person's account and a layout nobody
#                                   outside can use
#   - harness paths                 git-ignored, so a reader follows them to a
#                                   404 and learns the project leans on
#                                   something it cannot see
#   - A2DH skill paths              the most likely leak of all: they live in the
#                                   CHECKER'S OWN JSON, under `skill_path` and
#                                   `next_step`, so an agent drafting a reason
#                                   from a finding carries one in without
#                                   noticing
#   - internal project trees        say where the tooling lives, which tells a
#                                   reader nothing and exposes an internal tree
#   - email addresses               personal data
#
# A FAILURE HERE IS FIXED IN THE LEDGER ROW, then regenerated. It is never fixed
# by editing the mirror, and never by loosening this file.
set -eu

MIRROR=${1:-06_docs/p10-ledger.md}

if [ ! -f "$MIRROR" ]; then
  echo "lint-ledger: $MIRROR is missing — the ratification record is not in the repository."
  echo "  Regenerate it: python3 scripts/quality/p10-ledger-mirror.py"
  exit 1
fi

fail=0
report() {
  echo "lint-ledger: $MIRROR names $2"
  grep -nE "$1" "$MIRROR" | head -5 | sed 's/^/    /'
  fail=1
}

grep -qE '/Users/|/home/|/Volumes/'          "$MIRROR" && report '/Users/|/home/|/Volumes/' "an absolute machine path"
grep -qE '(^|[^A-Za-z0-9_])_a2dh/|\.a2dh'    "$MIRROR" && report '(^|[^A-Za-z0-9_])_a2dh/|\.a2dh' "a path to the local harness"
grep -qE '[0-9][0-9]_skills/'                "$MIRROR" && report '[0-9][0-9]_skills/' "an A2DH skill path"
grep -qE 'LI_PROJECTS|DESIGN_FOUNDATIONS'    "$MIRROR" && report 'LI_PROJECTS|DESIGN_FOUNDATIONS' "an internal project tree"
grep -qE '\bAGENTS\.md\b|\bCLAUDE\.md\b|copilot-instructions' "$MIRROR" && report '\bAGENTS\.md\b|\bCLAUDE\.md\b|copilot-instructions' "a git-ignored harness file"
grep -qE '[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}' "$MIRROR" && report '[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}' "an email address"

if [ "$fail" -ne 0 ]; then
  echo
  echo "  Fix the ROW in the machine-local ledger and regenerate:"
  echo "    python3 scripts/quality/p10-ledger-mirror.py"
  echo "  Do not edit $MIRROR by hand, and do not loosen this check."
  exit 1
fi

rows=$(grep -cE '^\| `' "$MIRROR" || true)
echo "lint-ledger: OK — $rows ratified row(s), no machine paths, no harness paths, no skill paths"
echo "  scope: it checks what the mirror SAYS, not whether each ratification is still deserved."
echo "  A row whose code has changed is a row to re-present, and nothing here can see that."
