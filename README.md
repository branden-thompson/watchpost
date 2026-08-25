# Watchpost

A terminal-native live weather station: a watchlist dashboard of NWS observations, forecasts and
alerts, marine conditions and tides where you are near the coast, and NOAA Weather Radio — the
real relays where they exist, or your location's own forecast read aloud where they don't.

## Install

macOS and Linux (amd64 / arm64):

```sh
curl -fsSL https://raw.githubusercontent.com/branden-thompson/watchpost/main/scripts/install.sh | sh
```

The installer picks the release for your OS and architecture, verifies its SHA-256 against the
release's `checksums.txt`, installs to `~/.local/bin` (or `/usr/local/bin` when that is writable and
`~/.local/bin` is not on your PATH), and tells you the next step. Knobs: `WATCHPOST_VERSION=v0.9.0`,
`WATCHPOST_INSTALL_DIR=…`. Windows: download `watchpost-windows-amd64.exe` from the release page.

Manual install always works too: download the binary and `checksums.txt` from
[Releases](https://github.com/branden-thompson/watchpost/releases), verify, `chmod +x`, put it on
your PATH.

Requirements: a UTF-8 locale and a font with the block/arrow glyphs (any modern terminal — not the
Linux VT console). Linux binaries link against glibc (not Alpine/musl); the radio voice on Linux
(Piper, installed the first time you tune in) needs glibc ≥ 2.29 and `libstdc++6`, and audio goes to
PulseAudio/PipeWire or ALSA (`libasound2`).

## First run

```sh
watchpost           # the dashboard — on a first run the Setup window opens over it
```

Setup is a window like every other: both questions are on screen — your default location
(type-ahead over the embedded index; `enter` keeps the current one on a re-run) and the optional
NASA FIRMS key (paste it, masked, `ctrl+r` reveals; leave it empty for the default data set — an
empty key also keeps a stored one). `tab` moves between the questions; `enter` on the key line
saves. `s` opens it again any time; `watchpost setup` starts the dashboard with it open. On
Linux/Windows the radio voice installs itself the first time you tune in.

Config lives in `$XDG_CONFIG_HOME/watchpost/config.toml` (`~/.config/…`), written 0600. The HTTP
cache and the voice live under `$XDG_CACHE_HOME/watchpost/` (`~/.cache/…`, `~/Library/Caches/…` on
macOS) and are safe to delete.

## Using it

| Key | Does |
|-----|------|
| `↑` `↓` | move between locations (favourites, then RECENT / SEARCHED) |
| `enter` | location details (hourly, extended, marine + tides, **fire** — hotspots and incidents nearby, station and distance) |
| `A` / `←` `→` | alert details / previous, next alert |
| `l` | look up a city or zip — lands at the top of RECENT and stays there across runs |
| `ctrl+a` / `shift+delete` | add the focused location to the watchlist (max 10) / remove it |
| `space` `+` `-` | radio: tune the focused location / volume |
| `r` | Repeat: Off · One · Watchlist (plays each favourite in turn) |
| `m` | Mode: Synth (your location's forecast, read aloud) · Nearest Relay (live NOAA Weather Radio) |
| `v` `V` `T` | visualizer · voice chooser (with preview) · player size |
| `f` `c` | ºF / ºC |
| `t` | colour theme (persisted; drop your own token JSON in `<config>/themes/`) |
| `s` | Setup: default location + optional NASA FIRMS key (opens by itself on a first run) |
| `a` `S` `?` `q` | About + data credits · API status · help · quit |

One-shot: `watchpost report "El Cajon, CA"` (plain text) or `--json` (schema `1.0.0-rc`, additive
changes only — `watchpost schema` prints it; the checked-in copy is `pkg/schema/`). Exit codes:
`0` ok · `1` no usable data or bad arguments · `2` a provider is degraded (an unkeyed `off` FIRMS
does not count); warnings never change the code. `providers[].status` is `ok` | `degraded` | `off`.
Shell completion: `watchpost completion bash|zsh|fish|powershell`.

Diagnostics: `WATCHPOST_DEBUG_TIMING=1` prints launch→full-view time on exit;
`WATCHPOST_DEBUG_PPROF=1` serves pprof on `127.0.0.1:6060`.

## Fire

Every location watches for wildfire the way it watches for alerts: a row wears an orange `◆` with a
count (named incidents nearby, or 1 for unnamed hotspots; bold when a hotspot is burning hard) and
the detail modal (`enter`) has a FIRE section — satellite
hotspots inside a 25 km ring (bearing, distance, strength in MW, satellite, age), named incidents
within 50 km (acres, % contained) and the Red Flag / Fire Weather alert when one is active. Two
sources need no key: NOAA's Hazard Mapping System (analyst-curated satellite detections, refreshed
every 10 minutes) and NIFC's WFIGS incident list. The plain report and `--json` carry the same data.

**NASA FIRMS (optional, free key).** FIRMS adds VIIRS detections minutes old, straight from the
satellites, before an analyst has looked at them. Get a MAP_KEY at
<https://firms.modaps.eosdis.nasa.gov/api/map_key/> (an email address; the key arrives at once),
then press `s` in the dashboard (or run `watchpost setup`), `tab` to the key line and paste it —
masked, `ctrl+r` reveals; the window shows a stored key's last four characters and whether FIRMS
accepts it — or add it to `config.toml`:

```toml
[providers.firms]
key = "your-32-character-map-key"
```

The key takes effect at once — no relaunch. It rides only in request paths to
firms.modaps.eosdis.nasa.gov and never appears in output or logs (a key that is not 32 hex
characters is refused before anything is written). Without a key FIRMS reads `off` in the API
status (`S`) and contributes nothing — HMS and WFIGS carry the default.

Thresholds are configurable (`[fire]` in `config.toml`; these are the defaults):

```toml
[fire]
radius_km          = 25   # hotspot ring around each location
incident_radius_km = 50   # named-incident ring
min_frp_mw         = 5    # hotspots below this fire radiative power are noise
bold_frp_mw        = 50   # at or above this the mark and the strength read bold
min_confidence     = "nominal"   # low | nominal | high — FIRMS points; HMS points are analyst-curated and always pass
```

## Data and credits

Watchpost reads public sources and shows their credits in the About window (`a`):
National Weather Service / NOAA (forecasts, observations, alerts, products, coastal waters,
transmitter list — public domain); NDBC buoys and CO-OPS tides/currents (NOAA); NOAA-NESDIS
Hazard Mapping System fire detections and NIFC WFIGS incidents (public domain); active fire data
from NASA FIRMS (<https://earthdata.nasa.gov/firms>, NASA open data — attribute LANCE/FIRMS);
GeoNames and Open-Meteo geocoding (CC BY 4.0, <https://creativecommons.org/licenses/by/4.0/>);
NWR audio relayed by wxradio.org and weatherUSA (community relays — relayed audio lags and is not
for life-safety use). Watchpost is not affiliated with NOAA, NIFC or NASA.

**Not a substitute for official warnings.** Everything here is fetched on a schedule and can lag;
for life safety use NOAA Weather Radio and Wireless Emergency Alerts. Coverage is US-only in 0.9.0
(NWS); non-US locations resolve but carry no weather data yet.

## Building from source

Go 1.25. `make build` (binary in `./dist`, version stamped from `git describe`), `make verify`
(fmt, vet, race, import-direction and watermark gates with positive controls), `make release-matrix`
(all targets, CGO off), `make install-test` (installer end to end against a local server). The
terminal UI kit (`go-studs`, MIT, same author) is carried in-tree under `third_party/go-studs` (its
LICENSE and NOTICE.md ride with it; import paths rewritten), so the tree builds anywhere with no
private access.

## Licence

MIT — see `LICENSE`. Use it freely; keep the copyright and permission notice (attribution).
