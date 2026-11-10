# Gates, checks and tooling — REVIEW red team, 0.19.5

`feature/quality-pass`, `7ca44489..ef9b9c27`. Filed whole as the reviewer returned it.

I checked the code at HEAD. I ran three demonstrations in my scratch directory: a model Makefile with the same recipe shape, a real interrupted sweep inside a throwaway `git worktree` (removed afterwards), and an SN-02 fixture. The repository working tree, `dist/` and the tracked record were not touched. I did not run `make verify`, `make mutant-*` or any fuzz.

---

### I-1 Important (OBSERVED): pressing Ctrl-C during `make mutant-verdicts` overwrites the tracked record with an empty, interrupted log

**Where:**
- `scripts/quality/mutant-verdicts.sh:101`: `trap 'exit 130' INT TERM`, new in this diff.
- `Makefile:421`: `...mutant-verdicts.sh ...; rc=$$?; $(MAKE) promote-verdicts || exit 1; ...`
- `Makefile:398-405`: `promote-verdicts` accepts any log that is non-empty and whose first line names a commit.

**Why it happens:** the new trap turns the script's death by signal into a normal exit 130. The recipe shell then sees a child that "handled" the interrupt, so it carries on to `promote-verdicts`. Before this diff the script had no INT trap, so it died by the signal and the recipe stopped there.

**Demonstrated:**
- **Model Makefile, same recipe shape, SIGINT sent to the process group:**
  - With the trap, the recipe went on (`after rc=130`) and the promote step ran (`promoted.txt` written).
  - With the trap removed, make reported `Interrupt: 2` and nothing was promoted.
- **Real script in a worktree** (`MUTANT_ONLY=mDE_a_queued`, interrupted 25 s after the baseline went green):
  - The log held only the header and `baseline ./... ... green`, with no verdicts.
  - `test -s` passed, and promote's `sed` pulled out `ef9b9c27…`.
  - So the real promote would have replaced `06_docs/mutants/verdicts/since-v0.19.3.log` (1,057 verdicts) with that six-line file and printed "record durable".
  - The next sweep then reads no killers from the record.

**Other ways in:**
- A partial run started with `MUTANT_ONLY=… make mutant-verdicts` gets promoted the same way. The script's header comment (lines 51-55) warns against this, but nothing refuses it.
- A run whose baseline is RED is also promoted. That one predates this diff.

**Fix:** promote only a log that ends with the summary line, whose total equals the number of mutants at the commit it judged, and refuse when `MUTANT_ONLY` is set. Alternatively, have the script write a completion marker that promote requires. The record is tracked in git, so a bad overwrite can be restored; that is why this is Important, not Critical.

### I-2 Important (HYPOTHETICAL for this record; the project has seen this class before, mH0): the sweep's safety claim about load is backwards

**Where:** `mutant-verdicts.sh:40-46` says: "A catch at any step has run.sh's attribution check … PARALLEL WORKERS LOAD THE MACHINE, and that cannot turn a survivor into a catch: a test that fails under load and passes alone fails the attribution check. Load can only make the run slower."

**What the code actually does:** the attribution check at `run.sh:119-124` restores the tree and re-runs the failing test alone on the unmutated tree. It reports INVALID only if the test fails again. So a test that flakes under load on the mutated run and passes when re-run alone is exactly what gets credited as CAUGHT.

**The crash path has no attribution check at all** (`run.sh:136-145`). A load-induced `panic: test timed out after 10m0s` becomes CAUGHT outright. The record already holds two timeout catches (m585, mDE). Both mutants' comments say they really hang, so these two are legitimate.

**Where the risk is:**
- Four workers are the default, and each `go test` runs package binaries in parallel, so heavy load is now the normal condition.
- New mutants with no recorded killer go to step 2 (the whole package with `-failfast`) under that load. A timing-sensitive test failing there would be recorded as the catch.

**What I observed:**
- Of the 1,046 named killers in the parallel record, all but the four batch-48 fixes (m250, m334, m389, m402) are the same as in the serial record.
- So nothing in this record shows a false catch.
- Killer-first also re-judges each catch on the next sweep, so a false catch would not stick.

**Fix:** before restoring the tree, confirm a catch by re-running the failing test alone on the mutated tree. A real catch reproduces; a load flake mostly does not. Then correct the comment.

---

### M-1 Minor (OBSERVED): an interrupt leaves processes and files behind, contrary to the comment

**Where:** `mutant-verdicts.sh:88-101` says "The worktrees are removed however the run ends, an interrupt included, and the workers stopped first." That is false in two ways:

- **The in-flight work is not stopped.** `worker &` runs in a non-interactive shell, so SIGINT is ignored in the background job and in what it starts. The current `run.sh` and its `go test` keep running after the script exits. `kill $pids` stops only the worker subshell. I confirmed the in-flight child finishing after the script had exited.
- **The cleanup itself aborts.** `cleanup()` runs under `set -e`, and `[ -d "$wt" ] && git worktree remove …` is the last command of its `&&` list. If the removal fails, errexit aborts the EXIT trap, so `git worktree prune` and `rm -rf "$work"` never run. I demonstrated the same abort with a minimal script under both `sh` and `bash`.

**Observed in the real run:**
- The script exited 255 rather than 130.
- `$TMPDIR/watchpost-sweep.*` was left behind, holding a half-deleted `w1` (2,308 files) that still contained `app/pump.go`.
- The repository's worktree registry was clean in this instance.
- The orphaned processes keep loading the machine (up to 10 minutes for a hanging mutant), which breaks the serialize-heavy-runs rule.

**Fix:** start the workers in their own process group and kill the group, and add `|| true` (or `set +e`) inside the cleanup.

### M-2 Minor: nothing ties the record to the corpus at HEAD

- The record judged e073cac3. Commit ef9b9c27 then changed the instrument (`mutant-verdicts.sh`) and restated two of the record's lines by hand, so the record is not byte-for-byte what the sweep wrote.
- That is true today: ef9b9c27 changed no mutant or code, and the count is 1,057 at both commits.
- But no check compares the record's commit and count against HEAD's corpus, so a stale record can ship silently.

### M-3 Minor: the sweep exits 1 on every run

- Seven kept equivalent mutants (m16, m43, mBM1, mBM2, mSC3, plus mAA3 and mAA4 from A-24) mean the sweep can never pass.
- The script reads no ratified list of equivalents, so a new survivor changes nothing in the exit code; someone has to compare it against prose by hand.
- Fix: a ratified-equivalents file that the script subtracts, failing on rows that no longer match anything, like the new stale-row rule in `tools/dupes`.

### M-4 Minor: the sweep script has no test of its own

- `mutant-verdicts.sh` grew from 195 to 322 lines of shell: the job queue, the `mkdir` lock, the worktrees, killer parsing, sorting and counting.
- It is still exempt as a port candidate (F-156, `cmd/watchpost/shell_test.go`).
- `harness_test.go` tests `run.sh` only. I-1 and M-1 are the kind of defect such a test would catch.

### M-5 Minor (OBSERVED on a synthetic fixture): SN-02 now catches fewer fused doc comments

- **Where:** `tools/authoring/rules.go`, `ownOpeners`. A later line opening with the declaration's own name now counts only if line 0 names a different declaration, or if the line above it ends a sentence.
- **Fixture:** a capitals-led paragraph ending mid-sentence, followed by `// helper returns the next key.`, is no longer flagged. The old rule flagged it. The control case (`// Other does…` followed by `// helper2 returns…`) is still flagged.
- This is deliberate, to stop false positives on wrapped prose, and the real-world risk is low.

### M-6 Minor: a P10 density floor set at zero

- A-18 sets `tools/gateoracle`'s floor to FLOOR 0.00. From now on, the density check can never fire for that package, even on a function with no checks.
- It follows the D-277 procedure and awaits the HUM LEAD's veto. Raising it now so the veto is informed.

### M-7 Minor: step 1 vets only the killer's package

- In step 1, `run.sh:86` runs `go vet` only on the package that defines the last killer.
- A mutant that breaks vet or test compilation in its own package, but whose killer lives elsewhere, reads CAUGHT where an unscoped run would read INVALID.
- `make verify` covers the compile half through `TestEveryMutantAppliesAndCompiles` in `mutant-check`, so the impact is low.

---

### Checked and sound

- **No gate removed:** `verify-gates`, `ci.yml` and `required-gates.txt` list the same gates as before (only comments changed). No new `t.Skip` or exemption weakens a gate. The new allocation pins skip under `-race` but are picked up by the unanchored `make alloc-budget` pattern.
- **Mutant tooling:**
  - Killer parsing: all 1,046 named killers parse, and every one is defined at HEAD.
  - Sorting, counting, and the NO EVIDENCE accounting are correct.
  - The partial-run timings guard works.
  - There is no `go.work`, so the sweep's worktrees build exactly what the tree builds.
  - `targetRE` now reads both quote styles, and the new `TestEveryMutantNamesItsTarget` holds it.
  - `TestAScopedRunDecidesByTheTestsItNames` proves the scoped run reports both CAUGHT and SURVIVED.
- **Lints and tools:**
  - The new `lint-imports` go-studs check has positive and negative self-test controls.
  - `dupes` now fails stale ratified rows, which strengthens it.
  - `lint-authoring -base auto` has `fetch-depth: 0` in both `ci.yml` and `release.yml`. On `main` itself the citation rule judges nothing, by design.
  - The atlas now draws only as-built diagrams; D-15 rules that and a test holds it.
- **Removed test helpers** were replaced with `slices` equivalents. The deleted `spokenName` test is covered by `synth/source_roles_test.go`.

**Verdict: DO NOT EXIT.** I-1 is a demonstrated way to overwrite the release's sweep record with an empty one, and I-2's written guarantee about the instrument is false.
