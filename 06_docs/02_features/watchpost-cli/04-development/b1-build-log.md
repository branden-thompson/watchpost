# B1a/B1b Build Log — Snapshot, NWS, Report Mode, Scheduler, Replay Harness

| Field | Value |
|---|---|
| Milestones | B1a + B1b (architecture §8) · BUILD · SEV-0 |
| Date | 2026-08-23 |
| Gate | HUM LEAD GO after B0 exit |

## Delivered

**B1a:** `platform/snapshot` (contract types §10.1, single-assembler merge, harmonize/fill_from per OQ-9, collections empty-not-null, race-tested immutable publications) · NWS provider (fixture-TDD; SI normalization; dual-UGC batched alerts; points cache) · Open-Meteo geocoder (query-matching-zip rule) · `modes/report` (--json + plain; M5 bidirectional parity with distinct sentinels + value-level reverse; no-secret golden; exit 0/1/2 with obs_stale carve-out) · `app.ReportOnce` (+>2h obs_stale warnings) · cobra tree · `pkg/schema` reflection generator (additionalProperties:false, nullable pointers, depth-bounded) + published `watchpost-report.v1.0.0-rc.schema.json` + `watchpost schema` · **live demo:** `report 92057` returned a real Extreme Heat Watch, exit 0.

**B1b:** `platform/sched` (Clock seam, tier cadence = single freshness authority, publish-per-cycle, idempotent Stop; fake-clock TDD) · **alert replay harness green: M3 100% coverage + M2 ≤60s asserted against fixture timeline + cancellation honesty** · `tools/alertrec` fixture recorder (statically bounded polling) · **RS-20 bench: 162µs / 984KB / 234 allocs per 25-loc × 2-provider × 168h snapshot rebuild → CLOSED by measurement.**

## D-19 (mock rulings) captured in 09-view-mocks/design-decisions.md; mocks arrived early (M-V1/2/3/5/6 + radio) — B3 unblocked when reached.

## P10 posture

0 live findings. Remediated in-line: alertrec loop statically bounded; sched cycle preconditions; fillFrom pointer flattening; mapAlert extraction; schema recursion depth-bounded. Exemptions added (ALL flagged for HUM LEAD ratification at the B1 exit gate): density for snapshot/nws/openmeteo/report/schema/sched/alertrec under the approved B0 pattern; pkg/schema reflect/recursion/size (dev-time generator, test-pinned); sched Now/After P10-01 false positives (syntactic cross-package); runTier main-loop allowlist case.

## Gates at close

`make verify` ALL GREEN · 11 packages `-race` ok · p10 0 live · schema validation tests green.

## B1 exit red-team — disposition ledger (rounds: code axis + 5-lens sectioned)

Verdicts: Code HOLD→remediated · Hygiene CONDITIONAL · Docs REVISE→fixed · Business SWC · InfoSec PASS · Junior-dev CONDITIONAL→fixed. Watermark sweeps clean (both dispatches, independent greps).

| Finding | Sev | Disposition |
|---|---|---|
| Code#1 mapAlert truncated CAP AffectedZones (probe-confirmed) | Critical | **Fixed** (two-pass) + regression via replay-through-REAL-NWS-provider test (also closes Code#7 harness-fidelity) |
| Code#2 harmonize/fillFrom untested despite TDD claim | Important | **Fixed**: harmonize_test.go (nws-wins, fill_from provenance, no splicing); TDD claim now backed |
| Code#3 fill_from staleness cutoff silently missing | Important | **Marked DEFERRED→B5** in code; B5 harmonization goldens must cover (ledger row) |
| Code#4 M5 forward parity not fail-closed | Important | **Spec corrected** to ratified granularity (JSON ⊇ TTY; TTY may summarize) per State-Guarantees calibration; reverse value-level test is load-bearing |
| Code#5 sched warn-then-panic | Important | **Fixed**: New returns error; refusal test added |
| Code#6 stale nws cache comment | Minor | **Fixed**; daily-refresh = B3 task (ledger row below) |
| Code#8 defer os.Exit | Minor | **Fixed**: typed exitCodeError mapped in main + range invariant |
| Code#9 Hints race / OnPublish concurrency | Minor | **Fixed**: deep copy + documented |
| Code#10 dead flag, double publish | Minor | **Fixed** |
| H-1 exemption count 17 not 9 | Minor | **Corrected here**: 17 entries await ratification (list at gate) |
| H-2 mixed commit e140c79 (mocks swept into feat) | Minor | **Ledger acknowledgment** (self-reported; human artifacts get their own docs commits hereafter) |
| H-3 ca22487 coarse granularity | Minor | Noted for B3 |
| H-4 .DS_Store | Minor | **Fixed** (removed + ignored) |
| D-1/D-2/J-1 extending.md stale banner, wrong actor (assembler vs scheduler), missing registry | Important | **Fixed**: status rewritten "as of B1"; scheduler injects Hints; registry marked B3 |
| D-3 fixture provenance overstated | Minor | **Fixed**: basic.jsonl marked SYNTHETIC |
| D-4 §8 delivered notes; §10.3 naming | Minor | **Fixed** (✅ annotations; v1.0.0-rc) |
| D-5 nws refresh ledger row missing | Minor | **Fixed** — see Open ledger below |
| B-1 forecast tier unexercised; scheduler test-only | Important | **Fixed**: TestForecastTierFires; §8 B1b cell annotated (runtime wiring = B3 by design — report mode is one-shot) |
| B-2 demo evidence prose-only | Minor | **Fixed**: b1-demo-transcript.txt committed (fresh live run) |
| S-1 UserAgent embeds personal email | Minor | Accepted for now (AI-1 requires contact); TODO(pre-public SHIP): config-sourced |
| S-2 UGC shape validation | Minor | **Transferred→B3** (zone format check) |
| J-2 alertrec undiscoverable | Minor | **Fixed**: extending.md fixture-recording paragraph |

**Open ledger (carried):** nws points-cache daily refresh (B3 long-running mode) · fill_from staleness cutoff (B5 goldens) · UGC shape validation (B3) · UserAgent config-sourcing (SHIP) · config→Hints wiring (B3) · real-feed alertrec fixture recording + replay (VALIDATE) · M1 instrumentation (B3).
