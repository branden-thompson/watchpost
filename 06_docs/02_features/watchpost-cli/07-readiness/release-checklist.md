# Release checklist — Watchpost CLI 0.9.0

Owner: HUM LEAD. Status legend: ☐ open · ☑ done · ☒ decided-no. Every ☐ that stays open at SHIP is
recorded as accepted risk in the ship report.

## A. Decisions the human owns (SEV-0 authority)

| # | Decision | Options | Status |
|---|----------|---------|--------|
| A1 | Repository visibility for the installer URL | **Public** (HUM LEAD, 2026-08-25) — `curl -fsSL https://raw.githubusercontent.com/branden-thompson/watchpost/main/scripts/install.sh \| sh` works as written; the token/alternate-host paths in the installer stay as options | ☑ |
| A2 | Licence for the code | **MIT** (HUM LEAD, 2026-08-25): free to use with attribution — the MIT notice requirement is the attribution. `LICENSE` added; README states it | ☑ |
| A3 | go-studs distribution | **In-repo copy** (HUM LEAD, 2026-08-25): `third_party/go-studs` (rendering, components, theme, tokens; MIT, same author; no tests/docs) as in-module packages with rewritten import paths — no `replace` directive (the previous one pointed at an absolute path on one Mac, so no other machine could build). `scripts/sync-go-studs.sh` refreshes it; NOTICE.md records the upstream commit. No CI secret needed | ☑ |
| A4 | Version | `v0.9.0` tag on the release commit; `make VERSION=0.9.0 build` stamps it (`watchpost --version`) | ☐ |

## B. Gates (evidence required, run as separate steps)

| # | Gate | Evidence | Status |
|---|------|----------|--------|
| B1 | `make verify` (fmt, vet, race, import direction, watermark, positive controls) | ALL GATES GREEN | ☑ (every UAT commit) |
| B2 | `a2dh p10 check` — 0 live findings | ☑ |
| B3 | `a2dh validate` — 18/18 | ☑ |
| B4 | `golangci-lint`, `staticcheck` — 0 issues | ☑ |
| B5 | `make release-matrix` — 5 targets, CGO off | ☑ (2026-08-25) |
| B6 | `make install-test` — install + tamper control | ☑ (2026-08-25) |
| B7 | Red-team BUILD exit (4 lenses: security, concurrency, fail-soft, Linux) — findings dispositioned | see `08-reports/red-team-build.md` |
| B8 | Perf: warm launch→full view ≤ 3 s; cold ≤ 8 s; steady threads ≤ 30; RSS ≤ 100 MB | warm 550 ms · 23 threads · 78 MB (2026-08-25); cold: see infra ledger |
| B9 | Linux end-to-end on a second machine (`linux-validation-protocol.md`) | ☐ — the 0.9.0 → 1.0 gate |

## C. Artifacts

| # | Artifact | Status |
|---|----------|--------|
| C1 | `README.md` (install, first run, controls, data credits) | ☑ |
| C2 | `CHANGELOG.md` 0.9.0 | ☑ |
| C3 | `LICENSE` (MIT) | ☑ |
| C4 | `.github/workflows/release.yml` (tag → verify → matrix → installer test → release) | ☑ untested until the first tag push |
| C5 | 7-folder docs complete for SEV-0 (01–07) | ☑ with this folder |
| C6 | BUILD exit report presented and approved | ☐ |

## D. Ship steps (in order, each gated on the previous)

1. Resolve A1–A4. 2. Merge `feature/watchpost-cli` → `release/v0.9.0` → PR to `main` (A2DH
template; never push `main` directly). 3. Tag `v0.9.0` on `main`; the release workflow publishes the
assets. 4. On the Linux laptop: run the validation protocol against the published installer. 5.
Record results in the VALIDATE report; fix-forward as `v0.9.x` if needed. 6. REFLECT.
