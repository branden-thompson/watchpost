# Plan red team: Project Hygiene (PH), 0.19.6 PLAN at `fbbfeb1a`

I worked read-only in `<scratch>/rtp-ph/`. The commands I ran: `git` (read-only), `grep`, `sed`, and one `go build ./cmd/watchpost` into scratch followed by `--version`. I ran no tests, no make targets and nothing over the network.

## 1. Is anything in the tree that should not be?

The branch is clean. Since `e57afeaf` the diff touches only `06_docs/`. No binary, secret or scratch file is tracked. `geodata.test` and the `refs/original` ref are gone, as D-19 says. The only local branches are `main` and `feature/re-architecting`, and there are no stashes or extra worktrees. I found no absolute or user path in the diff: the lenses and designs write `<scratch>`/`<scratchpad>`.

**F-1. Nothing in the clone shows that the branch exists on origin.** D-8 says the branch was "pushed with a lease on `d43faa30`" (`02-analysis/rulings.md:31`). Locally `feature/re-architecting` has no upstream (`git config` lists only `branch.main.*`) and there is no `refs/remotes/origin/feature/re-architecting`. Fetches in this clone go by URL (the reflog of `refs/remotes/origin/main`), and pushes apparently do too. That is also why `origin/main` still reads `b4ab7dfb`, two PRs behind `e57afeaf`. Someone inheriting the repository cannot tell from the clone what is published. UNVERIFIED, because I made no network call.
- Severity: Minor.
- Evidence: `02-analysis/rulings.md:31`; local refs.
- Simplify/Delete? N.
- Action: set the branch's upstream, or record the push's remote ref in the build log.

**F-2. D-8's owed purge of `1f9a34e7` has no follow-up row.** D-8 leaves "a purge before then is a request to GitHub support, the HUM LEAD's to make" (`rulings.md:31`). That commit names the machine's user and is still served by its hash. Nothing in `06_docs/follow-ups.md` tracks whether the request was made. The hash appears only in `rulings.md`, `ids.md` and `verify-runs.md`.
- Severity: Minor.
- Evidence: `rulings.md:31`.
- Simplify/Delete? N.
- Action: add one follow-up row naming the HUM LEAD's action and its deadline (when GitHub collects the object).

## 2. Does the build work from a clean clone, with the documented commands?

The binary builds and prints `watchpost version 0.0.0-dev`. There is no `go.work` and no `replace`. The plan's own derivations, though, do not survive a clean clone.

**F-3. The scripts behind the plan's batch lists are not in the tree, and one of them is a BUILD step.**
- B4 says "2b's class C4 by `tests_fn_ms.py` on 2a's code" (`04-development/implementation-plan.md:212`). That script exists only in the session scratch (`<scratch>/plan-MS/tests_fn_ms.py`), and `git ls-files` has no match for it.
- The CK design states "Everything is in `design-data/CK/`. Every script runs offline" (`design-CK-console-kit.md:48`). It then lists `declpos/`, `tests_ck.py`, `tests_fn.py`, `classify_ck.py`, `probe/`, `cmdname/` and `m-base.txt` (`:50-63`). `design-data/CK/` holds five TSVs and nothing else.
- CK-6's confidence line says the names were "printed by `design-data/CK/cmdname`" (`:620`), which is not a path in the tree.
- MS and AP likewise cite `classify_crossings.py`, `methods_ms.py`, `aptool/`, `closure.py` and others, none of them committed.
- B2 and B3 take `kit-CK.tsv`, `tests-CK-refs.tsv` and `tests-CK.tsv` as "the batch's lists" (`implementation-plan.md:181`, `:190`). Nobody can regenerate them once the session ends.

The record states an intention ("Everything is in"), not what was filed.
- Severity: Important.
- Evidence: `implementation-plan.md:212`; `design-CK-console-kit.md:48`, `:62`, `:620`.
- Simplify/Delete? N.
- Action: commit the generating scripts beside their TSVs under `design-data/`, or move B4's re-measure onto B0.12's committed graph tool. Correct `:48` and `:620` to say what is actually there.

**F-4. M3's targets would be ratified on figures from a tool nobody can run.** The targets 72→23, 93→40 and 752/68→374/38 (`implementation-plan.md:277-281`) come from A-4's scratch graph. The committed tool arrives in B0.12 (`:146-148`), after PLAN exit. The plan has no step that checks the committed tool reproduces the baselines on `e57afeaf` before targets fixed on the scratch tool's numbers bind BUILD. MS's banner even says the graph "is committed in batch 0" (`design-MS-map-seam.md:11-13`).
- Severity: Important.
- Evidence: `implementation-plan.md:146-148`, `:277-281`.
- Simplify/Delete? N.
- Action: add a B0.12 acceptance line saying the committed tool reproduces 72, 93 and 752-in-68 on `e57afeaf`, and that any difference sends the targets back to the HUM LEAD.

**F-5. The P10 re-key table is keyed by line numbers of a generated mirror, and it never names the ledger the gate reads.** `06_docs/p10-ledger.md` is regenerated from a machine-local, git-ignored ledger (`scripts/quality/p10-ledger-mirror.py`, header; `Makefile:540-549`), and `make p10` fails if the mirror differs from the committed copy. The table cites `:18`, `:49`-`:56`, `:63`, `:122`, `:129`, `:172` and `:190` (`implementation-plan.md:285-296`), and I checked that each row is there today. B2, however, adds three rows (D-15, D-23), which shifts every later line before B3, B5 and B6 apply their re-keys. The plan's move commit edits "P10 and baseline re-keys" (`:33`) without saying that the edit goes into the local ledger first.
- Severity: Minor.
- Evidence: `implementation-plan.md:285-296`; `Makefile:540-549`.
- Simplify/Delete? Y: key the rows by file and symbol, not by line.
- Action: key the table by file and symbol, and add one sentence saying re-keys are written to the local ledger and the mirror is then regenerated.

**F-6. benchstat is mandatory in the plan but installed `@latest`.** The plan makes benchstat mandatory (`implementation-plan.md:39-40`, `:213`, `:256`), while the Makefile installs it with "`go install golang.org/x/perf/cmd/benchstat@latest`" (`Makefile:529`). That is the same unpinned-tool pattern IS-M10 removed for govulncheck (`Makefile:70-75`).
- Severity: Minor.
- Evidence: `Makefile:529`.
- Simplify/Delete? N.
- Action: pin a version in B0.7.

## 3. Are the gates the project claims to run actually run, on every path?

**F-7. B0.13 cannot pass as written, and nothing checks the 0.19.6 record today.** `releaseRecord` still names `propagation-overlays` (`cmd/watchpost/record_test.go:20`), so every record check reads 0.19.0's folder. The PLAN commit's green verify did not read a line of 0.19.6. If B0.13 simply points it at 0.19.6:
- `readRecord(t, "01-objectives/requirements.md")` calls `t.Fatal` (`record_test.go:135-141`). 0.19.6 has no `requirements.md`: its R1 to R9 live in the brief. That breaks `TestEveryRequirementsTestIsInAPlanTask` (`:147`), `TestEveryRequirementsTestExists` (`:162`), `TestEveryFR7RowNamesItsTest` (`:455`) and `TestEveryCodeCitationResolves` (`:270-276`).
- `TestEveryCodeCitationResolves` rejects every line citation in a living document as "(cite its symbol)" (`:244-252`). `architecture.md` has 3 and `implementation-plan.md` has 10.
- The plan's one symbol citation, `` `console/broadcaster.go:New` `` (`implementation-plan.md:198`), is not a path from the root.
- The test requires at least 10 symbol citations (`record_test.go:280`), and the record has 1.

Package map §9 already flagged that "nothing reads 0.19.6's record" (`package-map.md` §9, the release-record row). The plan turns that into a one-line re-point and does not mention the tests it breaks.
- Severity: Important. It is a guaranteed red verify in batch 0, and the record's citation discipline has no gate until it is fixed.
- Evidence: `record_test.go:20`, `:135-141`, `:244-252`, `:280`; `implementation-plan.md:149-151`, `:198`.
- Simplify/Delete? N.
- Action: give B0.13 its full scope. Either give each record test a 0.19.6 form (the brief in place of `requirements.md`, with the FR-7 test retired or kept for 0.19.0's folder), or keep the 0.19.0 checks on 0.19.0's folder and add 0.19.6 checks alongside. In the same commit, convert the 13 line citations in `architecture.md` and `implementation-plan.md` to symbol form.

**F-8. B0.4's check depends on helpers that B2 and B5 create.** B0.4 requires "every package under `modes/tty/` calls `ttytest.Main`, every package under `app/` calls `apptest`'s invariants helper", with a floor of 2 packages (`implementation-plan.md:107-110`). But:
- `ttytest`, `Main` included, is gathered in B2 (`:177-178`);
- `apptest` is created in B5 (`:220`);
- D-24 calls it "batch 0's invariants helper" (`rulings.md:47`);
- package map §4 item 4 puts "one test-support helper" in batch 0.

On today's tree, `modes/tty/main_test.go` and `app/platform_sim_test.go` call neither helper. So B0.4 cannot be "seen failing on a planted case, then passing on today's tree" (`implementation-plan.md:90-91`).
- Severity: Important.
- Evidence: `implementation-plan.md:107-110`, `:177-178`, `:220`; `rulings.md:47`.
- Simplify/Delete? N.
- Action: create the two helper packages, holding just `Main` and the invariants helper, in B0, and leave the rest of the helper gathering in B2 and B5. Or state B0.4's rule as "calls its tree's helper once that helper exists", with a floor that rises in B2 and B5.

**F-9. B0.5's golden gate fails on B0.6's own work.** B0.5 says "A golden gate fails when a file under `testdata/` differs from `e57afeaf` except as a pure rename" (`implementation-plan.md:115-116`). B0.6 re-records `modes/tty/testdata/router-trace.golden` in a new form, keyed by index with command shapes and no `%T` (`router_trace_test.go:42`, `:66`; `design-CK-console-kit.md` CK-6, "The new form, written in batch 0"). B0.6 also adds the `app` trace's golden. B0.4 and B0.11 add planted fixtures under `testdata/` (`implementation-plan.md:142`). The plan's own rule, "A golden that changes is a regression, never a ruling candidate" (`:20-21`), contradicts B0.6. The plan does not say what the gate's baseline is.
- Severity: Important.
- Evidence: `implementation-plan.md:20-21`, `:115-116`, `:117-123`, `:142`.
- Simplify/Delete? N.
- Action: baseline the golden gate on B0's closing commit, naming the goldens B0 re-records or adds, rather than on `e57afeaf`. Say whether a new file counts as a difference.

**F-10. The per-batch scoped sweep has no supported command.** Step 6 asks for "the batch's mutants, derived (A-11), run with A-3's killers" (`implementation-plan.md:46-48`). Here is what the existing paths do:
- `make mutant-verdicts` promotes every run. `promote-verdicts` refuses a run that judged fewer than all mutants ("a MUTANT_ONLY run?", `Makefile:415`).
- The target then exits before `cache-clean` (`Makefile:436-440`).
- The script's own header says a partial run is "for checking a few by hand… `make mutant-verdicts` refuses to promote a partial run… so it would only fail there. A partial run records no timings" (`scripts/quality/mutant-verdicts.sh:53-57`).

So each of the roughly 8 scoped sweeps either fails through make and leaves its build variants on disk, which the Makefile calls "a vector for bad results" (`Makefile:276-289`), or runs by hand with no cleanup and no record path.
- Severity: Important.
- Evidence: `Makefile:415`, `:436-440`; `mutant-verdicts.sh:53-57`; `implementation-plan.md:46-48`.
- Simplify/Delete? N.
- Action: add a scoped-sweep target in B0.2 that feeds the derived list, writes its own log path, never promotes, and runs `cache-clean`. Classify it under `gates_test.go`'s `unlisted` table.

**F-11. The plan does not say where its new instruments run, and that placement needs a ruling.**
- B0.12's graph tool is "run by a make target" (`implementation-plan.md:147`).
- The pin comparator, escape probe and inlining count (B0.7) are given no home.
- `TestEveryGateShapedTargetIsListedOrExempt` (`cmd/watchpost/gates_test.go:247`) and `TestTheThreeGateListsAgree` (`:187`) force each new target into `verify`, `ci.yml` and `required-gates.txt`, or into an exemption.
- An exemption "leaves this list by a HUM LEAD ruling" (`06_docs/required-gates.txt:11-12`).
- M1 and M3 are claimed "every batch" (`project-brief.md:62`), and the plan does not say whether CI runs them.
- Severity: Important.
- Evidence: `gates_test.go:187`, `:247`; `required-gates.txt:11-12`; `implementation-plan.md:124-128`, `:146-148`.
- Simplify/Delete? N.
- Action: give each B0 instrument a named home, either a gate list or an exemption row, and put any exemption to the HUM LEAD before B0.

**F-12. D-24's simulator proof is a probe that no batch runs.** D-24 rests on "`make test-platforms` runs over `./app/...`" (`rulings.md:47`; `implementation-plan.md:228-229`). The target is on no gate list: `gates_test.go:79` calls it "the manual probe". Step 5 (`implementation-plan.md:35-45`) never runs it. What actually happens is that CI's two-OS matrix (`ci.yml:35-37`) runs the real GOOS, and `asPlatform` walks the fallback matrix inside the tests, so the effect is small.
- Severity: Minor.
- Evidence: `gates_test.go:79`; `implementation-plan.md:35-45`.
- Simplify/Delete? N.
- Action: add `make test-platforms` to B6's gate line, since that is the batch that relies on it.

**F-13. benchstat is "mandatory" but has no fail condition.** The plan says "benchstat… mandatory at 2a and 2b" (`implementation-plan.md:39-40`), while the Makefile says "Wall-clock benchmarks: recorded, never gated" (`Makefile:528`) and exempts `quality-bench` as having "no pass condition" (`gates_test.go:76`). The plan does not say what result fails a batch. With other agents and timings sharing this machine, that matters.
- Severity: Minor.
- Evidence: as above.
- Simplify/Delete? N.
- Action: state the condition, for example a significant regression at p<0.05 goes to the HUM LEAD, or call it "recorded" as the brief's M2 does.

## 4. Has anything named as owed quietly stopped being true?

The follow-ups D-26 and A-22 promised are there as F-235 and F-236. F-158's owed test is B1.1.

**F-14. M4's owed guards and routes are carried by no batch.**
- `m4-before.md:73-79` names four missing guards: the Overlays menu's hand-listed rows (`map_menu_test.go:54-63`), `switchRow`'s fall-through (`map_boxes.go:958`), no golden drawing a legend row, and all twelve console goldens drawing `DATA PULL: —`. It says "Each becomes a test in the batch that moves its file, or a follow-up".
- `:68-70` promises that Walkthrough 6 covers a map Setting that is also an Overlays-menu row.
- `:72` promises a bare-D-ID comment pass in each batch.
- A grep of the plan finds none of these. B4 and B8 carry only JN-3's `doc.go` (`implementation-plan.md:257-258`), and no follow-up row exists for them.

M4 is a REVIEW gate that the docs decide (`m4-before.md:56-57`), so these are the release's own items.
- Severity: Important.
- Evidence: `m4-before.md:68-79`; `implementation-plan.md:247-258`.
- Simplify/Delete? N.
- Action: put each into its batch (B4 or B8 for the map items, B3 for the console goldens) or into a follow-up row, by name.

**F-15. RM slice 4 is in the brief but in no batch.** The brief's scope says "RM's slices 1 to 4… RM-1's table recorded (C-10)" (`project-brief.md:107-108`). The plan carries slices 1 and 3 (B1) and slice 2 (B6). Package map §8 records RM-1's table "for 0.19.7".
- Severity: Minor.
- Evidence: `project-brief.md:107-108`; `implementation-plan.md:153`, `:223`.
- Simplify/Delete? Y, if slice 4 is already done in `lens-RM`.
- Action: say where slice 4 is done, or correct the brief.

**F-16. The plan does not create the as-built map.** Package map §9 says "0.19.6's as-built map… the first move batch creates it; `TestTheAsBuiltMapIsKeptInStep` knows two features (`app/maps_test.go:62-65`)". Step 8 assumes it already exists (`implementation-plan.md:51-53`). No batch creates it or extends that test, and there is no 0.19.6 `build-log.md`, which the test reads.
- Severity: Minor.
- Evidence: `app/maps_test.go:62-65`; `implementation-plan.md:51-53`.
- Simplify/Delete? N.
- Action: assign both to B2.

## 5. Does the record say what happened, or what somebody intended?

The M4 before reads as what happened: keys built by leave-one-out, a blind trace, and grades consistent with the trace files. The `afbcf18`-style bases in the keys are explained as `git archive` commits in scratch copies (`m4-key/t1-forecast-days-setting.md:3-8`). The leave-one-out runs and scripts are scratch-only, though, so the 36 runs cannot be re-checked (see F-3). Beyond F-3's "Everything is in":

**F-17. Small drifts between the plan and the documents it rests on.**
- The plan's goal names "A-1 to A-27" (`implementation-plan.md:13`), but A-28 exists (`rulings.md:85`).
- The brief says "Batch 0's twelve instruments" (`project-brief.md:106`), and the plan has thirteen (B0.1 to B0.13).
- The settled package map says move commits carry "no comment edits" (`package-map.md:149`), while R3 and step 4 allow "the paths documents and comments name" (`project-brief.md:83-85`; `implementation-plan.md:33`).
- AP's "Touches other agents" still sends `app`'s helpers to CK (`design-AP-app-split.md:1263-1265`), which A-24 overruled.
- Severity: Minor.
- Simplify/Delete? Y: one correcting pass.
- Action: correct each in the B0 documentation commit. The package-map line needs a new row, not an edit, since the map is SETTLED.

## Verdict

**NOT READY FOR THE PLAN GATE.** The decomposition is sound and well evidenced. Every code citation I checked holds:
- `dashboard.go:1156-1168`
- `maplayers.go:205-211`
- `router_lookup_test.go:134-137`
- `extending_symbols_test.go:57`
- `record_test.go:19-20`
- `executors.go:301`
- `README.md:597`
- `Makefile:35-41`
- the P10 mirror rows

Batch 0, however, contains three items that cannot pass as written (F-7, F-8, F-9). The per-batch sweep (F-10) and the new instruments' gate placement (F-11) have no defined path. BUILD would hit these walls in its first batch.

**Fix first: F-7.** Give B0.13 its real scope: re-point the record tests to a 0.19.6 shape and convert the plan's and architecture's line citations to symbols. Until then, no gate reads a word of this PLAN record, so every other citation in it, including the ones this review relied on, is unchecked by the project's own machinery.
