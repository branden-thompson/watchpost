# D-74 — TWO PROGRAMMES, TWO POWERS, ONE AIR

**Ruled by the HUM LEAD, 2026-09-10**, after the diagrams:

> Two powers.  Stay silent; while air was taken — it was a result of the Operator action
> (ctrl+b -> ctrl+o) so I am okay with it "restoring" state, like if Observer was first opened (with
> reasonable performance consideration in mind, we don't want to slam APIs by rapid user switching).

and, the requirement it serves:

> If I'm in the Broadcaster UI, then the Observer Mode must be Silent.

---

## The word that had to change first

The HUM LEAD caught it in the diagram: *"is 'Listener' == 'Radio Listener' within Broadcaster's
configured service area, or the active listener mode?"*

It meant the person at the keyboard, which is how the whole codebase used it — and that was fine while
there was one persona.  In a BROADCAST product the listener is the AUDIENCE.  So the air-ownership
code uses the station's own vocabulary:

| | |
|---|---|
| **MONITOR** | what the operator listens to OFF AIR — Observer's rotation |
| **PROGRAMME** | what the station puts OUT — the console's line-up |

and "listener" goes back to meaning the audience, where it belongs.

## What was actually wrong

**One `power` field served two programmes.**  `advances(MainTrack)` decided whether the station's
line-up advanced AND whether the operator's own listening rotated, so a running station could produce
both at once — the line-up they scheduled, and the watchlist they had been listening to underneath it.

**And three things declared it**: Observer's tune, Observer's stop, and the console.  A tune is the
operator LISTENING; it put their station ON AIR, which is why `ctrl+o` was refused after a round trip
(D-69's root, closed here).

## The spike that cost a build and saved a worse one

I proposed wiring the air to `lineup.CutOver` / `bed.carries` — ruled at FR-4.2/D-11/D-32, no
production emitter, apparently a designed seam waiting to be connected.

**It is not the air owner.**  `bed.carries` means the operator PARKED the station on the bed, and the
bed's own rotation pauses under it.  Wiring it would have frozen the watchlist rotation — mM3's defect
arriving by a new road.  The spike is why that is a sentence here instead of a UAT report.

## The model

```
power    the STATION's       Stopped | Running | OffAir     declared by the console
monitor  the OPERATOR's      running or not                 declared by a tune or a stop
air      which may perform   MONITOR | PROGRAMME            declared by the SURFACE
```

- `advances(MainTrack)` = `power == Running && air == AirProgramme && !bed.carries`
- `advancesMonitor()` = `monitor && air == AirMonitor && !bed.carries`
- the ALERT RAIL is exempt from both, unchanged: hazards read in either mode (its FENCE follows the
  surface — D-73)

**They are mutually exclusive by construction**, which is the property `TestOnlyOneProgrammeCanEverAdvance`
exists to state.

**`Monitored` is a bool, not a `Power`.**  A monitor has two states; a `Power` has three, and the
third — dead air — is a thing a STATION does.  Typed as a Power it would carry a state it can never
legitimately be in.

## Going back does not resume, and that is a performance ruling as much as a UX one

The monitor comes back *"like if Observer was first opened"*: the operator presses play.  **That is
also what keeps a rapid `ctrl+b` / `ctrl+o` flip from re-resolving a relay and re-fetching its
products on every swap** — the HUM LEAD's "don't slam APIs" condition, satisfied by the same
behaviour rather than by a throttle.

## Two plants survived, and both were the same mistake I had just recorded

**`y4` — "the deck plays while the console owns the air" SURVIVED.**  That is the defect itself, and
my tests asserted `monitorHasTheAir()` — the PREDICATE — and never that `needsRead` actually skips the
audio.  **Exactly `x5`'s shape, one ruling earlier**, where the fence was asserted and the filter was
not.  I recorded that lesson and then repeated it in the next batch.

**`y6`/`y8` — the two carriers.**  Deleting `mc.StopMonitor()` survived because `deck.Stop()` reports
the same event — indistinguishable WITH a deck, and the only stop there is WITHOUT one, so it is a
precondition rather than a duplicate.  Deleting `HandAir(AirMonitor)` survived because the deck reads
the ROUTER's owner while the Director reads its own: the deck would go on playing while the schedule
believed the console still held the air, and the rotation would never advance again.

Both are now asserted at the seam that can tell them apart: a station with no audio, and the
Director's own declarations.

**Eight plants, eight caught — three only after the tests grew.**

## What this does not do

The BED is still the monitor's rotation, not the station's.  The HUM LEAD's Lone Pine argument
(F-89) stands: a watchlist relay is not an appropriate bed for a station 300 miles away, and the
`←`/`→` selector is still drawn and inert.  That is the next step and it has its own fence — ruled at
**100 miles**, measured: 25 mi gives Bonsall ONE transmitter, 100 gives eight, and below 75 some real
stations get none at all.
