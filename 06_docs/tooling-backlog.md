# Tooling backlog — which catalogue rules to mechanise, and which not to

**Source:** the 0.16.0 BUILD-exit red team, 2026-09-16, tooling lens (a blind Distinguished Engineer
review of the gate system as the artefact under review). The brief is `06_docs/red-team-brief.md`.

> **FP estimates are the reviewer's, and two did not survive measurement.** `#5` and `#6` were both
> estimated FP ~0 and are moderate in this tree — one because a tool added the same day takes a
> non-constant program by design, the other because `reflect` has twelve importers. **Measure the
> surface before building a ratchet**; an estimate made from reading is not a count.

**Read the "do not build" section first.** It is the more valuable half. A gate that cannot fail, or
one that rewards ritual, costs more than the defect it was bought for — this project has now shipped
three of the first kind and ratified an exemption rather than perform the second.

## What is mechanised today

| Rule | Where |
|---|---|
| `AP-HIST-01`, `AP-DEAD-01`, `SN-02` (in part) | `tools/authoring`, `make lint-authoring` |
| `SEC-04` | `make vuln` (`govulncheck`) |
| `P10-01..10` | `make p10` — the harness CLI and ledger live outside the public tree, so it is a LOCAL gate |
| closed-set reachability | `tools/wires` |
| duplicate operations | `tools/dupes` |
| artefacts in the index | `TestNoTrackedBinaries` |
| gate-list completeness | `cmd/watchpost/gates_test.go` |

## Build — ranked

Ranked by leverage, then by false-positive risk. "FP" is the reviewer's estimate of how often the
detector would flag correct code; anything above *low* must ship with a ratified exemption ledger, the
way `dupes` and `wires` already do.

| # | Rule | Detector | Decidable by | FP | Size | State |
|---|---|---|---|---|---|---|
| 1 | project-hygiene: artefact placement | `git ls-files` → executable magic, or >1 MB outside an allowlist | git index | ~0 | ~30 lines | **DONE** — `cmd/watchpost/artifacts_test.go` |
| 2 | evidence soundness: **every required gate has a control, and the control RUNS** | parse `required-gates.txt`, resolve each to its recipe, extract the scripts/tools it invokes, assert each is exercised with a self-test flag by `gate-controls` or its own gate line | build | ~0 | — | **DONE** — `TestEveryRequiredGatesCheckerHasAControl`. Found two more uncontrolled checkers on its first run; `lint-ledger.sh` gained a real self-test. **Highest leverage.** Makes `gate-controls` derived rather than hand-maintained. Would have caught both dead gates closed in this round *and* the missing injector control |
| 3 | gate-system integrity: **conditional CI steps** | parse `ci.yml` as YAML; a required gate's step carrying `if:` or `continue-on-error:` must declare a reason the way `ciOnly` does | indentation, stdlib | 0 | ~40 lines | **DONE** — `TestNoRequiredGatesCIStepIsSilentlyConditional`. Both attacks (`if: false`, `continue-on-error: true`) verified against it; closes `F-117` |
| 4 | `P10-08` build-tag inventory | every `//go:build` tag appears in a ledger AND is named by a gate that both *compiles* and *runs* it | regex + gate list | ~0 | — | The repository has paid for this twice; a tagged test "stayed dark" until a human found it |
| 5 | `P10-09` pointer discipline | `depguard` on `unsafe`/`reflect` in `.golangci.yml`, plus AST for `**T` | config + AST | **moderate — measured, not estimated** | ~15 lines | **DEFERRED to 0.16.5.** The reviewer estimated FP ~0; the tree says otherwise. `reflect` is imported by **12 files** — one of them `platform/render/units.go`, whose use is already a ratified exemption, and the rest tests. `**T` genuinely has **0 sites**, so that half is a pure ratchet and cheap, but it is also the low-value half: `tools/wires` already records that `**T` is "exotic in this tree". The depguard half needs a ratified ledger first |
| 6 | `SEC-01` fixed-argv | AST: `exec.Command(Context)` with a non-constant program argument, or `sh`/`bash`/`cmd`/`powershell` plus `-c` | AST | **moderate — measured, not estimated** | ~20 lines in `tools/authoring` | **DEFERRED to 0.16.5.** The reviewer counted four live sites and estimated FP ~0; there are now **six**, and three take a non-constant program by design: `tools/treelock` runs an arbitrary command under a lock (that IS its job), its self-test re-runs itself via `os.Executable()`, and `synth/voice.go` runs the Piper binary from its install path. A ratchet needs a ratified ledger before it can go on, and ratification is HUM LEAD time |
| 7 | `AP-STALE-01` proxy | every `go test` in a gate recipe carries `-count=1`, so no gate is answerable from cache | regex (`makeLines` harness exists) | 0 | ~15 lines | **DONE** — `TestNoGateIsAnswerableFromCache` |
| 8 | `SEC-03` unbounded external input | AST: `io.ReadAll` / `json`,`xml`,`csv` decoders over an `http.Response.Body` not wrapped in `io.LimitReader` | AST | low | — | Ratchet; three sites already correct, and this app reads remote feeds for a living |
| 9 | `P10-04` one-page functions | `funlen` at 60 | config | 0 | — | **Behind the baseline ratchet**, so the pre-existing population does not block switch-on |
| 10 | `P10-06` package-level mutable state | AST: package-level `var` of non-func type assigned outside its declaration or `init` | AST | moderate | — | High value for a concurrent TUI. Needs a ratified ledger — registries and memos are the FP class |
| 11 | `AP-DEAD-01`, the other half | `x/tools/cmd/deadcode ./cmd/watchpost` for unreachable EXPORTED code; staticcheck's `unused` covers only unexported | build | moderate | — | Behind the ratchet. The "reinvented stdlib" half is not decidable — keep it a review lens |
| 12 | `AP-FEATURE-01` proxy: requirement traceability | every `FR-`/`NFR-`/`C-` ID in the requirements document is named by a test or gate; an ID that vanishes fails | text + AST | low | — | Real maintenance; a later release |
| 13 | prose-count drift | only decidable for annotated numbers | regex | — | Marginal. Two drifted counts found and REMOVED rather than corrected — a number in prose drifts again |
| 14 | `SN-01`, narrow slice | `x := pkg.New…()` where `x` is a ≤3-char abbreviation of the package — the catalogue's own `tbl`/`vp`/`ti` specimen | AST | moderate | — | Marginal |
| 15 | `P10-02` loop annotations | flag `for {` and `for range <chan>` with no justification comment | AST | — | — | Marginal — mostly pins a convention the code already annotates |

## Do NOT build

| Rule | Why not |
|---|---|
| `P10-05` invariant density | **Gameable by padding guard clauses. A metric that rewards ritual is worse than none.** Exemptions for `tools/authoring` and `tools/treelock` were RATIFIED by the HUM LEAD 2026-09-16 rather than the ratio satisfied, and the ledger rows say so |
| `AP-ASSUME-01` | Process, not code. Mechanising it yields ritual spike files |
| `P10-07`, parameter-validation half | Not decidable |
| `SN-02`, general form | Not decidable. The shipped detector handles the narrow case (a declaration whose own name arrives late in its doc) and admits the rest |
| business-quality axis, whole | Not decidable. Stays a review lens |
| `P10-03` | N/A in Go; the allocation pins cover the declared hot paths |
| `P10-10` | **Half-done, and the fix is config not code:** `.golangci.yml` enables only the DEFAULT linter set, which is not "the most pedantic setting". Widen it behind the ratchet |
| `SEC-02` | Half-covered by `errcheck`. The buildable slice is `_ = <sensitive call>` (Remove, Signal, Write, Run) — FP moderate, medium priority |

## Shell versus Go

The project moved from shell to Go after defects in the shell. The reviewer's split:

**Move to Go** — these are programs wearing a script's clothes:
- `scripts/lint.sh` — discards golangci-lint's exit code with `|| true`, an embedded Python heredoc, `comm` over sorted temp files, and partial-failure semantics it gets wrong. **It does not fail closed** (`F-118`)
- `scripts/lint-watermark.sh` — two divergent code paths, and its control tests the one the gate does not run (`F-119`). No unknown-argument guard, unlike its siblings

**Keep as shell** — right size, controls fire:
- `scripts/lint-imports.sh` (a grep is the right tool)
- `scripts/lint-injector.sh` (`strings` over a stripped binary is a shell job)
- `scripts/quality/mutant-anchors.sh` (executing the corpus with the corpus's own parser is correct)

## Cost — what to stop paying for

- **`mutant-check` on every push** costs ~400 s and asserts the corpus applies and compiles. It **cannot fail on a survived mutant**. The drift half already comes from `mutant-anchors` in 0.3 s. `MUTANT_POLICY` is one word: move it to nightly and spend the recovered time on `mutant-verdicts`, the only gate that measures the corpus's actual claim. (`F-121`, deferred to 0.16.5 per the release cadence)
- **`modes/tty/declset_test.go`** gates an activity, not a standing property: "a pure file move changes no declaration" is true only of move commits, every ordinary batch trips it, and the remedy is a one-flag golden re-capture no reviewer can distinguish from a suppressed defect. Scope it to move commits or delete it. (`F-122`)

## The four ways the three-list rule can still be defeated

Recorded because a known attack is cheaper to close than to rediscover:

1. **A step condition.** `ciTargets` matches `run: make <target>` and nothing else, so `if: false` or `continue-on-error: true` leaves all three lists agreeing while the gate never runs. Precedent for the shape already exists in `ci.yml`, so it reads as unremarkable. — build #3
2. **An empty recipe.** Nothing compares a gate's recipe to its behaviour; `fmt: @true` passes all three checks. — build #2
3. **A duplicate rule.** `targetDeps` returns the FIRST matching line; GNU make accumulates prerequisites across rules for one target, so the test can read one list while make runs another.
4. **Two copies of one decision.** *(closed 2026-09-16)* `TestEveryRequiredGateIsStillRun` hardcoded the gates `verifyOnly` declares. It now reads the map.

Also social rather than mechanical: deleting a gate from all three places is three one-line deletions in one commit. `required-gates.txt` requires a HUM LEAD ruling in `gates.md`; nothing checks for one.
