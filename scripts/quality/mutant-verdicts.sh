#!/bin/sh
# The full corpus VERDICT sweep: apply every mutant, run the tests, record what
# each one actually returns.
#
# WHY THIS EXISTS WHEN TWO GATES ALREADY WATCH THE CORPUS. `make mutant-anchors`
# proves every mutant still FINDS its line. `make mutant-check` proves every one
# still COMPILES with its tests. **NEITHER ASKS WHETHER ANYTHING STILL FAILS WHEN
# IT IS APPLIED**, and that is the only question that makes a corpus evidence
# rather than decoration.
#
# WHAT THAT BLIND SPOT COST, MEASURED 2026-09-13. A one-off sweep of the release's
# 82 newest mutants found two that passed every gate and measured nothing:
#
#   mAA2  a LIVE COVERAGE HOLE. The rule was pinned by a tautology — the test
#         helper re-derived production's own predicate and then asserted on it,
#         so deleting the guard changed nothing the test could see. On the
#         console two settings groups are empty; without that guard the operator
#         gets two headings announcing categories and showing none of them.
#   mAB1  A RULE THE PRODUCT RETIRED. It guarded D-104; D-106 replaced that rule
#         and deleted its detector. The anchor still matched, the mutation still
#         compiled, and it defended an abandoned design. Retired by the HUM LEAD.
#
# Neither shape is visible to anything else in the toolchain.
#
# NOT IN `verify`: it applies and tests every mutant one at a time, which is tens
# of minutes. Local, before BUILD exit and before SHIP — the same standing as
# `make journey`.
#
# A SURVIVOR IS ESCALATED BEFORE IT IS BELIEVED. Each mutant runs against the
# package it EDITS, because that is fast; a mutant caught by a test in another
# package would read as SURVIVED against that narrow set. So anything that
# survives is re-run against `./...` and only then reported. On 2026-09-13 that
# step turned two of four apparent survivors into CAUGHT (mCA, mDC) — which is
# the control this sweep is watched by: without escalation it reports false
# survivors, and it was seen doing so.
set -eu

out=${1:-dist/mutant-verdicts.log}
mkdir -p "$(dirname "$out")"
: > "$out"

total=0 caught=0 survived=0
for m in 06_docs/mutants/m*.py; do
  total=$((total + 1))
  tgt=$(sed -n 's/.*pathlib\.Path("\([^"]*\)").*/\1/p' "$m" | head -1)
  pkg="./$(dirname "$tgt")/"
  v=$(./06_docs/mutants/run.sh "$m" "$pkg" 2>&1 | tail -1)
  case "$v" in
    SURVIVED*)
      v=$(./06_docs/mutants/run.sh "$m" ./... 2>&1 | tail -1)
      case "$v" in
        CAUGHT*) v="$v [caught only against ./...]" ;;
      esac
      ;;
  esac
  case "$v" in
    CAUGHT*)   caught=$((caught + 1)) ;;
    SURVIVED*) survived=$((survived + 1)) ;;
  esac
  echo "$v" | tee -a "$out"
done

echo "" | tee -a "$out"
echo "mutant-verdicts: $total mutant(s) — $caught CAUGHT, $survived SURVIVED" | tee -a "$out"
if [ "$survived" -ne 0 ]; then
  echo "A SURVIVOR IS A RULE NOTHING MEASURES. Either the behaviour is unpinned and" | tee -a "$out"
  echo "owes a test, or the rule was retired and the mutant owes a HUM LEAD" | tee -a "$out"
  echo "retirement — which is RATIFIED, never self-issued." | tee -a "$out"
  exit 1
fi
