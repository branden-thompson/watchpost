# Changelog

All notable changes to Watchpost CLI. The format follows Keep a Changelog; versions follow SemVer.

## [0.9.1] — 2026-08-25

### Added
- **Linux/Windows voice profiles**: `V` lists a curated Piper catalogue — Lessac, Amy, Ryan, Joe
  (US), Alan, Alba (GB), medium quality, pinned SHA-256s from rhasspy/piper-voices v1.0.0. A voice
  you have not used yet downloads on first use (~63 MB) with progress in the player; the preview
  (`p`) does the same. The chooser used to show only the one voice that happened to be installed
  (Arch/CachyOS validation, UAT 118).

### Fixed
- Release job: a darwin-only test expectation, `lint-watermark` on a tag checkout, and the
  type-ahead budget under the race detector (CI-only; the v0.9.0 release itself is unaffected).

## [0.9.0] — 2026-08-25 — first public release (pre-1.0)

The first tagged build: everything from B0 (scaffold) through B5 (fire), driven UX-first through
116 UAT sessions. Details per session live in `06_docs/02_features/watchpost-cli/04-development/`.

### Added
- **Dashboard** (`watchpost`): 10-favourite watchlist + a 50-deep RECENT / SEARCHED stack, both
  persisted; per-row NOW / TODAY / TOMORROW / EXTENDED with harmonized NWS observations, hourly
  forecasts on demand, alert banner with severity ordering, sun times, `WX STN` + distance,
  location detail modal (`enter`) with marine/tides where within reach of the coast, alert details
  (`A`), lookup (`l`), add/remove (`ctrl+a` / `shift+delete`), theme chooser (`t`, live + persisted),
  help (`?`), About with data credits (`a`), API status (`S`), compact layouts for short terminals.
- **Marine**: NDBC buoys (5-day observations), CO-OPS tide predictions, water levels and currents,
  NWS coastal-waters forecast; inland locations degrade quietly.
- **Radio** (NOAA Weather Radio): Live relays (wxradio.org / weatherUSA, MP3, ICY metadata, stall
  watchdog, mount failover) and a **synthesized broadcast** of the location's own NWS products
  (HWO / SPS / NOW / ZFP, UGC-filtered to the location's zone and county) read by the macOS System
  Voice or a Piper voice on Linux/Windows (SHA-256-pinned installer). Controls: `space` play/stop,
  `+`/`-` volume, `r` Repeat: Off | One | Watchlist, `m` Mode: Synth | Nearest Relay, `v`
  visualizer (CLIAmp-style bars, 3 rows / 1 row), `V` voice chooser with preview, `T` size.
  Voice changes hand over mid-broadcast at the same spot with a hand-over line. The lead points
  listeners to the live NWR transmitter (callsign, site, frequency — read digit by digit) and a
  **Fire and Hotspot report** follows the forecast when fire data is known: hotspots inside the
  fire ring, the strongest one, named fires with size / age / containment, and nearby fires beyond
  the ring.
- **Fire** (B5): every location watches for wildfire like it watches for alerts — a counted orange
  `◆` row mark (bold when a hotspot burns at ≥ 50 MW) and a FIRE section in the detail modal: satellite hotspots
  within 25 km (bearing, distance, MW, satellite, age), NIFC WFIGS incidents within 50 km (acres,
  % contained), the Red Flag / Fire Weather alert. Sources: NOAA HMS (keyless, analyst-curated,
  10-minute refresh), WFIGS (keyless), NASA FIRMS VIIRS (optional free MAP_KEY, stored by
  the Setup window). Thresholds in `[fire]`; the report and `--json` carry the same data.
- **Setup window** (`s`, and by itself on a first run or `watchpost setup`): the default location
  with type-ahead and the optional NASA FIRMS key (paste it, or leave it empty for the default data
  set) — one form over the dashboard, `tab` between the questions; the key takes effect without a
  relaunch, the window shows a stored key's last four characters and whether FIRMS accepts it, and
  an unkeyed FIRMS reads `off` in the API status.
- **Themes**: Watchpost (default), High Contrast, Monochrome, Solarized Night, Synthwave '84 — each
  with its own title gradient; user token files in `<config>/themes/`. Empty states for the
  watchlist and the RECENT list; the masthead counts API health (`✔ ⚠ ✘`) and centres the stamp.
- **One-shot report** (`watchpost report <city|zip|lat,lon>`, `--json` with a published schema
  via `watchpost schema`), **setup** (`watchpost setup`), `--version`.
- **Platform**: tiered fetch scheduler with retry-before-cadence and incremental watchlist
  updates; token-bucket HTTP client with a priority lane, memory + disk cache honouring server
  cache headers, singleflight, bounded disk reads; pure-Go audio (oto) and MP3 decode; embedded
  NWR transmitter table (1,035 sites) and city/zip geodata; semantic theme tokens.
- **Release**: `scripts/install.sh` (`curl -fsSL … | sh`, SHA-256 verified, PATH guidance),
  `make release-matrix` (darwin/linux × amd64/arm64, windows/amd64, CGO off), `make install-test`
  (local end-to-end installer test with a tamper control), GitHub Actions release on `v*` tags.

### Known limitations (tracked for 1.0)
- Alert-interrupt tone (WAT) and the Live silence detector are not implemented.
- Windows: builds, untested end to end (audio, Piper, terminal glyphs).
- Great Lakes water levels and inland lakes (Tahoe) are not in the marine module.
- Linux end-to-end validation (audio device, Piper install, terminal fonts) is the 0.9.0 → 1.0 gate.
