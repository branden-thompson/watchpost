#!/usr/bin/env sh
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
#
# THE CONTROL RUNS THE GATE'S OWN CODE. `scan` below is called by the gate and by
# `--self-test` alike, so a control that passes proves the path the gate takes.
# A self-test with a private copy of the rules proves only that the rules match
# something, which is a weaker claim than it looks.
set -eu

case "${1:-}" in
  --self-test) ;;
  -*) echo "lint-ledger: unknown argument '$1'" >&2; exit 2 ;;
esac

# The internal-tree class is defined once, in Go (tools/internaltrees), and read
# here — a copy of its patterns in this file is what a workspace rename leaves
# behind.
TREES=$(cd "$(dirname "$0")/../.." && go run ./tools/internaltrees) || TREES=""
if [ -z "$TREES" ]; then
  echo "lint-ledger: tools/internaltrees produced no pattern — refusing to lint with a rule missing" >&2
  exit 2
fi

fail=0
report() {
  echo "lint-ledger: $MIRROR names $2"
  grep -nE "$1" "$MIRROR" | head -5 | sed 's/^/    /'
  fail=1
}

# scan applies every rule to $MIRROR and sets `fail`. The gate and the control
# both go through here.
scan() {
  MIRROR=$1
  fail=0
  grep -qE '/Users/|/home/|/Volumes/'          "$MIRROR" && report '/Users/|/home/|/Volumes/' "an absolute machine path"
  grep -qE '(^|[^A-Za-z0-9_])_a2dh/|\.a2dh'    "$MIRROR" && report '(^|[^A-Za-z0-9_])_a2dh/|\.a2dh' "a path to the local harness"
  grep -qE '[0-9][0-9]_skills/'                "$MIRROR" && report '[0-9][0-9]_skills/' "an A2DH skill path"
  grep -qE "$TREES"                            "$MIRROR" && report "$TREES" "an internal project tree"
  grep -qE '\bAGENTS\.md\b|\bCLAUDE\.md\b|copilot-instructions' "$MIRROR" && report '\bAGENTS\.md\b|\bCLAUDE\.md\b|copilot-instructions' "a git-ignored harness file"
  grep -qE '[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}' "$MIRROR" && report '[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}' "an email address"
  return 0
}

# --self-test: every rule must FIRE on a mirror carrying its leak, and the whole
# set must PASS on one carrying none. A checker that flagged everything would
# pass the first half of this table and be useless; a checker that flagged
# nothing would pass the second half and be worse.
if [ "${1:-}" = "--self-test" ]; then
  d=$(mktemp -d) || exit 2
  trap 'rm -rf "$d"' EXIT
  fired=0
  for probe in \
    'a row naming /Users/someone/thing' \
    'a row naming _a2dh/CONFIGS_a2dh.yaml' \
    'a row naming 02_skills/implementation/x.md' \
    'a row naming LI_PROJECTS/somewhere' \
    'a row naming 07__SOME_BUCKET/some-repo' \
    'a row naming 09-ARCHIVE/somewhere' \
    'a row naming AGENTS.md' \
    'a row naming someone@example.com'
  do
    printf '| `x` | %s |\n' "$probe" > "$d/leak.md"
    if scan "$d/leak.md" >/dev/null 2>&1 && [ "$fail" -eq 0 ]; then
      echo "lint-ledger self-test: FAILED — no rule fired on: $probe" >&2
      exit 1
    fi
    fired=$((fired + 1))
  done

  printf '| `platform/render/units.go` | a ratified reason naming only repo-relative paths |\n' > "$d/clean.md"
  scan "$d/clean.md" >/dev/null 2>&1
  if [ "$fail" -ne 0 ]; then
    echo "lint-ledger self-test: FAILED — a clean mirror was rejected; the gate would train authors to delete reasons" >&2
    exit 1
  fi
  echo "lint-ledger self-test: all $fired leak class(es) fired, and a clean mirror passed (OK)"
  exit 0
fi

MIRROR=${1:-06_docs/p10-ledger.md}

if [ ! -f "$MIRROR" ]; then
  echo "lint-ledger: $MIRROR is missing — the ratification record is not in the repository."
  echo "  Regenerate it: python3 scripts/quality/p10-ledger-mirror.py"
  exit 1
fi

scan "$MIRROR"

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
