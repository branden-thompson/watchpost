---
title: "0.16.0 P3 — UAT: the AUDIO OUT is unchanged"
date: 2026-09-09
phase: BUILD
sev: SEV-0
authority: HUM LEAD
status: "READY TO RUN.  Reframed twice on 2026-09-09; the second correction is the one that matters."
---

# What P3's UAT actually tests: the signal leaving the machine

**The correction, in the HUM LEAD's words:** *"Broadcaster UI only cares about the audio out working —
it is the human operator's job to effectively plug the headphone jack from the computer into the radio's
microphone.  The 'listener UAT' doesn't make much sense."*

**That is right, and the first two drafts of this document were confused about who is on the other end.**

| | Observer | Broadcaster |
|---|---|---|
| who hears it | **the person at the terminal** | **an audience on a radio channel**, through a physical patch from the audio jack to a microphone input |
| what the screen is for | the product | **the operator's instrument panel** — they are watching, not listening |
| "do not speak to me" (`[M]`) | a sensible control: silence my desk | **meaningless, and dangerous**: silencing the programme is DEAD AIR on a live channel |
| the level control | listening volume | **transmit GAIN** — how hot the signal is into the radio.  The mock draws it that way |

**So P3's UAT is a SIGNAL-PATH REGRESSION TEST.**  The question is not "does the listener enjoy it".
The question is: **does the same audio come out of the machine, continuously, in the same order, as it
did before the schedule took ownership of it.**

## Observer is the HARNESS, not the subject

**Observer is the only UI currently wired to the audio** — `ctrl+o` / `ctrl+b` do nothing in a running
build and the console accepts no input at all (F-72, `p3-console-reachability.md`).  So Observer is how
the signal path gets driven, and that is the only reason it appears here.

**What is being checked is what comes out of the jack.**  The screen-side observations below are
included because they are Observer regressions in their own right, and they are marked as such —
**they are not what the Broadcaster cares about.**

## How to run it

**A/B against the shipped release**, because the claim is that nothing changed and the comparison should
be against the thing it must match, not against memory.

```sh
git worktree add /tmp/wp-0150 v0.15.0
cd /tmp/wp-0150 && make build && mv dist/watchpost /tmp/watchpost-0.15.0
cd -              && make build          # this branch
WATCHPOST_DEBUG_RADIO=1 ./dist/watchpost
```

**Run the same steps on both, on the same locations.**  Headphones are enough; the patch to a radio is
not needed to hear whether the signal is right.

## The signal, which is what P3 is about

| # | Do this on both builds | A difference means |
|---|---|---|
| **1** | Tune a location with **no relay in reach**.  Let the report run to its sign-off | the words, the voices, the pacing or the sign-off moved |
| **2** | Let the report **end on its own** | **the rotation did not advance — DEAD AIR.**  The report's end is what moves the bed on and it now travels a different route.  On a live channel this is the worst outcome in the list |
| **3** | Set **repeat-one** and let a report end | the repeat stopped, or repeated a stale reading instead of re-fetching |
| **4** | Let a **severe alert arrive while a report reads** | the report did not PAUSE and resume in full — either it was talked over, or the rest of it was lost |
| **5** | Let an alert arrive **while a live relay plays** | the relay did not DIP and play on underneath |
| **6** | Tune a **live relay, then kill the network** so it fails while playing | **two voices at once**, or silence with no fallback.  The double-speak this batch exists to remove, on the path hardest to reach deliberately |
| **7** | Tune a relay that connects and then **goes silent** | same |
| **8** | Press **stop** mid-report | audio continued after the operator killed it, or the station restarted itself minutes later |
| **9** | Stop, then start again | it did not resume |
| **10** | Run **thirty minutes** through a full Watchlist rotation | a location was skipped, repeated, or the rotation stalled.  **A stall is dead air** |

**Two, six and ten are the ones that matter on a live channel.**  Every one of them is a way for the
transmitter to be carrying nothing, or carrying two things at once, and neither is recoverable by an
operator who is not listening on the other end.

## Observer's own regressions, checked because they are free

**Not Broadcaster concerns.**  Recorded separately so a failure here is not read as a signal defect.

| # | Do this | A difference means |
|---|---|---|
| **11** | Watch the **marquee** through a whole report | the per-segment detail line stopped tracking the speech (UAT 83) |
| **12** | Watch the **player row** and the **detail line** as a report starts | the station name, or the reason it is reading rather than relaying, changed |
| **13** | Press **`[M]`** during a report | **the broadcast stopped.**  `[M]` has never silenced the radio, and the merge nearly made it a stop button.  **It is an Observer control and must not follow the operator to the console** — see below |
| **14** | Press `[M]`, then let an alert arrive | the hazard was read anyway, or consumed silently and never offered again |

## What this raises for the console, and it is not P3's to answer

**`[M]` must not exist on the Broadcaster surface.**  A broadcaster that mutes is a transmitter carrying
dead air, and the operator — who is watching a screen, not listening — would not know.  P3(d)'s decision
to scope the mute check to the alert rail alone is consistent with that and was made for a weaker
reason; **the real reason is that silencing the outgoing programme is not a control a station should
have.**

**Two open questions for P5/P6, recorded here because this is where they surfaced:**

1. **Does the alert rail's mute apply on a Broadcaster at all?**  An operator holding hazards means
   hazards do not go out over the radio.  That is a safety decision, not an implementation one.
2. **GAIN is a transmit level, not a volume.**  The mock draws `GAIN - ████ + 100`; Observer's control
   is a listening volume.  They are different quantities that happen to share a widget, and whether they
   are the same setting is a ruling.

## What this UAT does NOT cover

- **The operator console.**  It is unreachable; F-72.
- **The alert rail's own content** — burst ordering, the divert count, the ladder.  Those shipped in
  0.14.0 and 0.15.0; cases 4, 5 and 14 test only that the merge did not disturb them.
- **The physical patch.**  Levels, impedance, and what the radio does with the signal are the operator's.
