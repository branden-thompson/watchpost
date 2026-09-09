---
title: "0.15.0 pre-Broadcaster UI Improvements — DEBRIEF (After Action Report)"
date: 2026-09-09
phase: REFLECT
sev: SEV-0
authority: HUM LEAD
status: "Shipped — v0.15.0 tagged on fd761ab, release run 34352664330 green, 8 assets published"
---

# 0.15.0 pre-Broadcaster UI Improvements — DEBRIEF

**Problem (locked at DISCOVER, 2026-09-07):** *Watchpost's own surfaces cannot be checked on demand,
and its rules have no single owner — so a defect that is invisible to the operator survives review,
and a rule written twice can be right in one copy and wrong in the other.*

**Shipped 2026-09-09.**  Solved on both halves: the station can be tested on demand, and every rule
that had two implementations now has one, enforced by a gate rather than by intention.

---

## 1. What was delivered

**Four defects a listener would have met**, each of which had survived prior releases because nothing
could see it: a location the weather service cannot serve shimmering for ever instead of saying `n/a`
(#13, three attempts); Settings writing to disk and re-opening on the old values, so a change could
not be undone from the window; `20` typed over a stored `50` becoming a five-thousand-mile radius,
saved silently; and a spoken fire report running two rings together, so a "no hotspots" line was
followed by a list of fires with nothing saying which ring they belonged to.

**The instruments, which were the larger half.**  Every gate now carries a **watched failure** — a
plant applied, compiled, and seen to turn it red.  A duplicate detector was built, calibrated and
gated.  The mutant corpus went to **172**.  `-trimpath` was set on every target, taking 473
build-path strings per binary to zero, **confirmed on the published artifact** rather than locally.

**A stated coverage boundary**, which is a deliverable and not a caveat: there is no geostationary
near-real-time fire detection path, and an evacuation ordered by a state or county agency is not an
NWS product.  Both were previously implied.  Both are now written where a user reads them.

## 2. What went well

- **Independent review earned its place as a gate rather than a courtesy, and there are numbers for
  it.**  The author's own BUILD-exit review found 5 findings and 0 blockers.  Three blind agents found
  ~20 and **3 blockers**.  A round scoped to the remediation found **2 regressions that the author's
  own fixes had introduced**.  Every one of those rounds ran against a tree that had passed
  `make verify`.
- **Instrument validation (INST-1..INST-5) paid for itself repeatedly, including on itself.**  Five
  plants across the release were bad plants rather than surviving defects: an allocation the compiler
  deleted, a guard whose removal was a compile error, a probe that bypassed the boundary it was
  testing, an `Update` whose returned model was discarded, and a malformed edit that never applied.
  **INST-3 caught a sixth at SHIP**, on a checker rather than a mutant — a PR-template control that
  "survived" had used the table's header text instead of the template's actual placeholder row.
- **Two gates were found unable to fail at all.**  `gate-controls` — the gate whose entire job is to
  prove the others still fire — was green while running no control, because two scripts treated an
  unrecognised flag as "run normally, exit 0".
- **The HUM LEAD's rulings consistently produced better outcomes than the plan.**  Pulling Metric D
  into the release ("build it now when the surface area is smallest") is the clearest: the detector
  found 10 duplicate groups while the surface was small enough to collapse 9 of them.

## 3. What to carry forward

- **An assertion whose subject is a WINDOW must hold the window open.**  A test asserting a second
  read was inert *while one was running* never held the first one open, so it passed on scheduling.
  It is the concurrent sibling of this release's other repeat — *a test that constructs the value
  under test cannot test where the value comes from*, seen **three times** — and both fail the same
  way: by passing.
- **A green gate on one platform is not a green gate.**  Every local gate runs on macOS.  "ALL GATES
  GREEN" was true and incomplete through BUILD exit, REVIEW and VALIDATE while the branch's Linux CI
  was red.
- **A negation in prose is invisible to a keyword parser.**  "Does **not** close #12" closed #12.
- **Read the whole document, not the greps.**  The README content audit found three wrong claims that
  six targeted greps had missed, including a severe-window description omitting **Emergency Orders**
  — the category carrying evacuation orders, which this same release had established is the most
  safety-relevant thing that window shows.  Each grep found what it went looking for; none asked
  whether everything present still described the app.
- **Publish the measurement, not the number.**  The exposure statement's own instruction — *re-derive
  rather than cite* — was vindicated twice: the counts moved between writing and shipping, and F-64's
  "~5x" was stale within one CI round of being written.

## 4. Process review

**The three findings that mattered most all arrived AFTER every phase had reported green.**  The
branch's CI was red and unread (F-63); the mutant corpus had crossed `go test`'s default timeout, so
the same commit was green and red four minutes apart (F-64); and an issue was closed by a sentence
saying it should stay open.  None of them were code the reviews missed.  All three were on the
**outward** surface — CI, the build's own clock, and the publish — which no exit gate looked at.

**That is this release's process finding, and it is a gap in the exit sequence rather than in the
work.**  `06_docs/required-gates.txt` lists what an exit is checked against and every entry is a local
gate.  The remedy is small and named in F-63: read the branch's last CI conclusion at each exit,
citing the run id and its platform legs the way every other gate cites its evidence.

**Where I got it wrong, plainly.**  Two of the three regressions in this release were introduced by my
own remediation.  The `mK8` mutant I added at SHIP is what pushed the corpus over the timeout.  The
sentence that closed #12 was mine, and its whole purpose was to prevent that.  Independent review
caught the first; the clock caught the second; a post-merge check I nearly skipped caught the third.

**What the plan got wrong, and what that is worth.**  Four deviations were recorded rather than
quietly absorbed: FR-6.6's timebox was never needed, FR-7.4 was ruled the *opposite* way once its
premise was measured (1,135 occurrences against an assumed ~200, 94% resolving), Metric D was pulled
in mid-BUILD, and the `--ascii` and AA exit conditions were both worse than an already-pessimistic
plan predicted.  A plan that survives contact unchanged usually means nobody measured it.

## 5. Follow-ups carried

**F-58** `journey` red by ruling · **F-59** 17 duplicate groups in test code, not gated · **F-60** the
voice-install cap ruling · **F-61** an age bound on the spoken fire read · **F-62** `Within [0] mi`
where 0 means All · **F-63** no exit reads CI state · **F-64** the corpus has no runtime budget.

**Also open:** #12 for the data-cache half of the memo-key audit (the frame path is closed and
guarded), the P10-08 exemption for `app/inject_release.go` (presented, not ratified), the demo
location in 67 fixture files, and harmonizing the repo's PR template with the canonical one.

**Owed to A2DH**, per HUM LEAD direction: P10 linked as a skill; **INST-1..INST-5 folded formally into
the FULL TDD directive** as another dimension of it; and the independence finding fed back into the
existing red-team skill, which this release validated again.

## 6. By the numbers

| | |
|---|---|
| Span | 3 days, DISCOVER 2026-09-07 → SHIP 2026-09-09 |
| Commits | 118, squashed to 1 on `main` |
| Changed | 211 files, +13,566 / −1,068 — of which **9,378 lines are code** and **4,188 documentation** |
| Mutants | 171 → **172**, 32 target files, zero without an `assert` |
| Gates | 14 local, every one with a watched failure; 2 CI-only |
| Metric D | 10 duplicate groups → **0 unexplained** (9 collapsed, 1 ratified) |
| Exposure | `-trimpath` 473 → **0** per binary, verified on the published artifact |
| Red-team rounds | 5, blind, across 4 phases — 5 / ~20 / 15 / 6 / 4 findings |
| Blockers by round | 0 / **3** / 0 / **1** / 0 |
| CI rounds at SHIP | 3, **every failure a real finding** |

**The curve is the point.**  Rounds finding pre-existing defects gave way to a round finding only the
author's own remediation, then to a clean sweep — and that is legible only because every round was
counted, including the ones that made the author look worse.

## Source documents

`08-reports/`: `project-brief.md` · `discover-report.md` · `plan-report.md` · `build-report.md` ·
`review-report.md` · `validate-report.md` · `red-team-discover.md` · `red-team-plan.md`
`07-readiness/`: `release-checklist.md` · `gates.md` · `exposure-statement.md` · `pr-body.md`
Repo-level: `06_docs/quality-observations.md` · `06_docs/quality-plan.md` ·
`06_docs/build-methodology.md` · `06_docs/follow-ups.md` · `06_docs/required-gates.txt`
