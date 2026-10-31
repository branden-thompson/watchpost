---
title: "0.19.0 — Propagation overlays — PROJECT BRIEF"
date: 2026-10-05
phase: DISCOVER (intake)
report_template: project-brief v1.1.0
level: LEVEL-1
sev: SEV-0
authority: HUM LEAD
directives: FULL GIT; FULL DOCS; FULL REPORTS; FULL DIAGRAMS; FULL RCC; FULL PLAN; FULL TDD; FULL INST
branch: feature/propagation-overlays
paired_releases: "go-ionomaps (new, D-1, D-2); go-tuiMaps v0.3.0 (D-3)"
issues: "branden-thompson/watchpost#25 (MUF), #12 (memo keys), #27 (the history roll-up, D-55)"
status: "APPROVED by the HUM LEAD 2026-10-05 (D-16) as the intake record. Problem statements LOCKED (D-7). Where DISCOVER's rulings changed it (D-17 to D-83), requirements.md is current and wins; this brief is not rewritten to match."
---

# New Major Dual Projects | `watchpost-0.19.0` + `go-ionomaps`

> **This is the intake brief, approved at D-16.** DISCOVER then answered its open questions and changed its
> scope: the path (D-40), foF2 kept (D-24), the Broadcaster's answer (D-57, D-76, D-77), targets from the dry
> run (D-49, D-75), the reference chart (D-76), no recording (D-78), and more. **`requirements.md` is the
> current record and wins on any conflict.** The brief is kept as written, with these banners and the
> corrections already noted inline.

**LEVEL-1; SEV-0; FULL GIT; FULL DOCS; FULL REPORTS; FULL DIAGRAMS; FULL RCC; FULL PLAN; FULL TDD; FULL INST**

This is watchpost's brief. go-ionomaps has its own
(`go-ionomaps/06_docs/02_features/go-ionomaps/01-objectives/project-brief.md`). go-tuiMaps v0.3.0 gets
its brief in DISCOVER, once the host requirements below are settled, as v0.2.0 did (0.18.0 D-11).

## Summary & Intent

Watchpost's Broadcaster serves FRS, GMRS and amateur radio (issue #25). On amateur HF, whether a
transmission reaches anyone depends on the ionosphere at that hour. Watchpost can tell an operator the
weather, but not whether the band they are on reaches the people they serve.

0.19.0 brings **HF propagation** into watchpost:
- the maximum usable frequency (MUF) and, if DISCOVER keeps it, the F2 layer's critical frequency (foF2), as overlays in the Observer's map window (D-1);
- an answer at the Broadcaster console, whose shape DISCOVER decides: the Broadcaster's own map (F-174) or something smaller (D-15);
- in words as well as in the picture (M5b, D-9).

Then, in this order (D-4):
- 0.18.0's accessibility carry-over;
- the memo-key audit (#12);
- the `say` hang and orphans (F-187, F-188).

**Why now.**
- The research is done (`../00-research/rcc-muf-overlays.md`, 0.18.0 D-238).
- go-tuiMaps v0.2.0 already draws a whole-globe grid with labelled contours.
- The history store was sized for "an ionosonde's MUF every 5-15 minutes" (0.18.0 D-179).
- Issue #25 asks for it.

**Who benefits.**
- The Broadcaster station operator relaying on HF (PS-1).
- The Observer listener who operates HF (PS-2).
- The listener without the picture: the carry-over, and MUF's own words (M5b).

**What happens if it is not built.** An operator relays a weather broadcast on a band that reaches
nobody they serve, and learns of it only if someone tells them. The Observer ham keeps a second
window open on a website (prop.kc2g.com), whose code and output carry no stated terms.

**Three projects, one outcome.**
- go-ionomaps computes, or obtains, the ionospheric fields. It is a new public Go library (MIT, D-2), with its own problem statement (PS-G, D-11).
- go-tuiMaps v0.3.0 learns to draw them (D-3).
- watchpost puts them in front of the operator.
- watchpost pins the other two's release candidates in BUILD and ships on their final tags.

## Problem Statements — LOCKED (D-7)

> **PS-1, the Broadcaster operator.** "A station operator relaying watchpost's broadcasts on amateur
> HF cannot tell, from the station, whether the band they transmit on will reach the area they serve
> at that hour, so a broadcast can go out on a band that carries it nowhere near its listeners, and
> the operator does not learn of it."

> **PS-2, the Observer ham.** "An amateur HF operator using watchpost cannot tell, without leaving
> it, which bands will carry a signal between their place and the places they want to reach at a given
> hour, so they choose by habit or from a separate website, and call into bands that are closed or
> miss ones that are open."

Both were locked as written (D-7). The scorecard is in `problem-statement.md`. MUF governs HF
(3-30 MHz) only. FRS and GMRS are UHF, and this release does nothing for them.

## Requirements

These are intake requirements. DISCOVER refines them into `requirements.md`, where each row carries
its instrument (REFLECT L1).

### R-1 — The Observer ham's answer (PS-2)
- **R-1.1** MUF, and foF2 if DISCOVER keeps it, as overlays in the Observer's map window (D-1), with their valid time and age.
- **R-1.2** The answer between the listener's place and the places they want to reach, by UTC hour and band. How a "place to reach" is chosen is a DISCOVER question; the watched places are the obvious first set.
- **R-1.3** The same answer in words, with no picture (`--ascii`, "Instead of the map"), shipped with the layer (M5b, D-9).

### R-2 — The Broadcaster operator's answer (PS-1)
- **R-2.1** Whether the operator's band reaches the served area at this hour, from the console (M1, M2).
- **R-2.2** The operator is told, on screen and in words, when that band stops reaching it (M3).
- **R-2.3** The shape is decided in DISCOVER (D-15): the Broadcaster's own map (F-174: the tower, its service radius and the line-up's locations), or something smaller such as a band line on the console.
- **R-2.4** How the station learns the operator's band and served area is a DISCOVER question. A Setting is the default (P-2).

### R-3 — Sources, terms and credit (issue #25, need 1)
- **R-3.1** Every source behind a propagation answer is credited where watchpost credits its sources (About), under its own terms (G-M1).
- **R-3.2** Nothing is fetched before the operator asks, as 0.18.0's map (0.18.0 D-21, D-25), unless a ruling says otherwise (for example, for R-2.2's notice).
- **R-3.3** Keyless sources first (0.18.0 D-185). Each new third party is listed in "What leaves the machine".

### R-4 — Cost (issue #25, need 2)
- **R-4.1** watchpost's added CPU and memory with propagation on stay within G1, its target set in DISCOVER. Heavy computation, if any, belongs to go-ionomaps and is measured there (G-G1).

### R-5 — Honesty
- **R-5.1** The data's age is always said. Stale or partial data is said as stale or partial, never drawn as current (as 0.18.0's M3 and M4; G-M4 at the library).

### R-6 — Accessibility carry-over, after MUF (D-4)
- **R-6.1** F-191 to F-194, F-196 to F-198, F-180 and F-205, as carried from 0.18.0 (its D-244 to D-247, D-255, D-273).

### R-7 — Memo keys carry identity (#12, D-14)
- **R-7.1** Every memo key audited for identity against position. Each field left out of a key carries its reason. The guard (`modes/tty/memo_completeness_test.go`, F-30) already exists.

### R-8 — The gates stay reliable (D-15)
- **R-8.1** F-187 (an orphaned `say` outlives watchpost) and F-188 (`make test-say` hangs now and then) are fixed.

### R-9 — Learnings become instruments (REFLECT L1-L8)
- **R-9.1** Each metric's instrument is run once before PLAN exit (L1).
- **R-9.2** A pre-push hook runs the docs-reading tests (L5).
- **R-9.3** The code-quality brief asks about cross-layer rulings without an end-to-end test (L2).
- **R-9.4** Ruling questions cite `file:line` for every claim about the code (L3).
- **R-9.5** UAT verdicts become D-rows before their batch closes (L8).

## Host requirements on go-tuiMaps v0.3.0 (intake candidates)

From the research's gap list (`../00-research/rcc-muf-overlays.md`, "Missing"). DISCOVER settles the
list, and v0.3.0's own brief decides how.

| # | Need | Why | From |
|---|---|---|---|
| **HR-1** | A field that continues over water, settable by the host | a world MUF field otherwise shows land only | research gap 1; `internal/render/frame.go:92` has `FieldsOverWater` with no public setter |
| **HR-2** | A fill and legend for a host-defined field: a host colour ramp through `CheckRamp`, or `muf` and `fof2` presets | borrowing the temperature preset mislabels the legend and `Describe` | research gap 2 (`facts.go:188-207`, corrected by wave 1) |
| **HR-3** | Contour labels that carry a unit (MHz) | "12" alone is not an answer | research gap 3 |
| **HR-4** | A day/night terminator (candidate) | the ionosphere follows the sun | research gap 4 |
| **HR-5** | Station points coloured by value (candidate) | measured values beside the modelled field | research gap 5 |

## Host requirements on go-ionomaps (intake candidates)

| # | Need | Why |
|---|---|---|
| **GR-1** | MUF(3000) and foF2 over the globe for a UTC hour, with the time they are valid for and their age | R-1.1, R-2.1, R-5.1 |
| **GR-2** | The point-to-point answer for a path at a given hour and band (or what a host needs to compute it) | R-1.2, R-2.1 |
| **GR-3** | An error or an age, never a silently old or empty field (G-M4) | R-5.1 |
| **GR-4** | Each source's name and terms, readable by the host, so it can credit them (G-M1) | R-3.1 |
| **GR-5** | No HTTP client of its own that bypasses the host's single door: a host-settable fetcher (User-Agent, timeout; "allowed hosts" struck at A-12) | 0.18.0 HR-6's lesson (go-tuiMaps opened its own client) |
| **GR-6** | Its cost per update stated and bounded (G-G1) | R-4.1 |

## What leaves the machine

Propagation is global, so a request for the world's field need not say where the operator is. A
point-to-point request, or a planner keyed on the station's locator (as KC2G's planner is), would say
where the station is. DISCOVER lists every new party (GIRO, KC2G, any other source), what each learns
and when, as 0.18.0's D-31 table did.

## Metrics of Success — RULED (D-8, D-9)

Normative in `problem-statement.md`: M1-M3 (PS-1), M4, M5, M5b (PS-2), and G1, shared. They are not
restated here, so there is one copy. Targets are set from PLAN's dry run, each by its own ruling (D-49); M1
and M4's baseline is the IRI climatology, not KC2G (D-23).

## Technical Constraints

Measured against watchpost `538f3dbb` and go-tuiMaps `origin/main` (`d1c4d5e`, the v0.2.0 line), not
assumed.

### C-1 — The map is never wider than the place's region
0.18.0 D-8 and D-28, held by `TestNoFrameIsWiderThanTheRegion` (`modes/tty/map_bound_test.go:81`). A
propagation path's far end usually lies outside the region, and a MUF field is global. A wider view
needs a ruling amending D-8 and D-28, or an answer that does not need one (words, a table).

### C-2 — go-tuiMaps fields stop at the shore
`render.Input.FieldsOverWater` (`internal/render/frame.go:92`, read at `internal/render/field.go:182`)
has no public setter. HR-1.

### C-3 — No fill for a host-defined field
`classInk` returns 0 for preset 0, and the legend has no colours (research; the code is `facts.go:188-207`, corrected by wave 1). The presets
are fixed (`internal/colour/preset.go`). HR-2.

### C-4 — The Broadcaster has no map
`g` is bound in the Observer's keymap only (`modes/tty/dashboard.go:584`). The Broadcaster's map has
been a follow-up since 0.18.0 (F-174).

### C-5 — The reference implementation has no licence
`arodland/prop`, the code behind prop.kc2g.com, has no licence (research line 17). Its method can be
reimplemented from the published science. Its code cannot be copied. Binding on go-ionomaps.

### C-6 — The measured data is rate-limited and non-commercial
KC2G reads GIRO through a private FTP account (research line 26). GIRO's public FastChar endpoint
returned 429 on its first probe (research line 54). GIRO's data is CC BY-NC-SA 4.0. Live probes share
the HUM LEAD's IP and are budgeted.

### C-7 — The history store already has a place for it
It was sized for "an ionosonde's MUF every 5-15 minutes" (0.18.0 D-179), and fetched readings are
recorded (0.18.0 D-224, D-226). Whether MUF and eSSN are stored is a DISCOVER ruling.

### C-8 — Versions and import rules
- watchpost is `go 1.25.13` and pins go-tuiMaps `v0.2.0` (`go.mod:3`, `go.mod:9`).
- `modes/` and `platform/` may not import `domains/` (`scripts/lint-imports.sh`), which decides where propagation code may live.

## Other Considerations

- **D-4 reverses 0.18.0 D-256's order**, against the recommendation: MUF leads, accessibility follows. REFLECT's anti-pattern 5 was in-scope accessibility left to the end, where it is cheapest to cut. M5b (D-9) keeps the new layer from adding to that debt. The carry-over itself is still last.
- **Path A, B or C is DISCOVER's RCC** (research §Options):
  - A: consume KC2G's published grid, with permission;
  - B: GIRO plus our own assimilation;
  - C: the full Go reimplementation, as "port" in D-1 implies.
- **Writing to KC2G** (Andrew Rodland) is on the HUM LEAD's clock. It matters to path A and to the courtesy of a reimplementation, and is best decided once the RCC has framed what to ask.
- **Standing principles:** P-1 to P-5; Settings, not flags; errors need a path (0.18.0 D-124); keyless first (D-185); Open-Meteo is never probed for research.
- **Standing rules carried:**
  - rulings one at a time, with minor items as A-n rows under D-13;
  - blind red team;
  - both CI platforms green before merge;
  - no AI attribution;
  - no code in PLAN (0.18.0 D-13).
- **Issues of record:** #25, #12 and #27 (D-55). The release PR closes them. #18, #23, #9 and #24 were closed with their evidence (D-14).

## Discovery Handoff Package

**Areas to investigate**
1. The published sources and their terms: KC2G, GIRO (FastChar, DIDBase), Australia's Space Weather Services, NOAA SWPC, others. This also tests PS-G's claim (D-11).
2. Path A / B / C for go-ionomaps, with cost (G-G1), fidelity (G-M3) and the licence of each building block (IRI, PyIRI).
3. The reference reception reports for M1 and M4: WSPR, the Reverse Beacon Network, PSKReporter. Their access, terms and noise.
4. The baseline for M1 and M4, and the data's real cadence, which together set every target (D-8, D-9; since ruled: the IRI climatology, D-23, and targets from PLAN's dry run, D-49).
5. The view: C-1's bound against a global field. Words, a table, a wider frame, or a propagation view of its own.
6. PS-1's shape: the Broadcaster's map (F-174) or a smaller answer; how the band and served area are known.
7. go-tuiMaps v0.3.0's list (HR-1 to HR-5) and its brief.
8. Storage: whether MUF, foF2 and eSSN are recorded (C-7).

**Stakeholders.**
- The HUM LEAD: authority, UAT, amateur operator.
- KC2G: reference and possible source.
- GIRO / UMass Lowell: data owner.
- Listeners without the picture (R-1.3, R-6).

**Risk signals**
- **RK-1:** the reference has no licence (C-5). A reimplementation must be traceable to published sources, not to its code.
- **RK-2:** the data's terms (CC BY-NC-SA) and rate limits (C-6). GIRO offers access "only for educational and non-commercial research purposes", which binds every copy that fetches, not only whoever redistributes (corrected after the red team, IS-5).
- **RK-3:** the validation burden of path C (G-M3). The fit can grade itself unless readings are held out.
- **RK-4:** compute cost on a terminal app (issue #25, G1, G-G1).
- **RK-5:** three paired releases at SEV-0. The ceremony costs time (0.18.0 REFLECT on minor items, now D-13).
- **RK-6:** accessibility last again (D-4). Mitigated for MUF itself by M5b.
- **RK-7:** the D-8/D-28 bound against a global field (C-1).

**Open questions**
- **OQ-1** Which reception reports are M1's and M4's reference?
- **OQ-2** Is foF2 shown, or MUF alone (D-1, "potentially")?
- **OQ-3** How are the places to reach chosen (R-1.2)?
- **OQ-4** Path A, B or C, and is KC2G written to?
- **OQ-5** Does R-2.2's closure notice fetch while no map is open (R-3.2)?
- **OQ-6** Are MUF, foF2 and eSSN recorded in the history store?
- **OQ-7** Does the D-8/D-28 bound give way for propagation?

## Completeness Check

```
PROJECT BRIEF — COMPLETENESS CHECK
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  [✓] Header              — dual projects; go-tuiMaps v0.3.0 paired (D-3)
  [✓] Directives          — LEVEL-1, SEV-0, eight phase directives (D-1)
  [✓] Summary / Intent    — what, why now, who benefits, cost of not building
  [✓] Problem statements  — PS-1, PS-2 LOCKED (D-7); PS-G in go-ionomaps's brief (D-11)
  [✓] Requirements        — R-1..R-9 at intake; requirements.md in DISCOVER
  [✓] Host requirements   — HR-1..HR-5 (go-tuiMaps), GR-1..GR-6 (go-ionomaps), candidates
  [✓] Metrics of Success  — ruled (D-8, D-9); targets in DISCOVER
  [✓] Tech Constraints    — C-1..C-8, measured at 538f3dbb / d1c4d5e
  [✓] Considerations      — D-4's cost, RCC paths, KC2G, principles, issues
  [✓] Handoff package     — 8 areas, stakeholders, RK-1..RK-7, OQ-1..OQ-7

  [✓] Rulings             — D-1..D-15 verbatim; A-1..A-5 for veto
  [✓] Approval            — APPROVED as presented (D-16).  INTAKE CLOSED.
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```
