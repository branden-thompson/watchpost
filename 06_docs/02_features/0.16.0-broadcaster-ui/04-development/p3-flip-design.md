---
title: "0.16.0 P3(b)+(d) — the flip: what retiring startSynth actually costs, and the ruling it needs"
date: 2026-09-09
phase: BUILD
sev: SEV-0
authority: HUM LEAD
status: "DESIGN RECORDED.  ONE ARCHITECTURE QUESTION IS OPEN AND IT IS THE HUM LEAD'S.  The edit is not made."
---

# The flip

**P3(a) and P3(c) have landed.  What remains is the change that hands the air to the schedule** — the
two relay-failure fallbacks route through it, `startSynth`'s direct path retires, and `app/maintrack.go`
is deleted.  The plan describes that as one change, and it is, **but it is not the change the plan
described.**

## The plan said "the direct `StartSource` path retires behind the arbiter".  That sentence hides a
## substitution.

**S0 concluded the merge was mostly wiring, and for the SCHEDULE half it was — five seams, all live.**
The audio half is different.  Reading a report through the arbiter does not move the existing playback
behind a new owner; **it replaces the player.**

| | The rotation today | The card path |
|---|---|---|
| what plays it | `synth.Source` streamed through `engine.StartSource` | a `lineup.Script` read line by line through the narrator's `speaker` |
| how a line becomes audio | the source pulls segments as it goes | `radio.go:694 render` → `synth.AlertNarration` → PCM in memory → `engine.Preview` |
| the player row | `"Watchpost Synth (<voice>)"`, with the source's rate | a preview over whatever the engine was doing |
| the station line | `setMode("synth", "Watchpost Synth · "+ref.Label, why)` | **nothing sets it** |

**The per-line latency concern does NOT apply**, and it is worth saying so plainly because it was my
first reading and it was wrong: `readScript` walks the script part by part and each part is rendered
just before it plays, so a report plays segment by segment exactly as the source does.  **The whole
report is not buffered.**

**Seven things are genuinely lost**, and each of them is behaviour a listener can see or hear today.

| # | What retires with `startSynth` | Where it lives now | Severity if simply dropped |
|---|---|---|---|
| G-1 | **the rotation's end signal.**  `cycleEnded` reads `player.EndedTitle` off `d.source`; that is what emits `lineup.Ended{}`, and `Ended` is what advances the bed and the Watchlist | `radio.go:859`, `radio.go:807` | **BLOCKER.**  With no source, `Ended{}` never fires and **the rotation stops advancing entirely** |
| G-2 | **the marquee**, `setDetailTimed(seg.Text, spoken)` per segment, paced to the voice (UAT 83) | `radio.go:521` | MAJOR — the detail line goes static for the whole report |
| G-3 | **correspondent handoff lines**, `src.SetHandoffLine(d.composer.HandoffLine)` | `radio.go:529` | MAJOR — the cast stops introducing itself mid-report |
| G-4 | **repeat-one**, `src.Loop(d.repeat == tty.RepeatOne)` (UAT 93) | `radio.go:531` | MAJOR — a listener's repeat setting silently stops working |
| G-5 | **the station and player rows**, `setMode` + `StartSource`'s title | `radio.go:459`, `radio.go:535` | MAJOR — the radio panel stops naming what is playing |
| G-6 | **first-use voice install with progress in the player** (HUM LEAD ruling: first-run install) | `radio.go:461` | MINOR — `resolveVoice` starts a background install, but the foreground progress is `startSynth`'s |
| G-7 | **the segment's DELIVERY — `Role`, `SelfIntro` and `Pause`** — and **the `why` a read is happening at all** | `synth.Segment` (compose.go) vs `lineup.Part` (script.go); `needsRead`'s `why` | **MAJOR, and found by the red team.**  `scriptFromSegments` keeps `Text` and drops the other four fields, because `lineup.Part` has nowhere to put them.  Under Shape A the cast collapses to one `cast.Standard` voice for the whole report, `SelfIntro`'s double-introduction suppression has nothing to suppress, and UAT 112.3's per-segment pauses are gone.  **Separately, `why` has no transport at all**: it is one of three listener-facing diagnostics and in `live` it is passed to a `startSynth` that is never called |

**G-7 is the one that decides between the two shapes below**, because Shape A must rebuild it and Shape B
never touches it: if the source still plays the report, the roles, the introductions and the pauses are
already right, and the card's `Script` is a DISPLAY artefact whose job is the console.

**G-1 alone makes "delete the direct path" wrong as written.**  A card's `Finished` would have to take
over what `Ended{}` does, and that is a design decision, not a deletion.

## Two shapes.  The difference is what the arbiter is FOR.

### Shape A — the card carries the words (what the plan literally says)

`BuildCard` composes the report into a `Script`; `Speak` reads that script through the narrator.  The
synth source is gone.

- **All six gaps must be rebuilt** on the card path: a display hook in `readHooks`, handoff lines in
  `readScript`, a loop concept in the schedule, a station-row owner, and a new source of `Ended{}`.
- The Broadcaster console gets the report's words for free, which FR-1 wants.
- **It re-implements a player that already works, on the path that carries every ordinary broadcast.**

### Shape B — the card carries the ORDER; the source still carries the report  *(my recommendation)*

The arbiter owns **who speaks**.  It does not have to own **how a report is spoken**.

- `BuildCard` composes the segments **once** and keeps them; the card's `Script` is built from them
  **for display** (the console, the lineup, the operator's "what is about to go out").
- `Speak` for a `LocationReport` **takes the air through the arbiter** — so a takeover suspends it and
  resumes it, which is the whole point — **and starts the existing `synth.Source` over those same
  segments**, returning `Finished` when it ends and `Failed{Routed}` when it is cut short.
- `startSynth`'s body becomes that executor's, called from one place, with the schedule deciding when.
  **The direct path still retires; the player does not.**
- All six gaps stay closed because nothing about the player changed.  G-1 becomes `Finished`, which is
  what the card path already emits.

### Shape B has no cost, and that was worth checking rather than assuming

**The one thing Shape B seemed to need is that the arbiter can suspend and resume a `synth.Source`, not
just a clip.  It already does, in production, today.**  The chain is five calls and every one of them
is live:

| | |
|---|---|
| `app/mastercontrol.go:102` | `dip()` — *"THE ONE CARRIER of 'put the broadcast down'"* |
| `app/radio.go:645` | `duck()` → `engine.Suppress()` |
| `player/engine.go:526` | `giveWayLocked()` |
| `player/engine.go:497` | `StartSource` sets `live=false` — *"a rendered cycle: it waits rather than plays on under an alert"* |
| `player/engine.go:533` | **`return 1, true` — hold: *"a rendered report waits and is heard in full afterwards"*** |

**That is the HUM LEAD's own ruling, already implemented:** *"With the main rotation we can PAUSE the
read, let the alert rail drain, insert a transition read, then resume the read at normal volume.  The
duck is important for the stream because that relay is fundamentally out of our control."*  The engine
decides which treatment to apply **from the source kind, re-read every tick**, so a relay dips and a
rendered report holds — and a fallback from one to the other mid-alert follows the audio.

**So a takeover reading over a running rotation ALREADY pauses the report and resumes it.**  Shape B
does not add that behaviour; it keeps it.  Shape A is the one that would have to re-earn it — and would
have to re-earn it as a NARRATION, where `giveWayLocked` has no case for "a report", because a report
would no longer be a source.

**This is the S0 discipline applied again: the question was answered by reading five live call sites,
with zero code written.**  The estimate that Shape B needed new suspend/resume capability was wrong, in
the same direction as every other estimate this release has checked.

## The red team found something earlier than any of this, and it is fixed

**The `live` stage could never start the station at all.**  The Director begins `Stopped` on purpose, and
the only thing that ever told it otherwise was `setMode`'s transition edge — which on the synthesised
path is reached only from `startSynth`, which `live` does not call.  So the first need arrived at a
stopped Director, `advances(MainTrack)` refused the card, no mode changed, the Director was never
powered, and every subsequent need was refused identically: **a permanently silent station with a
permanently empty lineup and no fault raised**, because nothing failed and nothing was ever admitted.

**The asymmetry was the defect.**  Stop is reported from `Stop`, where the listener acts; start was
reported wherever the audio happened to begin.  `tune` reports it now, before the relay/synth fork, and
a structural gate asserts the send is not inside a branch — because no offline fixture has a live relay,
so "which medium wins cannot change whether the station is on" is a POSITION, and a position is what a
walk can assert.

**This is the second time this exact wire has broken**, and `setMode`'s own comment recorded the first.
The pin written then drove `setMode` directly, so it passed throughout: a pin on the CARRIER rather than
on the RULE cannot see the carrier become the wrong one.  It drives `tune` now.

## THE RULING NEEDED

**Shape A or Shape B.**  It decides what the flip is, and both readings are defensible from the plan's
own words: *"the card then travels the ordinary path — `BuildCard` composes it, `Speak` reads it
through the arbiter"* is Shape A; *"the direct `StartSource` path retires **behind the arbiter**"* is
Shape B.

**My recommendation is Shape B, and the evidence above strengthens it from a preference to a
measurement.**  Shape A rebuilds a working player on the path that carries every ordinary broadcast,
inside the batch already named the most dangerous in the release — **and it would also have to rebuild
the give-way rule**, because a report that is no longer a source has no case in `giveWayLocked` and
would inherit a narration's treatment instead of a rendered cycle's.  The release's stated value
(A1: *"it needs to function as intended out of the gate"*) is not served by re-earning eight behaviours
that already work.

**What Shape B actually changes is small and is exactly the merge:** the SCHEDULE decides when a report
starts, instead of the deck deciding on its own.  Everything about how it then plays, pauses and
resumes is untouched.

**This is the go/no-go the plan requires before P4 begins.**  It is recorded here rather than decided,
because read order, pacing, what the marquee shows and what repeat means are HUM LEAD rulings.

## What is NOT blocked by this ruling

Nothing in P3(a) or P3(c) depends on it — the producer, the event, the card, the staging switch and the
property test are the same under both shapes.  **What changes is only what `Speak` does with a
`LocationReport`**, which is one executor case.
