# MVS-D-90 — the operator's relay choice is the row's one owner

**Status:** BUILT, 2026-09-11.  Closes **F-98**.  Files **F-99**.
**Reported by:** HUM LEAD, UAT 2026-09-11.

> "Relay control ← (no relay tuned) → doesn't 'stick' to my choice — no matter what I choose it will
> constantly go back to (no relay tuned) so there's no way to ensure which bed relay I've actually
> selected if I want the [b] Bed to be active."

---

## THE MECHANISM

**Two publishers of `BedMsg`, and the frequent one did not know about the operator.**

| Publisher | Sends | Fires |
|---|---|---|
| `stepBedRelay` (app/bedrelay.go) | `BedMsg{Relay: relayLine(chosen)}` — the STATION's NWR relay, by callsign | once, on a keypress |
| the `Publish` executor (app/executors.go) | `BedMsg{Relay: describeBed(v.Bed)}` — the DIRECTOR's `bed.ref`, a **watchlist LOCATION key** | **on every settle, i.e. every tick** |

`describeBed` returned `""` whenever `b.Ref == ""`, which is the ordinary state of a station whose
operator is choosing a relay before going on the air.  So the sequence was:

1. operator presses `→`; the deck tunes; the row shows `WNG637 San Diego Marine CA 162.425 MHz · 31mi`
2. the schedule settles, about a second later
3. the row publishes `""` and reverts to `(no relay tuned)`

**It is the same shape `noteBedCarrying` already fixed for `Carrying`, left unfixed one field along** —
and `bedrelay_test.go` has carried a test for that half since D-78:

> "The Director publishes it and the selector publishes it, and a selector that GUESSED would flicker
> the row between them."

The relay itself had no such agreement.

---

## THE FIX

**The app REMEMBERS the choice, and the settle asks for it.**

```go
lp.bedRelay = relayLine(chosen)   // set by the only thing that tunes the station's bed
func (lp *livePipelines) selectedRelay() string
```

`describeBed` asks `x.selected()` first, and falls back to the Director's label when the operator has
chosen nothing.

**`bedPick` could not stand in for this.**  It is an INDEX, and its zero value is a real relay — so a
station nobody had touched would report its nearest transmitter as though the operator had selected it,
claiming a tune that never happened.  That is mutant `mY4`.

**The fallback is not dead code.**  Until the operator touches the selector, what the bed is on is the
monitor's rotation, and the Director's label is the only honest thing to say about it.  It has a test
of its own (`TestAnUnchosenRelayLeavesTheDirectorsBedOnTheRow`) and mutant `mY3` covers the case where
an absent choice blanks the row instead of deferring.

**`bedSeams`** bundles `noteBed` and `selected` into one parameter.  Both halves exist for the same
reason — each is how one publisher of the row learns what the other knows — and passed separately they
would have been an eleventh and twelfth argument to a function that already had ten, with nothing
saying they belonged together.

---

## THE INSTRUMENT WAS VALIDATED BEFORE THE FIX WAS CLAIMED

Per the standing rule.  `describeBed`'s new clause was removed and the test re-run:

```
--- FAIL: TestTheOperatorsRelayChoiceSurvivesTheNextSettle
      chose:     "WNG637 San Diego Marine CA 162.425 MHz · 31mi from TOWER GPS"
      published: ""
```

Exactly the reported symptom, produced by the test on the old code.

---

## WHAT THIS EXPOSED, AND DID NOT SETTLE — F-99

**The station's bed and the monitor's bed are two owners of one output**, and this change gave one of
them the row:

- the console's selector walks the **station's NWR fence by callsign** (D-77/D-78) and tunes the deck
  directly;
- the Director's `bed.ref` is a **watchlist location key**, and its dwell (`advanceBed`) re-tunes the
  deck through `cutTo`;
- **neither tells the other.**

With Observer's watchlist rotation active (`Dwell > 0`, `bed.live`, `bed.ref != ""`), the dwell can
re-tune the deck out from under the operator's selection **while the row still shows what they picked**.

**Before D-90 the row was wrong all the time, so this was invisible.**  Now it is reachable, and it is
a RULING rather than a defect to close quietly:

1. the station's bed overrides the monitor's rotation while the console holds the air, or
2. the rotation wins and the row follows it, or
3. the operator's selection SUSPENDS the dwell for as long as the console holds the air.

D-74's air-ownership model is the obvious place to settle it — `advancesMonitor()` already gates the
dwell — but which way it goes is the HUM LEAD's call, and it is filed as F-99 rather than chosen here.
