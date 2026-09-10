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

| Member | Why it is unwired | The writer it awaits | Batch |
|---|---|---|---|
| `Effect.Duck` | declared with a working executor and no producer.  Its own comment says why: *"the Director gains it with the main-track absorb (T3.2)"* | **the Director** — D-32 makes the duck a decision of the pure `Step`: is a priority card on the air, and is the bed live | **P5** |
| `Effect.Restore` | Duck's other half, and unwired for the same reason | **the Director** — the pair is closed, so every `Duck` is followed by exactly one `Restore` | **P5** |
| `Power.OffAir` | nothing can produce STANDBY, so the state the swap gate points at is unreachable.  **This row IS RS-3 and F-72** | **MasterControl** — it *"declares ON AIR or STANDBY, and everyone complies, the Director included"*, reaching the Director as `Powered{To: OffAir}` from **FR-5.4**'s named control | **P5** |
| `Origin.FromOperator` | the operator's controls do not exist, so no card is ever the operator's | **the Operator, through the Producer** — D-36 makes `[space]` the promote that writes it, and the origin is fixed at proposal and never rewritten | **P4** |
| `State.Refused` | `Propose` returns an error instead, so nothing reaches the state | **the Director** — D-35's capped discard pile, per DR-1's one writer.  **Not a track**: `held()` counts tracks and a parked card would stop the fault window ever firing | **P4** |

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
