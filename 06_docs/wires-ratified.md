# Ratified unwired members — the closed-set completeness check

**`make wires` checks that every member of every closed set names a production WRITER and a production
READER.  Target: 0 unexplained.**

A row here is an exemption, and it works exactly like a metric-D exemption and a P10 exemption: **the
reason is RATIFIED by the HUM LEAD, never self-issued.**

## Every row here is OWED, not accepted

**HUM LEAD ruling, 2026-09-09:** *"they can join the ledger now, but need to be removed once wiring is
in place."*

So these are not permanent exemptions.  Each names **the writer it is waiting for** and **the batch that
owes it**, and the row leaves when the wiring lands.

**AND THE EXPIRY IS ENFORCED, NOT REMEMBERED.**  `wires` fails on a **stale row** — a member that is
ratified here and is no longer unwired.  A promise to clean up later is the exact shape this project
keeps paying for; this one cannot be forgotten, because wiring the member is what breaks the build until
the row goes.

**Rows are read ONLY between the fence below.**  Without it every table row starting with a backticked
dotted name was an exemption, so the "already closed" table further down silently re-created two
obligations that had just been discharged.  **A ledger where writing ABOUT a member exempts it fails in
the direction of silence**, and silence here means an unwired member nobody is told about.

<!-- wires:exemptions -->

| Member | Why it is unwired | The writer it awaits | Batch |
|---|---|---|---|
| `Event.Moved` | the operator promoting or demoting a card.  The event, the reorder mutator and FR-3.3's own mutant landed in P4's pure half; **the key that presses it is P4's UI half** | **the Operator, through the console** | **P4 (UI)** |
| `Event.Dropped` | the operator taking a card out of the running order.  **Blocked on FR-3.7's CONFIRM**, which is a console control — the pure half deliberately does not invent one | **the Operator, through the console** | **P4 (UI)** |
| `Event.Restored` | the operator's undo.  Same half, same block | **the Operator, through the console** | **P4 (UI)** |
| `Event.CutOver` | the operator moving the programme between the lanes.  The event and the state landed in the track-model batch; **the control that presses it is P4's** | **the Operator, through the console** — D-11 and FR-4.2's cut-over.  Caught by this gate on the commit that added it, which is what the gate is for | **P4** |
| `Event.Offered` | the Producer offering the Director cheap template cards to fill the lineup with (D-40).  The pure half — the choice rule, the depth, the dedupe — landed with 9 caught plants; **nothing in production offers yet**, and the writer is one step away | **the Producer, through the `Publish` executor**, which already returns events — so no new effect, exactly as D-40's option 2 was ruled.  **RATIFIED by the HUM LEAD 2026-09-10**, under the standing pattern "they can join the ledger now, but need to be removed once wiring is in place" | **P4 (top-off wiring)** |
| `State.Refused` | `Propose` returns an error instead, so nothing reaches the state | **the Director** — D-35's capped discard pile, per DR-1's one writer.  **Not a track**: `held()` counts tracks and a parked card would stop the fault window ever firing | **P4** |

<!-- /wires:exemptions -->

## Rows that have already left, because the wiring landed

**The expiry is not theoretical — it fired on the commit that wired the duck.**

| Member | Was owed to | Closed by |
|---|---|---|
| `Effect.Duck` | the Director, at P5 | the track-model batch: `givingWay()` decides it from state the Director already holds, and `settle` emits the change |
| `Effect.Restore` | the Director, at P5 | the same commit — the pair is closed by construction, edge-triggered, so a drain of several cards dips once (MVS-D-67) |
| `Power.OffAir` | MasterControl, at P5 | **P4's keymap-and-standby slice.**  `mastercontrol.GoToStandby` declares it and carries it to the Director — the sentence 0.14.0's role model left open, *"it does NOT own ON AIR / STANDBY"*, closing.  **This row WAS RS-3 and F-72** |
| `Origin.FromOperator` | the Operator through the Producer, at P4 | **P4's pure half**: `onRestored` proposes a NEW card from the pile and attributes it to the human who put it back (FR-3.4).  It had existed for two releases with nothing ever constructing it |

**`make wires` went red the moment they gained a writer**, named both rows as STALE EXEMPTIONS, and
stayed red until they were deleted.  That is the whole mechanism working on its second day: the promise
to remove a row was not remembered, it was enforced.

## What is NOT here, and why that matters more than what is

**Three findings were resolved rather than ratified**, on the same day this file was written:

- **`narrationClass.narrateRotation`** — **deleted.**  Unreachable in a default build, and D-33 rules
  the programme is not a narration.  `mainTrackLive` and `ownsTheAir` went with it.
- **`State.Proposed`** — **the tool was wrong.**  It is the zero value, so every `Card{…}` writes it and
  no syntax walk can see that.  A ratified row would have recorded a false thing as ratified; the tool
  has a `Zero` verdict instead.
- **`Band.CloseBand`** — **the code was fixed.**  `Band.String` named one member and let the other fall
  through; it is a registry now, matching the pattern origins, states and slots already use.

**An exemption records that a finding is ACCEPTABLE.  None of those three was acceptable — they were
wrong**, and a ledger that had absorbed them would have made three defects look like decisions.

## Where this came from

Proposed at 0.14.0's round-3 red team, against a defect shape counted **nine times in one release**:
*"a rule implemented and pinned in one layer, not carried by the layer that would deliver it."*  Named
in that release's debrief as **"the single highest-value thing 0.15.0 could inherit."**  Unbuilt through
two releases, and 0.16.0 P3's audio merge was another instance — `Duck` and `Restore` sat here with no
producer while the batch that was meant to give them one routed around them.
