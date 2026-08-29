# VALIDATE report — severe-alerts-modals (0.13.0)

**Phase:** VALIDATE → SHIP gate · **SEV-0 · COLLABORATE** · **Tier:** STANDARD VALIDATE (the SEV-0 default) ·
**Date:** 2026-08-29 · **Branch:** `feature/severe-alerts-modals` · **Validated at:** `30392e6` → `bd9659e`
(two P1 fixes landed in-phase, below) · **Evidence:** `07-readiness/validate/` (the journey log, the soak samples
and log, the regression and benchmark record)

## 1. Environment (Step 1 — mandatory)

| Check | Result |
|---|---|
| Toolchain | Go 1.27.0 darwin/arm64 (the CI toolchain); `go.mod` floor 1.25 |
| Binary | `dist/watchpost` `bd9659e`; `dist/watchpost-0.13.0` (the version-stamped preview the HUM LEAD used for the screenshots and for finding P1-2) |
| Configuration | a real `config.toml` (50 locations, 9 favourites); a fresh HOME for the first-run journey — Setup opens, saves, the dashboard fills |
| Connectivity — the global feeds | `WATCHPOST_LIVE=1`: USGS 3 events · NHC 2 · NWS 0 — the 0 is the curated severe-warning query on a quiet day, not a parse failure: the raw active-alerts endpoint returned **192 features, 192 decoded, 0 undecodable** through the new entry-by-entry parser |
| Connectivity — the location providers | `watchpost report "Oceanside, CA" --verbose`: nws · nws-marine · ndbc · coops · coops-obs · hms · wfigs · firms all **ok**, the Heat Advisory, the fire incidents, the 7-day forecast rendered; exit 0 |

## 2. Core tests — P0 + P1 (Step 2 — mandatory)

`scripts/quality/validate-journey.expect`, the real binary on a pty against the live feeds, a fresh HOME, every step
asserted on the screen — **22 / 22**:

P0: the masthead on a first run · Setup opens · Setup saves and closes · first data · the watchlist names the
location · `w` opens the window · the tab row · `esc esc` closes · `l` opens Lookup · Lookup opens Details on the
looked-up location from the first frame · `q` quits cleanly.
P1: `→` moves to Watches · the Sig. Quakes tab · `enter` opens a record · **`space` reads the event over the radio
(the panel shows `EVENT ·`)** · Details fills with data · `A` opens the alert modal · `t` opens the chooser ·
Watchpost Light applies live · `S` opens Status · Status gauges the severe index · `?` opens Help.

Plus `make pty-severe` (the BUILD-era journey: `w`, tabs, `enter/esc/esc`, `ctrl+s`, `q`) — ok.

## 3. Regression (Step 3 — mandatory)

`go test -race -count=1 ./...` on the final tree — **40 packages ok, 0 failures** (`after-soak.txt`). `make verify`
ALL GATES GREEN on every commit; `make p10` 0 live / 0 unmatched; `a2dh validate` 18/18. Goldens and declaration
sets unchanged since REVIEW except the deliberate re-records noted in the objectives.

## 4. Performance (Step 4 — mandatory at SEV-0)

**Baseline comparison** (`go test -bench`, count 5, this machine, current tree vs a `v0.12.0` worktree):

| Benchmark | v0.12.0 | 0.13.0 (`bd9659e`) | Delta |
|---|---|---|---|
| Frame 133×44, memo hit (every tick) | 112–129 µs · 62 KB · 554 allocs | 155–164 µs · 80.6 KB · 655 allocs | **+38 % time · +30 % bytes · +18 % allocs** |
| Severe window frame, memo hit | — (new) | 1.27 ms · 868 KB · 2 051 allocs | new surface |
| Overlay compositor alone | — | 1.05 ms · 765 KB · 1 376 allocs | the known lever (round 3) |

The frame delta exceeds the 10 % rule and is **expected, root-caused and accepted**: 0.13.0 repainted the
dashboard — the masthead box, the boxed alert module, the boxed player with its marquee band, the painted
column-title row — each one a `Block` pass per frame; every step was measured and pinned in `modes/tty/bench_test.go`
as it landed (the 80×24 memo-hit pin moved 996 → 1 224 → 1 312 → 962 as the layout learned to build its rows once).
A frame is still 0.16 ms — a tick budget of 6 000 frames a second against a 4 Hz tick. The severe window's 1.27 ms is
dominated by the compositor (1.05 ms), the follow-up named since round 3.

**Soak (RS-8, plan 4.5)** — one hour, the real binary (`df59528` — the pre-fix build; the two P1 fixes are key
handling and hook wiring, no bearing on memory), the real config (50 locations), radio off, viz off, the severe
window opened for a minute every ten minutes, sampled every five minutes (`soak-1h.csv`):

| | first sample (08 s) | 5–55 min | trend |
|---|---|---|---|
| RSS | 116 MB (launch burst) | 86–115 MB, oscillating with the window cycles | none |
| post-GC heap in use | 44.5 MB | 44.6 → 50.0 → 47.1 MB (peak at the 45-min window open, back down after) | plateau ~47 MB |
| heap objects | 118 k | 119–125 k | flat |
| goroutines | 350 | 328–335 | flat |
| threads | 26 | 25 → 32 after 25 min, then flat | runtime warm-up, bounded |
| fds | 21 | 15–21 | flat |
| http cache | 787 files / 35.7 MB | 787–788 / 35.6 MB | flat (the coalesced disk tier) |
| RECENT publishes | 27 | +3 per 5 min | the expected cadence |

No leak signature in any column; the window's open/close cycles neither grew the heap nor the goroutine count.
Note (anti-pattern 3): the race suite and the benchmarks ran *after* the soak, not during it; a `make verify`
overlapped minutes 20–25 (the CPU column's 2.9 %).

## 5. Findings — both P1, both fixed in-phase and pinned

| ID | Sev | Finding | Fix |
|---|---|---|---|
| V-1 | P1 | On a real pty a lone `esc` is delivered fused with the next key; an **uppercase** key arrived as alt+shift+letter with no text and the split left `shift+s` — bound to nothing. `esc` then `S`/`V`/`T`/`M`/`A` were lost after any window. Found by the journey (`S` after the theme chooser). | `splitEscFusion` reads the letter (`30392e6`); `TestLoneEscThenKeyIsNotLost` + the journey |
| V-2 | P1 | `[space]` Read was muted on every build: the round-5 B-04 guard wired the hook only when the radio deck existed at config time, but the deck is attached after the dashboard is built. Found by the HUM LEAD on the stamped build. | the hook decides at the press (`bd9659e`); `TestNarrateEventDecidesAtThePress` + the journey's read step |

Both are in the class the journey now covers on every run (the real binary, the real input layer).

## 6. README content audit (VALIDATE exit gate)

Done at REVIEW (round 5 D) and redone in this phase: the README was rewritten for its audience — people who want a
fast, light weather app in a terminal — with every claim checked against the tree, the memory claim taken from the
soak, and seventeen 0.13.0 captures by the HUM LEAD (six animations built with ffmpeg: the window through its
categories, the alert modal paging, a breaking takeover, Watchpost Light, the radio on air, three window sizes).

## 7. Known issues and limitations (anti-pattern 7)

- The 80×24 floor holds with one favourite; each favourite past the first needs a row (documented; a design change
  to window the favourites table is a later release).
- Light terminals are served by the Watchpost Light theme; a dark theme on a light terminal paints its own dark
  ground (by design since REVIEW).
- Coverage is US-only (NWS); the NWS national query carries the curated severe products only — advisories and
  statements come from the watchlist.
- Follow-ups carried to REFLECT: the multi-alert circle visualisation (awaits a HUM LEAD mock); the compositor
  cost of an open window (the perf lever).

## 8. Stakeholder sign-off (Step 7 — SEV-0: human, with the evidence)

**Owed to the HUM LEAD before the transition — R6, blocking:** the radio smoke on macOS (a synth cycle to the
sign-off; a relay; a `space` read paused by a takeover and resumed; `V` audition during a read; `+`/`-` during a
read; quit during a read) and the Linux half (`07-readiness/linux-validation-protocol.md`: `make verify`,
`make pty-severe`, then the same radio steps with audio). The HUM LEAD's captures during this phase already show
a synth cycle on air, Nearest Relay live and the voice chooser on macOS; the remaining items need the HUM LEAD's
ear.

**Verdict (agent):** every automatable gate of STANDARD VALIDATE passes on the final tree — environment,
connectivity, the 22-step core journey, the full `-race` regression, the performance baseline with its one
expected and accepted delta, the one-hour soak with no trend — and the two defects the phase found are fixed and
pinned. **Ready for SHIP on the HUM LEAD's R6 confirmation and sign-off.**

## 9. Sign-off

**HUM LEAD, 2026-08-29:** "Radio smoke on MacOS works" — R6 on macOS passed. **Accepted risks, recorded:** the Linux
half (`make verify`, `make pty-severe`, the radio steps with audio) runs post-release; the takeover pause/resume
check is run later the same day when events are more numerous (the stamped build is looping Repeat: Watchlist
meanwhile). VALIDATE exit approved → SHIP.
