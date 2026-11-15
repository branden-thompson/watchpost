---
title: "0.19.6 DISCOVER exit red team — DQ, docs quality"
date: 2026-11-15
phase: DISCOVER
sev: SEV-0
status: "Filed whole as returned; dispositioned in ../red-team-review.md"
---

# 0.19.6 DISCOVER exit red team: Docs Quality axis

**Tree:** `feature/re-architecting` at `66d8a350`; the code is the same as `e57afeaf`. I worked read-only. My scratch files are in `<scratch>/`.

**Checked against the tree and found correct:**
- Package sizes: 31,346 lines in 88 files, 20,184 in 76, and `platform/render` at 5,989.
- `Dashboard` has 752 methods in 68 files; `livePipelines` has 108 in 21; `Broadcaster` has 92.
- 363 test files, 362 of them white-box.
- Mutant targets: 539 in `modes/tty`, 153 in `app`; 289 in `map_*` and 80 in `broadcaster*`.
- Hub fan-in is 71 of 87, and the map reads 72 symbols (from the graph TSVs).
- The baseline sweep: 1,051 caught, 7 survived, 0 no evidence, 3,976 s, the baseline 138 s.
- Mutant-seconds: 8,013 and 3,213 out of 14,101.
- The four `app` packages sum to exactly 76 files and 20,184 lines.
- Every code line cited in package-map §2 and §8 except one (DQ-7). D-10, D-11 and D-12 citations sampled. F-158's code claims hold.
- Targeted doc-to-code tests ran green in `./cmd/watchpost` (`TestWhereThingsHappenNamesRealSymbols`, `TestEveryFileIsInItsPackagesMap`, `TestEveryCodeCitationResolves`, `TestEveryRulingNamingCodeCitesItsLine`, `TestFollowUpsCiteTestsThatExist`, `TestEveryRatifiedP10RowNamesCodeThatExists`, `TestTheAtlasMatchesAFreshRender`).

---

## 1. Does each document answer the question its title asks?

Yes for the required reading, the rulings, the brief and all nine lenses. Each lens answers its stated question and lists what it checked. The package map answers "what is the shape" (§2 to §6), with two gaps.

**DQ-1. §7 is titled "the questions for the HUM LEAD, in order" but starts at 5.** There has never been a 1 to 4 (the same since `b69a4773`). A reader cannot tell whether questions were lost.
- Severity: Minor
- Evidence: `package-map.md:163-174`
- Simplify/Delete? Y
- Action: number from 1, or say that 1 to 4 were D-1 to D-4.

**DQ-1b. RM slice 4 is unclear: done or still owed?** The brief lists "RM-1's table recorded" as work in scope. The package map only points at the table, for 0.19.7.
- Severity: Minor
- Evidence: `project-brief.md:75-76`; `package-map.md:181-182`; `lens-RM-role-model.md:438-439`
- Simplify/Delete? Y
- Action: say slice 4 is closed by the pointer, or copy RM-1's table into the map.

## 2. Does anything contradict the code, another record document, or a document elsewhere?

**DQ-2. The package map's status contradicts itself.** Its front matter says "PROPOSED … each fork below is put to the HUM LEAD one at a time". Its own body (line 12) and the required reading call it "the settled document every later finding is judged against". The approved brief takes its scope from map §2. No row records that the map itself was approved; D-13 approves the brief.
- Severity: Important
- Evidence: `package-map.md:7`, `:12`; `00-REQUIRED-READING.md:26`; `project-brief.md:6`, `:72-74`; `rulings.md:36`
- Simplify/Delete? N
- Action: an A-row ("the map is settled as the scope D-13 approved"), with the status changed to match, or put the map to the HUM LEAD.

**DQ-3. The map says its choices are A-rows put for veto, and those rows do not exist.** The claim covers "names, the alias rule, `internal/`, the lock decision, the comparator, the ratchet".
- `rulings.md` has only A-1 to A-8, and none of them is one of these. D-13 put "A-1 to A-8" for veto.
- C-12 also replaces 0.19.5 A-14's re-vet by hand. That too is unrecorded.
- C-5, C-6, C-7 and C-8 settle questions the lenses addressed to the HUM LEAD, not disagreements between lenses (the D-7 class):
  - TG questions 2 and 3;
  - AS question 3 ("the HUM LEAD may want it ruled rather than taken as an A-row");
  - PB question 1.
- Two questions are dispositioned nowhere: TG question 4 (the allowed-edge ledger as a required gate) and PB question 2 (a beat wall-time benchmark in the tree).

- Severity: Important
- Evidence: `package-map.md:176-177`, `:98`; `rulings.md:44-52`, `:36`; `lens-TG-tooling-gates.md:474-481`; `lens-AS-app-seams.md` (Open questions 3); `lens-PB-performance-baseline.md` (Open questions 1-2)
- Simplify/Delete? N
- Action: write A-9 onward for C-4 to C-8, C-11, C-12 and the two package names. List TG question 4 and PB question 2 as dispositioned or still open.

**DQ-4. The order of work moves the docs to the end, against three other rules.** §6 item 6 says "The closing sweep, the docs and the atlas, REVIEW". That contradicts:
- R6, every batch redraws its diagrams in the same commit;
- the map's own §5, "the atlas redrawn in the same commit";
- RC-6, "redrawn in the batch, not at REFLECT".
- Severity: Important
- Evidence: `package-map.md:158`, `:140`; `project-brief.md:66-67`; `lens-RC-the-record.md:289-291`
- Simplify/Delete? Y
- Action: rewrite item 6 as "the closing sweep, REVIEW".

**DQ-5. The rule for fixing known doc errors leaves the largest one with no batch.** The rule is "fixed in the batch that moves the code they describe" (map §8, A-8, brief scope).
- RC-7 item 1 is the error: windows are a stack, three documents say one window. Its code does not move: §2 keeps "the window stack" at the root (`modes/tty/dashboard.go:1591-1596`).
- So these lines are never in a moving batch:
  - `docs/where-things-happen.md:15` and `:95`;
  - `docs/extending.md:291-294`, which also says the test "proves nothing stacks";
  - `modes/tty/dashboard.go:1590`, "open shows m alone".
- The same applies to RC-7 item 5 (`releaseRecord`) and to JN-5's `modes/tty/dashboard.go:641` and `where-things-happen.md:128`.
- RC recommended fixing these in the first docs batch, before PLAN cites them. The map changed that without giving a reason, which D-7 requires.
- Severity: Important
- Evidence: `package-map.md:77`, `:183-185`; `rulings.md:52`; `project-brief.md:76-78`; `lens-RC-the-record.md:296-306`, `:331-332`
- Simplify/Delete? N
- Action: put RC-7 items 1 to 4 and 6, and JN-5's two lines, into RM slice 3's step, before any move.

**DQ-6. "Imports one way" is stated wrongly and nothing enforces it.** The map says "`deck` and `maps` → `kit` only (AS-9)".
- Both import `modes/tty` today: `app/mastercontrol.go:467` (`tty.TickerBreakingMsg`), and `app/maps.go`, `maplayers.go` and `history.go` all import `modes/tty`. AS-9 itself counts deck 199 and maps 548 uses of `tty`. A junior reading the map's sentence is misled.
- AS-9 says nothing in the compiler stops `deck` and `maps` importing each other, so a `lint-imports` row is needed. Batch 0 item 9 adds only `modes`/`platform`/`domains` ↛ `app` and `platform` ↛ `modes`.
- Severity: Important
- Evidence: `package-map.md:69`, `:126-128`; `lens-AS-app-seams.md:430-432`
- Simplify/Delete? N
- Action: say "among `app`'s children". Add deck ↛ maps and maps ↛ deck to item 9, or adopt TG-1's allowed-edge ledger.

**DQ-7. One citation points at the wrong file.** The map cites `docs/architecture.md:528`, but `docs/architecture.md` has 111 lines. The text quoted is in `06_docs/02_features/watchpost-cli/03-architecture-design/architecture.md:528`; GP cites it correctly. That disagreement between doc and code (Bubbles listed as built) is also missing from A-8's list of fixes.
- Severity: Minor
- Evidence: `package-map.md:187`; `lens-GP-go-practice.md:379`
- Simplify/Delete? Y
- Action: correct the path and give the fix a batch.

**DQ-8. Figures that disagree with the tree or with each other.** One correction row would cover all six:
- `package-map.md:25` says "705 of 1,097" tests. The tree has 1,085 `func Test` in `modes/tty`. TG says 1,085, PB ran 1,083, RX says 1,097. For `app` the figures are 687 in the tree, 689 (RX) and 680 (PB).
- `lens-RC-the-record.md:265` still says "255 anchors (145 tty, 110 app)". A-6 superseded that figure, yet the file is marked "findings checked".
- `lens-RC-the-record.md:257` says 183 "`…go:Func` spans". There are 168 (TG-10 is right). 183 counts every path mention.
- `lens-JN-junior-navigation.md:63` says `Broadcaster` has 90 methods. The map and grep say 92.
- D-10 says "16 ratified rows (`:18`, `:49-56`, `:63`, `:172`, `:190`, `:212-213`)", but that list is 14 rows. The 16 needs `:122` and `:129`, which RC-4 includes. "Seven (`:49-56`)" is eight lines.
- `package-map.md:160-161` says "the deck moved whole keeps 64 of its 101 killers". That uses RX's rougher cluster (deck, arbiter and executors), not the map's own deck (112 mutants, `:36-37`).

- Severity: Minor
- Evidence: as listed; `rulings.md:33`, `:50`; `lens-RC-the-record.md:205-206`; `lens-RX-regression-net.md:136`
- Simplify/Delete? Y
- Action: one A-row of corrections.

## 3. Is something claimed as verified that nothing verifies?

**DQ-9. The release's baseline sweep record is not committed.** A-3 says it was "Promoted to `06_docs/mutants/verdicts/since-v0.19.5.log`", and the required reading states the result as fact. `git status` shows the file untracked (`??`).
- Map §4 item 2 plans to commit it in batch 0, but nothing warns the reader that the record the whole release is judged against exists on one disk only.
- The 57 % and 23 % figures in D-1 and the brief come from `dist/mutant-verdicts.timings`. `.gitignore:1` ignores `dist/`, and the next sweep overwrites the file.
- Severity: Important
- Evidence: `rulings.md:47`, `:24`; `00-REQUIRED-READING.md:24`; `project-brief.md:24-25`; `lens-RX-regression-net.md` RX-12
- Simplify/Delete? N
- Action: commit the log now (a `.log` file takes the full gate). Or have A-3 say it stays uncommitted until batch 0, and copy the per-package seconds into a committed record with their source.

**DQ-10. M3's instrument and its baseline exist only in the session scratchpad.** M3 is the brief's main measure of principle 2. Its graph tool stays outside the tree during DISCOVER (A-4), and M3 is to be "re-run on each batch's layout".
- No batch 0 item commits the tool or its `e57afeaf` output. The tool lives in a temporary directory.
- PB's beat-time probe is in scratch too. Item 7 covers only the sizes and escapes.
- Severity: Important
- Evidence: `rulings.md:48`; `project-brief.md:48`; `package-map.md:101-133`
- Simplify/Delete? N
- Action: a batch 0 item that commits the graph tool and its baseline TSVs, or re-bases M3 on an instrument that is in the tree.

## 4. Audience order: designers, then PMs, then engineers?

**DQ-11. The brief puts engineering figures before the one fact a designer or PM needs.**
- It opens with method counts, fan-in and share of the sweep (`project-brief.md:13-25`).
- The user-facing fact, that nothing the listener sees or hears changes, is R1 at `:56`. It comes after a seven-row metrics table.
- That table uses IDs the brief never defines: PB-1, PB-7, JN P2-5, C-8, "W6 and W8", "the 7 equivalents".
- The package map is written for engineers throughout.
- The HUM LEAD could rule go or no-go (D-9 to D-13 show it). Judging M3 and M5 needs JN-8.
- Read order is the HUM LEAD's call, so this is a fork:
  - (a) leave it as it is;
  - (b) add three lines above the problem: what users notice (nothing, R1), what is ruled at PLAN (M3 targets, the re-key table), and what is out. Define W6, W8 and the P2 IDs where they appear.

  Evidence favours (b).
- Severity: Minor
- Evidence: `project-brief.md:13-25`, `:46-52`, `:56`
- Simplify/Delete? N
- Action: present the fork to the HUM LEAD.

**DQ-12. The brief says every metric is measured each batch, but one is not.** `project-brief.md:41` says "Each metric is measured at the end of every batch". M4 is a REVIEW gate (`:49`).
- Severity: Minor
- Simplify/Delete? Y
- Action: name M4 as the exception.

## 5. Is a number published without its blind spots beside it (INST-5)?

**DQ-13. M1's "any rise in a count fails" ignores noise PB already recorded, and its baseline is ambiguous.**
- PB recorded "a one-allocation flicker on the map, severe and Propagation windows": map 3,094 to 3,095, severe 5,068 to 5,069, Propagation 1,209 to 1,210. A comparator built exactly as C-8 is worded fails on noise, and that pushes people to re-record or loosen it.
- M1 lists pinned figures (console 122,387 B, publish 82,322 B). Today's measurements are 122,399 to 122,405 and 82,335 to 82,340. PB-1 calls these "equal". The brief does not say which number the comparator records.
- Severity: Important
- Evidence: `project-brief.md:46`; `package-map.md:94`; `lens-PB-performance-baseline.md:95-96`, `:142`, table A
- Simplify/Delete? N
- Action: state the flicker beside M1. Define the baseline as the maximum over N runs, with the window pins' ±1 stated.

**DQ-14. M3 can be gamed in the ways JN warned of, because the brief dropped JN's guards.** M3 publishes targets ("each falls from its baseline": 68 files, 82 %, map 72, Settings 102). JN paired each criterion with a guard. M5 cites the P2 IDs but carries none of these:
- P2-2: measure by type, not by file name, and count reach through accessors;
- P2-3: count fields reached through a parameter;
- P2-6: the leave-one-out check;
- P2-7: the count of interfaces with one implementation;
- P2-9: the cap on files per row.

The 82 % is a ratio, and the split shrinks its denominator: the root loses the console, the map and the kit. It can rise under a good split, and fall under a bad one.
- Severity: Important
- Evidence: `project-brief.md:48`, `:50`; `lens-JN-junior-navigation.md:321-328`
- Simplify/Delete? N
- Action: carry the guards into M3 and M5 when PLAN fixes the targets. Measure the hub by type.

**DQ-15. The 57 % and 23 % in D-1 and the brief do not say what they are shares of.** They are shares of 14,101 mutant-seconds summed over 4 workers. The same brief quotes 3,976 s of wall time in M2.
- Severity: Minor
- Evidence: `project-brief.md:24-25`, `:47`; `lens-PB-performance-baseline.md:129`
- Simplify/Delete? Y
- Action: name the basis.

**DQ-16. The graph's blind spots are not stated where its figures are used.**
- The graph omits files with build tags: `app/dump_windows.go`, `modes/tty/map_property_full_test.go` and the race-tagged tests. RX notes this; A-4 and M3 do not.
- JN-6's "all 200 test files" counts only files that reach an unexported symbol. `modes/tty` has 213 test files.
- Severity: Minor
- Evidence: `lens-RX-regression-net.md:325`; `lens-JN-junior-navigation.md:260`; `rulings.md:48`
- Simplify/Delete? Y
- Action: add the blind spots to A-4 and M3.

## 6. Does the record list everything the moves invalidate, and which batch fixes each?

RC-6 is a good inventory and checks out against the tree. But nothing in the brief or the map gives any item a batch.

**DQ-17. No batch owns these items, and several are not mentioned at all:**
- **0.19.6's own as-built page, and its row in `TestTheAsBuiltMapIsKeptInStep`.** The test knows only two features (`app/maps_test.go:62-65`). It moves with the maps package, since it reads `../06_docs` (`:57`). `tools/prepush/main.go:28` names `./app` for it.
- **`releaseRecord`** (`cmd/watchpost/record_test.go:20`). Re-pointing it is not one line. I checked this by reading, not by running:
  - `readRecord` stops with `t.Fatal` when a file is missing (`:137-139`). The 0.19.6 folder has no `01-objectives/requirements.md`, `03-architecture-design/architecture.md` or `04-development/implementation-plan.md` (`:271-275`).
  - `TestEveryRulingNamingCodeCitesItsLine` needs at least 50 Go citations in the rulings (`:177`). 0.19.6's rulings have 6.
  - Result: the brief, the map and the coming PLAN are held by no record check.
- **Eight atlas diagrams name the two packages:** 1 in `docs/architecture.md`, 3 in `as-built-air-and-deck.md`, 1 in `map-ready-geometry/…/as-built.md`, 3 in `as-built-map.md`. Only `as-built-map.md`'s batch count is tested.
- **Prose no test holds:**
  - `README.md:597` and `Makefile:35-41` ("modes/tty, the slowest package");
  - `docs/extending.md:257-262`, the per-package golden and declset flags;
  - the specimens in `06_docs/code-standards.md`;
  - `quality-plan.md:151` and `perf-measurement.md`;
  - the 0.16.0 required reading's paths;
  - `follow-ups.md`: F-158 itself names `app/maintrack.go` and `app/arbiter.go`, both moving to the deck (`follow-ups.md:90`). Only test names are checked there (`docs_test.go:188`);
  - 49 Go comments in 35 files that name these paths, 15 of the files outside the two packages (`lens-RC-the-record.md:279`).
- **RC-5's documentation half.** `where-things-happen.md:147-149` and `06_docs/ids.md` (generated by `tools/ids`) tell the reader to use "`git log` on the file". After a move that needs `--follow`. Batch 0 item 10 fixes only the tool.
- **TG-10's silent base-name match** (`extending_symbols_test.go:57`). The split adds duplicate basenames (`doc.go`, `kit`).
- **No rule says which documents are living and which are history.** 231 tracked documents name a file in the two packages; `observer-maps/04-development/implementation-plan.md` alone has 268 references.

- Severity: Important
- Evidence: as listed; `package-map.md:101-161`; `project-brief.md:66-67`, `:76-78`; `lens-RC-the-record.md:249-292`
- Simplify/Delete? N
- Action: add a map §9, "What the moves invalidate". One table, one row per document, diagram or check, saying whether a test holds it and which batch fixes it, plus a one-line rule for living versus history.

**DQ-18. "kit" is a poor name, by the record's own rules.**
- "The kit" already means go-studs and the render table kit: `docs/extending.md:281`, `:295`; `README.md:592`; `cmd/watchpost/docs_test.go:209` ("the vendored kit"); `modes/tty/broadcaster.go:1268`.
- The map creates two packages named `kit`. That is the duplicate-basename defect JN-5 counts, and M5 requires "the user's or the role's word".
- `app/internal/kit`, at 217 lines, falls under P2-1's rule that a package under 300 lines needs a stated reason.
- Severity: Minor
- Evidence: `package-map.md:49`, `:67`; `project-brief.md:50`, `:72-74`; `lens-JN-junior-navigation.md:320`
- Simplify/Delete? Y
- Action: name both packages in an A-row before PLAN.

**DQ-19. The settled map cites code by line number.** Its citations (`:78`, `:184`) will drift in batch 0. The repo's own rule for living documents rejects that: "lines drift as the code moves, so a living document cites the symbol" (`cmd/watchpost/record_test.go:248-252`).
- Severity: Minor
- Simplify/Delete? Y
- Action: cite symbols, for example `dashboard.go:update` and `dashboard.go:open`.

---

**DO NOT EXIT DISCOVER.** No new survey is needed. Every fix is one docs commit: DQ-2 and DQ-3 (the map's status and its missing A-rows), DQ-4, DQ-5 and DQ-17 (the invalidation table, with a batch for each item), and DQ-9 (commit the baseline record, or say it is uncommitted). The lenses are sound and their figures check out. What a PLAN author cannot yet rely on is the settled document's own status and its list of what it breaks.

**Fix first: DQ-17, with DQ-4 and DQ-5.** PLAN's batches are drawn from this list. As written, the window-stack error and the release's own record checks never fall due in any batch, and the docs are scheduled for REVIEW.
