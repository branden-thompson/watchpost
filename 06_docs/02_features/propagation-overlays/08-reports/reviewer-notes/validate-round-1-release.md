# VALIDATE exit red team (D-338): the release reviewer's report, verbatim

Brief: kept with the evidence outside the tree (`red-team-validate-r1/prompt-release.txt`). Dispositions: `../red-team-validate.md`.

**VR — release reviewer, VALIDATE exit (watchpost `ecfd616c`, go-ionomaps `01f5dbb` (rc.17), go-tuiMaps `57a9c29`)**

Read-only; one `go test -overlay` probe (`<scratch>/rt-v1r/vr_probe_test.go`) and the journey's raw log, escape-stripped (`<scratch>/rt-v1r/j11.txt`).

## 1. True

**VR-1 — Important. The Propagation mode does not refresh while the Best Bands chart (or any window) is open over the map; on Hourly that costs a whole hour.** `CHANGELOG.md:19-20` says "how often the Propagation mode refreshes (hourly by default)"; FR-4.10 says "while the mode is open, the fields are refreshed hourly". But `askPropagation` asks only when `d.modal == modalMap` (`modes/tty/map_prop_update.go:52`), and `applyPropTick` then skips and sets the next tick a full period away (`modes/tty/map_prop_update.go:208-221`); closing the window asks nothing. My probe: open P, press B, deliver the tick → asked=1, next tick 1h0m; after esc, still asked=1. *Likely:* PS-1's own path is g, P, B (D-331) and the chart is the answer an operator reads and leaves open. A chart left open never refreshes; one open when the hourly tick lands leaves the map unrefreshed for 2 h (data up to ~2 h 55 min old), and the chart itself never says STALE — it shows only the snapshot's "HH:MM UTC" (`modes/tty/map_prop_chart.go:92`, `:276`). D-332 made Hourly the default in VALIDATE, so a skipped tick now costs 6× what it did. *Action:* (a) let the tick ask under a window over the map, or ask on the window's close when a tick was skipped — the existing ask, not a new mechanism; or (b) rule it a known limit and say it in Known and in the chart.

**VR-2 — Important. First run: esc now saves the chosen location (D-336) but its chip still says "esc Cancel".** `modes/tty/setup_layout.go:513-517` labels esc "Cancel" on the Data rows; `modes/tty/setup.go:249-254` saves on the first run with a pick. The journey shows "esc Cancel" on the first-run Setup (`j11.txt:35`). The code's own comment says a Cancel chip over a save "would be lying" (`setup_layout.go:510-512`). *Likely:* every new listener sees it. *Action:* say "Save" (or "Close") on the first run with a location chosen — the agent's word (D-13).

Clean, against code: the Added items (`100 W, 0 dBi` `map_prop_chart.go:276`; "TYPICAL, NOT A FORECAST" `map_prop_ahead.go:153`; Settings rows `map_prop_settings.go`); the agreement line (`map_prop_chart.go:376`) and notice (`map_prop_update.go:303-313`) read as A-102/A-99 say; STALE follows the refresh (`map_prop_settings.go:52`) and readings assimilated are at most an hour old (`internal/hybrid/hybrid.go:19`); the Fixed items (`platform/httpx/memo.go:41,145`, `httpx.go:442-457`, `platform/config/config.go:520-527`, `platform/history/cache.go:100`); Go floor from `go.mod`. D-331's PS-1 wording is true: the chart line is drawn on both consoles' chart.

Minors:
- **VR-3** `README.md:92` "`tab` moves between the questions" — tab changes tabs (`setup.go:262`, D-62; the chip says "tab Next tab"); esc is not mentioned. Fix the sentence.
- **VR-4** `CHANGELOG.md:27` "GIRO is asked at most once an hour" reads as one request; it is a burst of up to 40 (rule 4, D-87). `README.md:260` already says "one burst an hour" — use those words.
- **VR-5** `modes/tty/map_prop_ack.go:23` "the Lowell Global Ionospheric Radio Observatory" is not a real name — "GIRO, the Global Ionospheric Radio Observatory" or "the Lowell GIRO Data Center".
- **VR-6** `CHANGELOG.md:75-76` "A side-by-side run against 0.18.2 measures what this changes over days" reads as still to come; item 6 has shown it (at most 0.7 MB/day against 8–9). Say it.

## 2. Complete

- **VR-7 (Minor)** Known lists three of the timing model's accepted limits; agenda item 7 (`validate-agenda.md:25`) names five. Missing: with 39 or more stations not flagged, the flagged come back one an hour; a reply with only an old reading counted as data (go-ionomaps `update-timing-model.md` "What this model accepts"). The seed has 39 live stations, so the first is the normal state. D-335 approved Known "as drafted", so not a challenge — either the agenda says the two were left out, or Known carries them.
- **VR-8 (Minor) — a 0.19.0 change the changelog does not name:** with the map open in Radar or Forecast mode, `B` used to fall through to the Observer and swap to the Broadcaster console (`router.go:162`); it is now the map's key (`map_pane.go:930`) and outside the Propagation mode does nothing (`map_prop_chart.go` `openPropChart`). `L`, `U`, `F`, `/` are newly the map's too. Name it in Changed (`ctrl+b` still swaps).

## 3. Fair

- **VR-9 (Minor) — figures without their basis.** The Broadcaster limit (`CHANGELOG.md:43-46`) was measured on `c8a99e48` (rc.13) at the every-10-minutes refresh, then the default (`git show c8a99e48:modes/tty/map_prop_settings.go`): "13 MB an hour, within the 19 allowed" is the non-default setting's cost and ceiling (Hourly measured 2.66 against 3.3); 62.5 MB is rounded down to "62"; gates ran on the machine during it (the agenda says so, the changelog does not). The accuracy figures (`CHANGELOG.md:49-54`, and on screen) were scored at `2ecdaa36` — before rc.17's Pacific box — on six quiet days (F10.7 105–109, Kp < 5), M1's lead +1.3 [+0.4, +2.3] (`m1-m4-instrument.md:240-248`); none is said. *Action:* a clause each (setting, build, conditions), or re-score M1/M4 at rc.17 offline with `reachscore`.
- **VR-10 (Minor) — agenda.** Item 5's "every G-M3 target met" cites only the fresh days — the days the box was drawn from; add PLAN's independent week (1.28 against 1.32, D-334). Item 4 records D-332's live check as done; the fix it led to (A-103's 2 h bound) has not been seen live, and VR-1 shows STALE will appear after a skipped tick. The agenda's front matter still says "OPEN".

Not overclaimed: PS-1 ("in part", no floor, M2 unmeasured), the Pacific ("starts from a typical day", no figure claimed), accuracy ("only a little better").

## 4. Terms

**VR-11 — Important. No station's data provider is credited, against D-41 and FR-4.1.** D-41 rules "GIRO and each station's provider are credited in About"; FR-4.1 repeats it (`requirements.md:123`); go-ionomaps `NOTICE:13` says "Acknowledge each station's data provider." About carries one GIRO line (`app/credits.go:67-69`); `TestEveryPropagationSourceIsCredited` checks hosts and badges only, so it cannot fail on this. *Likely:* GIRO is on by default, so every Propagation user's map draws on station data whose providers go unacknowledged — a ruling not followed and a provider's terms not met. *Action:* credit the providers in About (by name, or a pointer to GIRO's station-provider list), or the HUM LEAD amends D-41/FR-4.1.

Otherwise terms are said where met: GIRO's non-commercial terms in About, Settings (`map_prop_settings.go:144`), README (`:260`, `:391`), changelog, go-ionomaps README and NOTICE; README credits match About line for line; PyIRI's MIT text carried. The acknowledgement window — the first thing every Propagation user sees — names GIRO without its terms (optional).

**VR-12 (Minor, from 0.18)** go-tuiMaps asks for the OSM/OpenMapTiles credit "with the links" in About (`assets/NOTICE`); About gives no URL (`credits.go:62`).

## 5. The first run

Yes — a new listener today gets through. The three post-fix runs on the batch-80 tree passed, `fails=0` (`j80-9`, `j80-10`, `j80-11`): Setup (type, enter, esc), first data, Lookup → Details, P with its acknowledgement, MUF drawn, the chart with its agreement line, the console's map in the Propagation mode. (`j80-1`..`j80-8` failed on the chart-settling defect A-104(4) fixed.) esc without a pick is not a dead end: Setup reopens (`config.go:523-525`) and enter falls back to the displayed default (`setup.go:551-553`). Caveats: VR-2's chip, VR-3's README sentence; one machine, 133×44.

At SHIP: the changelog's "go-ionomaps v0.1.0 / go-tuiMaps v0.3.0" needs the re-pin from rc.17/rc.10 (`go.mod:9-10`).

## Verdict

**Do not exit VALIDATE.** Three Importants: VR-1 (no refresh while the chart is open, and the chart never says STALE), VR-2 (the first-run "esc Cancel" chip), VR-11 (station providers not credited, against D-41). VR-2 is one word; VR-1 and VR-11 each need a small change or a ruling. The Minors can batch.
