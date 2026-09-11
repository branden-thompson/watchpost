# D-81 — THE LANE DECIDES WHO READS, AND THE PROGRAMME IS NOT A NARRATION

**HUM LEAD, UAT 2026-09-11:** *"Shift + Enter Audio in Broadcaster Does not work … No audio …
The Schedule does its 'churn' bug again … I wonder if it's because the COMPOSER is actually NOT
fulfilling its data join … So at this point, I cannot UAT the card flow and lineup effectively."*

**Not a new ruling.  Two existing ones, finally built** — and the record already held both, which is
the third time this release that drawing the answer out of the documents beat deriving it from the
code.

---

## What was actually wrong

The composer was fine.  `deck.segments` fetches obs and alerts on demand for whatever ref it is
handed, so a pool location has data whether or not a pipeline ever asked for it.  That was checked
first, because it was the HUM LEAD's own hypothesis and the cheapest one to falsify.

The cause was one line in `app/executors.go`:

```go
func onTheRail(s lineup.Slot) bool { return s == lineup.BreakingAlert }
// speak: "no reader for this slot: the arbiter reads the rail, and the rail only"
```

Every location report was **admitted → built → declined**, which fails the card, discards it and
benches its location for five minutes under D-67's cool-off.  With the pool at 25 and the console
drawing ten slots, the whole pool benches in one pass and every slot reads `waiting for the line-up`.
**That is the churn, exactly, and the silence with it.**

**The refusal was deliberate and correct.**  D-33 (HUM LEAD, 2026-09-09) ruled that *a chosen read
REPLACES the bed rather than speaking over it*, so the programme is not a narration and has no
business on the narration path.  What was never built is the replacement.

---

## The shape was already ruled, twice

**BD-9** (`multi-voice-support/04-development/director-build-log.md:809`) — a re-required read:

> *"A report's `speak` is **the engine Source adapter** — which **IS** the main-track absorb."*

**D-33**, from the operator's side: the read replaces the bed.

So the reader is a **source swap on the broadcast engine** — a `synth.Source` over the card's words,
started with `StartSource` — and not a narration through the arbiter.  No ruling was needed; the
question I was about to put to the HUM LEAD was already answered in the file the session is required
to re-read.

### What the ruled shape buys, and why it is not merely one option of two

`StartSource` calls `setLive(false)`, and `giveWayLocked` re-reads the source kind **every 50 ms**:

- a **relay** DIPS — live radio plays on under the alert;
- a **rendered report** HOLDS — it has nowhere to be, so dipping would lose its words for good.

Which is **D-24 exactly**: *"we can PAUSE the read, let the alert rail drain … then resume the read at
normal volume."*  A reader built on the clip path would have had to reimplement that rule, and
reimplementing it is how the wrong duck decision was made the first time (`app/radio.go:666`:
*"Asking the deck's mode here fixed an answer the audio could outlive."*).

### And the bed needs no hand-back

My first account of this — written before the code — said the bed returns when the card finishes.
**That was wrong.**  Main and bed are mutually exclusive (D-11, FR-4.2) and the engine has one
source, and `advances(MainTrack)` already requires `!bed.carries`.  While the programme is on the
air there is nothing underneath it to restore.

---

## D-81 — the LANE decides who reads, not the SLOT

The one genuinely new decision, and it is forced by a transition.

`Speak` now carries `Track`.  The rail reads through the arbiter; everything else goes to the
broadcast engine.  **The slot cannot answer this question**: a transition's slot says only that the
Director minted it, and

- a **hand-back after a takeover** is read on the RAIL, in the breaking correspondent's voice, with
  the duck still down;
- a **stale notice** standing in for a dropped report is the PROGRAMME.

One reader for both would say the wrong thing in the wrong voice.

`Track` is carried ON THE EFFECT for BD-8's reason, like `Slot` and `Refs`: the publish for the same
step runs concurrently with the dispatch, so an executor that asked the published lineup which track
held the card would race it.

### The guard that comes with it

`Track`'s zero value is `MainTrack`, so an effect built without one routes to the engine **by
default** — and a rail card down that path loses its attention tone (the engine plays words; the tone
is the arbiter's to sound) and its per-alert callouts.  A tornado warning, delivered as the weather.

So `broadcast` refuses a rail SLOT by name.  It is the direction a mistake would actually go, it is
constructible, and it is therefore testable — which is why there is no mirrored check on the rail's
side.

---

## The half that had to come with it: STANDBY stops the words

Found by asking what happens to a read already going out when the operator presses GO TO STANDBY.

`silenceTheProgramme` on the Director's side takes the card off the air — but **a read that has
started is a worker blocking on the engine**, not something a schedule can recall.  Without a second
half, the console says the station is off and the station talks to the end of the report.

The same hole the other way: moving back to Observer must leave things *"like if Observer was first
opened"* (HUM LEAD, 2026-09-10), and Observer first opened is **silent**.

**MasterControl silences it**, which is its role stated exactly — and the `carry` field's own comment
had already written the argument for the bed: *"a Director holding every card still leaves a relay
playing, and that is not dead air."*  A main-track card is the same sentence with a different
subject.

**It CANCELS; it never halts.**  `engine.Halt` waits for the audio goroutine, and that goroutine is
calling back into the program — so a halt reached from `Router.Update` sends to a loop that cannot
receive.  That is **D-79**, and it is what cost the HUM LEAD his terminal twice on 2026-09-11.
`stopRead` returns at once; the worker already waiting on the read does the halting, on a goroutine
that is allowed to block.

---

## What this does NOT fix, and the HUM LEAD needs to rule on it

**A breaking alert now waits behind a report.**  `Lineup.OnAir` holds the invariant *"at most one
card holds the air"*, across both tracks — so a rail card cannot take the air while a main-track card
is on it.  Pre-emption is not implemented anywhere.

That is not new and it is not accidental: FR-5.6 records it as *"a fourth condition the `Power` enum
does not yet express"*, and the required reading says in as many words that it is **P5's**.  But it
was INERT until now, because a main-track card left the air instantly (declined).  With the reader
built, the wait is real and it is as long as a location report.

**It is a safety question and the answer is a HUM LEAD ruling.**  Two shapes:

| | What the listener hears | Cost |
|---|---|---|
| **Pause and resume** (D-24, DR-16, S-4) | the report stops mid-sentence, the rail drains, the report resumes where it stopped | the fourth power condition, and the engine already supports the hold |
| **Drop and re-read** | the report stops, the rail drains, the report starts again from the top | cheaper; the listener hears a minute of the same report twice |

D-24 already says pause-and-resume, three times.  Raised here because the cost is now visible and the
alternative is cheap, not because the ruling is in doubt.

---

## Recorded, not resolved

- **G-7 stands.** A card carries `Text`; `lineup.Part` has nowhere to put a Role, a self-introduction
  or a pause, and DR-1 keeps the domain out of the schedule.  So the main track reads in the
  station's own voice, where the rotation would have used per-report correspondents.  **What cannot
  differ is the words** — the card is composed once, at standby, and recomposing at read time would
  be a second Composer saying something other than what the operator has been reading in the slot.
  Filed as **F-92**.
- **F-71 is unchanged.** A main-track card's exit still clears the band.  It is still benign for the
  reason its own comment gives — one card holds the air at a time, and `cue` has one writer — and the
  reader does not make it live.
