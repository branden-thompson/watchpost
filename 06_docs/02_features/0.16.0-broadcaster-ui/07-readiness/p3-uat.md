---
title: "0.16.0 P3 — UAT: Observer must be indistinguishable"
date: 2026-09-09
phase: BUILD
sev: SEV-0
authority: HUM LEAD
status: "READY TO RUN.  Reframed 2026-09-09 on the HUM LEAD's own statement of it."
---

# P3's UAT is an OBSERVER REGRESSION TEST

**The HUM LEAD's framing, and it is the correct one:** *"the UAT would be to ensure that when I use
Observer, I cannot tell the difference in the audio outputs."*

**That is the whole test, and here is why it is sufficient.**  P3 changed **who decides when a report
plays**, and nothing else.  It did not change:

| | |
|---|---|
| **who controls the radio** | Observer, exactly as in 0.15.0 — tune, `[m]` Synth/Relay, `[r]` repeat, `[space]`, stop, volume, the Watchlist |
| **what plays** | the same `synth.Source`, the same segments, the same cast, the same handoffs, the same sign-off |
| **how it plays** | the same engine, the same give-way rule — a rendered report HOLDS under an alert, a live relay DIPS |

**What changed is one sentence:** the deck used to decide for itself that a location needed a read and
start the audio; now it reports the fact, the Director makes a card, and the card's read runs through
the narration arbiter.  **The listener's controls, the words and the sound are all meant to be
identical.**

**So the pass criterion is a COMPARISON, not a checklist.**  Any perceptible difference is a failure,
including ones no case below anticipates — which is the point of framing it this way rather than as a
list of features to tick.

## There is no operator UI to test, and that is not an oversight

**`ctrl+o` / `ctrl+b` do nothing in a running build.**  `broadcasterKeyMap()` is written and
`Router.keys` is never assigned, so no key reaches the console; the console renders the running order
and the station state when it is SENT them, and accepts no input at all.  **`shift+enter` — ON AIR /
STANDBY — is declared in a keymap nobody installs.**

**Nothing in the tree ever produces `lineup.OffAir`.**  Every power change comes from Observer:
`radioDeck.tune` reports RUNNING, `radioDeck.Stop` reports STOPPED.

**That is P4, P4.5 and P5's work and it has not been done.**  It is recorded here so this UAT is not
read as covering an operator surface, and because **it hides a trap that must be closed before the
console is ever reachable** — see `p3-console-reachability.md`.

## How to run it

**A/B against the shipped release.**  The claim is indistinguishability, so the comparison should be
against the thing it must be indistinguishable FROM, not against memory.

```sh
git worktree add /tmp/wp-0150 v0.15.0
cd /tmp/wp-0150 && make build && mv dist/watchpost /tmp/watchpost-0.15.0
cd -                       && make build          # this branch
WATCHPOST_DEBUG_RADIO=1 ./dist/watchpost          # and /tmp/watchpost-0.15.0 for the other leg
```

**Run the same steps on both, on the same locations, and listen.**  Fourteen prompts follow; each one
is a place a difference would be audible or visible, not a feature list.

| # | Do this on BOTH builds | A difference here means |
|---|---|---|
| **1** | Tune a location with **no relay in reach**.  Let the report run to its sign-off | the words, the voices, the pacing or the sign-off moved |
| **2** | Watch the **marquee** through the whole report | the per-segment detail line is not tracking the speech |
| **3** | Watch the **player row** and the **detail line** as it starts | the station name or the reason it is reading changed |
| **4** | Let the report **end on its own** | **the rotation did not advance.**  The report's end is what moves the bed on, and it now travels a different route — this is the single most likely regression |
| **5** | Set **repeat-one** and let a report end | the repeat stopped working, or repeated a stale reading instead of re-fetching |
| **6** | Let a **severe alert arrive while a report reads** | the report did not PAUSE and resume in full |
| **7** | Let an alert arrive **while a live relay plays** | the relay did not DIP and play on underneath |
| **8** | Press **`[M]`** during a report | **the broadcast stopped.**  `[M]` has never silenced the radio, and the merge nearly made it a stop button |
| **9** | Press `[M]`, then let an alert arrive | the hazard was read anyway, or consumed silently and never offered again |
| **10** | Press **stop** mid-report | audio continued, or the station restarted itself minutes later |
| **11** | Stop, then start again | it did not resume reading |
| **12** | Tune a **live relay, then kill the network** so it fails while playing | the fallback did not happen, did not say why, or **two voices spoke at once** |
| **13** | Tune a relay that connects and then **goes silent** | same |
| **14** | Run **thirty minutes** through a full Watchlist rotation | a location was skipped, repeated, or the rotation stalled |

**Cases 4, 8 and 12 are the ones to spend attention on.**  Four is the merge's new route for an old
fact; eight is the failure it most nearly shipped; twelve is the double-speak the whole batch exists to
remove, on the path that is hardest to reach deliberately.

## The log, for attributing anything you hear

`~/Library/Caches/watchpost/debug/radio.log`, timestamped, on the new build only:

| Line | What it means |
|---|---|
| `needs-read fresh=<bool> ref=<lat,lon> why=<reason>` | the deck noticed nothing is carrying this location.  **`fresh=false` is a decision, not an error** — the need arrived after you stopped or moved on |
| `segment key=… spoken=…` | which segment the read reached, and for how long |
| `schedule:declined:…` | a card the schedule refused, with the reason in words |

**Pair them.**  Every `needs-read … fresh=true` should be followed by segments for that location.
Segments with no `needs-read` in front of them would be an audio path nobody asked for — which is
exactly the defect.

## What this UAT does NOT cover

- **The operator console.**  It is unreachable; see above.
- **The alert rail's own content** — burst ordering, the divert count, the ladder.  Those shipped in
  0.14.0 and 0.15.0; cases 6, 7 and 9 test only that the merge did not disturb them.
- **Cold-start voice installation**, which is the first-run install ruling and not a P3 behaviour.
