---
title: "0.18.0 — REVIEW"
date: 2026-10-05
phase: REVIEW
sev: SEV-0
authority: HUM LEAD
status: "APPROVED by the HUM LEAD 2026-10-05 (D-279); VALIDATE next. Batch 138 green on the hosted full CI, macOS and ubuntu (run 37348280922)."
---

# REVIEW — 0.18.0

Four blind reviewers (D-265), each briefed from `06_docs/red-team-brief.md`, read
`dd3f2a52..20eab984` (batch 137) and ran what their lens needed in their own scratch directories:
the REVIEW phase lens (Principal QA, `QA`), Accessibility (`AX`), InfoSec (`IS`) and Performance (`PF`).
All four said **do not ship as it stands**. The QA and Accessibility reviewers found the same Critical
independently (QA-1 = AX-1).

## Dispositions

- **Fix** - changes no behaviour, UI or API, or carries out a ruling: batch 138.
- **HUM** - ruled by the HUM LEAD, one at a time.
- **0.19.0** - carried in `06_docs/follow-ups.md` with its reason.

| ID | Sev | Finding | Disposition |
|---|---|---|---|
| QA-1, AX-1 | Critical | Before the alerts feed lands - every open, 7-10 s live - the description says "No alert on the map covers or comes near" a place the station holds a warning for; no loading word anywhere; `TestNoAlertIsAStatedState` pins the sentence | HUM: **D-266, say loading** |
| AX-2 | Critical | Without a picture (`--ascii`, "Instead of the map"), `O` opens an undrawn Overlays menu that owns the arrows and saves changes | HUM: **D-267, the menu drawn as text** |
| QA-3 | Important | One hung secondary input spends the feed's 30 s, and zone resolution runs on an expired context | HUM: **D-268, alerts first** |
| QA-2 | Important | The answer-key test skips scenario 09 (failure) and has no 0-of-N scenario | Fix: 09 scored through the real parts; 0-of-N stays D-250's |
| QA-4, PF-1 | Important | History pruning visits 256 directories a pass from the first dataset, and empty month and series directories are never removed: past ~250 series, later series are never pruned (probe: 347 of 600 expired files kept after a month) | Fix: a cursor carried between passes, empty directories removed, a steady-state test |
| QA-5 | Minor | Every settled pan cancels the feed, zone fetches with it | Fix: zone fetches finish into the cache under their own bound |
| QA-6 | Important | RK-1's scripted-PTY journey on the real binary was never built | HUM: **D-271, build one** |
| QA-7 | Important | The answer-key test compares prefixes, ignores unkeyed alerts and asserts `Relation()`, not the words read | Fix: exact words, unkeyed alerts fail, the description asserted |
| QA-8 | Minor | M3's test reads a helper's string, not the screen | Fix: asserts `View()` |
| QA-9 | Important | Unknown place zones read as "does not lie in it" in the diagnostics | HUM: **D-274** - "unknown" |
| QA-10 | Minor | A failed in-view alerts request is silent | Fix: to the diagnostics (D-124) |
| QA-11 | Important | A panic in the map library ends the whole station | Fix: recovered in the map's calls and commands, the map released and marked failed, the diagnostics told |
| QA-12, QA-14, IS-4 | Important | watchpost's copy of the library's transport, and httpx's private-address refusal, refuse a local proxy: behind one the basemap never loads; the refusal misses CGNAT, 0/8, 198.18/15, NAT64 | Fix: the proxy the transport resolved is exempt; a `netip.Prefix` deny list; a proxy test |
| QA-13 | Minor | A wall-clock bound under `-race` in the gate | Fix: counts overlap instead |
| QA-15 | Minor | One basemap source; RK-5's fallback unresolved | HUM: **D-272, ship with one** |
| QA-16 | Important | The words path answers the problem statement wrongly while loading and when stalled; M1/M5/M6 not measured as defined | QA-1, QA-3 fixed (D-266, D-268); M1 D-252; M5, M6 D-251 |
| QA (D-250) | — | Revisit D-250 with QA-1 and QA-3's evidence | HUM: **D-274, D-250 stands** |
| AX-3 to AX-9 | Important to Minor | The cursor never placed (F3); the description alerts-only (F4); the edge warning without a picture (F5); state words (F6); two-column Maps tab (F7); outline digits without a key (F8); "With the map" naming and overflow line (F9) | HUM: **D-273, all to 0.19.0** with RK-12 - F-205 |
| IS-1 | Important | The opt-in debug server checks no Host or Origin: DNS rebinding reads it, a no-cors POST writes a dump; the response carries the dump's absolute path | Fix: loopback Host only, any Origin refused, no path returned |
| IS-2 | Important | Clear map data leaves the tide predictions in view and the `epa-uv-cities` record | HUM: **D-269, clear both**; CO-OPS requests name `application=watchpost.<product>` so the clear is exact (**D-277**) |
| IS-3 | Important | README says watchpost "sends nothing about you", against the map's own table | Fix: the sentence points at the map's table |
| IS-M1 | Minor | No committed fuzz target for the eight upstream parsers | Fix |
| IS-M2 | Minor | Upstream numbers not range-checked (a 1e308 gust drawn and kept 30 days) | Fix: physical ranges at parse; WFIGS incidents and perimeters off the Earth left out |
| IS-M3 | Minor | The station and CO-OPS clients allow http and private addresses | HUM: **D-270, harden both**; the radio directory, plain http, its own client (**D-276**) |
| IS-M5 | Minor | The debug server's bind failure and the map's Purge error are swallowed | Fix |
| IS-M6 | Minor | Map problems carry full error text with station ids and city names | Fix: host and status |
| IS-M7 | Minor | EPA's `state` in the path unescaped | Fix |
| IS-M8 | Minor | Clear's raw OS errors, with home paths, shown to the listener with no path to resolve | Fix: a count and "see diagnostics" (D-124) |
| IS-M9 | Minor | The README's history paragraph omits datasets and paths | Fix |
| IS-M10 | Minor | `govulncheck@latest` unpinned | Fix: pinned |
| PF-2 | Important | `View` walks the whole history store (30 s cache); a key press walks it twice | Fix: off the UI path, a running byte count |
| PF-3 | Important | Nothing bounds concurrent map Work loops: a held pan key starts N | Fix: a bound on loops in flight |
| PF-4 | Important | Clear map data unlinks the tile cache on the UI goroutine | Fix: the disk part off it |
| PF-5, PF-9 | Important, Minor | Temperature boxes, then rain/waves/UV/air, and radar boxes are fetched one after another | Fix: the temperature's boxes together; the rest after them is F-204. Radar unchanged: a box's six frames at once already hold its client's 30 a second (D-130), so boxes together finish no sooner |
| PF-6 | Minor | A cancelled leader fails the area-alert fetches that joined it | Fix: the shared fetch under its own bounded context |
| PF-7 | Minor | The cost estimate scans every US city 25 times on the UI goroutine | Fix: the view's state codes memoised |
| PF-8 | Minor | Every history `Put` re-reads the whole day | Fix: the held day updated in place |
| PF-10 | Minor | `SameOverlay`'s deep compare costs milliseconds per feed | Fix |
| PF-11 | Minor | Clearing the HTTP cache reads every body for its header | Fix: the header line alone |
| PF-12 | Minor | Radar budget trimming re-decodes PNG headers in a nested loop | Fix: each frame's charge once |
| PF-13 | Minor | bodymemo parses under its lock | Fix |
| PF-Q6 | Important | No outbreak-scale or long-retention benchmark; no multi-day soak | Fix: an accelerated prune soak and a 600-series steady state (PF-1); `SameOverlay` held at outbreak scale (200 alerts x 8 zones x 300 points); the feed's outbreak-scale timing is VALIDATE's, with M5 and M6 (D-251) |

## Found while fixing

| ID | Sev | Finding | Disposition |
|---|---|---|---|
| RV-1 | Important | The window's `feedForLayers` has dropped the feed's times since W10, so D-98's spans were never applied: every alert and quake on every frame | HUM: **D-275**, kept as UAT saw it; the time-based reveal is F-203 |
| RV-2 | Minor | The radio relay's directory is plain http, so the shared client could not be https-only | HUM: **D-276**, its own client |
| RV-3 | Minor | A sorted-query cache key let no prefix pick out CO-OPS's predictions | HUM: **D-277** |
| RV-4 | Minor | `FuzzMRMSTimes` found MRMS's times returned in the document's order, against `Times`' oldest first | Fix: sorted; the crasher kept as a regression seed |
| RV-5 | Minor | P10 after the fixes: `domains/uv` and `platform/history` under the density bar; the temperature row's "no fuzzer" untrue | HUM: **D-278**, ratified and reworded |

## REVIEW exit

**Recommended: approve REVIEW exit; 0.18.0 proceeds to VALIDATE.** Every finding above is fixed,
ruled or carried with its reason. Batch 138 holds the fixes:
- `make verify` green on `9e1bc292`, locally;
- P10 clean (`make p10`: 0 live, 0 unmatched, 0 unratified);
- the whole module green under the race detector;
- the scripted-PTY journey on the real binary (D-271), which on its first runs found q hanging the station.

The hosted full CI run on `9e1bc292` is the condition, as D-264's was.

**VALIDATE inherits:**
- M5 by its protocol and M6's thresholds (D-251), the thresholds to the HUM LEAD before SHIP;
- the release notes (the CHANGELOG's 0.18.0 entry) checked against what ships.

**SHIP inherits:**
- `release/0.18.0` squash-merged to `main` through a PR closing #22;
- local `main` reset to `origin/main` (D-259).

**Blind spots:**
- The REVIEW reviewers read the release at batch 137; batch 138's own changes were read only by their authors and the gates.
- The journey's Linux run is first exercised by CI.
- Timings in this release are one machine's, under load from parallel work.
