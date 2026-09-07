# The role model — MVS-D-77

**HUM LEAD, 2026-09-04.** A correction to `director-architecture.md`, raised by a reachability audit
and settled by the Broadcaster mock. It supersedes the parts of DR-3 and DR-11 named below.

## Why this exists

Three findings arrived together, and they turned out to be one finding.

1. **A reachability audit** found the Director hears only six events in production — `Powered` ×2,
   `Tuned`, `Programme`, `Ended`, `Tick` — all about the bed. `Arrived` has no production sender, so
   the card lifecycle is unreachable. (An earlier version of this audit overstated the unreachable
   surface at ~1,100 lines; `lineup.Plan` **is** live, called from `app/ticker.go:330` on every real
   takeover, so planning is exercised. The unreachable half is scheduling and execution, ~500–600
   lines.)

2. **The effect set cannot express the takeover's pacing.** MVS-D-72 — ruled after the HUM LEAD heard
   a real alert — specifies `tone · 2 s · header · 1 s · alert · 1 s · … · 2 s · tail`, plus the
   render-overlap that removes ~1 s of dead air before every line. `Speak{ID, Slot, Text}` carries
   none of it, and the word "tone" appears nowhere in the architecture. Retiring `ticker.railBurst`
   today would ship no tone, no pauses, a dead gap before every line and no tail — the exact defect
   the HUM LEAD described as *"feels like something is 'broken' if I were just listening"*.

3. **The card model was wrong.** The code proposes one card per alert; the Broadcaster mock draws a
   burst as ONE card whose content is the whole script, addressed by the operator as one slot.

The third explains the second. Pacing had no owner because it falls *between* cards in a model where
a burst is many cards, and *inside* a card in the model the UI actually needs.

## The five roles

### 1. Card Producer — decides that a card should exist

Owns: what cards the schedule should contain, and when. Reads the incoming data, the listener's
settings and the app's configuration; knows the content *types* (a burst is due, a location report is
due, the credits are due).

Does **not** own: what the card says. A produced card carries its subject and headline and no words.

Today: `lineup.Plan` produces bursts — selection, the Max bound, the diverted count. Nothing produces
location reports or credits; the rotation does that implicitly.

### 2. Card Composer — decides what the card says

Owns: fulfilling a proposed card. Fetches the data, fills the script's placeholders with the freshest
values, and returns the finished script.

Does **not** own: arrangement, and **not** the UI. It returns its text to the Director; the Director
publishes. One publisher for the panel, or there is a window in which two writers disagree about what
is on screen — the stale-band defect of T3.5 in a new place.

**Producer and Composer are separate, and the separation is TEMPORAL rather than stylistic.** DR-7
holds that a card is proposed with no words and its text materialises at standby; PD-2 measured the
cost (1.03 s cold, 1–3 ms warm); PD-3 exists solely to police the gap between the two moments. One
role composing at proposal time would make every card stale by the time it aired, which is the
failure PD-3 was written against.

Today: the `BuildCard` executor — `radioDeck.segments()` plus `synth.Composer` — composes **location
reports only**. No composer exists for a takeover card; `ticker.railBurst` assembles one inline from
`burstHead()`, `breakingLine()` and `burstClosingLine()`.

### 3. The Director — arrangement, and the operator's will

Owns: the order. Takes produced cards and decides arrangement, priority and scheduling, and publishes
the whole schedule so the Broadcaster can hydrate its display.

Its ONE additive act is the **inter-card transition** — "we now return to our regularly scheduled
programming" — because only the Director knows that two adjacent cards came from different composers
and need a handoff between them.

**"Execute the will of the Human Operator."** When the operator promotes, quashes or drops a card,
the Director adjusts or removes the affected transitions itself. The operator never manages
transitions and never has to keep track of them.

Today: `platform/lineup.Director` owns arrangement. The transition RULES do not exist — a transition
card is constructed in exactly one place (the staleness notice) and nothing decides when a handoff is
required. Operator edits do not exist at all.

### 4. The Reader (the executor) — reads what it is handed

Owns: performing the LIVE card. Follows the script it was given, including the tone and the pauses.

The analogy is the correspondent on camera: they read the story in front of them, someone hands them
a breaking alert, they read that. They do not decide the running order.

Today: `executors.speak` plus the narration arbiter. **It has no pacing** — `s.line(text)` renders
then plays, in series.

### 5. MasterControl — owns the air

Owns: the mechanics of what is and is not broadcast, so nothing collides. And it is the **clock
keeper**: it declares ON AIR or STANDBY, and *everyone complies, the Director included.*

STANDBY pauses the broadcast schedule. Alerts continue to be produced and handled **visually** while
on standby — the Director must keep processing them, because they can grow stale, and a stale card is
worthless the moment MasterControl says "we're back ON AIR".

Today: `app/mastercontrol` owns the band and the duck. It does **not** own ON AIR / STANDBY.

## The card model correction

**A burst is ONE card.** The tone category, the header, the per-alert lines and the divert tail are
*intra-card content*, owned by the Producer and Composer. The Broadcaster mock draws it this way —
one panel, one slot number — and the operator's DROP / DELAY / PROMOTE therefore act on the burst, not
on one alert inside it.

This supersedes:

- **DR-3's** "items ... read first" where it implies per-alert admission to the rail.
- **DR-11's** Max-and-divert as the *Director's* admission bound. The rule is unchanged — emergency
  orders lead, the remainder fills the budget, the rest is diverted and counted — but it belongs to
  the **Producer**, and its result is written into the card's own script ("and `<diverted>` events").
- **The slot registry**: `BurstHead` and `DivertNotice` retire from being slots. They are parts of a
  takeover card's script.

**This is a smaller change than it sounds.** `lineup.Plan`'s per-alert cards are already vestigial:
`ticker.go:335-345` unwraps them straight back into events, using `Plan` purely as a selector. The
composer's pieces already exist as functions. What is missing is the ROLE that owns them.

## The script-carrying Speak

A takeover card's content is not a flat string. `Speak` must carry the script's PARTS so the Reader
can pace them per MVS-D-72. Inferring the breaks by splitting text works until an alert contains a
newline, and the Broadcaster displays those same parts as distinct lines (see the mock's `[T]` panel).

This is a deliberate change to a closed effect set (PL-6): the set is the architecture's interface,
and it grows here on purpose, once, rather than by accident later.

## STOP is not STANDBY

Observer's STOP and Broadcaster's STANDBY are one flag today and need opposite behaviour.

| | Observer STOP | Broadcaster STANDBY |
|---|---|---|
| Meaning | "do not play me a programme" | "we are off air — dead air" |
| The programme | stops | stops |
| **Hazards** | **still read** | **do not read** |
| Alerts meanwhile | — | produced and shown, never aired; may go stale |

Today `advances(AlertRail)` returns true unconditionally, and the reasoning is sound for Observer:
*"stopping the radio stops the PROGRAMME, not the hazards; mute is the control for 'do not speak to
me'."* Under one flag a Broadcaster on STANDBY would put a tornado warning to air.

They must be two states. The T3.3 staleness table's row 5 passed precisely because it asked the
Observer question.

## STANDBY — MVS-D-78

**HUM LEAD, 2026-09-05.** One meaning in both UIs, at two layers.

**The argument that settled it.** Observer and Broadcaster both produce one AUDIO OUT and one visual
channel; the only difference is who is listening. Observer's "don't talk to me" and Broadcaster's
"go to STANDBY / DEAD AIR" are therefore the same state, and the TV analogy governs the visual half:
the picture stays live and the subtitles keep advancing — only the sound stops.

**Where the analogy deliberately ends.** A TV does not PAUSE when muted; the programme runs on and
you miss what was said. A weather radio may not do that, because the thing you would miss is the
thing the product exists to say. So the audio schedule HOLDS what it has not yet spoken.

	State      | Visual   | Audio out | Fresh alerts
	-----------+----------+-----------+--------------------------------------
	Running    | updates  | reads     | consumed as read
	Stopped    | updates  | programme silent, HAZARDS STILL READ | consumed
	Standby    | updates  | SILENT    | STAY NEW — read on resume; stale ones expire
	[M] before | updates  | silent    | consumed silently  ← the defect this fixes

**The architecture already worked this way**, which is why the change is small. In one ticker cycle
the severe index and the ticker tape are sent BEFORE the takeover and unconditionally, and the
comment at the seen-mark says the rest: *"the takeover marks each as it reads it, and whatever it
never reached stays new deliberately."* Standby is simply not starting the takeover.

**One declaration, two compliers, three roles untouched.** MasterControl DECLARES ON AIR / STANDBY
and everyone complies. Two entities must ACT on it, and neither alone suffices:

- **The Director** holds the schedule. Gating only the audio would let cards be marked read and
  consumed silently — the `[M]` defect one layer down.
- **MasterControl** silences the bed. The bed is not a card; it is a continuously playing relay or
  synth, and the Director only says when to TUNE. A Director holding every card still leaves a relay
  playing, which is not dead air.

The **Producer** and **Composer** carry on — alerts keep arriving as cards and appear in the queue,
exactly as the Broadcaster mock draws them. The **Reader** never needs to know: a held schedule hands
it nothing.

**What may still leave the schedule while on standby**, and nothing else (HUM LEAD): an **operator
decision** — drop, promote, quash — and **staleness**, where reading the card on resumption would be
*harmful* because it no longer reflects reality. That second is a sharper statement of PD-3 than "the
observation is old": the window exists to stop the station asserting something untrue, not merely
something stale.

**The safety trade, stated plainly.** Under standby, hazards do NOT read. That is a change from
Stopped, which deliberately lets them through. Nothing is hidden — the `[w]` window and the ticker
show everything throughout — but the audio channel stays quiet until the listener says otherwise.

## A consequence the model has, found at T3.10a (2026-09-05)

**The planner's ordering must stay visible.** The first attempt collapsed `Plan` to a single card and
broke 48 tests — most of them pinning SAFETY rules about the read order: the ladder, emergency orders
leading and spending the budget, the fence, the freshness boundary, tie-breaking, and the divert count
that is spoken aloud. Those rules still exist under the one-card model; what changed is only what the
SCHEDULE holds.

So `Burst` carries both: `Cards` is the Producer's ORDERING (one per alert, the Composer's input, and
what every ordering rule is asserted against) and `Takeover` is the ONE card the rail holds. Collapsing
them would have left forty pins with nothing to look at, which is a real loss of measurement dressed
up as a simplification.

**And DR-7's one-ahead build no longer applies WITHIN a burst.** The standby pre-build was designed
against per-alert cards: card *n* reads while card *n+1* is built. A burst is one card now, so there
is no next card inside it — the intra-burst equivalent is the READER's overlap, which renders each
part during the one before it (T3.8, MVS-D-72). DR-7 still governs the gap BETWEEN cards: a takeover
reads while the location report behind it is built.

That is coherent, but roughly thirty Director-level tests encode the per-alert version, and several
pin DR-7's one-ahead using the alerts of one burst as the cards. Those need RE-AUTHORING against the
new model — a burst reading while a following card builds — rather than editing. Re-authoring a
safety pin is not a rename, and it is the part of T3.10a that wants care rather than speed.

## What this does not change

The Director stays pure. Approach C, the closed effect set as a principle, the card state machine,
DR-7's proposal-without-words, PD-3's staleness window, DR-24's paired release, DR-21's graded
surfacing — all stand. The roles name what already exists and give the two unnamed ones a home.
