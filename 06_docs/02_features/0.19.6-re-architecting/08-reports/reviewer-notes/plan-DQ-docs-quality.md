# Plan red team, Docs Quality (DQ): 0.19.6 PLAN at `fbbfeb1a`

I worked read-only and checked the claims against the code (no `.go` file differs from `e57afeaf`). My scratch work, the method list and the assertion-line grep, is in `<scratch>/`. I ran no tests, no make and no network calls.

## 1. Does each document answer the question its title asks?

Mostly yes. The plan gives an order, the batches and what is to be ratified. The architecture page gives the layout. m4-before gives a graded before. Two gaps:

**DQ-1. The planned architecture leaves one import edge undecided, and hands it to a pilot that does not test it.** `architecture.md:77-78` says `mapview` imports `console` for one constant, `NoTransmitter`, and "the map batch moves it to the kit if the pilot shows that reads better". The pilot's list (`implementation-plan.md:58-65`) has no such item. "Reads better" is not something a pilot measures. In the code the constant is `bcNoTransmitter` (`modes/tty/broadcaster_header.go:141`), read once by the map (`modes/tty/map_prop_origin.go:51`).
- Severity: Minor.
- Evidence: `architecture.md:63`, `:77-78`; `implementation-plan.md:56-67`.
- Simplify/Delete? Y.
- Action: decide the edge at PLAN, either kit or console, and record it as an A-row. Drop the pilot reference.

**DQ-2. The AP design's title and body use names that A-14 and A-20 retired.** The title says "the maps package, the kit". The headings say `app/internal/maps` (`design-AP-app-split.md:717`) and `app/internal/kit` (`:806`). `mutants-AP.tsv` and `livepipelines-AP.tsv` use `maps` as a destination. A-14 forbids `kit`, and forbids `maps` because it shadows a standard-library package (`rulings.md:71`). The page's status line says its choices were ruled. A BUILD reader has to translate every name.
- Severity: Minor.
- Evidence: `design-AP-app-split.md:2`, `:717`, `:806`; `rulings.md:71`, `:77`.
- Simplify/Delete? N.
- Action: put a one-line name map at the top of the AP page (maps → `mapdata`/`Feeds`, kit → `run`), or rename in place.

## 2. Does anything contradict the code, or another document?

**DQ-3. B4's "no assertion changes" is contradicted by the code.** The plan says 2a re-points "798 call references in 68 files" with "no assertion changes (design MS-7)" (`implementation-plan.md:209`). MS-7 says each of those references "goes to `d.mapModel` with `d.mapEnv()` … never an assertion's value" (`design-MS-map-seam.md:515`). Package map C-1's "the net untouched" rests on this (`package-map.md:90`).

In the code, the re-homed methods are called inside assertions:
- `methods-MS.tsv` re-homes `mapFits` and `mapBodySize`.
- `mapBodySize` reads the root's `modalMax` (`modes/tty/map_pane.go:423-433`). `crossings-MS.tsv` classes `modalMax` as `ENV` (`Env.Room.Rows`).
- So `if !d.mapFits() {` (`modes/tty/map_floor_test.go:43`) and the `t.Fatalf(… d.mapBodySize() …)` beside it (`:44`, `:94`) have to change.
- Over `modes/tty/*_test.go`, 329 lines in 49 files call a re-homed method on `d.` inside an `if` condition or a failure message. The most common are `layerOn` (40), `mapMode` (27), `mapAsk` (20) and `mapStatusLine` (17).

Calls to methods that need no `Env` compile unchanged through promotion. The record has no per-method signature that would say which do not. D-14 permits these changes, but only under its proof. The plan sets the opposite expectation, so B4's changed-line table will contradict the plan on day one.
- Severity: Important.
- Evidence: `implementation-plan.md:209`; `design-MS-map-seam.md:515`, `:519`; `modes/tty/map_pane.go:423-433`; `modes/tty/map_floor_test.go:43-44`.
- Simplify/Delete? N.
- Action: say which of the 154 test-called methods gain an `Env` parameter or return the model. Count their references in assertion lines. Restate B4 as "assertions change only under D-14, listed in the changed-line table", and correct MS-7 and C-1's wording.

**DQ-4. B0.4 cannot pass on today's tree.** B0 items are "seen failing on a planted case, then passing on today's tree" (`implementation-plan.md:90`). B0.4 requires every package under `modes/tty/` to call `ttytest.Main`, and every package under `app/` to call `apptest`'s helper (`:106-111`; CK-7, `design-CK-console-kit.md:637-655`). Neither helper exists until B2 (`:172-178`) or B5 (`:215-220`). Today's `TestMain` is three inline lines (`modes/tty/main_test.go:20-22`). As written, B0.4 either fails at B0 or is quietly weakened.
- Severity: Important.
- Evidence: `implementation-plan.md:90`, `:106-111`, `:178`, `:220`; `modes/tty/main_test.go:20-22`.
- Simplify/Delete? N.
- Action: either land `ttytest.Main` and the `apptest` invariants helper in B0, or state B0.4's B0 form (for example, the parent's invariants inline) and the batch that tightens it.

**DQ-5. The P10 re-key table's proof for the Router rows is false for `Dashboard`.** The table re-proves `:49`–`:55` as "the surfaces' `Update` and `View` are in other packages" (`implementation-plan.md:288`). `Dashboard` stays in `modes/tty` with the Router (`architecture.md:27`). The ledger's own proof is that `surfaceView`'s implementers hold no Router field (`06_docs/p10-ledger.md:50`; `modes/tty/router.go:849-857`). The table goes to the HUM LEAD for ratification under FULL P10 (D-4), so a wrong reason would be ratified.
- Severity: Important.
- Evidence: `implementation-plan.md:288`; `architecture.md:27`; `06_docs/p10-ledger.md:50`; `modes/tty/router.go:849-857`.
- Simplify/Delete? N.
- Action: restate the proof per surface. The console is in another package. The Dashboard is unchanged, and neither holds a Router field. Note whether 2b adds same-named `mapview` methods that P10-01 will flag.

**DQ-6. The architecture's "Imported by" column disagrees with its own graph.** The table (`architecture.md:27-37`) says:
- `console` is imported by `modes/tty` and `app`, but the graph also draws `DECK --> CONSOLE` and `MAPVIEW --> CONSOLE` (`:54`, `:63`);
- `mapview` is imported by `modes/tty` and `app`, but the graph draws `MAPDATA --> MAPVIEW` (`:55`);
- `modes/tty` is imported by `app` and `cmd/watchpost`, but the graph draws `DECK --> TTY` and `MAPDATA --> TTY` (`:50-51`), and has no `CMD --> TTY`. In the code, `cmd/watchpost` imports `modes/tty` only in `cmd/watchpost/docs_test.go`.
- Severity: Minor.
- Evidence: as cited.
- Simplify/Delete? N.
- Action: regenerate the column from the graph and mark test-only importers.

**DQ-7. A design table disagrees with its design and with A-27.** AP-10 and A-27 give deck 116, maps 13, root 23 and 1 test anchor (`design-AP-app-split.md:1146-1151`; `rulings.md:84`). `mutants-AP.tsv` gives 113, 12, 27 and 1:
- m450, mAK4 and mRF3 are `app` there, where AP-10 moves them to the deck (`:1157-1158`);
- m183 is `app`, where AP-10 moves it to the maps package.

B6 cites 116 (`implementation-plan.md:233`). A-11's derived list catches the difference at run time, but the committed table is the stale one.
- Severity: Minor.
- Evidence: as cited.
- Simplify/Delete? N.
- Action: regenerate `mutants-AP.tsv` after A-23.

**DQ-8. The alias rule is stated more widely than the code allows.** B0.9 says "a check that no type alias survives", and the architecture and M6 say the same (`implementation-plan.md:137`; `architecture.md:73`; brief `:73`). Today the tree already has three aliases: `SevereTab` (`modes/tty/severe.go:23`), `MapView` (`modes/tty/map_boxes.go:1020`) and `TickerCategory` (`modes/tty/ticker.go:37`). MS-6 moves `MapView` into `mapview` as an alias (`design-MS-map-seam.md:481`). GP-11 limits the rule to aliases that point into the split tree (`lens-GP-go-practice.md:483`). The plan drops that limit.
- Severity: Minor.
- Simplify/Delete? N.
- Action: write GP-11's scope into B0.9, or list the three existing aliases.

**DQ-9. "Model sizes unchanged" has no direction.** The brief's M1 says "model sizes unchanged" (`project-brief.md:68`). MS-10 projects that `Dashboard` falls by about 100 B and that `Config` loses 15 func fields (`design-MS-map-seam.md:646`). B0.7's size probe has no rule (`implementation-plan.md:128`), and B0 fixes floors.
- Severity: Minor.
- Simplify/Delete? N.
- Action: state "at or below its baseline" in B0.7, and recommend the matching wording for M1 to the HUM LEAD.

**DQ-10. Smaller cross-document slips** (Minor, one batch, Simplify Y):
- `implementation-plan.md:13` cites "A-1 to A-27", but A-28 (M4's tasks) exists (`rulings.md:85`).
- `implementation-plan.md:266-267` says B9's key is built "before the docs are rewritten". B1 rewrites them first (`:153-170`). 0.19.5 D-24 says "before REVIEW".
- Context, which the plan follows correctly: the settled package map still says a move commit makes "no comment edits" (`package-map.md:147-149`), against R3 (`project-brief.md:83-85`). It also says scan floors are "taken from the tree" (`:111`), against A-12 and B0's fixed floors (`implementation-plan.md:91`). Under D-311 a BUILD agent judges against that page. The `deck`/`maps`/`kit` names at `package-map.md:72` are also stale.
- The "every batch" order (`implementation-plan.md:30-34`) gives each batch a prep commit and a move commit. B1 and B4 move nothing, and B4's four prep steps (`:202-208`) do not say how many commits or gates they are.

## 3. Is a claim made that nothing verifies, in a document that reads as verified?

**DQ-11. The record's computed figures rest on scripts and inputs that are not in the tree, and B4 depends on one of them.**
- All three designs read "FILED — checked by the lead against e57afeaf".
- MS says "All scripts and tables are in `design-data/MS/`" (`design-MS-map-seam.md:39`). That directory holds five TSVs and no scripts. None of `tests_fn_ms.py`, `cross_raw.py`, `methods_ms.py`, `file-edges.tsv`, `cross-package-edges.tsv` or `test-reach-MS.tsv` is tracked (`git ls-files`: none).
- B4's re-measure, which D-9 and D-22 require before B8, is to be run "by `tests_fn_ms.py` on 2a's code" (`implementation-plan.md:212`). BUILD cannot run a script that exists only in a past session's scratch directory.
- Severity: Important.
- Evidence: `design-MS-map-seam.md:39`, `:44-52`, `:567`; `implementation-plan.md:212`.
- Simplify/Delete? N.
- Action: commit the design scripts with B0.12's graph module, or name B0.12's tool as the re-measure and require it to reproduce `tests-fn-MS.tsv`'s 288/120 on `e57afeaf` first. Correct MS:39.

## 4. Is the audience order right (designers, then PMs, then engineers)?

This is a fork for the HUM LEAD. I am not ruling on it.
- **The plan** opens on the branch, R1, and then nine engineering steps. What the HUM LEAD ratifies at PLAN exit (M3's targets, the P10 re-key table, M4) is in the last 40 lines (`implementation-plan.md:271-306`). The plan has no "In short", unlike the brief (`project-brief.md:11-28`). Option A: move a short "For ratification" summary to the top, with the tables left where they are. Option B: keep today's order, which follows the work.
- **The architecture page** puts "What changes for a reader", the one table a non-engineer needs, last (`architecture.md:121-131`), after three diagrams. It has the same two options.
- **m4-before** is in the right order: what it is, the grade, what was found, what it means.

## 5. Is a number published without its blind spots beside it (INST-5)?

**DQ-12. The M3 table put up for ratification leaves out three of the brief's figures, and drops its design's caveats.** The brief requires each of these to fall, with targets fixed at PLAN (`project-brief.md:70`):
- the hub's fan-in, by type;
- each area's reach into the model ("Settings 102");
- the files and packages walkthroughs W6 and W8 enter;
- the legend key.

The plan's table (`implementation-plan.md:275-281`) has the map's reach, `Dashboard`'s method count (which is not fan-in), the console, and the legend key. Nothing in the plan or the designs projects fan-in, Settings' 102, W6 or W8 (grep: none).

The caveats MS-9 states are dropped:
- "the 40 depends on MS-4 A";
- the legend target "holds only if 2b's `doc.go` carries that row list";
- in 2a the model's methods are still promoted into `Dashboard`'s method set (`design-MS-map-seam.md:626-633`).

The "Today" figures (72, 93, 13) come from the uncommitted scratch graph ("reproduced from `file-edges.tsv`", MS-9; CK-9 `:721`). They are ratified before B0.12 commits the tool that will judge them. No step checks that the committed tool reproduces them.
- Severity: Important.
- Evidence: `project-brief.md:70`; `implementation-plan.md:273-281`; `design-MS-map-seam.md:624-633`; `design-CK-console-kit.md:721`.
- Simplify/Delete? N.
- Action: add the missing rows with projections, or put their omission to the HUM LEAD. Add a blind-spot column. Make B0.12 reproduce the "Today" column before the targets bind.

**DQ-13. If M4 passes at REVIEW, it will mostly show that these four tasks' doc fixes were made, not that the split helped, and the record does not say so next to the target.**
- REVIEW runs the same four tasks (brief M4, `project-brief.md:71`; `implementation-plan.md:301-306`).
- m4-before concludes "the docs, not the code's shape, decide that gate" (`m4-before.md:56-57`).
- B1.4 then fixes the specific routes these four tasks failed on (`implementation-plan.md:165-170`).
- The keys are committed on the branch the builder works on (`08-reports/m4-key/`).
- The before is one blind reviewer, one run per task. m4-before does not say who graded the run (`m4-before.md:22-29`).

A 0-to-4 change at REVIEW would show that these fixes were made, not that "a reader can follow a behaviour to its code" improved for the split. That is principle 2's traceability claim.
- Severity: Important. This is a recommendation to the HUM LEAD, since M4 is in the approved brief.
- Evidence: as cited.
- Simplify/Delete? N.
- Action: put the fork to the HUM LEAD. A: record the blind spot (tasks known, run once, grader named) beside M4's target. B: also run one task fixed now but sealed from the builder.

## Opinion

**NOT READY FOR THE PLAN GATE.** Most fixes are edits to the record, not to the design. But three of them would stop BUILD:
- B0.4 cannot pass at B0 (DQ-4);
- B4's re-measure names a script that is not in the tree (DQ-11);
- B4 promises "no assertion changes" where the code shows otherwise (DQ-3).

Two more would put wrong or incomplete items in front of the HUM LEAD for ratification: the M3 table (DQ-12) and the Router rows' P10 proof (DQ-5).

**Fix first: DQ-3.** 2a's "net untouched" is the premise of D-9's two-step map move, and of the claim that 2a proves no regression for free. `modes/tty/map_floor_test.go:43` shows it does not hold as written. The changed-line table and D-14's proof need to be planned for B4, not found there.
