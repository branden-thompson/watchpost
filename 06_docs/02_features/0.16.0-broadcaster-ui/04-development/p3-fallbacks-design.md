---
title: "0.16.0 P3(b) — the two relay-failure fallbacks: the design, before the edit"
date: 2026-09-09
phase: BUILD
sev: SEV-0
authority: HUM LEAD
status: "DESIGN RECORDED.  The edit is deliberately NOT made in the same sitting as the analysis."
---

# P3(b) — the fallbacks

**S0 called this the real work, and it is the only part of the merge that is a behaviour change rather
than a rewiring.**

## What exists

`startSynth` has **three** production callers.  One is the rotation's ordinary tune.  **Two are relay
failures, and both bypass the schedule entirely:**

| Site | Trigger | What it does today |
|---|---|---|
| `app/radio.go:789` | the relay reached `player.Failed` while live | `go d.startSynth(ref, "relay unavailable — "+st.Err, gen)` |
| `app/radio.go:974` | `readSynth` — the relay went silent | `d.startSynth(ref, "the relay was silent", gen)` |

**Both start audio directly.**  Under the merge, the arbiter owns the air — so a fallback that starts
synth itself would be **the second speaker the whole batch exists to remove**, and it would appear on
the failure path, which is the worst place to find one.

## The shape the codebase already prescribes

**The deck must not decide this.**  `radio.go:790-795` states the rule for the neighbouring case, and
it applies unchanged:

> *"THE DECK REPORTS, THE DIRECTOR DECIDES (T3.2b).  What used to be a `time.AfterFunc` here … is now
> two facts the deck is the only thing able to observe."*

And the deck **already has the seam**: it reports `lineup.Ended{}` (`radio.go:804`) and
`lineup.Tuned{...}` (`radio.go:811`) through `tell`, which reaches `schedule.carry`.

**So the change is not "call something else here". It is: report the FACT, and let the Director
propose the card.**

## The design

1. **A fact, not a request.**  The deck reports that the relay it was carrying has failed or fallen
   silent — an event in the closed set, named for what happened rather than what should follow.
2. **The Director decides.**  It proposes a `LocationReport` card for the affected location onto the
   main track, exactly as the ordinary rotation now does.  **One producer, one path.**
3. **`startSynth`'s direct path retires in the same change.**  That deletion is what closes the
   double-speak window — S0 was explicit that the window is transitional, and that a guard is not what
   closes it.

## Why the edit is not in this commit

**This is the release's dangerous batch**, and its plan gives it its own red team, a UAT shared with
nothing, a timing property test over randomised arrival timing, a dark path and a go/no-go before P4
and P5.

**The failure path is where a rushed change does the most damage**: it runs when something is already
going wrong, it is the hardest thing to exercise deliberately, and a defect there is invisible until a
relay dies during severe weather.

**The analysis is committed; the edit is not.**  That is the same discipline the plan applied to the
merge itself — S0 answered the question before three batches were sunk on an assumption — applied one
level down.

## What the edit will need, so it is not rediscovered

- **A new event** in `lineup`'s closed set, and the set grows deliberately and once (PL-6).
- **A test that the fallback produces a CARD**, not audio — the double-speak property asserted, per
  FR-2.5, rather than built.
- **A test at the seam the deck actually routes through** (`tell` → `carry`), not past it: the 0.15.0
  build log records being burned twice by driving through the wrong seam.
- **`-race` on every run**, because this is the join between a pure state machine and a mutex-guarded
  audio owner.
