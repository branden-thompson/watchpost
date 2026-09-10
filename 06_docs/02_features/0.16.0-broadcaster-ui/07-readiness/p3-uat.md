---
title: "0.16.0 P3 — UAT: the AUDIO OUT is unchanged"
date: 2026-09-09
phase: BUILD
sev: SEV-0
authority: HUM LEAD
status: "NOT READY.  The flip it was written for was reverted on 2026-09-09.  Against the dark state there is nothing to regress: Observer's audio path is untouched by design, and the ten signal cases have no subject until the flip lands again."
---

# HELD — there is nothing to test yet

**The flip was reverted on 2026-09-09** (`04-development/p3-flip-postmortem.md`).  In the dark state
the deck still plays every report through its own path, so **Observer's audio is unchanged by
construction** and the ten signal cases below have no subject.

**The HUM LEAD said this before the revert and was right:** *"UAT cannot really begin until we wire up
the Broadcaster UI mock — unless it's purely as regression UAT of Observer."*  It is the second half,
and against the dark state even that is vacuous.

**This document stands as written and becomes live the day the flip does.**

---

# What P3's UAT tests: the audio out, unchanged

**THE BRIEF ALREADY SAYS THIS, and the first two drafts of this document re-derived it instead of
citing it.**  Recorded because the error is the interesting part: three framings were reasoned out from
the code while `01-objectives` had settled the question at DISCOVER.

| The question | Where it was already answered |
|---|---|
| who is on the other end | **Brief, *Who benefits*:** *"the human listener on the radio channel, who never sees it and feels every mistake made on it."*  There is no listener at the terminal |
| what "on the air" can even mean | **FR-5.5:** *"Watchpost has NO RADIO PATH — it produces audio, and a separate transmitter the application cannot observe puts it over the air."*  That is the headphone jack, and it is a ratified requirement with its own exit criterion |
| why the two surfaces share the audio | **Brief, *Summary & Intent*:** *"Watchpost can already run a station.  It cannot yet be OPERATED as one."*  Audio out is audio out; Broadcaster adds operation, not a second pipeline |

**So P3's UAT is a SIGNAL-PATH REGRESSION TEST.**  P3 changed **who decides when a report plays** and
nothing else — not who controls the radio, not what plays, not how it plays.  The question is whether
the same audio comes out of the machine, continuously and in the same order, now that the schedule owns
the timing.

**Observer is the HARNESS, not the subject.**  It is the only UI currently wired to the audio — the
console reaches nothing, for the reason in `p3-console-reachability.md` — so it is how the signal path
gets driven, and that is the only reason it appears below.

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

## What this sharpens, against requirements that already exist

**P3(d) scoped the mute check to the alert rail alone**, and justified it as *"`[M]` has never silenced
the radio."*  **The stronger reason follows from FR-5.5:** on a broadcaster the programme is going out
over a transmitter and the operator is watching a screen rather than listening, so a mute is dead air
nobody would notice.  The decision stands; its justification is now the ratified one.

**Two questions this raises against requirements that are already written:**

1. **Does the alert rail's mute apply on a Broadcaster at all?**  `[M]` is not among Broadcaster's
   controls in the mock, and **FR-5.1** makes STANDBY (`OffAir`) the thing that holds the rail.  Whether
   a listener-side mute reaches an operator surface at all is a **P6 settings** question — **R-2.2**
   already divides shared properties from per-surface ones, and this is a candidate for the second list.
2. **GAIN is a transmit level; Observer's control is a listening volume.**  **R-4.4** requires the GAIN
   control *"drawn as the mock draws it"* and **C-6** already measured the gap: *"volume exists; gain
   does not, and nothing persists it."*  So this is P6's, with the answer partly recorded.

## What this UAT does NOT cover

- **The operator console.**  It is unreachable; F-72.
- **The alert rail's own content** — burst ordering, the divert count, the ladder.  Those shipped in
  0.14.0 and 0.15.0; cases 4, 5 and 14 test only that the merge did not disturb them.
- **The physical patch.**  Levels, impedance, and what the radio does with the signal are the operator's.
