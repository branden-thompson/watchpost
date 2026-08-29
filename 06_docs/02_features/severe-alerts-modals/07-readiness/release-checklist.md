# Release checklist — Watchpost 0.13.0 (severe-alerts-modals)

Owner: HUM LEAD. Status legend: ☐ open · ☑ done · ☒ decided-no. Every ☐ that stays open at SHIP is
recorded as accepted risk in the ship report.

## A. Decisions the human owns (SEV-0 authority)

| # | Decision | Status |
|---|---|---|
| A1 | `--ascii` header form: spread `E V E N T` (the ratified mock) vs plain `EVENT` (objectives FR-13); the `>` pointer/play mark collision (B-08a/b) | ☑ plain `EVENT`, mark `*` (2026-08-29) |
| A2 | Light-background terminals: supported (a light token set + the AA gate over `WindowBGLight`) or documented as unsupported (R5-C-03) | ☑ Watchpost Light (2026-08-29) |
| A3 | The P10 ledger rows (16, listed in `gates.md`) — ratify | ☑ (2026-08-29) |
| A4 | README screenshots re-captured on the 0.13.0 UI (dashboard, the window, Details) — HUM LEAD capture, as for 0.12.0 | ☑ 2026-08-29 — 14 captures: dashboard, themes, status, the player on air, the window as a six-frame animation, the [A] modal as a three-frame animation |
| A5 | Version `v0.13.0`; CHANGELOG date at SHIP | ☑ 2026-08-29 — CHANGELOG dated; the version is the tag (`make` stamps `git describe`), so the bump commit is the release commit itself |

## B. Gates (evidence in `gates.md`)

| # | Gate | Status |
|---|---|---|
| B1 | `make verify` — ALL GATES GREEN | ☑ (every commit) |
| B2 | `make p10` — 0 live, 0 unmatched | ☑ |
| B3 | `a2dh validate` — 18/18 | ☑ |
| B4 | Allocation budgets within pins | ☑ |
| B5 | `make pty-severe` | ☑ |
| B6 | Red-team BUILD (rounds 3, 4) and REVIEW (round 5) — findings dispositioned | ☑ see `08-reports/` |
| B7 | **R6 relay + audio smoke — macOS and Linux (BLOCKING)** | ☑ macOS 2026-08-29 (HUM LEAD); ☐ Linux post-release (accepted risk); ☐ takeover pause/resume later 2026-08-29 (accepted risk) |
| B8 | 1-hour PPROF soak (RS-8) | ☑ 2026-08-29 — no trend (`07-readiness/validate/soak-1h.csv`) |
| B9 | Linux: `make pty-severe` and the race suite (`linux-validation-protocol.md`) | ☐ post-release (accepted risk); the release PR's Linux CI runs the race suite |
| B10 | Live feeds smoke (`WATCHPOST_LIVE=1`) | ☑ 2026-08-29 — USGS 3 · NHC 2 · NWS 0 (curated query; raw 192/192 decoded) |

## C. Artifacts

| # | Artifact | Status |
|---|---|---|
| C1 | README (0.13.0 keys, Setup, scripts, rules, diagnostics; screenshots per A4) | ☑ |
| C2 | CHANGELOG 0.13.0 (Added / Changed / Fixed) | ☑ |
| C3 | 7-folder docs complete for SEV-0 (01–07 + 08-reports) | ☑ |
| C4 | REVIEW report presented and approved | ☑ 2026-08-29 |
| C5 | VALIDATE report presented and signed off | ☑ 2026-08-29 |

## D. Ship steps (FULL GIT)

1. ☑ VALIDATE exit approved → `release/v0.13.0` (`828feda`) from `feature/severe-alerts-modals`.
2. ☑ PR #2 to `main` with the harness template (problem statement and metrics read forward from
   `08-reports/project-brief.md`); Linux CI green (race + pty) on the push run and the PR run.
3. ☑ Squash-merged (`baec8b0`); tag `v0.13.0`; `release.yml` built the matrix and published (run
   33264986559, green first time); the installed artifact reports 0.13.0.
4. ☑ Release and feature branches deleted; `main` (dev trunk) at the feature tip. → DEBRIEF (REFLECT)
   completes `06-key_learnings/`.
