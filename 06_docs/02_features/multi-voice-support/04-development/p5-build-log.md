# P5 — UAT, the performance pass, and BUILD exit

**Gate:** `07-readiness/gates.md` §1. **Base:** `e31f8a5` (P4 exit). **Batch:** 51 commits.
**Owner:** agent, under HUM LEAD direction throughout — this batch is hands-on UAT, so nearly every
change here answers something observed on a running binary rather than a planned task.

P0–P4 were planned batches. P5 is not: it is the batch that exists because a SEV-0 release is
exercised by a human before it ships, and what they find becomes work. It closes with the BUILD-exit
red team (`08-reports/red-team-build.md`).

## 1. UAT dispositions

Every finding the HUM LEAD raised against the running binary, and where it landed. **The rulings are
the record**: several of these changed ratified scope, and until now they existed only as commit
subjects — which is a finding in its own right (see §4).

| # | What the HUM LEAD found or asked for | Disposition | Commit |
|---|---|---|---|
| 1 | The correspondent picker should read as chips; re-opening Settings shows the launch-time cast, not the file; `ctrl+r` | Fixed | `1be7b5c` |
| 2 | One picker column; the press blink must end | Fixed | `fef896c` |
| 3 | The hand-over line, the picker styling, apply-on-close | Fixed | `6b8e4a7` |
| 4 | `enter` on a text field must not save the window | Fixed | `b8f55cd` |
| 5 | The tone and correspondent groups should say their state outright | Fixed | `efd9fc0` |
| 6 | ALERTS — TONE needs its own column breakpoint, then one class per line | Fixed | `094cb58`, `58b6bc7` |
| 7 | `[S]` should drop CAST/TONES/CONFIG — they are canonically in Settings; `[s] Setup` becomes `[s] Settings` | **Fixed, and it amends four locked FRs** — see §2 | `3040b7a` |
| 8 | The theme chooser collapses into Settings as a WATCHPOST UI group; `[t]` and `[M]` leave the header and deep-link | Fixed | `46b34b8` |
| 9 | The Settings columns must balance themselves as groups are added | Fixed | `0b0d042` |
| 10 | The correspondent note sits flush and must never resize the window; shorter wording; drop the size | Fixed | `2bcee8f`, `157160e`, `1ae4cc9` |
| 11 | The tone gutter should align with the correspondent controls | Fixed | `fa87d1f` |
| 12 | A stored cast that disagrees with the window it is presented in must reconcile | Fixed | `249049d` |
| 13 | Ticker events that stay for days need a date; Special Weather Statements should follow the ALERTS — EVENTS preference | Fixed | `85594e5` |
| 14 | `MARITIME` → `MARINE`, in the UI and on the air | **Fixed — a scope change, ruled in session** | `903ab7c` |
| 15 | Every severe category needs a ticker lane; `Sig. Quakes` → `Disasters` | **Fixed — a scope change, ruled in session** | `5949dd1` |
| 16 | A multi-lane burst should sound one tone, name who declared it once, then read each title | Fixed | `8830eb5` |
| 17 | The lane colours: swatches, then a hex ruling, a normalized ramp, and `[w]` tints mirrored darker | Fixed — colours are the HUM LEAD's own pass | `b20f3cd`, `c5cf708`, `dfc2d65` |
| 18 | A location changeover collides with an alert; alerts must always have priority | **Fixed** — see §3, where the BUILD-exit red team found the fix incomplete | `4afe731`, `d30afd5` |
| 19 | Ticker times should follow the Radio Convention; military reads differently and spells callsigns in NATO phonetics | **Fixed — a scope change**: `units`/`clock` were ruled BACKLOG at MVS-D-21 and built here | `d30afd5`, `f80a5be`, `254b2f6` |
| 20 | Lag between the alert tone, the centred takeover and the readout | Fixed | `cb10ff2` |
| 21 | The MIL ticker showed the feed's zone, not the listener's | Fixed | `e45d707` |
| 22 | The spoken tail still said "Setup" | Fixed | `60d9f70` |
| 23 | Monochrome does not theme the Settings modal, then any modal — the blue slate | Fixed | `6ab0cd6`, `45a9146` |
| 24 | Lookup should show a shimmering placeholder row immediately | Fixed | `2c04b71` |
| 25 | A truncated provider error should wrap so nothing is lost | Fixed | `0d92fdb` |
| 26 | `[S]` reworked into three aligned tables with ERROR TYPE and BLAME | Fixed | `e15f41c` |
| 27 | `www.nhc.noaa.gov` shows ✘ in `[S]` while the header says fine | Fixed — an unmeasured host is not a failing one | `3a200a9` |
| 28 | `API Status` → `Watchpost Status`, with an UPTIME/version row and an hourly release check | **Fixed — a scope change**: no FR asked for an update check | `eaf4e6d` |
| 29 | Reconcile "9 APIs" against 8 endpoint rows; drop the redundant "requests since launch" | Fixed | `f39294b`, `24883ab` |
| 30 | Any table we need is a go-studs table — that is why it was vendored | **Fixed, and recorded as a standing rule** | `255d406` |
| 31 | A comprehensive performance pass before BUILD exit | Fixed — `02-analysis/perf-pass-discover.md` | `119c0f7` |
| 32 | Chosen costs must be documented so they are not re-litigated; we do not reimplement lipgloss | **Fixed, and recorded as a standing rule** — `docs/accepted-costs.md` | `03b9000` |
| 33 | The ticker is not rotating through all the lanes — Disasters and Marine dominate | Fixed | `3601128` |
| 34 | `W A T C H P O S T` → `WATCHPOST Observer`, delineating the Observer edition | **Fixed — a scope change**: pre-emptive naming for an unbuilt Broadcaster edition, ruled deliberately | `afb9aca` |

## 2. Rulings this batch made that amend earlier decisions

Recorded here because the objectives and the gate table still describe the earlier state. **The
original text is not edited**: an amendment supersedes it and is dated, so what was locked stays
legible.

| ID | Amends | Ruling |
|---|---|---|
| **MVS-D-47** | FR-7, FR-10, FR-13, FR-14, NFR-5 and the M5 evidence row in `gates.md` §2 | The `[S]` cast table is deleted (UAT #7). The surface those requirements name is now **`watchpost report --verbose`**, which carries the same answers without the dashboard. The two diagnostics that lost their only visual surface — a `[keys]` override naming a retired action, and an unknown `[radio.*]` key — are carried as an open question for 0.15.0, not silently dropped. |
| **MVS-D-48** | MVS-D-26 (`[M]` toggles All Tones On ↔ Mute) | `[M]` opens Settings at the tone rows instead of toggling. A one-key panic mute while the radio is talking is deliberately gone; the keymap help now says so. |
| **MVS-D-49** | MVS-D-21 (a 12/24-hour preference is BACKLOG) | The backlog item is overturned: `units` and `clock` ship in 0.14.0, because the ticker's own times were wrong without a preference to follow. |
| **MVS-D-50** | NFR-8 (no trace of `[V]` or `[T]`) | `[T]` and the min player are gone as required. **`V` survives** as a deep-link into Settings at the correspondents — the chooser modal it named is what was retired, not the key. |
| **MVS-D-51** | The 0.9.0 deferral of R-12(b) `--no-animation` | The 0.9.0 rationale — "no animation exists to disable beyond the shimmer" — was made untrue by the 0.12.0 ticker. `--no-animation` and `no_animation` ship. |
| **MVS-D-52** | — (new) | The hourly release check is **opt-in** (`update_check`, default off), disclosed in the README, and renders only version numbers this app parsed. |
| **MVS-D-53** | 0.13.0's decision that `ticker_radius_mi` scopes the tape only | The preference scopes the `[w]` window too. A deliberate reversal; the CHANGELOG names the surprise. |
| **MVS-D-55** | MVS-D-51 (`--no-animation` and `no_animation` ship) | **The flag is withdrawn before it ever shipped.** Frozen, the band showed one alert of thirty at 80 and 133 columns and two at 200, permanently, while its own counter said thirty — an accommodation that hid the alerts from the listener who most needed the text. Paging it was built and measured: all thirty become readable, but in 43 minutes of air on a single lane and **3 h 39 m across six**, which is the normal case. That is a technicality, not an accommodation. The reduced-motion surface is **`[w]`**, which lists the same alerts as a navigable table and is already where the spoken reports send a listener for detail. R-12(b) is answered by `[w]`, not by a flag. |
| **MVS-D-56** | MVS-D-54, and the 0.14.0 release scope | **0.14.0 holds for the read-lineup redesign** (`01-objectives/read-order-design.md`). The burst rules are not fixable as a defect — two rounds moved the failure boundary without removing it, because the ordering is a preference being treated as a constant. There is no schedule pressure and the requirement is that it reliably works, so it gets a FULL RCC and PLAN, after the remaining round-2 defects are resolved. The Director becomes the sole owner of the lineup, the queue, takeover insertion and ticker/read synchronisation — foundational for the Broadcaster edition, so the bar is bullet-proof rather than sufficient. |
| **MVS-D-57** | SAM-D-10's six-tab taxonomy, on the products it left unplaced | Any product naming **Warning**, **Watch** or **Advisory** is that thing first. Any *other* product naming **Statement** goes to **Spec. Statements** — not only the Special Weather Statement, as before. **Marine** products go to the Marine tab. A Coastal Flood Statement used to reach no tab at all: the office had told the listener something and the app decided not to pass it on. |
| **MVS-D-58** | SAM-D-10 / objectives §5, which declared the Air Quality Alert not shown | An **Air Quality Alert is an Advisory**. It is an advisory in everything but the word, and a listener told to stay indoors should find it where the other advisories are. Matched by exact product name, not by "Alert", which would sweep in products from other programmes on a coincidence. |
| **MVS-D-59** | SAM-D-10's six-tab taxonomy; the window's own name | A seventh category: **Forecasts and Outlooks**, worn as **Forecasts** on the tab the way Special Statements is worn as Spec. Statements. A Hydrologic Outlook is `urgency: Future`, `certainty: Possible`, `severity: Unknown`, covers a whole forecast area and reads as discussion — issued *below* a watch. It had been reaching no tab at all. **No ticker lane**: the marquee is for what is happening, not what might. Its tint is the only one in the set carrying no hue — a neutral slate, least chroma for the least urgent thing the window shows — and it is registered in the contrast list rather than passing by omission. The window is renamed **NOTABLE EVENTS AND FORECASTS**, covering non-weather hazards (a landslide is not weather) and products that are not bad news. |
| **MVS-D-60** | SAM-D-10, and MVS-D-57's remaining gap | **The civil-emergency family is routed.** A red team's sweep of the live 111-product catalogue found eight products still reaching no tab — including `Evacuation Immediate`, the highest-urgency product the Weather Service issues. Ruled: `Civil Emergency Message` and `Local Area Emergency` → **Disasters**; `Child Abduction Emergency`, `Blue Alert` and `911 Telephone Outage` → **Spec. Statements**; `Extreme Fire Danger` → **Watches**. `Evacuation Immediate` gets its own category **and lane**, *Emergency Orders* — **pending a colour ruling**, see below. `Administrative Message`, `Test` and `Test Message` stay not-shown, which is correct. **`Evacuation Prepare` and `Evacuation Aware` do not exist in the NWS catalogue** — the staged Order/Warning/Advisory form is other services' (Genasys, CalFire). Their rulings are on file for the day a source provides them; no dead branch was added for a product nobody issues. |
| **MVS-D-61** | MVS-D-60; SAM-D-10's tab set | **Emergency Orders** is an eighth category **and an eighth ticker lane**, carrying `Evacuation Immediate` — the one product that is an instruction rather than a description. Everything stays in the one window rather than a separate `[e]` modal, which the HUM LEAD preferred and which measurement showed is possible: the bar fits at every width including the 80-column floor. The tab wears **Emergency**, as Forecasts and Outlooks wears Forecasts. **Its colour is a PLACEHOLDER** pending the ruling — a new colour, or the red with every other lane shifted down. |
| **MVS-D-62** | MVS-D-61's placeholder; the 0.12.0 lane palette | **Emergency Orders takes THE RED; Disasters takes purple.** The red is the strongest signal the palette has, and an instruction to evacuate is the strongest thing the app can say — so the two are matched. No other lane shifts; the swap is between those two alone. **And every colour-named token is renamed to say what it IS**: `TickerRedBG`→`TickerDisasterBG`, `TickerOrangeBG`→`TickerWarningBG`, `TickerYellowBG`→`TickerWatchBG`, `TickerBlueBG`→`TickerMarineBG`, and the `EventCat` tints likewise — identifiers *and* their serialized token strings. A token named for a colour stops being true the moment the colour changes, which is exactly what just happened. |
| **MVS-D-63** | MVS-D-61's tab position; the window's category heading | **Emergency leads the tab bar**, left of Warnings — the only category that tells a listener to act rather than describing what is happening, and it sits first even when empty, which it usually is. The window still OPENS on Warnings unless a takeover just fired, in which case it opens on that takeover's own tab. **And the category heading is removed**: `Advisories — 14 active` sat directly above `14 Total Category Events`, the same number twice. The total line carries what the heading uniquely had — the cap, as `Showing N of M`, and any dead source. F-20 records the two things that would earn the heading back: a windowed running total, or a sub-type breakdown. |
| **MVS-D-64** | MVS-D-56; F-17 | **DISCOVER APPROVED for the Station Director & the Lineup** — a LEVEL-1 / SEV-0 major sub-feature, 2026-09-01, *"APPROVED; GO 4 PLAN"*. The Lineup becomes a first-class object with exactly one writer: three tracks (main track, alert rail, selectable bed), planning separable from execution, and every other system reading the lineup to decide what to pre-load, pre-build and sync. **The sub-feature carries its own ruling series** — L-1…L-8, S-1…S-8, T-1…T-5, R-1…R-6, Q-1…Q-4, RT-1…RT-9 — recorded with verbatim context in `01-objectives/lineup-model.md`, rather than ~35 numbers in this ledger. Requirements DR-1…DR-21 in `01-objectives/director-requirements.md`; risks RD-1…RD-12 in `02-analysis/director-risks.md`; the phase record and its critical analysis in `08-reports/discover-report-director.md`. Six decisions carried to PLAN as PD-1…PD-6, each owed a recommendation with reasoning. |
| **MVS-D-54** | MVS-D-12's burst script, on the point it did not settle | A burst is read **most-severe-first**, as ratified — the order does not change. What changes is the time bound: it is derived from `maxBreaking` rather than set independently, so a takeover can always read the whole burst the count bound selected. At a flat thirty seconds an outbreak spent the budget on tornado warnings and the live hurricane behind them was never spoken. A full eight-event burst now runs longer than before, which the HUM LEAD ruled acceptable in preference to reordering the read. |

## 3. Where the red team found this batch's own work wanting

Three of the fixes above were incomplete, and the BUILD-exit red team caught them. Recorded because
the pattern matters more than the instances: **each was a fix that satisfied the observed symptom
without closing the case behind it.**

- **UAT #18 (alert priority).** The fix chose hold-vs-dip once, from the deck's mode, at the moment
  the alert arrived — so a listener who tuned a relay mid-alert got a paused stream at full volume,
  and a relay that fell back to synth played dipped for the life of the report. The engine decides now
  and re-reads it every watch tick (`ef088ef`).
- **UAT #26 (the `[S]` rework).** The tone was assigned by cell position while the column set varies
  by form, so muting the second cell muted PROVIDERS at wide widths and STATUS at the app's own
  supported floor. Cells carry their tone now (`67d92c5`).
- **UAT #31 (the performance pass).** Its own allocation pins measured a dashboard with an empty
  ticker — the one state the app does not enter — and read ~10 % under the real cost at every size.
  The fixture carries a live tape now and every pin was re-taken (`e387db8`).

- **UAT #33 (the lane rotation).** Capping the tape per lane left the audio path untouched: the
  takeover still took its eight by severity alone, so an outbreak silenced the hurricane behind it —
  and because the cycle seen-marked everything on the same tick, silenced it permanently. The burst
  keeps a floor per lane now, and an event is seen-marked when it has been *read* rather than when
  it was queued, so a burst the takeover could not reach comes back instead of being lost.

Each was found by the remediation review loop (`06_docs/remediation-review-loop.md`), not by the
gates: every one of them passed `verify`, `p10`, the allocation pins and the goldens. The round's
full record — the six safety defects, what is still open, and what the round cost — is
`08-reports/red-team-build-round2.md`.

## 4. Process findings

- **The rulings in §1 and §2 existed only as commit subjects.** `"HUM LEAD, UAT 2026-08-30"` appears
  98 times in this diff as the governing authority and resolved to nothing; four separate red-team
  lenses traced a dozen findings back to it independently. §1 and §2 are the fix.
- **A gate was recorded green while failing.** P4's log records NFR-8's grep at zero over a README
  that still taught the retired `T` key and shipped the `radio-min` screenshot NFR-8 names.
- **The p10 evidence chain is broken at P1**: `gates.md` records a `tree_hash` that is not a git
  object and does not match `p10-p1.json`. The JSON is the evidence; the hand-copied column is a
  second place to be wrong.

## 5. Gates at exit

| Gate | Result |
|---|---|
| Unit + race | green; `-race -count=2` clean on the packages this batch changed |
| `make verify` | ALL GATES GREEN |
| `a2dh validate` | 17/18, one check skipped with declaration (the pass rule in `gates.md` §1 reads 17/17 and predates the eighteenth check) |
| `make p10` | **0 live · 0 unmatched** — one new ledger row, `app/release.go:start`, the same allowlisted poller shape as `sched.runTier` and `app/ticker.go:run`; **presented for ratification at this gate** |
| p10 ledger | **One new exemption, RATIFIED by the HUM LEAD 2026-09-01**: `platform/category` P10-05 invariant density. The registry is an immutable table plus five total accessors; six invariants sit where wrongness is reachable (a category with no row, a half-set lane, a gap in the rotation that would silently truncate the band, `All` listing each category once), and the table's completeness is pinned by tests rather than run-time checks. Same pattern as the 46 existing P10-05 package exemptions — `domains/radio/cast` and `platform/plaintext` are the closest. |
| `make alloc-budget` | green, every pin re-taken against a live marquee |
| PTY | `make pty-severe` green |
| Docs | `TestWhereThingsHappenNamesRealSymbols`, `TestAcceptedCostsNamesRealSymbols` green; NFR-8 grep re-run to zero |
| Goldens | re-recorded for the lane label and the masthead; reviewed row by row |
