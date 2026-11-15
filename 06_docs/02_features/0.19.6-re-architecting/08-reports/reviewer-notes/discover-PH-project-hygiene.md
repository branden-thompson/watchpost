---
title: "0.19.6 DISCOVER exit red team — PH, project hygiene"
date: 2026-11-15
phase: DISCOVER
sev: SEV-0
status: "Filed whole as returned; dispositioned in ../red-team-review.md"
---

# 0.19.6 DISCOVER exit red team: Project Hygiene axis

Reviewed at `feature/re-architecting` `66d8a350`. The code is identical to `e57afeaf`. Scratch is `<scratch>/`, which holds a clean clone and a list of mutant targets.

**What I checked and found true.** Most of the record's figures hold:
- Line and file counts (31,346 lines in 88 files; 20,184 in 76; test lines 39,637 and 28,477; 363 test files, of which one, `one_band_writer_test.go`, is external).
- Method counts: `Dashboard` 752 methods in 68 files, `livePipelines` 108 in 21, `Broadcaster` 92.
- `dashboard.go` is used by 71 of the other 87 files.
- Mutants: 692 = 539 + 153, with 289 in `map_*` and 80 in `broadcaster*`. This recount used the sweep's own `pkgof` rule.
- The baseline record: 1,058 / 1,051 / 7 / 0 at 3,976 s, and it equals the header plus `dist/mutant-verdicts.log`. The 57 % and 23 % shares match 8,013 s and 3,213 s out of 14,101 s.
- Import edges (43; 15 direct and 16 transitive `platform` packages), A-4's 3,171 / 2,381, and the ruling citations I spot-checked.

---

## 1. Is anything in the tree that should not be?

**H-1. The baseline sweep record exists in one copy, and the plan to commit it with the first full gate means hand-moving it before every commit until then.** Severity: **Important**.
- The file is untracked and not ignored. The `.gitignore:5` re-include makes it visible to git.
- `tools/docslane` counts untracked files as part of the change (`tools/docslane/main.go:76`). It refuses this tree now: I ran `go run ./tools/docslane` and it named `since-v0.19.5.log` and exited 1.
- Even so, eight docs-lane runs after the promotion are green (`06_docs/verify-runs.md`, rows from 12:44Z on 11-14 to 16:37Z on 11-15). The file's mtime (Nov 15 08:37 local) matches the last docs-lane output. So the file is being moved out of the tree for each docs commit and moved back afterwards. That is an inference; how it is done is UNVERIFIED.
- The only other copy is in the session scratch directory (`<scratch>/`), which does not survive a reboot.
- The record never mentions this step. A-3 (`rulings.md:47`) says "Promoted to …", which reads as durable. Only RX-12 (`lens-RX-regression-net.md:406`) says the file is untracked.
- Simplify/Delete? N.
- Action: commit it now, in a commit of its own. The code is `e57afeaf`'s, so a full `make verify` at DISCOVER exit costs one run. Until then, keep a second copy inside the repository's ignored `dist/`, and write the move-aside step into the required reading. Changing the docs lane to let a promoted verdict record through would be a tool ruling for the HUM LEAD; I do not recommend it.

**H-2. Leftover local refs.** Severity: Minor.
- `refs/original/refs/heads/feature/0.16.0-broadcaster-ui` is a filter-branch backup. It holds 433 commits that are not on `main`, and 72 lines of their diffs match the user's name or machine paths.
- `refs/remotes/gh/main` (`a7e77990`, 2026-10-22) belongs to a remote that no longer exists (`git remote -v` lists only `origin`).
- Neither is pushed by default, and both predate 0.19.6.
- There is also a 6 MB Mach-O test binary, `geodata.test`, at the repository root (ignored by `.gitignore:10`).
- Evidence: `git for-each-ref`; `git log -p refs/original/... --not main | grep -c` gives 72.
- Simplify/Delete? **Y**.
- Action: `git update-ref -d` both refs and remove `geodata.test`. Nothing in the record depends on them.

**H-3. The survey's instrument and raw data live outside the repository.** Severity: **Important**. This is listed under Question 1 because the record depends on files the tree does not hold.
- A-4 (`rulings.md:48`) keeps `pkggraph` out of the tree. Yet M3 and M5 (`project-brief.md:48`, `:50`) are "re-run on each batch's layout", and their targets are "fixed at PLAN from the graph".
- The tool and its TSV files exist only in the session scratch, and the lenses cite `<scratchpad>/q196/...` outputs.
- `pkggraph` is a module of its own (`module pkggraph`, requiring `golang.org/x/tools v0.52.0` and `golang.org/x/sync v0.24.0`). Brought in as it stands, it would be a nested module that `./...` skips, so every gate would pass it vacuously. Brought into watchpost's `go.mod` instead, it would raise `x/sync` from v0.22.0, which the product uses: a dependency change in a release that promises no behaviour change, with the re-pin checklist that comes with it.
- A-3's per-package sweep times (the 57 %/23 % in D-1 and in the brief's problem statement) come from `dist/mutant-verdicts.timings`. That file is ignored and rewritten by the next sweep. The committed record carries only per-test times like "(0.00s)". A-3 does not say where its figure came from.
- Simplify/Delete? N.
- Action: make it a batch 0 item to bring the graph tool in as a package under `tools/`, inside the main module, with its dependency change ruled. Alternatively, commit the baseline TSVs, a summary of them, and the tool's source hash before the scratch directory is lost. Record the per-package seconds next to A-3, stating their source.

## 2. Does the build work from a clean clone, and will it after the moves?

- Today: yes. A `git clone` of the branch into `<scratch>/` builds with `go build ./...` on go1.27.4 and at the floor (go1.26.10). `go vet -tags property ./modes/tty ./platform/history` is clean, and `make build` followed by `--version` prints `0.19.5-13-g66d8a350`. There is no `go.work` and no `replace` directive.
- After the moves, two risks the record does not name:
  - **H-3**: the `x/tools` dependency, if the graph tool joins the tree.
  - **H-4**: a clean clone runs the sweep with no killers until the record is committed. A-2 says so for the post-tag case, but the same silent path applies today: `scripts/quality/mutant-verdicts.sh:154` uses `[ -f "$killers" ]`, so a `MUTANT_KILLERS` that names a missing file falls quietly to step 2. Severity: Minor. Simplify/Delete? N. Action: A-2's fix must also fail loudly when an explicitly named killer record is missing, must order "newest" by the version in the file name (in a clean clone every file has the same mtime), and must print which record it used.

## 3. Are the claimed gates run on every path, and which would go quiet after a move?

The record's catalogue (TG-2, TG-3, TG-6, TG-8, PB-7, RX-9) is accurate where I checked it: `vet-tags`, `test-platforms`, `quality-bench`, the `wires` `num` sentinel (`tools/wires/main.go:311`), the reply guard (`router_lookup_test.go:152`), the `tty` qualifier (`one_band_writer_test.go:35`), `map_vision_test.go:166-167` with no floor and its glob error discarded, and the race-skipped pins. Four things it misses or leaves undecided:

**H-5. Any later full sweep overwrites the baseline that 0.19.6 is to be judged against, and nothing in the record says so.** Severity: **Important**.
- `VERDICTS_RECORD := …since-v$(LAST_TAG).log` (`Makefile:378`). `LAST_TAG` stays `0.19.5` until the 0.19.6 tag.
- `promote-verdicts` writes that file with `>` (`Makefile:417`), and by design "a cycle's newest sweep replaces that cycle's last" (`Makefile:376`).
- So the BUILD-exit sweep (`Makefile:359`, "owed at BUILD exit") and the closing sweep each replace A-3's record. The closing sweep then reads BUILD-exit's killers, through `MUTANT_KILLERS=$(VERDICTS_RECORD)` at `Makefile:436`.
- RX-12's statement that "the only reader of that path is the promotion target" (`lens-RX-regression-net.md:404`) is wrong: the sweep reads it too.
- Batch 0 item 2 (`package-map.md:106-107`) and M7 (`project-brief.md:52`) compare against "A-3's" with no commit named.
- Simplify/Delete? N.
- Action: the comparator reads the baseline as `git show <A-3 commit>:path` with the commit pinned in the record. Alternatively, A-3's record is copied to a name that is never promoted to, such as `baseline-0.19.6.log`. Either choice is an A-row.

**H-6. The deck move leaves the platform simulation with two choices: go quiet, or need an exported setter that only tests call.** Severity: **Important**.
- The GOOS seam is unexported and marked "Tests only": `setRuntimeGOOS` (`app/cast.go:51-52`).
- `app`'s `TestMain` sets it from `WATCHPOST_TEST_GOOS` (`app/platform_sim_test.go:26-29`).
- Every production user of the seam (`cast.go`, `voices.go`, `hostfacts.go`) goes to the deck (`lens-AS-app-seams.md:163-165`, `:186-187`).
- After the move, root tests that drive the deck run on the host's real OS whatever `make test-platforms` sets. That is quiet. Or the deck exports a setter used only by tests, which C-3 forbids (`package-map.md:89`) and M5's zero target refuses (`project-brief.md:50`).
- Batch 0 item 4 ("TestMain invariants that travel", `package-map.md:113-115`) assumes a helper can set the seam in every package, which needs exactly that export.
- Simplify/Delete? N.
- Action: put the choice to the HUM LEAD at PLAN, with options. One is the deck reading the env var behind a build tag that only tests compile. Another is the deck's own `TestMain` plus `test-platforms` derived over `./app/...`, with no root-level simulation of the deck.

**H-7. Two quiet guards are found but not put on the batch 0 list.** Severity: Minor.
- `releaseRecord` still names `propagation-overlays` as "the release in flight" (`cmd/watchpost/record_test.go:19-20`). It is already untrue, since 0.19.5 had a record of its own. So no record test reads 0.19.6's plan or rulings.
- RC-6's "Not held" documents drift silently: follow-ups paths, the as-built maps, `code-standards.md`, and 49 Go comments (`lens-RC-the-record.md:265-290`).
- TG-10 and RC point both items to PLAN, but `package-map.md:101-133` lists neither and §8 records neither.
- Simplify/Delete? N.
- Action: add both to batch 0, or to §8 as owed with the batch that owes them.

**H-8. `make p10` is local by ratified design, and the brief does not say so.** Severity: Minor.
- M6's "make p10 green at the end of every batch" (`project-brief.md:51`) runs on no CI path.
- It needs the external `a2dh` CLI and the ignored `.a2dh-p10-exemptions.yml` (`Makefile:550-560`, `required-gates.txt` p10 comment).
- Its only durable evidence will be what the build log records. That is a ratified design (2026-09-17).
- Simplify/Delete? N.
- Action: M6 states that it is a local gate and that the build log carries each batch's `dist/p10.json` summary.

Nothing else found. `lint-baseline` fails in both directions (`scripts/lint.sh:78-90`). `alloc-budget` is unanchored over `./...`. The prepush selector and the run selectors are guarded (`record_test.go:326`, `gates_test.go:746`). The `dupes` and `authoring` directory skips (`tools/dupes/main.go:238`, `tools/authoring/main.go:44`) do not match any proposed name. Naming guidance for the A-rows still owed: avoid `mutants`, `dist` and `vendor`. Also avoid `maps` for the tty map package: `modes/tty/help_about.go` imports the standard library's `maps`.

## 4. Is anything named as owed now untrue, or will the split make it untrue?

**H-9. The R3 "move only" commit cannot be green as defined.** Severity: **Important**.
- R3 (`project-brief.md:60-62`, `package-map.md:144-146`, `lens-RX-regression-net.md:434-436`) allows only `git mv`, package clauses, imports and mutant paths, with "no comment edits". Three things break that:
  - **Qualifiers.** Every use of a symbol that crosses the new line needs a `pkg.` qualifier at its use site. That cannot be done in place before the move (you cannot write `kit.X` inside the package that still declares `X`), and it drifts every mutant anchor whose text contains the name. No lens mentions qualifier edits.
  - **The file maps are comments.** `doc.go`'s file map is a comment block (`modes/tty/doc.go:18-22`). `TestEveryFileIsInItsPackagesMap` (`cmd/watchpost/docs_test.go:351-395`) fails the move commit unless that comment is edited.
  - **Required same-commit edits.** D-10 puts P10 re-keys "in its move commit" (`rulings.md:33`). R6 and TG-10 require doc rows, the lint baseline and the atlas in the same commit.
- So either the move commit is red, or R3 is broken in every batch.
- A-7's pilot asks whether a "byte-identical move" counts as changed. With qualifiers the moved files are not byte-identical, so the pilot must measure that case.
- Simplify/Delete? N.
- Action: at PLAN, write R3's allowed-edit list in full: qualifiers, `doc.go` map rows, ledger and baseline re-keys, doc path rows, atlas. The union declaration check and `git diff -M --word-diff` then prove that nothing else changed. Size the anchor drift that qualifiers cause in the PLAN pilot.

**H-10. The P10 re-key count is overstated, and the table at PLAN should list what actually moves.** Severity: Minor.
- D-10 and TG-5 say 16 file-keyed rows, "10 in modes/tty and 6 in app" (`rulings.md:33`; `lens-TG-tooling-gates.md:207-208`). The cited lines (`p10-ledger.md:18`, `:49-56`, `:63`, `:172`, `:190`, `:212-213`) are 14 rows: 9 for tty and 5 for app. The other two are the directory density rows (`:122`, `:129`), which do not break when a file moves.
- Under the map, the router and view rows stay at the root (`:49-56`).
- The rows whose files move are `broadcaster_rail.go` (to the console), `cast.go` (to the deck) and `every.go` (to the kit). `everyTick` also changes name if it is exported, so that row changes its symbol as well as its path.
- Add the seven proof re-checks and the prefix inheritance of the density rows.
- Simplify/Delete? **Y**. The table is smaller than ruled.
- Action: the re-key table at PLAN lists rows by destination.

**H-11. F-22 says no guard exists, but one does; the split will make F-158's file paths wrong, and no check reads them.** Severity: Minor.
- F-22 says "nothing enforces" the band's single writer (`follow-ups.md:167`). The guard it asks for exists and cites it: `TestOnlyMastercontrolWritesTheBand` (`modes/tty/one_band_writer_test.go:19`). RC repeats F-22 as current evidence (`lens-RC-the-record.md:126`).
- F-158 (`follow-ups.md:90`) names `app/maintrack.go` and `app/arbiter.go`, which move into the deck. Follow-ups paths are not checked; `TestFollowUpsCiteTestsThatExist` reads test names only.
- F-73's new due date is consistent with D-11 and A-8.
- `CLAUDE.md` still calls 0.19.6 "next". That is ratified to stay until PLAN (D-5).
- Simplify/Delete? **Y** for F-22.
- Action: close or reword F-22 as an A-row. Re-point F-158 in the deck's move batch.

**H-12. M7's closing figure cannot be met if the corpus grows.** Severity: Minor.
- "The closing full sweep matches A-3's: 1,051 caught" (`project-brief.md:52`, `package-map.md:148-149`). Under FULL TDD, batch 0 adds instruments and probably mutants, so the total will not stay 1,058.
- Simplify/Delete? N.
- Action: restate M7 as "every A-3 mutant keeps its class, or is re-ruled; every new mutant is caught; none NO EVIDENCE".

## 5. Does the record say what happened, or what somebody intended?

**H-13. The "settled document" says it is PROPOSED and claims decision rows that were never written.** Severity: **Important**.
- `package-map.md:7` reads: "status: PROPOSED — each fork below is put to the HUM LEAD one at a time". Yet D-9 to D-13 have ruled every fork, and the required reading calls it "the settled document" (`00-REQUIRED-READING.md:26`).
- §7 (`package-map.md:163-177`) starts at question 5, and its item 9, the brief, does not say "answered, D-13".
- It says "the agent's own choices (names, the alias rule, `internal/`, the lock decision, the comparator, the ratchet) are A-rows, put for veto with the brief". It also promises two names "(name an A-row)" (`:48-49`) and the citation-lint fix "as an A-row" (`:129`).
- None of these rows exists: A-1 to A-8 (`rulings.md:45-52`) cover none of them. D-13 approved "A-1 to A-8". C-4 to C-8 reached the HUM LEAD only as references inside M1, M5 and M6.
- A PLAN author cannot tell which of C-4 to C-8 have been ratified.
- Simplify/Delete? N.
- Action: change the status to SETTLED. Either write A-9 onward for C-4 to C-8 and the citation-lint fix and put them for veto, or have the HUM LEAD confirm that D-13's approval of M1, M5 and M6 covers them. Leave the names as owed A-rows for PLAN.

**H-14. M1's baseline quotes the pinned figures, not what the code measures today, and PB contradicts its own table.** Severity: Minor.
- PB's verdict says every pin "equals the figure it was pinned at" (`lens-PB-performance-baseline.md:14-16`). Its own table shows the console at 122,399-122,405 B against 122,387 pinned, and the publish at 82,335-82,340 B against 82,322 (`:79-80`).
- I ran the pins once: idle 168 allocations / 75,987 B, radio 327 / 68,471, console 377 / 122,393, publish 203 / 82,337.
- M1 (`project-brief.md:46`) quotes the pinned figures (`beat_alloc_test.go:60-61`, `:106`) as if they were measurements.
- Simplify/Delete? N.
- Action: batch 0's comparator records its baseline from repeated runs on `e57afeaf` (min and max), and the brief quotes that.

**H-15. The lenses give different test counts.** Severity: Minor.
- `modes/tty` has 1,084 `Test` functions (1,085 counting `TestMain`); `app` has 687.
- RX and the package map say 1,097 and 689 (`lens-RX-regression-net.md:91`, `:99`; `package-map.md:25`). TG says 1,085 (`lens-TG-tooling-gates.md:439`).
- The test-inventory instrument (item 5) needs one counting rule.
- The tty sizes in §2 also add up to 31,600-31,900 against 31,346 (`package-map.md:46-49`).
- Simplify/Delete? N.
- Action: fix the counting rule when the inventory is built in batch 0. RC's "255 anchors (145, 110)" (`lens-RC-the-record.md:264`) is covered by A-6.

---

## Opinion

**EXIT DISCOVER.** Nothing here is Critical. The figures hold up against the code far better than usual, and the quiet-guard catalogue is thorough. H-5, H-6, H-9 and H-13 are PLAN-level corrections and none of them changes the package map. One finding has to be dealt with at exit rather than in PLAN.

**Fix first: H-1, together with H-5.** Commit `06_docs/mutants/verdicts/since-v0.19.5.log` now, gated by one full `make verify` on code identical to `e57afeaf`. Pin its commit in A-3, so the comparator reads the baseline from history and not from a path that the BUILD-exit and closing sweeps overwrite by design. Today 0.19.6's whole regression yardstick is one untracked file, moved in and out of the tree by hand for every docs commit, with its only backup in the session scratch directory.
