---
title: "0.16.0 DISCOVER — research wave 1 findings and cross-cutting synthesis"
date: 2026-09-09
phase: DISCOVER
sev: SEV-0
authority: HUM LEAD
status: "Wave 1 COMPLETE — all four areas reported and hand-verified.  Every claim carries a citation; claims that contradict a prior record are marked CORRECTS."
---

# Wave 1 — what the code says about the four architecture questions

**Method.** Four read-only agents ran in parallel against the tree at `906b70f`, each with an
enumerated question set, an evidence-citation requirement and a word budget.  **Every load-bearing
claim below was then re-verified by hand**, and two were found wrong or overstated.  Findings are
recorded with the correction, not silently fixed, because a research claim that decays invisibly is
the failure mode this release opened with (see the brief's C-2).

---

## 1. The schedule: what exists, what does not

### 1.1 The display seam ALREADY EXISTS, and it is the best news in the wave

`lineup.Publish` (`platform/lineup/director.go:176-188`) is emitted on every settle
(`director.go:583`) and **carries the whole `Lineup` by value**, with the reason written down: the
pump dispatches asynchronously, so a reader that fetched the Director's current lineup at effect time
would get whatever it had become by then, not what was published.

Its executor is deliberately empty (`app/executors.go:190-204`), and says so:

> *"NOBODY READS THE LINEUP YET, and the seam that pretended otherwise is gone (red team 2026-09-05).
> It was `func(lineup.Lineup) {}` — a no-op with a nil-guard around it, dispatched every second —
> which reads as a wired feature and is not one.  The Broadcaster surface that consumes a published
> lineup is [the next release]'s. … THE EFFECT STAYS.  It is architecture (PL-6)."*

**Implication.**  Broadcaster's display does not need a new seam invented for it.  It needs a
consumer written for one that is already emitted, already snapshot-correct, and already named for
this release.  A red team also already removed the fake version of it, so the trap is pre-cleared.

### 1.2 The main track has NO producer — a bigger gap than intake found

**Exactly one line in production code queues onto `MainTrack`:** `director.go:686`, inside
`readInstead`, and it queues the **stale-transition notice**.  Verified by grep across `app`,
`platform` and `modes`, excluding tests.

So the mock's ten-slot stack has no source.  Intake reported that the schedule's *write* side was
missing; wave 1 finds that for the main track, the *producer* is missing too.  Broadcaster needs
both: something that puts ordinary reads into the stack, and something that lets an operator reorder
them.

### 1.3 Operator-edit durability is a SMALLER problem than intake feared — CORRECTS the brief's OQ-5 framing

`Plan` runs only inside `onArrived` (`director.go:398-427`) and derives a **new, independent takeover
card** which is queued onto `AlertRail` alone.  `Lineup.Queue` only ever appends
(`lineup.go:117`); it never sorts or replaces.  **Nothing re-derives `MainTrack` ordering on a
weather fetch.**

**Therefore the fear that a re-planning Director silently overwrites an operator's edit is not
supported by the current code.**  Order is positional in a stored slice, and an append cannot
reshuffle what precedes it.  The brief's OQ-5 residue narrows accordingly: the question is not
"does a re-plan destroy the edit" but "what records that a human made it".

**Derived vs stored**, since only stored state can carry an operator's intent:

| Item | Stored or derived | Citation |
|---|---|---|
| `MainTrack` / `AlertRail` order | **Stored** (slice order) | `lineup.go:44-46,117` |
| `OnAir()` | Derived per call | `lineup.go:197-211` — *"DERIVED, NEVER STORED"* |
| `Next()` | Derived per call | `lineup.go:175-188` |
| `HasTakeover()` | Derived | `plan.go:410` |
| `Card.State` | Stored field | `card.go:306` |

### 1.4 Three hazards that ARE real

1. **`Lineup.Set` refuses to reorder, by design.**  It is the only mutator for a held card, and its
   comment is explicit: *"the schedule's order is the planner's decision and a state change must not
   reorder it"* (`lineup.go:213-215`).  A promote routed through `Set` would update fields, return no
   error, and leave read order unchanged — **a UI that lies about order rather than one that visibly
   fails.**  This is the most dangerous of the three precisely because it looks like it works.
2. **The fence is admission-only.**  `Fence.Admits` is checked once, in `candidates()` at plan time
   (`plan.go:298`), and never re-asked of an admitted card (`fence.go:96` — *"not a sort key"*).  A
   promote path bypasses it silently unless it deliberately re-checks.
3. **A second rail arrival is swallowed.**  `onArrived` discards the `Queue` error when the rail is
   occupied (`director.go:418`), correct today as "a repeated burst is not a second read", but
   indistinguishable from any other suppression once an operator is watching the rail.

### 1.5 The stale-drop rule was written FOR this scenario — forward design that paid

`platform/lineup/stale.go` drops any `Standby` card older than `StaleAfter = 15 * time.Minute`
(`stale.go:34`).  Its header is worth quoting in full because it anticipates 0.16.0 by name:

> *"OBSERVER CANNOT REACH THIS. … It becomes reachable in Broadcaster, where an operator can hold the
> rail.  It is implemented here anyway because the alternative is that Broadcaster inherits a latent
> stale-read on the SAFETY path and a listener is the one who finds it (PD-3)."*

**This is not a defect and should not be filed as one.**  It is a safety bound implemented ahead of
the surface that makes it reachable.  It leaves exactly one design question, which is DISCOVER's:
the listener is told *"That report is out of date and has been dropped."* (`stale.go:44`), and the
**operator is told nothing about which card went.**  An operator who promotes a card, holds the rail,
and loses it at fifteen minutes currently has no way to learn that their specific edit was discarded.

### 1.6 The mock draws a card type the model has no slot for

`Slot` has exactly four members — `LocationReport`, `SevereRead`, `BreakingAlert`, `Transition`
(`card.go:104-112`).  The mock's cards map cleanly onto these **except one**:
`WATCHPOST CREDITS READ` (mock line 55).

**And it may be an obligation rather than a decoration.**  `app/credits.go:16-20` says the credits
list *"is a licence obligation, not a courtesy"* because GeoNames and Open-Meteo are CC BY 4.0.
Whether broadcasting derived data over FRS/GMRS discharges that obligation on air is **a question for
the HUM LEAD, not an answer this document will give.**

---

## 2. Layout: how far below 150 columns the console can go

### 2.1 The column budget, measured

Every well-formed row sums to exactly 150.

| Element | Width | Fixed or elastic |
|---|---|---|
| Outer frame (both borders) | 2 | Fixed |
| Left rail (the LIVE / UP NEXT / BED / SCHEDULED lettering) | 4 | Fixed |
| Takeover panel | **67** | Fixed — constant in every row it appears |
| Gutter, takeover to card | 2 | Fixed |
| Card panel | **71**, or **66** where the scrollbar sits beside it | **Elastic** — the only element that changes width to absorb the scrollbar |
| Trailing gutter + scrollbar column | 4 | Fixed |

### 2.2 Height: the mock is nearly twice its own minimum

Fixed chrome runs from the header to the bed row plus the scrollbar's top arrow — **39 lines**.  The
scrolling viewport is the rest.  A minimum readable card is **4 lines**.

| | Lines |
|---|---|
| Fixed chrome | 39 |
| Scrollbar bottom arrow | 1 |
| One readable card | 4 |
| **Minimum height** | **44** |
| The mock as drawn | 73 |

**So 29 of the mock's lines exist to show more cards, not to work at all.**  D-4's instinct is
confirmed on the height axis: the floor is far below the mock.

### 2.3 Width: 130 is achievable, 100 genuinely breaks

| Width | Card panel left | Verdict |
|---|---|---|
| 150 | 71 | The mock as drawn |
| **130** | **52** | **Achievable.**  The visual scrollbar must become a text counter; long card titles truncate |
| 120 | 42 | Achievable with harder truncation |
| 100 | — | **Breaks.**  The takeover panel cannot go below roughly 50 without its body text wrapping onto more lines, which is a functional break, not a cosmetic one.  Side-by-side must give way to a stacked layout |

**D-4's preference is satisfiable.**  130 works for the skeleton, which is the number the HUM LEAD's
instinct named.

### 2.4 CORRECTS an intake finding: there are TWO breakpoint vocabularies, not one dead one

Intake reported that `platform/term` defines `Breakpoint`, `BreakpointFor` and `HeightCompact` and
that nothing calls them (`term.go:60-87`).  That is true, and it is **half the story**.

Observer *does* reflow — through its **own** scale: `radioBP` with `radioNarrow`/`radioMedium`/
`radioWide` at thresholds `radioMediumCols = 84` and `radioWideCols = 146`
(`modes/tty/radio_panel.go:122-134,145-155`).  These do not match `term.Breakpoint`'s 40/60/80/120 at
any boundary.

**So the platform defines a responsive vocabulary nobody uses, and the one surface that needed
responsiveness rolled its own.**  That is the F-54 shape — an operation implemented twice — and
**Broadcaster would be the third implementation unless DISCOVER settles it.**  This is now a
requirement-level question rather than a wiring task.

**Observer's reflow precedent, which Broadcaster should follow rather than reinvent:** shed labels
before shedding controls; *remove* a control rather than grey it out once its feature is gone
(`radio_panel.go:164,490-492` — *"a control that toggles nothing can never be drawn"*); reflow and
centre what remains; scale one proportional bar between a floor and a cap; and keep width and height
compaction on separate axes, never conflated.

---

## 3. Key bindings: one flat namespace that cannot express two surfaces

### 3.1 The inventory

**28 actions**, derived from `defaultKeyMap()` (`modes/tty/dashboard.go:306-340`).  Three are bound
to more than one key: quit (`q`, `ctrl+c`), severe (`w`, `W`, `ctrl+s`), volume-up (`+`, `=`).

### 3.2 The namespace cannot be scoped, and the mock already collides

`KeyMap` is `map[Action]Binding` with **no surface dimension** (`platform/term/term.go:113`), and
`Merge`'s duplicate check runs over one `seen` set covering the whole merged map
(`term.go:144-164`), returning a hard error naming both claimants.

**The mock's own key row already collides with Observer on four keys** — `s` Settings, `a` About,
`S` Status, and `A`, which Observer binds to alert-details and the mock uses for a takeover card.

Merge's other behaviours, each cited: an override naming an unknown action is **silently dropped**
(`term.go:135-139`); an unparseable key string is **accepted** and simply never matches a real
keypress, failing silently at runtime rather than at build time (`term.go:153`).

### 3.3 CORRECTS the agent: the multiplexer risk is real, the flow-control risk is not

The research reported that Observer's `ctrl+s` collides with terminal flow control (XOFF) and
`ctrl+d` with shell end-of-file.  **Both are refuted for a running full-screen application**, and the
distinction is the useful part:

- **Line-discipline chords are neutralised by raw mode.**  Bubble Tea calls `term.MakeRaw` on start
  (`bubbletea/v2@v2.0.9/tty_unix.go:19`), and `MakeRaw` clears `IXON` along with `ICRNL`, `ISTRIP`
  and the rest (`charmbracelet/x/term@v0.2.2/term_unix.go:29`).  `ctrl+s` and `ctrl+d` therefore
  reach the application normally.
- **Multiplexer prefix chords are NOT**, because tmux and GNU screen intercept them in a different
  process, upstream of the application, where raw mode has no bearing.  `ctrl+a` — which Observer
  **ships today** — is screen's default prefix, and `ctrl+b` is tmux's.

**The in-repo comment conflates the two mechanisms** (`dashboard.go:313-316` warns that `ctrl+d` is
"EOF in many terminals and some multiplexers claim it first"), which is precisely why the wrong one
was flagged.  Two mechanisms, two different answers.

---

## 4. The root seam: what a router must actually own

### 4.1 The program boundary is thinner than expected, which favours the router

**`tea.NewProgram` is called with NO options at either call site** (`app/dashboard.go:350,375`).
Full-screen mode is set per frame inside `View` (`v.AltScreen = true`, `modes/tty/view.go:43`) and
the background colour is requested by `Init` returning `tea.RequestBackgroundColor`
(`modes/tty/dashboard.go:524`).  **Both are model behaviour, not program configuration**, so a router
inherits them simply by delegating `Init` and `View`.  That removes the most obvious objection to
D-2's ruling.

### 4.2 Only THREE program-scoped messages are handled, and five are not referenced at all

Verified by counting production references:

| Message | Production references | Handled at |
|---|---|---|
| `tea.KeyPressMsg` | 13 | `dashboard.go:588-589` |
| `tea.WindowSizeMsg` | 1 | `dashboard.go:562-564` |
| `tea.BackgroundColorMsg` | 1 | `dashboard.go:573-575` |
| `tea.FocusMsg` · `tea.BlurMsg` | **0** | nowhere |
| `tea.SuspendMsg` · `tea.ResumeMsg` | **0** | nowhere |
| `tea.ColorProfileMsg` | **0** | nowhere |

**Read this two ways, and both matter.**  The router's fan-out surface is small — three messages, not
a dozen — which makes D-2 cheap.  But the five unhandled ones are a standing gap that a second
surface makes worse rather than creates: a suspend/resume pair matters more for an operator console
that is *supposed to keep broadcasting* than for a weather dashboard, and focus/blur is how a surface
learns it is no longer being watched.

Every custom message in the tree (`SnapshotMsg`, `RadioStatusMsg`, `TickerMsg`, `SevereMsg` and the
rest) is Observer-shaped data defined in `modes/tty` — **none is program-scoped**, so none needs
fanning out.

### 4.3 What is already shared for free, and what is not

| Subsystem | Ownership | Shareable by two surfaces? |
|---|---|---|
| Theme and tokens | **Package-level global** in `platform/render` (`themes.go:24-34,223`) | **Already shared, for free** — no router work at all |
| `darkBG` | A **field** on `Dashboard` (`dashboard.go:382`), set only from `BackgroundColorMsg` | **No** — the router must fan the one message to both, or each surface re-requests it |
| `term.KeyMap` | Merged once and copied into `Dashboard` (`dashboard.go:422,426`) | Yes — a value, immutable after merge |
| `memo` / `mmemo` | Per-model pointers (`dashboard.go:405-406`), keyed on Observer's own render keys | **No** — Observer-specific by construction |
| Ticker, severe deck, radio deck | Owned by `livePipelines`, **outside** `Dashboard` | Yes — they already write in via `p.Send` |
| `mastercontrol` | One instance, "THE ONE OWNER OF EACH OUTPUT (D-1)" (`mastercontrol.go:12`) | **Must be shared, never duplicated** |

### 4.4 The real cost of the router, stated plainly

`mastercontrol` is constructed with `send func(tea.Msg)` bound once to `p.Send`
(`app/dashboard.go:64`), and `radioDeck` holds a single `p *tea.Program` (`app/radio.go:35`).
Introducing a router does not change how many programs exist — still one — **but every closure and
struct in `app/` that captures `p` or `p.Send` today implicitly assumes the one model is
`Dashboard`.**  The router shape requires auditing all of those senders — mastercontrol, the ticker,
the severe deck, the radio deck, the event reader — to confirm that a message sent to the *program*
reaches the *router*, which then routes to the active surface **without the sender ever knowing which
surface is live**.

**That audit is the true price of D-2, and it is a PLAN task with a name.**

### 4.5 Where the STANDBY gate belongs, and why that seam

The D-1 precondition must be checked **in the router's own `Update`, at the single branch that turns
a swap request into a model change** — never inside Broadcaster's key handler.

The reasoning is this codebase's own history rather than taste.  If the router delegates the key to
the active surface first and lets Broadcaster decide, then every future path that could request a
swap — a programmatic message, a menu action — must re-implement the same guard.  That is precisely
the *"two carriers of one rule"* failure `mastercontrol.go:14-18` documents and D-1 exists to
prevent.  One branch in the router is the seam a bypass would have to go *through* to avoid.

The router should read Broadcaster's state through a small exported predicate rather than reaching
into its fields.

**An existing confirmation pattern is present and is NOT to be wired here.**  `view.go:39,52-58`
composites a `confirmOverlay` on top of the active modal, gated by a bool
(`debug.go:51,216-226`), with the same shape used for watchlist removal.  It is the template if a
confirmation is ever wanted — but **D-1 withdrew the confirmation branch for 0.16.0**, so it informs
shape only.

---

## 4. Cross-cutting synthesis

Required after every research wave.  Composition, convergence, contradiction, and what it means for
the next wave.

1. **Composition — the read path is nearly free and the write path is nearly absent.**  §1.1's
   `Publish` seam plus §1.2's missing producer compose into a sharp scope statement: Broadcaster can
   *display* the schedule almost immediately, and can *change* it only after new events, a new
   producer and a reorder-capable mutator all exist.  **Demo-able early, correct late** — which is a
   sequencing risk, because a console that shows a stack it cannot change will look nearly done.
2. **Composition — D-4 is satisfiable, but it lands on an unsettled duplication.**  §2.3 proves 130
   works; §2.4 shows there are already two breakpoint vocabularies.  Wiring the platform one, adopting
   Observer's, or writing a third are three different answers, and PLAN cannot pick one without a
   requirement.
3. **Convergence — three independent areas each say the same thing: the seam exists, the consumer
   does not.**  `Publish` is emitted and unread (§1.1); `term.Breakpoint` is defined and uncalled
   (§2.4); `Origin.FromOperator` is declared and never constructed (brief C-1).  **Three artefacts
   built for this release and left deliberately unwired.**  That is a strong signal the foundation
   claim was substantially honest, and it also means "it exists" must never be read as "it works" —
   every one of the three needs its first real consumer, and none has ever been exercised.
4. **Contradiction resolved — operator-edit durability.**  Intake framed OQ-5 as a durability risk
   (§1.3 refutes it) and the HUM LEAD ruled the charter governs.  Both are now consistent: nothing
   re-derives order, so the surviving question is *representation* — what marks a card as
   operator-willed — not *survival*.
5. **Contradiction found — the mock exceeds the model.**  §1.6's credits card has no `Slot`, and it
   carries a possible licence obligation.  This must be ruled before FR enumeration closes.
6. **Risk signal update — M3 (unsafe mode switches) is unchanged and still the top risk.**  Nothing
   in wave 1 touched the audio path; the brief's C-3 finding stands, and D-1's STANDBY-first ruling
   remains the mitigation.
7. **Risk signal update — a NEW risk, and it is a UI-integrity one.**  §1.4's `Set` hazard produces a
   console that *displays* an order the reader does not follow.  Under this release's instrument
   rules that demands a gate asserting the rendered order equals `Next()`'s walk, not merely that a
   promote call returned without error.
8. **Composition — D-2 is cheaper at the top and dearer underneath than it looked.**  §4.1 and §4.2
   make the router's own surface small: no program options to reproduce, three messages to fan out.
   §4.4 relocates the cost to the senders in `app/` that captured `p.Send` assuming one model.  **The
   ruling stands; the estimate moves.**
9. **Convergence — the codebase keeps answering "one owner" and it answers it again here.**  §4.5's
   placement of the STANDBY gate derives from the same rule as `mastercontrol`'s single ownership and
   the single config writer.  D-1 and D-2 are the same principle applied at two levels, which is a
   reason to trust both.
10. **A NEW risk that is not Broadcaster's fault but becomes Broadcaster's problem.**  §4.2's five
    unhandled program messages include suspend/resume, which matters far more for a console that is
    meant to keep broadcasting than for a dashboard.  Filing this as a follow-up rather than scope.
11. **Implication for wave 2.**  Four questions are now well-posed:  (a) what produces main-track
    cards, since nothing does;  (b) which breakpoint vocabulary wins, given there are already two;
    (c) whether the credits read is a licence obligation;  (d) the sender audit of §4.4, which is the
    one item that could still move D-2's estimate materially.  Items (b) and (c) are HUM LEAD
    rulings; (a) and (d) are research.

---

## 5. Evidence notes

- **D-9 drift candidates now have specifics.**  Two mock lines are not 150 cells: **line 9** (the
  GAIN row, 130) and **line 73** (the closing scrollbar arrow, 145).  Both are lines whose content
  ends early and would have carried trailing spaces in the original, which is exactly what is lost in
  transit.  They are almost certainly transcription artefacts rather than design, and they are the
  first thing to check if the source file ever surfaces.
- **F-62 sharpened.**  `setup_esc_save_test.go:204-223` covers "radius changes", "radius unchanged"
  and "radius back to All", and deliberately covers two analogous invalid-value cases — an empty
  language and a zero dwell — each asserting the write does not happen.  **There is no equivalent
  case for a filtered radius of zero**, which is F-62 exactly.  The encoding is also never explained
  to a user anywhere in the README or the form, so the failure is silent by construction.
