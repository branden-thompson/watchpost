# Plan red team: Business Quality (BQ), 0.19.6 PLAN at `fbbfeb1a`

Scope read: the implementation plan, the architecture, the three designs and their tables in `design-data/`, D-21 to D-26, A-19 to A-28, `08-reports/m4-before.md`, plus the brief, package map and lens JN where the plan rests on them. I checked the claims below against the code at `e57afeaf`. I ran nothing except `grep`, `sed` and `awk`, and I wrote nothing to the tree.

---

## 1. Does the work meet the requirement that was written down, or the one that was remembered?

**BQ-1. M3's target table for ratification leaves out four of the brief's figures, and one of them is likely to get worse.**
The brief's M3 (`01-objectives/project-brief.md:70`) names these figures:
- the hub's fan-in, measured by type (JN P2-2);
- each area's reach into the model by type, "the map reads 72 … and Settings 102";
- the files read and packages entered by walkthroughs W6 and W8 and a legend key (P2-6);
- each area's inputs listed in its `doc.go`.

It says "Each falls from its baseline" and that the target for each is "fixed at PLAN from the graph projected on the planned layout". The plan's table (`04-development/implementation-plan.md:273-281`) has only five rows: the map's reach (×2), `Dashboard`'s methods and files, the console's reach, and the legend key. Missing:
- **Hub fan-in by type.** "752 in 68" counts the methods and files that declare methods. It is P2-2's size half, not its fan-in. `lens-JN-junior-navigation.md:321` defines fan-in as the files using the type.
- **Settings' reach, 102** (`lens-JN-junior-navigation.md:322`). No design projects it. I grepped all three designs and the plan: no projection exists. Settings stays at the root beside `Dashboard` (D-25), so it may not fall at all.
- **W6 and W8** (`lens-JN-junior-navigation.md:325`: today W6 enters 3 packages, W8 enters 1; the target is "packages whose internals the change needs: at most today's").
- **The `doc.go` inputs.** The plan's B8 docs line (`implementation-plan.md:257-258`) covers only the legend rows. Step 8 (`:50-53`) names "`doc.go` file maps", which is a different thing from declared inputs.

W6 matters most. Under D-25, a map Setting touches `platform/config`, the root (`setup_rows.go` id, table row, `mapRow`), `mapview` (choice, words, row lines, arrow, `Row` value) and `app` (`design-MS-map-seam.md:345-350`). That is 4 packages against today's 3 (`docs/extending.md:177-188`). D-25's own counter-argument says it: "a map Setting is declared in two packages through a mirrored row enum" (`02-analysis/rulings.md:48`). The legend key also does not "fall" on packages (1 → 1, `implementation-plan.md:281`).

As drafted, the PLAN gate would ratify a principle-2 yardstick that skips the figure most likely to get worse. Someone defending the release would be asked exactly that question.
- Severity: **Important**
- Evidence: `project-brief.md:70`; `implementation-plan.md:273-281`; `design-MS-map-seam.md:620-634`, `:345-350`; `lens-JN-junior-navigation.md:321-325`; `package-map.md:143`
- Simplify/Delete? N
- Action:
  - Project all four missing figures on the planned layout with the committed graph tool's method and add them to the table.
  - Where a figure rises (W6's packages, likely), put the trade to the HUM LEAD as a fork, with D-25's two-package declaration as the cost. Do not leave it unmeasured.

**BQ-2. B0 has no instrument for M5's front-door rule (P2-4).**
The brief says the front doors of `modes/tty` and `app` "grow only by the new packages' entry points" (`project-brief.md:72`). B0.12 (`implementation-plan.md:145-148`) holds only the deck's surface and D-26's three exports. The root gains at least `Config.Map` (A-22). It is UNVERIFIED whether the graph tool reports front-door exports; `package-map.md:143` lists them as a graph output, but the plan does not say so.
- Severity: Minor
- Evidence: `project-brief.md:72`; `implementation-plan.md:145-148`
- Simplify/Delete? N
- Action: name the instrument in B0.12. An exported-name list for each of the two roots, beside the deck's, is enough.

**BQ-3. The plan's citations stop at A-27, but the plan relies on A-28.**
The plan's goal cites "A-1 to A-27" (`implementation-plan.md:13`), and the architecture page does the same (`architecture.md:20`). A-28 (M4's tasks, `rulings.md:85`) is the basis of `implementation-plan.md:301-306`.
- Severity: Minor
- Evidence: as cited
- Simplify/Delete? N
- Action: read "A-1 to A-28".

**BQ-4. M7's hosted full CI is not in the plan.**
M7 requires "hosted full CI green on both systems before SHIP" (`project-brief.md:74`). B9 (`implementation-plan.md:260-267`) lists the sweep, the figures and M4, but not hosted CI.
- Severity: Minor
- Evidence: as cited
- Simplify/Delete? N
- Action: add it to B9, naming quick against full (D-128/D-129).

## 2. What does the user lose if this ships as it stands?

**BQ-5. The split leaves the Settings-to-fence wiring unpinned, which is the defect that shipped before as D-154.**
Today `restationTo` (`app/pool.go:186`) calls `mc.Refence()` (`:207-209`) and `rebed` (`:217`). Two root tests drive `restationTo` and catch it:
- mRF3, by `TestMovingTheStationsRegionTellsTheDirectorTheNewFence` (`app/pool_test.go:266`, `:278`);
- mAK4, by `TestAMovedStationReResolvesItsRelays`.

mRF3's own header records why it exists: the fence "had no path to the rail at all … a fix that is present in mechanism and absent in wiring" (`06_docs/mutants/mRF3_moving_the_region_never_tells_the_director.py:2-6`).

Under A-23 (`rulings.md:80`) and R-AP-1 (`design-AP-app-split.md:1102-1125`), both tests are rewritten to drive `Deck.StationMoved` directly (`design-data/AP/tests-AP.tsv:7`, `:49`: "Root call unpinned"). The mutants are re-pointed into the deck, so the scoped sweep shows the same killers and every BUILD instrument stays green. Yet deleting the root's `StationMoved` call would fail nothing. The user-visible effect is that narrowing the service radius in Settings leaves the old fence on the rail. The same applies to `Attach`'s root call (m450, `tests-AP.tsv:5`).

The plan's B6 (`implementation-plan.md:223-236`) never says where this "unpinned wiring" is recorded. It is an A-row (agent decision, open to veto) that loosens principle 4's net. That is not a minor choice.
- Severity: **Important**
- Evidence: `app/pool.go:186-218`; `app/pool_test.go:278`; `tests-AP.tsv:5`, `:7`, `:49`; `design-AP-app-split.md:1102-1125`; `rulings.md:80`
- Simplify/Delete? N
- Action:
  - Keep one root test that drives `restationTo`, or the Settings save that reaches it, and reads the fence through the deck's held exported surface. MasterControl is part of that surface (A-12). Whether that read is possible is UNVERIFIED.
  - Add a vetted mutant that deletes the root call, and do the same for `Attach`.
  - If no production read serves it, bring it to the HUM LEAD as a D-row with D-154 cited, not as an A-row.

**BQ-6. The plan has no README row or CHANGELOG entry for the floor raise, the one change a user can see.**
B0.13 raises `go.mod` to 1.26.11 (`implementation-plan.md:149-151`). `README.md:587` still says "Go 1.26.10 or newer (`go.mod`)", and I found no test that holds the README to `go.mod`. The last floor raise had a CHANGELOG line (`CHANGELOG.md:151`). B1.4 adds an `[Unreleased]` heading but no entry. Source builders at 1.26.10 are the users who notice.
- Severity: Minor
- Evidence: `README.md:587`; `CHANGELOG.md:151`; `implementation-plan.md:149-151`, `:169`
- Simplify/Delete? N
- Action: B0.13 rewrites `README.md:587` and adds the CHANGELOG entry in the same commit.

Apart from BQ-5, R1 looks well held. I checked D-21's clock against the code: both clocks default to `time.Now` (`modes/tty/dashboard.go:1001`; the console's `now` at `modes/tty/broadcaster.go:141-143`), so handing the Dashboard's clock to the console changes nothing in production.

## 3. What was cut, and is the cut recorded where the next reader will find it?

**BQ-7. There is no follow-up row for "a lock of their own".**
D-16 rules that "A lock of their own stays 0.19.7 (A-16)" (`rulings.md:39`). A-16 promised a follow-up row for it if the shared lock was kept (`rulings.md:73`). `06_docs/follow-ups.md` has F-228 to F-236 (`:97-105`) and none of them is the lock. 0.19.7's planner reads the follow-ups, not 0.19.6's rulings.
- Severity: Minor
- Evidence: `rulings.md:39`, `:73`; `follow-ups.md:97-105`
- Simplify/Delete? N
- Action: file the row, naming 0.19.7 and citing D-16 and A-19's "no section calls into another package".

**BQ-8. M4's REVIEW pass will mostly measure the doc fixes aimed at its four tasks, not the split, and the record does not say so.**
`m4-before.md:56-57` finds that "the docs, not the code's shape, decide that gate". B1.4 (`implementation-plan.md:165-170`) then fixes exactly the routes the four tasks failed on, on today's layout. The REVIEW trace uses the same four tasks (`project-brief.md:71`). A 4-of-4 at REVIEW would therefore show that those named fixes landed. It would not show that the re-architecture improved traceability. Meanwhile T1 crosses more packages after the split (BQ-1).

The record does not state what M4's pass will and will not prove. That leaves room for the release's principle-2 claim to be overstated at REFLECT.
- Severity: **Important**
- Evidence: `m4-before.md:40-42`, `:56-70`; `implementation-plan.md:165-170`, `:266-267`; `project-brief.md:71`
- Simplify/Delete? N
- Action (the HUM LEAD's fork, since D-18 fixed the tasks; both options below are cheap):
  - (a) run the same blind trace once after B1, on the unchanged layout with the docs fixed, so REVIEW's delta is the split's alone;
  - (b) add one held-out task at REVIEW that no B1 fix targeted.
  - Whichever is chosen, write into `m4-before.md` what the gate attributes to the docs and what to the layout.

**BQ-9. If 2b's unservable tests are not absorbed by D-22, they arrive at the last batch with no plan for them.**
MS-7 counts 120 C4 functions in 52 files, as an upper bound by rule. 53 of them assert across Settings and the map, which D-22's drawn-value class may not cover (`design-MS-map-seam.md:530-545`). D-22 sends any remainder to the HUM LEAD "singly" (`rulings.md:46`). D-9 makes 2b happen regardless (`rulings.md:32`), and B8 is last (`implementation-plan.md:247-253`). B4's re-measure (`:212`) reports a count, but nothing says what happens if the residual is large. That could mean dozens of single rulings at the release's last code step, with nothing left to fall back on.
- Severity: **Important**
- Evidence: as cited
- Simplify/Delete? N
- Action:
  - B4's re-measure classifies each C4 function against D-22, not just counts them, and reports the residual.
  - The plan states a pre-agreed way to handle that residual (one list ruling, or a named threshold that returns to the HUM LEAD before B5), so B8 cannot stall.

**BQ-10. One import edge is left to a pilot that does not test it.**
`architecture.md:77-78` says `mapview` imports `console` for one string constant (`bcNoTransmitter`, used at `modes/tty/map_prop_origin.go:51`), and that the constant moves "if the pilot shows that reads better". The pilot's list (`implementation-plan.md:58-65`) does not include it, and readability is not something a pilot measures. M5 wants every DAG edge with its reason.
- Severity: Minor
- Evidence: as cited
- Simplify/Delete? Y (delete the edge)
- Action: decide now as an A-row. Moving the constant to `frame` removes a map-to-console edge for one string.

The other cuts are recorded:
- the stutter, F-236;
- D-26's three exports, F-235;
- D-2 and A-8's list, F-228 to F-234;
- the tone-latency reconciliation, F-234.

## 4. Is there a cheaper way to get the same outcome for the user?

**BQ-11. "About 18 full gates" cannot be derived from the plan.**
`implementation-plan.md:84` repeats the brief's figure, but the plan never says how many gated commits B0 (13 instruments), B1, B4 (four prep steps, `:202-208`) or B9 take. Each batch's gate has also grown since the brief: benchstat at `-count 10` over 17 benchmarks in the two trees (`modes/tty/bench_test.go` holds 12), the pin comparator, `make p10` and the scoped sweep. The cost the HUM LEAD approved is no longer checkable against the plan.
- Severity: Minor
- Evidence: `implementation-plan.md:84`, `:35-48`, `:202-208`; `project-brief.md:18-24`
- Simplify/Delete? N
- Action: a commit and gate count per batch in the batch map, and a restated total.

Otherwise I found nothing cheaper that gives the same proof. The M4 midpoint in BQ-8 is the one cheap addition, at one agent trace. The batch order, the shared lock (D-16) and keeping `mapdata` (A-23, `design-AP-app-split.md:782-805`) are proportionate.

---

## Verdict

**NOT READY FOR THE PLAN GATE.** The structure of the plan is sound and BUILD can execute it batch by batch. But the PLAN gate exists to ratify M3's targets, and the table up for ratification leaves out four of the figures the brief wrote down, including the one most likely to rise (W6's packages under D-25).

**Fix first: BQ-1.** Project hub fan-in by type, Settings' reach, W6 and W8 on the planned layout, add them to the target table, and bring any figure that rises to the HUM LEAD as a fork. BQ-5, the unpinned station wiring that matches D-154, comes next and should be settled before B6's plan is final.
