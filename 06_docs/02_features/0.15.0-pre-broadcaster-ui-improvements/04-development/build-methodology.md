---
title: "0.15.0 — BUILD methodology and hand-off"
date: 2026-09-07
phase: BUILD
sev: SEV-0
status: "Live working document — read this first after a context compaction"
---

# BUILD methodology and hand-off

**Read this first after a compaction.**  It carries the procedures that produced good results in
B1–B3 and the ones that wasted time, so they do not have to be rediscovered.  The *rules* about what
makes a gate valid live in `06_docs/quality-observations.md`; this file is about **how to work**.

## Where we are

| | |
|---|---|
| Branch | `feature/0.15.0-pre-broadcaster-ui-improvements`, nothing pushed |
| **B1** | **DONE** — survey (bucket 1 = 4, not 49) · FR-2.2 coupling derived from `severeEvents()` · #18 evacuation tone (3× dual tone) audible · `platform/closedset` extracted at the third caller |
| **B2** | **DONE** — FR-1.1 `config.Mutate` (6 writers → 1) · FR-1.3 band gate · `platform/singleowner` extracted at the second caller · FR-1.4 answered *no lock*, by experiment |
| **B3** | **IN PROGRESS** — both entry conditions discharged: `test-tags` in `verify`; `scripts/lint-injector.sh` as a `release-matrix` post-step |
| Gates | Last full run green: `VERIFY=0 ALLOC=0 PLATFORMS=0` at B2 exit |

**Immediately next:** FR-5.  The measurement is done (below); the fix is `modalDebug`'s scroll, then
land the reachability property as a baseline-and-ratchet.

**Then:** FR-4 (scenario payloads, 2-minute expiry, per-line marking, pre-emption rule), then FR-6.4.

**FR-6.4 IS NOT BLOCKING — HUM LEAD ruling, 2026-09-07.**  *"Watchpost is not a web application, so
recommendations are useful, but never blocking functionality, at least until I have data or users
that insist otherwise."*  WCAG 2.2.1 is **aspirational** here.  OQ-2 is therefore not a gate on the
batch: improve the relay-fault timeout if it is cheap and clearly better, and do not hold work or
raise conformance as a defect.  Revisit only on evidence — a real user, or data.

### FR-5's measurement, so it is not repeated

At 80×24, driving the **real key path** (`model.Update(tea.KeyPressMsg{…})`, 80×Down + 8×PgDn + End
+ 8×Right), comparing content against an 80×200 render with footer chrome excluded:

| modal | unreachable | reading |
|---|---|---|
| **debug** | **9** | the real defect — includes the `›` scenario cursor |
| setup | 4 | ambiguous; group headers, may need a driver not yet found |
| help | 5 | ambiguous; wrapped fragments |
| relay-fault | 2 | ambiguous; its own test says `Fall-Thru` *is* reachable |
| details · add · remove · alerts · status · about · severe | **0** | clean |

**Decision:** fix `debug`; land the property as a **baseline-and-ratchet** with the four ambiguous
counts pinned and named for follow-up.  Do not silently declare them fine, and do not block FR-5 on
investigating all four.

## What worked — keep doing these

**1. A watched RED, never a compile error.**  When the first attempt at a test fails to *build*, add
stubs returning the wrong answer so the RED is an assertion with a message.  A build failure proves
the code does not compile; it does not prove the test can fail.  *(Used for #18's mapping and for
`ClassFor`/`ToneRepeats`.)*

**2. Validate every gate in BOTH directions, against a representative fixture.**  Fires on a planted
violation; passes on a clean subject.  `lint-injector` passed its self-test while blind, because the
fixture was unstripped and every shipped artifact is `-s -w`.

**3. Measure before concluding, and measure again after correcting.**  Every red-team recommendation
used this build was a hypothesis: two were wrong on inspection.  My own correction to one of them was
also wrong.  *A correction is a new claim.*

**4. Hand-roll first; extract at the second or third REAL caller.**  `closedset` designed at PLAN had
the wrong shape and a call-site count of five against a reality of one.  Written by hand three times,
the true skeleton was obvious in a minute.  `singleowner` was extracted from two working
implementations and was right first time.

**5. Run the full gate set on a STILL tree.**  Never edit while `make verify` runs — the mutant
harness patches source files and your edits move them underneath it.  Three mutants reported
UNMEASURED that were fine.  Background the gates, do **read-only** work, and wait.

**6. Commit messages carry the evidence, not just the change.**  Each records what was watched
failing and what the numbers were.  They are the audit trail when a claim is later questioned.

**7. Correct the PLAN when measurement disproves it.**  B3's entry said "symbol-level, not
string-level"; measurement reversed it, and the plan was amended the same commit.  A plan carrying a
claim its own author has disproved is the defect this release is about.

**8. Delete temporary probes; keep instruments that pin a count.**  `classifier_crosstable_test.go`
stayed because it pins the divergence at 7.  The FR-5 probe was deleted once its measurement was
recorded here.

## What did not work — do not repeat these

**1. Grep/regex analysis of my own code to answer structural questions.**  Wrong three times in one
session (feature-detection of test shapes; the `modalSetup` routing; the "modals tested at ≤24"
survey).  **Read the code, or drive it.**

**2. Under-driven probes.**  8 `nav-down`s to scroll 39 lines at 12 per screen reported 23
unreachable lines that were simply not scrolled to.  With 80 presses it was 5.  **Exhaust the input
before reporting a limit.**

**3. Driving through the wrong seam.**  `handleNav` does not route `modalSetup` at all, so the probe
moved the dashboard's selection instead and reported 32 unreachable lines.  Through the real key
path it was 4.  **Check what the seam actually routes before trusting what it returns.**

**4. `git checkout --` to revert an untracked file.**  It silently does nothing; a temporary edit
survived into a test run and produced a false failure.  Use `cp` to a backup, or `python` to reverse
the exact edit.

**5. Backticks inside a double-quoted `git commit -m`.**  zsh runs them as command substitution.
Always `git commit -F -` with a **quoted** heredoc (`<<'MSG'`).

**6. Assuming a red-team finding is actionable as written.**  Both lenses that were wrong were right
that *something* was wrong.  Take the finding; re-derive the fix.

## Standing constraints that apply to every commit here

- **No AI attribution** anywhere — enforced by `lint-watermark`.
- **Tier B at PLAN only**; BUILD writes real code, and no plan gains implementation code.
- **P10 exemptions are presented for ratification**, never self-approved.
- **UX rulings are the HUM LEAD's.**  OQ-2 is ruled: accessibility guidance is advisory here, not a gate.
- **Every batch pushes on first commit** (NFR-4) — not yet done for this branch; still local.
- **UAT before REVIEW exit**, two questions: regression *and* yield.  See the implementation plan.
