# Code Quality red team: the 0.19.6 PLAN record (HEAD `fbbfeb1a`, code = `e57afeaf`)

The record paths below are relative to `06_docs/02_features/0.19.6-re-architecting/`. Short names: **PLAN** = `04-development/implementation-plan.md`, **ARCH** = `03-architecture-design/architecture.md`, **MS / CK / AP** = the three `design-*.md` files, **RUL** = `02-analysis/rulings.md`. Code paths are relative to the repository root. I ran only `grep`, `sed` and `git` reads. There was no `go test`, no make and no build.

---

## 1. Understandability

**CQ-1. B2's move commit cannot compile as planned. Two root tests read kit-internal helpers, and the plan puts their fix in B3 under a ruling that does not cover them.** `tests-CK-refs.tsv:37` (`TestTheConsoleReadsTheOperatorsFireThreshold`, landing in `tty`, reading `fireHot`) and `:66` (`TestTheFireDetailListsEveryNamedIncident`, reading `fireCount`) are class X, "kit-internal". Both break when `internal/frame` moves in B2. PLAN:174-182 (B2) does not mention them. PLAN:190-191 (B3) files "the 20 of D-22's class". But D-22 (RUL:45) covers *drawn values*. CK:793-795 says the 9 non-drawn X references (`fireCount`, `fireHot`, `cardTone`, `cardTitle`/`detailBody`, `.ascii`) "come back singly with their killers", and nothing schedules that.
- Severity: Important
- Evidence: PLAN:174-191; CK:505-530, :793-795; RUL:45
- Simplify/Delete? N
- Action: Split the 20 into the 11 drawn values and the 9 singles. Put the 2 kit-internal singles (and `staleAfter`, CK:526) in B2. Bring all 9 to the HUM LEAD before B2, not mid-batch.

**CQ-2. 2a's hooks type is unexported, so `app` cannot fill it.** MS:259-261 has `Config` hold `Map *mapHooks`, which "app fills once" as a keyed literal. The `app` package cannot name `tty.mapHooks`. MS:502 then calls 2b's `mapview.Hooks` "a qualifier change only", which holds only if 2a's name is already `Hooks`.
- Severity: Important (B4 step 2 hits a wall)
- Evidence: MS:253-263, :499-502; PLAN:204-205; `app/dashboard.go:424-466`
- Simplify/Delete? N
- Action: Export the type in 2a under its final bare name (`tty.Hooks`), so that 2b really is qualifier-only. Record it as an A-row.

## 2. Maintainability

**CQ-3. Decisions are deferred "to the pilot", but the pilot's scope does not contain them.** The pilot (PLAN:56-65) lists five probes. The designs push at least four more decisions onto it:
- whether `NoTransmitter` moves, which decides the `mapview → console` edge (ARCH:77-78);
- whether `ChoicesOf` is folded into `New` (MS:453-454);
- "Fields like `Role`'s are resolved in the pilot" (MS:474);
- "the exact method list, which the pilot fixes" (MS:507).

CK's Choice 8 has no ruling at all: the `raceEnabled` copies under P10-08 and `CheckGolden`'s flag under P10-06 (CK:803-810). It also conflicts with RX-5's "a copied helper is not allowed". Nothing in D-21 to D-26 or A-19 to A-28 settles it.
- Severity: Important
- Evidence: as cited; `grep raceEnabled|CheckGolden` over RUL and PLAN returns nothing
- Simplify/Delete? Y (most of these can be decided now by grep, for example `New` as `ChoicesOf`'s only caller)
- Action: Decide each now or add it to the pilot's list. Put Choice 8 to the HUM LEAD before B2.

**CQ-4. B4, the riskiest batch, has no stated commit or gate granularity.** In one "prep, in steps", B4 introduces `Env`/`Want`/`Step`, re-homes 387 methods, rewrites 11 split sites with seen-failing tests, re-points 798 test references and 146 anchors, and rewrites `app`'s hook literal (PLAN:202-213). It does not say whether the four steps are four commits each with a gate, or one commit. With one commit, a scoped-sweep failure cannot be localised. The "about 18 full gates" figure (PLAN:84) does not say which.
- Severity: Important
- Evidence: PLAN:23-35, :84, :200-213
- Simplify/Delete? N
- Action: State that B4 is four prep commits, each gated, and re-state the gate count.

**CQ-5. Work promised in M4's before is missing from the batch map.** `08-reports/m4-before.md:68-79` promises the following, "each a test in the batch that moves its file":
- Walkthrough 6 covering a map Setting that is also an Overlays row;
- the Overlays menu's hand-listed rows (`modes/tty/map_menu_test.go:54-63`);
- the `switchRow` fall-through (`modes/tty/map_boxes.go:958`);
- a legend-row golden;
- a console golden with a populated DATA PULL line.

None of these is in B4, B8 or B3. Two of them collide with other rules:
- A new or re-captured golden collides with B0.5's golden gate.
- "Fixing" `switchRow` would be a behaviour change under R1, unless it is only a guard.
- Severity: Important (T1 depends on them for M4's 4-of-4 at REVIEW)
- Evidence: m4-before.md:68-79; PLAN:247-258
- Simplify/Delete? N
- Action: Give each item a batch row. Mark which are guards only and which need a new golden, and say how the golden gate admits that golden (see CQ-11).

## 3. Complexity

**CQ-6. The "named sub-model rule" changes the engine. Calling the engine a second time avoids that.** In 2b, A-22 (RUL:79) and MS:87-94 add a rule to the walk. `walkAt` only treats `f.Anonymous` structs as the model itself (`modes/tty/memo_completeness_test.go:167-170`). A test-held set of names therefore needs a new parameter on `PerturbEach`, which changes the console guard's call site too. The simpler construction: the root's `perturbations` calls `ttytest.PerturbEach` twice. The first call is on `Dashboard`, with `mapModel` excused. The second is on `base.mapModel`, and its emit puts the perturbed model back into a copy of `base` and applies `dashboardWriter`. Reflect plus `unsafe` already reaches another package's unexported fields. This needs no engine change, no new instrument and no P10 re-ratification (CQ-15).
- Severity: Important
- Evidence: memo_completeness_test.go:121-131, :160-190; MS:87-94; PLAN:249-250
- Simplify/Delete? Y (delete the rule)
- Action: Re-plan B8 commit 1 around the two-call form, or bring the trade to the HUM LEAD.

## 4. Necessity

**CQ-7. `Env` carries four derived sizes, and one of them runs the open window's whole layout on every message and every frame.** MS:170-173 lists the four:
- `Rows = d.modalMax()`
- `Ceiling = d.mapCeiling()`
- `Cols = d.mapWindowCols()`
- `ModalWidth = d.modalWidth()`

MS:397 builds one `Env` per message after `resume`. MS:467 gives `MemoKey(w Window, e Env)`, which the root calls for every window (`modalKeyFor` reads the map choices for every modal, `modes/tty/memo.go:385-391`). `modalWidth()` does real work for some windows:
- Status: `statusWidth` builds `statusBlocks(0)`, `modes/tty/status.go:48-51`.
- Settings: `setupWidth` walks every tab, `modes/tty/setup_layout.go:43-53`.
- Help and About: `helpPlan` and `aboutPlan` build blocks, `modes/tty/help_about.go:111-114`, `:343-357`.

The map arms never call these today. So with Settings or Status open, every beat pays for the window's layout, and so does every memo hit. The window pins measure `View()` only (`modes/tty/bench_test.go:179`). The beat pins run with no window open. So M1 may not see this, and it would land on the Update path unmeasured. All four values are functions of `Width`, `Height` and `Window`, and the map already has `Width(w Window)` (MS:452).
- Severity: Important
- Evidence: as cited
- Simplify/Delete? Y (delete all four fields and derive them inside the map)
- Action: Remove them from `Env`. Give `MemoKey`'s choices part no `Env` parameter. Add a beat pin with Settings open.

**CQ-8. The shared-lock check's `…Locked` allowance exists for a form A-19 says is never needed.** AP-3.5 rules 2 and 3 (AP:665-666) let a cross-package call under the lock through when the callee's name ends in `Locked`. AP-3.4 (AP:648-649) writes the same convention on the field. A-19 (RUL:76) says no section calls into another package and no `…Locked` form is needed. So the allowance is a name-based escape hatch with zero users. Rule 4's allowance is also by bare name (`now`, AP:667), so any func value named `now` passes.
- Severity: Minor
- Evidence: AP:648-667; RUL:76
- Simplify/Delete? Y
- Action: Rule 2 fails on any cross-package call under the lock. Key the allowance to declarations, not names.

**CQ-9. `Env.Ticks` exports a test seam.** `d.ticks` is nil in production and set in one test file (`modes/tty/dashboard.go:819-823`, `:1067-1069`). MS:184 makes it an exported field of `mapview.Env`, which M5's P2-5 exists to stop in spirit. Also, D-26 keeps `ChipKnown` exported although it has no production user anywhere (MS:488). MS's own C-6 recommended unexporting it (MS:720).
- Severity: Minor
- Evidence: as cited; RUL:49
- Simplify/Delete? Y
- Action: Put both to the HUM LEAD: the map returns the tick it wants in `Want` (or the root's `after` stays the only tick source), and `ChipKnown` is unexported.

## 5. Name semanticism (SN-01)

**CQ-10.**
- **`frame` outgrows its definition.** A-20 defines it as "the parts of a frame both surfaces draw" (RUL:77; ARCH:30). MS-3 then adds 14 declarations (`Surface`, window chrome, `spliceBox`, `refOf`), the selection function over `Pick` and the modal wrap (MS:190-191, :307). Those are root-and-map parts, not both surfaces' parts. MS:365-369 rejects putting the controls in the kit for exactly this "util shape" reason. `frame` is also already a local variable in map code (`modes/tty/map_pane.go:498`, `:511`), the shimmer phase, and `MapFrame`.
- **`app/internal/run`** holds `EveryTick`, `LockedCopy`, `UserAgent` and `RequestStats`, which share no role. `run.EveryTick` reads as a verb. AP's own counter-argument concedes it (AP:1256). Three of its seven helpers are values the root could pass in `Config` (`UserAgent`, `CacheDir`, `UserCacheSubdir`).
- **`deck.Deck.deck`** (AP:420) is the radio player inside "the deck".
- Severity: Minor
- Simplify/Delete? Partly (shrink `run` by passing values)
- Action: Name the selection and chrome helpers' home honestly, or keep them out of `frame`. Rename `Deck.deck` to `player`.

## 6. Code documentation

- **The import graph disagrees with itself.** ARCH:63 and :77 draw `MAPVIEW --> CONSOLE`. MS:305 says mapview imports "controls, the kit, platform" only. Minor; ARCH:63, MS:302-308. Action: decide `NoTransmitter`'s home now (CQ-3).
- **`MemoStats` has two lock contracts on one type, and no comment says so.** The embedded mutex guards the whole memo slot (`modes/tty/memo.go:155-165`; `broadcaster_memo.go:139-147`). `Hit` and `Miss` run with the lock held; `Counts` takes it (CK:115-117). Minor. Action: the design states that the comments on `Hit` and `Miss` say "lock held".
- **The record cites a stale range.** PLAN:13 and ARCH:20 cite "A-1 to A-27", but A-28 exists (RUL:85). Minor.

## 7. Testability

**CQ-11. `mapview.Window` mirrors the root's `modal` enum through two hand-written switches (`mapEnv`, `applyMapWant`), and no test proves the two are inverse and complete.** A missing arm would turn `Want{Open: WindowPropFreq}` into a silent no-op.
- Severity: Minor
- Evidence: MS:217-220
- Simplify/Delete? N
- Action: Add a round-trip test over every `Window` value in B4.

## 8. P10 conformance

**CQ-15. The engine B8 changes is one D-23 ratified as unchanged.** D-15 (RUL:38) and D-23 (RUL:46) ratify P10-01 rows for `walkAt` and `bump` on the grounds that the engine is "unchanged in body". A-22 and MS:92-93 then change the walk in B8. PLAN:247-258 has no P10 line for that change. The ledger may change "only by ratification" (D-4).
- Severity: Important
- Simplify/Delete? Y (CQ-6 removes the change)
- Action: Take CQ-6's route, or plan B8's re-ratification of the `walkAt` row explicitly.

The rest of P10 is clean as planned:
- The re-key table (PLAN:285-296) matches CK-8 and AP-5.
- P10-04: `Answer` repeats today's `routeMap` switch (`modes/tty/dashboard.go:1262-1297`), so it adds no complexity.

## 9. Evidence soundness (verifiers fail closed)

**CQ-12. benchstat is "mandatory" but has no failing condition.** PLAN:39-40, :213 and :256, the brief's M1 and B-2 (`08-reports/red-team-review.md:93`) name the comparison but no threshold or verdict. A step that only prints a table cannot fail, so it passes principle 1 by default. Another timing job is running on this machine right now, which is also why a noise rule is needed.
- Severity: Important
- Simplify/Delete? N
- Action: Have the HUM LEAD rule a criterion before B0. For example, any statistically significant slowdown above x % fails, or goes to the HUM LEAD. Include a rule for noisy runs.

**CQ-13. B0.5's golden gate fails on B0.6's own planned change.** The gate fails when "a file under `testdata/` differs from `e57afeaf` except as a pure rename" (PLAN:115-116). B0.6 rewrites `modes/tty/testdata/router-trace.golden` (`modes/tty/router_trace_test.go:66`) in a new form. B0 also adds `app/testdata/deck_trace.golden` and `app/testdata/sharedlock/`. If the gate admits additions, a re-capture done as delete-plus-add passes.
- Severity: Important
- Simplify/Delete? N
- Action: Take the gate's baseline as B0's closing commit, with B0's (and CQ-5's) added files listed exactly, and refuse any other addition.

**CQ-14. The test inventory has no rule for a test's identity across a move or a split.** B0.5 promises "the same test inventory" (PLAN:112-116; brief M7). But by design:
- AP-9 renames the non-killer half of each split test (AP:1094-1096);
- 288 tests move package in B8 (MS:545);
- D-22 rewrites tests.

Keyed by (package, name), every move commit reads as a loss. Keyed by name alone, a deleted test hides behind a same-name one in another package.
- Severity: Important
- Simplify/Delete? N
- Action: Write the identity rule and a reviewed per-batch delta table, as A-11 does for anchors.

**CQ-16. Other instruments can pass vacuously.**
- B0.4's `TestMain` check keeps a floor of 2 packages for the whole release (PLAN:110; CK:653), while 11 packages are planned. Raise the floor per batch to a fixed list of the packages that exist.
- AP-3.5's lock check reads a committed *planned* file-to-package map (AP:657-659). Nothing says what it does with a file the map omits, or when the map is retired. Unmapped files must fail, and the map should be deleted once the moves land.
- Severity: Minor

**CQ-17. M7 cannot see the pins A-23 gives up.** Under R-AP-1, the anchored statements of m450, mAK4 and mRF3 move into the deck with their killers. The root's new `dk.Attach` and `dk.StationMoved` call lines carry no mutant (AP:1103-1128; RUL:80). Deleting `dk.StationMoved(ctx)` from `restationTo` would be caught by nothing: the trace drives `StationMoved` directly (AP:972-974). Yet the closing sweep "matches A-3's" mutant by mutant, and reports no loss.
- Severity: Important
- Simplify/Delete? N
- Action: Add a root mutant that drops each call, killed through the root's `SetTransmitter` and the deck's tier-2 `tty.Station` surface. Where that is truly impossible, list the pins lost in the M7 report rather than counting the sweep as equal.

## 10. The evasion lens

**CQ-18. The graph tool is "its own module under `tools/`" (PLAN:147; A-12), and Go's `./...` stops at a nested `go.mod`.** So `vet`, `lint`, `race`, `p10`, `dupes`, `vuln` and `tidy` (`Makefile:297`) never see the tool's code. Its make target must also hop into another module. Every existing tool is in the main module (`tools/` has no `go.mod`), and `go.mod`/`go.sum` have no `golang.org/x/tools`, which is why the module split looks wanted. A gate's oracle that sits outside every gate is the drift case the evasion lens exists for. The tool gates M3 and the prep commits' crossings (AP:1276-1277), and the lock check reads its input.
- Severity: Important
- Simplify/Delete? N
- Action: Either take `x/tools` into the main module, or add the nested module explicitly to vet, lint, test, vuln and the licence script, held by `TestTheThreeGateListsAgree`'s kind of check.

Nothing else in the plan re-exports `PATH`, sources a script, or checks for the CI environment.

---

## Verdict: NOT READY FOR THE PLAN GATE

The shape is sound, and the designs are unusually well evidenced against the code. But BUILD would hit two walls:
- **CQ-1:** B2 does not compile without rulings that are scheduled a batch late.
- **CQ-2:** 2a's unexported hooks type cannot be filled by `app`.

Three proofs could pass without proving anything: benchstat without a criterion (CQ-12), the golden gate against B0's own goldens (CQ-13), and the test inventory without an identity rule (CQ-14).

**Fix first: CQ-7.** It is the one finding that could ship a real performance regression past every planned instrument. The fix is a deletion: four derived fields out of `Env`, and no `Env` on the memo key's choices part.

Word count: about 2,300.
