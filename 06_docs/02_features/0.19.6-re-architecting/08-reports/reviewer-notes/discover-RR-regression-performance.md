---
title: "0.19.6 DISCOVER exit red team — RR, regression and performance risk"
date: 2026-11-15
phase: DISCOVER
sev: SEV-0
status: "Filed whole as returned; dispositioned in ../red-team-review.md"
---

# 0.19.6 DISCOVER exit red team: regression and performance risk (RR)

Reviewed at `66d8a350`. The code is identical to `e57afeaf`. I worked read-only. The tree is unchanged: `git status` shows only the pre-existing untracked `since-v0.19.5.log`. Things I checked myself:
- **Anchors:** I executed all 1,058 mutants with writes neutered (`<scratch>/anchors2.py`). 692 anchor in the two packages, which confirms A-6.
- **Compiler output:** `go build -gcflags=-m ./modes/tty`. The one model box is at `router.go:463:9` and the `Init` boxes at `:320`. 421 inlined `Dashboard` calls sit in `map_*.go`. PB-2 and PB-3 are confirmed.
- **Allocation pins:** I ran them six times (`-count=6`).

## 1. Proof of no regression, by kind of behaviour

| Kind | Instrument in the record | Gap |
|---|---|---|
| Rendered frames | 63 goldens; the router trace's frame digest | Re-capture is not gated (F3) |
| Key handling, window stack | Router trace, widened (§4 item 6); whole-model tests | The trace keys on Go type names (F2) |
| Timing and ticks | `app/cadence_test.go` (fetch cadences only); ticks injected as messages | **None for UI tick arming.** The trace records only `cmd != nil` and never runs a command (`router_trace_test.go:60-63`) |
| Post-update chain order | none (F7) | GP-3's test-held list was never adopted |
| Message hops | none (F7) | GP-2's Msg-type pin was never adopted |
| Audio sequencing, ducking | The app trace (unspecified); existing duck tests; tone latency (not gated) | F8 |
| Persistence | `TestMain` isolation only (batch 0 item 4) | **None.** Nothing checks what `savePreference` writes as it moves to `app/internal/kit` |
| Concurrency, locks | `make race` over `./...` | **None for deadlock** (F9) |
| Error paths | none named | **None.** The trace feeds no failure; `mapPanicked` (`dashboard.go:1159`) is covered only by existing tests |

## 2. The instruments themselves

**F1 is under question 4 (Critical).** The findings for this question follow.

**F2. The router trace cannot stay byte-identical through the kit move. It also never runs a command.**
- The trace prints `%T` (`router_trace_test.go:61`). The golden holds the package-qualified name `tty.tickMsg` on 38 lines.
- TS-4 moves `tickMsg` into the kit (`lens-TS-tty-seams.md:167`). The golden must then change on a pure move. §5's "both traces byte-identical" and M7 (`project-brief.md:52`) cannot hold.
- The trace also reaches `r.broadcaster.now` (`router_trace_test.go:56`), an unexported field that only tests set (`broadcaster.go:141-143`, `:282`). So the console move must edit the instrument that judges it.
- Severity: Important. Simplify/Delete: N.
- Action: in batch 0's widening, key messages by their index in the alphabet, not `%T`. Set the clock through production API that survives the move. Record each step's command as a stable description, not a boolean.

**F3. "Goldens byte-identical" fails open.**
- `-update-golden`, `-update-cadences` and `-update-declset` rewrite their goldens (`golden_test.go:20`, `:36-41`; `app/cadence_test.go:21`; `app/declset_test.go:15`).
- No gate refuses a changed golden. A grep of `Makefile`, `scripts`, `tools` and `.github` finds none. A re-captured golden is green.
- RX-7's "`git diff -M` over `testdata` shows 100 % renames" (`lens-RX-regression-net.md:271`) never became a batch-0 item.
- Severity: Important. Simplify/Delete: N.
- Action: add a batch-0 gate that fails when any `testdata` golden differs from the baseline commit, except as a pure rename. Show it failing on a planted re-capture.

**F4. The proof that a move lost nothing checks names, not bodies, and "move only" cannot be done as defined.**
- `declset.Set` records "kind name" lines only (`platform/declset/declset.go:21-23`). A body edited in a move commit passes the union check.
- R3 says a move commit carries "`git mv`, package clauses and imports" only (`project-brief.md:60-62`; `package-map.md:144-146`). That is impossible. The console code uses 23 package-level root names, among them `centerText`, `tickMsg`, `SnapshotMsg`, `cursorSlot` and `mastheadSlot` (`<scratch>/` over the graph's `file-edges.tsv`). Each needs a qualifier added inside a function body.
- Severity: Important. Simplify/Delete: N.
- Action: add a union check over declaration plus body hash, with package qualifiers normalised. It would make a move commit fully mechanical. Restate R3 to allow qualifier edits and nothing else, held by that check.

**F5. The test inventory cannot see a test that starts skipping, or lost subtests.**
- It counts top-level names only (`package-map.md:116-118`).
- Several tests skip depending on their fixture or tree, and a move can flip them:
  - `memo_completeness_test.go:439` ("NOT covered"; this is TS-9's guard)
  - `broadcaster_uat_test.go:345`
  - `broadcaster_onair_tone_test.go:93`
  - `app/mapairuv_test.go:283`
- Severity: Important. Simplify/Delete: N.
- Action: record each test's `go test -json` outcome (PASS or SKIP), subtests included, in all three build modes. Compare outcomes, not just names.

**F6. The pin comparator (C-8) as worded will either flake or be loosened by hand.**
- C-8 says "any rise in a count fails" (`package-map.md:94`; `project-brief.md:46`). I measured the severe window's miss at 5,068 and 5,069 across six runs (`bench_test.go:258`). PB records the same flicker on the map and Propagation windows (`lens-PB-performance-baseline.md:96`).
- The frame and window pins count allocations only (`testing.AllocsPerRun`, `map_alloc_test.go:62-66`). They have no byte figure.
- Severity: Important. Simplify/Delete: N.
- Action: take the baseline as the maximum over N runs, recorded with N. Add bytes to the window pins. Make the comparator fail when a baseline pin produces no measurement (skipped, renamed or moved).

**F6b. Some performance checks are recorded, not gated, and one allows a self-issued exemption.**
- Batch 0 item 7 says the escape and size probes are "recorded" (`package-map.md:122-124`), while M1 says "no new model escape". Nothing fails.
- `-m` output is keyed by `file:line` (`router.go:463:9`), so a diff by position breaks the moment code moves.
- "No fewer inlined calls … without a stated reason" (`project-brief.md:46`) is a reason the agent issues itself.
- Severity: Important. Simplify/Delete: N.
- Action: gate escapes by function and type. Count inlining per function, not per file, because 2a re-homes functions across files. A reduction needs a HUM LEAD ruling.

**Item-by-item:**
- Items 1, 8, 9, 10 and 12: clean for my axis.
- Item 2 (the baseline record):
  - It is still untracked at HEAD (`git status`), although A-3 says it was "Promoted" (`rulings.md:47`).
  - Items 1 and 2 must land in that order: once A-2's fallback exists, nothing the sweep reads from HEAD carries the 0.19.5 record until it is committed (`mutant-verdicts.sh:128`, `:204`).
  - Minor. Simplify/Delete: N. Action: commit it first, in batch 0.
- Item 11, tone latency (`lens-PB-performance-baseline.md:366-395`):
  - "Then held" (`project-brief.md:46`) names no mechanism.
  - The benchmark is wall-time and never gated.
  - Its setup is white-box in `app` (`tone_latency_test.go:94-100`), so it breaks at the deck move.
  - Minor. Simplify/Delete: N. Action: hold its 185 allocations as a pin, and state where it lives after the move.

**M1 to M7:**
- M1: see F6, F6b and F10.
- M2: sound as "recorded".
- M3 to M6: outside my axis, except that M5's "zero test-only exports" collides with C-3 (F1).
- M7: depends on F1 to F5.

## 3. The mutant net

**F11. The re-vet can lose mutants, or weaken them, without the comparator seeing it.**
- **Scope.** `MUTANT_ONLY` matches mutant file names (`mutant-verdicts.sh:128-130`), and those do not contain target paths. "The batch's mutants" (C-12) is therefore a hand-built list, and a missed mutant is never judged.
- **Count.** Step 1's "80 mutants" (`package-map.md:51`) already differs from RX's 92 for the console cluster: `request.go` 8, `poolnote.go` 2, `relayfault.go` 2. 30 more target likely kit-host files (masthead, memo, cursor, tick); UNVERIFIED how many of those anchor on kit declarations.
- **Strength.** Mutants apply `s.replace(old, new)`, which replaces every occurrence. Nothing checks that an anchor is unique.
  - Today 3 in-scope mutants already match twice: `mC0`, `mC2` and `mU3` on `app/executors.go` (measured).
  - 2a re-types 381 methods, so anchors can start matching more sites (a stronger mutant, trivially killed) or fewer.
  - RX-12's comparator checks class and killer name only (`lens-RX-regression-net.md:412-414`). It cannot see a mutant re-pointed to a different site that is equally easy to kill.
- **Fail-closed parts that work.** A mutation that fails to compile reads INVALID, which counts as no evidence (`06_docs/mutants/run.sh:87`, `:95`). Killers are found by name anywhere in the tree (`mutant-verdicts.sh:159-161`).
- **Race.** No baseline catch is race-only, and none is "./... only" (grep of the log). But step 4 runs `-race` on the mutant's own package only (`:258`), so race-only detectors that stay at the top would be lost for moved code.
- Severity: Important. Simplify/Delete: N.
- Action:
  - Derive each batch's list from targets (`pkgof`) plus mutants whose last killer's file moved.
  - Have the comparator assert that it covered exactly that list.
  - Record each mutant's target file and occurrence count at baseline, and compare them.
  - Review each 2a text re-point as an old/new anchor table.

## 4. Tests across the new boundaries

**F1. C-3's rule cannot be met for step 1 or for 2b as written. Critical.**
- **The rule.** C-3 (`package-map.md:89`), D-9 (`rulings.md:32`) and M7 together say: assertions do not change, there is no test-only export, and whole-model tests cannot move below the model (RX-2).
- **Step 1, the console.** Router-level tests assert on the console's unexported fields:
  - `router_test.go:67`, `:77`, `:95`, `:109`
  - `router_bedplay_test.go:35`, `:40`, `:68`
  - `scroll_keys_test.go:18`
  - and 11 files in all.

  Production never reads several of these fields (`statusNote`, `selected`, `now`). After the move, each assertion needs either a test-only getter (forbidden) or a rewrite (forbidden).
- **2b, the map.** 63 test files make 586 references to `mapPane.` internals inside assertions, for example:
  - `map_events_test.go:36`, `:73`, `:90`
  - `map_feedflight_test.go:48`, `:110`
  - `map_clear_test.go:64`
- **"Setup only" is undefined.** Even 2a changes the identifiers on assertion lines.
- **What it means.** Read strictly, C-3 says "the seam is not taken", which contradicts D-9.
- Severity: Critical. Simplify/Delete: N.
- Action: put the rule to the HUM LEAD (my recommendation). A changed assertion must observe the same value through production API. Per test, it is proved by:
  - the same killers still killing its mutants, in the scoped sweep;
  - an outcome-equal inventory (F5).

  Per batch, a reviewer needs to see:
  - a table of every changed test line, classified as setup or assertion, each assertion line paired with the mutants it kills before and after;
  - an empty golden diff (F3);
  - the comparator's coverage (F11).

## 5. Performance

**F7. Two costs the record names have no rule or instrument holding them.**
- Package-map lines 28-29 name func-field contexts and message hops as the costs a cut forces.
- **Func fields.** TS-13's `Env` "carries … the map hooks" (`lens-TS-tty-seams.md:452-454`), against PB-3's rule that `Env` carries plain values only (`lens-PB-performance-baseline.md:207-209`). The disposition table has no C-row reconciling the two.
- **Message hops.** GP-2's rule (no new `tea.Msg` and no new command where code calls directly today, with the Msg-type set pinned; `lens-GP-go-practice.md:157-161`) is not in §4, §5 or M7.
- **The post-update chain.** §2 says the post-update steps "do not move" (`package-map.md:78`), but TS-13 moves the nine arms into `Model.After` (`lens-TS-tty-seams.md:459`). No test-held order list exists (GP-3).
- This matters because the test harness flattens batches into FIFO order (`dashboard_test.go:151-192`), so a new hop passes every suite.
- Severity: Important. Simplify/Delete: N.
- Action: add to batch 0 a Msg-type set pinned across the tree and a test-held chain-order list. Write down a map-package rule: `Env` holds no func field read on the frame path, enforced by a static check.

**F10. CPU, latency, startup and binary size are unmeasured in the binding metrics.**
- M1 and §5 hold allocations only (`project-brief.md:46`; `package-map.md:139`).
- The map seam's risk is CPU, not allocation. Copying a ~3.4 KB `Model` plus `Env` per call allocates nothing (`lens-TS-tty-seams.md:380-382`), and neither do lost inlining sites (PB-3).
- PB's wall-time, startup and size baselines exist (`lens-PB-performance-baseline.md:98-114`) but are in no metric.
- Severity: Important. Simplify/Delete: N.
- Action: add a benchstat comparison of beat and frame wall-time to M1 per code batch, paired back-to-back at `-count 10` and recorded. Make it mandatory at 2a and 2b. Record startup (`inittrace`) and binary size at open and close.

Clean: nothing in the map forces boxing or a `tea.Model` return, provided the Router holds the console and the map model concretely. TS-13's signatures do (`lens-TS-tty-seams.md:433-436`).

## 6. Order of work

**F12. 2b carries the most risk, and its net is weakest when it lands.**
- 2b moves 289 mutants (`package-map.md:35`), with 122 straddling files (`rulings.md:32`) and 586 internal references (F1).
- D-9 removed the checkpoint, so 2b's tests are rewritten in the same batch whose behaviour they prove (`lens-TS-tty-seams.md:479-488`).
- The rest of the order is sound: console first, deck before map (RX-3).
- Severity: Important. Simplify/Delete: N.
- Action: split 2b into three commits:
  1. Re-wire the straddlers in place, onto the API that will survive, with the code unmoved, proved by scoped sweep and outcome inventory.
  2. Re-measure.
  3. The move.

  Put to the HUM LEAD a measurable stop condition for 2b, for example a straddler that cannot be served without a test-only export.

**F13. F-158's owed test has no stated assertion or path, and it is born straddling the deck seam.**
- The row describes a hazard: `Restore` lifts the dip under a tornado read, and audio survives only through the arbiter (`06_docs/follow-ups.md:90`).
- A slice-1 test (`package-map.md:154`) passes on today's code, so it must pin today's behaviour. The record does not say which path (`WATCHPOST_MAINTRACK` on or off), or that it must not pin the hazard.
- It is white-box in `app` and is written before the deck move.
- Severity: Minor. Simplify/Delete: N.
- Action: state the assertion and the path in PLAN. Write it against the deck's planned entry point.

**F8. The app trace is unspecified and will straddle the deck seam.**
- §4 item 6 names it only as "the deck and the Director step on fixed seeds" (`package-map.md:119-121`). It names no entry point, no message set and no record format.
- Built in `app` against `newStation`, `newExecutors` and `testArbiter` (`app/station_test.go:51-76`), it breaks at the deck move. That is F2's flaw, in the batch with the most audio risk.
- Severity: Important. Simplify/Delete: N.
- Action: specify it in DISCOVER or PLAN against the deck's planned exported entry point (AS-4, AS-7), with stable keys and executed effects.

## 7. Concurrency

**F9. "No new lock, so nothing to prove" (C-7) overstates what stays the same.**
- The record names no lock rule and no deadlock instrument.
- Lock sites leaving `app` for other packages while sharing `lp.mu`:
  - the deck: `bedrelay.go`, 18 sites
  - the maps package: `history.go` 4, `mapwaves.go` 6
- Root code that already touches deck and maps state under the lock:
  - `commit` reads `lp.program` (a bed field) under `lp.mu` (`app/dashboard.go:1074-1116`).
  - `stopAll` records a deadlock where history writes take `lp.mu` (`:1028-1031`).
- Once those fields sit behind another package's API, an accessor that locks would self-deadlock under `commit`. Go's mutexes are not reentrant, and `-race` does not detect deadlocks. A hang surfaces only as the 30-minute timeout (`Makefile:35-38`).
- Lock order and goroutine ownership are otherwise unchanged, and `make race` runs `./...`, so race coverage travels.
- Severity: Important. Simplify/Delete: N.
- Action: in PLAN, list every `lp.mu` critical section that touches a moving field. Make it a rule that a cross-package method called under `lp.mu` is a `…Locked` form that never locks, held by a static check. Add a deadlock test with a short timeout over `commit` and `stopAll`.

## Verdict

**DO NOT EXIT DISCOVER.** The record is close. Most findings are batch-0 specification fixes that PLAN could absorb (F2 to F11). But F1 is a contradiction among C-3, D-9 and M7, the rules every batch is planned under. PLAN cannot write the console batch, the first code batch, without breaking one of them. That needs the HUM LEAD's ruling.

**Fix first: F1.** Restate "a pin's assertions do not change" as "a pin observes the same value through production API, with the same killers", put it to the HUM LEAD, and amend batch 0 with F2, F3 and F5 so that the restated rule can be checked mechanically.
