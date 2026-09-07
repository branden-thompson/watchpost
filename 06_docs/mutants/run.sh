#!/bin/bash
# run.sh — apply one mutant, decide whether the tests catch it, restore.
#
# A mutant is only evidence if it COMPILES. A mutation that breaks the build
# makes `go test` print "FAIL … [build failed]", which a naive grep for FAIL
# reads as a caught mutant — the instrument then reports coverage that is not
# there. Three false REDs reached commit messages that way before this script
# existed. So: assert the edit applied, build, and only then run.
#
#   ./run.sh <mutant.py> <packages…>
set -u
mutant=$1; shift
pkgs=${*:-./...}
root=$(git rev-parse --show-toplevel)
cd "$root" || exit 2

before=$(git status --porcelain)
if [ -n "$before" ]; then
	echo "SKIPPED $mutant — working tree dirty, a mutant needs a clean base"
	exit 2
fi

# ARMED ONLY ONCE THE TREE IS KNOWN CLEAN, so the restore can only ever revert
# what this script itself applied. Armed above the check, it fired on the
# SKIPPED path too — reverting the uncommitted work of whoever ran it, seconds
# after telling them nothing would be done. Cost real edits at 0.14.0 T2.2.
#
# AND IT RESTORES ONLY WHAT THE MUTANT TOUCHED, recorded at the moment it is
# applied. It used to revert `git diff --name-only` — every modified file in the
# tree — which is only safe while nothing else changes the tree. A clean START is
# not a clean WHOLE RUN: a run takes minutes, and anything edited meanwhile was
# reverted on exit with no warning and nothing to recover from. Cost ten
# uncommitted files at 0.14.0 T3.10b. The clean-tree check above stays, because
# a mutant still needs a clean base to be evidence.
mutated=""
restore() { [ -n "$mutated" ] && git checkout -- $mutated 2>/dev/null; return 0; }
trap restore EXIT

# A mutant is only evidence against a GREEN baseline. A tree that is already
# failing — a stale declset, a flaky timing test — makes every mutant look
# caught, which is the same false-coverage bug in a different costume.
if ! go test $pkgs -count=1 >/dev/null 2>&1; then
	echo "SKIPPED $mutant — the unmutated tree is not green; fix that first"
	exit 2
fi

python3 "$mutant" || { echo "UNAPPLIED $mutant — the mutation did not match"; exit 2; }
mutated=$(git diff --name-only)   # what to put back, and the ONLY thing to put back
if [ -z "$mutated" ]; then
	echo "UNAPPLIED $mutant — no file changed"
	exit 2
fi
gofmt -w $(echo "$mutated" | grep '\.go$') 2>/dev/null

if ! go build ./... 2>/dev/null || ! go vet $pkgs >/dev/null 2>&1; then
	echo "INVALID  $mutant — does not compile; not evidence either way"
	exit 2
fi

out=$(go test $pkgs -count=1 2>&1); code=$?
if echo "$out" | grep -q "build failed"; then
	echo "INVALID  $mutant — test build failed"
	exit 2
fi
if echo "$out" | grep -qE "^--- FAIL"; then
	line=$(echo "$out" | grep -E '^--- FAIL' | head -1 | sed 's/^--- FAIL: //')
	name=${line%% *}  # drop the "(0.00s)"
	top=${name%%/*}   # a subtest's parent is what -run needs

	# ATTRIBUTION: THE FAILURE MUST BE THE MUTATION'S (2026-09-06).
	#
	# The green-baseline gate above samples the unmutated tree ONCE, so a test
	# that fails 1-in-N passes the baseline and then fails on the mutated run
	# for its own reasons — and this line reported it as evidence about the
	# rule. That happened: mH0 was reported CAUGHT by a panicking-executor test
	# while the machine was loaded, and mH0 mutates a remainder guard on a hold.
	# The rule was in fact UNPINNED, and the false verdict is what hid it.
	#
	# So: restore, and ask whether that same test fails WITHOUT the mutation. A
	# genuine catch cannot. The re-run is one test, and only on a catch, so the
	# cost is a rounding error against the full run above.
	#
	# INVALID rather than a new word, because that is exactly what INVALID
	# already means here — not evidence either way — and every driver that reads
	# these verdicts already knows it.
	git checkout -- $mutated 2>/dev/null
	mutated=""
	if ! go test $pkgs -count=1 -run "^${top}$" >/dev/null 2>&1; then
		echo "INVALID  $mutant — $name fails on the UNMUTATED tree too; the failure is not attributable to this mutation"
		exit 2
	fi
	echo "CAUGHT   $mutant — $line"
	exit 0
fi
# A MUTANT THAT CRASHES THE BINARY IS CAUGHT, NOT SURVIVED. A panic on a
# goroutine takes the test process with it, so no test ever reports "--- FAIL"
# and the grep above finds nothing — which read as "nothing fails" and
# understated coverage. It is the inverse of the false RED this script was
# written to prevent, and it was found by mB3 (the pump's recover deleted),
# where the suite quite plainly did not survive the mutation. The exit code is
# the honest signal: go build and go vet have already gated a bad tree above,
# so a non-zero test run here is the mutation being detected.
# A KNOWN REMAINING GAP: this path has NO ATTRIBUTION CHECK. A crash names no
# test, so the targeted re-run the --- FAIL path uses is not available, and
# confirming it would mean a second FULL package run. A flaky panic under load
# would therefore still be credited to the mutation, which is the same hole that
# hid mH0 on the other path. It is rarer — a panic is not a timing assertion —
# and the cost is real, so it is stated here rather than closed. If a crash
# verdict ever looks surprising, re-run it on a quiet machine before believing it.
if [ $code -ne 0 ]; then
	echo "CAUGHT   $mutant — the test binary did not survive it: $(echo "$out" | grep -m1 -E '^(panic|fatal error|SIGSEGV)' || echo "go test exited $code with no test-level failure")"
	exit 0
fi
echo "SURVIVED $mutant — NOTHING FAILS. This behaviour is unpinned."
exit 1
