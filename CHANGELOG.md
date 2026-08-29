# Changelog

All notable changes to Watchpost CLI. The format follows Keep a Changelog; versions follow SemVer.

## [Unreleased]

## [0.13.0] — 2026-08-29

A window for the ticker's other half: every active severe weather and disaster event, browsable in
one place, with the full record of each — the complement to 0.12.0's tape, which shows them one at a
time.

### Added
- The **Severe Weather / Disaster Events window** (`w`, or `ctrl+s`): six categories in importance
  order — Warnings · Watches · Advisories · Spec. Statements · Sig. Quakes · Tropical — browsed with
  `←` `→`, the category's events with `↑` `↓` (Declared newest first), `enter` for the full record in
  the Alert Details shape, `esc` back to the table, `esc` `esc` to close. The window is the **union**
  of the global feeds (USGS, NHC, NWS) and the tracked locations' alerts, de-duplicated by the
  event's id, so Advisories and Special Weather Statements come from your watchlist (the national
  query does not carry them) and every warning within reach is listed once. Each category paints the
  window in its fixed tint — red for quakes, orange for warnings, yellow for advisories, a deeper
  amber for watches and an olive for statements, blue for tropical — theme-independent (Monochrome
  keeps its greys); the group header is a distinct tint of the same hue. A **DETECTION** column says
  how an event was established (the warning's SOURCE line — Radar Indicated, Spotter Reported — else
  the alert's certainty; a quake's review status), and DECLARED is the issue time (the onset, when
  later, reads as "Starts" in the record); the table's column ladder gives EXPIRES, then DETECTION,
  then DECLARED as the terminal narrows, and the window's width tops out at 130 columns. A dead source is stated on the category line ("NHC unavailable"), the title
  carries the newest successful fetch, and opening the window within ten minutes of a breaking event
  lands on its category (Warnings otherwise).
- Storm **names** on the ticker tape and in the narration ("Hurricane Dolly has been reported for…").
- `[space]` inside the window **reads the focused event over the radio** — the alert itself (what, where,
  how long, the description and instructions), the panel showing `EVENT · …` while it plays and returning to
  the broadcast afterwards; the ▶ marks the row being read.
- A **narration arbiter** owns the correspondent's voice: a breaking takeover is always first — an event
  read on air is **paused** for it and resumes where it stopped afterwards; reads duck the broadcast and
  queue behind a takeover; the broadcast is ducked once and restored once around any run of them. The
  visualizer follows every narration (an event read, a voice preview) except a takeover, which plays aside.
- **Pronunciation rules as files**: how the correspondent reads time zones ("PDT" → "Pacific Daylight
  Time"), state codes, product abbreviations and the words spelled for the voice live in
  `domains/radio/pronounce/rules/<table>.txt`, one rule per line, read by name.
- **Report scripts as files.** Everything the correspondent says — the location broadcast's lead,
  conditions, alerts and sign-off, the fire and seismic reports inside it, the event read, the breaking
  takeover and the voice preview — lives in `domains/radio/script/scripts/<report>/<part>.txt`, found by
  name, with `global/head.txt` and `tail.txt` inherited by any report without its own; the same tree under
  `<config dir>/scripts/` re-words a phrase. Each report is a self-contained folder. The event read opens
  with the Watchpost notification notice ("not intended for life safety use") and closes with its sign-off.
- The `[S]` status window gauges the severe index (rows against its 500-row cap).
- `WATCHPOST_DEBUG_RADIO` also logs every synthesized segment as it reaches the air and why a cycle ended.

### Changed — the dashboard facelift (HUM LEAD UAT)
- The **header** is a titled box: the masthead and the Updated stamp ride its top rule, the controls and
  the API count share the row inside, and `[S] Status` joins the controls.
- The **alert module** is one row in a box on the Alert Details tint (red for warnings, yellow for
  advisories): count, the event and place in the modal's own dress, Issued and Expires; the body lives
  behind `[A]`. The full layout gains the rows (a 50-row terminal now holds a 21-row window).
- The **radio player** is a box: the station in the head, the spoken line as a marquee band with the
  visualizer inside it, the controls centred; the VOL bar gives before the title on narrow terminals.
- **Help** and **API Status** lay out in two columns when the terminal is wide enough.
- The watchlist's **column titles** are a painted row: each title centred in bold white on a darker tint
  of its group's band, the segments touching across the gutters like the bands above; WX STN and ZIP
  cells centred.
- Every painted text/ground pair is lifted to WCAG AA when a theme loads — the hue kept, the text moved
  toward white or black only as far as 4.5:1 needs; where no text tone can read on every ground it shares
  (bold white on the gold ticker lane) the ground deepens instead. The same pass gates every theme.
- **Watchpost Light**, the thirteenth theme (second in the picker): a light ground, dark text, pale bands,
  tints, lanes and alert tiles, for light terminals. Every theme now paints its own ground whatever the
  terminal reports.
- Under `--ascii` the severe window's header reads plain `EVENT` (a screen reader spells a spread word letter
  by letter), the play mark is `*` (never the pointer's `>`), every remaining mark has its ASCII form
  (`+ x ~ _ . - |`), arrow keys are named in words in every chip, and a muted chip keeps its tone.
- Feeds decode entry by entry: one malformed value skips its own entry, never the whole source.
- An alert's own update replaces it in the tables and `[A]` too, not only in the window.

### Fixed
- Looking up a new location opened Details on the previous top RECENT row until its data arrived; it
  opens on the looked-up location from the first frame, blank until then.
- An alert spanning more than 50 zones never reached a location in the 51st zone or beyond (a 0.12.0
  regression).
- A lone `esc` followed by a key was lost (the terminal fuses the pair as alt+key); every such chord —
  a letter, `enter`, an arrow, a control byte — now reads as `esc` then the key.
- With colour off or under `--ascii` a warning and an advisory read the same; the alert module says its
  class in text.
- A read paused for a breaking takeover was closed by the takeover's own tone on the real audio engine;
  held lines survive and resume where they stopped, with the volume knob as it is.
- The severe index read the very snapshot the dashboard was sorting (a data race); it keeps its own copy.
- A feed name carrying a newline added rows to the frame; the tape is one line, windowed by cells.
- Provider station names, config labels and a hostile alert's text reach the screen as text on every
  surface (Details, the modal titles, the tape, the radio head); a modal title never overflows its window.

### Changed
- The breaking-event narration now ends "Press W in Watchpost for the full report on this event"
  (a burst closes with "For the full report on any of these events, press W in Watchpost.").
- Every alert field — the record, the alert module's headline line and the compact line included —
  passes the terminal-escape boundary before it is drawn; the ticker's seen-store is written `0600`,
  re-tightened on every save, and capped at 20 000 ids.

## [0.12.0] — 2026-08-27

A global event ticker — the world's largest active hazards scroll across the top of the dashboard,
independent of your watchlist, and a breaking one is read aloud over the radio. Three feeds join the
weather picture at the scale of the planet: significant earthquakes (USGS), tropical cyclones (NHC)
and US severe-weather warnings (NWS).

### Added
- A **global event ticker band** at the top of the dashboard: a ticker-tape of active hazards from
  three keyless feeds — significant earthquakes (USGS), tropical cyclones (NHC) and US
  severe/tornado warnings (NWS) — mapped to one event model, deduped and stacked most-recent-first.
  It is a **separate pipeline** from the per-location snapshots: these events belong to no tracked
  location.
- **Category lanes.** The tape rotates through four lanes — Severe Earthquakes, Tropical Cyclones,
  Warnings, Watches — every 90 seconds, each lane's events `•`-separated and scrolling, with their
  issued and expires times; an event drops from the tape the moment it expires. A left indicator
  shows the lane's `[count][glyph]`, in a fixed per-category colour: earthquakes red, warnings
  orange, watches yellow, tropical cyclones blue.
- A **breaking-news takeover.** When a new severe event arrives the ticker switches to its lane and
  shows that one event, centred, holding through its narration (at least 5 seconds); simultaneous
  events queue by severity and are read in turn. The event is **read aloud over the radio** — the
  synth or the live relay ducks for the announcement and returns — with US state codes expanded
  ("ND" → "North Dakota") and the same pronunciation rules as the broadcast.
- A **redesigned Setup window**: settings grouped by concern (Data Access; Severe Weather / Disaster
  Events) with an **Alert Notification Preference** — every severe event, or filtered to within N
  miles of your location. The radius scopes the whole ticker. Grouped questions read white, their
  support text grey, their working status coloured; selection is shown by glyph, not colour alone.
- **Seven Omarchy Quattro themes** — Tokyo Night, Gruvbox, Nord, Catppuccin, Everforest, Kanagawa
  and Osaka Jade — bringing the built-in palette count to twelve, each meeting the AA contrast
  floor, applied live and persisted like the rest.
- USGS, the NHC and the NWS national alerts feed credited in the About window (all keyless, public
  domain).

### Performance
- The HMS fire provider **coalesces and single-flights** its fetches and holds a bounded staleness
  window, and the HTTP cache gained a **large-entry tier** (separate from the small-entry cache) so
  the multi-megabyte HMS archive is no longer re-read from disk on every access. Steady-state
  allocation churn dropped roughly by half and disk reads by three-quarters; the ticker itself adds
  well under a megabyte at rest.

### Security
- The ticker renders **feed text as plain text** — control and escape sequences are stripped before
  anything from an external feed reaches the terminal, so a hostile or compromised feed cannot drive
  the display. A GeoJSON `coordinates` blob is scanned iteratively (no unbounded recursion), and
  feed-supplied type and place fields are length-bounded, so a malformed feed cannot overflow the
  stack or the narration render. Untrusted text never rides in a process argument — the spoken
  narration reaches the voice over stdin or a `0600` temp file.

## [0.11.0] — 2026-08-27

Seismic data — the fourth hazard beside weather, marine and fire. USGS earthquakes near a tracked
location, filtered by a magnitude-graduated distance rule: a small quake shows only if it is close, a
large one from far away, because that is who feels it.

### Added
- A **SEISMIC section in Location Details**: recent nearby earthquakes from the USGS real-time feed,
  largest-magnitude first, each with its magnitude, distance and bearing, depth, age and a
  felt-likelihood label. A circle-family glyph ramp — `○` below feeling, `●` felt, `◉` significant — in
  a violet mark distinct from fire's orange `◆`; `--ascii` reads it `.` `o` `O`. A tsunami or a
  high-level PAGER alert reads in the warning tone. A cold or unavailable feed reads "seismic data
  unavailable"; a feed that answered with nothing reads "no recent seismic activity" — never a false
  "none". The whole list shows (the feed is capped at 20 per location).
- A **row mark on the main table**: the strongest recent quake's felt-band glyph (`○`/`●`/`◉`), one
  glyph, between the play and fire marks.
- A **radio Seismic Activity report** in the synthesized broadcast: the USGS notice, the count, the
  strongest three quakes read with magnitude, distance, depth and age and a spoken felt-likelihood,
  then a pointer to the details view for the rest.
- A **`[seismic]` config section**: `enabled`, `lookback_days` (default 7), `types` (default
  `earthquake`, extendable to blasts/explosions), and the magnitude→radius `radius_bands_mi` rule.
- USGS Earthquake Hazards Program credited in the About window (keyless, public domain).

### Performance
- Earthquakes are regional, so requests are **shared**: a per-location near-field query plus a regional
  query snapped to a fixed grid, so nearby locations collapse onto one request, with a bounded, gauged
  parse memo — the FIRMS-tile precedent. The concentric split (near vs regional) fetches 4–31 KB per box
  where a single wide query would pull ~1 MB of low-magnitude events the rule then discards. The section
  renders only inside the Details modal, so the frame allocation budget is unchanged.

## [0.10.2] — 2026-08-27

UX pass on the dashboard (HUM LEAD UAT) and two follow-up fixes.

### Changed
- The group bands — `L O C A T I O N · T O D A Y · T O M O R R O W · E X T E N D E D` and
  `R E C E N T / S E A R C H E D` — are three rows tall, a band-coloured row above and below the
  label, so they read heavier. They are the last thing to give on a short terminal: the modules
  minimize first, and the bands drop back to one row only when the RECENT window could not
  otherwise keep its three-row floor.
- The header's Updated stamp carries its age — `Updated: 08/27/2026 06:49:44 PDT (2 Minutes Ago)` —
  and reads green while data is fresh, yellow once no fetch has succeeded for more than five
  minutes, grey before the first data. Narrow terminals drop the age before the date, as before.
- The alert module is one header row — the alert count at the left inset, `⚠ EVENT · Location`
  centred, `[A] Details [←] Previous [→] Next` at the right inset — a blank, then the fixed
  three-line body with an 8-column inset (the budget keeps the module from jittering as the terminal
  narrows). Two rows shorter than before; the RECENT window gains them.
- Every floating window's title reads bold white against its tile (a `modal.title` theme token;
  each built-in theme sets its own, all at AA contrast). Location Details keeps its fill rule in the
  panel's tone with the name and the Updated stamp bold white.
- The Help window groups the bindings by feature — NAVIGATE, WATCHLIST, RADIO, DISPLAY, APP — with a
  bold-white header per group; a rebound key stays in its group, and the row-marks legend and the
  chips follow.

### Changed
- The watchlist control row reads, left to right, `[l] Lookup Location  [enter] Details
  [ctrl+a] Favorite  [shift+del] Unfavorite`. Favorite and Unfavorite are a promote/demote pair:
  Favorite lights only on a recent/searched row (it adds that location to the watchlist) and dims
  on a watchlist row; Unfavorite lights only on a watchlist row and dims on a recent one. Lookup,
  on the far left, searches a location into the RECENT list. (Adding a brand-new location is now
  Lookup it, then Favorite it — the old ctrl+a search-straight-to-watchlist is gone.)

### Fixed
- Nearest Relay plays the location you are on in one press. Before, if the radio was already
  playing another location, the first `space` stopped it and only the second tuned the newly
  selected one — which read as "the relay doesn't work here". `space` now plays the focused
  location and stops only when that location is already the one playing.
- `WATCHPOST_DEBUG_RADIO=<file>` appends one line per radio engine state change (time, state,
  mount, error, title) for diagnosing playback.
- The RECENT / SEARCHED table appears at once again: the seeded rows are on screen as soon as the
  dashboard is up (measured 159 ms from launch) and fill in cell by cell under the loading shimmer as
  their data lands, instead of the empty state holding for up to five seconds and the rows landing
  all at once (a side effect of 0.9.6's publish window, which now applies only after the launch burst).

## [0.10.1] — 2026-08-27

### Fixed
- `/debug/counters` now runs a garbage collection before reading the memory rows, as the
  diagnostic dump always did: the 5-minute soak samples are post-GC live-heap readings, the series
  the pass's growth statistic is defined on. Before this the endpoint reported whatever the heap
  held between cycles (found at the start of the 7-day validation soak; the hourly dumps were
  unaffected).

## [0.10.0] — 2026-08-27

Performance & quality pass, batch Q6 (seams).

### Changed
- One floating window at a time, by construction: the dashboard's ten "is this window open"
  flags became a single `modal` value, and a test asserts on the rendered frame — for every
  window and every way of opening one — that at most one is drawn and that `esc` never moves the
  focus. Before this, Help could open over Alerts, About or Status, Remove over Help or Details,
  and a failed voice change reopened the Voice chooser over whatever was open.
- Single owners for shared details: the acreage format, the compass arithmetic behind the three
  word tables (dashboard abbreviations, spoken wind, spoken bearings), the condition vocabulary,
  the two list caps (10 favourites, 50 recent), the control-row footers and wrapped rows. No
  output changes.

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
