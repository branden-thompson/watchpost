# D-73 — ONE RAIL, AND THE FENCE FOLLOWS THE SURFACE

**Ruled by the HUM LEAD, 2026-09-10**, as the acceptance test for air ownership:

> As long as when I switch to Broadcaster I don't hear alerts outside my service radius, then that's
> fine ("it just works") and when I switch back to Observer, that alert track has to re-adapt to
> whatever my filter settings dictate ("it just works").

---

## The approved architecture had a defect, and the code is where it was visible

The HUM LEAD approved **two schedules over shared infrastructure** (D-72's planning round, option 4).
Building it revealed what the plan could not see: **`startSchedule` builds one Director, and that
Director owns BOTH the main track and the alert rail.**  Two schedules means two Directors, each
holding a rail, each fed by the same `tickerDeck` — so **every hazard would be read twice.**

The rail is the safety path.  Raised before building rather than solved quietly, and the HUM LEAD
approved the substitution: **one schedule, one rail, and what MOVES is the fence around it.**

**The lesson is not "the plan was wrong".**  It is that an architecture approved from a description is
approved on the description's terms, and the first honest chance to check it against the tree is
during the build — which is exactly when it must be raised rather than worked around.

## The spike that saved a wrong build

I proposed wiring the air owner to `lineup.CutOver` / `bed.carries`, on the reading that
`bed.carries` means "the bed holds the programme, so the main track pauses" — which is ruled
(FR-4.2, D-11, D-32) and has **no production emitter**.  It looked like a designed seam waiting to be
connected.

**The spike said no.**  `bed.carries` means the operator has PARKED the station on the bed, and the
bed's own rotation pauses under it too (`advanceBed`, `dwellElapsed` and `stalledRotation` all gate on
`advances(MainTrack)`, which is false while it carries).  It is a manual cut-over, not an air owner.
Wiring it would have frozen the listener's watchlist rotation — mM3's defect, arriving by a new road.

## What the air owner actually is

**The surface the operator is looking at**, which is where the HUM LEAD pointed in the first place:
*"maybe the router handles this."*  `swapTo` is the ONE place a swap is granted, so it is the one
place that can say so — and a REFUSED swap announces nothing, because nothing moved.

| | Observer active | console active |
|---|---|---|
| the rail's fence | the listener's alert radius, around their default location | the station's service radius, around its transmitter |
| a console with no epicentre | — | **the listener's fence**, never "All" |

**One function answers it** — `scopeFor` — and BOTH readers ask it: `fence()`, which decides the
rail's ORDER, and `scopeToRadius`, which decides what reaches the rail at all.

## The plant that survived was the one that mattered

`x5 — the feed filter ignores the scope and reads the listener's setting` **SURVIVED** the first run.
`fence()` was asserted and `scopeToRadius` was not — and they do different jobs: the fence decides
order, the filter decides **what the operator HEARS**.  A rail correctly ordered around alerts that
should not be on it fails the ruling completely while passing every test written for it.

That is the same shape as this release's other survivors, one function along: the assertion stopped
at the half that was easy to reach.

## Two details that are the feature, not the plumbing

**The watchlist stopped being a parameter.**  `scopeToRadius` took the listener's locations for their
ORIGIN; the origin comes from the scope now.  Leaving the parameter would leave the next reader a
spare answer to a question the function no longer asks — which is how a fence comes to be measured
from two places.

**A swap nudges the cycle.**  The ticker runs every two minutes; without the nudge the tape would go
on showing the other surface's alerts for up to that long, which is the opposite of "it just works"
in both directions.  Buffered by one and sent without blocking, so a flurry of swaps is a single
pending re-scope rather than a queue of cycles doing network work — the same shape, and the same
reason, as the injector's own wake.

## Still open: the PROGRAMME, as distinct from the rail

This closes the hazard half of the HUM LEAD's second blocker.  **The programme half is not built**:
with the station Running, Observer's watchlist rotation (`needsRead` → `startSynth`) and the station's
line-up (the schedule's `Speak`) both advance on the SAME gate — `advances(MainTrack)` — so both can
play.  And `app/radio.go` declares `Powered{Running}` from Observer's TUNE, which is why listening on
one surface puts the other ON AIR (D-69's root).

Fixing it needs that one gate split into two questions — the listener's rotation and the station's
line-up — with the owner deciding which is true.  Filed as **F-88** rather than half-built.

**Six plants, six caught** (one only after the test grew to reach the filter).
