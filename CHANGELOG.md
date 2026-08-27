# Changelog

All notable changes to Watchpost CLI. The format follows Keep a Changelog; versions follow SemVer.

## [Unreleased]

## [0.9.8] — 2026-08-26

Performance & quality pass, batch Q5 (network, bytes and fan-out).

### Changed
- Conditional GETs: an expired cache entry that carries `Last-Modified` / `ETag` is offered to the
  server (`If-Modified-Since` first); a `304` renews it without downloading the body. The `S` modal
  and `report --verbose` count the renewals and the bytes they saved.
- NASA FIRMS requests are made per 5° map tile instead of per location: every tracked location
  inside a tile shares one request per satellite source, a location's search box that straddles a
  tile edge fetches every tile it touches (at most four), and the hotspots each location sees are
  unchanged. A tile is parsed once per body change.
- One connection policy for every HTTP client — the data clients, the live-radio stream reader and
  the voice-model downloader — with an idle timeout that outlives a 10-minute refresh cycle, so a
  session is reused instead of a new TLS handshake per host per cycle.
- Tide and current predictions are cached until UTC midnight (they are astronomical and the request
  window is keyed by the UTC date) instead of for an hour; a location's NWS grid resolution is
  redone after a day and dropped when the location leaves the lists; the gridpoint that fills
  TODAY's HIGH/LOW after 6 PM is decoded once per body change instead of once per location.
- `watchpost report` fetches its seven data kinds in parallel: a cold report takes seconds, not tens.

## [0.9.7] — 2026-08-26

Performance & quality pass, batch Q4a (go-studs correctness patches, each a patch beside the kit).

### Changed
- The location tables' colours belong to the theme: column headers, row numbers and attribute
  cells take three new tokens (`table.header`, `table.muted`, `table.name`; defaults are the
  colours the kit painted before, so the default theme looks the same), the `t` chooser restyles
  them (Monochrome no longer shows a purple header; High Contrast, Synthwave '84 and Solarized
  Night carry their own, all ≥ 4.5:1), and `NO_COLOR=1` now silences the whole frame. **One look
  change:** with `TERM` unset or `dumb` the kit used to leave the header and those cells plain
  while the rest of the dashboard was tinted; they now follow the theme like everything else.
  User theme files may set the three tokens.
- The table no longer opens `/dev/tty` on every frame (the kit probed the terminal size on each
  construction and never read it); the size is probed on first use, through `golang.org/x/term`,
  with no platform-specific code.
- Qualified colour composites (`1;38;5;220`, truecolor backgrounds) render as written through the
  kit's `SGR`, which builds each escape in one buffer; user theme values in that form now work.
- `Plain` (the screen-reader and `--json` text) drops U+FE0E/U+FE0F variation selectors, whose
  width terminals disagree on.
- go-studs is carried as the pinned upstream commit plus patches under
  `third_party/go-studs/patches/` (`LOCAL_CHANGES.md` lists them); `scripts/sync-go-studs.sh`
  re-applies them and refuses a drifted checkout.

## [0.9.6] — 2026-08-26

Performance & quality pass, batches Q2–Q3 (structure; the render path on the app side).

### Added
- `--ascii` (also `watchpost setup --ascii`): the row marks and the Help legend in ASCII forms
  (`>` playing, `R` on repeat, `n*` fires, `n!` alerts) — the B6 promise, wired.
- `counters.json` / `/debug/counters` carry `total_alloc` and `mallocs`, so a soak can read the
  allocation rate per phase.

### Changed
- The animation tick runs only while something animates — a loading row, a volume blink, the
  radio marquee (when the visualizer is off), the `S` modal or Location Details. An idle dashboard
  renders nothing between events.
- The two location tables are rendered once per input change and reused on every tick, marquee
  and visualizer frame between, and the frame is finished in one pass: a steady frame at 133×44
  allocates 546 times and 62 KB (was 10,044 and 436 KB) and takes ~100 µs (was ~660). The frame's geometry (compact mode, module heights, the RECENT
  window) is resolved once per frame instead of about eight times.
- The RECENT pipeline publishes at most once per five seconds (was once per 50 ms window): a
  tier tick across the fifty seeded locations is one snapshot, not ~47. Scheduler tiers fire on a
  fixed grid from start, so their phases no longer drift apart with their own fetch times.
- The HMS fire archive is parsed by a streaming, hand-decoded walk: ~88 ms / 33 MB / 605k
  allocations per 27.5k placemarks (was 104 ms / 75 MB / 1.05 M), with the inflated file never
  held in memory; satellite and method names are shared strings.
- The WFIGS incident layer is decoded once per body change (was once per location fetch).
- The embedded location index is loaded once at launch (was twice); a cached text body is
  handed to its parser without a copy (read-only by contract, pinned in every consumer).
- Source layout: the five largest files split by responsibility (`modes/tty/dashboard.go` 3,386
  lines → 14 files; `platform/render`, `app`, `domains/weather/nws`, `platform/snapshot`
  likewise); `docs/where-things-happen.md` is the map, checked by a test.

## [0.9.5] — 2026-08-26

Performance & quality pass, batches Q0–Q1 (instrumentation; the weatherUSA relay defect; a bounded
disk cache; one retry layer with a per-host failure memo).

### Added
- Diagnostics for long-running sessions: the `S` modal lists request counters per host since
  launch (attempts, network bodies, cache and negative-cache hits, bytes) and the publish counters
  per pipeline; `kill -USR1 <pid>` (macOS/Linux) or `GET /debug/dump` on the opt-in
  `WATCHPOST_DEBUG_PPROF=1` server writes a profile set with `counters.json` under the cache
  directory's `profiles/` (a minute apart at most, newest twelve kept); `/debug/counters` serves
  the same counters live; `watchpost report <loc> --verbose` appends the counters per host.
- Build gates: `go mod tidy -diff`, `go mod verify` and `govulncheck` in `make verify`; frame
  allocation pins in `make alloc-budget`; wall-clock benchmarks in `make quality-bench`.
- Redirect policy: every HTTP client (data and relay audio) follows at most three redirects and only
  to the same scheme and host; anything else is refused with the reason.

### Fixed
- Nearest Relay: the weatherUSA relay directory is reachable again. Its server only speaks a TLS key
  exchange Go removed in 1.22, so the directory and its ~120 mounts were silently missing from every
  Go build since; the directory and mounts are now fetched over plain HTTP — which is what the relay
  serves its audio on anyway (relay policy; the mounts were always `http://…/NWR/*.mp3`). A relay
  directory that stops answering is now said once in the `S` modal (`radio_unavailable`) instead
  of contributing nothing in silence, and is asked again at most every 5 minutes.
- Disk cache: it no longer grows without bound. Short-lived products (observations, alerts) are
  never written to disk; expired files are swept at launch and daily (24 h grace for entries a
  conditional GET could renew); orphaned files from the pre-0.9 format are removed; the directory
  is capped at 256 MB, oldest first. The sweep only ever touches files it wrote, never follows
  symlinks, and refuses to run outside a `watchpost` cache directory.

### Security
- Release binaries are built with Go 1.27.0. Releases 0.9.0–0.9.4 were built by CI with Go 1.25.0,
  whose standard library (`net/http`, `crypto/tls`, `net/url`, `encoding/xml`) has advisories fixed
  since (GO-2026-6088/6089/6090/6218); `govulncheck` now runs on every build and fails it on a known
  reachable vulnerability. Upgrade.

### Changed
- Retries: the dashboard's HTTP clients make one retry per request (the scheduler already re-tries
  at 10/20/40 s), and a host that stops answering is avoided for 20 s on the background lane — the
  favourites' lane and alerts always try, so a single failing station can never delay them. A
  server's `Retry-After` is honoured (clamped to 5 minutes). During a provider outage the app now
  makes a few hundred attempts an hour instead of ~23,000.

## [0.9.4] — 2026-08-25

### Changed
- Location Details: the FIRE rows use the same `◆` mark as the watchlist (was `▲`). README gains
  screenshots rendered from the dashboard with live data.

## [0.9.3] — 2026-08-25

### Fixed
- Narration reads road and highway abbreviations as words — "SWY S-2" is "State Highway S-2",
  "Palomar Mountain Rd." is "Road", "I-15" is "Interstate 15", "US-101" is "U S 101" (voice only;
  the marquee keeps the product text).

## [0.9.2] — 2026-08-25

### Fixed
- Voice chooser (Linux/Windows): the wait between `p` and the first sound is explained in the
  window — `preparing Amy…`, then `installing Amy voice… 40% (25 MB)` while a voice downloads,
  `loading Amy…` while Piper reads the model (a few seconds on every run), cleared when the sound
  starts. Preview timeout 60 s.

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
