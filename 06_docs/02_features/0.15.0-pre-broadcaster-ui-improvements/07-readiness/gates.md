# Quality gates — 0.15.0 pre-broadcaster-ui-improvements (SEV-0)

**Every gate here carries an evidence line naming a failure that was actually WATCHED.** That is the
whole point of the table and the reason it exists as a B6 task rather than an assumption: a gate
nobody has seen fail is a gate nobody has measured, and this release has already found three of them
— a `--ascii` scan covering two surfaces of thirteen, an AA gate that iterated the list it was
lifting, and a controls gate that could not fire on itself.

**"Watched" means one of two things, and the table says which:**

- **Plant** — a defect was introduced deliberately, the gate was run, and it fired. The date and the
  plant are named so the claim can be re-run.
- **By construction** — the gate's failure path runs on every invocation against known-bad input, so
  every green build is also a watched failure. `gate-controls` is the only gate of this kind, and it
  is why the others can be plants rather than standing fixtures.

A row whose evidence is a **plant that survived** is recorded too. Those are the most useful lines in
the table: one of them was a real hole, and one was a bad plant, and telling those apart is the work.

---

## 1. `make verify` — the gates every batch exits through

| Gate | Command | Evidence: a failure watched |
|---|---|---|
| `fmt` | `gofmt -l` less `06_docs/` | **Plant 2026-09-08:** a misformatted file under `modes/tty` → **CAUGHT**. |
| `vet` | `go vet ./...` | **Plant 2026-09-08:** `fmt.Printf("%d", "not a number")` → **CAUGHT**. |
| `vet-tags` | `go vet -tags watchpost_debug ./...` | **Plant 2026-09-08:** the same defect placed behind the tag → **CAUGHT**, and **plain `vet` passed on that same plant**. That contrast is the gate's whole justification, and it is now measured rather than argued. |
| `test-tags` | `go test -tags watchpost_debug ./app` | **Plant 2026-09-08:** a failing test behind the tag → **CAUGHT**. `vet-tags` compiles that file and asserts nothing in it, so the two gates are not redundant. |
| `tidy` | `go mod tidy -diff` · `go mod verify` | **Plant 2026-09-08:** a stray `require` added to `go.mod` → **CAUGHT**. |
| `vuln` | `govulncheck ./...` | **Plant 2026-09-08:** `gopkg.in/yaml.v2@v2.2.1` added AND CALLED from a new package → **CAUGHT**, naming GO-2022-0956 and GO-2021-0061. The call matters: govulncheck reports on REACHABILITY, so a vulnerable module merely present in `go.mod` proves nothing about this gate. |
| `race` | `go test -race -count=1 ./...` | **Plant 2026-09-08:** two unsynchronised increments across a goroutine → **CAUGHT**, and confirmed by what it SAID (`WARNING: DATA RACE`) rather than by its exit code. A nonzero exit is also what a compile error looks like; this batch is about gates that report the right thing, so the message was read. |
| `lint` | `scripts/lint.sh` (golangci-lint v2.13.1 + staticcheck, baseline + ratchet) | **Plant 2026-09-08, both directions:** a NEW finding of a rule already in the baseline → **CAUGHT**; and a baseline row matching nothing on disk → **CAUGHT**. A ratchet that only tightens is half a gate: the stale row is how a baseline outlives the finding it excused. |
| `lint-imports` | `scripts/lint-imports.sh` | **By construction:** `gate-controls` runs it against two known-bad fixtures (a `modes/` and a `platform/` file importing `domains/`) on every build, and fails if either goes undetected. |
| `lint-watermark` | `scripts/lint-watermark.sh` | **By construction:** same, one known-bad control. |
| `gate-controls` | the four `--self-test` invocations | **Plant 2026-09-08 — SURVIVED, and it was a real hole.** Renaming the self-test flag left the gate GREEN: `lint-imports.sh` and `p10-unmatched_test.sh` ignored an unrecognised argument and ran their normal path to exit 0. The gate whose entire job is to prove the other gates still fire could not fire on itself. Both scripts now reject an unknown argument; re-planted → **CAUGHT**. A second plant — a control's known-bad fixture edited to be good — → **CAUGHT**. |
| `alloc-budget` | `go test -run 'AllocBudget$' ./...` | **Plant 2026-09-08:** an escaping 64-byte allocation per ticker tick → **CAUGHT**. **The first plant SURVIVED and is recorded here on purpose:** `_ = make([]byte, 64)` never escapes, so the compiler stack-allocates it and there is nothing for the gate to count. A plant the optimiser deletes measures the optimiser, not the gate. |
| `mutant-check` | `go test -tags mutants ./06_docs/mutants -v` | **By construction, 171 times over.** Every mutant in the corpus is an edit that MUST make the suite fail; a `SURVIVED` verdict is the gate reporting that some test cannot fail. The harness has its own tests (`06_docs/mutants/harness_test.go`) covering the failures a mutation harness has: it refuses a red baseline, refuses a dirty tree, and calls a crash `CAUGHT` rather than `SURVIVED`. |

## 2. Gates over the gates

| Gate | Command | Evidence: a failure watched |
|---|---|---|
| CI ≡ verify | `go test ./cmd/watchpost -run TestCIAndVerifyRunTheSameGates` | **Both directions.** A naive convergence deletes the repo's only allocation gate: `verify` omits `release-matrix` and `install-test`, CI omits nothing `verify` has. The test carries a `ciOnly`/`verifyOnly` map and fails when either set rots — a gate added to one side and not the other, and an exemption left behind after the gate it excused was added. |
| Mutant policy is a switch | `go test ./cmd/watchpost -run TestTheMutantPolicyIsASwitchAndNotARewrite` | Per HUM LEAD (2026-09-08): all three schedules — `push`, `nightly`, `label` — are written out in `ci.yml`, so choosing between them is one word in the Makefile and never a workflow edit, and no mode's path has to be rebuilt after being deleted to make room for another. |

## 3. Journeys and surfaces

| Gate | Command | Evidence: a failure watched |
|---|---|---|
| Core journey | `make journey` (fresh `HOME`, the real binary, real feeds) | **Watched, and currently RED for a reason that is recorded rather than absorbed.** 26 of 28 steps pass; the two Lookup steps fail on a freshly seeded install (**F-58**), reproducible in ~40 s with `scripts/quality/lookup-stall-probe.expect`. The bound is deliberately NOT widened to make it green: 4×25 s already fails honestly, and stretching it would convert a measured stall into a tick — the call F-44 made for the 11.4 s `space` read. The read block, including B6's new precondition, passed on both runs. |
| `--ascii` completeness | `go test ./modes/tty -run TestASCIIFramesCarryNothingButASCII` | **Three plants 2026-09-08:** a `★` that was never in the glyph set → **CAUGHT** (the scan asks a question that needs no list); a window made to render nothing → **CAUGHT as NOT COVERED**, which is the failure a coverage scan is most likely to have and least likely to report; a set entry stripped of its ASCII form → **CAUGHT in all three windows that draw it**. On its first honest run it found six leaks across four windows. |
| AA completeness | `go test ./platform/render -run TestEveryTokenIsMeasuredOrExcused` | **Two plants 2026-09-08:** a newly declared token left unregistered → **CAUGHT**; an excuse written for a token that IS registered → **CAUGHT**. The gate it replaces could only fail if the lifter failed to converge. |
| Goldens · declsets | `go test ./modes/tty -run Golden`, `-run DeclarationSet` | **By construction:** both compare against a recorded artefact and fail on any drift; both moved during B6 and were re-recorded deliberately, with the reason in the commit. |

## 4. What the sweep cost, and the two rows worth reading twice

Thirteen plants, about twenty minutes of gate runs. `mutant-check` was deliberately NOT re-run for
this: its 171 corpus verdicts are already 171 watched failures, and re-running it to prove that would
have cost hours to learn nothing.

**Two gates did not fire on the first plant, and they are the reason the table exists.**
`gate-controls` was GREEN while running no control at all — a real hole, in the gate whose whole job
is to catch that class. `alloc-budget` was GREEN because the compiler deleted my plant before the
gate could see it — a bad plant, and the gate was fine. A roster showing only CAUGHT would have
hidden the one skill needed to read it: telling those two apart.
