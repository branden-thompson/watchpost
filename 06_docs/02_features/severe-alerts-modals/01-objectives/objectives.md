# Severe Weather / Disaster Events modals — objectives (DISCOVER / RCC)

**Feature:** `severe-alerts-modals` · **Target:** `0.13.0` · **Status:** DISCOVER — **EXITED** 2026-08-28
("APPROVED; GO 4 PLAN"); E-1..E-3 ratified per recommendation (SAM-D-24). PLAN open.
**Owner:** HUM LEAD (Branden Thompson) · **Opened:** 2026-08-28 (right after the 0.12.0 DEBRIEF)
**Brief:** `08-reports/project-brief.md` (approved 8e97c06, amended v1.2.0) · **Data:**
`02-analysis/data-shape.md` · **Red-team:** `08-reports/red-team-discover.md`

> **Decision namespace.** This feature's decisions are `SAM-D-n`. The bare `D-15` / `D-19` that appear in
> shipped source comments (`modes/tty/dashboard.go:9-10`, `:207`) are *watchpost-cli* rulings (keys are
> data; default bindings) and are unrelated. Symbols marked **(proposed)** do not exist yet.

## 0. What changes for the user (plain language)

Press **`w`** anywhere on the dashboard and a window opens listing every active severe-weather and
disaster event the app knows about, sorted newest first, in six tabs: Warnings, Watches, Advisories,
Special Weather Statements, Significant Quakes, Tropical. `←`/`→` change tab, `↑`/`↓` move down the
list, `enter` opens the full report for one event, `esc` goes back, `esc` again closes. Each tab is
colour-coded the same way in every theme. The spoken alert now says "press w" instead of "play the
broadcast of that location", and storms are announced by name ("Tropical Storm Dolly"). On a calm day
the window says so, rather than showing six empty tables.

## 1. Problem (locked, criterion-5 evidence corrected at red-team)

> A Watchpost user who has just been told a severe weather or disaster event is active cannot see which
> events are active, or read the full details of any one of them, without leaving the app or recalling and
> searching a location by hand.

**Criterion 5 (verifiable) — corrected evidence.** For a **US weather** event the path today is `l` →
type the place → `enter` → `esc` → `A` (`modes/tty/modal_location.go:95-98` opens Details for the
looked-up location; `A` renders the full record) — *recall and type a place name*, ≈ 5 actions plus
typing. For a **significant quake or tropical cyclone** there is **no path**: they have no
`snapshot.Alert` analogue anywhere. The brief's "∞ for any untracked-location event" overstated the weather
half; the statement itself stands (red-team A-F1, remediated).

## 2. Research waves

Six parallel lenses (A-1 feeds, A-2 population/de-dup, A-3 modal plumbing/keys, A-4 go-studs table,
A-5 colour tokens, A-6 performance), each with file:line evidence, then verify-before-accept spot-checks,
then a seven-lens red-team (four axes + DISCOVER PM lens + Perf/A11y/InfoSec/JuniorDev personas) whose
accepted findings are folded in below and dispositioned in `08-reports/red-team-discover.md`.

Corrections the red-team forced on this document: the memo target (FR-10), the pre-cap set's missing
labels (FR-11), the storm-name grammar (FR-12), the K2D baseline (§1), RS-1/RS-5 downgraded to HELD,
RS-11 promoted, five new risks, `--ascii` and empty-state requirements, ~15 citation fixes.

## 3. Functional requirements (refined by research + red-team)

| ID | Requirement | Refined by research |
|---|---|---|
| FR-1 | One action [**`w`**, alias `ctrl+s`] opens the modal from any dashboard state (brief R-1; SAM-D-19) | `w`, `W`, `ctrl+s` unbound (`modes/tty/dashboard.go:208-238`). `ctrl+s` is XOFF outside raw mode: bubbletea `MakeRaw` clears IXON in-app, an outer tmux/`stty ixon` can still eat it (RS-4) → `w` primary, `ctrl+s` alias. Reachability: a `"severe"` action resolved in `toggleModal` (`dashboard.go:618`) reaches the base dashboard and every read-only modal, and is inert while Setup / Add / Remove / choosers capture the keyboard (`handleKey` pre-switch, `dashboard.go:485-494`). Help group named: NAVIGATE (`helpGroups()`, `help_about.go:62`) |
| FR-2 | [`←`/`→`] switch six tabs in the order Warnings · Watches · Advisories · Spec. Statements · Sig. Quakes · Tropical; the current tab is **highlighted** (SAM-D-17). Opens on the most-recent-breaking category within 10 min, else Warnings; **the first body line names the open category** so a listener/`--ascii` user knows where focus landed | Re-interpret `alert-prev/next` inside the modal via `handleNav` (`nav.go:19-23`, precedent `:93-102`) → `handleSevereNav` **(proposed)** — no new arrow bindings. Selection = glyph + bold (watchpost-cli R-12a, `setup.go:343-376`); the selected tab also carries its category tint. **Tabs are a data table** **(proposed)** `severeTabs` in the `defaultKeyMap` idiom — label, colour token, population rule, order — so a seventh tab is one row, not eight edits (red-team C-19). The 10-min window is a named constant with the `d.now` clock injected for tests (`dashboard.go:302`) |
| FR-3 | [`↑`/`↓`] move the focused row; rail when the table exceeds the viewport (R-3) | the rail spans the table rows only (SAM-D-26 M-2), so `ScrollPanel`'s whole-body rail does not fit; `railify` (`body.go:293`) is the table-shaped precedent and becomes a glyph-aware `render.Railify` shared by RECENT and the window (PLAN; red-team C-7 objected to reusing it *inside* `ScrollPanel`, which is not what PLAN does — B-3 reconciled). Row marks via `rowMarks`'s fixed prefix (`platform/render/table.go:265-311`). Sort happens **at publish**, never per frame (red-team P9) |
| FR-4 | [`enter`] replaces the body with the record; [`esc`] back to the table; second [`esc`] closes (R-4, SAM-D-9). **The detail view's chip row reads `[esc] Back  [esc esc] Close`** (A11y Y1) | **No esc-backs-out precedent** — every `esc` closes outright (`setup.go:71`, `modal_chooser.go:19`, `modal_location.go:29`). Two designs for PLAN to compare: `severeDetail bool` (touches 2 switches) vs `modalSevereDetail` enum + breadcrumb (touches 5: `dashboard.go:485-495`, `nav.go:20-23`, `view.go:41-60`, `:81-95`, `:102-113`) — A-3 argued discipline, red-team C-4 argued cost. Either way the esc/esc rule gets a pinned test in `modal_test.go` |
| FR-5 | The record renders every field in the **frozen per-class render list** (`data-shape.md §4`, column "Render v1") — `COV` = 100 % of that list, encoded as a table-driven fixture (R-5, brief M2 v1.2.0). **`render.Plain` at every field path** (see NFR-6) | Denominator was "every field the feed supplies"; red-team (A-F3, S6, S7, C-18) showed that imports GIS/kmz URLs, seismic telemetry and attacker-named `parameters` keys no requirement asked for and the non-goals forbid. The render list is what the user reads; SAM-D-14 retention is honoured on the decoded set (§ data-shape §3) |
| FR-6 | Category colours standard + theme-independent **hue**, tinted on the active modal substrate, monochrome greys (R-6, SAM-D-7) | Independence today is by *omission* (Quattro omits Ticker tokens, `quattro.go:96-161`); a guard test **with a positive control** is required (red-team C-15a). Tokens **(proposed)**: four hue tokens (`EventCatRedBG/OrangeBG/YellowBG/BlueBG`, fixed RGB triples — *not* the Quattro `mix(p.red, p.darkBg…)` formula, which reads the active palette and is theme-dependent, red-team J4/Y3) + a `CategoryTone(cat, dark)` helper **(proposed)** beside `ModalTone` (`sgr.go:189`) that mixes the fixed hue onto `ModalBGDark`/`ModalBGLight`. No new text token (`AlertModalText` exists, `theme.go:89`); Watches/Advisories/Statements share Yellow (no duplicate token — C-17). Registration sites: `defaultTheme()` + Monochrome override only |
| FR-7 | Tone / inset / rail / chips match `[A]` (R-7) | `floatModalToned` + `render.ModalTone` (`view.go:160-169`, `sgr.go:189-194`); title `ModalTitle` + `─` fill |
| FR-8 | Narration tail + burst closing point at the modal (R-8; wording in PLAN, SAM-D-11); the spoken line **names what to press and confirms what opens** (A11y Y5) | `alertNarration` `app/ticker.go:381`, `burstClosing` `:387` — two strings, `NAR` |
| FR-9 | Browse view: total count, "Updated" stamp, per-row Declared / Expires, default sort Declared DESC (R-9, SAM-D-8). **Zone rule:** rows with a tracked-location tie use that location's clock (F17 precedent `alerts.go:186-194`); rows without one show the **viewer's local clock with the zone abbreviation** — one column never mixes zones silently (red-team B-F7) | Count = de-duped rows the tab renders (`data-shape.md §5.4`); "Updated" = newest successful source fetch (`dataAsOf`, `alerts.go:170-184`). 12-hour clock is the app convention (`app/ticker.go:390`) — accepted limitation |
| FR-10 | **Modal memo** for the modal family (R-10, SAM-D-13): the open modal's composed overlay string is rebuilt only when its inputs change | **Corrected target:** `modalLines()` (`view.go:100`) is called only from the scroll clamp (`nav.go:92`) — memoising it saves nothing per frame (red-team P1/C-1). The per-frame path is `View → modalView → floatModalToned → wrapModal` (`view.go:29-38`, `:160-166`); the memo wraps **`modalView(o)`**'s result, keyed as `bodyKey` is (`memo.go:22-41`; slot precedent `bodyMemo` `memo.go:45`) with `tickerGen`, a minute clock bucket, **without `shimmer`** (P4), and wrapping **only the visible window** (P5). Positive-control test: a frozen-clock run with a loading row must still hit. `tickNeeded` (`dashboard.go:336-341`) lists the open modal so its "Updated" stamp repaints when the ticker is empty (red-team P8) |
| FR-11 | Every ticker event present; larger free de-duped set as bonus (R-11, SAM-D-6) | The pre-cap set at `app/ticker.go:153` has **no `Location` yet** — `Locate` runs at `:168-169`, *after* `Active` (`:155`) and the radius branch (`:161-167`); the publish path must run `Locate` on the pre-radius set **before** filtering (red-team C-2). Publish = **a field on `TickerMsg`** (`modes/tty/dashboard.go:33-36`), not a new message type (C-17). De-dup key = normalised `urn:oid:` id **with grammar validation** (S4). Retained set hard-capped at 500 most-recent-wins, "showing N of M" (SAM-D-22) |
| FR-12 | **Storm names** on the tape and in narration; every decoded field retained (SAM-D-14, SAM-D-20) | `nhc.go:34` declares `name` and never reads it. **Not** by folding the name into `Type`: `Article()` keys on `Type[:1]` (`event.go:86-92`) → "*A* Tropical Storm Dolly"; a `Name` field **(proposed)** and a name-aware `Sentence()` ("Tropical Storm Dolly has been reported for …", no article) — golden written as a failing test first (red-team C-3) |
| FR-13 | **`--ascii` forms** for every new mark: tab selection glyph, row pointer, rail, spaced-letter headers (plain `EVENT`) — the shipped affordance README:91-92 documents for screen readers (red-team B-F2, new) | `render.Glyphs` (`platform/render/units.go:45`); panel primitives already have ASCII fallbacks (`panel.go:61`, `:80`) |
| FR-14 | **Empty state designed:** a calm day (the discovery probe returned zero national severe events) shows a single line per empty tab — "No active <category> events · Updated <t>" — and the Advisories/Statements tabs add "tracks your watchlist" when the watchlist is empty (SAM-D-16; red-team B-F9, new) | `domains/globalfeed/testdata/nws_active_severe.json` is `features: []` |
| FR-15 | **Severity glyph column** in the browse table so severity is conveyed within a tab by more than colour (A11y Y6, new) | `globalfeed.Severity` tiers exist (`event.go:25-31`) |

## 4. Non-functional requirements

| ID | NFR | Evidence chain |
|---|---|---|
| NFR-1 | **Zero new network fetches**, asserted by an httpx round-trip counter in the test (red-team C-16) | every source fetched in full per cycle; the cap is applied only at `Merge` (`app/ticker.go:145-181`); TTLs `usgs.go:27`, `nhc.go:12`, `nws.go:15` |
| NFR-2 | Closed-frame allocation budgets unchanged (`bench_test.go:34-45`). Open-modal budgets are **measured, then pinned at × 1.05** — the numbers have ONE owner, `03-architecture-design/plan.md §0` (the intake pair was withdrawn at red-team P2/C-6); a modal-open row is added to `frameAllocBudget`, and **80×24** is pinned as the worst case (P6) | `modes/tty/bench_test.go:25-45`, `:111-145` |
| NFR-3 | Re-parse churn bounded: an unchanged source body is not re-decoded. **Key on httpx's own "not modified / served from cache" fact** (conditional GETs already run, `platform/httpx/httpx.go:511-512`, `cache.go:222-232`) — surfaced as a flag from `GetJSON`; sha256 content-hash only as fallback; instantiate the existing generic `Memo[T]` (`domains/fire/memo.go:13-30`), error entries bounded to the TTL (red-team P7, S8, C-17) | SAM-D-14 keeps 1–3 KB prose; without this ≈ 650 MB/day churn (`data-shape.md §6`) |
| NFR-4 | Retained event set hard-bounded (P10-03: 500) and gauged in [S] | `MaxEvents` idiom (`stack.go:54`); `boxmemo` gauge (`domains/seismic/usgs/boxmemo.go:18`) |
| NFR-5 | Hostile-feed bounding on **both** paths: every new globalfeed field (`clampField`, `event.go:69`; numeric ranges; slice caps) **and** `snapshot.Alert` built by `mapAlert` (`domains/weather/nws/alerts.go:88-110`), which is unbounded today (red-team S2). `parameters` = an allowlist of named keys, never rendered as attacker-controlled labels (S7). URLs are not rendered in v1 (S6) | `data-shape.md §4` |
| NFR-6 | Feed text never addresses the terminal (S-F6) — **currently false** on the `[A]` path this feature reuses: `render.Plain` covers `Event`/`Description` only; `AreaDesc` (`alerts.go:218`), `Instruction` (`:226`), `Headline` (`:306`) are raw (red-team S1). Fix at one choke point in `alertRecordLines` + an escape-injection test, in this feature's BUILD | `app/ticker.go:365`, `alerts.go:201` |
| NFR-7 | ~~Radio untouched (R6) — narration change confined to `app/ticker.go` strings~~ **Superseded at UAT (2026-08-28):** the radio IS in this diff — the engine's held/aside preview lines and the visualizer tee, the composer speaking from the script tree, the pronunciation tables, the diagnostic log. R6 (the manual relay + audio smoke) is therefore a **blocking** VALIDATE item on both platforms, not an owed courtesy | R6 at VALIDATE, macOS and Linux |
| NFR-8 | Key bindings stay data (watchpost-cli ruling) and self-document in Help under NAVIGATE | `defaultKeyMap` (`dashboard.go:208`), `helpGroups()` (`help_about.go:62`) |
| NFR-9 | Interactive journey machine-verified in a real PTY: a sibling of `scripts/quality/soak-phases.expect` sends `w`, expects `─ SEVERE`, then `\x1b` twice; a second journey sends `\x13`. **This proves in-app delivery only** — the outer-tmux/`ixon` case is documented as a known limitation, not claimed verified (red-team C-15c) | no Go PTY harness (`go.mod` has no `creack/pty`) |
| NFR-10 | go-studs first; gaps logged as upstream candidates | `LocationTable` pattern (`platform/render/table.go:349-383`); one candidate: "per-row prefix/mark column + cursor index on `DataTable`" |
| NFR-11 | **Narrow-terminal layout:** the 110-col mock renders whole at ≥ 117 cols (`modalWidth`, `view.go:77-96`; 73/93/113 content cols at 80/100/120); EVENT is the single fill column and truncates first; goldens at 80/100/120 (red-team B-F6, new) | `alertCompactLine` degrade precedent (`alerts.go:302-322`) |
| NFR-12 | **Superseded guard:** a `references` entry suppresses a live alert only when it comes from the same sender and product and is newer by `sent` — on both the national path (`nws.go:131-138`, unguarded today) and the location path (`References` carried, never consumed, `alerts.go:102-104`) (red-team S3, RS-10/RS-16) | — |
| NFR-13 | **Seen-store hardening:** `seen.json` written `0600` in a `0700` dir (today `0644`/`0755`, `app/ticker.go:478-479`) with a size/entry cap on load (S5) | config precedent `platform/config/config.go:217`, `:232` |
| NFR-14 | Contrast gate that can **see backgrounds:** `TestThemeTokenContrastAA` iterates fg tokens only (`theme_test.go:199-217`); extend to fg×bg pairs including the four category tints on both substrates (A11y Y2) | — |

## 5. Constraints & dependencies

**Technical:** two data paths with different id forms (`data-shape.md §5.2`); national NWS curated to 8
products (`domains/globalfeed/nws.go:28-35`) so Warnings/Watches are a *union* of national + tracked
location alerts (SAM-D-15); neither go-studs table has cursor/viewport/rail (`data_table_row.go`,
`table.go` — the latter truncates by bytes, `:362-368`, and re-probes width, so it is not adoptable as-is);
`TickerCategory` has four lanes (`modes/tty/ticker.go:31-42`) — the six tabs are a **new** enum/table,
not that type; first `tz.Location` per zone does disk I/O (`platform/tz/tz.go:36-42`) — resolve at publish.
**Kit rules:** `NoAutoStyle: true` (patch 004), explicit width (001), bounded loops (008), composite SGR
pass-through (003), public-tree wording scrub (`LOCAL_CHANGES.md`).
**Dependencies on existing systems:** `app/ticker.go` cycle + seen-store; `modes/tty` modal enum /
`toggleModal` / `handleNav` / `modalView`; `platform/render` tokens, `ScrollPanel`, `bracketTitle`,
`rowMarks`; `snapshot.Location.Alerts` via `mapAlert`; `platform/httpx` cache validators;
`scripts/quality/*.expect`.
**Accepted limitations (documented, not risks):** English substring classification of NWS product names
(four of six tabs are US-only by construction — the README's "global" applies to quakes and cyclones);
12-hour clock; the 10-minute opening-tab window is a named constant chosen by HUM LEAD, testable via
`d.now`; `w` mnemonic names the first tab, not the modal (accepted by SAM-D-19).
**Organisational:** HUM LEAD approves every gate and directs the colour pass; SEV-0 HUMAN LEAD;
single-stakeholder discovery (HUM LEAD is also the primary listening user) — no external user evidence
exists for this feature (red-team A-F2, B-F3, recorded honestly).

## 6. Risk assessment

| # | Risk | Sev | Lik | Status | Mitigation |
|---|---|---|---|---|---|
| RS-1 | Widening `Event` ripples (seen-store, bounds, memory) | M | M | **HELD** | seen-store stores ids only (`ticker.go:461-465`, `:477`) — verified; per-field bounds (NFR-5) and the parse memo (NFR-3) are design intent until BUILD proves them |
| RS-2 | Two colour schemes collide (tape per severity, modal per category) | M | H | HELD | modal tokens are new and separate; the tape is untouched; guard test with positive control |
| RS-3 | Scope creep to a "world events browser" | M | M | HELD — **was realising** through COV=all-fields; cut by FR-5 v1.2.0 | non-goals; render list frozen; retained-set cap |
| RS-4 | `w`/`ctrl+s` eaten by tmux/`stty ixon` outside the app | L | M | HELD | `w` primary removes the XOFF exposure; PTY proves in-app delivery; limitation documented |
| RS-5 | Per-tick rebuild × six tables | M | H | **HELD** | FR-10 re-targeted at `modalView`; budgets measured at PLAN (NFR-2); the tick is already pinned by the ticker (`dashboard.go:339-341`), so the rebuild cost is real today |
| RS-6 | Source mismatch across tabs / dishonest counts | M | H | HELD | union rule + normalised id + honest count (`data-shape.md §5.4`) |
| RS-7 | Duplicate rows from the id-form mismatch | H | H | HELD | normalise with OID grammar validation; regression test with both forms |
| RS-8 | Re-parse churn from retained prose | M | H | HELD | NFR-3 keyed on httpx not-modified; measured in the BUILD soak |
| RS-9 | Hidden second modal state (esc/esc) drifting from Q6 exclusivity | M | M | HELD | PLAN compares bool vs enum with the switch-count evidence; either design ships with the pinned esc/esc test |
| RS-10 | Location-path superseded alerts shown twice | M | M | HELD | NFR-12 |
| RS-11 | Fixed dark tints on a **light-background terminal** (served today via `ModalTone(dark=false)`, `sgr.go:189-194`; `d.darkBG` from the terminal probe, `dashboard.go:417`) | M | M | SUPERSEDED (`categoryBlend = 1.0` at UAT: no mixing; AA holds on both substrates by the gate — REVIEW R5-A-07) (promoted from L/L — red-team C-8) | `CategoryTone` substrate mixing is mandatory, not optional |
| RS-12 | **Accessibility regression** — glyph-dense mock without ASCII forms; six tints with no bg contrast gate | M | H | HELD (new — B-F2, Y2) | FR-13, NFR-14 |
| RS-13 | **Empty / low-signal modal** — the normal day shows few or no events; the 500 cap may never engage | M | H | HELD (new — B-F9/B-F10) | FR-14; cap kept as the P10 bound, not as a UX expectation |
| RS-14 | **Narrow-terminal truncation** | M | M | HELD (new — B-F6) | NFR-11 |
| RS-15 | **Metric baselines wrong** — K2D "∞" overstated for weather events | L | — | RESOLVED (A-F1) | §1 corrected; brief M1 amended |
| RS-16 | **Warning suppression via `references`** — a crafted or malformed alert can mark a live warning superseded | H | L | HELD (new — S3) | NFR-12 |
| RS-17 | **Feed prose reaching the terminal unstripped** on the reused `[A]` path | H | M | HELD (new — S1; pre-existing defect) | NFR-6 fix in BUILD |

## 7. Open questions — all resolved

| ID | Question | Ruling | Status |
|---|---|---|---|
| OQ-10 | Warnings/Watches: national only or the union with tracked-location products? | **Union** ("QQ-10: Union") | RESOLVED → SAM-D-15 |
| OQ-11 | Advisories / Statements with an empty watchlist | Empty tab + hint; tabs never disappear ("Rec Approved") | RESOLVED → SAM-D-16, FR-14 |
| OQ-12 | Opening tab | Most-recent-breaking within 10 min, else Warnings; **current category highlighted** | RESOLVED → SAM-D-17, FR-2 |
| OQ-13 | Long NWS prose | **Retain + parse memo** ("Rec Approved") | RESOLVED → SAM-D-18, NFR-3 |
| OQ-14 | `ctrl+s` XOFF exposure | **`w` primary**, `ctrl+s` alias | RESOLVED → SAM-D-19, FR-1 |
| OQ-15 | Storm names in narration too | **Yes** | RESOLVED → SAM-D-20, FR-12 |
| OQ-16 | Per-class detail structs vs flat `Event` | **Per-class structs** | RESOLVED → SAM-D-21 (red-team C-18 asks to re-open — see §11) |
| OQ-17 | Retained-set cap + wording | 500 most-recent-wins; `showing 500 of 1,431` | RESOLVED → SAM-D-22, NFR-4 |

**Escalations for HUM LEAD at the exit gate** (red-team findings that touch a ruling — see §11).

## 8. Cross-cutting implications synthesis (post red-team)

1. **Composition: SAM-D-14 + churn ⇒ parse memo, keyed on httpx's existing "not modified" fact** — keeping
   every decoded field at a 2-minute re-parse makes churn the dominant cost; httpx already knows when a
   body is unchanged (P7), so the memo is a flag, not a hash.
2. **Convergence ×3 on the memo target:** Perf P1, Code C-1 and the PM lens (B-F11) independently found
   FR-10's target off the render path — the strongest signal of the round; re-targeted at `modalView`.
3. **Convergence on the publish path:** A-2/A-6 (pre-cap set is transient), C-2 (and unlabelled) — the
   publish path runs `Locate` on the pre-radius set and rides a `TickerMsg` field.
4. **Convergence on "extend existing seams":** `handleNav` re-interpretation, `LocationTable` composition,
   `bodyMemo` pattern, generic `Memo[T]`, `TickerMsg` field, `ScrollPanel` rail — no new mechanisms (C-7,
   C-17 deleted four proposed ones).
5. **Contradiction resolved: COV.** "100 % of supplied fields" (brief) vs the non-goals and the
   InfoSec/Business lenses → COV is 100 % of a **frozen render list**; SAM-D-14 retention applies to the
   decoded set, and the GIS/telemetry/parameters tail is **not decoded in v1** (an explicit narrowing of
   SAM-D-14's reading — escalated, §11).
6. **Contradiction for HUM LEAD: SAM-D-21.** Per-class structs were ruled; C-18 argues a generic
   key/value render (FR-5) is the case where a flat struct wins. Escalated, not re-decided here.
7. **Contradiction recorded, declined: event-named narration instead of a modal** (B-F1). Naming the event
   in the narration is *also* being done (FR-12) and does not give quakes/cyclones a record (§1) — the
   modal is the ratified answer (SAM-D-1..13); the alternative is recorded as a considered negative.
8. **Risk-signal updates:** RS-1/RS-5 downgraded to HELD (mitigated-by-intent is not mitigated); RS-11
   promoted; RS-15 resolved; five new (RS-12/13/14/16/17), all HELD with concrete mitigations.
9. **Open-question updates:** OQ-10..17 resolved; no open questions; three escalations (§11).
10. **Implications for PLAN:** approaches — (i) per-class vs flat detail (pending §11), (ii) bool vs enum
    for the detail view, (iii) memo scope (family vs new modal); mocks at 80/100/120 cols in colour and
    `--ascii`, detail per class, empty state, narration script; the perf spike that sets NFR-2's numbers;
    the `severeTabs` data table; the TDD list (id grammar + normalisation, superseded guard on both paths,
    esc/esc, PTY `.expect` ×2, memo hit with a loading row, token-independence guard with positive
    control, fg×bg contrast, name-aware `Sentence()` golden, 80-col golden, httpx fetch counter,
    escape-injection on every rendered field).

## 9. Decisions (DISCOVER) — verbatim

| # | When | Decision | Verbatim |
|---|---|---|---|
| SAM-D-14 | 2026-08-28 (mid-research) | Retain every decoded field; storm names surface on the ticker | "We should not throw away and data we're parsing - esp storm names - we can figure now where that data gets displayed (ticker for sure for names)" |
| SAM-D-15 | 2026-08-28 | Warnings/Watches = union of national + tracked-location products | "QQ-10: Union" |
| SAM-D-16 | 2026-08-28 | Empty-watchlist tabs stay, with a hint | "QQ-11: Rec Approved" |
| SAM-D-17 | 2026-08-28 | Opening tab = most-recent-breaking (10 min) else Warnings; current category highlighted | "QQ-12: Rec Approved - the category should be highlighted so the user knows where they are" |
| SAM-D-18 | 2026-08-28 | Retain long prose + parse memo | "QQ-13: Rec Approved" |
| SAM-D-19 | 2026-08-28 | Primary key `w`; `ctrl+s` alias | "QQ-14: Oh good point - let's figure out a different key. Maybe [w] Warnings?" |
| SAM-D-20 | 2026-08-28 | Storm names in narration too | "QQ-15: Yes" |
| SAM-D-21 | 2026-08-28 | Per-class detail structs | "QQ-16: Per Class Structs" |
| SAM-D-22 | 2026-08-28 | Retained-set cap 500, "showing N of M" | "QQ-17: Rec Approved" |
| SAM-D-23 | 2026-08-28 | Red-team dispatched with the full lens set + all four personas | "GO" |

(SAM-D-1..13 are the intake decisions in `08-reports/project-brief.md`.)

## 10. Non-goals (confirmed for v1 unless HUM LEAD reopens)

No end-user sort modes, filters or search; no fetching **or rendering** of richer-record URLs (USGS
detail, NHC advisories, GIS/kmz); no storm cones / ShakeMaps; no new data sources (W-Pacific typhoons
remain a later add); no notification changes beyond the narration wording; unclassified location
products (Air Quality Alert, other Statements) are not shown in v1; **considered and declined:**
event-named narration *instead of* a modal (B-F1 — it is done *as well*, FR-12, and does not give
quakes/cyclones a record); extending `[A]` to page through global events (no location to scope it to);
a `--json`/report-mode listing (not a TUI answer to the locked problem).

## 11. Escalations for HUM LEAD (rulings touched by red-team)

| # | Red-team finding | Ruling touched | Options | Recommendation |
|---|---|---|---|---|
| E-1 | C-18: with FR-5 now a generic per-field render, a flat `Event` may beat per-class structs | SAM-D-21 | (a) keep per-class structs; (b) re-open at PLAN as an approach comparison | **(b)** — cost is one PLAN approach, and the render list decides it |
| E-2 | A-F3/S6/S7: COV = "every supplied field" imports GIS URLs, telemetry, attacker-named parameters | SAM-D-14's reading; brief M2 | (a) decode + retain everything, render the frozen list; (b) decode + retain the render list + name/URLs-not-rendered, leave the GIS/telemetry tail undecoded in v1 | **(b)** — "don't throw away what we parse" is honoured; we simply don't parse the tail yet |
| E-3 | C-17: a separate Yellow token for Statements is a duplicate today | SAM-D-7 | (a) one Yellow token shared by three tabs; (b) three tokens, one value | **(a)** — add a token the day a colour differs |

## BUILD amendments (2026-08-28, red-team round 3)

- **FR-10** — `tickNeeded` does not list the severe window: nothing in it is time-relative (the title stamp is
  absolute; the record's durations are composed at publish). Details and Status keep their ticks. Documented
  deviation (p3-build-log dev. 1), ratified with the BUILD exit.
- **FR-13** — the `--ascii` column headers keep the ratified mock's spread form (`E V E N T`, plan §5.4) pending
  the HUM LEAD ruling recorded in `08-reports/red-team-build.md` (B-08a); the objective's "plain `EVENT`"
  wording is the alternative on the table.
- **FR-15** — met by glyph count (`⚠⚠`/`!!` for the Red tier), not by tint; the glyph's per-tier tint is part
  of the HUM LEAD colour pass (B-03).

## UAT amendments (2026-08-28, HUM LEAD)

- **FR-15 waived.** The severity glyph column was removed at UAT ("they all have the same glyph, so it's not
  useful"): within a tab the category itself carries the class, and the record carries the CAP severity. The
  marks column is the pointer and the play mark (5 cells); EVENT took the two cells.
- **FR-9 clarified — "Declared" is the issue time.** A Heat Advisory issued at 09:00 for 20:00 showed
  "declared 20:00" (the onset) and read as bad data. Declared = `sent` (then `effective`, then `onset` when
  nothing else exists) on both paths; the record's timing line reads `Declared <issued>   Starts <onset>
  Expires <until>   (~window from the start)` when the hazard begins after the declaration. The ticker's
  event clock follows the same rule.
- **FR-6 tints set:** Warnings `#633500`, Sig. Quakes `#550909` (the tokens are now the on-screen values —
  blend 1.0); Watches/Advisories/Statements and Tropical stay as built pending the rest of the pass.
- **FR-13 / column headers:** the window's column headers are the dashboard's group bands (`GroupText` on
  `SevereHeaderBG`, the bracketed spaced lettering with colour off) — one header style across the app.
- **Column headers (items 7–10, 2026-08-28):** the EVENT band spans the marks + number + event columns (the
  dashboard's LOCATION group precedent) and keeps the spaced lettering; LOCATION / DECLARED / EXPIRES read
  plain; the bands meet in the gutters as the dashboard's group bands do, while the rows keep their gutters;
  the band's background is the open category's hue lifted 20 % toward white (`SevereHeaderTone`, themed with
  the hue tokens), its text fixed bold white (AA on every hue and theme — the themes' off-whites are not).
- **Marks (items 11–12):** the ▶ appears only on the event being read over the radio (the breaking item), in
  the dashboard's `StatePlaying` green; the › pointer alone marks the focused row.
- **Gutter (item 13):** 4 cells between the data columns (the mock's 2); the ladder's EVENT widths follow
  (`TestSevereColumnsLadder`).
- **[space] in the window (UAT, option B — 2026-08-28):** reads the FOCUSED EVENT over the radio, never the
  dashboard's location underneath: the event's own script (product for place, the record's meta as a
  sentence, "in effect for about …", the description and instructions trimmed at sentence ends — no clock
  times, no "Press W" tail), opening with "This is a Watchpost Severe Weather Notification Report.
  Notifications may be delayed and are not intended for life safety use." and closing with "This concludes
  this Watchpost Severe Weather Notification Report." (HUM LEAD 2026-08-28), spoken through the same duck / speak / restore path as a breaking takeover; the
  radio panel shows `EVENT · <product> · <place>` with the script as its paced marquee, then returns to what
  was on; the ▶ rides the row being read. A second [space] while one is reading is inert. Without a voice
  the overlay still shows for a reading-length hold. `[space] Read` joins the window's chips.
- **Narration priority (HUM LEAD 2026-08-28, "build it robust"):** one arbiter (`app/narrate.go`) owns the
  voice. Classes, highest first: `narrateBreaking` ("<event> has been declared" — always first, cuts a read
  on air) · `narrateRead` (a [space] read — ducks the broadcast, waits behind a takeover, is cut by one).
  Equal classes queue in arrival order; the broadcast is ducked once for a run of sequences and restored once
  when nothing waits; muted or voiceless sequences run their visuals and never dip the broadcast. A new
  narration source is a class constant and a `Run` call. The engine gained `Interrupt` (cuts the line in
  flight, the broadcast untouched) for the pre-emption *(deleted at red-team round 4, A-15 — dead once the
  arbiter paused instead of cutting)*.
- **Scripts as files (HUM LEAD 2026-08-28):** every spoken phrase of a report is a `text/template` file at
  `domains/radio/script/scripts/<report>/<part>.txt`, with `global/head.txt` and `global/tail.txt` inherited
  by any report that has none of its own; the app asks the library by (report, part) and never names a
  file; the same tree under `<config dir>/scripts/` overrides a phrase; a new report is a new folder. Reports
  today — **every** spoken report, each modular in itself (HUM LEAD 2026-08-28): `weather-radio` (head ·
  live · span · conditions · alert · tail), `fire-report` (head · count · strongest · incident · outside),
  `seismic-report` (head · count · quake · felt · more · link), `event-report` (head · opening · meta ·
  window · instructions · tail), `breaking` (single · burst-line · burst-closing), `voice-preview`
  (sample). The synth's `Composer` speaks from the tree; the Go computes only the words the frames take.
  The head/tail wording keeps the reports consistent with each other and compliant with the data
  providers' notification terms (not for life-safety use). **Why this shape:** a future "report package"
  ("location + maritime + fire, no seismic") is a list of folder names — each report already carries its
  own notice and speaks as a unit, so the frame need not know what is inside it.
- **A takeover pauses a read, then resumes it (HUM LEAD UAT bug, 2026-08-28):** a read that was on air
  when a takeover landed collided with it — the read's 45-second script renders for several seconds, and
  a takeover arriving mid-render found nothing in flight to cut, so the render finished and played over
  the announcement. Now a higher class SUSPENDS the lower one: its line in flight pauses (the engine holds
  the preview: `PausePreview`/`ResumePreview`), its holds stop counting air time, and a line it is still
  rendering waits for the air; when the takeover ends the read resumes where it stopped and finishes.
  Pinned by the narrator tests (suspend/resume ordering, the render-under-suspension case, air-time
  accounting) and the engine's pause test. The earlier cut (`Interrupt`) was removed at round 4 (A-15).
- **DETECTION column (HUM LEAD UAT 2026-08-28):** between LOCATION and DECLARED, 17 cells: how the event
  was established when the feed says — the NWS "SOURCE:" sentence read into a kind ("Radar Indicated",
  "Radar Confirmed", "Spotter Reported", "Law Enforcement", "Emergency Mgmt", "Public Reported",
  "Satellite") when the description carries one, else the CAP certainty ("Observed", "Likely",
  "Possible"; never "Unknown"); a quake reads its review status ("Reviewed", "Automatic"); a storm reads
  blank. `domains/severe.Detection`, pinned by table tests. The ladder drops EXPIRES, then DETECTION, then
  DECLARED; the window's ceiling is 130 cols so a 133-col terminal shows every column (the ratified 110
  ceiling and §5 mock are amended — the mock generator is not re-run; the goldens are the pin).
- **[space] reads the whole record (HUM LEAD UAT 2026-08-28):** no clipping — the parser's 4 000-rune prose
  bound is the only limit; the engine's play ceiling is 10 minutes and the render bound 2 minutes; a takeover
  pauses the read rather than cutting it, so length no longer costs the announcement anything.
- **Pronunciation rule — time zones (HUM LEAD 2026-08-28):** in every script a zone abbreviation is read as
  its name: PDT "Pacific Daylight Time", MDT "Mountain Daylight Time", CDT "Central Daylight Time", EDT
  "Eastern Daylight Time", UTC "Coordinated Universal Time" — plus their standard-time siblings, Alaska,
  Hawaii, Atlantic and GMT. The rule lives in `synth.Pronounce` (the voice-only pass every Say goes through:
  the broadcast, the takeover, the event read, the voice preview); the screen keeps the abbreviation.
- **Pronunciation rules as files (HUM LEAD 2026-08-28):** the tables behind the voice — zones, states,
  states-ambiguous, abbreviations, words — live in `domains/radio/pronounce/rules/<table>.txt`
  (`KEY<TAB>spoken form`, `#` comments); the synth's normaliser and its voice-only pass load them by name.
  Compiled in (no hot swap): the point is that a maintainer adds a rule by adding a line and never touches
  the logic. Pinned by the convention test, the contract test (every rule the synth relies on) and the
  synth's own pronunciation tests, which passed unchanged.
- **Help window (HUM LEAD UAT 2026-08-28):** a blank line of air under the title; two columns when the
  terminal is wide enough (the window sized from the widest group line: left + 4 + right + 3 + chrome), one
  column with the panel's scroll otherwise; each group rolls as a unit, the registry order kept and split
  once where the two columns balance best (NAVIGATE and RADIO lead the registry so they make the left
  column, as the mock draws); every binding stays listed (the mock omitted `?`, `q` and the alert `←`/`→`
  rows — they render under NAVIGATE). The `enter` label reads "Location Details" (the parenthetical made the
  column three cells too wide for 133 cols).
- **Watches and Spec. Statements get their own tints (HUM LEAD UAT 2026-08-28):** `EventCatWatchBG` and
  `EventCatStmtBG` beside the Advisories yellow (SAM-D-24 E-3's single yellow token is superseded). The
  asked values (#886800, #7D8800) fell below the 4.5:1 AA floor for the table's white text (2.9:1 / 2.2:1 on
  the off-white text of Gruvbox-style themes); HUM LEAD chose the same hues darkened to the floor: Watches
  #5E4800 (4.84:1), Spec. Statements #4C5400 (4.51:1). The header band lifts a light hue toward black
  instead of white, so it keeps its contrast on brighter tints. Monochrome gives them their own greys.
- **API Status window (HUM LEAD UAT 2026-08-28):** a blank line of air under the title; PROVIDERS beside
  REQUESTS when the terminal is wide enough (the window sized from the two blocks' widths), stacked
  otherwise; PIPELINES, ISSUES and DUMPS follow full width. The two-column rules are one owner
  (`modes/tty/columns.go`) shared with the Help window — the next window that widens takes them as they are.
- **Radio player facelift (HUM LEAD UAT 2026-08-28):** the head reads `WATCHPOST WEATHER RADIO • ♪ <station>`
  with `VOL - ████░░ + 55 ▶ PLAYING` at the right; under it the marquee TRACK — `│…│` across the module, a `░`
  fill when idle, LIVE RADIO on a relay, the voice's sliding window (centred when short) while it speaks; the
  visualizer rows sit inside the same track (three wide, one narrow); the `━━━` play line is gone (there was
  never a timeline to scrub); wrapped control rows centre under the first. A narrow player drops the title
  first, then reads the station's short form (`RadioStatusMsg.Short` — the event read sends
  `EVENT · <code> · <place>` with `domains/severe.ProductCode`: SPS, TOR, SVA … or the initials), then
  shortens. The compact (Size: Min) player keeps its two rows, title in capitals (`WWRADIO` when squeezed).
- **The player is a box (HUM LEAD UAT 2026-08-28):** `render.Box` — square corners, the rows inset 3 from
  each border, painted in the radio tone; the head one cell in, the track flush with the inset, the controls
  two cells in (the mock's rhythm). The box's two rules count in the layout (`BoxHeight`), so the recent
  window is two rows shorter at a given height (28 rows: 6; 39: 13; 50: 19); its borders and insets take 8
  cells of width, so the narrow head's short station shortens with … below roughly 96 cols (pinned at 84: `♪ EVENT · SPS…`).
- **The player's box is heavy (HUM LEAD UAT 2026-08-28):** `render.Box` draws the heavy box-drawing rules
  (┏━┓ ┃ ┗━┛) so the outline reads thicker than the light rails of the track inside it; `--ascii` keeps `+-|`.
- **The visualizer follows the narration on air (HUM LEAD UAT bug, 2026-08-28):** an event read played
  through the engine's preview player, which bypassed the visualizer's tap — the bars sat flat with Viz on.
  Every preview now tees into the tap like the broadcast, and ONE owner decides which narration the bars
  follow: `app/narrate.go: vizFor(class)` — every narration (an event read, a voice preview) except a
  breaking takeover, whose tone and lines play `PreviewAside` (no tap). Pinned by the engine's tap tests and
  the narrator's voice script (`speak:` vs `aside:`).
- **Alert module facelift (HUM LEAD UAT 2026-08-28):** the module is ONE row in the heavy box on the Alert
  Details modal's tint — warnings on `alert.modal.warning.bg`, advisories on `alert.modal.advisory.bg`:
  `02/02  ⚠ FLOOD ADVISORY - Temecula, CA  • Issued: 08/26 8:00 AM • Expires: 12/31 12:59 PM` with the
  paging chips at the right. The headline/body lives behind [A] ("dive in for details"); the full layout
  reclaims the rows (the 2026-08-27 five-row module is retired: 50 rows now hold a 21-row window, 39 a
  13-row). Narrow terminals give up the expires stamp, then the issue stamp, then the chip labels, then the
  title itself (UAT 35). *(Window sizes in this and the following amendments are each superseded by the next;
  the standing numbers are the header facelift's: 50 rows → 21, 39 → 13, 28 → 6 with thin bands.)* The compact layout keeps the one tinted row without the rules so the 28-row window
  holds (UAT 34). Without an alert the box stands, muted (`No active alerts · <label>`), so the layout never
  jumps (UAT 5.2 / 19.1).
- **No blank between the player and the alert box (HUM LEAD UAT 2026-08-28):** the two boxes separate
  themselves; the row goes to the table (50 rows: a 22-row window; 39: 14). At 28 rows the freed row lets
  the bands stay thick, so the window sits at its 3-row floor there (bands give last, UAT 2026-08-27).
- **The module's title wears the modal's dress (HUM LEAD UAT 2026-08-28):** `⚠ EVENT` is bold in the alert's
  modal tone (`alert.modal.warning.fg` / `alert.modal.advisory.fg`) — the same `1;<tone>` the record's title
  carries inside [A] — and the place is bold white (`modal.title`); the count and stamps stay in the modal's
  text tone.
- **Header facelift (HUM LEAD UAT 2026-08-28):** the masthead is a heavy box whose top rule carries the title
  (`┏━━ W A T C H P O S T  v… ━━━`) and the Updated stamp at the right (`  Updated: … (Just Now) ━━┓`); the
  controls and the API summary share the one row inside, `[S] Status` joining the controls before `[?] Help`.
  `render.BoxTitled` / `boxRule` / `BoxRuleWidth` are the one owner of a titled rule (Box is the untitled
  case). Ladders: the stamp gives (age, date, label) before the rule could overflow — at 80 cols the age
  gives; the version leaves the title before the title would; inside, the mute label shortens then drops,
  then the API summary loses its label, then leaves. Height 2 → 3 (chromeLines 11): 50 rows hold a 21-row
  window, 39 a 13-row; 28 rows thin the bands for a 6-row window again. 80x24 memo-hit budget re-pinned
  1 224 → 1 312 (the box's Block pass every frame — the stamp ticks).
- **BUILD-exit red-team round 4 (2026-08-28, after UAT):** 47 findings across three sectioned reviewers; 37 fixed.
  Behaviour changes worth knowing: a read paused for a takeover now survives the takeover's tone and lines on the
  real engine (held lines by identity) and resumes with the knob as it is; a read cancelled while suspended is
  discarded, never resumed; a line never starts under a takeover (it starts under the arbiter's lock); provider
  prose is Plain'd at the radio-status seam and in every script; the header box carries no tone of its own; the
  alert module shows `[severity]` in text when colour is off or under `--ascii` (R-12a); at the 80×24 floor the
  window shrinks below three rows rather than overflow the terminal; `esc` then `enter` is not `ctrl+m`; the
  severe chips have an 80-col `--ascii` floor form; the layout builds its rows once per frame (80×24 memo-hit
  962). Report: `08-reports/red-team-build-round4.md`.
- **Rulings on round 4 (HUM LEAD 2026-08-28):** (1) B-05 — the alert tones are LIFTED to AA on their tints at theme
  registration (`render.LiftToAA`, `withAA`): the hue is kept and only its lightness moves, as far as 4.5:1 needs
  (the default's warning red #BE5454 → a brighter red; the advisory pair already read); every theme's two pairs are
  pinned (`TestAlertTonesReadAAInEveryTheme`). The "dull" feel of the imported Omarchy themes is this same
  contrast floor — the lift is the one place to widen if more pairs are added to it. (2) C-07 — `.a2dh.yml` and the
  ledger stay LOCAL (not exposed publicly); the gate ledger says so. (3) C-06 — the rows were presented for
  ratification. Also: the Vista tail is considered resolved by A-01 (the held-line fix).
- **Lookup opens on the looked-up location (HUM LEAD UAT bug, 2026-08-28):** `[l]` set the focus to the first RECENT
  index and opened Details before the rebuilt RECENT snapshot arrived, so the modal read the row that held that
  index (the old top entry) for a frame or a fetch. The dashboard now remembers the lookup (`lookupRef`) and
  `selectedLocation` answers with an empty record in that location's name until the snapshot's row at the focus IS
  that location; hydrate stays off meanwhile; esc drops the wait.
- **Dashboard table column titles (HUM LEAD UAT 2026-08-28, the final tweak):** the column-title row reads like the
  severe window's header band — ONE row, each column's segment painted in a DIP of its group band's tint
  (`render.TableHeaderTone`: the band's hue mixed 35 % toward black — `tableHeaderDip`, the one knob — so the row
  follows the theme; a non-truecolor tint is used as it is), the label centred in bold white, adjacent segments
  MEETING at the gutter midpoints so the row touches end to end like the bands above; the rows below keep their
  gutters. The marks column is painted with no label; the EXTENDED spacer is a gap the neighbours share; the
  number column's label is the mock's `##.`. Colour off: the bands' bracket form per segment. The kit's `Header()`
  is no longer drawn — both header rows are ours over the one geometry (`groupHeader`, `columnHeader`). Pins moved
  from fixed label offsets to "centred over its column".
- **WX STN and ZIP data cells centred (HUM LEAD UAT nit, 2026-08-28):** identifiers of (mostly) one length, not
  arithmetic — the kit's `"center"` alignment on the two columns; every other column keeps its alignment. Pins moved
  one cell (`KCRQ` at 38, `92057` at 38 in the 115/131-col fixtures).

## REVIEW amendments (2026-08-29, red-team round 5 — four lenses)

- **The severe index keeps its own copy of the locations** (R5-B-01): the publisher's snapshot goes to the
  dashboard, which sorts alerts in place; the deck clones the locations and their alerts at `SetLocations`.
- **A cancelled narration wakes and is discarded** (R5-B-02): the job's context ending broadcasts the
  arbiter's cond var; `settle` discards a suspended job whose context ended rather than resuming it.
- **A voice audition is never the line in flight** (R5-B-03): `Engine.Audition` — the chooser's sample drives
  the bars but a takeover's pause holds the narration's line, not it.
- **`[space]` Read is offered only with audio** (R5-B-04): the hook is nil on a no-audio build; the chip mutes.
- **The tape is one line, windowed by cells** (R5-C-01): a feed name's newline/tab becomes a space; a wide-rune
  name never runs past the terminal.
- **A lookup lands by identity** (R5-C-02, B-09): the focus survives a priority publish; the rebuilt RECENT is
  searched for the ref wherever it landed; any other window drops the wait.
- **Provider text is text on every surface** (R5-C-05): labels are cleaned once at the snapshot assembler
  (`platform/plaintext`, the leaf both render and snapshot use); station and buoy names at the Details sites;
  bidi overrides and zero-width runes are dropped with the controls (R5-C-14); a modal title never overflows
  (R5-C-11); truncation is by display cells (R5-C-10); an override script part is bounded at 64 KB (R5-C-15).
- **Every alt chord reads as esc then the key** (R5-C-08): arrows, backspace, delete, non-ASCII runes too.
- **The renderer clamps a focus past the tab** (R5-C-07); **the alert module's page follows the list as shown**
  (R5-C-09); **ids get their own 200-rune bound** so both paths normalise alike (R5-B-05); `cap1` reads runes
  (R5-B-06); a closed window forgets its record view (R5-A-04).
- **NFR-1 is pinned as a boundary** (R5-A-01): `domains/severe` imports no network client — the counter form
  the objectives named is superseded by this structural pin. NFR-14 gains the column-title dip (R5-A-06).
- **The 80×24 floor holds with one favourite** (R5-C-06): every favourite past the first needs a row (the
  favourites table is never windowed) — stated in the README, not changed.
- Open for HUM LEAD: light-background terminals fail AA on every text token in every theme (R5-C-03 —
  `WindowBGLight` is a placeholder); the dark-theme pairs outside the gate (R5-C-04); the `--ascii` rulings
  (R5-A-02: B-08a/b); per-feature JSON decoding so one malformed value cannot silence a source (R5-C-12);
  the breaking goroutine's place in the shutdown wait set (R5-B-07); `severe.Guard` on the `[A]` path
  (R5-A-08); README screenshots (R5-D-01).

## REVIEW rulings and follow-ups closed (HUM LEAD 2026-08-29)

- **FR-13 ruled — plain `EVENT` under `--ascii`** (B-08a); the play mark's own ASCII form `*` (B-08b); every
  remaining mark (`✔ ✘ ♪ ▌ ░ — ·`) through the one glyph owner with ASCII forms; arrow keys named in words in every
  chip; a muted chip keeps its tone under `--ascii` with colour on (R5-C-13 closed).
- **Light terminals supported — Watchpost Light** (R5-C-03): the one light theme; a theme paints its own
  ground whatever the terminal reports (the light slots mirror the dark ones). The alert tile and the severe
  window keep their dark grounds in every theme; the window's title, header and marks are fixed tones like its
  tints (no one tone reads on both a dark tint and a light window at AA).
- **The AA lift widened to every painted pair** (R5-C-04, R5-A-06): `aaPairs` is the register the gate checks
  and registration lifts — text toward white or black, whichever reads on every ground it shares; a ground
  deepens when no text can (the gold ticker lane).
- **P10 rows ratified** (16, incl. `platform/plaintext`); the four stale reasons refreshed in the local ledger.
- **Screenshots**: the HUM LEAD's capture once 0.13.0 is stamped in the binary, before the public release.
- **The harness CLI's bare name** stays where it is the gate command (the Makefile variable, the script
  usage); the flow map's prose names the gate instead.
- **Follow-ups closed:** R5-A-08 (an alert's update replaces it in the tables and `[A]` — `dropSuperseded` at
  the publishers); R5-B-07 (the takeover goroutine in the shutdown wait set; a render bound to its job's
  context); R5-C-12 (feeds decode entry by entry); R5-C-14 (a combining run capped at two); the dead
  `table.header` token removed; the detail-modal memo (carried since 0.11.0) is the modal memo of AX-4 — resolved;
  the ticker's stage-2 duck (0.12.0) is the arbiter's duck on both relay and synth — resolved. Still open by
  design: the multi-alert circle visualisation (awaits a HUM LEAD mock).
- **Watchpost Light, corrected (HUM LEAD screenshot, 2026-08-29):** the frame's base tone was built as
  `38;5;<token>` — valid for a 256 index, but the Light theme's truecolor `TextBase` produced
  `38;5;38;2;40;40;40` (index 38, faint, black BACKGROUND): every dark ground in the screenshot and the two-tone
  box rules. `render.FgSGR` is the one owner of "a foreground token as an escape"; `frameText` uses it. The
  theme itself is pale everywhere now — the category tints, the ticker lanes, the `[A]` alert tiles, the chips —
  with dark text; the window's title, header and marks are the theme's tokens again (the band text is bold
  white unless the theme's title reads better on the band); the dip and every ground move in the text's
  direction. Watchpost Light is second in the picker. Pinned: `TestTheLightThemePaintsNoDarkGround`,
  `TestFgSGRFormsAValidEscape`.

## VALIDATE findings (2026-08-29)

- **`esc` then an UPPERCASE key was lost on a real pty (P1, found by the VALIDATE journey):** a lone `esc` has no
  timeout in the input layer, so it is delivered fused with the next key; an uppercase key arrives as
  alt+shift+letter with no text, and the split left `shift+s` — bound to nothing. `esc` then `S`/`V`/`T`/`M`/`A`
  all failed after any window. `splitEscFusion` now reads the letter (upper-cased, Shift cleared); pinned in
  `TestLoneEscThenKeyIsNotLost` (the model) and `scripts/quality/validate-journey.expect` (the real binary).
- The core journey (`validate-journey.expect`, P0 + P1 on a fresh HOME against the live feeds — Setup, first data,
  the window's tabs and a record, Lookup → Details, `[A]`, the theme chooser → Watchpost Light, Status, Help,
  quit) runs 20/20; `soak-1h.expect` + `soak.sh` are the RS-8 soak on the real binary (radio off).
- **`[space]` Read was muted for everyone (P1, found by HUM LEAD on the 0.13.0-stamped build, 2026-08-29):** the
  round-5 B-04 guard wired the hook only when the radio deck existed at config time — but the deck is attached
  AFTER the dashboard is built (`attachRadio` needs the model), so the hook was always nil and the chip always
  muted. The hook now decides at the press (inert with no deck — B-04 kept). Pinned by
  `TestNarrateEventDecidesAtThePress`; the VALIDATE journey now presses `space` in the window and expects the
  panel's `EVENT ·` overlay (22 steps).
