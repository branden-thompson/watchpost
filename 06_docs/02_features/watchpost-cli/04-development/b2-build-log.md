# B2 Build Log — Locations, Type-Ahead, Setup Wizard

| Field | Value |
|---|---|
| Milestone | B2 (architecture §8) · BUILD · SEV-0 |
| Date | 2026-08-23/24 |
| Gate | D-20 GO |

## Delivered

- `domains/locations/geodata`: go:embed GeoNames cities15000 + US postal (CC-BY, snapshot 2026-08-23; S2 compact design). 34k cities / 41k zips; <10ms type-ahead budget test (7µs-class measured in S2); representative-zip per AI-8 (query-match wins, else lowest for place, else nearest centroid). Binary: geodata +1.3MB as S2 predicted (11.8MB at the geodata commit); B2 CLOSE total 14.4MB — the wizard's charm deps added ~2.5MB (D1 correction).
- `domains/locations.Resolver`: embedded-first; Open-Meteo fallback with `geocode_fallback` warning; type-ahead NEVER hits the network (AI-8 ToS); "City, ST" disambiguation (Portland vs Portland, ME pinned); exact-name rule for full resolves. Wired into `watchpost report` — verified live: `report "Portland, ME"` → 04101 + KPWM obs, offline-resolved.
- `app.SetupModel` + `watchpost setup` (bubbletea v2 FIRST USE; API spike run first per AP-ASSUME-01 — caught tea.View.Content vs assumed .String()): M-V6 anatomy; Q1 type-ahead w/ zip-adorned hints; Q2 greyed "coming in v0.2" (D-19); Q3 optional FIRMS key (OQ-19); esc saves nothing; finish persists 0600 config. 5 model flow tests.
- **Scripted-PTY machine verification (calibration honored):** literal prompt-matching fails by design under the v2 cell renderer (text fragments between cursor moves — recorded as the finding for future PTY tests); outcome-based script (blind keys → "Setup saved"; config content/perms asserted by the invoking commands, now documented in the script header) **PASSED** against the real binary: Oceanside, CA / 92057 / 0600. Script: `b2-setup-pty.expect`.
- **PTY-found data bug fixed:** zip-path resolve saved tz='' — TZ now backfilled from the city index; regression test added; PTY re-run confirms `tz = 'America/Los_Angeles'`.

## P10 posture
0 live. (Exemption delta this milestone: 7 entries — 3 density + 4 recursion false-positives; the round briefing said 8, reconciled here — H3.) New exemptions (all D-20-ratified classes): locations/geodata/app density ×3; Resolve/View/Update/Save syntactic-recursion false positives ×4; field() rewritten to bounded-counter form (real fix).

## Open ledger additions
- v2 cell-renderer defeats literal expect matching — future PTY tests are outcome-based (documented here for VALIDATE).
- Setup Q2 becomes real at v0.2 (location services); wizard re-run flow (`watchpost setup` over existing config) currently overwrites priority[0] only — B3 add-location modal (M-V3) handles multi-location.

## Gates at close
verify ALL GREEN · 14 pkgs `-race` ok · p10 0 live · validate 18/18 · PTY outcome PASS.

## B2 exit red-team — disposition ledger

Code axis: BLOCK→remediated. Sectioned: Hygiene PASS-w/-concerns · Docs REVISE→fixed · InfoSec CONDITIONAL · Junior-dev CONDITIONAL→fixed · A11y-mini PASS-w/-minors. Watermark clean (two methods).

| Finding | Sev | Disposition |
|---|---|---|
| Code#1 qualified type-ahead saved wrong city (probe: "Portland, ME"→OR) | Critical | **Fixed** + real-index wizard regression |
| Code#2 setup re-run wiped config (probe-confirmed) | Critical | **Fixed**: Save loads + mutates only wizard-owned fields; preservation regression |
| Code#3 centroid scan per keystroke (29.5ms) + budget test blind to it | Important | **Fixed**: RepresentativeZipFast on type-ahead; single-call budget test (which then caught the 12.5ms 'a' case → top-N selection scan fix) |
| Code#4 stale measured claims | Important | **Fixed** |
| Code#5 runtime sorts deletable | Important | **Fixed**: data pre-sorted at build; Load asserts sortedness fail-closed |
| Code#6 int32 guard | Minor | **Fixed** |
| H1 FIRMS prompt asks everyone (OQ-19 said fire-zone-conditional) | Important | **Ruled D-22 (better design)**: zone INFERRED from the user's chosen location via NWS /points; Q3 zone-aware when known, generic otherwise; tested |
| H2 <10ms is a test not a benchmark | Minor | Accepted as-is (headroom 1000x); noted |
| D1 stale binary size in exit evidence | Important | **Fixed** (this log) |
| D2 PTY config-assert claim overstated | Minor | **Fixed** (script header documents the asserts) |
| D3/JD2 S2 doc pointer dangling; refresh pipeline undocumented | Important | **Fixed**: pointer corrected; tools/geotrim/refresh.sh commits the pipeline |
| S1 IS-6 checksums due | Important | **Fixed**: SHA-256 pins in checksums_test.go + refresh.sh update rule |
| S2 FIRMS key echoed while typing | Important | **Ruled D-22**: masked by default (EchoPassword), ctrl+r reveal toggle; tested |
| S3 key in scrollback (inline TUI) | Minor | Residual noted; S2 masking largely mitigates |
| JD1 PTY script could clobber real config | Important | **Fixed**: refuses to run without sandboxed XDG_CONFIG_HOME + full header |
| A1 hints 2–5 unselectable | Minor | **Fixed**: ↑/↓ selection with positional marker + footer hint |
| A2 "[esc] Exit to Main Screen" misleading | Minor | **Fixed**: "Exit without saving" |
