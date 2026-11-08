#!/usr/bin/env sh
# The full corpus VERDICT sweep: apply every mutant, run the tests, record what
# each one actually returns.
#
# WHY THIS EXISTS WHEN TWO GATES ALREADY WATCH THE CORPUS. `make mutant-anchors`
# proves every mutant still FINDS its line. `make mutant-check` proves every one
# still COMPILES with its tests. **NEITHER ASKS WHETHER ANYTHING STILL FAILS WHEN
# IT IS APPLIED**, and that is the only question that makes a corpus evidence
# rather than decoration.
#
# WHAT THAT BLIND SPOT HIDES: a mutant that passes every other gate and measures
# nothing.
#
#   a LIVE COVERAGE HOLE. The rule is pinned by a tautology — the test helper
#         re-derives production's own predicate and asserts on it, so deleting
#         the guard changes nothing the test can see.
#   A RULE THE PRODUCT RETIRED. Its anchor still matches and its mutation still
#         compiles, and it defends an abandoned design.
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
# survives is re-run against `./...` and only then reported. Without escalation
# the sweep reports false survivors.
set -eu

out=${1:-dist/mutant-verdicts.log}
mkdir -p "$(dirname "$out")"
: > "$out"

# THE TIMINGS DO NOT FOLLOW THE LOG, and that matters: the log
# is PROMOTED out of dist into the release's readiness tree, because it is the
# run's record and the hygiene protocol requires a record to be durable and
# tracked. The timings are neither.
#
# THEY ARE AN OPERATIONAL INPUT, NOT A RECORD. The next run reads them to
# compute a MEASURED eta rather than a guessed one — which is the whole reason
# they exist. They are regenerable from any
# run, so committing them would put a machine-specific stopwatch into the
# repository's history for no reader. They stay in dist, and `HYGIENE_KEEP`
# names them so the cleanup spares them.
timings=${MUTANT_TIMINGS:-dist/mutant-verdicts.timings}
mkdir -p "$(dirname "$timings")"

list=$(ls 06_docs/mutants/m*.py)
total=$(echo "$list" | wc -l | tr -d ' ')

# pkgof is the package a mutant edits, from its first pathlib.Path literal in
# either quote, or ./... when the target is not a Go file (a Makefile or a
# workflow, whose detectors live anywhere), or nothing when it finds no target,
# which the loop refuses rather than testing the repository root.
pkgof() {
	t=$(sed -n "s/.*pathlib\.Path([\"']\([^\"']*\)[\"']).*/\1/p" "$1" | head -1)
	[ -n "$t" ] || return 0
	case "$t" in
	*.go) dirname "$t" ;;
	*) echo "..." ;;
	esac
}

# THE ETA IS MEASURED, NOT GUESSED, and that is the whole of this block.
#
# AN ESTIMATE BY HAND IS WRONG BY MULTIPLES. A sample of RECENT mutants is not
# representative — their detectors fail early, so `go test` exits early. So the
# estimate is not a judgement call: every run records seconds-per-mutant per
# package, and the next run reads them.
#
# NO HISTORY MEANS NO ESTIMATE, said out loud. A made-up number is what this
# exists to stop.
say_eta() {
	if [ ! -f "$timings" ]; then
		echo "  ETA: unavailable — no timing history yet ($timings). This run will record it."
		return
	fi
	secs=0
	for m in $list; do
		p=$(pkgof "$m")
		r=$(grep "^$p " "$timings" 2>/dev/null | tail -1 | awk '{print $2}')
		[ -z "$r" ] && r=$(awk '{s+=$2; n++} END {if (n) printf "%d", s/n; else print 30}' "$timings")
		secs=$((secs + r))
	done
	if [ "$secs" -lt 90 ]; then
		echo "  ETA: under a minute, from the per-package rates this corpus actually ran at"
	else
		echo "  ETA: ~$((secs / 60)) min, from the per-package rates this corpus actually ran at"
	fi
}

echo "mutant-verdicts: $total mutant(s)" | tee -a "$out"
echo "  log: $out  (watch this file; the run streams into it)" | tee -a "$out"
say_eta | tee -a "$out"
echo "  every line carries its position, so any read of this file says where the run is" | tee -a "$out"
echo "" | tee -a "$out"

# THE BASELINE IS ESTABLISHED ONCE PER PACKAGE, not once per mutant.
#
# `run.sh` proves the unmutated tree is green before every mutation, which is
# right for single use and is the same ~100 s answer 99 times over on ./app. The
# sweep holds ONE clean tree across the whole run — `run.sh` refuses a dirty one,
# and that check is NOT skipped — so the answer cannot differ between mutants in
# a package. Measured: it halves a full sweep.
baseline_ok=""
ensure_baseline() {
	case " $baseline_ok " in *" $1 "*) return 0 ;; esac
	printf 'baseline %s ... ' "$1" | tee -a "$out"
	if ! go test "$1" -count=1 >/dev/null 2>&1; then
		echo "RED — the unmutated tree is not green; fix that first" | tee -a "$out"
		exit 2
	fi
	echo "green" | tee -a "$out"
	baseline_ok="$baseline_ok $1"
}

: > "$timings.new"
n=0 caught=0 survived=0 noevidence=0
for m in $list; do
	n=$((n + 1))
	pk=$(pkgof "$m")
	case "$pk" in
	"") echo "mutant-verdicts: $m names no target this script can read" | tee -a "$out"; exit 2 ;;
	...) pkg="./..." ;;
	*) pkg="./$pk/" ;;
	esac
	ensure_baseline "$pkg"
	start=$(date +%s)
	v=$(MUTANT_BASELINE=assumed ./06_docs/mutants/run.sh "$m" "$pkg" 2>&1 | tail -1)
	case "$v" in
	SURVIVED*)
		# A SURVIVOR IS ESCALATED TWICE BEFORE IT IS BELIEVED, and only survivors
		# pay for it.
		#
		# ./... because a mutant run against the package it EDITS reads as
		# SURVIVED when its detector lives elsewhere.
		#
		# AND THEN -race, because some detectors only work under it. The press
		# gate's own test measures 20/20 with it and ~81/100 without. A survivor
		# that needs the race detector is not a survivor.
		v=$(MUTANT_BASELINE=assumed ./06_docs/mutants/run.sh "$m" ./... 2>&1 | tail -1)
		case "$v" in
		SURVIVED*)
			v=$(MUTANT_RACE=1 ./06_docs/mutants/run.sh "$m" "$pkg" 2>&1 | tail -1)
			case "$v" in
			CAUGHT*) v="$v [caught only under -race]" ;;
			esac
			;;
		CAUGHT*) v="$v [caught only against ./...]" ;;
		esac
		;;
	esac
	# ONLY A REAL VERDICT IS TIMED. A SKIPPED run takes no time and means nothing,
	# and a history full of them makes the next ETA say "~0 min" — which is how
	# this script's first smoke test reported a three-hour corpus. Timing what did
	# not happen is worse than having no history, because it looks like history.
	case "$v" in
	CAUGHT* | SURVIVED*)
		# KEYED BY THE PACKAGE `pkgof` NAMES, so the next run's lookup matches.
		echo "$pk $(( $(date +%s) - start ))" >> "$timings.new"
		;;
	esac
	case "$v" in
	CAUGHT*) caught=$((caught + 1)) ;;
	SURVIVED*) survived=$((survived + 1)) ;;
	# SKIPPED, UNAPPLIED AND INVALID ARE NOT VERDICTS. Each means the mutation
	# was never measured — a dirty tree, an anchor that no longer matches, an
	# edit that will not compile — and a sweep that counts them as nothing at all
	# reports "0 CAUGHT, 0 SURVIVED" and exits 0. THAT IS A GREEN WALL, and this
	# script wrote one on its first smoke test.
	*) noevidence=$((noevidence + 1)) ;;
	esac
	echo "[$n/$total] $v" | tee -a "$out"
done
# AND A HISTORY IS ONLY REPLACED BY A HISTORY. A run that measured nothing must
# not wipe the rates the last good run recorded.
if [ -s "$timings.new" ]; then
	mv "$timings.new" "$timings"
else
	rm -f "$timings.new"
fi

echo "" | tee -a "$out"
echo "mutant-verdicts: $total mutant(s) — $caught CAUGHT, $survived SURVIVED, $noevidence NO EVIDENCE" | tee -a "$out"
echo "  per-mutant timings recorded in $timings; the next run estimates from them" | tee -a "$out"
if [ "$noevidence" -ne 0 ]; then
	echo "NO EVIDENCE is not a pass. $noevidence mutant(s) were never measured —" | tee -a "$out"
	echo "SKIPPED (dirty tree), UNAPPLIED (the anchor moved) or INVALID (the edit" | tee -a "$out"
	echo "does not compile). A sweep that reports only what it managed to run is a" | tee -a "$out"
	echo "sweep whose green means nothing." | tee -a "$out"
	exit 1
fi
if [ "$survived" -ne 0 ]; then
	echo "A SURVIVOR IS A RULE NOTHING MEASURES. Either the behaviour is unpinned and" | tee -a "$out"
	echo "owes a test, or the rule was retired and the mutant owes a HUM LEAD" | tee -a "$out"
	echo "retirement — which is RATIFIED, never self-issued." | tee -a "$out"
	exit 1
fi
