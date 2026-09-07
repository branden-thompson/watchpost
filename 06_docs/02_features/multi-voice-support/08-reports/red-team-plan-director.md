# Red-Team — PLAN exit, Station Director & the Lineup

`red-team: SHIP-WITH-CONDITIONS (was NO-GO; both Criticals closed) · single-agent · scope:feature(director sub-feature) · personas:[safety-critical, perf, junior-dev, a11y]`

**2026-09-01 · SEV-0 · HUM LEAD.** Axes: all four (floor). Phase lens: **PLAN (Principal Architect)**,
plus **DISCOVER (Distinguished PM)** folded in — the DISCOVER gate was run ad-hoc without the
framework structure, and PLAN rests entirely on it, so the lens is applied retrospectively here
rather than in a separate round.

**Personas.** Approved set: safety-critical, perf, junior-dev, a11y. **InfoSec deliberately skipped**
— HUM LEAD approved "if risk is nominal", and the judgement with its evidence: this scope adds no
credential, PII, or injection surface; the existing feed-text defences (`render.Plain` /
`plaintext.Line` escape stripping, S-F6) and the `0600`/`0700` seen-store are untouched; the only new
outward-facing content is a spoken website address in a Broadcaster edition that is not built here.

**Standards resolved** at `_a2dh/AGENTS_a2dh.md` (match 1), plus the code-authoring catalog for
`AP-*` / `SN-*` / `P10-*` IDs.

## Scope

`03-architecture-design/{director-architecture, plan-decisions, director-build-plan}.md` ·
`app/cutover_latency_test.go` · by reference the DISCOVER artifacts they rest on.

## Tool evidence (safety-critical persona)

`a2dh p10 check --json`, read from `findings[]` rather than the exit code — the CLI exits 0 when every
finding is *exempted*, which it did.

| | |
|---|---|
| Findings | **20 · all exempted · 0 live** |
| Tools | 7, **all RAN, none SKIPPED** (exemptions, golangci-lint, staticcheck, go-vet, govulncheck, p10analyzers, p10:density) |
| Run record | `tree_hash` present, `ran_at` current, `base` = `merge-base:main` @ `186d97c` |
| Exemptions added by this scope | **none** |

**A prior finding, correctly closed — not re-raised.** The exemption ledger is gitignored
(`.gitignore:12`), so exemption deltas are unreviewable in a diff. A previous red team rated this
**Critical** (`watchpost-performance-quality-pass/08-reports/red-team-plan.md:48`); it was
**dispositioned** by relabelling the gate "local, HUM LEAD" with `make p10` (`Makefile:77`) failing
loud when the CLI is absent. That disposition stands. Its narrow residual for *this* plan is PL-16.

## Findings

| # | Axis / Lens / Persona | Finding | Severity | Evidence | Simplify/Delete? | Action |
|---|---|---|---|---|---|---|
| **PL-1** | Architect · Perf | **The build plan's own pump pseudocode runs effects INLINE**, contradicting the architecture's non-blocking claim. `for ev := range events { d, fx = d.Step(ev); run(fx) }` executes a 2.14 s `BuildCard` on the pump, stalling the whole schedule — reintroducing Approach B's deadlock class inside the approach chosen to avoid it. | **Critical** | `director-build-plan.md` T2.1; contradicted by `director-architecture.md` ("the 2.14 s build is an `Effect` the pump runs **off** the `Step` path") | N | T2.1 states effects are **dispatched asynchronously**, results returning as events. Add a test: a long-running effect does not delay the next event. |
| **PL-2** | Architect (failure propagation) | **The pump is an unsupervised single point of failure with a LARGER blast radius than today.** `startTakeover` already guards a panic explicitly, because one "would wedge the marquee for the life of the process" — and that is one surface. Under C a panic in the pump wedges **both tracks and the bed**. No DR requires supervision or recovery. | **Critical** | `app/ticker.go:245-251` (the existing guard and its stated reason); `director-build-plan.md` Phase 2 has no supervision task | N | A DR for pump supervision, and a test that a panicking effect does not kill the schedule. |
| **PL-3** | Perf (Necessity) | **The plan adds standby machinery without first challenging the 2.14 s itself.** `segments()` issues 11 requests sequentially, and most are independent — only `products` depends on `office`. Concurrency could cut the gap materially at far lower risk than a standby lifecycle. The Necessity lens was never applied to the measurement. | Important | `app/radio.go:330-365` (straight-line fetches); perf-protocol §6 | **Y — possibly deletes T3.3's urgency** | Measure a concurrent `segments()` before building standby. Standby may still be right; the comparison is owed. |
| **PL-4** | Business (Observability) | **No runtime signal that the Director is working once shipped.** S-1/L-8 ruled Observer's rail not *inspectable* — a UI decision — but nothing replaces it for diagnosis. Against RD-1 (≈30 defects on this code, every one gate-passing), a scheduler with no observability is how the next one goes unnoticed. | Important | `director-requirements.md` DR-1…DR-21 (no observability requirement); `02-analysis/director-risks.md` RD-1 | N | A DR for a diagnostic surface: the lineup in `[S]`, or a `WATCHPOST_DEBUG_RADIO` line per card transition — the emitter exists and P-1 just proved its value. |
| **PL-5** | Docs (cross-document consistency) | **The evidence bundle contradicts itself.** `read-order-design.md` still presents the superseded ten-rung scale **and** the per-category `Max` row that R-1/R-2 removed, and lists G-1…G-11 as open when most are closed. `README-director-handoff.md` still reads "Status: SPECIFIED, NOT BUILT … It gets a FULL RCC and a FULL PLAN." A reader opening either is contradicted by `lineup-model.md`. | Important | `read-order-design.md:58-107,128-141`; `README-director-handoff.md:5-7` | N | Stamp both superseded/updated, pointing at `lineup-model.md`. |
| **PL-6** | Code Quality · Architect (scope mirage) | **The effect vocabulary is never enumerated.** T2.2 says "effect executors as adapters" and names four by example. Unbounded work hidden in a vague task is the scope-mirage anti-pattern, and the effect set is the architecture's actual interface. | Important | `director-build-plan.md` T2.2 | N | Enumerate the closed effect set in PLAN, before BUILD. |
| **PL-7** | SafetyCritical (`P10-02`) | **DR-11 asserts the emergency read's bound is "data-derived" without naming the datum.** The tools cannot decide a bound proof — that is exactly what the persona reviews by judgement, and a reviewer needs the number, not the adjective. | Important | `director-requirements.md` DR-11 | N | Name the bound (the feed's per-lane cap) and pin it with a test. |
| **PL-8** | Perf (shared resource) | **Standby pre-build competes with the alert path for the shared httpx token bucket** (`RatePerSec` default 5). `synth.WithPriority` gives Director work a private render lane; there is **no equivalent for HTTP**, so a pre-build in flight can delay an alert's own fetches. | Important | `platform/httpx/httpx.go` Config.RatePerSec; `domains/radio/synth/limit.go:18-24` (the render precedent that has no HTTP twin) | N | State the interaction; either a priority lane or a rule that pre-build yields to alert fetches. |
| **PL-9** | DISCOVER lens (risk quality) | **The risk register rates impact but not likelihood.** The lens requires probability *and* impact; RD-1…RD-12 carry severity alone, so "HIGH" conflates "certain and survivable" with "unlikely and fatal". | Important | `02-analysis/director-risks.md` (all rows) | N | Add a likelihood column. |
| **PL-10** | Code Quality | **The ticker's cannot-comply case is unspecified.** D-C-2 asked it in as many words — "what happens if the ticker cannot comply — a modal is open, a takeover is already drawn?" S-8 answered the *contract* (fire-and-trust) but never the *failure*. DR-17 and DR-18 are both silent. | Important | `director-charter.md` D-C-2; `director-requirements.md` DR-17, DR-18 | N | **Needs a HUM LEAD ruling.** |
| **PL-11** | DISCOVER lens (implicit requirements) | **Retiring the wall-clock backstop leaves burst DURATION unbounded.** G-10's reasoning was that the count bounds the burst — but count bounds the number of reads, not their length, and the ~8 s figure is an explicitly unvalidated assumption M3 is supposed to confirm. With Max settable to ALL and Emergency Orders exempt from Max, nothing bounds time. | Important | `lineup-model.md` (G-10); `app/ticker.go:367-374` (the 8 s assumption, stated as an assumption) | N | **Needs a HUM LEAD ruling**: state that duration is deliberately unbounded and that M3 validates the assumption, or restore a stated bound. |
| **PL-12** | A11y | **DR-21's error modal arrives with no accessibility requirement** — no contrast tokens registered (F-18's live gap: an unregistered token is never measured), no keyboard dismissal, no focus handling. | Minor | `director-requirements.md` DR-21; `06_docs/follow-ups.md` F-18 | N | Fold the requirements into DR-21. |
| **PL-13** | JuniorDev | **A stated understandability claim that does not survive its own test.** Approach C is justified with "a reviewer who knows the UI already knows this" — but bubbletea's `Update` lives in `modes/tty`, the Director lives in `app/`, and `scripts/lint-imports.sh` deliberately keeps those layers apart. A newcomer to `app/` may never have opened `modes/tty`. | Minor | `director-architecture.md` (Approach C rationale); `scripts/lint-imports.sh` | Y — the claim | Soften it, or cite the specific file a reader should open. |
| **PL-14** | Project Hygiene | **Seven exemption entries carry "Ratify at the &lt;X&gt; gate" with no recorded ratification**, while six others in the same file use an explicit "RATIFIED by the HUM LEAD &lt;date&gt;" convention — so the file demonstrates the convention and seven rows do not meet it. May be stale text with the ratification recorded in a build log; unverified per entry. | Minor | `.a2dh-p10-exemptions.yml` (7 blocks: `app/cast.go`, `app/director.go` ×2, `domains/radio/script`, `domains/radio/pronounce`, `platform/plaintext`, `app/release.go`) | N | Reconcile at BUILD exit; per-entry, either point at its ratification or obtain one. |
| **PL-15** | Project Hygiene — **self-reported** | **A commit message on this branch claims something the commit cannot evidence.** `022e98a` says "Recorded in **the ledger** and the build log"; the commit contains **only** the build log, because the ledger is gitignored. The claim is true of this machine and unverifiable in the repo. Authored by me, this session — the same "no claim without a command" failure the plan enforces on everyone else. | Minor | `git show --stat 022e98a` (1 file changed: `p5-build-log.md`); `.gitignore:12` | N | Record the correction; use "recorded in the local ledger (untracked) and mirrored to the build log" wording hereafter. |
| **PL-16** | Business · Hygiene | **The plan's p10 gate says exemptions are "presented for ratification, never self-approved" but not WHERE.** With the ledger untracked, the tracked build log is the only reviewable record — the convention prior work established but this plan omits. | Minor | `director-build-plan.md` Phase 5 | N | Phase 5 names the build-log mirror as the ratification record. |

| **PL-17** | Perf — *found by this round's own challenge* | **The PD-2 baseline was measured under unrealistic conditions, and the published figure was wrong.** The perf lens asks "was the baseline measured under realistic load, or only the trivial case?" It was not: `TestCardBuildCost` built its client without `RatePerSec`, taking httpx's default of **5/sec** where the app uses **30**. Eleven requests at 5/sec ≈ 2.2 s — **the instrument was timing its own misconfiguration** and reporting **2.14 s** as a network figure. The tell was in the data: three samples within 0.1 s of each other is a token bucket in lockstep, not a network. | **Critical** | `app/cutover_latency_test.go` (original client literal); `platform/httpx/httpx.go:232` (default 5); `app/app.go:123` (`RatePerSec: 30`) | N | **FIXED.** `cardBuildClient` is now a named constructor carrying the reason. Re-measured at n=5: **1.03 s median**. The conclusion survives, the number did not. |

## Dispositions applied during this round

| # | Ruling / result |
|---|---|
| **PL-3** | **RESOLVED — measured, and against the recommendation.** `TestCardBuildConcurrentFloor` (n=5) floors at **0.96 s** versus **1.03 s** sequential, with an identical 11 requests. The calls funnel through a shared `/points` resolution that serialises however they are issued, so the build is **not latency-parallelisable**. Standby is the remedy, not one of two options. The finding was right to demand the measurement and wrong about its outcome — which is the correct result for a Necessity challenge. |
| **PL-11** | **RULED (HUM LEAD).** Burst duration stays bounded by **event count, not read length**. The rationale is stronger than the one the plan carried: a wall-clock bound is not merely redundant, it is **unsound** — read rate varies by voice and by machine, so a time bound would silently change *which events get read* depending on the host. Recorded as the reason G-10 retires the backstop. **Broadcaster is flagged as where duration may still need its own answer.** |
| **PL-10** | **PARTIALLY RULED.** The modal case is **not a defect and is verified**: `view.go:30` composites the modal *over* the content and `dashboard.go:479` handles `TickerBreakingMsg` with no modal guard, so the cue always lands and a modal can only occlude it. The remaining half — a second takeover while something unsynced is reading — awaits a concrete example, supplied below. |

### The concrete example PL-10 needs

Tracing the cue's lifecycle produced one, and it is the same asymmetry this feature exists to remove.

`TickerBreakingDoneMsg` — the message that releases the band back to normal rotation — is sent on
**one** path: the last line of the takeover closure (`app/ticker.go:426`). There are **five early
returns above it**, and none sends it: a failed attention hold, a failed burst head, `readBreaking`
returning false, a failed closing line, and the context ending mid-sequence.

Meanwhile the **audio** side of the same sequence is released unconditionally — the director's
`release`/`settle` runs whatever happens, which is why the comment at the third early return reads
"the director still restores." **It restores the audio. Nothing restores the band.**

So the shape is: **the audio path has guaranteed release; the visual path releases only on the happy
path.** Today this is near-unreachable — the only trigger is app shutdown, when it no longer matters.
**Under the Lineup it becomes ordinary**: a card can be discarded, cancelled or superseded mid-read,
and every one of those is an early return.

**So PL-10 is not "what if the ticker refuses" — it is "what guarantees the ticker is released."**
Proposed rule, for ratification: **the cue and its release are paired with the same guarantee the duck
already has** — whatever ends a card, `mastercontrol` emits the clear. Under Approach C that is
expressible as a rule about `Step`'s output rather than a discipline about call sites: any transition
out of ON AIR emits a release effect.

## Verified clean (stated, not padded)

- **p10:** 0 live findings, 7/7 tools RAN, no undeclared skip, run record current. No exemption added by this scope.
- **`app/cutover_latency_test.go`:** env-guarded skip with a documented invocation and an owner section (perf-protocol §6) — a documented instrument, not an abandoned test. It **fails rather than reporting zero** when the build yields no segments or never reaches the network, which is the fail-closed shape the Code Quality axis asks for.
- **Stray files:** `app/`, `cmd/watchpost/`, `modes/tty/tea_debug.log` are zero-byte, untracked and gitignored (`.gitignore:2`). Not a hygiene finding.
- **`AP-HIST-01`:** the PLAN artifacts describe current state and decisions; where history appears (the rejected approaches, the reverted attempts) it is deliberate rationale, which the axis permits.
- **R-12a:** the ticker callout carries text *and* colour via `category.Label()`; colour is not the only channel.

## Cross-cutting synthesis

**Convergence.** PL-1 and PL-3 are the same blind spot from two directions: the plan treated 2.14 s as
a fact to *route around* rather than a number to *attack*, and then wrote a pump that would have paid
it on the critical path anyway. Perf and Architect found it independently.

**Contradiction.** PL-1 is a direct contradiction *within* the PLAN bundle — the architecture says
effects run off the `Step` path; the build plan's pseudocode runs them on it. PL-5 is the same class
between documents. Two contradictions inside one bundle is the signal the Docs axis exists to catch.

**Composition.** PL-2 + PL-4 compose into the release's real exposure: a larger blast radius than
today **and** no runtime signal when it fires. Either alone is manageable; together they mean a wedged
schedule would be silent and undiagnosable — which is precisely RD-1's history repeating with a wider
radius.

**Risk status.** RD-3 (concurrency) was rated the highest open risk at DISCOVER and PL-1/PL-2 are its
first two concrete instances, which is mild confirmation the register pointed at the right thing —
while PL-9 shows the register is not yet rating likelihood.

## Disposition ledger

Every finding carries an explicit disposition; none is dropped silently.

| # | Disposition |
|---|---|
| PL-1 | **Fixed** — T2.1 now dispatches; the inline `run(fx)` is replaced and the reason recorded beside it |
| PL-2 | **Fixed** — DR-22 (pump supervision) + task T2.4 |
| PL-3 | **Fixed by measurement** — concurrency floors at 0.96 s vs 1.03 s; standby stands. perf-protocol §6 |
| PL-4 | **Fixed** — DR-23 (runtime observability) + task T3.6 |
| PL-5 | **Fixed** — `read-order-design.md` carries a superseded banner; `README-director-handoff.md` restated with a current-artifact index |
| PL-6 | **Fixed** — the effect set is enumerated and declared closed in `director-architecture.md` |
| PL-7 | **Fixed** — DR-11 names `globalfeed.MaxPerLane` (30) as the bound, so the read is a `range` over a bounded slice |
| PL-8 | **Fixed** — DR-7 requires the pre-build to yield to the alert path, with a test |
| PL-9 | **Fixed** — likelihood × impact table added to the risk register; RD-4 raised, RD-9 downgraded |
| PL-10 | **Ruled (HUM LEAD) and fixed** — DR-24 + task T3.5: any transition out of ON AIR emits a release effect |
| PL-11 | **Ruled (HUM LEAD)** — G-10 retired, with the soundness rationale recorded in `lineup-model.md` |
| PL-12 | **Fixed** — DR-21 and T3.4 carry the modal's AA-registration, keyboard and focus obligations |
| PL-13 | **Fixed** — the Approach C claim is narrowed to a pointer-to-read |
| PL-14 | **Deferred to BUILD exit** — the seven pending exemption entries are reconciled at the Phase 5 gate |
| PL-15 | **Recorded** — the false claim in `022e98a` stands in the record; the wording rule is adopted going forward |
| PL-16 | **Fixed** — Phase 5 names the build-log mirror as the ratification record |
| PL-17 | **Fixed** — instrument corrected, re-measured at n=5, every citation updated, and a seventh anti-vacuity rule added to the build plan |

## Verdict

**NO-GO as written → SHIP-WITH-CONDITIONS after remediation (2026-09-01).**

Both Criticals are closed: PL-1 by making the pump dispatch rather than run, PL-2 by DR-22 and T2.4.
PL-17, raised and fixed inside this round, was the most consequential of the three — it invalidated
the measurement PD-2 rested on, and it was found by the perf lens asking whether the baseline was
taken under realistic conditions.

**The conditions:** the three HUM LEAD rulings are given (PL-10, PL-11, and PL-3's go-ahead to
measure); the PLAN report is approved; and **a second red-team round runs after BUILD Phase 2**, per
Step 9 — this round's find-rate and the foundational scope both call for it, and PL-2's supervision
plus PL-1's dispatch add surface that has not had its own adversarial pass.

### The original verdict, for the record

**NO-GO as written.**

Two Critical findings block the transition to BUILD: **PL-1** (the pump's own pseudocode reintroduces
the blocking class Approach C was chosen to eliminate) and **PL-2** (an unsupervised single point of
failure with a wider blast radius than today's, with nothing requiring recovery).

**Both are plan-text edits with named tests. Neither changes the architecture** — Approach C survives
both, and PL-1 is in fact a contradiction *of* the architecture by the build plan rather than a flaw
in it. Once PL-1 and PL-2 are applied and the three rulings below are given, this converges to
**SHIP-WITH-CONDITIONS** and BUILD can start.

## Needing a HUM LEAD ruling

| # | Question |
|---|---|
| **PL-10** | What happens when the news ticker **cannot** comply with a cue — a modal is open, or a takeover is already drawn? |
| **PL-11** | Is burst **duration** deliberately unbounded now that the wall-clock backstop is retired, with M3 validating the read-length assumption — or is a stated bound restored? |
| **PL-3** | Should a concurrent `segments()` be measured before standby is built? (Recommendation: yes — it is one instrument run, and it may shrink T3.3.) |

## Round-2 assessment (Step 9)

**A second whole-base round is warranted after remediation**, on two of the three stated triggers:
**foundational scope** (this changes how audio is scheduled and how the system exists at runtime) and
**material find-rate** (two Criticals and nine Importants). The third — significant new code from
remediation — will apply if PL-2's supervision and PL-1's async dispatch add real surface.

The next round runs fresh lenses given these findings and the fix commits, told to re-verify the
fixes hold, hunt what this round missed, and attack what the fixes introduced.
