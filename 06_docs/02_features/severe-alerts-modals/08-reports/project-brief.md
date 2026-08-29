# Project Brief — Severe Weather / Disaster Events Modals

| Field | Value |
|---|---|
| Report | project-brief v1.1.0 — **amended v1.2.0 at DISCOVER red-team (2026-08-28), see Amendment Log** |
| Phase | pre-DISCOVER (collect-brief handoff) |
| Date | 2026-08-28 |
| Author of record | Branden Thompson (HUM LEAD) |
| Feature | `severe-alerts-modals` · target **0.13.0** (OQ-8) |
| Branch | `feature/severe-alerts-modals` (off `feature/global-ticker` @ `7a45c8c`, the 0.12.0 DEBRIEF) |
| Directives | LEVEL-1 · SEV-0 · HUM LEAD · FULL GIT · FULL DOCS · FULL REPORTS · FULL RCC · FULL PLAN · FULL DIAGRAMS · FULL TDD |
| Standing lens | **Performance & resource** — a dedicated concern and red-team lens at every phase (OQ-9) |
| Status | Brief APPROVED — HUM LEAD, 2026-08-28 ("1. Ratified 2. Approved"); report APPROVED ("Approved; Go 4 RCC"), committed 8e97c06 |
| Decision namespace | Intake decisions D-1..D-13 below are **SAM-D-1..13** (feature-scoped); bare `D-15`/`D-19` in source comments are watchpost-cli rulings |

---

## Executive Summary

Watchpost 0.12.0 added a global ticker that announces the world's largest active severe weather and
disaster events, but nothing in the app lets a person open those announcements: the only path to details
is to remember a place name and go find it. This feature adds an always-available, browsable window of every
active severe event — grouped by kind, most important first — that opens into the full record of any one of
them, and re-points the spoken alert at that window. The brief is approved; DISCOVER opens next.

## Summary & Intent

0.12.0 shipped the global severe-event ticker: a single-row marquee and narration of the largest active
hazards (significant earthquakes, tropical cyclones, US severe/tornado warnings and watches). It announces
but cannot be opened — its events belong to no tracked location by design (`domains/globalfeed/event.go:2-5`),
so the `[A]` Alert Details modal, which is scoped to the *focused* tracked location
(`modes/tty/alerts.go:131-147`), cannot show them. The narration's tail — *"Play the Watchpost Radio
Broadcast of that location for more details"* (`app/ticker.go:382`, `:387`) — therefore asks the listener to
remember a place and search for it, and for an event at an untracked location there is no path at all.

This feature adds the complement: a **Severe Weather / Disaster Events modal**, reachable from any dashboard
state with one action, that lists every active event in six categories ordered by likely importance —
Warnings · Watches · Advisories · Special Weather Statements · Sig. Quakes · Tropical — and drills into the
full record of any event in the same visual language as `[A]`. Each category carries its standard colour.
The narration tail is re-pointed at the modal.

- **Who benefits:** the person at the terminal at the moment an alert is read aloud or scrolls past.
- **Why now:** the ticker exists; without a way in, it is a headline with no story behind it.
- **If we do nothing:** the narration keeps sending listeners on a memory exercise, and most announced
  events remain unreachable inside the app.

## Locked Problem Statement

> **"A Watchpost user who has just been told a severe weather or disaster event is active cannot see which
> events are active, or read the full details of any one of them, without leaving the app or recalling and
> searching a location by hand."**

| # | Criterion | Score | Evidence |
|---|---|---|---|
| 1 | Bad Outcome | ✓ | an inability — cannot see the set, cannot read a record |
| 2 | Affected Humans | ✓ | the user at the moment of an announced event (listener or viewer) |
| 3 | Tech Agnostic | ✓ | no ticker, modal, feed or key named |
| 4 | Non-prescriptive | ✓ | a modal, a page, a list, a printed report would all address it |
| 5 | Verifiable | ✓ | announce an event; ask a user to reach its details; count steps — today a US weather event needs `l` + typing a place + `enter` + `esc` + `A` (≈ 5 actions), and a significant quake or tropical cyclone has **no path at all** *(v1.2.0: corrected from "for an untracked location, no path exists" — red-team A-F1)* |

**Score: 5/5 — LOCKED.** Ratified by HUM LEAD 2026-08-28 ("1. Ratified").

**Refinement trace.** Raw input carried the problem in mechanism terms — *"having to lookup a location from
memory"* — scoring 2/5 (+3 partial: "users" unspecific, "the ticker" a component, "from memory" the
mechanism not the observable). One pass named the affected human (the person just told), removed the
component, and stated the observable: no view of the active set and no record for most of its events.

## Metrics of Success

| ID | Metric | Symbol | Type | Definition (direction) | Measured in |
|---|---|---|---|---|---|
| M1 | Keystrokes to detail | `K2D` | Primary | Distinct actions from the dashboard to a named event's full record. Today *(v1.2.0)*: **∞ for a significant quake or tropical cyclone** (no record exists in the app); **≈ 5 + typing a place name** for a US weather event (`l`, query, `enter`, `esc`, `A`); ≥ 4 for a tracked location. Target **≤ 4 actions** (open, category, row, enter) with no typing — lower is better | scripted-PTY test (acceptance criterion; no post-ship telemetry exists by design) |
| M2 | Detail coverage | `COV` | Secondary | *(v1.2.0)* Share of the **frozen per-class render list** (`02-analysis/data-shape.md §4`, column "Render v1") that the detail view renders. Target **100 %**, higher is better. *Was "100 % of the fields the feed supplies" — red-team A-F3/S6/S7/C-18 showed that mandated rendering GIS/kmz URLs, seismic telemetry and attacker-named `parameters` keys the non-goals forbid* | table-driven fixture test over `domains/globalfeed/testdata/` (the probe samples, promoted there at PLAN exit — red-team B-1) |
| M3 | Narration re-point | `NAR` | Tertiary | The single-event tail and the burst closing direct the listener to the modal instead of "the broadcast of that location" — 2/2 strings | `go test ./app -run Narration` |
| M4 | Radio untouched + frame budget | `R6·PERF` | Maintenance | Live relay/audio PTY smokes green on both platforms; closed-modal frame allocations unchanged vs the 0.12.0 pprof baseline (~111 M total, ticker 0.6 M — `06_docs/02_features/global-ticker/08-reports/debrief.md:32-36`; procedure `watchpost-performance-quality-pass/06-key_learnings/quality-baseline.md:98`); open-modal per-tick cost bounded by the memo (R-10). No regression | `WATCHPOST_LIVE=1 go test ./app -run LiveRelay`; `WATCHPOST_DEBUG_PPROF` |

**Anti-solution check (Step 5).** A modal that lists only the ticker's thin fields (type · location · time)
satisfies `K2D` without solving the problem — there are no *details*; `COV` closes that hole. A fast modal
bought by rebuilding it every tick fails `R6·PERF`; a cheap modal bought by capping the list at the ticker's
30 fails the "every active event" reading of the problem (OQ-2). The four are read together.
*(v1.2.0)* M1–M3 are **acceptance criteria** verified before ship; `R6·PERF` is the only metric observable
after ship (the app has no telemetry, by design) — red-team A-F5.

## Requirements

Verifiable form; the stated key preferences (HUM LEAD's) in brackets are carried into PLAN as the default
binding, not as the requirement.

| ID | Requirement |
|---|---|
| R-1 | From any dashboard state, one action [**`w`** — *v1.2.0, SAM-D-19: `ctrl+s` is XOFF outside raw mode and was demoted to an alias*] opens the Severe Weather / Disaster Events modal |
| R-2 | While the modal is open, the user can switch between categories [`←`/`→`] in this order — **Warnings · Watches · Advisories · Spec. Statements · Sig. Quakes · Tropical** — with the current category visibly selected |
| R-3 | While the modal is open, the user can move focus through the category's events table [`↑`/`↓`]; the focused row is marked; the table scrolls under a rail when it exceeds the viewport |
| R-4 | With a row focused, one action [`enter`] opens that event's details **in place** — the record replaces the table body; [`esc`] returns to the table; a second [`esc`] closes the modal |
| R-5 | The details view renders every field the event's source feed supplies (`COV` = 100 % per source), in the `[A]` Alert Details record format (`alertRecordLines`, `modes/tty/alerts.go:198-236`) |
| R-6 | Each category renders in its standard, **theme-independent** colour — Red (disasters: Sig. Quakes), Orange (Warnings), Yellow (Watches, Advisories), Blue (Tropical) — as *tints* matching the existing alert-modal tints, never pure hues (monochrome theme renders greyscale, as the ticker does) |
| R-7 | The modal's tone / inset / scroll-rail / key-chip scheme matches the `[A]` Alert Details modal (`render.ModalTone`, `floatModalToned`) |
| R-8 | The severe-alert narration tail and the burst closing direct the listener to the modal instead of the location broadcast (wording scripted in PLAN with the mock — OQ-7) |
| R-9 | The browse view shows the category's total count, the data "Updated" stamp, and per-row Declared / Expires; default sort **Declared DESC** (end-user sort modes deferred — OQ-4) |
| R-10 | **Modal memo (folded in, OQ-9):** an open modal is rebuilt only when its inputs change (snapshot/feed version, selection, scroll), not every ~300 ms tick — the carried "detail-modal memo" follow-up, bounded to the modal family |
| R-11 | Every event in the ticker appears in the modal (minimum); the larger de-duplicated active set is listed where a source already delivers it for free (OQ-2) |

## Technical Constraints

- Details are bounded by what the three keyless feeds publish: USGS `summary/significant_week.geojson`, NHC
  `CurrentStorms.json`, NWS `alerts/active` (0.12.0 sourcing; no paid or scraped sources).
- The ticker's `globalfeed.Event` model is **thin** — id, class, severity, type, place, location, lat/lon,
  point flag, superseded, at, until, source (`domains/globalfeed/event.go:34-47`). It carries no
  description, instruction, magnitude, depth, wind or pressure. **OQ-1 resolved: widen the model** with
  per-class detail fields captured at fetch time (no extra network hop inside the modal).
- The national NWS query is **curated to 8 products** (`domains/globalfeed/nws.go:27-36`: Tornado / Extreme
  Wind / Hurricane / Severe Thunderstorm / Flash Flood Warnings; Tornado / Hurricane / Severe Thunderstorm
  Watches). Advisories, Special Weather Statements — and any *other* warning product (Winter Storm, Flood,
  High Wind…) — reach the app only through the **tracked locations'** own alert lists
  (`snapshot.Location.Alerts`, `platform/snapshot/types.go:168-187`), classified by
  `render.AlertIsWarning` (`platform/render/sgr.go:149`). The modal therefore merges two sources; NWS alert
  IDs are shared between them, so de-duplication is by ID.
- The ticker stack caps at 30 (`domains/globalfeed/stack.go:54`) and may be radius-filtered
  (`ticker_radius_mi`, `platform/config/config.go:150`); the modal lists the pre-cap fetched sets (OQ-2).
- Radio is sacred (R6 gate): the narration change touches `app/ticker.go` only; relay and audio paths must
  not regress; the Linux half of the 0.12.0 R6 gate is still owed.
- UI primitives come from `third_party/go-studs` where they exist (`components/table.go`,
  `data_table_row.go`) — no hand-rolled table; gaps become upstream candidates (feedback memory).
- `w`, `W` and `ctrl+s` are unbound today (`modes/tty/dashboard.go:208-237`; `s` = Setup, `S` = API
  Status). *(v1.2.0)* `w` is the primary key; `ctrl+s` remains an alias whose in-app delivery is proven by
  a PTY test (bubbletea v2 raw mode clears IXON), while an outer tmux/`stty ixon` layer is a documented
  limitation (RS-4).
- The 300 ms tick (`modes/tty/dashboard.go:330`) is kept alive whenever the ticker has events
  (`tickNeeded`, `:339-341`), so an open modal is rebuilt per tick today; R-10 bounds that for this modal
  family *(v1.2.0: the memo target is the composed overlay at `modalView`, not `modalLines()` — red-team
  P1/C-1; the earlier `:562` cite pointed at the Details hydrate, unrelated)*.

## Other Considerations

- **Browse mock** (HUM LEAD, 2026-08-28) — reproduce exactly: title bar with "Updated" stamp; tab row with
  `[←] Prev [→] Next`; spaced-letter column headers `E V E N T · L O C A T I O N · D E C L A R E D ·
  E X P I R E S`; `›  ▶` focus marks; `001.` numbering; `▲ █ ▼` rail; `nnn Total Category Events`; chips
  `[↑↓] Navigate [enter] Event Details [esc] close`. Colour is a separate pass HUM LEAD directs. Tab order
  amended by OQ-6 (six categories).
- **Detail mock** — `[A]` Alert Details format, extended with the category colour styles.
- **Prior art:** `alertDetailsModal` / `alertRecordLines` (`modes/tty/alerts.go:131-236`); modal nav
  (`modes/tty/nav.go:19-22`, `handleModalNav`); ticker lanes `tty.TickerCategory` — `CatQuake`,
  `CatTropical`, `CatWarning`, `CatWatch` (`modes/tty/ticker.go:31-42`), a subset of the six tabs; the
  0.12.0 DISCOVER/PLAN/DEBRIEF record (`06_docs/02_features/global-ticker/`).
- **Carried follow-ups from the 0.12.0 DEBRIEF (all four dispositioned):** detail-modal per-tick memo —
  **folded in as R-10**; ticker stage-2 audio duck — adjacent (same narration path), **not in scope**;
  Linux R6 half — verification only, still owed; multi-alert circle viz — **Deferred**, stays on the
  0.12.0 backlog pending HUM LEAD's design/mock *(v1.2.0: added — red-team hygiene A-1)*.
- **Non-goals (proposed, DISCOVER to confirm):** no end-user sort modes (OQ-4), no filters/search, no
  pagination UI beyond the rail, no storm cones / ShakeMaps, no new data sources, no notification changes
  beyond the narration wording.
- **Stakeholders:** HUM LEAD (approver; HUMAN LEAD mode; colour pass; Linux validation); the listening user
  (R-8 wording); junior human developers and future agents (the record).

## Discovery Handoff Package

### Areas to Investigate

| # | Area | Concrete artefacts |
|---|---|---|
| A-1 | **Field inventory per source** — exactly what each feed carries that the parsers drop (the `COV` denominator) | `domains/globalfeed/usgs.go:78-92`, `nhc.go:57-70`, `nws.go:75-171` vs the live/fixture JSON in `domains/globalfeed/testdata`; `snapshot.Alert` fields for the location-sourced products |
| A-2 | **Population + de-dup** — pre-cap fetched sets per source; union with tracked-location alerts; de-dup by NWS ID; superseded handling; per-category totals; radius interaction | `domains/globalfeed/stack.go` (`Merge`, `MaxEvents`), `app/ticker.go:170-201`, `app/dashboard.go` snapshot publish path |
| A-3 | **Modal plumbing** — one `modalSevere` state hosting two views (table ↔ record) with the esc/esc rule; how `modalAlerts` opens/closes/scrolls; key-map additions (`ctrl+s`, modal-scoped `←`/`→`) | `modes/tty/nav.go:19-45`, `modes/tty/dashboard.go` (`handleKey`, `modal` enum `:586-592`), `floatModalToned` |
| A-4 | **go-studs table fit** for the mock (spaced headers, focus marks, numbering, rail) within the scrub/patch constraints | `third_party/go-studs/components/table.go`, `data_table_row.go`, `LOCAL_CHANGES.md`, `patches/` |
| A-5 | **Colour tokens** — ~~a theme-independent Blue sibling to `TickerRedBG/OrangeBG/YellowBG`~~ *(v1.2.0: wrong — `TickerBlueBG` already exists, `platform/render/theme.go:74`, HUM LEAD's 0.12.0 colour pass; see objectives §2)* category→tint mapping matched to `AlertModalWarnBG/AdvBG`; monochrome behaviour | `platform/render/theme.go`, `quattro.go`, the SGR/AA test constraints (quattro memory) |
| A-6 | **Performance & resource (standing lens)** — memo design (R-10) across the modal family; open-modal per-tick cost with six tables; widened `Event` memory × pre-cap sets; seen-store/cache shape impact | `WATCHPOST_DEBUG_PPROF` baseline; `modes/tty/dashboard.go:339`, `:562`; `app/ticker.go` seen store |
| A-7 | **Location sort key readiness** (deferred sort) — distance to the default location given `HasPoint=false` zone-only alerts | `domains/globalfeed/locate.go`, `event.go:42` |

### Stakeholders to Consider

HUM LEAD (approves every gate; directs the colour pass; runs the Linux R6 half) · the listening user (R-8
wording) · junior human developers and future agents (documentation audience) · upstream go-studs (any table
gap captured as an upstream candidate).

### Risk Signals

| # | Risk | Why |
|---|---|---|
| RS-1 | Widening `Event` ripples | the seen-store persists events (`app/ticker.go:186-192`) and the P4 F5 field-bounding defence (`event.go:62-75`) must extend to every new field; larger structs × pre-cap sets raise memory (A-6) |
| RS-2 | Two colour schemes collide | the tape colours per **event severity** (a Tornado Warning is Red, `nws.go:28`); the modal colours per **category** (Warnings Orange). Resolved for the modal by OQ-3; the tape is untouched — DISCOVER must state the rule so neither leaks into the other |
| RS-3 | Scope creep toward a "world events browser" | six tabs, larger sets, sorts, filters, search all beckon; non-goals above bound v1 |
| RS-4 | `ctrl+s` swallowed by terminal/tmux (XOFF) on some setups | a hard-to-reproduce "key does nothing" bug; mitigation: PTY proof + a documented fallback binding |
| RS-5 | Per-tick modal rebuild × six tables | the open-modal path is the hot path while browsing; R-10 is the mitigation and `R6·PERF` the guard |
| RS-6 | Source mismatch across tabs | Warnings/Watches mix national (curated 8) and tracked-location products; Advisories/Statements are tracked-only — the "Total Category Events" line must be honest about coverage (OQ-10) |

### Open Questions

| ID | Question | Status |
|---|---|---|
| OQ-1 | Where do details come from | **Resolved** (HUM LEAD): (a) widen `globalfeed.Event` with per-class detail fields at fetch time |
| OQ-2 | What the modal lists | **Resolved** (HUM LEAD): "Must include all events in the ticker minimum, but if we can get a larger list for free that is properly de-duped, then that's a bonus" — the pre-cap fetched sets qualify |
| OQ-3 | Colour rule | **Resolved** (HUM LEAD): per-category, standard codes, theme-independent (always red/orange/yellow/blue), tints matching the existing alert modals — "not a pure orange or a pure blue" |
| OQ-4 | Sort | **Resolved** (HUM LEAD): Declared DESC as the tie-break chain; end-user sorts later |
| OQ-5 | Drill-down | **Resolved** (HUM LEAD): replaces the body; `esc` backs out to the table; double-`esc` closes |
| OQ-6 | Category set + order | **Resolved** (HUM LEAD): Warnings · Watches · Advisories · Spec. Statements · Sig. Quakes · Tropical; Advisories and Statements pull only for the tracked locations ("we should already be getting this info for free" — confirmed, see Technical Constraints) |
| OQ-7 | Narration wording (R-8) | **Deferred to PLAN** (HUM LEAD) |
| OQ-8 | Version | **Resolved** (HUM LEAD): **0.13.0** |
| OQ-9 | Fold the modal memo in; perf lens | **Resolved** (HUM LEAD): yes — R-10; "dedicated performance / resource concerns and lenses applied to all aspects of this project" |
| OQ-10 | **Warnings/Watches tab population** — national curated set only, or the union with every warning/watch product carried by the tracked locations (Winter Storm, Flood, High Wind, …)? And which tab do location-sourced *non-severe* warnings land in? | **Open** — for DISCOVER (A-2) |
| OQ-11 | Where do Special Weather Statements and Advisories sit when the tracked-location list is empty (fresh install) — empty tab with a "tracks your watchlist" hint? | **Open** — for DISCOVER (A-2/A-3) |
| OQ-12 | Opening tab: always Warnings, or the category of the most recent breaking event? | **Open** — for DISCOVER (A-3); intake recommendation: most-recent-breaking within N minutes, else Warnings |

### Problem statement status
Refined via `refine-problem-statement` (2/5 → 5/5, scorecard above) and **locked** by HUM LEAD 2026-08-28;
measurements validated for anti-solutions (Step 5).

### Sharpening observations
- SH-1: R-1..R-4 named keys; the brief keeps them as the stated default binding and states the underlying
  need, so PLAN owns the binding (modal-scoped `←`/`→` rebind from `alert-prev/next` is precedent, as
  `esc`/`↑↓` rebind inside `[A]`).
- SH-2: category colour vs severity colour are different schemes — resolved for the modal (OQ-3), the tape
  keeps its severity tiers (RS-2).
- SH-3: R-5's "all available details" became verifiable through `COV` and the A-1 field inventory.
- SH-4: the three-key sort reads as a tie-break chain dominated by Declared — confirmed as intended (OQ-4).
- SH-5: "MAJOR feature" at `0.12.5` conflicted with SemVer — resolved to 0.13.0 (OQ-8).
- SH-6: OQ-6 doubled the category count and introduced a second data source (tracked-location alerts);
  R-11, RS-6 and OQ-10/11 carry that into DISCOVER rather than leaving it implicit.

## Decisions

| # | When (PDT) | Decision | Verbatim | Rationale |
|---|---|---|---|---|
| D-1 | 2026-08-28 | Feature initiated at LEVEL-1 / SEV-0 with the full directive set | "INIT NEW MAJOR FEATURE \| '0.12.5-severe-alerts-modals' LEVEL-1; SEV-0; FULL GIT; FULL DOCS; FULL REPORTS; FULL RCC; FULL PLAN; FULL DIAGRAMS; FULL TDD" | HUM LEAD directive; matches the 0.11.0/0.12.0 posture |
| D-2 | 2026-08-28 | Branch `feature/severe-alerts-modals` created off `feature/global-ticker` before any file was written | — (FULL GIT) | the linear working history `main-publish` mirrors; branch-before-write |
| D-3 | 2026-08-28 | Problem statement locked | "1. Ratified" | 5/5 scorecard above |
| D-4 | 2026-08-28 | Brief approved | "2. Approved" | — |
| D-5 | 2026-08-28 | Details captured at fetch time by widening the Event model | "QQ-1. a" | no network hop inside the modal; details as fresh as the ticker |
| D-6 | 2026-08-28 | Modal population = ticker events minimum, larger free de-duplicated sets as a bonus | "Must include all events in the ticker minimum, but if we can get a larger list for free that is properly de-duped, then thats a bonus for the end user." | the pre-cap fetched sets are free |
| D-7 | 2026-08-28 | Per-category standard colours, theme-independent, modal-matched tints | "these are standard color codes for event type, so should be theme indepent (always red/orange/yellow/blue) and tints should match existing alert modals (so not a 'pure' orange or a 'pure' blue, etc)" | consistency with `[A]` and the ticker's fixed tiers |
| D-8 | 2026-08-28 | Default sort Declared DESC; user sorts later | "confirmed; we'll handle end-user sorts later." | bounds v1 |
| D-9 | 2026-08-28 | Drill-down replaces the body; esc → table; esc esc → close | "Replaces body (no need to open yet another modal) [esc] backs out back to table, double-esc closes modal" | one modal state, two views |
| D-10 | 2026-08-28 | Six categories in importance order; Advisories/Statements from tracked locations only | "[ Warnings ] [ Watches ] [ Advisories ] [Spec. Statements ] [ Sig. Quakes ] [ Tropical ] … those should only pull for our current list of locations (not every single on in the nation), so we should already be getting this info 'for free'" | importance-first; no new national query |
| D-11 | 2026-08-28 | Narration wording in PLAN | "Do it in PLAN" | scripted with the mock |
| D-12 | 2026-08-28 | Version 0.13.0 | "0.13.0" | SemVer minor for a new feature |
| D-13 | 2026-08-28 | Modal memo folded in; performance lens standing at every phase | "Yes; we should also ensure we have dedicated performance / resource concerns and lenses applied to all aspects of this project." | R-10; A-6; red-team perf persona every round |

## Source Documents

| Document | Location | Status |
|---|---|---|
| 0.12.0 global-ticker objectives (DISCOVER) | `06_docs/02_features/global-ticker/01-objectives/objectives.md` | found |
| 0.12.0 DEBRIEF (carried follow-ups) | `06_docs/02_features/global-ticker/08-reports/debrief.md` | found |
| Prior brief precedent | `06_docs/02_features/watchpost-performance-quality-pass/08-reports/project-brief.md` | found |
| Feature objectives (this feature) | `06_docs/02_features/severe-alerts-modals/01-objectives/objectives.md` | to be authored in DISCOVER |

## Next Steps

1. HUM LEAD approves this report → commit as the first commit on `feature/severe-alerts-modals`.
2. **PHASE: DISCOVER (FULL RCC)** opens with A-1..A-7; OQ-10/11/12 resolved before exit; objectives.md and
   the data-shape analysis (`02-analysis/`) authored; red-team at exit with the full lens set **plus the
   performance persona** (D-13); discover-report PRESENTed for approval.
3. PLAN (FULL PLAN, FULL DIAGRAMS): approaches, the ASCII mocks (browse + detail + narration script, OQ-7),
   memo design (R-10), perf budget, TDD test list.

## Appendix — Technical Notes (engineers)

*(v1.2.0)* Superseded by `01-objectives/objectives.md §3–§5` and `02-analysis/data-shape.md §1/§5`, which
carry the corrected citations; deleted here rather than maintained twice (red-team docs B-10). One intake
claim in the deleted text was wrong and is corrected there: NWS alert ids are **not** identical across the
two paths (URL form vs bare `urn:oid:`; `data-shape.md §5.2`).

## Amendment Log

| Version | Date | Change | Source |
|---|---|---|---|
| 1.1.0 | 2026-08-28 | Initial brief; approved by HUM LEAD ("Approved; Go 4 RCC") | intake |
| 1.2.0 | 2026-08-28 | Criterion-5 evidence and M1 baseline corrected (weather events reachable via `l`; quakes/cyclones ∞); M2 `COV` redefined against a frozen render list with an in-repo fixture; R-1 key `w` (SAM-D-19); A-5 Blue-token claim struck; 300 ms tick / memo-target cites corrected; fourth 0.12.0 follow-up dispositioned; Technical Appendix retired; decision namespace `SAM-D-n` | DISCOVER red-team (`08-reports/red-team-discover.md`): A-F1, A-F3, A-F5, B-1..B-6, B-10, hygiene A-1/A-2, P1, C-1 |
