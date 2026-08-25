# Watchpost CLI — Objectives

Source of truth: [`../08-reports/project-brief.md`](../08-reports/project-brief.md) (APPROVED 2026-08-23).

## Problem Statement (locked, 5/5)

> Terminal-centric people who monitor weather across one or more locations — especially those in severe-weather or wildfire regions — have no single glanceable surface for current conditions, forecasts, and live alerts, and so they juggle multiple apps, sites, and radio, seeing alerts late or not at all.

## Metrics of Success (ratified)

| ID | Metric | Type | Target |
|---|---|---|---|
| M1 | Time to Situational Awareness | Primary | ≤ 3s warm / ≤ 8s cold, 10 locations |
| M2 | Alert Surfacing Latency | Primary | ≤ 60s from issuance |
| M3 | Alert Coverage | Primary | 100% NWS |
| M4 | Multi-Location Scale | Secondary | ≥ 25 locations within M1/M8/M9 |
| M5 | Machine-Mode Fidelity | Secondary | 100% TTY↔JSON parity |
| M6 | Component Upstream Yield | Tertiary | higher is better |
| M7 | Correction Count | Maintenance | lower is better |
| M8 | Memory Footprint | Primary | ≤ 40 MB @10 loc; ≤ 80 MB @25 loc + radio; flat heap over 1h |
| M9 | CPU Budget | Primary | ≤ 1% idle; ≤ 5% animated; ≤ 10% radio + animated |

## Scope of v0.1 (per OQ rulings)

NWS (US observations/forecast/alerts) · keyless global provider for non-US current/forecast (alerts are US-only) · multi-location views with dive-in · real-time alerts · NOAA weather radio — **best-effort** (community streams cover ~10% of transmitters; pure-Go playback; NWS text-product fallback) · `--json` / `--report-only` (exit codes 0/1/2) · `watchpost setup` · fire hotspots — **NOAA HMS keyless default + NIFC WFIGS incidents; NASA FIRMS as optional user key** (D-10/OQ-19) · configurable view playlist architecture.

**Repo status:** private until the go-studs distribution mechanism is decided (OQ-17 SHIP gate).

Post-v0.1: evacuation orders (R-9b, via NWS CAP events + opt-in county layers), paid-provider harmonization UX refinements, `--watch` JSON-Lines + `--fail-on-stale` (OQ-18).

## Directives

LEVEL-1 · SEV-0 (HUMAN LEAD) · FULL GIT · FULL DOCS · FULL REPORTS · FULL DIAGRAMS · FULL RCC · FULL PLAN · FULL TDD · Theme: BRTOPS
