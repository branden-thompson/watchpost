---
title: "0.16.0 — SPIKE charter: the audio merge"
date: 2026-09-09
phase: PLAN
sev: SEV-0
authority: HUM LEAD
status: "CHARTERED by D-25.  Runs BEFORE P0."
---

# SPIKE — the audio merge

**Chartered, timeboxed, isolated, and disposed of on a stated condition.**  An unrun spike and an
unbounded spike are the two anti-patterns; this charter exists so this is neither.

## Why it runs first

**HUM LEAD, D-25:** *"Spike the merge first - again we should have most of this infra built - the main
thing is going to be the operator controls."*

The plan had the merge fourth of eight, behind three UI batches.  The unknown it carries lives entirely
in `platform/lineup` and `domains/radio/player` — **no router and no console are needed to answer it**,
so three batches of unrelated work would have been sunk before its cost was known.

## The question, and it is ONE question

**Can the rotation's reads travel the card path — `Propose → BuildCard → Speak` through the narrator
arbiter — instead of `Tune → startSynth → engine.StartSource`, without a window in which both paths
can speak?**

Everything else about P3 follows from the answer.  **If the answer is yes, the merge is wiring and the
HUM LEAD's expectation holds.  If no, the release's shape changes and it is better to know now.**

## Timebox

**One working session.**  If the question is not answered in that session, the spike ends and reports
*"not answered in the box"* — which is itself a finding, and a more useful one than a spike that runs
until it succeeds.

## Isolation

Runs in **its own git worktree**, never the working tree.  Nothing it produces is committed to the
feature branch.  Red-team probes and spikes that mutate a shared tree produce anomalies other work
then has to reconcile.

## The output — exactly three things

1. **A yes or no**, with the evidence that decided it.
2. **The double-speak window characterised**: does it exist, how wide is it, and what closes it.
   `mark`/`readAloud` dedupes alerts by id for the rail only and is never consulted by the rotation's
   path, so "nothing dedupes" is the starting hypothesis, not the conclusion.
3. **A revised P3 estimate**, stated against what the spike actually touched.

## Disposal

**The spike's code is deleted.**  What survives is the three outputs above, written into
`04-development/`.  If a probe proved load-bearing it is re-authored as a real test in P3 rather than
promoted from spike code — the standing rule is that temporary probes are deleted and instruments that
pin a count are kept.

## What it must NOT do

- **Not** build the console, the router, or any operator control.  Those are the plan's other batches
  and the HUM LEAD has already named them as the real work.
- **Not** ship behind a flag.  A spike that ships is not a spike.
- **Not** grow into P3.  P3 is re-planned from the spike's answer.
