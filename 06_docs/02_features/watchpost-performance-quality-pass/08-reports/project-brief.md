# Project Brief — Watchpost Performance & Quality Pass

| Field | Value |
|---|---|
| Report | project-brief v1.4.0 |
| Phase | pre-DISCOVER (collect-brief handoff) |
| Date | 2026-08-26 |
| Author of record | Branden Thompson (HUM LEAD) |
| Branch | `feature/watchpost-performance-quality-pass` (off `feature/watchpost-cli`) |
| Directives | LEVEL-1 · SEV-0 · HUM LEAD · FULL GIT · FULL DOCS · FULL REPORTS · FULL DIAGRAMS · FULL TDD |
| Status | APPROVED — HUM LEAD, 2026-08-26 ("APPROVED; Go 4 RCC") |

---

## Summary & Intent

Before Watchpost takes on new features, HUM LEAD wants the whole 0.9.4 codebase put under an adversarial,
evidence-led quality pass: code structure, code quality and architectural quality, with particular attention
to memory allocation and management, API connectivity and "chattiness", caching and memoization, and
opportunities to refactor, combine or split functionality. The trigger is an observation from a long-lived
session — the process was watched for many hours and its thread count appeared to grow slowly — and the
larger motive is foundation: a terminal application lives for days, and every future feature stands on this
code.

"No major changes" is an acceptable outcome. What is not acceptable is not knowing. The record — measured
baselines, attributed profiles, rationale for every change and for every non-change, tests that pin the
result — is a first-class deliverable, written so that junior human developers and future agents can use it
as a context corpus and as the comparison point for the next quality pass.

## Locked Problem Statement

> **"People who leave Watchpost running for days cannot tell whether its slowly rising resource use is
> normal or a defect that will eventually degrade the terminal session they live in — and the record
> cannot tell them either."**

| # | Criterion | Score | Evidence |
|---|---|---|---|
| 1 | Bad Outcome | ✓ | uncertainty today, a plausible degradation tomorrow, and a record that cannot answer |
| 2 | Affected Humans | ✓ | people on multi-day sessions (HUM LEAD now; every terminal-station user); future maintainers |
| 3 | Tech Agnostic | ✓ | no threads, heaps, caches, languages or libraries named |
| 4 | Non-prescriptive | ✓ | profiling, refactoring, documentation and "leave it" all remain valid answers |
| 5 | Verifiable | ✓ | sample the process over time; read the record for an answer |

**Score: 5/5 — LOCKED.** Ratified by HUM LEAD 2026-08-26 with this brief.

**Refinement trace.** Raw input named the symptom in solution terms ("31 threads / 193 MB … thread count
slowly growing"). One pass moved the numbers into evidence, named the affected humans (long-session users
and maintainers), and expressed the bad outcome as the *inability to tell* — which is what a quality pass
actually fixes, whether or not it changes code.

## Metrics of Success

| ID | Metric | Symbol | Type | Definition (direction) | Measured in |
|---|---|---|---|---|---|
| M1 | Long-run resource stability | `M-STAB` | Primary | Over ≥ 24 h with radio on: thread count and **physical footprint** (`vmmap`, RSS reported alongside) reach a plateau; slope after warm-up ≈ 0. Flat is the goal; lower is better | soak harness samples + attributed profiles → infra ledger |
| M2 | Steady-state chattiness | `M-CHAT` | Secondary | Outbound requests per provider per hour at steady state, **at unchanged data cadences** (lower is better) | an httpx request counter over a 1 h soak |
| M3 | Zero regression | `M-REG` | Primary | Every golden, test and gate green; launch → full view ≤ 550 ms warm; radio Synth and Relay pty smokes unchanged; the Linux protocol re-run PASS | `make verify`, pty smokes, release checklist |
| M4 | Code-quality floor | `M-P10` | Maintenance | P10 live findings 0; non-third-party exemptions ≤ 56 (lower is better); per-package coverage ≥ today's (app 16 % flagged for lift) | `a2dh p10 check`, `go test -cover` |
| M5 | Record completeness | `M-DOC` | Maintenance | Every change and every considered-and-rejected change carries rationale, evidence and a test or measurement; a future pass can diff against this one | the 7-folder feature record; REVIEW lens |

**Anti-solution check (Step 5).** A flat footprint bought by disabling caches raises `M-CHAT`; fewer
requests bought by fetching less often breaks the "unchanged cadences" clause; fewer exemptions bought by
deleting checks fails `M-REG`; a record padded with ceremony fails `M-DOC`'s "rationale and evidence" test.
The five are read together.

**Provisional targets (OQ-5 — HUM LEAD asked for a recommendation; to be confirmed against DISCOVER's
measurements).** For a Go TUI that holds 60 locations of weather, a 27k-point fire archive, an embedded
gazetteer, an HTTP cache and a rendered-audio cache, and drives a pure-Go audio device:

| Quantity | Today (evidence) | Provisional target |
|---|---|---|
| Idle footprint, radio off | 78 MB (exit measurement) | ≤ 90 MB |
| Plateau footprint, radio on, 24 h | 116–175 MB sampled; peak 290 MB | ≤ 160 MB plateau; peak ≤ 220 MB |
| Threads after 24 h | 30–31 | ≤ 32, and **attributed** (Go vs OS audio) |
| Requests/hour, 10 favourites + 50 RECENT | unmeasured | measure first; then no more than cadence math predicts |
| Warm launch → full view | 550 ms | ≤ 550 ms |

## Requirements

| ID | Requirement |
|---|---|
| R1 | No functional regression — behaviour, output, keys, goldens, schema |
| R2 | P10 and code style adhered to; exemptions only where they carry real value, never ceremony |
| R3 | Every resulting change documented to the quality lenses — junior human developers first |
| R4 | Best practices adhered to (A2DH implementation skills, TDD) |
| R5 | No performance regression — this pass only makes things better |
| R6 | Radio / Synth functionality cannot regress |
| R7 | The record is a corpus: baselines, profiles, rationale, tests, and a comparison method for future passes |

## Technical Constraints

- Go 1.25 module; pure-Go audio (`oto`) and the Piper subprocess are the platform boundary; public repo, MIT.
- `third_party/go-studs` is HUM LEAD's own kit (OQ-1): findings there are **captured and, with documentation
  and HUM LEAD approval, changed in the local copy** — then a candidate for upstream.
- `tools/` is out of scope except where a finding lands there.
- Profiling hooks may ship in the binary (OQ-2 approved) provided they are opt-in and cost nothing idle.
- Release cadence decided by the final shape of change (OQ-7): point releases per landed batch, or one minor.

## Other Considerations

- **Prior art in the record:** infra ledger baselines (UAT 73/74/92/98/123 — two 1 h soaks, no trend);
  debugging ledger D1–D10 (pty harness lessons); red-team round 3 P1 (HMS parsed once per RECENT scheduler,
  4.5 GB/tick, memoized) and P2–P6 (fail-soft); REVIEW C2 (snapshot cloning per broadcast cycle).
- **Size:** 33 packages, 16.5k LOC + 9.6k test LOC; largest units `modes/tty/dashboard.go` 3,332 lines,
  `platform/render/render.go` 1,188, `app/dashboard.go` 797, `domains/weather/nws/provider.go` 760.
- **Exemptions:** 132 P10 exemptions, 56 outside `third_party/`.
- **UX nits found during the pass are fixed in-pass** (OQ-6) since the code is already open.
- **Stakeholders:** HUM LEAD (approves architecture and every go-studs change); junior human developers
  (documentation audience); future agents; end users on multi-day sessions (macOS and Arch Linux).

## Evidence captured at intake (2026-08-26)

Live process **PID 67943** (`~/.local/bin/watchpost` v0.9.4, up 9 h 26 m by the process clock):

| Fact | Value |
|---|---|
| Memory | `ps` RSS 249–254 MB; `vmmap` physical footprint **175 MB → 116 MB** across two samples an hour apart (peak 290 MB); malloc zones ≈ 10 MB, so the bulk is the Go-managed heap — unattributable in the shipped build (no profile hook) |
| Threads | 31 → 30: 23 Go runtime threads created at launch; 5 Apple audio threads created on first tune-in (`caulk.messenger`, `AQConverterThread`, `IOThread`); 2–3 macOS libdispatch workqueue threads that appear and retire (`start_wqthread → __workq_kernreturn`, one serving the `AQClient` queue) |
| Reading | Two bounded ratchets, neither a leak on this evidence: Go never reaps idle OS threads (the count follows the peak of concurrent blocking syscalls — `say` subprocesses, DNS, cache file I/O), and Apple's GCD pool grows and shrinks by ones. The gap this pass closes is *attribution over 24 h+*, not a known defect |
| Baseline sampler | hourly `ps`/`vmmap`/thread samples of PID 67943 for 24 h → `02-analysis/baseline-pid67943.log` (OQ-3) |

## Discovery Handoff Package

### Areas to Investigate

| # | Area | Concrete artefacts |
|---|---|---|
| A1 | Thread inventory, attributed over 24 h | `runtime.ThreadCreateProfile` / `NumGoroutine` via an opt-in hook; blocking-syscall sources: `domains/radio/synth/voice.go` (`say`), `platform/httpx` (DNS, disk cache), `domains/radio/player` (oto); separate the GCD pool |
| A2 | Heap attribution | heap profile of a long-running instance; suspects by design: `platform/snapshot/assembler.go Snapshot()` deep copy per publish, httpx memory tier + disk cache, synth PCM cache (40 segments), HMS memo (27.5k points), spectrum/tap buffers, per-frame render with the visualizer at 50 ms |
| A3 | Chattiness at steady state | requests/hour/provider: priority tiers (`app/dashboard.go startPriority`), the RECENT pipeline's per-location schedulers (`newFor`), FIRMS 2 req/location/10 min, HMS/WFIGS per 10 min, singleflight and cache-header honouring in `platform/httpx`; the `report` path's fetch fan-out |
| A4 | Structure | split candidates: `modes/tty/dashboard.go` (model, keys, modals, detail, radio panel, setup), `platform/render/render.go`, `app/dashboard.go`; the 56 exemptions — which still buy value; `app` coverage 16 % |
| A5 | Caching/memoization inventory | every cache with owner, bound, invalidation and measured hit rate: httpx tiers, synth PCM, geodata index, theme registry, HMS memo, `detailLines`/row rendering per frame |
| A6 | go-studs seams | what the app works around in the kit (raw SGR path, background colours, KeyCap) — candidates for local changes with HUM LEAD approval |

### Stakeholders to Consider

HUM LEAD (architecture and go-studs approvals; runs the Linux re-validation) · junior human developers (the
documentation must read for them) · future agents (the corpus) · end users on multi-day sessions · upstream
go-studs (changes captured for upstreaming).

### Risk Signals

| # | Risk | Why |
|---|---|---|
| RS-1 | Refactor-induced regression in the radio path | the deck/engine/source concurrency is the most stateful code; a "harmless" split is where it bites — mitigated by pinned tests, pty smokes, the soak and R6 |
| RS-2 | Scope creep into features | the pass is quality only; the brief and PLAN must say what "fix in-pass" excludes |
| RS-3 | Doc-heavy scope vs. time | R3/R7 are the deliverable; PLAN sizes the record per batch, not at the end |
| RS-4 | P10 ceremony vs. value | HUM LEAD's own warning; every exemption and every check must argue its value |
| RS-5 | Measurement noise | `ps` RSS and `vmmap` footprint disagree by ~70 MB on one process and footprint swung 175 → 116 MB in an hour; M1 names footprint and samples long |
| RS-6 | Finding nothing | possible and acceptable — the record must then say so with evidence, not silence |

### Open Questions

| ID | Question | Status |
|---|---|---|
| OQ-1 | Scope of `third_party/go-studs` and `tools/` | **Resolved**: go-studs in scope on the capture → document → approve → change basis; tools out unless a finding lands there |
| OQ-2 | Profiling hook in the shipped binary | **Approved** (opt-in, zero idle cost) |
| OQ-3 | Keep PID 67943 as the 24 h "before" baseline | **Approved**; sampler running |
| OQ-4 | Metric unit | **Approved**: physical footprint, RSS alongside |
| OQ-5 | Numeric targets | **Open** — provisional targets above; confirm after DISCOVER's measurements |
| OQ-6 | UX nits found in-pass | **Resolved**: fix in-pass |
| OQ-7 | Release cadence | **Open** — decided by the final shape of change |
| OQ-8 | How long is "long-run" for M1 | **Resolved** (HUM LEAD): multi-day minimum on both platforms; the expectation is continuous operation for weeks or months, like a server. M1's bar is therefore "no growth term at all", proven by attribution, not just a 24 h plateau |
| OQ-9 | Should the `report` path (and the fetch fan-out generally) be in scope? | **Resolved** (HUM LEAD): yes — every aspect of the app, including how requests are fetched and fanned out; the standing priority is end-user performance (it must be and feel fast), which bounds but does not exclude fetch-plan changes |

### Problem statement status
Refined via `refine-problem-statement` (5/5, scorecard above); measurements validated for anti-solutions.

### Sharpening observations
- SH-1: the symptom (threads) and the intent (whole-codebase quality) are different sizes; the brief carries
  both by making the problem the *ability to tell* and the requirements the *foundation*.
- SH-2: "no major changes" is a legitimate outcome — the record is the product either way.
