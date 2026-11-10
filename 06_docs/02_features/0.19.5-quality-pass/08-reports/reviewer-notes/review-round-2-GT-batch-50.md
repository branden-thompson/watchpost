# Gates, tooling and measurement, batch 50 — REVIEW red team round 2, 0.19.5

`ef9b9c27..55b10c57` (D-41). Filed whole as the reviewer returned it, its scratch path shortened to
`<scratch>`.

I ran everything in scratch worktrees and toy repos under `<scratch>/rt/round2/`. The worktrees are removed and no stray processes are left. I did not run make verify, mutant-check or fuzz. The repository's record was not touched. The repo's only change, a modified `rulings.md`, was there before I started.

### What checks out

- **Pins match what the tests measure (OBSERVED, `go test -v` at 55b10c57):**
  - Beats: idle 168 allocs / 75,989 B, radio 327 / 68,473 B, console 377 / 122,388 B.
  - Publish: 203 allocs / 82,330 B.
  - Frame miss: 5,523 / 11,322 / 14,234 / 1,136 allocs.
  - All of these are within a few bytes of the pins and the build-log table.
- **The frame-miss pins went up only because of the new measurement.** At HEAD, clearing `memo.ok` alone gives exactly the old pins (5,387 / 11,185 / 14,098 / 985). Adding the tape memo gives +65, and adding the masthead brings it to the new pins. No hidden regression was re-pinned.
- **The v0.19.3 figures reproduce.** I ported the threaded `beatCost` and the deep-copy publish pin onto a v0.19.3 worktree:
  - idle 172,963 B (claimed 172,954)
  - console 310,493 B (claimed 310,454)
  - no-change publish 425,018 B (claimed 423,842)
  - The threaded minus unthreaded idle cost is 16,440 B, which matches "16 KB cursor cell".
- **The publish pin is honest.** `d` is not threaded, so every iteration compares the original locations against the gob copy, and `reflect.DeepEqual` cannot short-cut on shared slices. A memo that stopped working would read about 424 KB and fail the pin.
- **m812** applies, and I ran it through run.sh (`MUTANT_RUN=^TestWidthAllocBudget$`): CAUGHT. m733's new anchor is unique. No stale `plaintextEsc` / `escByte` / `versionIn` / `releaseNamed` remains. The tests for tools/ids, tools/authoring, platform/plaintext and platform/render pass, as does `go vet`.
- **promote-verdicts:**
  - It accepts the real batch-48 log (the record minus its first line) under GNU sed, BSD sed and `LC_ALL=C`.
  - `TestPromoteRefusesASweepThatDidNotFinish` passes.
  - Explicit `LAST_TAG` / `DIST` / `VERDICTS_RECORD` on the test's command line beat conflicting values in an inherited `MAKEFLAGS`. Nothing leaked to the injected paths.
- **Mutation harness tests:** they all pass (`-tags mutants`, harness tests only). `TestACatchMustRepeatAlone` reads CAUGHT under the old run.sh, which I confirmed separately in a toy repo.
- **Interrupt handling on macOS (bash 3.2 as `sh`), with two workers mid-test:**
  - SIGINT to the script under GNU make exits 130.
  - SIGTERM exits 143.
  - In both cases every worker, `go test`, test binary and grandchild `sleep` is gone, both worktrees are removed and the work dir is deleted.
- **The gate ran on this exact content.** The batch's `verify-runs.md` row (tree `23feb8b0945b`) equals 55b10c57's tree without that file, rebuilt the way `tools/runlog` builds it.

### Findings

**1. run.sh's new repeat check marks a genuine order-dependent catch as INVALID, and its message says the opposite of the truth.**
- **Where:** `06_docs/mutants/run.sh:109-112`.
- **OBSERVED** in a toy repo:
  - A cache `Reset()` is mutated to do nothing.
  - `TestAPuts` registers `t.Cleanup(Reset)`, and `TestBStartsEmpty` asserts the cache is empty.
  - The mutated suite fails, and that failure is caused by the mutation.
  - The new run.sh reports `INVALID … failed beside the suite and passes alone … not attributable to this mutation`. The old run.sh reports `CAUGHT`.
- **Why the check cannot tell them apart:** any mutant whose only detector is a later test seeing leaked state looks like this. That covers cleanup, restore and reset functions, and a lock removal seen only between parallel tests. `TestACatchMustRepeatAlone`'s probe has the same shape as this genuine catch. It is only "not evidence" because the probe lies about the baseline (`MUTANT_BASELINE=assumed` over a suite that is red by design). The real sweep has a green full-suite baseline, which already excludes deterministic order-dependence on the unmutated tree.
- **Same root, also missed:** only the first `--- FAIL` is re-checked (line 98). A load flake listed first, in another package under `./...` or ahead of `-failfast`, now turns a run that also holds a real catch into INVALID. That case is HYPOTHETICAL.
- **Impact:** it fails safe, as NO EVIDENCE with exit 1, never a false pass. But the record (commit e073cac3) was judged by the old run.sh, so the new instrument has never judged the corpus. The SHIP sweep will be its first run, and any flips will arrive carrying a misleading "not attributable" verdict.
- **Fix:** on a repeat-alone pass, re-run the same scope (not the lone test) on the unmutated tree. If that is green, the failure is the mutation's. Also treat the SHIP sweep's INVALIDs as the check's validation.
- **Minor.**

**2. Re-raising the signal does not stop make when the signal goes to the script alone.**
- **Where:** `scripts/quality/mutant-verdicts.sh:111-118`, and ruling A-26 ("raises an interrupt again … so `make` stops").
- **OBSERVED:** under GNU make with SIGINT enabled, `kill -INT <script pid>` gave `recipe continued rc=130`. SIGTERM did the same (rc=143).
- **Why:** the `mutant-verdicts` recipe is `script; rc=$?; $(MAKE) promote-verdicts`. The re-raise only helps when the terminal interrupts the whole process group, where bash's wait-and-cooperative-exit handling applies. A `kill`, a timeout wrapper or a CI cancel aimed at the script still reaches `promote-verdicts`. Only the new summary refusal (A-26's other half) keeps the record safe.
- **Fix:** reword the comment and A-26, or have the recipe check `rc >= 128` before promoting.
- **Minor.**

**3. An interrupt leaves `go-build*` directories behind, contrary to the build log's claim.**
- **Where:** `mutant-verdicts.sh:100-104`; the claim is "left no process, directory or worktree behind" (build log, red-team-review).
- **OBSERVED:** after both the SIGINT and the SIGTERM runs, `$TMPDIR` kept one `go-build…` directory per worker. These are `go test` work dirs, each holding a linked test binary. `go test` is killed with SIGTERM, so it cannot clean them up.
- **Fix:** `export GOTMPDIR="$work/gotmp"` so the existing `rm -rf "$work"` takes them.
- **Minor.**

**4. SIGTERM during the baseline waits for the whole baseline, which is never killed.**
- **Where:** `mutant-verdicts.sh:181`.
- **OBSERVED:** in a toy repo with a 25 s baseline, `kill -TERM` took 20 s to take effect. bash defers traps until a foreground child finishes, and the baseline subshell is not in `$pids`. On the real repo that is up to the 144 s baseline.
- **Also (HYPOTHETICAL):** `treeof` lists the tree once before `kill`, so a child forked in that gap survives. And where `pgrep` is missing, only the worker shells are killed, which is the old behaviour.
- **Fix:** run the baseline in the background and `wait` on it, adding its pid to `pids`.
- **Minor.**

**5. `missEveryMemo` does not empty every memo the frame reads.**
- **Where:** `modes/tty/tape_memo_test.go:40-44`; A-25 and the bench comments rely on its doc ("every memo the frame reads … the whole frame's cost").
- **OBSERVED:** `bodyMemo.cursor` (`memo.go:91`) is not reset, and its key is the frame's content, so a rebuilt but identical frame is served from it. Clearing it too adds 4 allocations and about 19 KB per miss at 133x44 (5,523 to 5,527 allocs; 363,420 to 382,163 B).
- **Impact:** the alloc-count pin is effectively unaffected. The doc claim, and `BenchmarkFrame_133x44_Miss`'s "a key, a resize" bytes, understate a real miss.
- **Fix:** reset `d.memo.cursor` too, or narrow the doc.
- **Minor.**

**6. The README's new gate timing is contradicted by this commit's own verify row.**
- **Where:** `README.md:603-604` ("about 65 minutes … the last two green runs … 3,812 and 4,003 s").
- **OBSERVED:** the same commit adds a green `make verify` of 2,386 s (about 40 min) on exactly this tree (`verify-runs.md:232`). The "last two" are now 3,812 and 2,386 s, and the typical figure is unsettled.
- **Minor.**

**7. F-121's heading still says "~400 s per push".**
- **Where:** `06_docs/follow-ups.md:199`.
- **OBSERVED:** the row's body was re-recorded to 3,310 s and 131 minutes, but the bold heading still reads "~400 s", so the row contradicts itself.
- **Minor.**

**8. The v0.19.3 radio figure moved to "170 KB" on a number I could not reproduce.**
- **Where:** `CHANGELOG.md:43`, from build-log "radio 169,733".
- **OBSERVED:** my port of the same method on v0.19.3 measured 168,744 B threaded and 168,725 B unthreaded, twice, which rounds to 169 KB, the old CHANGELOG figure. Idle and console reproduce within 0.03 %, so the radio figure probably came from a slightly different `playingDash` port.
- **Minor, trivial.**

**9. The promote test covers less than its comment says.**
- **Where:** `Makefile:399` ("TestPromoteRefusesASweepThatDidNotFinish holds all four") and `cmd/watchpost/release_paths_test.go:68-74`.
- **OBSERVED** by reading the cases:
  - No case has a summary without a commit line, so the commit-name check is held only indirectly. With an empty `c`, `ls-tree` fails and the count check refuses instead.
  - The "whole sweep" case passes the "tracked path" step only because `git check-ignore` exits 128 on a record outside the repository, and `!` turns that into success. It prints "tracked path" for an untracked temp file. Hardening that line would break the test, and the test does not show it.
  - The test hand-copies the summary format. Nothing feeds a real `mutant-verdicts.sh` summary into `promote-verdicts`, so rewording the script's summary would make promote refuse every good sweep while this test stays green. I tested this by hand against the real log; no test does.
- **Minor.**

**10. The repeat is pure overhead on step-1 scoped runs.**
- **Where:** `run.sh:109`.
- **OBSERVED:** on m812, the new run.sh took 3–4 s against 2 s for the old one. When `MUTANT_RUN` is already `^top$`, the first run was the test alone, so the repeat adds a test-binary link and run for each of the ~1,050 step-1 catches. Nobody has measured what this adds to the 57-minute sweep, and the tooling backlog and Makefile timings now predate it.
- **Fix:** skip the repeat when the scope equals `^${top}$`.
- **Minor.**

No Critical or Important findings. Every finding either fails safe (NO EVIDENCE, or a refused promote) or is a wrong statement in a doc or comment.

**Verdict: EXIT** (no Critical or Important; 10 Minor, with finding 1 worth fixing before the SHIP sweep).
