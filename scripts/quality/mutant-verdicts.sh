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
# NOT IN `verify`: it applies and tests every mutant. Local, before BUILD exit
# and before SHIP — the same standing as `make journey`.
#
# IT JUDGES THE COMMIT, NOT THE WORKING TREE. Each worker is a git worktree of
# HEAD outside the repository, so the working tree can be edited while a sweep
# runs, and an uncommitted change is not in the verdicts. The record's first
# line names the commit.
#
# HOW A MUTANT IS JUDGED, cheapest first, each step only when the one before
# did not catch it:
#
#   1. its last killer. The previous record names the test that caught it, and
#      that test runs alone, in whichever package defines it. Most mutants end
#      here in seconds, against minutes for their package's whole suite.
#   2. the package it edits, whole, stopping at the first failure.
#   3. `./...`, because a detector can live in another package.
#   4. the package under `-race`, because some detectors only work under it.
#
# Only a mutant that survives all four is SURVIVED. A catch at any step has
# run.sh's attribution check: the failing test must pass with the mutation put
# back, or the verdict is INVALID.
#
# PARALLEL WORKERS LOAD THE MACHINE, and that cannot turn a survivor into a
# catch: a test that fails under load and passes alone fails the attribution
# check. Load can only make the run slower. MUTANT_JOBS sets the workers.
#
#   MUTANT_JOBS=<n>       workers, default 4
#   MUTANT_KILLERS=<log>  the record step 1 reads; with none, every mutant
#                         starts at step 2
#   MUTANT_ONLY=<regex>   judge only the mutants whose file names match. For
#                         checking a few by hand: call this script with a log
#                         path of your own, since `make mutant-verdicts`
#                         promotes its log as the release's record. A partial
#                         run records no timings.
set -eu

out=${1:-dist/mutant-verdicts.log}
mkdir -p "$(dirname "$out")"
: > "$out"
root=$(git rev-parse --show-toplevel)
case "$out" in /*) ;; *) out="$root/$out" ;; esac
# The paths are absolute because the workers run in other directories, but
# the log is promoted into a tracked record, and an absolute path there names
# this machine's home directory (TestThePublishedTreeNamesNoPersonOrMachine
# fails on it). Every path the log prints goes through `shown`.
shown() { case "$1" in "$root"/*) echo "${1#"$root"/}" ;; *) echo "$1" ;; esac; }
jobs=${MUTANT_JOBS:-4}
killers=${MUTANT_KILLERS:-}
only=${MUTANT_ONLY:-}

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
case "$timings" in /*) ;; *) timings="$root/$timings" ;; esac
mkdir -p "$(dirname "$timings")"

work=$(mktemp -d "${TMPDIR:-/tmp}/watchpost-sweep.XXXXXX")
# The worktrees are removed however the run ends, an interrupt included, and
# the workers stopped first; `git worktree prune` drops what a killed run left
# registered.
pids=""
cleanup() {
	[ -n "$pids" ] && kill $pids 2>/dev/null
	for wt in "$work"/w*; do
		[ -d "$wt" ] && git -C "$root" worktree remove --force "$wt" >/dev/null 2>&1
	done
	git -C "$root" worktree prune >/dev/null 2>&1
	rm -rf "$work"
}
trap cleanup EXIT
trap 'exit 130' INT TERM

head=$(git -C "$root" rev-parse HEAD)
git -C "$root" ls-tree --name-only HEAD 06_docs/mutants/ | grep '/m[^/]*\.py$' | sort > "$work/all"
if [ -n "$only" ]; then
	grep -E "$only" "$work/all" > "$work/list" || true
else
	cp "$work/all" "$work/list"
fi
total=$(wc -l < "$work/list" | tr -d ' ')
[ "$total" -gt 0 ] || { echo "mutant-verdicts: no mutant matches" | tee -a "$out"; exit 2; }

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

# THE LAST KILLERS: mutant file, then the top-level test that caught it, from
# the previous record. A crash names no test, so a mutant caught that way
# starts at step 2.
: > "$work/killers"
if [ -n "$killers" ] && [ -f "$killers" ]; then
	sed -n 's/^\[[0-9]*\/[0-9]*\] CAUGHT  *\([^ ]*\) — \(Test[A-Za-z0-9_]*\).*/\1 \2/p' "$killers" > "$work/killers"
fi
# Where each test is defined: name, then package directory. A name defined in
# more than one package keeps every one; step 1 prefers the mutant's own.
git -C "$root" grep -n -E '^func Test[A-Za-z0-9_]+\(' HEAD -- '*_test.go' |
	sed -n 's/^HEAD:\([^:]*\):[0-9]*:func \(Test[A-Za-z0-9_]*\)(.*/\2 \1/p' |
	while read -r name file; do echo "$name $(dirname "$file")"; done > "$work/tests"

# THE ETA IS MEASURED, NOT GUESSED, and that is the whole of this block.
#
# AN ESTIMATE BY HAND IS WRONG BY MULTIPLES. A sample of RECENT mutants is not
# representative — their detectors fail early, so `go test` exits early. So the
# estimate is not a judgement call: every run records seconds-per-mutant per
# package, and the next run reads them, divided across the workers.
#
# NO HISTORY MEANS NO ESTIMATE, said out loud. A made-up number is what this
# exists to stop.
say_eta() {
	if [ ! -f "$timings" ]; then
		echo "  ETA: unavailable — no timing history yet. This run will record it."
		return
	fi
	secs=0
	while read -r m; do
		p=$(pkgof "$root/$m")
		r=$(grep "^$p " "$timings" 2>/dev/null | awk '{s+=$2; n++} END {if (n) printf "%d", s/n}')
		[ -z "$r" ] && r=$(awk '{s+=$2; n++} END {if (n) printf "%d", s/n; else print 30}' "$timings")
		secs=$((secs + r))
	done < "$work/list"
	secs=$((secs / jobs))
	if [ "$secs" -lt 90 ]; then
		echo "  ETA: under a minute, from the per-package rates this corpus actually ran at"
	else
		echo "  ETA: ~$((secs / 60)) min on $jobs workers, from the per-package rates this corpus actually ran at"
	fi
}

echo "mutant-verdicts: $total mutant(s), commit $head, $jobs workers" | tee -a "$out"
if [ -n "$(git -C "$root" status --porcelain)" ]; then
	echo "  the working tree has uncommitted changes; they are not in this sweep" | tee -a "$out"
fi
echo "  log: $(shown "$out")  (watch this file; the run streams into it)" | tee -a "$out"
say_eta | tee -a "$out"
echo "  every line carries its position, so any read of this file says where the run is" | tee -a "$out"
echo "" | tee -a "$out"

i=0
while [ "$i" -lt "$jobs" ]; do
	i=$((i + 1))
	git -C "$root" worktree add --detach -q "$work/w$i" "$head"
done

# ONE BASELINE, THE WHOLE SUITE, BEFORE ANY MUTANT. A mutant is only evidence
# against a green tree, and every worktree is the same commit, so one run
# answers for all of them and for every step's package.
t0=$(date +%s)
printf 'baseline ./... ... ' | tee -a "$out"
if ! (cd "$work/w1" && go test ./... -count=1 >/dev/null 2>&1); then
	echo "RED — the unmutated tree is not green; fix that first" | tee -a "$out"
	exit 2
fi
echo "green" | tee -a "$out"
t1=$(date +%s)

# killer_pkg names the package that defines a mutant's last killer, its own
# package first when the name is defined in more than one.
killer_pkg() {
	k=$(grep "^$1 " "$work/killers" | head -1 | cut -d' ' -f2)
	[ -n "$k" ] || return 0
	dirs=$(grep "^$k " "$work/tests" | cut -d' ' -f2)
	[ -n "$dirs" ] || return 0
	if echo "$dirs" | grep -qx "$2"; then
		echo "$k $2"
	else
		echo "$k $(echo "$dirs" | head -1)"
	fi
}

# judge runs the four steps for one mutant, in the current worktree, and
# prints run.sh's last line, marked with the step that caught it past step 2.
judge() {
	m=$1 pk=$2
	case "$pk" in
	...) pkg="./..." ;;
	*) pkg="./$pk/" ;;
	esac
	v=""
	kp=$(killer_pkg "$m" "$pk")
	if [ -n "$kp" ]; then
		v=$(MUTANT_BASELINE=assumed MUTANT_RUN="^${kp%% *}\$" "$runsh" "$m" "./${kp#* }/" 2>&1 | tail -1)
		case "$v" in CAUGHT*) echo "$v"; return ;; esac
	fi
	v=$(MUTANT_BASELINE=assumed "$runsh" "$m" "$pkg" 2>&1 | tail -1)
	case "$v" in
	SURVIVED*)
		v=$(MUTANT_BASELINE=assumed "$runsh" "$m" ./... 2>&1 | tail -1)
		case "$v" in
		SURVIVED*)
			v=$(MUTANT_RACE=1 "$runsh" "$m" "$pkg" 2>&1 | tail -1)
			case "$v" in
			CAUGHT*) v="$v [caught only under -race]" ;;
			esac
			;;
		CAUGHT*) v="$v [caught only against ./...]" ;;
		esac
		;;
	esac
	echo "$v"
}

# THE QUEUE. Each worker takes the next mutant when it is free, so a slow
# package does not hold the others. `mkdir` is the lock: it either creates the
# directory or fails, in one step, on every filesystem this runs on.
echo 0 > "$work/next"
take() {
	until mkdir "$work/lock" 2>/dev/null; do sleep 0.05; done
	n=$(($(cat "$work/next") + 1))
	echo "$n" > "$work/next"
	rmdir "$work/lock"
	echo "$n"
}

# run.sh comes from the working tree, not the worktree, so the instrument
# that judges is the one being edited; the mutants and the code are HEAD's.
runsh="$root/06_docs/mutants/run.sh"
worker() {
	cd "$work/w$1"
	while :; do
		n=$(take)
		[ "$n" -le "$total" ] || return 0
		m=$(sed -n "${n}p" "$work/list")
		pk=$(pkgof "$m")
		if [ -z "$pk" ]; then
			v="UNAPPLIED $m — names no target this script can read"
		else
			start=$(date +%s)
			v=$(judge "$m" "$pk")
			case "$v" in
			# ONLY A REAL VERDICT IS TIMED. A SKIPPED run takes no time and means
			# nothing, and a history full of them makes the next ETA say "~0 min".
			CAUGHT* | SURVIVED*) echo "$pk $(( $(date +%s) - start ))" >> "$work/timings.$1" ;;
			esac
		fi
		echo "[$n/$total] $v" | tee -a "$out" >> "$work/verdicts"
	done
}

i=0
while [ "$i" -lt "$jobs" ]; do
	i=$((i + 1))
	worker "$i" &
	pids="$pids $!"
done
wait
pids=""
t2=$(date +%s)

# THE RECORD READS IN CORPUS ORDER. Workers finish out of order, so the log
# is rewritten with its verdicts sorted by position once all are in.
head -n "$(grep -n '^baseline ' "$out" | head -1 | cut -d: -f1)" "$out" > "$work/record"
sort -t/ -k1.2n "$work/verdicts" >> "$work/record"
cp "$work/record" "$out"

caught=$(grep -c '^\[[0-9]*/[0-9]*\] CAUGHT' "$work/verdicts" || true)
survived=$(grep -c '^\[[0-9]*/[0-9]*\] SURVIVED' "$work/verdicts" || true)
noevidence=$((total - caught - survived))

# AND A HISTORY IS ONLY REPLACED BY A HISTORY. A run that measured nothing, or
# only part of the corpus, must not wipe the rates the last full run recorded.
if [ -z "$only" ] && cat "$work"/timings.* > "$work/timings.new" 2>/dev/null && [ -s "$work/timings.new" ]; then
	cp "$work/timings.new" "$timings"
fi

echo "" | tee -a "$out"
echo "mutant-verdicts: $total mutant(s) — $caught CAUGHT, $survived SURVIVED, $noevidence NO EVIDENCE" | tee -a "$out"
echo "  took $((t2 - t0)) s: the baseline $((t1 - t0)) s, the mutants $((t2 - t1)) s on $jobs workers" | tee -a "$out"
if [ -z "$only" ]; then
	echo "  per-mutant timings recorded in $(shown "$timings"); the next run estimates from them" | tee -a "$out"
fi
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
