---
title: "The corpus sweep: how it is launched, and the three things that made launching it wrong possible"
date: 2026-09-14
phase: BUILD
sev: SEV-0
authority: HUM LEAD
status: "ALL THREE BUILT AND APPROVED 2026-09-14."
---

# How `make mutant-verdicts` should have been launched — and why it could be launched wrong at all

## What I did

```
make mutant-verdicts 2>&1 | tail -40      # run in background
```

**`tail` buffers to EOF.**  So for 130 minutes the background task's output file was EMPTY, there was
no way to see progress from it, and the HUM LEAD had to ask where we were.

## What it should have been

```
make mutant-verdicts                       # run in background, UNPIPED
```

The script already `tee`s every verdict to `dist/mutant-verdicts.log` as it goes — that tee is the
only reason the question could be answered at all.  Unpiped, the task output streams too and either
source answers "where are we".

**THE RULE, GENERALLY: never pipe a long-running background command through a filter that buffers.**
`tail`, `sort`, `wc` and `grep` without `--line-buffered` all hold everything until the producer
exits, which converts a progress stream into a single result at the end.  Filter when READING the
log, never between the producer and the file.

## The second failure, which is the worse one

**My ETA was ~95 minutes and the real figure is ~4 hours — wrong by more than 2x.**

I derived ~33 s per `app` mutant from a batch of **18**, and extrapolated it to **99**.  That sample
was the release's NEWEST mutants: their detectors fail early, so `go test` exits early.  The corpus
average is much higher because a mutant caught by a test late in a ~100 s suite costs the whole suite.

**I sized 314 units of work from a sample chosen for being recent rather than representative**, and
the standing instruction — cost out batch work BEFORE launching, and never let the HUM LEAD be the one
to notice hours-long work — is exactly what that violates.  I also did not check the estimate against
reality at any point; the 60-minute mark would have shown a 2x overrun with 40 minutes of it left to
spend rather than 130.

## The deterministic fix, so neither can happen again

Both failures are *conventions I broke*, and a convention is not a guard.  Four changes make the
outcome structural:

1. **Every verdict line carries its position** — `[170/314] CAUGHT …`.  Any read of the log, by any
   means, says where the run is.  No counting, no separate status file, nothing to remember.
2. **The script prints a banner before the first mutant**: the corpus size, the log path, and an ETA.
   A wrong launch is then visible in the log itself within seconds of starting.
3. **The ETA is MEASURED, NOT GUESSED.**  The script records elapsed time per mutant and writes a
   per-package summary at the end; the next run reads that file to estimate.  The first run says
   "no timing history — estimate unavailable", which is honest, and every run after is derived from
   what this corpus actually did on this machine.  **The estimate stops being a judgement call, which
   is the thing that failed.**
4. **The Makefile comment states the launch**: unpiped, background, watch the log.  The recipe already
   passes the log path explicitly, so the one thing a caller could get wrong is the pipe — and (1)
   and (2) make even that self-correcting, because the log is still complete and still positioned.

**Not proposed: refusing to run when stdout is not a terminal.**  CI pipes deliberately, and a gate
that fails on how it was invoked rather than on what it measured is a gate that gets worked around.

---

## Addendum, measured at the 190-minute mark

**The real rates, solved from two observed windows rather than sampled:**

| target | seconds per mutant |
|---|---|
| `app` | **203** |
| everything else | **19** |

A 10.7x spread, and it is structural: `./app`'s suite is ~100 s and `run.sh` runs it TWICE per
mutant — once on the clean tree for the baseline, once mutated.

**THE FIRST MEASUREMENT WAS RIGHT AND I DISCARDED IT.**  The very first `app` mutant timed took
**3 m 21 s** = 201 s, within 1% of the 203 s solved for here.  I then ran an 18-mutant batch, saw a
~33 s average, and used that — the 18 were the release's NEWEST mutants, whose detectors fail early.
**I replaced a direct measurement with a convenient average and did not notice they disagreed by 6x.**
That is worse than sampling error: the contradicting evidence was already in hand.

**The rule:** when a new aggregate disagrees with a direct measurement by an order of magnitude,
the disagreement is the finding.  Reconcile it before using either.

## The second thing the sweep exposed: it does twice the work it needs to

`run.sh` establishes the clean baseline **per mutant**, so 56 `app` mutants cost 112 runs of a ~100 s
suite.  On a sweep the tree is clean and identical before every one of them — **the baseline is the
same answer 56 times.**

Running it ONCE per package and reusing it would take a full sweep from ~6.6 h to roughly half that.
It does not weaken the guard: `run.sh` keeps its own per-mutant baseline for single use (where the
tree's cleanliness is exactly what is in question); the SWEEP is the caller that knows it is holding
one clean tree across the whole run, and it is the caller that should say so.

**Whether that is worth building is the HUM LEAD's call**, and it is the more valuable half of this
note — the ETA work stops a wrong estimate, this stops half the cost.


---

## What was built, 2026-09-14 — all three approved by the HUM LEAD

| | |
|---|---|
| **One baseline per package** |  skips the clean-tree TEST RUN and nothing else.  The clean-tree CHECK is not skippable, which is the whole safety argument — a mutant that failed to restore is refused before the baseline question arises.  Two controls watch that: , and its partner proving the knob still reaches a verdict rather than making the sweep fast and worthless.  ~6.5 h → ~3.5 h |
| ** on escalation** | Only survivors pay.  's detector measures 20/20 under  and ~81/100 without, so the sweep read it on a bad day and reported an unmeasured rule.  A survivor that needs the race detector is not a survivor |
| **A measured ETA, and position counters** | Every run records seconds-per-mutant per package; the next run reads them.  With no history it SAYS SO rather than inventing a number.  Every line carries  |

## And the two defects the smoke tests found in the fix itself

**1. It wrote a green wall.**  Five mutants SKIPPED on a dirty tree, "0 CAUGHT, 0 SURVIVED", **exit 0**.
SKIPPED, UNAPPLIED and INVALID are not verdicts — each means the mutation was never measured — and a
sweep that counts them as nothing reports success for having run nothing.  It fails on them now, with
the count in the summary line.

**2. It timed what did not happen.**  Every skipped mutant recorded a zero-second timing, and the next
run estimated a three-hour corpus at "~0 min".  **Timing a non-event is worse than having no history,
because it LOOKS like history** — which defeats the entire purpose of the change.

**THE SECOND ONE IS THE LESSON.**  The instrument built to stop a wrong estimate produced a wrong
estimate on its first run, and neither defect was found by reasoning about the script.  Both were
found by running it on a real corpus and reading what it said.
