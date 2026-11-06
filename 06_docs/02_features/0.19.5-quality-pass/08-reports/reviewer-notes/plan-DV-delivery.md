# Delivery — plan red team, 0.19.5

## Verdict: DO NOT EXIT

The plan carries almost every finding and ruling, and its order is mostly sound. Six Important gaps
would each cost a batch, a hosted cycle, or a metric. All six are fixed by editing the plan; the
approach does not need rework:

- the console beat cannot reach 140 KB with the tasks planned, and this is knowable today;
- there is no batch map;
- the D-11 and D-12 rules arrive in the wrong order;
- two Q7 checks need git history that CI does not fetch;
- P10 is checked only at phase exit;
- Q1.2 changes the clock without a ruling.

Evidence for M2 was measured on the PF lens's own prototype copy (`<scratch>/q195/pf/wp`). It was
re-run read-only, outside the repository: `BenchmarkPF_Router{Tick,Viz}Beat_133x44` and
`BenchmarkPF_RouterConsoleTickBeat`, 1,000 iterations, `-memprofilerate 1`. The figures reproduced
exactly: 104,908 / 112,467 / 161,290 B a beat. Profiles are in `<scratch>/q195/rt/dv/m-*.out`.

## Findings

### DV-1 [Important]: M2's console target (140 KB) is out of reach of every task in the plan; the idle and radio targets depend on one unprototyped M change
- Plan task(s): Q3.1 (second step), Q4.1 to Q4.4, Q4's "M2's verdict", Q6.1, Q6.2.
- Observed:
  - **Per-beat bytes left after the S fixes** (allocation profile of the prototype copy):

    | Beat (today → after S fixes) | What the medium work can still remove |
    |---|---|
    | Idle 104.9 KB | `Router.update` box 8.0; `Dashboard.onTick` box 6.6 and `armTick` box 6.7 (PF-1 M; the lens estimates 13.5); masthead `Dashboard.header` 7.9 (PF-7); ticker `tickerLane` 7.5 plus `tickerLoopLen` 6.5 (PF-8) |
    | Radio 112.5 KB | the same boxes plus `takeCmd` 6.6 (PF-1 M); `header` 7.9 (PF-7); `tickerLane` 3.8 (PF-8); `radioMaxRows` 7.9 (PF-3 step 2) |
    | Console 161.3 KB | `Router.update` box 8.0 (PF-1 M). **Nothing else in the plan.** The console beat runs no `Dashboard` code at all: the Router sends the tick only to the Broadcaster (`router.go:507-511`). PF-7's memo is Observer's `body.go` header, PF-8 is Observer's ticker, PF-9 is Observer's `bodyMemo` (the console's `consoleMemo` already hits), and PF-10 is in the assembler. |

  - **Arithmetic.**
    - Idle: 104.9 − 13.5…21 − about 7 − 10…13 ≈ 64 to 74 KB, against 80. Met, but only if PF-1 M lands; without it, about 86.
    - Radio: 112.5 − 13.5…21 − 7 − 3.5 − 5…7 ≈ 74 to 83 KB, against 90. Met; without PF-1 M, about 96.
    - **Console: 161.3 − 8 ≈ 153 KB, against 140. It misses by about 13 KB.**
  - **What the console spends instead**, all outside every PF finding:
    - `Broadcaster.railed`: 33 KB a beat, 13.5 of it `PadTo` over memoised rows;
    - `lineup.Projection`: 14.3 KB (9.5 from `mainTrack` in View, 4.8 from `tickNeeded` in Update);
    - `Broadcaster.header`: 8.8 KB, the same masthead helpers PF-7 memoises for Observer (`broadcaster_header.go:25-29`);
    - `stationSection`: 26 KB.
  - **PF-1 M is bigger than the plan says.** Q3.1's second step needs the handlers to return `Dashboard`. That is 49 `func (d Dashboard) … (tea.Model, tea.Cmd)` handlers in 16 files. It is unprototyped (the lens: "M … about another 13.5 KB"), yet it carries more of the idle and radio targets than PF-7 and PF-8 together.
  - **The verdict is recorded too early.** It comes at the end of Q4, but Q6.1 and Q6.2 then change the console frame and its allocation budget. The figure recorded is not the figure that ships.
- Consequence: the console target fails at the end of Q4, about seven batches in. The HUM LEAD is then asked a question that could be asked at the PLAN gate. Separately, if Q3.1's second step is cut for size, idle and radio miss as well.
- Proposed fix to the plan:
  - Take the console figure to the HUM LEAD at the PLAN gate, with options (this does not re-open D-4):
    - (a) widen Q4.2 so the masthead memo also serves `Broadcaster.header`, worth about 8.8 KB;
    - (b) add a task caching `Projection` per lineup generation behind `consoleMemo`'s completeness guard, worth about 14 KB; or memoise `railed`;
    - (c) accept the measured figure.
  - Prototype PF-1 M, PF-7 and PF-8 in the scratch copy before BUILD.
  - Re-measure M2 and re-lower the pins after Q6, not only after Q4.

### DV-2 [Important]: The plan has no batch map; as written, the gates run once per task
- Plan task(s): the "Every task follows the same order" list (steps 4 and 7); A-11.
- Observed:
  - Step 4 runs `make verify` per task, and the commit is "chained to the verdict". Step 7 reads "each batch's hosted run", but no batch is ever defined. A-11's "0.19.5 batch N" numbers are never assigned.
  - Q1 to Q9 hold 62 tasks. `06_docs/verify-runs.md` shows `verify` at 1,840 to 2,056 s.
- Consequence:
  - Per task: about 62 × 33 min ≈ 34 h of local verify, and up to 62 serialised hosted runs (about 155 h).
  - Per work package (nine batches): Q3 would put six layout-changing edits behind one verify, and a golden or `TestFrameAllocBudget` failure after PF-2, PF-3 and PF-4 together is costly to bisect.
- Proposed fix to the plan: add a batch table. My recommendation is about 14 full-gate batches (about 7.7 h of verify in all), with the risky changes alone:

  | Batch | Contents | Why |
  |---|---|---|
  | B1 | Q1, plus Q7.7 | D-5's batch; see DV-4 for Q7.7 |
  | B2 | Q2.1 to Q2.7 | the one production seam, with the app-side pins |
  | B3 | Q2.8 to Q2.12 | goldens, the beat pin, timing tests |
  | B4 | PF-1 first step, PF-11, PF-4, PF-5 | by construction or byte-identical |
  | B5 | PF-2 alone | every width changes |
  | B6 | PF-1 second step, plus PF-3 | the 49-handler change |
  | B7 | PF-10, PF-7, PF-8, Q4.5 | |
  | B8 | PF-9 alone | the stale-row risk; then the interim M2 figure |
  | B9 | Q5 | |
  | B10 | Q6.1, plus Q6.2 | one console change, one UAT look |
  | B11 | Q6.3 | 41 mutants re-vetted by hand |
  | B12 | Q6.4 | |
  | B13 | Q7 | |
  | B14 | the Go-comment work of Q8 and Q9 | |

  The Markdown-only work of Q8 and Q9 (Q8.2, Q8.3's text, Q8.7's docs, Q9.2, Q9.6) goes in separate `.md`-only commits, so it takes `verify-docs` (about 145 s). Q8.4 cannot: it regenerates the atlas (`.html`).

### DV-3 [Important]: The D-11 and D-12 rules arrive after the code they govern, so Q9.8 re-touches everything Q1 to Q8 touched
- Plan task(s): Q7.6, Q8.1, Q8.6, Q9.4, Q9.8.
- Observed:
  - Q7.6 lands the release-qualified-citation check on added comment lines.
  - The index that makes a bare ID resolvable (`ids.md`) arrives later, in Q8.6.
  - The all-caps rule's text arrives last, in Q9.8, which then says "Code that Q1 to Q9 touched is brought to the all-caps rule".
  - Meanwhile Q8.1 restates comments that carry bare IDs. Q9.4 moves seven essays, IDs kept; `stationLine` alone cites about 15 0.16.0 IDs (lens JR-8), and `broadcaster.go` holds 110 `D-n` cites.
  - "Touched" is undefined. Q3.1's second step alone touches 49 handlers in 16 files, so Q9.8 drifts toward the 618-comment sweep D-12 did not choose.
  - No tool in the tree computes "added lines" today (`grep` of `tools/authoring`, `tools/plancode`, `Makefile`).
- Consequence:
  - Q8.1's and Q9.4's moved lines trip Q7.6, and qualifying them needs an index that does not exist yet. That is a failed gate, then research, then a re-run.
  - Q9.8 becomes a large final batch that re-edits files Q3 to Q8 already reviewed.
- Proposed fix to the plan:
  - Move D-12's and A-2's text in `code-standards.md`, and Q8.6's `ids.md`, into Q7, beside Q7.6.
  - Apply the all-caps rule inside each task, to the hunks it changes ("touched" = changed hunks).
  - Q9.8 then only verifies.

### DV-4 [Important]: Q7.6 and Q7.7 depend on git history the hosted run does not have, and Q7.7 lands after the hazard it removes
- Plan task(s): Q7.6, Q7.7.
- Observed:
  - `ci.yml:39`'s `actions/checkout` sets no `fetch-depth`, so it fetches depth 1 with no tags (`release.yml:20` sets `fetch-depth: 0`, `ci.yml` does not).
  - `Makefile:13` is `VERSION ?= $(shell git describe --tags --match 'v*' --always --dirty …)`. Two problems follow:
    - In CI that falls back to a bare hash.
    - Mid-BUILD on the branch it is `0.19.3-N-g<hash>`. So `verdicts/<version>.log` names a file per commit, not per release.
  - "A test pins both derivations" (Q7.7) and "added lines" (Q7.6) both need a base or a tag that CI lacks.
  - DEF-4 is live until Q7.7 (batch 13 in DV-2's map). Q6.3 re-vets 41 mutants. If any `make mutant-verdicts` runs before then, it overwrites 0.16.0's shipped record (`Makefile:382`).
- Consequence: a test that reads the latest tag passes locally and fails on both hosted legs. That costs a 2.5 h cycle and a fix batch. The record naming does not do what D-16 says ("named from the tree's version").
- Proposed fix to the plan:
  - Q7.7 derives the record name from the release in flight, read from a tracked source, not `git describe`, and its test injects the tag and version.
  - Q7.6 states its diff base and what it does in a depth-1 clone (fetch the base, or run locally only, like `verify-docs`).
  - Move Q7.7 to B1 with the other DEF rows.

### DV-5 [Important]: P10 runs only at phase exit, and Q6.3 and Q6.4 change what it checks
- Plan task(s): step 4 of the task order, Q3.1, Q6.3, Q6.4, Q9.7.
- Observed:
  - `p10` is in `quality` only, not in `verify-gates` (`Makefile:298`, `:324`); the checker lives outside the tree.
  - **Q6.3's renamed file.** The ledger row `app/director.go | awaitAir` (`06_docs/p10-ledger.md:64`) is HUM-LEAD-ratified, and earlier re-keys of such rows went back to the lead ("RE-KEYED … RATIFIED by the HUM LEAD 2026-10-03", `:50`).
  - **Q6.3's other misses.** `app/testdata/declset.txt:150-…` lists `(*director).Run`, `.admit` and the rest. Neither the ledger row nor the declaration set is in Q6.3's list.
  - **Q3.1's stale rationale.** The ratified `Router.Update` / `Router.update` rows argue from `r.surface()`. Q3.1 removes its call in `View` (`router.go:525`), leaving `Init` (`:316`) as its only caller.
  - **D-14's promise.** Q6.4's "keyed literal … inside P10" (P10-04) is promised, but checked only at BUILD exit.
- Consequence: the BUILD-exit `make p10` fails on an unmatched row and a live finding. A ratification then has to be asked for at the phase gate, after every batch has landed.
- Proposed fix to the plan:
  - Add `make p10` to step 4 for every batch that adds, renames or reshapes functions (B6, B11, B12, B14).
  - Add to Q6.3: the ledger re-key, put to the HUM LEAD, and the declaration-set re-capture.
  - Add a standing line to step 4: re-capture the declaration set in a guarded package (app, render, snapshot, history, nws) when its declarations change.

### DV-6 [Important]: Q1.2 changes the console's clock, which no ruling covers
- Plan task(s): Q1.2.
- Observed:
  - D-5 rules on "DEF-2: the Broadcaster's location pool shows °F and miles whatever the units Setting". It says nothing about the clock.
  - Q1.2 says "`Broadcaster.opts()` carries Units **and Clock**". Today the console's `Opts` has a zero `Clock`, which is `Clock12` (`platform/render/clock.go:26`).
  - `Broadcaster.header` draws `mastheadStamp(o, …)` (`broadcaster_header.go:28`). That calls `o.Clock.Stamp` / `TimeSec` (`masthead.go:53-54`).
  - So a 24-hour or military listener's console masthead changes on carrying Clock. Q1.2's pin checks only the pool's units.
- Consequence: R1 stops batch 1, the batch D-5 fixed, for a ruling mid-BUILD. Otherwise the change ships unpinned and unruled.
- Proposed fix to the plan: get the clock half ruled at the PLAN gate (a D-row, or an A-row citing FR-6.2), and add a pin on the console masthead stamp under `Clock24`.

### DV-7 [Minor]: Findings and rulings carried only partly
- Plan task(s): Q1.3, Q2.10, Q2.12, Q4.5, Q5.4, Q5.6, Q6.1, Q8.2, Q9.2, Q9.9.
- Observed:
  - **DR-2:** only `tempWords` (Q1.3) is carried. "`conditions` calls `windWords`", "`depthWords` calls `distanceWords`" and "one file" are in no task, and Q5.5 lists only DR-8 to DR-14 and DR-16. M3 needs every DR finding landed or ruled.
  - **DR-7(a):** the third arithmetic form, `domains/severe/record.go:41` `compass`, is not named in Q5.4.
  - **DC-4:** after the move, "table.go: the ONLY go-studs consumer" is still false (`lineup_table.go` and `status_table.go` import it too), and Q6.1 does not fix the wording.
  - **DC-10:** the "0.16.5" mentions in `code-standards.md:5`, `perf-measurement.md:202-209` and `tooling-backlog.md:39,40,78` are carried by no task; Q8.2 covers `follow-ups.md` only.
  - **JR-6:** 19 comments cite "architecture §n" with no path, and `extending.md:173`'s ambiguous §1.1 is not carried.
  - **RN minor list:** Q2.12 omits `app/stats.go:210-231` `sources`, `app/voices.go:160` `PreviewVoice` and the duplicated `centre` closure.
  - **Q2 is not tests-only.** Q2.12's sweep clock needs a production change (`platform/history/prune.go:357-363` reads `time.Since`), and so does Q2.10: the publisher reads `time.Now()` at `app/pipelines.go:102` and `:308`. Also, F-51 is already closed ("FIXED 2026-09-06 without … adding a clock"), so Q2.10 "joins" a row that Q8.2 deletes.
  - **PF minor list:** five items have no disposition (`cfg` held by value, `radioLines`, `mapModalKey`, `statsFingerprint`, the engine poll).
  - **JR minor list:** Q9.9 does not say whether the 77 one-letter fields and the 59 "0.x.y:" stamps are in scope.
- Consequence: M3 and M4 cannot be signed off at REVIEW, and the group tasks are too vague to execute or review.
- Proposed fix to the plan: name each item in its task, or record it as declined with a reason (an A-row where it is a choice).

### DV-8 [Minor]: Duplicate and overlapping edits
- Plan task(s): Q4.2, Q4.3, Q4.5, Q5.6, Q6.4, Q7.2, Q7.3, Q7.7, Q8.7, Q9.1, Q9.3.
- Observed:
  - **Package docs, twice.** Q8.7 fixes the `modes/tty` and `app` package docs (DC-11: `dashboard.go:1-11`, `app/app.go:1-2`), and then Q9.1 replaces them with `doc.go`. Q9.1 also does not say the old package comments are deleted.
  - **"Twenty", twice.** Q8.7 (DC-15) and Q9.3 both fix the "twenty" count in `setup_rows.go`.
  - **The stale ledger row, twice.** Q5.6 (DR minor list) and Q7.2 both delete row `7dbf0b6d0c66`.
  - **The Makefile's comments, three times:** Q7.3 (narrations at `:157`, `:354`, `:467-470`, `:583`), Q7.7 (`:236-256`, `:376-382`) and Q8.7 (`:80-88`, `:141-193`, `:343-347`, `:470-474`).
  - **The constructor line.** Q4.2 and Q4.3 each add a memo pointer to the 1,410-character literal (`dashboard.go:1000`, beside `memo: &bodyMemo{}`), which Q6.4 then rewrites.
  - **A subsumed task.** Q4.5's "`tickerLoopLen` reads the frame's bullet" disappears with Q4.3 (the lens says so).
  - **No line conflicts where I checked:** Q3.4's `clamp` (`broadcaster.go:681-752`) against Q6.1's `cardLane` (`:1348-1441`) and Q6.2's `boxRule` (`:1665`); Q3.1's router and `dispatch` against Q6.3 (app) and Q6.4 (`dashboard.go:958-1000`).
- Consequence: each piece is small, but each costs a second review of the same lines.
- Proposed fix to the plan:
  - Give each overlap one owner: Q9.1 owns the two package docs, Q8.7 owns DC-15, and Q7.2 owns the ledger row.
  - Do all the Makefile comment work in one Q7 task.
  - Run Q6.4 before Q4, or note that the memo fields go into the residual literal.

### DV-9 [Minor]: The HUM LEAD's steps sit late, behind work that needs their answers
- Plan task(s): Q9.4, Q6.2, Q8.3, and D-2 (which has no task).
- Observed:
  - **Q9.4's sample comes last.** The `stationLine` sample is put to the lead in the last work package. Two of JR-8's seven functions are rewritten long before: `Broadcaster.clamp` (Q3.4) and `Router.Update` (Q3.1). Each must carry its essays through the rewrite, and Q9.4 then edits them a second time.
  - **D-18's look comes after the build.** It is shown only after Q6.2 lands. A "no" from the lead reverts Q6.1's render function as well.
  - **The screenshots have no "after" condition.** A-9's retakes are "before SHIP", but nothing says "after Q6.2". `broadcaster-settings.png` should show the new card rule.
  - **D-2 is silent.** No lens finding lands in go-tuiMaps or go-ionomaps (PF: "Every finding sits in watchpost's own code"), and the plan does not say so.
- Consequence: the lead's answers arrive after the work they shape, so finished work is redone.
- Proposed fix to the plan:
  - Show the `stationLine` sample at B2, so B4 and B6 use the approved shape.
  - Show a scratch-rendered D-18 golden diff before B10.
  - Ask for the screenshots after B10.
  - State "D-2: no library patch needed".

### DV-10 [Minor]: Q2.8's 80x24 console golden is the TOO SMALL notice, and 160x74 is not captured
- Plan task(s): Q2.8; Q1.2, Q3.4, Q6.1 and Q6.2 rely on it.
- Observed: the console refuses below 100 columns or 44 rows (`bcMinCols = 100`, `bcMinRows = 44`, `broadcaster.go:620-660`). Q2.8's 80x24 console frames therefore pin only the notice. The width M2 measures, and the mock's size (160x74), have no golden.
- Consequence: the console's net for Q6's moves is one size, 133x44.
- Proposed fix to the plan: capture the console at 133x44 and 160x74; keep 80x24 for the other surfaces.

### DV-11 [Minor]: PF-9's and PF-10's gains are not visible to any pin
- Plan task(s): Q4.1, Q4.4, and M2's "frame pins lowered".
- Observed: the beat pin measures a memo hit with no publish, so PF-9 (miss avoidance) and PF-10 (assembler copies) move none of its figures. `BenchmarkAssembler25Locations` (998 KB a publish) has no budget.
- Consequence: their savings can regress silently after landing.
- Proposed fix to the plan: add a bytes pin on a no-change publish's frame, and one on the assembler's publish.

## What the plan gets right
- **Coverage is nearly complete.** Every RN, PF, DR, DC and JR finding has a task, and so does each DEF row (DEF-1 Q1.1, DEF-2 Q1.2, DEF-3 Q1.3, DEF-4 Q7.7, DEF-5 Q1.6). Every ruling D-5 to D-18 and A-2 to A-11 is carried. The gaps are the partial carries and vague groups in DV-7.
- **The order is right.** The regression net comes before tidying, the checks before the docs passes, and the console goldens (Q2.8) before the card lane and rule moves (Q6.1, Q6.2).
- **M2 is held, not just measured.** Each memo needs a completeness guard, pins are lowered to what lands, and "a miss goes to the HUM LEAD, targets not quietly moved" is the right instinct; DV-1 only moves it earlier.
- **The house rules are written in:** restore by copy, read MAKE_EXIT, push only when no hosted run is in progress, and the record moves with the code.

## What I did not check
- The regression-risk depth of each pin, and whether each mutant would be killed (the RG lens's job).
- The readability of the result (the JR lens's job).
- Whether hosted CI really takes about 2.5 h per push, or how `mutant-policy` affects it; I took the brief's figure.
- The P10 thresholds themselves (the checker is outside the tree), so whether Q6.4's helpers fit P10-04 is unverified.
- The DV-1 savings are taken from profile flat and cumulative bytes on the lens's prototype. PF-1 M, PF-7 and PF-8 were not prototyped, so their ranges are estimates.
