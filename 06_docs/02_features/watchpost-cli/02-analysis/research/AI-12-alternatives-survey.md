# AI-12: Alternatives Survey — Terminal Weather Tools & Alert Channels

Scope: honest competitive scan for the DISCOVER-phase problem statement. Named tools **atmo**, **rainy**, and **tuiweather** could not be located as maintained public repos (GitHub topic `weather-tui` is empty; searches surfaced only 0–3-star hobby projects). Verified tools below.

## 1. Survey Table

| Tool | Live/TTY vs one-shot | Multi-loc | Alerts | Radio | Fire | Multi-provider | JSON | Charts/viz | Maintained | License |
|---|---|---|---|---|---|---|---|---|---|---|
| **wego** (Go) | One-shot | One per run | No | No | No | Yes — 8 backends (one at a time) | Yes (json frontend/backend) | ASCII art table | Yes, slow cadence | ISC |
| **wttr.in** (curl service) | One-shot (users loop it in tmux) | Yes — `wttr.in/A:B:C` | No | No | No | No (WorldWeatherOnline) | Yes — `format=j1`/`j2` | ASCII/emoji/PNG; v3 sixel maps | Very active (~100M queries/day) | Apache-2.0 |
| **stormy** (Go) | One-shot + basic live poll | No | No | No | No | Open-Meteo or OWM | No | ASCII icons | Yes | MIT |
| **wthrr** "weathercrab" (Rust) | One-shot | One per run | No | No | No | Open-Meteo only | No | Yes — hourly graphs | Yes (v1.1.1) | MIT |
| **girouette** (Rust) | One-shot | One per run | Yes — alerts segment (OpenWeather, on request, not push) | No | No | OpenWeather only | No | No | Yes (v0.7.4) | MIT/Apache-2.0 |
| **termidar** (Go, bubbletea-era TUI) | **Live animated TUI** — NEXRAD radar playback | One at a time (ZIP switch) | Yes — NWS alerts with visual indicators | No | No | RainViewer + Iowa Mesonet + NWS | No | Radar frames in-terminal | New/tiny (2 stars) | MIT |
| **clawea** (Go TUI) | Interactive TUI | No | No | No | No | Open-Meteo | No | Paged UI | Yes, small | MIT |
| **ws4kp** WeatherStar 4000+ (browser, not TTY) | **Live cycling dashboard** ("Local on the 8s") | One location | Yes — SPC outlooks/severe screens | Music playlist (not NOAA radio) | No | api.weather.gov only | No | Retro radar + graphics | Very active (2k stars) | MIT |
| **Phone/WEA + NWS ecosystem** | Push, always-on | Current location only (cell broadcast) | Extreme warnings only, seconds-fast | NOAA Weather Radio is a separate device/stream | No | n/a | api.weather.gov is free JSON | n/a | n/a | n/a |

## 2. Differentiators vs. Closest Prior Art

1. **Multi-location live dashboard + national summary** — Closest: wttr.in multi-location (one-shot text) and termidar (live, single location). Nothing does live multi-location. *Genuine gap.*
2. **Real-time NWS alerts ≤60s** — Closest: termidar and girouette show alerts on fetch; neither streams. Sub-minute *polling of api.weather.gov* is trivial for anyone, though — the differentiator is the surfacing, not the data. *Gap real but thin.*
3. **Fire hotspots** — No terminal tool found; NASA FIRMS is web-only. *Genuine gap, niche audience.*
4. **NOAA radio streaming** — No terminal tool; but `mpv <stream-url>` already works, and ws4kp ships ambient audio. *Convenience feature, not a moat.*
5. **Multi-provider harmonization/diff** — wego already speaks 8 providers, but only one per run, no comparison. *Genuine gap; wego is 80% of the plumbing.*
6. **Cycling "Local on the 8s" playlist** — **ws4kp substantially covers this** (cycling screens, retro radar, severe-weather screens, music) — just in a browser, single-location, US-only. Watchpost's claim here reduces to "in the terminal + multi-location."
7. **--json agent mode** — **Already covered**: wttr.in `format=j1`, wego's json frontend, and api.weather.gov itself. Not a differentiator; table stakes.
8. **go-studs charts** — wthrr already draws forecast graphs in-terminal. Implementation choice, not a differentiator.

## 3. WEA / Phone Reality Check

Every modern US phone already receives WEA pushes — within seconds, no app — for tornado, destructive severe thunderstorm, considerable+ flash flood, hurricane/storm surge, extreme wind, dust storm, snow squall, tsunami (weather.gov/wrn/wea). For the **"take shelter now" life-safety case at your current location, the phone already wins** and watchpost must not market itself as that layer. What watchpost honestly adds: (a) **breadth** — watches, advisories, statements, non-destructive warnings WEA never pushes; (b) **remote monitoring** — WEA is cell-broadcast to your current location only; watchpost can watch family members' towns; (c) **desk presence** — ambient awareness without picking up a phone.

## 4. Opinion & Positioning

**Does "no single glanceable surface" survive?** Narrowly, yes — no *terminal* tool combines live multi-location conditions + streaming alerts + cycling display. But the unqualified existence claim is false in spirit: the pieces exist (wttr.in for glanceable conditions, termidar for live radar+alerts, ws4kp for the cycling dashboard), and the discovery should say so.

**Recommended positioning sentence:** "Closest prior art is WeatherStar 4000+ (a browser-based cycling local-weather dashboard) and termidar (live terminal radar with alerts); watchpost differs by being terminal-native, multi-location with a national summary, sub-minute alert surfacing for the full watch/advisory spectrum, and multi-provider diffing — with JSON output as table stakes, not a differentiator."

**Strongest counter-argument:** a ws4kp browser tab + phone WEA + `curl wttr.in?format=j1` already delivers the cycling glanceable dashboard, life-safety pushes, and agent-readable JSON at zero build cost; watchpost's residual value is terminal-nativeness and multi-location alert breadth, which serves a small population.

## Sources

- https://github.com/schachmat/wego
- https://github.com/chubin/wttr.in
- https://github.com/ashish0kumar/stormy
- https://github.com/ttytm/wthrr-the-weathercrab
- https://github.com/gourlaysama/girouette
- https://github.com/N-Erickson/termidar
- https://github.com/Cladamos/clawea
- https://github.com/netbymatt/ws4kp
- https://www.weather.gov/wrn/wea
- https://github.com/topics/weather-tui (empty topic)
