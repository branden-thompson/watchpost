# v. 0.16.0 — Broadcaster UI

*This issue body is the project brief of record, per the standing rule that the finalized brief
becomes the issue description.  Source: `06_docs/02_features/0.16.0-broadcaster-ui/01-objectives/project-brief.md`.
Intake closed 2026-09-09; the problem statement is LOCKED and rulings D-1..D-10 are recorded below.*

---

## New Major System Feature | `Broadcaster-UI`

**LEVEL-1; SEV-0; FULL GIT; FULL DOCS; FULL REPORTS; FULL DIAGRAMS; FULL RCC; FULL PLAN; FULL TDD; FULL INST**

## Summary & Intent

Watchpost can already run a station.  It cannot yet be *operated* as one.

0.16.0 adds **Broadcaster**, a second full-screen surface that runs alongside Observer and swaps with
it on `ctrl+b` and `ctrl+o`.  Where Observer is built for a person watching the weather, Broadcaster
is built for a person putting it on the air over FRS, GMRS, or CBRS: a rolling lineup of ten reads
climbing toward a LIVE slot, a priority layer above it for takeovers, an ON AIR / STANDBY control, an
audio bed drawn from real relay transmitters, and a settings set tuned for a tighter service area and
a higher request cadence.

**Why now.**  0.14.0 built the Director, the cards, and the two-track lineup with this release named
in its comments.  0.15.0 made the surfaces inspectable and gave every gate a watched failure.  The
foundation is real, and section *Technical Constraints* below measures exactly how real: the **read**
side of the schedule is finished, and the **write** side does not exist at all.

**Who benefits.**  Two people, and issue #10 already named both.  The **human operator** running the
station, who is the primary user of this UI.  The **human listener** on the radio channel, who never
sees it and feels every mistake made on it.

**What happens if this is not built.**  The operator keeps running a broadcast through an interface
built for a solitary observer: unable to see what airs next, unable to change it, and unable to
confirm they are transmitting at all.  That is the problem statement, and it is stated properly in
`01-objectives/problem-statement.md`.

## Problem Statement — LOCKED

> **"A person broadcasting hyper-local weather over a two-way radio channel cannot see what is about
> to be transmitted, cannot change its order before it goes out, and cannot confirm at a glance
> whether they are currently on the air — so they operate the station blind, and their listeners hear
> the wrong thing at the wrong time."**

**LOCKED by the HUM LEAD, 2026-09-09** (*"Lock; Approved"*).  Scored 5/5 against the five criteria.
Full evaluation, the diagnosis of the input it came from, and the five anti-solution-hardened
measures of value are in `01-objectives/problem-statement.md`.  Every scope argument in this release
traces to this sentence.

## Requirements

Stated as given, sharpened where a requirement was not independently verifiable, and numbered so
DISCOVER can trace each one.

### R-1 — Mode switching

- **R-1.1** The operator switches to Broadcaster with `ctrl+b` and to Observer with `ctrl+o`.
- **R-1.2** **RULED (D-1).**  Broadcaster must be in **STANDBY** before a swap to Observer is
  permitted.  The alternative — an explicit confirmation handling the sign-off gracefully — is
  withdrawn from this release, because C-3 measured that the graceful-stop primitive it needs does
  not exist at any layer.
- **R-1.3** A switch taken deliberately mid-utterance must not truncate a word, leave dead air, or
  leave an audio path running with no owner.  Measured by M3.

### R-2 — Settings, shared and unique

- **R-2.1** Broadcaster provides its own Settings modal, specific to its experience.
- **R-2.2** **Shared** properties — theme, units, and the like — are the same data in both surfaces
  and survive a switch.
- **R-2.3** **Unique** properties — default location, service radius, and the like — must not
  collide with, nor pollute, the other surface.
- **R-2.4** An existing user's configuration file must survive the change.  A person upgrading from
  0.15.0 loses nothing and is not asked to reconfigure.

### R-3 — The lineup as a stack of cards

- **R-3.1** The lineup is a rolling view of **ten** cards, slots `0` through `9`, moving upward
  toward the LIVE position.
- **R-3.2** A separate layer sits above it for **priority takeovers**, with their own key bindings.
  The mock shows `[T]` and `[A]` for two takeover cards and `[B]` for the audio bed.
- **R-3.3** Choosing a slot `0`-`9` opens a modal for that event, where the operator inspects details
  and acts on the card.
- **R-3.4** The actions are at minimum **promote**, **demote**, and **drop**.  Any action must take
  effect without interrupting audio output.  Measured by M2.

### R-4 — Station state and audio

- **R-4.1** The operator sets **ON AIR** and **STANDBY**.  ON AIR means continuously broadcasting.
  **In mock for this release.**
- **R-4.2** An **audio bed** of relay streams is supported, **in mock**, and the operator selects
  which relay to use.
- **R-4.3** The offered relay selection is sensible for the operator's location and service radius.
- **R-4.4** A **GAIN** control is present, drawn as the mock draws it.

### R-5 — Minimum size

- **R-5.1** Broadcaster declares a minimum width and height, because it is an operating surface and
  an operator must not be shown a broken one.
- **R-5.2** Below that minimum, Broadcaster shows a **clear notice** rather than rendering past the
  terminal bounds.  Silent overflow is a defect, not a degradation.  Measured by M5.
- **R-5.3** **AMENDED (D-4).**  The mock measures **150 x 73**, and that is the *reference
  rendering*, not the minimum.  Broadcaster should support narrower than 150 through dynamic
  resizing and breakpoints; **150 becomes the floor only if the responsive approach fails.**  PLAN
  vets the approach, and C-10 records that `platform/term` already defines the breakpoint vocabulary
  and nothing calls it.

### R-6 — Inferred from the mock — PROVISIONAL BY RULING (D-8)

These are read off the mock rather than stated in the intent.  **They are deliberately provisional
and are refined through UAT** — this is new UX, and the ruling expects them to move.  None may be
treated as locked, and any of them may change without a scope change.

- **R-6.1** A masthead carrying BROADCAST LOCATION, TOWER GPS as a latitude and longitude pair, and
  SERVICE RADIUS in miles.  **The mock renders the pair as `<lat>, <lon>` by ruling D-10**; the real
  values are R-7's concern, not the mock's.
- **R-6.2** The Observer key row is retained — Settings, About, Status, Help, Quit — plus the API
  health counter.
- **R-6.3** The station-state banner reads as a single line offering both states, with the inactive
  one shown as the alternative.
- **R-6.4** `SHIFT + ENTER` toggles the station state.
- **R-6.5** The lineup scrolls, and the scroll indicator sits on the right edge.  This is what makes
  a terminal shorter than the full stack acceptable.
- **R-6.6** Cards carry a class marker — the mock renders `•PRIORITY•` and `•STANDARD•`.
- **R-6.7** The audio bed row names the transmitter, its frequency, and its distance from TOWER GPS,
  and carries its own STANDBY state.

### R-7 — The named location and its coordinates must harmonize

**Stated by the HUM LEAD at intake, 2026-09-09, and not inferred from the mock.**  This is a real
requirement rather than a provisional one.

- **R-7.1** In the running application, the broadcast location's **name** and its **coordinates** are
  one fact with one source of truth.  A masthead that says `Bonsall, CA` beside a coordinate pair
  somewhere else is a defect, not a display choice.
- **R-7.2** The coordinate pair is a **first-class value**, not a formatted string assembled for the
  masthead, because the planned Go TerminalMap service-radius visualization consumes it.
- **R-7.3** The service radius is expressed against that same pair, so the radius, the relay
  selection of R-4.3, and any future map all measure from one origin.

**Why this is a requirement and not a PLAN detail.**  C-11 records that a service-radius map is
greenfield — `platform/geo` offers `HaversineKM`, `BearingDeg` and `CompassIndex` and nothing else,
and there is no canvas or plotting anywhere.  The map is out of scope for 0.16.0, but the thing that
would make it expensive later is shipping a masthead whose coordinates are cosmetic.  R-7 is the
cheap half of that future, paid now.

**The mock is explicitly exempt.**  D-10 ruled the mock's coordinates a placeholder precisely because
illustration does not need real ones.  R-7 governs the code; it does not govern the mock.

## Metrics of Success

Five, defined and anti-solution-hardened in `01-objectives/problem-statement.md` §5: **M1** next-item
certainty, **M2** time to correct the running order, **M3** unsafe mode switches, **M4** settings
bleed, **M5** silent overflow.

**M3 is the one this release is most likely to fail**, and it is the only requirement in this brief
that can leave a physical transmitter keyed with no software owner.

## Technical Constraints

**Everything in this section was measured against the tree at `f29cc1f` during intake, not assumed.**
Citations are file and line.  This section exists because the intent says the foundational pieces are
in place, and that claim deserved a measurement rather than agreement.

### C-1 — The read side of the schedule is finished.  The write side does not exist.

`platform/lineup` is pure value code with a two-track model: `MainTrack` and `AlertRail`, and the
rail drains first unconditionally (`lineup.go:171-188`).  A `Card` already carries `Origin`, `Refs`,
`Max`, and a `State` machine (`card.go:233-307`).  Rendering ten cards is available today.

**There is no promote, demote, reorder, or skip anywhere in `platform/lineup` or `app`.**
`Lineup.Remove` exists and its own comment says it "is not the Operator's DROP control, which arrives
with the Broadcaster UI" (`lineup.go:248-249`).  `Origin.FromOperator` is declared and given a display
string (`card.go:203,214`) and **is never constructed in production code** — verified.  `Card.Max` is
present and inert by the same design.

**This is the single biggest gap in the release**, and it is a gap the architecture anticipated: the
lineup is held as mutable state in one serial goroutine, not recomputed each tick, precisely so an
operator's edit can survive a cycle.

### C-2 — No second surface can drive the audio today, and one prior claim does not hold

`mastercontrol` is the sole owner of the band and the duck, enforced by an AST gate
(`app/mastercontrol.go:12-24`; `modes/tty/one_band_writer_test.go`).  `platform/singleowner` is a
**compile-time linter**, not a runtime primitive.

**The "generational reader" recorded as evidence that a second surface could drive the same audio
concurrently does not exist as such.**  The only occurrence in the tree is a comment in
`app/severe_read_test.go:517` describing the severe-read path.  There is no concurrency mechanism of
that name.  `newPump`'s own comment is the honest version: *"the second caller is the Broadcaster,
which is exactly when nobody will be looking for it"* (`app/pump.go:69-70`).

**Also corrected:** `Burst.Cards` does not exist.  It was deliberately removed — `Burst` holds one
`Takeover` whose `Refs` are the single ordering, and the comment records that a second ordering had
no production consumer (`plan.go:188-200`).  Any planning document repeating the two-field shape is
stale.

### C-3 — There is no graceful stop.  Audio is cut, and that was deliberate.

`Engine` is the single owner of output (`domains/radio/player/engine.go:53`).  Stopping pauses the
player "at once", with the comment *"silence at once; Close follows (UAT 81: stop must feel
instant)"* (`engine.go:592-594`), and `StopPreview` hardened this further after a read stayed audible
past its stop (`engine.go:695-701`).

**No fade, no crossfade, no drain-then-stop exists at any layer.**  "Sign-off" today is *script text*
a report ends with when it completes normally (`synth/compose.go:202-203`), not a teardown mechanism;
it contributes nothing if playback is killed mid-sentence.  **R-1.2 therefore asks for a primitive
that does not exist**, which is why OQ-1 offers the STANDBY-first rule as an equal alternative.

### C-4 — Continuous output does not exist either

All audio is event-driven: a report is composed, spoken, and ends.  A relay is continuous *while
tuned*, treated as a five-minute rotation window (`app/radio.go:372`).  Nothing keeps a carrier
running when the operator is not tuned to one.  **ON AIR as a real continuous state needs a primitive
the codebase does not have** — which is exactly why R-4.1 is scoped to mock for this release.

### C-5 — The relay catalogue already holds everything the mock's bed row draws

`stream.Table` is an embedded CSV of **1,035 transmitters** (`stream/table.go:64`), each carrying
`Callsign`, `Site`, `State`, `FreqMHz`, `Lat`, `Lon`, `PowerW`, and `Status` (`table.go:28-37`).
`Table.Nearest(lat, lon, n)` returns entries sorted by distance (`table.go:122`).  The mock's
`KIG78 Coachella CA 162.400 MHz · 41mi from TOWER GPS` is **computable today**.  What is missing is
only the notion of a service radius as a *selection bound*; selection is currently per-location.

### C-6 — Volume exists; gain does not, and nothing persists it

`Engine.Volume(pct int)` clamps 0-100 and applies live (`player/engine.go:223`).  It is a **single
global value** across relay and synth, drawn already as a block-glyph bar with `[-]`/`[+]` stepping
(`modes/tty/radio_panel.go:59-80`).  It is **not persisted** — it defaults to a hardcoded `55` on
construction (`modes/tty/dashboard.go:426`).  The word "gain" appears nowhere in the codebase.

### C-7 — The configuration has no namespacing at all, and the migration pattern already exists

`Config` (`platform/config/config.go:235-270`) is flat with typed sub-structs and **no scoping,
profile, or edition mechanism** — every key has exactly one value slot.  There is no schema version
field.

The good news is the machinery around it.  `Mutate` is the sole write path and is enforced by an AST
gate (`config.go:413-424`; `one_writer_test.go`).  Writes are atomic by temp-file and rename at 0600
(`config.go:430-479`).  Decoding is **non-strict**, so unknown keys are ignored, and `keepUnknown`
exists specifically so a save does not destroy keys the running build does not know
(`keep.go:14-24`).  `withToneCompat` (`config.go:335-352`) is a working one-shot migration from
0.13.0 and is **the template R-2.4 should follow**.

Config lives at `~/.config/watchpost/config.toml` on **both** macOS and Linux — hand-rolled XDG, not
`os.UserConfigDir()` (`config.go:279-302`).

### C-8 — The root model has no seam for a second surface

`Dashboard` is handed directly to `tea.NewProgram` (`app/dashboard.go:350`) and **there is no mode,
screen, or surface abstraction above it** (`modes/tty/view.go:19`).  The modal system is one-at-a-time
over a closed `modal` enum whose own comment names Broadcaster as the reason it has no default case
(`view.go:130-136`), and the header already carries an edition concept (`render.EditionObserver`,
`platform/render/sgr.go:51`).

So the modals and the render toolkit are genuinely additive; **the root is not.**  Something must
become the model `tea.NewProgram` receives.  **That is the central architecture fork for PLAN, and it
is OQ-2.**

`platform/render` is inherited wholesale — themes, AA contrast, ASCII fallback, panels, wrapping,
`Wordmark`, and the `term.KeyMap` engine.  The masthead, layout, and status band are **not** in
`render`; they are Observer's own and must be built again.

### C-9 — `ctrl+b` and `ctrl+o` are free in this codebase, and `ctrl+b` is not free in the world

Existing chords are `ctrl+c` quit, `ctrl+s` severe, `ctrl+d` debug, and `ctrl+a` add-location
(`dashboard.go:309-319`).  Neither `ctrl+b` nor `ctrl+o` appears anywhere.  Bindings live in one flat
namespace and `term.Merge` rejects duplicate claims across the whole map (`platform/term/term.go:159-162`).

**`ctrl+b` is tmux's default prefix key**, and `ctrl+a` — already taken here — is GNU screen's.  An
operator running Watchpost inside a multiplexer will not reach Broadcaster.  The repo already knows
this class of problem: the `ctrl+d` binding carries a comment about terminals claiming chords first
(`dashboard.go:313-316`).  **This is OQ-3.**

### C-10 — There is no minimum-size gate, and the code that would provide one is written and unused

**F-55 is confirmed, and sharpened.**  Rendering `Dashboard{width:20, height:5}` produces a frame
**57 cells wide and 26 lines tall** — the ledger says 25, so the row is off by one and should be
corrected.  Size is a bare assignment with no clamp (`dashboard.go:562-564`); `opts()` floors only the
*content* column count at 40 (`view.go:328`); the fixed chrome never shrinks (`layout.go:70-86`).

**`platform/term` already defines the answer and nothing calls it.**  `Breakpoint`, `BreakpointFor`,
and `HeightCompact` exist at `term.go:60-87`, and `BreakTooNarrow` carries the comment
*"<40: print 'terminal too narrow'"*.  Verified: **no call site outside `term.go`.**  R-5 is
therefore cheaper than it looks, and F-55 is Broadcaster's problem before it is Observer's.

### C-11 — A service-radius map is greenfield

`platform/geo` provides `HaversineKM`, `BearingDeg`, and `CompassIndex` and nothing else.  There is no
canvas, no braille rendering, and no plotting anywhere in the tree.  A TerminalMap port would be new
construction, not an integration.

## Other Considerations

- **This brief replaces the body of GitHub issue #10 on approval**, per the standing rule that the
  finalized brief is the issue description.  Bugs and documents continue to be logged as issues.
- **A Go port of Rust's TerminalMap is a stated future direction** for both visualization generally
  and the Broadcaster service-radius view specifically.  It is **not in this release's scope**, and
  the architecture should avoid foreclosing it.  C-11 says the ground is empty, which is the easy
  case: the constraint on PLAN is to keep the service radius a first-class value that a renderer
  could later consume, not to invent the renderer.
- **The mock of record** is `01-objectives/mock-broadcaster-v1.txt` — 73 lines, 70 of them exactly
  150 cells.  Adopted by ruling D-9.  It was retyped from the conversation rather than copied from a
  source, and that risk is recorded there rather than resolved.
- **The mock's masthead reads `v 0.14.0`.**  Cosmetic in a mock, and worth naming so it does not
  travel into the build.
- **The mock spells `•STANRARD•`** in every standard-card badge.  Read as `STANDARD`; confirm.
- **Carried follow-ups this release should absorb or explicitly decline:** F-55 (min size, now R-5),
  F-26 (a STOP ALL control, blocked on a key-binding decision that R-1 and OQ-3 must make anyway),
  F-27 (inter-card transitions, scripts written and pinned, wiring outstanding), F-49
  (`~/.watchpost/` as one cache root, which interacts with R-2.4's migration and should be decided
  with it, not after), and F-62 (below).
- **F-62 needs correcting rather than closing.**  Intake research reported it refuted, and the
  refutation is half right: the form has two rows, not one, and "All locations" is its own radio row
  (`setup_form.go:150-152`), so the default path is sound.  **But the failure it names is still
  reachable** — selecting *Within* and entering `0` renders `Within [0] mi` while `alertRadiusChoice`
  returns 0, which means All locations (`setup.go:683-693`, and the function's own docstring says so).
  **This was traced in code and not run**; DISCOVER should verify it against the binary.
- **A documentation citation does not resolve:** the 0.15.0 debrief lists
  `06_docs/build-methodology.md`, which lives at
  `06_docs/02_features/0.15.0-pre-broadcaster-ui-improvements/04-development/build-methodology.md`.

## HUM LEAD Rulings D-1..D-7 — recorded 2026-09-09

**All seven ruled at intake.**  Each is quoted, then read into the work.  Where a ruling defers a
decision to PLAN, that is recorded as the disposition rather than as an open question, so nothing
below is still waiting on an answer.

### D-1 (OQ-1) — Mode-switch safety

> *"Require STANDBY before swap."*

**RULED: STANDBY-first.**  Broadcaster must be in STANDBY before a swap to Observer is permitted.
**R-1.2 is now settled** and its alternative branch — a confirmation with a graceful sign-off — is
withdrawn from this release.

**What this buys.**  C-3 measured that no fade, drain, or graceful-stop primitive exists at any
layer, and that stopping is a deliberate hard cut.  STANDBY-first means 0.16.0 does not have to
invent one, and does not have to validate a fade it has no instrument for.  It also aligns with the
role model, where **MasterControl "declares ON AIR or STANDBY, and everyone complies, the Director
included"** (`multi-voice-support/03-architecture-design/role-model.md:88-91`).

**What it still owes.**  R-1.3 does not go away.  The swap is now *gated*, not *safe*: DISCOVER must
still establish what happens if the operator is in STANDBY while audio is genuinely in flight, and
M3 still counts every truncated word and ownerless audio path.

### D-2 (OQ-2) — Root architecture

> *"Router Approved (this is an implementation detail, can be vetted in PLAN)."*

**RULED: a router model, subject to PLAN vetting.**  A thin root model holds the two surfaces and
owns the swap; `Dashboard` stays the Observer-only model.

**Recorded with it, because PLAN must weigh it:** the counter-argument from intake stands.  Every
prior "second thing" in this codebase — Setup, Debug, Severe, RelayFault — was added as more fields
and more cases on the same `Dashboard`, and a router must deliberately share the key map, the
window-size dispatch, and the theme and background-colour handling that a single model gets for free
(C-8).  PLAN owns proving the router does not duplicate those three.

### D-3 (OQ-3) — Key bindings

> *"Make the keybindings routable, but we should probably change - we can finalize in PLAN."*

**RULED: bindings are rebindable, and the specific chords are PLAN's to finalize.**  `ctrl+b` and
`ctrl+o` are the working placeholders and are **not** the committed answer.

`term.Merge` already supports user overrides and rejects duplicate claims across the whole map
(`platform/term/term.go:120,159-162`), so "routable" is largely inherited rather than built.  PLAN
carries the chord decision, and C-9 is its evidence: `ctrl+b` is tmux's default prefix, and `ctrl+a`
— already taken here — is GNU screen's.

**This ruling also unblocks F-26**, the STOP ALL control, which has been waiting on a key-binding
decision since 0.14.0.  PLAN should settle both in one pass rather than twice.

### D-4 (OQ-4) — Minimum size

> *"We should look to make it support narrower than 150 if possible through dynamic resizing and
> breakpoints - this is why I originally though 130 would be good, but again we can vet in PLAN, but
> I am also okay with making 150 the minimum if needed."*

**RULED: pursue narrower than 150 through dynamic resizing and breakpoints; 150 is an acceptable
floor only if the responsive approach fails.**  The order of preference is explicit, and 150 is the
fallback rather than the target.

**R-5.3 is amended accordingly**: the mock's measured 150 x 73 is the *reference rendering*, not the
minimum.  The minimum is whatever the breakpoints can carry with the surface still operable.

**This ruling is cheaper than it looks, and C-10 is why.**  `platform/term` already defines
`Breakpoint`, `BreakpointFor` and `HeightCompact` with five width classes — including
`BreakTooNarrow` carrying the comment *"<40: print 'terminal too narrow'"* — and **nothing calls
any of it** (`term.go:60-87`, verified: no call site outside `term.go`).  The responsive vocabulary
this ruling asks for is written and unused.  PLAN's job is to wire it, decide which classes
Broadcaster supports, and decide what the surface sheds at each step down.

**The instrument obligation rides with it (FULL INST).**  A breakpoint set is a derived subject list
under INST-1: the size sweep behind M5 must walk the breakpoint enum rather than a hand-written list
of sizes, or it will go stale the first time a class is added.

### D-5 (OQ-5) — Operator authority over the Director

> *"This should be answered in the Director's charter - the Director serves the will of the operator.
> We have put some constraints on the human operator largely for SAFETY and LOGISTIC reasons, and
> some of the decisions are abstracted away (inter-card transitions) because it's the director's job
> to understand which inter-card scenarios require which transitions and which don't.  If this is
> insufficient, we can account for this as a design need in PLAN."*

**RULED: the charter governs, and it is sufficient on its face.**  Verified against the documents
rather than accepted on report — the ruling's substance is already written down, verbatim:

> **"Execute the will of the Human Operator."**  *"When the operator promotes, quashes or drops a
> card, the Director adjusts or removes the affected transitions itself.  The operator never manages
> transitions and never has to keep track of them."*
> — `multi-voice-support/03-architecture-design/role-model.md:62-76`

So the answer to "is the operator's edit authoritative" is **yes**, and the inter-card transition is
the Director's own additive act, deliberately abstracted away from the operator.

**The gap this ruling exposes, which DISCOVER must carry.**  The same section closes with the
present-tense truth: *"The transition RULES do not exist — a transition card is constructed in
exactly one place (the staleness notice) and nothing decides when a handoff is required.  Operator
edits do not exist at all."*  The charter answers the *authority* question completely.  It does not
answer the *durability* question C-1 raised — how long an operator's edit survives against a Director
that re-plans on the next arrival — and that remains a DISCOVER item, now correctly framed as a
design need rather than as an authority dispute.

Charter references for DISCOVER: `01-objectives/director-charter.md` (D-C-5 asks the sibling question
for Observer), `01-objectives/director-requirements.md`, `03-architecture-design/role-model.md`.

### D-6 (OQ-6) — The shared/unique settings split

> *"Recommendation approved - we can see the full set and rule on each in DISCOVER."*

**RULED: theme, units, clock and update-check are shared; locations, radius and recents are
Observer's.  The full field table is presented for per-field ruling at DISCOVER.**

The ambiguous entries intake could not classify, carried forward by name: the correspondent cast
(`Radio.Cast` / `Radio.Voices`), per-action key bindings (`Keys`), provider credentials
(`Providers`), the fire and seismic threshold blocks, and `TickerMuted`, which mirrors
`Radio.Tones` and becomes ambiguous the moment two surfaces can mute independently.

### D-7 (OQ-7) — Documentation folders

> *"Use the folders needed when the appropriate document is created."*

**RULED: create a folder when its first document is written, not in advance.**  `FULL DOCS` is
satisfied by the documents that exist having the right home, not by seven empty directories.

**Applied immediately:** intake scaffolded eight directories.  Seven held no document and are
removed, leaving `01-objectives/` for this brief, the locked problem statement and the mock.  The
rest are created as their first document lands.  (Git does not track empty directories, so none of
the seven would have been committed in any case — the removal keeps the working tree honest.)

## Directive note — `FULL INST`

The intake directive read `FULL TDD + IDD`.  Intake flagged that the repository had deliberately
retired the name "IDD" — `06_docs/quality-plan.md` records the rename and the reason: *"nothing here
drives development; every rule is applied to an instrument, and a reader told 'we use IDD not TDD'
would write no test first."*

**HUM LEAD ruling, 2026-09-09:**

> *"You are free to rewrite TDD+IDD as TDD+instrumentation_rules, or we can flag a new proposed term
> for A2DH upstream, probably something like FULL TDD; FULL INST; <-- that's my instinct on this."*

**ADOPTED: `FULL TDD; FULL INST`.**  The directive line at the head of this brief now reads that way,
and 0.16.0 is the first release to run under it.

**`FULL INST` means:** INST-1 derived subject lists, INST-2 silence as a distinct verdict, INST-3 a
surviving plant indicts the plant first, INST-4 answer the known before the unknown, INST-5 state the
blind spot in the instrument's own output.  It is a **dimension of TDD, never a replacement** — the
test still comes first; `FULL INST` governs the instrument that test becomes.

**Filed as an upstream A2DH proposal.**  `FULL INST` is proposed as a first-class li-A2DH directive
alongside `FULL TDD`, which is the destination `quality-plan.md` already named for these rules.
Tracked as an A2DH backlog item rather than a watchpost one; the watchpost usage in this release is
the evidence that proposal will cite.

## D-8 and D-9 — the last two rulings, 2026-09-09

### D-8 (X-1) — R-6 stays PROVISIONAL, and that is the intent

> *"PROVISIONAL is fine; since this is new UX - A LOT of things will be discovered and refined in
> UAT."*

**RULED: the seven mock-inferred requirements stay provisional by design, and are refined through
UAT rather than resolved before DISCOVER.**  This is a stronger statement than "not yet confirmed":
R-6 is not a gap in the brief waiting to be closed, it is the part of the brief that is *expected* to
move.

**What this changes about how the release runs.**

- **R-6 items may change without a scope change.**  A provisional requirement that shifts under UAT
  is the process working, not a deviation to justify.
- **UAT stops being a late checkpoint and becomes a design instrument.**  0.15.0 already showed this
  in the small — UAT found the two defects the gates could not, and the build methodology records
  *"UAT before REVIEW exit, two questions: regression and yield."*  For 0.16.0 the yield question
  carries more weight than the regression question, because the surface is new.
- **PLAN must not over-specify R-6.**  Locking mock-derived layout into the plan would convert a
  provisional item into a commitment the UAT ruling explicitly declines to make.  Tier B signatures
  and flow skeletons, not finished layout.
- **The locked problem statement is what does not move.**  R-6 flexes; the sentence in
  `problem-statement.md` and the five measures M1-M5 do not.  That is the anchor a UX release needs
  in order to let its requirements move without losing its scope.

### D-9 (X-2) — the transcription is adopted as the mock of record

> *"That's fine.  We can also create a .txt file specifically for it if needed."*

**RULED: the transcription stands, and the `.txt` already exists.**
`01-objectives/mock-broadcaster-v1.txt` was created at intake and committed at `745eab7` — 73 lines,
70 of them exactly 150 cells.  It is now the **mock of record** for 0.16.0.

**The consequence, stated plainly because adopting it silently would hide it.**  The file was retyped
from the conversation rather than copied from a source, so **no diff has ever excluded transcription
drift**.  Making it canonical does not remove that risk — it *fixes* it, by making any difference
from the original invisible from here on.  The Migration Fidelity rule exists for exactly this shape,
and the honest disposition is to name it rather than to claim a fidelity nobody measured.

**What that means in practice, and it is deliberately modest:**

- Geometry claims taken FROM the file are sound, because the file is what will be built against.
  The measured 150 x 73 stands, and D-4 already ruled it a reference rendering rather than a floor.
- A claim about what the ORIGINAL mock intended is only as good as the transcription.  Two known
  candidates are recorded in *Other Considerations*: the masthead's `v 0.14.0`, and `•STANRARD•`
  appearing in every standard-card badge where `STANDARD` is meant.
- **If the original file surfaces, it should be diffed against this one rather than replacing it**,
  so any drift is recorded as a finding instead of being quietly corrected.  Until then, no diff is
  owed and none is pending.

**A `v2` supersedes rather than overwrites.**  Later mocks land as `mock-broadcaster-v2.txt` and so
on, keeping the version this brief's requirements were read from readable next to the ones that
followed.

## D-10 — the mock's coordinates, ruled 2026-09-09

> *"Replace the specific GPS a placeholder while keeping \"Bonsall, CA\" the demo location PURELY IN
> THE MOCK; not that that location was super sensitive, but it was for illustration purposes.  That
> GPS is important because it will power the planned GO terminalMap visualization, so the real code
> will (obviously) need to ensure both the named location and gps coordinates harmonize, but again
> for the mock it's not needed."*

**RULED, and applied.**  `mock-broadcaster-v1.txt` now renders `TOWER GPS:  <lat>, <lon>`, and
`Bonsall, CA` is retained as the demo location.  The placeholder follows the mock's own idiom — it
already writes `<alert>`, `<location>`, `<declared>` and `<expires>`.

**The geometry is provably unchanged.**  The placeholder was padded to the width of the pair it
replaced, so every column to its right holds position.  Measured before and after: **74 lines, every
line the same cell width, maximum 150.**  Widths compared element by element and found identical, so
no layout claim in this brief is affected and D-9's measurements still stand.

**This closes F-66.**  The coordinate pair no longer appears anywhere in the tree — verified by
`git grep` returning nothing.  The town name stays, unchanged, and remains covered by the existing
demo-location ruling of 2026-09-08.

**The ruling also created a requirement, which is the more important half.**  The reason the
coordinates matter — that they will power the planned Go TerminalMap service-radius visualization,
and that the real code must keep the named location and the pair in agreement — is captured as
**R-7**, above.  It is a stated requirement, not a provisional one, and it is the cheap half of a
future the mock is exempt from.

## Completeness Check

```
PROJECT BRIEF — COMPLETENESS CHECK
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  [✓] Header              — Scope, type and name defined
  [✓] Directives          — LEVEL-1, SEV-0, seven phase directives; IDD flagged
  [✓] Summary / Intent    — What, why, who benefits, cost of doing nothing
  [✓] Requirements        — 7 families; R-6 provisional by ruling, R-7 stated by the HUM LEAD
  [✓] Metrics of Success  — 5, anti-solution hardened
  [✓] Tech Constraints    — 11, each measured against the tree at f29cc1f
  [✓] Considerations      — Issue #10, TerminalMap, mock provenance, 5 carried follow-ups

  [✓] Rulings             — D-1..D-10 recorded verbatim with dispositions
  [✓] Outstanding         — none.  X-1 and X-2 closed as D-8 and D-9.

  Required sections: 4/4 complete
  Overall: INTAKE CLOSED.  DISCOVER MAY OPEN.
           R-6 is PROVISIONAL BY RULING (D-8) and is refined through UAT.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```
