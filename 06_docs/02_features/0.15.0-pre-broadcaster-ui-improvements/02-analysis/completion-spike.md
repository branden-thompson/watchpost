---
title: "0.15.0 — B5 entry: the completion-signal spike"
date: 2026-09-08
phase: BUILD · B5 entry condition
sev: SEV-0
status: "Answered. The signal lands; the bound was already in the right place and the wrong shape to test."
---

# The completion-signal spike

**Charter:** can one completion event be threaded out of `playClip` to the
Director, and what does it cost?  **Timebox:** one hour.  **Output:** where FR-9's
bound lives, and this batch's size.  **Disposal:** the seam is kept if it is the
real design; deleted if not.

## What the read's timing actually is today

`speaker.deliver` calls `play(c)` — fire-and-forget into `Preview` /
`PreviewAside` — and returns `c.dur`, a duration **computed from the PCM's own
length**.  `speaker.hold` then sleeps that, in `holdStep` steps, through
`awaitAir` so a suspension does not count.

**The Director's model of "the read finished" is "I slept the computed length."**
Nothing observes the audio.

## The bound was already there, and already the right shape

`playClip`'s watcher polls the player every 50 ms for at most 12,000 iterations
— and `if !held { i++ }` means **a held line does not spend the budget**.  That
is exactly what the exit condition asks for: *held-excluded elapsed time, on the
read's own player, never on the live stream.*

**So PL-D-5 was not just mis-sited, it was redundant.**  Revision 1 wanted the
bound in `player.Engine.watch`, which times the RADIO STREAM: one caller,
`playPCM`, reached from the relay and `StartSource`.  A bound there would abort a
station that had played for hours, and would not touch a read at all.  The read
already had its bound; what it did not have was **anyone being told how it
ended**.

## What the spike built

- `playClip` returns `<-chan ClipEnd`, buffered and closed after its one value,
  so a caller that walks away cannot pin the goroutine that closes the player.
- `ClipEnd.Spent` says WHICH ending: `false` when the player ran out of audio,
  `true` when the watcher's budget ran out first — a read that neither finished
  nor errored, which is the shape FR-9 is about.
- `PreviewWatched` / `PreviewAsideWatched` expose it; `Preview`, `PreviewAside`
  and `Audition` keep their signatures and discard it, so no existing caller
  moved.

## The finding that changes the batch

**The bound could not be reached by a test.**  Ten minutes of air time is right
for a listener and impossible for a suite, so the fault path — the whole point of
FR-9 — had never been observed.  The budget now lives in a field the package's
own tests shorten.  *A bound nobody can reach in a test is a bound nobody has
watched fail.*

Both endings are now pinned, and both were watched failing:
`TestAClipReportsThatItFinished` and
`TestAClipThatNeverEndsReportsItsBudgetSpent`.

## What this means for the rest of B5

**The completion signal is a WATCHDOG, not a replacement for the pacing.**  The
Director's `hold(c.dur)` stays: it paces the read against air time and is what
makes a suspension free.  The signal answers a different question — *did the
audio actually end, and how* — which is what turns a stall into a reported fault
instead of a schedule that waits forever.

That keeps the batch small: the app wires `deliver` to the channel and treats
`Spent` as the fault, rather than rebuilding how a read is timed.
