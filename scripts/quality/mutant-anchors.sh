#!/usr/bin/env sh
# Mutant anchor gate: every mutant in the corpus must still MATCH the tip.
#
# WHY THIS EXISTS WHEN `mutant-check` ALREADY ANSWERS IT. It answers it in ~400
# seconds, because it also COMPILES every mutation — which is the other half of
# the question and the expensive half. This asks only "does the anchor still
# find its line", takes about three seconds, and is meant to be run BEFORE a
# commit rather than after one.
#
# THE COST IT IS PAYING BACK. A mutant is coupled to the LITERAL TEXT of the rule
# it guards, so refactoring that line disarms it silently. Four drifted in one
# session on 2026-09-13 — mAF1 twice, m85, mAN1 — every one found by the slow
# gate, after the commit that broke it. The rules were never unmeasured for long,
# but "my new mutants pass" was repeatedly mistaken for "the corpus still
# measures what it did", and those are different sentences.
#
# A DRIFTED MUTANT IS NOT A PASSING MUTANT. It is an UNMEASURED rule, which is
# why this fails rather than warns: the mutation would apply to nothing, the
# harness would report no failure, and a corpus of those is a green wall that
# guards nothing.
#
# IT RUNS EACH MUTANT'S OWN ASSERTION, and that is the whole design. The obvious
# implementation reads `old = "..."` out of the file with a regex and looks for
# that string — a SECOND parser for the corpus, which can disagree with the
# harness that actually applies these. It already did: the first draft of this
# check decoded anchors with `unicode_escape` and reported a false drift on the
# one mutant whose anchor contains a `◆`, because that codec reads UTF-8 bytes as
# latin-1. Four mutants also COMPUTE their anchor rather than writing it as a
# literal, and no regex sees those at all.
#
# So the mutant is EXECUTED with `write_text` neutered. The `assert old in s` it
# already carries is the check, run against the real tree, and nothing is
# written. One parser, the same one, and the answer cannot drift from the gate it
# is standing in for. Every mutant in the corpus performs exactly one write and
# no other file operation, which is what makes that safe; the driver fails loudly
# on anything else it is asked to do.
set -eu

MUTANTS="06_docs/mutants"

case "${1:-}" in
  ""|--self-test) ;;
  *) echo "mutant-anchors: unknown argument '$1'" >&2; exit 2 ;;
esac

# driver checks a list of mutant files against the working tree and prints one
# line per drifted mutant. It writes nothing, ever.
driver() {
  python3 - "$@" <<'PY'
import pathlib, sys

# THE ONE SIDE EFFECT, REMOVED. Neutering the write is what makes this a read-only
# check; the mutant's own `assert` still runs against the real file first, which
# is the thing being asked.
written = []
pathlib.Path.write_text = lambda self, *a, **k: written.append(self)

bad = 0
for name in sys.argv[1:]:
    f = pathlib.Path(name)
    try:
        exec(compile(f.read_text(), name, "exec"), {"__name__": "__mutant__", "__file__": name})
    except AssertionError as e:
        print("  DRIFTED  %s — the anchor no longer matches the tip (%s)" % (f.name, e or "assert"))
        bad += 1
    except FileNotFoundError as e:
        print("  MISSING  %s — it edits a file that is gone (%s)" % (f.name, e))
        bad += 1
    except Exception as e:                                    # noqa: BLE001 - reported, not swallowed
        print("  BROKEN   %s — %s: %s" % (f.name, type(e).__name__, e))
        bad += 1
print("__COUNT__ %d %d" % (len(sys.argv) - 1, bad))
PY
}

if [ "${1:-}" = "--self-test" ]; then
  # TWO CONTROLS, AND BOTH MUST FIRE. A gate that has only ever been watched
  # passing has not been watched at all: this one's failure mode is reporting a
  # clean corpus, which looks exactly like success.
  tmp=$(mktemp -d)
  trap 'rm -rf "$tmp"' EXIT INT TERM
  mkdir -p "$tmp/src" "$tmp/m"
  printf 'package x\n\nfunc live() bool { return true }\n' > "$tmp/src/live.go"

  cat > "$tmp/m/good.py" <<PY
import pathlib
p = pathlib.Path("$tmp/src/live.go"); s = p.read_text()
old = "return true"
assert old in s, "good"
p.write_text(s.replace(old, "return false", 1))
PY
  cat > "$tmp/m/drifted.py" <<PY
import pathlib
p = pathlib.Path("$tmp/src/live.go"); s = p.read_text()
old = "return maybe"
assert old in s, "drifted"
p.write_text(s.replace(old, "return false", 1))
PY
  cat > "$tmp/m/missing.py" <<PY
import pathlib
p = pathlib.Path("$tmp/src/gone.go"); s = p.read_text()
old = "anything"
assert old in s, "missing"
p.write_text(s.replace(old, "", 1))
PY

  out=$(driver "$tmp/m/good.py" "$tmp/m/drifted.py" "$tmp/m/missing.py")
  fired=0
  echo "$out" | grep -q "DRIFTED  drifted.py" && fired=$((fired + 1)) \
    || echo "mutant-anchors self-test: a DRIFTED anchor was not reported" >&2
  echo "$out" | grep -q "MISSING  missing.py" && fired=$((fired + 1)) \
    || echo "mutant-anchors self-test: a MISSING target was not reported" >&2
  echo "$out" | grep -q "good.py" \
    && echo "mutant-anchors self-test: a MATCHING anchor was reported as drifted" >&2 \
    || fired=$((fired + 1))
  # AND THE TREE WAS NOT TOUCHED, which is the property that lets this run on a
  # dirty working copy — the reason it can be used before a commit at all.
  if grep -q "return true" "$tmp/src/live.go"; then
    fired=$((fired + 1))
  else
    echo "mutant-anchors self-test: the check WROTE to the tree; it must not" >&2
  fi
  if [ "$fired" -ne 4 ]; then
    echo "mutant-anchors self-test: $fired of 4 controls fired" >&2
    exit 1
  fi
  echo "mutant-anchors self-test: all 4 controls fired (OK)"
  exit 0
fi

out=$(driver "$MUTANTS"/m*.py)
count=$(echo "$out" | sed -n 's/^__COUNT__ \([0-9]*\) \([0-9]*\)$/\1 \2/p')
set -- $count
total=${1:-0}
bad=${2:-0}
echo "$out" | grep -v '^__COUNT__' || true
if [ "$bad" -ne 0 ]; then
  echo "mutant-anchors: $bad of $total mutant(s) no longer match the tip."
  echo "Re-point each at the line its rule now lives on. A drifted mutant is an"
  echo "UNMEASURED rule, not a passing one — the mutation applies to nothing and"
  echo "the harness reports no failure."
  exit 1
fi
echo "mutant-anchors: $total mutant(s), every anchor still matches the tip"
