---
title: "0.16.0 P3 — the dark run: how to observe the merged producer before it owns the air"
date: 2026-09-09
phase: BUILD
sev: SEV-0
authority: HUM LEAD
status: "SUPERSEDED 2026-09-09 by p3-uat.md.  The staged switch this describes was DELETED with the direct path at P3(d); kept as the record of a control that was designed, built and then made unnecessary by the ruling."
---

# The dark run — SUPERSEDED, AND KEPT

**This protocol described a control that no longer exists.**  `WATCHPOST_MAINTRACK` was the staged
switch between the producer landing and the flip; the HUM LEAD ratified Shape B, which made the flip
small enough to land in one change, and the switch was deleted with the direct path.  **A switch that
outlived the merge would have been a second way for the station to behave — the thing being removed.**

**What replaces it is `p3-uat.md`.**  What SURVIVED it is the instrument: the `needs-read` line is still
written, and pairing it with what the station actually did is still how a read is attributed.

**Kept rather than deleted** because the control was real, was built, and stopped being needed for a
reason worth being able to find later.  Everything below is the record of that.

---

# The dark run (as designed)

**The plan's control, in its own words:** *"the merged producer runs and is observed WITHOUT owning the
air before it owns it, so its decisions can be compared against the live path's for a period."*

**This is that run, and it can be done now** — the producer, the switch and the instrument all landed
in P3(a).  Nothing here waits on the flip.

## What is being asked

**Does the merged producer decide the same things the live path does, at the same moments?**  Three
questions, and each has a falsifying observation:

| Question | What would falsify it |
|---|---|
| **Does it fire when the station reads?** | a synthesised read starts with no `needs-read` line before it |
| **Does it fire when the station does NOT read?** | a `needs-read` line with no read after it — a card the merged station would have queued and the live one never did |
| **Does it fire ONCE?** | two `needs-read` lines for one location between two reads |

## How to run it

```sh
WATCHPOST_MAINTRACK=dark WATCHPOST_DEBUG_RADIO=1 watchpost
```

**Both are opt-in and both default off.**  A station without them is the station that shipped in
0.15.0, which is the property every P3 commit so far has preserved.

Then use the station normally for a period — a Watchlist rotation, a relay that drops, a stop and a
start.  **The relay-failure cases are the ones worth reaching deliberately**, because they are the two
sites that are hardest to exercise and the ones the flip changes most: tune to a live relay and let it
fail, and tune to one that goes silent.

The log is at `~/Library/Caches/watchpost/debug/radio.log` (macOS) — `userCacheSubdir("debug")`.

## What the log says

Three kinds of line share the file, timestamped, which is what makes the comparison possible at all:

| Line | Written by | What it means |
|---|---|---|
| `needs-read stage=dark fresh=true ref=<lat,lon> why=<reason>` | **the merged producer** | the deck reported that nothing is carrying this location |
| `<state> mount=… title=…` | the engine | what the live path actually did |
| `segment key=… spoken=…` | the live path's source | the read the live path is performing |
| `schedule:declined:Speak(…):the main track is dark; …` | the schedule | the card the merged station would have read, stopping one call short of the voice |

**`fresh=false` is a decision, not an error.**  It means the need arrived after the listener had stopped
or moved on, and the producer dropped it deliberately.  It is recorded because *"the live path started a
read here and the producer did not"* has two possible causes, and this is what tells them apart.

## Reading the result

**Pair them.**  Every `needs-read … fresh=true` should be followed by the live path starting a synth
read for the same `ref`, and every synth read should have one in front of it.  A `schedule:declined`
line for that card confirms the merged station got as far as having words and stopped where it was
meant to.

**What a mismatch means, before assuming the producer is wrong:** the ordinary tune reports the need
and starts the read on the SAME call, so those pair trivially.  **The two fallbacks are where a real
divergence would show**, because one of them runs on a goroutine off the engine's own status callback.

## What this run does NOT establish

- **Nothing about audio.**  In dark the card never reaches the voice, so this says nothing about how a
  report SOUNDS through the arbiter.  That is the flip's UAT, and the flip has an open ruling
  (`04-development/p3-flip-design.md`).
- **Nothing about timing under load.**  The randomised interleavings are the property test's
  (`platform/lineup/merge_property_test.go`); this run is about agreement, not about races.
- **The cost is real while dark:** a report is composed twice, so the location's products are fetched
  by both paths.  That is the price of observing the real producer rather than a simulation of one, and
  it is bounded by the rotation's own cadence.
