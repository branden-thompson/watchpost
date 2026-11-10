# Documents, comments and junior readability — REVIEW red team, 0.19.5

`feature/quality-pass`, `7ca44489..ef9b9c27`. Filed whole as the reviewer returned it, its scratch
path shortened to `<scratch>`.

I found no Critical findings and three Important ones, so the verdict is DO NOT EXIT. The three are a missing CHANGELOG entry, gate timings left stale after batch 47, and a stale phase line in the release's required reading.

**Scope and how I checked it.** I reviewed `7ca44489..ef9b9c27` on the docs, comments and junior-readability axis. The scratch dir is `<scratch>/rt/redteam-docs/`.

Mechanical checks, all clean:
- **Test names.** Every `Test…` name in README, `docs/*.md`, code-standards, both required-reading files and CHANGELOG exists at HEAD.
- **Citations.** Every `0.19.5 D-n` / `A-n` citation in Go, Markdown, shell and Python resolves to a row in `rulings.md` (D-1..D-40, A-1..A-24).
- **File references in new comments.** Every file name in the diff's added Go comments exists. The one `file:line` (`data_table_row.go:521`) points at the right line.
- **Re-capture flags.** Each `-update-*` flag and the package it belongs to match the list in "When a golden fails".
- **Walkthrough symbols and files.** Spot checks matched: all 20 Walkthrough 6 symbols, Walkthrough 8's `readBody`/`bcReadCardRows`/`cardLane`/`bcMinCols`, the 12 console goldens, setupTable's 59 code lines, and the bench and console pins (141–155, 309/9655).
- **Tests.** `go test ./tools/ids ./tools/atlas` and `go test -run 'Extending|WhereThings|…|Index|Doc' ./cmd/watchpost` pass.

## Important

**I-1. CHANGELOG `[Unreleased]` leaves out three ruled changes a user can see.**
- `CHANGELOG.md:24-34` ("Changed") lists D-6, D-7, D-8, D-18/D-32 and D-9. It does not list:
  - **D-33**: `p` now previews only on a voice (cast) row. Before, it played the default voice on Theme, both HISTORY pickers and both relay-replay rows.
  - **D-36**: the Reduce motion row is now a picker, so its chips read "←→ Motion".
  - **A-22**: the two RELAY REPLAY rows' chips read "←→ Rotation" and "←→ Language" instead of "←→ Voice  p Preview".
- The project's own rule (`docs/extending.md` Walkthrough 8 step 7) says a change the operator can see goes in `[Unreleased]`. The D-33 ruling names the cost to a listener who learned `p` on Theme.
- **Fix:** add three "Changed" bullets.

**I-2. The verify and mutant-check timings went stale at batch 47 (vet now runs on all 1,057 mutants).**
- `README.md:603-604` says "about **40 minutes** … The last eight green runs took 2,342 to 2,481 s."
  - `06_docs/verify-runs.md` at HEAD: the last eight green runs took 2,164–4,003 s. The two since batch 47 took 4,003 and 3,812 s, about 65 minutes.
- `06_docs/tooling-backlog.md:78` says mutant-check "costs 2,253 s locally". `Makefile:190` says "At 1032 mutants it ran 2253 s".
  - `dist/mutant-check.log` (Nov 9) ends `ok … 3310.593s`.
- The Makefile's 240m timeout is argued from these figures, and its own comment says "re-record the measurement here". The hosted-runner figure (93 min at 835 mutants, before vet ran on the 647 single-quoted mutants) now likely understates CI by a large margin.
- **Fix:**
  - Restate README step 3 from `verify-runs.md` (about 65 minutes).
  - Re-record 3,310 s at 1,057 mutants in `Makefile` and `tooling-backlog.md`.
  - Re-measure on a hosted runner, or say it is unmeasured since batch 47.

**I-3. The 0.19.5 required reading's phase row is stale.**
- `06_docs/02_features/0.19.5-quality-pass/00-REQUIRED-READING.md:24` reads: "**BUILD** … (Q1 to Q9 in batches B1 to B15); the prototypes D-22 names come first, then B1."
- This file is mandatory at session start and after compaction. What is true now:
  - BUILD ran 49 batches.
  - The After-BUILD `make mutant-verdicts` sweep is done: 1,050 CAUGHT / 7 SURVIVED / 0 NO EVIDENCE (batch 49).
  - REVIEW, then M5 grading and the UAT, are next.
  - D-40 opens 0.19.6 right after ship.
- A session rebuilding its context from this row would think BUILD is just starting.
- **Fix:** restate the row as it stands: After BUILD, the sweep's verdict, REVIEW next, D-40.

## Minor

- **M-1, wrong package for a named test.** `docs/extending.md:58` (Walkthrough 2, step 4b) says `chipGrounds` (`modes/tty/map_parts.go`) is held by `TestEverySourceIsCreditedOnce`. That test is in `app/credits_test.go`. The page's own rule is that a test is in the step file's package unless the file is named. So `go test ./modes/tty -run '^TestEverySourceIsCreditedOnce$'` runs nothing. **Fix:** name `app/credits_test.go`.
- **M-2, imports claim is wrong.** `app/doc.go:3-5` says cmd/watchpost "imports only the one-shot report's renderer (modes/report) and the schema generator (pkg/schema)". `go list` shows it also imports `app`, `platform/httpx`, `platform/invariant`, `platform/snapshot` and `platform/term`. **Fix:** say it imports no domain and no `modes/tty`, plus those helpers.
- **M-3, misleading kind comment.** `modes/tty/setup_rows.go:51-52` says rowToggle "←→ or space flip it". `setupSpace` sends a rowToggle row that `mapPrefArrow` does not own to `toggleClass`, which only tone rows answer. So a new rowToggle row ignores space, as Walkthrough 6 warns. The M5 reviewer flagged this and it was never fixed. **Fix:** say the kind's keys are answered per row (tone rows by `toggleClass`, map rows by `mapPrefArrow`).
- **M-4, stale comment on the dwell variable.** `app/radio.go:527-534`, the comment above `watchlistDwell` for the `WATCHPOST_WATCHLIST_DWELL` variable that README now documents, says "The Settings row reads this same function when it arrives." The row exists: Settings sets `d.relayDwell` ("Repeat: Watchlist - Rotate every"). The block is also another declaration's doc fused onto `watchlistDwell`. **Fix:** drop "when it arrives" and give `watchlistDwell` its own opener.
- **M-5, wrong release on a citation.** `06_docs/02_features/observer-maps/03-architecture-design/as-built-map.md:46-47` cites "0.19.3 D-364". The same page (`:58`, `:63`, `:105`) and `ids.md` file D-364 under 0.19.0 (propagation-overlays). "0.19.3 D-364" does not resolve through the index. **Fix:** write "0.19.0 D-364".
- **M-6, batch mapping is false after B7.** `implementation-plan.md:66-68` says each plan batch from B4 on is "one number later in the build log". That stops holding at B7: plan B7 became build-log batches 8 and 9, B10 became 12–15, B14 became 20–27, and B15 became 28–41. **Fix:** replace it with "the build log's headings name their plan tasks".
- **M-7, plan names an unbuilt test.** `implementation-plan.md:183` names `TestABeatCostsWhatItCosts`; it was built as `TestBeatAllocBudget`. **Fix:** name the real test, or mark the name as the plan's.
- **M-8, stale status.** `02-analysis/discover-findings.md:6` still has `status: "DRAFT for the HUM LEAD…"`, but DISCOVER exited at D-19.
- **M-9, stale "before" figure.** `CHANGELOG.md:46` says the no-change publish frame fell "from about 362 KB". 362,318 B is the figure after batch 10. At v0.19.3 it was 378,756 B (build-log batch 10 table). **Fix:** "from about 379 KB".
- **M-10, quote does not match its source.** `0.16.0 …/00-REQUIRED-READING.md:75` quotes "Asking the deck's mode here *fixed* an answer…" as verbatim. `platform/lineup/duck_test.go:18-19`, which the note now names as its home, reads "would fix". **Fix:** quote the current text, or mark the quote as the old `app/radio.go` wording.
- **M-11, stale Makefile comment.** `Makefile:169-171` (the mutant-check comment, a block the diff edited further down) says the per-mutant verdicts "come from run.sh … and are recorded by hand in the batch build logs". They come from `make mutant-verdicts` into `06_docs/mutants/verdicts/since-v$(LAST_TAG).log` (D-16). **Fix:** point it at the derived record.
- **M-12, README alt text describes screenshots not yet taken.** `README.md:58` and `:340` now describe the tabbed Settings. `docs/img/setup.png` and `broadcaster-settings.png` date from 0.16.0 (`237e2a5b`, Sep 18), before tabs existed. This is tracked (A-9, the plan's VALIDATE list), but it must not ship as is. Confirm the retake is on the SHIP checklist.
- **M-13, package map omits key packages.** `docs/architecture.md:80-95` ("A map of the packages") names 11 of the 24 `platform/` packages. It leaves out `platform/report` (the report kinds, which Walkthrough 4 calls the main way the code grows) and `platform/units`, `term`, `category`, `plaintext` and `invariant`. **Fix:** add `platform/report`, and say the table is the main packages, not all of them.
- **M-14, broken sentence.** `06_docs/perf-measurement.md:23` reads "`soak.sh` already records, per sample, — `rss_kb`, …". **Fix:** drop the comma before the dash.
- **M-15, rule scope misstated.** `06_docs/red-team-brief.md` says `make lint-authoring` decides `AP-CITE-01` "over the whole tree". `AP-CITE-01` judges only citations new since the merge-base, as `ids.md` and code-standards say. **Fix:** "AP-CITE-01 over what the branch adds".
- **M-16, history in a comment.** `modes/tty/ticker.go:263-264` says "copying it into a rune slice every frame was most of the frame's bytes". That is past tense where the rule asks for the code as it is now. **Fix:** "would be most of the frame's bytes".

**Outside scope:** the untracked `watchpost/CLAUDE.md` still sends a starting session to the 0.19.0 propagation-overlays required reading as "the release in flight". It is not part of the diff.

DO NOT EXIT
