# Plan Report — Watchpost CLI

| Field | Value |
|---|---|
| Report | plan-report (FULL depth) |
| Feature / Branch | watchpost-cli · `feature/watchpost-cli` |
| Phase | PLAN exit · SEV-0 · HUMAN LEAD · Theme BRTOPS |
| Date | 2026-08-23 |
| Status | **APPROVED — HUM LEAD, 2026-08-23 ("G-9: a. Approved for Exit")** |

## 1. Executive Summary

PLAN produced, and the HUM LEAD approved in-phase: the **project structure** (Option C, domain-first + thin platform — D-14), the **architecture & phased roadmap** (Snapshot pivot, 5-tier scheduler, render seam, LIVE+SYNTH radio, hot-swappable KeyMap — D-15/D-16), backed by **measured evidence** (spikes S1/S2) and **source-verified dependency facts** (G-5). The PLAN-exit red-team (4 axes + phase lens + 4 personas, then a targeted round 2) found no Critical and no architecture error; its ~37 findings were dominated by named-but-undefined specifications, all closed by the §10 addendum in `architecture.md` and hygiene fixes, with residuals transferred to BUILD entry. Recommendation: **proceed to BUILD (milestone B0)**.

## 2. Plan of Record (pointers, not duplicates)

| Artifact | Content | Status |
|---|---|---|
| `03-architecture-design/structure-proposal.md` | Option C tree, conventions, dependency rule | APPROVED (D-14) |
| `03-architecture-design/architecture.md` | System design §1–§9 + specification addendum §10 (types, taxonomies, radio mechanics, exec safety, a11y mechanisms, test seams, RS-19..21, BUILD-entry preconditions) | APPROVED (D-16) + §10 remediations this exit |
| `03-architecture-design/risk-register-pxi.md` | P×I-scored register; RS-8/RS-18 closed by measurement | Approved with packet |
| `03-architecture-design/spikes/{S1,S2,G5}` | Measured radio pipeline (1.83% CPU, flat heap), geodata memory (~18 MB total, 7 µs lookups), bubbletea v2.0.9 claims table | Evidence of record; code/data durable in `04-development/spikes/` |

**Dependency pins:** Go ≥1.24 · `charm.land/bubbletea/v2 v2.0.9` · `lipgloss/v2 v2.0.6` · `bubbles/v2 v2.2.0` · `oto/v3 v3.4.1` · `go-mp3 v0.3.4` (vendored) · go-studs local `replace` · `jsonschema/v6`.

## 3. Roadmap of record

B0 skeleton+gates(+release matrix stub, §10.12) → B1a snapshot/NWS/report (schema v1.0-**rc**) → B1b scheduler/replay-harness/25-loc bench → B2 locations/setup → B3 TTY core (+stale badges, mocks NEED INPUT w/ placeholder fallback) → B4 radio LIVE+SYNTH (mocks NEED INPUT w/ fallback) → B5 multi-provider/fire (+**schema v1.0 ratification gate**) → B6 playlist/polish/soak(+install.sh dry-run). **T-M (curl install, D-17)** delivery rides the SHIP gate (public repo required); dry-run proves it earlier (§10.12). Every R-1..R-13 and T-A..T-L maps to a milestone (T-J → SHIP). v0.2 deferrals unchanged.

## 4. Decisions this phase

D-14 (structure Option C) · D-15 (keybindings hot-swappable; only `?` locked) · D-16 (architecture + PD-1..PD-4: no `internal/` wrap v0.1; no stored/emitted diffs; single golden system; two height breakpoints). **Alternatives-considered record (red-team F-4):** structure had three authored options; Snapshot-pivot vs per-view fetching was decided by M5-parity-by-construction (a per-view fetch model cannot guarantee TTY/JSON parity structurally); single-assembler vs per-domain locks chosen for race-freedom simplicity with RS-20 bench as the escape hatch; 5-tier scheduler is entailed by the five distinct cadence classes, not invented.

## 5. Critical Analysis

`red-team: SHIP-WITH-CONDITIONS · multi-agent (4 sectioned dispatches + targeted round 2) · scope:feature(PLAN corpus) · personas:[infosec, a11y, perf, junior-dev]`

Full findings + disposition ledger: **`08-reports/red-team-plan.md`**. Round-1 verdicts: Code SWC · Hygiene CONDITIONAL · Docs REVISE · Business SWC · Phase SWC · InfoSec CONDITIONAL · A11y CONDITIONAL · Perf PASS. Watermark sweeps clean. Standing rule adopted from the recurring meta-finding: *a research recommendation is not real until it has an architecture line.* Round-2 outcome: converging — 5/7 fix groups verified holding, spike numbers independently reproduced; its 3 Majors fixed in §10.11/doc edits except G-9, which is escalated to this gate. Convergence closes with the HUM LEAD spot-check + G-9 ruling.

## 6. Conditions carried to BUILD

**B0 (BUILD entry, §10.10):** `docs/extending.md` (two worked examples from the junior-dev traces) · activate `languages:[go]` + `a2dh p10 check` · `.gitignore` Go artifacts · bubbles/help adapter spike at B3 entry · render primitives on demand.
**Standing BUILD rules:** severity-as-text; `NextFrame` only from `Update`; argv-only TTS exec with hostile-fixture tests; no-secret-in-output golden; https-preferred streams; dataset+transmitter-table checksums; `sync.Once` oto Context never closed.
**Gates ahead:** B5 schema v1.0 ratification (HUM LEAD) · SHIP gate: go-studs distribution + IP/OSPO + docs scrub + NWS roadmap check + OQ-14 revisit.

## 7. Metrics status

M1 (B3, pass/fail on fixtures) · M2/M3 (B1b replay harness, fake clock) · M4/M8/M9 (B1b bench + B6 soak; budgets **strengthened by measurement**: M9 radio 5× margin, M8 ~22 MB headroom) · M5 (B1a parity + secret golden) · M6 (post-SHIP only, per G-4a redefinition) · M7 (REFLECT).

## 8. Exit Gate Checklist (PLAN)

| Gate | Status |
|---|---|
| plan_documented (FULL PLAN bar) | PASS — §1–§10 incl. types/taxonomies after remediation |
| structure_approved | PASS (D-14) |
| architecture_approved | PASS (D-16 + presented remediations) |
| risks_scored (P×I) | PASS (+RS-19..21) |
| spikes_complete | PASS — S1/S2 measured, G-5 verified |
| critical_analysis_complete | PASS — rounds 1+2, ledger in red-team-plan.md |
| report_published | PASS on commit |
| a2dh validate 100% | PASS (see exit run) |
| **G-9 ruling** | **RULED (a)** — `report --every` ratified as the R-12d screen-reader live surface (plain-text only; JSON `--watch` stays v0.2). OQ-18 amended accordingly (D-18). |
| human_approval | **PASS** — D-18, 2026-08-23 |

## 9. Recommendation

Approve PLAN exit → **PHASE TRANSITION: PLAN → BUILD (CODE)**, starting at milestone B0. Mock schedule M-V1..M-V7 recorded (§10.12): first asks are **M-V6 (setup view) at B2 entry**, then **M-V1/M-V2 (dashboard + detail) at B3 entry** — I will ping you at each.
