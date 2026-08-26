# SHIP Report — Watchpost CLI 0.9.4

| Field | Value |
|---|---|
| Phase | SHIP (li-A2DH: VALIDATE → **SHIP** → REFLECT) · SEV-0 · HUM LEAD |
| Entry | VALIDATE approved 2026-08-25 ("validate approved"); release checklist: no ☐ left |
| What shipped | `github.com/branden-thompson/watchpost` — public, MIT · releases `v0.9.0` → `v0.9.4` (latest `v0.9.4`) · five binaries + `checksums.txt` + LICENSE + THIRD_PARTY_LICENSES.md per release · `curl -fsSL https://raw.githubusercontent.com/branden-thompson/watchpost/main/scripts/install.sh \| sh` |
| Verdict | **SHIPPED** |

## What a user gets

`watchpost` — a terminal-native live weather station: NWS observations, forecasts and alerts for a
10-favourite watchlist plus a 50-deep RECENT list; marine buoys, tides and currents where the coast is
in reach; wildfire hotspots and named incidents (NOAA HMS, NIFC WFIGS, NASA FIRMS with a free key) as
a row mark and a FIRE section; NOAA Weather Radio — live relays and a synthesized broadcast with the
lead that points to the real transmitter and frequency, the forecast, a Fire and Hotspot report, and
six correspondent voices on Linux/Windows (macOS uses its own); one-shot plain and JSON reports with a
published schema; themes; a Setup window on first run. Install is one line, verified by SHA-256.

## Ship mechanics (for the record)

- `main` = one squash commit of the feature tree plus fix-forward commits (never a rebase); the
  working branch `feature/watchpost-cli` (~190 commits) stays private to this machine.
- Every release: tag `v*` → `release.yml` (verify → matrix → installer smoke test → publish). Runs
  green for `v0.9.1`–`v0.9.4` first time; `v0.9.0` needed three CI-only fixes first.
- Identity: personal account and key throughout (verified before each outward action); the
  employer's `gh` account and SSH key were never used. Details in the publish memory / UAT 117.

## Versions

| Tag | Content |
|---|---|
| v0.9.0 | The 0.9.0 exit: everything from B0 through B5 (fire), REVIEW fixes |
| v0.9.1 | Linux/Windows voice profiles (curated Piper catalogue, download on first use) |
| v0.9.2 | Voice chooser explains the wait (download progress, model loading) |
| v0.9.3 | Narration reads road and highway abbreviations as words |
| v0.9.4 | `◆` in Location Details FIRE rows; README screenshots |

## Conditions carried into 1.0 (from §11.9 and the reports)

Schema `1.0.0-rc` → `1.0.0` ratification (HUM LEAD); keyless global weather provider (non-US);
`report --every` (the screen-reader live surface); `--ascii` / `--no-animation` flags; WAT tone and
alert interrupt on the radio; fire spread direction when a feed can give it; user theme files
validated for raw-SGR shape.

## Source Documents

`08-reports/validate-report.md` · `08-reports/review-report.md` · `08-reports/build-report.md` ·
`07-readiness/release-checklist.md` · `CHANGELOG.md`.
