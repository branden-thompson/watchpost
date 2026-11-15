---
title: "0.19.6 DISCOVER exit red team — BQ, business quality"
date: 2026-11-15
phase: DISCOVER
sev: SEV-0
status: "Filed whole as returned; dispositioned in ../red-team-review.md"
---

## 0.19.6 DISCOVER exit red team: Business Quality axis

**Verdict: DO NOT EXIT DISCOVER.** The package map is sound to plan from and needs no new survey. The brief is not ready: three of its metrics can be met while the principle they stand for is not, the case for the release cites a gate mechanism the code does not have, and the release's cost is stated nowhere. Fixing this takes one amendment to the brief, put to the HUM LEAD as a single ruling, plus four document fixes.

**What I checked against the code.** Package sizes: `modes/tty` is 31,346 lines in 88 files, `app` is 20,184 in 76, `platform/render` is 5,989. `Dashboard` has 752 methods in 68 files, 381 of them in the 31 `map_*.go` files. `livePipelines` has 108 methods in 21 files. `dashboard.go` is used by 71 non-test files. The post-update steps are at `modes/tty/dashboard.go:1156-1171`. The cited symbols exist. The baseline record shows 1,051 caught, 7 survived, 0 no evidence, in 3,976 s. All of these match the record.

Paths below are relative to `06_docs/02_features/0.19.6-re-architecting/` unless they start `06_docs/`, `app/`, `modes/` or `scripts/`.

---

### Q1. Does the work meet the written requirement? Is any metric a proxy?

**1.1 The problem statement says every mutant re-runs its package's whole suite. The code and the survey show it does not.**
- Where it is said: `01-objectives/project-brief.md:24-25` ("They dominate the gate. Every mutant re-runs its package's whole suite"). D-1's question says the same (`02-analysis/rulings.md:24`).
- What the code does: with a killer record, step 1 runs only the last killer (`06_docs/mutants/run.sh:41-42`, `:93`, with `-failfast`). PB-6 measured that 504 of tty's 539 mutants end at step 1, at about 7 s each, nearly all of it compile and link (`02-analysis/lens-PB-performance-baseline.md:275-280`).
- What the split buys: about 9 % off the sweep (`lens-PB…:323-324`). It "barely moves `make verify`" (PB-5, `lens-PB…:24-26`); `race` gets about 10 % shorter.
- A-6 corrected a wrong figure in D-6's question. Nothing corrects this premise, which sits in the brief the HUM LEAD approved (D-13).
- Severity: Important. Simplify/Delete: N.
- Action: amend the brief's fourth bullet to PB-5's and PB-6's figures, and record the correction as a row, as A-6 did. Anyone defending the release on gate time will be caught by `lens-PB` itself.

**1.2 M1 checks allocations only. The wall-time comparison the performance lens asked for was dropped, so M1 can pass while frames get slower.**
- Principle 1 says "Performance". M1 (`project-brief.md:46`) covers allocation pins, escapes, model sizes and inlined calls.
- PB-3 says the pins cannot see the main risk of the map seam: "an indirect call allocates nothing" (`lens-PB…:206-209`). PB recommends benchstat runs of the beat and frame benchmarks at the map batch and at BUILD exit (`:212-213`, row P-8 at `:408`).
- P-8 appears nowhere in the brief or the package map (no match for benchstat or wall time).
- The map's planned `Env` carries "the map hooks" (`lens-TS-tty-seams.md:444-447`). That is the func-field context PB-3 warns against (`lens-PB…:210-211`), and the brief has no rule against it.
- M1 also accepts fewer inlined calls "without a stated reason". That reason is the agent's own, against the standing rule that an exemption is ratified by the HUM LEAD.
- Severity: Important. Simplify/Delete: N.
- Action: add P-8 to M1 (no statistically significant regression, at the map batch and BUILD exit). Add PB-3's rule as a requirement: no func field read on the frame path. Make "a stated reason" a HUM LEAD ruling.

**1.3 Under 2b, M3's "map's reach into the model" is met automatically, and the guard that would stop that was not carried over.**
- M3 (`project-brief.md:48`) targets "each area's reach into the model (map 72…)".
- Once the map is its own package, the compiler forbids it from reaching `Dashboard` at all, so the count falls to zero by construction. The real coupling moves into `Env` and `mapview.Hooks` (`lens-TS…:444-447`, `:462`), which bring about 25 `Config` hooks (`:206`).
- JN P2-3 names this exact gaming and its guard: "count fields reached through the parameter" (`lens-JN-junior-navigation.md:322`). M5 lists P2-1, P2-4, P2-5, P2-7 and P2-9 to P2-11 as guards, but not P2-3's own guard.
- Severity: Important. Simplify/Delete: N.
- Action: M3 counts the fields reached through `Env` and the hooks as the map's reach, measured by type, as JN P2-2's guard also asks (`lens-JN…:321`).

**1.4 M4 (the blind trace) has no "before", so it cannot show that understandability improved.**
- M4 (`project-brief.md:49`) asks for 4 of 4 on the new layout.
- The only baseline is 0.19.5's 3 of 4, on different tasks (`lens-JN…:327`).
- JN's guard, a key built by leave-one-out before the docs are rewritten, is missing from M4.
- Severity: Important. Simplify/Delete: N.
- Action: at PLAN, run the same four tasks blind on `e57afeaf` and record the score. M4 is then a before-and-after comparison, and it carries JN's key-building guard.

**1.5 GP-2's "no new message hop" rule appears in no requirement and no instrument.**
- GP-2 shows that a new hop can stay green under the test harness's FIFO drain and still behave differently live. Its rule is no new `tea.Msg` and no new command where the code calls directly today, and a pin on the `*Msg` set (`lens-GP-go-practice.md:161-165`).
- The package map names the cost (`02-analysis/package-map.md:29`). R1 to R7 (`project-brief.md:56-68`) and batch 0 (`package-map.md:105-133`) omit it.
- M7's union-declaration check covers moves modulo a rename table (`lens-RX-regression-net.md`, RX-10). It does not cover 2a's prep, which adds new types.
- Severity: Important. Simplify/Delete: N.
- Action: add GP-2's rule as R8, and the `*Msg` pin as a batch-0 instrument.

**1.6 No metric is labelled for "maintainability" or "traceability".**
- Principle 2 names three properties. M3, M4 and M5 are all labelled "understandability" (`project-brief.md:48-50`).
- Severity: Minor. Simplify/Delete: N.
- Action: map each metric to the property it proves (for example, M5's docs-to-code checks and the atlas DAG for traceability), or say which property has no measure.

D-3 (Charm not re-implemented) is held by R4 and R5 (`project-brief.md:63-65`). That area is clean.

### Q2. What do the user and the maintainer lose, and is the cost stated?

**2.1 Neither the gate time nor the calendar time of the release is stated anywhere.**
- No match for calendar, days, weeks, hours or effort across the record.
- My estimate, from §6 (`package-map.md:153-158`): about eight code batches after batch 0, each two commits (R3). If every non-`.md` commit runs the full gate, which is the fast-docs-lane rule as I understand it (UNVERIFIED: I did not find it written in this tree), that is 16 or more `make verify` runs. At the 2,418 s median (`06_docs/verify-runs.md`; recent runs 2,061-4,003 s), that is about 11 hours or more.
- On top of that: batch 0's twelve instruments, the scoped sweeps, the partial-move demonstrations (C-11), the A-7 pilot, a 66-minute closing sweep, and hosted CI.
- What the user loses: nothing in the binary, if M7 holds. But the user-visible changes of 0.19.7 (D-2's audible changes, F-73's pause, MVS-D-74) now queue behind this release (`02-analysis/rulings.md:34`), and the record never says so.
- Severity: Important. Simplify/Delete: N.
- Action: the brief states the batch count, the expected gate hours and what 0.19.7 waits on, so the HUM LEAD approves a priced release.

**2.2 The root `modes/tty` stays the largest package in the program.**
- After the split, the root is about 16,900 lines (`package-map.md:46`), still about 2.8 times `platform/render`.
- `Dashboard` keeps 371 methods in 38 files (counted from the tree, `map_*.go` excluded). JN's own target is about 15 files (`lens-JN…:321`).
- The brief's first problem bullet ("too big to hold in one head", `project-brief.md:16-17`) therefore remains true for the root, and the brief does not say so.
- Severity: Minor. Simplify/Delete: N.
- Action: state the end-state figures next to the problem bullets.

### Q3. Is every cut or deferral recorded where the next reader will find it?

**3.1 Most of 0.19.7's deferrals exist only in a 0.19.6 A-row, not in the follow-ups file.**
- `06_docs/follow-ups.md:3-5` says a follow-up not in that file "is forgotten". The required reading says follow-ups live in `06_docs/follow-ups.md` only (`00-REQUIRED-READING.md:29`).
- Only F-73 (`06_docs/follow-ups.md:145`) and F-158 (`:90`) name 0.19.7.
- These appear in A-8 (`02-analysis/rulings.md:52`) and nowhere in follow-ups:
  - RM-6's five steps
  - RM's three questions
  - MVS-D-74
  - AS's Producer cut
  - C-9's Charm upgrade
- These, deferred elsewhere in the record, have no follow-up row either:
  - C-7's "lock of their own" (`package-map.md:93`)
  - C-8's return to the ×1.05 budgets (`:94`)
  - the Bubbles note in `docs/architecture.md:528` (`:187`)
- No 0.19.7 directory exists.
- Severity: Important. Simplify/Delete: N.
- Action: one follow-up row per deferred item, each naming 0.19.7.

**3.2 The required reading, which every session reads first, still says D-2 may come with this release.**
- `00-REQUIRED-READING.md:15-17` says the arbiter's dissolving "may come with it". `:40-41` says "D-2's work follows the role model…".
- D-11 rules "Nothing of D-2 in 0.19.6" (`02-analysis/rulings.md:34`).
- Severity: Important. Simplify/Delete: N.
- Action: rewrite both passages to match D-11.

**3.3 The package map says six of the agent's choices went to the HUM LEAD as A-rows. Five of those rows do not exist.**
- `package-map.md:176-177` says the alias rule, `internal/`, the lock decision, the comparator and the ratchet "are A-rows, put for veto with the brief". A-1 to A-8 (`02-analysis/rulings.md:45-52`) contain none of them, and D-13 (`:36`) ratified only "A-1 to A-8".
- The one that matters most is C-6 (`package-map.md:92`). It rejects JN's and TG's package-size ceiling (`lens-TG-tooling-gates.md:476`) and lets a size rise through "an A-row with its reason". That makes the agent the judge of D-1's own purpose.
- Severity: Important. Simplify/Delete: N.
- Action: write the missing rows. Put C-6 to the HUM LEAD singly, as a fork, with the ceiling as the option that was set aside. Make a ratchet rise a D-row.

**3.4 The package map's front matter and question list are stale.**
- Its status still reads "PROPOSED" (`package-map.md:7`), while the required reading calls it "the settled document" (`00-REQUIRED-READING.md:26`).
- §7's questions are numbered 5 to 9 (`package-map.md:168-174`).
- Severity: Minor. Simplify/Delete: Y.
- Action: set the status to what D-9 to D-13 ratified, and renumber.

**3.5 The baseline that every later judgement depends on is not committed.**
- `06_docs/mutants/verdicts/since-v0.19.5.log` is untracked (`git status`). A-3 says it was "promoted" (`02-analysis/rulings.md:47`); batch 0 item 2 commits it (`package-map.md:106`).
- The per-package split (57 % and 23 %) is only in `dist/mutant-verdicts.timings`, which is git-ignored (`.gitignore:1`) and rewritten by every full sweep (`scripts/quality/mutant-verdicts.sh:329`).
- Severity: Minor. Simplify/Delete: N.
- Action: commit the record now, and copy the per-package seconds into A-3 or the record.

### Q4. Is there a cheaper way to the same outcome?

**4.1 Map step 2b has no stop rule, and D-9 and C-3 contradict each other for exactly that case.**
- C-3 says "a seam that cannot be served without a test-only export is not taken in 0.19.6" (`package-map.md:89`).
- D-9 makes 2b certain and removed the checkpoint after 2a (`02-analysis/rulings.md:32`).
- 2b involves 122 straddling test files and 512 references (`lens-TS…:425`). Whole-model tests cannot move into the map package, because that would create an import cycle (`lens-RX…`, verdict). TS calls step 2 "a redesign, not a move" (`lens-TS…:24-26`).
- If 2b needs test-only exports, the record gives two contradictory answers.
- Severity: Important. Simplify/Delete: N.
- Action: recommend to the HUM LEAD (this does not reverse D-9) that a 2b needing any test-only export comes back as a ruling, with 2a's re-measured straddle count.

**4.2 The order spends most of the budget before the release's biggest unknown is sized.**
- 2a and 2b come last (`package-map.md:157`). 2a is "where principle 2's largest gain is" (`:55`), and its straddle count sizes 2b.
- Doing 2a straight after the console and kit would price the release early, and put the largest gain in place even if 2b stalls.
- Severity: Minor. Simplify/Delete: N.
- Action: put the order to the HUM LEAD at PLAN as an option.

**4.3 No option with fewer packages was evaluated for `app`.**
- AS compared Map I only with options that have more packages (`lens-AS-app-seams.md:309-317`).
- The deck boundary carries `app`'s compiler-enforced rule (`package-map.md:72-73`).
- `app/internal/kit` is 217 lines (`:67`), below JN's roughly 300-line floor, which needs a stated reason (`lens-JN…:320`). None is given.
- Severity: Minor. Simplify/Delete: Y.
- Action: at PLAN, record why `maps` and `kit` repay their cost, or keep `maps` in the root.

### Q5. Is the release's size proportionate, and is the measurement honest?

**5.1 The evidence for principle 2 comes from a tool that lives in a temporary directory outside the repository.**
- M3's baselines (72, 102, 82 %) and M5's test-only-export count come from A-4's tool. It stays "out of the tree" (`02-analysis/rulings.md:48`) and runs from the session's scratch directory (`<scratch>/`).
- M3 is to be "re-run on each batch's layout" (`project-brief.md:48`). If the scratch directory is wiped, neither the baseline nor any later figure can be reproduced, and a reviewer cannot check them.
- Severity: Important. Simplify/Delete: N.
- Action: commit the tool (tests-only, or under `scripts/`) and its baseline outputs in batch 0, or name a replacement instrument before PLAN.

**5.2 The gain is credible only as stated in the lenses.** About 51,500 lines move. The measured gains are:
- compiler-enforced boundaries for the deck, the console and the map;
- `Dashboard` going from 752 methods to 371;
- about 9 % off the sweep, with `verify` roughly flat (Q1.1).

M3 has no target figures yet (D-13's own counter-argument, `02-analysis/rulings.md:36`), and M4 has no baseline (Q1.4). Today the record cannot show that the gain justifies the cost; with Q1.3, Q1.4, Q2.1 and Q5.1 fixed, it could.

---

**Fix first: Q1.2.** As written, M1 can pass while frames get slower, through the planned `Env` hooks that the allocation pins cannot see. That breaks D-1's first principle, and the fix is cheap: carry P-8's benchstat runs and PB-3's no-func-field rule into the brief.

Nothing outside my scratch directory was written. Everything was run read-only.
