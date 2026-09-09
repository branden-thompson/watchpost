## Summary / BLUF

**Four defects that a listener would have met, and a release that spent most of its effort on why
they were not caught sooner.** A location the weather service cannot serve shimmered for ever instead
of saying `n/a`; Settings wrote your alert radius and relay language to disk and then re-opened
showing the old ones, so the change could not be undone from the window; typing `20` over a stored
`50` produced a five-thousand-mile radius, saved silently; and the spoken fire report ran two
different rings together, so "no hotspots within a 16 mile fire ring" was followed by a list of fires
with nothing saying which ring they belonged to.

The larger half of the work is the instruments. Every quality gate now carries a **watched failure**
— a plant that was applied, compiled, and seen to turn it red — and two of those plants found that
the gate could not fail at all. A duplicate-implementation detector was built and gated. The mutant
corpus grew to **172**.

**Reviewer ask:** read `platform/snapshot/assembler.go` first. It is where "we asked and got nothing"
became distinguishable from "we could not ask", and getting that wrong is what put a permanent
shimmer on a real listener's screen for three attempts running.

## Problem & Solution Overview

### Problem Statement

*Watchpost's own surfaces cannot be checked on demand, and its rules have no single owner — so a
defect that is invisible to the operator survives review, and a rule written twice can be right in
one copy and wrong in the other.*

(Locked at DISCOVER exit, 2026-09-07; `08-reports/project-brief.md`.)

### Proposed Solution

Give the station a bounded, on-demand test of itself, so confirming it works does not mean waiting
for real weather. Give every rule one owner and prove it with a detector that runs as a gate. Make
every gate demonstrate that it can fail, against a deliberately broken input, before its green is
believed. Then fix the user-visible defects those instruments surface.

## Intended Outcomes / Value Impact

### Business Outcomes

An operator can confirm the station works in under two minutes instead of waiting on the weather, and
the surfaces that tell them what is happening no longer lie in the two ways this release found: a
row that says "loading" when the answer is "never", and a spoken report that states an absence it
cannot know. The failure mode the release is really about is the confident one — a screen or a voice
asserting something it has no evidence for.

### Metrics of Success

| Metric Name | Symbol | Type | Definition | Measured In |
|-------------|--------|------|------------|-------------|
| Time to confirm the station works | **T** | Primary | **Lower is better;** wall-clock from operator intent to a verified end-to-end alert — tone, words, ticker and window. Was unbounded (you waited for real weather). **Mechanism shipped and exercised at UAT** (a tornado warning test event, then a real thunderstorm warning taking over mid-read). **No stopwatch figure was taken**, so this is reported as delivered, not as a number | Operator UAT; the diagnostics surface's bounded test event |
| Unexplained duplicate implementations | **D** | Primary | **Lower is better;** operations implemented more than once with no written, ratified reason. Target 0, **measured 0** — re-run at SHIP: 1 group at ≥ 25 nodes, ratified, 0 unexplained. Was 10 groups; 9 collapsed | `go run ./tools/dupes`, gated in `make verify` |
| Memo keys with no completeness guard | **K** | Secondary | **Lower is better;** memo keys lacking a test that fails when the struct they key gains a field. **The frame path is closed**: `TestTheMemoKeyCoversEverythingTheFrameShows`, with `TestTheNestedWalkExcusesNothingQuietly` closing the quiet-skip hole F-30 named. The data-cache half rests on FR-3.2's ruling; **#12 stays open** for the full audit | `modes/tty/memo_completeness_test.go` |
| Gates never observed failing | **G** | Maintenance | **Lower is better;** gates never watched failing against a deliberately broken input. **Every gate in `gates.md` now carries one.** Two named residuals: `sync-go-studs` and `p10-unmatched` have only their `--self-test` half in `verify` | `07-readiness/gates.md`, one evidence line per gate |

## Scope of Change

**Touched:** the snapshot assembler and its fetch seam (`platform/snapshot/`), the fire read
(`domains/radio/synth/fire.go`, `app/fire.go`), Settings (`modes/tty/setup*.go`), the update check
(`app/release.go`), the render glyph sets (`platform/render/`), the quality scripts
(`scripts/lint.sh`, `scripts/quality/`), a new duplicate detector (`tools/dupes/`), the mutant corpus
(`06_docs/mutants/`, 171 → 172), and the docs (README, CHANGELOG, `06_docs/quality-plan.md`,
`06_docs/build-methodology.md`).

**Deliberately not in scope**, each a recorded ruling rather than an omission: the geostationary fire
detection path and non-NWS evacuation sources (**0.16.0** — the boundary is now stated in the README
and CHANGELOG rather than left implied); a per-session voice-install cap (**F-60**, a listener-facing
ruling); an age bound on the spoken fire read (**F-61**, likewise); and `make journey`, which is
**red by HUM LEAD ruling** (**F-58**) and documented as such rather than quietly excluded.

## Caveats for the (Human) Reviewer

- **`make journey` is RED and that is intended** — 26 of 28, F-58. It is the one gate not green, it is
  named in `required-gates.txt`, and it is a ruling, not an oversight.
- **The exposure statement is a point-in-time measurement and says so.** Re-derive with
  `python3 scripts/quality/exposure-scan.py` rather than citing its table; the counts move with the
  tree. `internal-url` and `credential` are clean — the two credential-shaped values in the whole
  history are hex-ramp test fixtures.
- **`-trimpath` is the largest exposure fix and it is a build-flag change**, 473 occurrences of the
  build path per binary to 0. Worth confirming on the release artifacts rather than locally.
- **Three of this release's own findings were regressions introduced by its own fixes**, all caught by
  independent review rather than by the author. The defect curve is in `08-reports/build-report.md`.

## Testing Done

- [x] Local code review completed
- [x] Unit tests added/updated
- [x] Integration tests pass
- [x] Manual testing performed

- **`make verify` — ALL GATES GREEN**, 14 gates including `race` and the 172-mutant corpus, on the
  release tree. `make journey` red by ruling (F-58).
- **CI was RED and is fixed.** `go test -race` failed on `ubuntu-latest` while `macos-latest` passed
  in the same run. Not a platform defect: a test asserted a second read was inert while one was
  running, without holding the first one open, so it passed on scheduling rather than on the guard.
  Pinned to the idiom already used by the neighbouring test. Validated before trusted — green 25/25 →
  plant the removed guard → compiles → **red 25/25** → revert → green 25/25 — and kept as mutant
  `mK8`. Carried as **F-63**: no phase exit read the branch's CI state.
- **Independent red teams at every phase exit**, blind to each other: BUILD ~20 findings / 3 blockers,
  a scoped round finding 2 regressions the author had just introduced, REVIEW 6 / 1, VALIDATE 4 / 0.
- **Install verified end to end for the first time** — `install.sh` against a server on the same code
  path as a GitHub download, asset names reconciled across `release-matrix`, `release.yml` and the
  script, checksum verified, tamper control fired.
- **HUM LEAD UAT, 2026-09-09**: fire read (quiet / one-incident / busy), Settings preserved on `esc`,
  Location Details tables, radio — all good.

## Screenshots

**No capture is stale, and that was checked rather than assumed.** The Settings diff routes marks
through the glyph set for `--ascii` correctness, so the default rendering is character-for-character
unchanged; the fire line this release removed was a spoken phrasebook entry
(`scripts/fire-report/outside.txt`), never on screen. Existing captures in `docs/img/` stand.

## Additional Context

### Problem Statement Evaluation

The statement was locked at 5/5 and held. The release's own history is the evidence: the two
user-visible defects that mattered most were both invisible to the operator until an instrument was
pointed at them, and issue #13 took **three attempts** because each fix addressed a different reason
the same row could not tell "no data for you" from "we could not ask".

### Anti-Solution Check

The statement names no solution. It does not say "build a diagnostics window", "write a duplicate
detector", or "add mutants" — those are how it was answered, and one of them (Metric D's instrument)
was pulled into the release mid-BUILD on a HUM LEAD ruling precisely because the statement left the
choice open.

## For Agents

Read `06_docs/build-methodology.md` and `06_docs/quality-plan.md` before changing anything here. The
rules that this release added and that are easiest to violate accidentally: **INST-1..INST-5** (an
instrument is not believed until it has been watched failing), the **single-owner** policy for rules
(a second copy is a defect, not a convenience), and the P10 exemption ledger, where **exemptions are
presented for ratification and never self-approved**.

## What

Ships 0.15.0. Fixes four user-visible defects and rebuilds the instruments that failed to catch them:
every gate now carries a watched failure, duplicate implementations are detected and gated, and the
mutant corpus is at 172. Closes **#14** (the release), **#13** (the permanent shimmer) and **#17**
(the completion-path stall). Does **not** close #12 — the memo-key audit is partly done and stays
open.

## How to see it

- **The shimmer fix:** add a location the weather service has no data for; the row reads `n/a` rather
  than shimmering, and still reads `n/a` after a restart. Pull your network instead and rows keep
  shimmering — that is the distinction, not a bug.
- **Settings:** `s`, change the alert radius and the relay language, `esc`, re-open. It shows what it
  saved. Type `20` over a stored `50` and you get `20`, not `5020`.
- **The fire read:** `r` on a location with fires; each half of the report names its own ring, and a
  half whose feed did not answer says so instead of asserting nothing is there.
- **The gates:** `make verify`; `go run ./tools/dupes`; `go test -tags mutants ./06_docs/mutants -v`.

## Checks

- [x] `make verify` is green (fmt, vet, race, import direction, watermark gates) — `make journey` red
      by HUM LEAD ruling, F-58, documented
- [x] `golangci-lint run ./...` and `staticcheck ./...` are clean — baseline + ratchet, 6 baselined,
      no new findings
- [x] Tests added or updated for the behaviour that changed
- [x] Docs touched where the behaviour is described (README, CHANGELOG, `docs/`, `06_docs/`)
- [x] No secrets, personal addresses, or machine-local paths in the diff — and `-trimpath` removes the
      build path from the artifacts as well

## Notes for the reviewer

**Carried, each with the reason it is carried, not as a backlog dump:** **F-63** (no exit reads the
branch's CI state — this release was bitten by it), **F-60** and **F-61** (both listener-facing
rulings, not code), **F-62** (`Within [0] mi` renders where 0 means All locations — errs toward more
alerts, which is why it is filed rather than rushed), **F-58** (`journey`, red by ruling), **F-59**
(17 duplicate groups in test code, internal, not gated), and **F-55**.

**Deliberately left out:** the demo location still present in 67 fixture files, and the geostationary
fire source — 0.16.0 work, with the coverage boundary now stated rather than implied.
