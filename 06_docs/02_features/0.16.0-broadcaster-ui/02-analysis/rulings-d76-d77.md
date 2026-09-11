# D-76 — THE CUT-OVER BELONGS TO THE MONITOR, AND D-72 TOOK IT

**Found by DRAWING THE FLOW for the HUM LEAD, 2026-09-11**, not by a gate.

D-72 moved all three of `startSchedule`'s list-reading seams from the listener's watchlist to the
station's pool, and gave a reason:

> Feeding one from the pool and another from the watchlist would let the Director schedule a location
> its own Composer could not resolve, which the D-67 cool-off would then bench for five minutes
> apiece.

**That is true of `propose` and `compose`.  It is not true of `cutTo`**, which serves `advanceBed` —
the OPERATOR'S OWN rotation, moving through their own watchlist.  A watched location outside the
station's pool stopped resolving, and the tune died as `schedule:tune-unknown` with nothing said.

**Two of three is the worst kind of wrong**: the reasoning was sound, it covered most of the case, and
the exception was invisible until the flow was drawn as two programmes side by side.  The diagram the
HUM LEAD asked for found a defect that four gate runs had not.

## The plant that made the test honest

`a1 — the cut-over resolves against the pool again` **SURVIVED** the first run.  The test compared the
two lists through `refFor` and proved nothing about which one `startSchedule` builds `cutTo` from —
the same call-site blindness `producer()` was named to remove.  It drives `startSchedule` now and
watches the deck: a tune that RESOLVED reports the need it found, and one that did not reports
nothing at all, which is exactly how the regression hid.

---

# D-77 — THE BED IS THE STATION'S, AND ITS FENCE IS ITS OWN SETTING

**Ruled by the HUM LEAD, 2026-09-10**, from a case the code could not have argued with:

> I may, in Observer, choose 10 locations COMPLETELY OUTSIDE the Transmitter service area I am running
> on Broadcaster … I want to listen to Lone Pine in Observer because I have a personal affinity for
> that location.  However, I run a hyper-local GMRS in broadcaster for Bonsall, with a service radius
> of 25 miles.  Lone Pine's "Fresno" relay is **completely inappropriate** for being the relay.

And the code agreed: `lineup.Programme{Watchlist, Dwell}` has exactly ONE producer — Observer's
`SetRepeat` — so the Director's bed IS the listener's watchlist relay rotation wearing the console's
`[B] BED` label.  The `←` / `→` / `[B]` controls are drawn and bound to nothing.

## A separate fence, because relays are sparse

| from | 25 mi | 50 mi | 75 mi | 100 mi | 150 mi |
|---|---|---|---|---|---|
| **Bonsall, CA** | 1 | 3 | 5 | **8** | 11 |
| **Lone Pine, CA** | 0 | **0** | 3 | 4 | 10 |
| **Chicago, IL** | 1 | 6 | 9 | 16 | 35 |
| **Minot, ND** | 1 | 1 | 3 | 4 | 11 |

**A 25-mile bed fence gives the operator ONE choice**, which makes `←`/`→` decorative — and that is
the same 25 miles that is a perfectly good SERVICE radius.  The two numbers move for different
reasons: one is who the station is FOR, the other is what hardware happens to exist nearby.  Deriving
one from the other would couple two facts that have nothing to say to each other.

**Default 100, bounds 25–150**, all three measured: 100 reaches a real choice everywhere sampled; the
floor is where even Chicago reaches one; the ceiling is where the bed starts carrying a forecast for a
region the station's listeners are not in — which is the objection that started this.

## The count is free; whether you can TUNE it is not

`transmitters.csv` says a transmitter EXISTS.  Whether anyone STREAMS it comes from the relay
directory, over the network.  So:

- **Settings** shows *"N transmitters in range"* — a structural fact about the fence that never lies
  and needs no network, safe to recompute per keystroke
- **The bed's own row** shows what actually resolved, at tune time, where the app is already making
  that call

A settings preview that asked the directory would make a request per keystroke and read ZERO offline,
which is a different lie.

## One owner for the advice, and it is ready for a row that does not exist

> Even if we immediately don't make this setting visible in settings, it should be architected this
> way so it is literally a "flip of a switch".

`bedReachFor` returns the count and the sentence; `bedAdvice` is the sentence alone, so the wording
can be asserted without a table and the counting without a wording.  The thresholds are measured
rather than chosen — **none** and **one** are the two states worth naming, because one is a station
with no bed and the other is a selector with nothing to select.

**The default carries no warning**, deliberately: a default that arrives wearing one is a warning the
operator learns to ignore.

## Seven plants, five caught, and both survivors were vacuous tests

`a2 — an out-of-service transmitter is offered` **SURVIVED** because the test swept the fence around
Bonsall for a status it would never find there.  It asserts at a place where a dead transmitter
actually exists now, and asserts the fixture is dead FIRST.

That is the third time this release a plant has found an assertion that could not fail.  The shape is
always the same: the test looks for something in a set that does not contain it, and passes.
