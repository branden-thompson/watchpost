# Plan red team, RR: regression and performance risk (0.19.6 PLAN, HEAD `fbbfeb1a`)

`F/` stands for `06_docs/02_features/0.19.6-re-architecting/`. I checked every code claim on the tree, whose `.go` files match `e57afeaf`. I wrote nothing in the tree.

## 1. The net per batch

**RR-1. The router trace never builds a map, so B4 (2a) and B8 (2b) have no whole-behaviour trace.**
- The trace's Dashboard is `benchDash`, which is `NewDashboard(Config{Version: "bench"})` (`modes/tty/bench_test.go:89`, used at `modes/tty/router_trace_test.go:50`). That config has no `NewMap`, `MapFeed`, `SetUI` or `PropagationUpdate`.
- The trace never runs a command (`router_trace_test.go:56-61`). CK-6 describes every watchpost closure as `fn` (`F/03-architecture-design/design-CK-console-kit.md:605-618`).
- RX-8 asked for the trace to be widened to reach the map, Propagation and Settings (`F/02-analysis/lens-RX-regression-net.md:291-292`). Plan B0.6 (`F/04-development/implementation-plan.md:118-121`) widens only the alphabet, the key, the clock and the command form.
- So in the batch with the most change (387 methods re-homed, 11 split sites changing control flow, `Want`/`Env`), the "byte-identical trace" in M7 records frames with no map in them. B4 falls back to the map unit tests, whose 798 call references are rewritten in that same batch (`design-MS-map-seam.md:515`), plus the mutants and the PTY journey.
- Severity: **Critical**. Simplify/Delete: N.
- Action: B0.6 builds the trace's Dashboard the way `map_golden_test.go:107` and `map_alloc_test.go:30` do: `embeddedMap`, offline feeds, a recording `SetUI`, a Propagation seam. It runs each returned command, bounded and synchronous, and feeds the messages back. A watchpost command is recorded by the type of the message it returns. Recorded on `e57afeaf`-identical code, as today.

**RR-2. A-11's derived re-vet list is defined only by files that move, so in-place batches have no list.**
- A-11 says: mutants whose target is in "a moving file", or whose last killer's file moves (`F/02-analysis/rulings.md:68`).
- B4 moves no file. Every prep commit moves none. That is exactly where the tests are rewritten under D-14 and D-22: B3's 128 references, B4's 798, B8.1's re-wiring.
- The plan says B4's scoped sweep follows the 146 re-pointed anchors (`implementation-plan.md:210-211`) but never derives its list.
- A mutant whose target file stays put and whose killer had its call rewritten is never re-vetted. That includes the other 143 map mutants and any root mutant killed by a rewritten map test.
- Severity: Important. Simplify/Delete: N.
- Action: extend the derivation to include (a) mutants whose enclosing function's body hash changed and (b) mutants whose last killer is a test function with a row in the batch's changed-line table. The comparator asserts that list exactly.

**RR-3. Two root call lines lose their pins, and only an A-row accepts the loss.**
- R-AP-1 moves m450 (`attachDeck`) and mAK4/mRF3 (`restationTo`) into the deck. The root's new `dk.Attach` and `dk.StationMoved` calls get no mutant and no test; they are "recorded as wiring with no pin" (`design-AP-app-split.md:1102-1125`; A-23, `rulings.md:80`).
- Today, deleting the re-station step in `restationTo` is caught. After B6, deleting `dk.StationMoved()` is caught by nothing. The deck trace calls `StationMoved` directly (`design-AP-app-split.md:956`), not through `restationTo`.
- A reason for a weaker net is the HUM LEAD's to ratify, not an A-row's.
- Severity: Important. Simplify/Delete: N.
- Action: add a root test that drives `SetTransmitter` on a `livePipelines` holding a real `deck.New` deck and observes the re-fence through what the deck publishes, with a vetted mutant on the call. Otherwise put the loss to the HUM LEAD singly.

## 2. Test changes

**RR-4. The shared hooks pointer can weaken tests silently, and `Config.Timed` keeps compiling with a new meaning.**
- In 2a the hooks become one pointer shared by every copy of the model. The 41 writes after construction are "UNVERIFIED that none" is affected (`design-MS-map-seam.md:517`).
- The static test still allows a write "through the model" (`design-MS-map-seam.md:262-263`), and that write mutates the struct every copy shares.
- Concrete case: `modes/tty/map_floor_test.go:146-153` sets `d.cfg.Timed` after construction, then asserts no "below-floor" event. `Config.Timed` survives for the Router (`design-MS-map-seam.md:264-268`), so the line compiles. The map now calls its own copy of the hook, so `got` stays empty and the negative assertion passes vacuously.
- That line is unchanged, so it never appears in D-14's changed-line table.
- Severity: Important. Simplify/Delete: N.
- Action:
  - Tests change hooks only by copy-on-write (copy the hooks value, set the field, swap the pointer), held by the static test.
  - Rename the Router's field in B4's prep so every old `cfg.Timed` write fails to compile and is triaged.

**RR-5. A rebuilt pin fixture can hide a regression, because the comparator fails only on rises.**
- The console beat pin's fixture is rebuilt through `console.New` and messages (`design-CK-console-kit.md:562-566`).
- The comparator fails only on a rise (`implementation-plan.md:37-38`; A-10). A fixture that loses state measures less and passes.
- Severity: Important. Simplify/Delete: N.
- Action: in any batch where a pin's fixture or test changes, that pin must equal its baseline count (bytes within 0.1 %). A fall needs an explanation in the build log.

**RR-6. D-22's drawn-value assertions are unchecked where no mutant sits.**
- D-22's own strongest counter says so (`rulings.md:45`).
- The changed-line table accepts an assertion whose "mutants before and after" list is empty (`implementation-plan.md:49-50`). Nothing in the plan mitigates this.
- Severity: Important. Simplify/Delete: N.
- Action: each D-22 assertion with no mutant gets a vetted mutant before it changes, killed before and after.

**RR-7. 2b's 288 C3 tests are rebuilt on a hand-made `Env` fixture tied to nothing.**
- The tests are rebuilt on `New` plus an `Env` fixture (`design-MS-map-seam.md:531-534`). No check ties that fixture to what `mapEnv` produces.
- `mapEnv`'s and `applyMapWant`'s window switches are new root code with no mutants.
- Severity: Minor. Simplify/Delete: N.
- Action: a root test writes `mapEnv`'s output for the fixture's state to a `testdata` golden, and the `mapview` tests read it (no export needed). Add vetted mutants on both switches in B4.

**RR-8. The golden gate conflicts with two goldens the plan must change. BUILD will stop at B0 or B5.**
- The gate fails when a `testdata` file differs from `e57afeaf` (`implementation-plan.md:115-116`).
- `modes/tty/testdata/router-trace.golden` exists at `e57afeaf`, and B0.6 re-records it in the new form.
- `app/testdata/declset.txt` (1,141 declarations) is enforced by `TestDeclarationSetUnchanged` (`app/declset_test.go:28-39`). It must change whenever `app`'s top-level declarations change, from B5's `everyTick`→`EveryTick` onward, and in B1 if a vocabulary rename touches `app`.
- No plan or design text mentions the `app` declset.
- Severity: Important. Simplify/Delete: Y (retire the `app` declset pin into the union check).
- Action: the gate's baseline is the close of B0, with a fixed list of B0's re-recorded and new goldens. Retire `app`'s declset pin in B0, or state its re-capture rule.

## 3. Order and state

**RR-9. The S8–S10 save order claim is wrong at one site, and the instruments cannot see it.**
- The design says the root puts the save command "first in the batch, so the test harness's FIFO flattening … sees today's order" (`design-MS-map-seam.md:150`).
- That holds at `modes/tty/map_pane.go:1086` and `map_temp.go:488`/`:497`. At `modes/tty/map_prop.go:126` today's order is `tea.Batch(cmd, save)`, with save last.
- `uiApplyCmd` also captures its prefs when it is called (`setup_ui.go:184`). `switchMode` captures them before about eight more steps (`map_temp.go:482-497`), so moving the call after the map returns moves the snapshot point. Whether any of those steps writes a saved field: UNVERIFIED.
- The trace cannot see either: `SetUI` is nil in `benchDash`, so save is nil, and even if set, both commands print as `fn`.
- Severity: Important. Simplify/Delete: N.
- Action:
  - Treat S8–S10 as true split sites: cut the map function at the save, and carry its batch position in `Want`.
  - Each site's test pins the batch position and the saved prefs, written on today's code first (RR-12).

**RR-10. Clean, with three small gaps.**
- Clean: sharing one `Env` across the nine post-update steps is safe by construction. A `mapModel` method cannot write root fields, and `mapPanicked` (`map_work.go:152-163`) touches only map state and the hooks. The lock's sections and `stopAll`'s order (`app/dashboard.go:1023-1045`) match AP-3.2's rows.
- Gaps (Minor, Simplify/Delete: N):
  - Production's `Mu: &lp.mu` wiring is tested only inside the deadlock test's own setup (`design-AP-app-split.md:683`). Action: a root test that the deck's and the maps' `mu` are `&lp.mu`.
  - The order test (B0.8) parses selector calls. 2a adds a `mapEnv` call to `update`'s body (`design-MS-map-seam.md:394`), yet MS-5 says only 2b edits the list (`:409`). Action: specify a parse rule that survives 2a unedited, such as the listed names in order, each exactly once.

## 4. Performance

**RR-11. Building `Env` on every message evaluates `modalWidth()`, which lays out whichever window is open.**
- `Env` carries `ModalWidth: d.modalWidth()` (`design-MS-map-seam.md:171`) and is built on every message (`:190`, `:394`).
- `modalWidth` (`view.go:154-197`) runs:
  - `helpPlan` (`help_about.go:111`);
  - `aboutPlan` (`help_about.go:343-357`);
  - `statusBlocks` (`status.go:48-51`);
  - `setupWidth`, which measures every tab (`setup_layout.go:43-53`).
- So every shimmer tick, every 20/s viz tick and every key while Help, About, Status or Settings is open pays for a layout. The map reads this value only for its own windows.
- The beat pins run with no window open (`beat_alloc_test.go:17-26`), and the Help and Settings pins time `View` only (`bench_test.go:155`, `:335-360`).
- Severity: Important. Simplify/Delete: Y.
- Action: `Env` carries cheap values only. `ModalWidth` is set only when a map window is open (`mapWindowCols` there), and is otherwise zero.

**RR-12. Wall time, escapes and sizes have gaps the plan relies on.**
- Severity: Important. Simplify/Delete: N.
- benchstat (`implementation-plan.md:39-40`):
  - It has no failure criterion and no quiet-machine rule. PB's "no statistically significant regression … quiet machine" (`lens-PB-performance-baseline.md:408`) was not carried into the plan.
  - Moved benchmarks change their keys (`design-CK-console-kit.md:567-569`), and the plan has no "no baseline benchmark without a pair" rule.
  - "Mandatory at 2a and 2b" reads differently from the brief's "each code batch".
- Escapes are compared "by function" (`implementation-plan.md:41-42`), but 387 functions are renamed `Dashboard`→`mapModel` and then change package. A-26 maps only split functions.
- M1's "model sizes unchanged" (`project-brief.md:68`) contradicts MS-10's predicted drop of about 100 B (`design-MS-map-seam.md:648`).
- Unmeasured: `Env` (about 200 B), `Model` (about 3.6 KB), `SetMirror`'s cross-package inlining (left to the pilot).
- No allocation pin covers `Update` with the map open. `BenchmarkPropRenderMap` (`map_prop_alloc_test.go:81`) gives wall time only.
- Action:
  - State benchstat's fail rule, the machine rule and the pairing check.
  - Key escapes and inlining through `methods-MS.tsv` and the move map, with a per-package total by type as a backstop.
  - Change "unchanged" to "not larger".
  - Add an `Update` pin with the map open.

## 5. Mutants

**RR-13. The comparator would not notice a mutant re-pointed onto the wrong code.**
- The comparator only *reports* a changed target file or occurrence count (`implementation-plan.md:97-98`). A re-anchored mutant now mutating different code, but still killed by the same killer, passes.
- Killers are matched by top-level test name only (`scripts/quality/mutant-verdicts.sh:155`, `:226-235`). Duplicate names already exist, such as `TestAPlaceAbroadIsSaidByItsCountry` and `TestDeclarationSetUnchanged`.
- MS-8 says "about 6" root mutants but lists 8 (`design-MS-map-seam.md:600-601`). A derived list must be exact.
- Severity: Important. Simplify/Delete: N.
- Action: record each anchor's enclosing function at baseline, and fail when its mapping through the re-home and move maps differs. Compare killers as package plus name.

## 6. Sequencing

**RR-14. Some instruments land with the change they guard.**
- The 11 split-site tests are written in B4's prep commit, step 3 (`implementation-plan.md:206-207`), with the change itself. They never show they pass on today's code.
- The `FillStats` pin and its mutant land in B7's prep (`implementation-plan.md:242-243`).
- B0.4 requires calls to `ttytest.Main` and `apptest`'s helper (`implementation-plan.md:106-110`), which don't exist until B2 and B5, so R2's "seen passing on today's tree" cannot hold at B0.
- Severity: Important. Simplify/Delete: N.
- Action:
  - Land the split-site tests and the `FillStats` test in a commit before the change, green on `e57afeaf`-identical code.
  - State B0.4's check by behaviour (temporary config and state directories, UTC, `WATCHPOST_TEST_GOOS`), and add the helper-call rule in B2 and B5 with its planted failure.

## Verdict

**NOT READY FOR THE PLAN GATE.** Nothing here needs a new design. Every fix is plan text or a batch-0 instrument. But RR-8 stops BUILD at B0 or B5, and RR-1, RR-2 and RR-11 are silent holes in the riskiest batch.

**Fix first: RR-1.** Make the router trace build a real map, run its commands and name them by the message they return. It is the only instrument meant to catch what nobody predicted in 2a and 2b, and today it is blind there.
