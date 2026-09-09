---
title: "0.16.0 — S0 SPIKE RESULT: the audio merge"
date: 2026-09-09
phase: BUILD
sev: SEV-0
authority: HUM LEAD
status: "ANSWERED, inside the box.  No spike code was written; nothing to dispose of."
---

# S0 — the audio merge: ANSWERED

## The question

> **Can the rotation's reads travel the card path — `Propose → BuildCard → Speak` through the narrator
> arbiter — instead of `Tune → startSynth → engine.StartSource`, without a window in which both paths
> can speak?**

## Output 1 — the answer

# **YES — architecturally.  The window is TRANSITIONAL, not inherent.**

**And it is answered without writing a line of spike code**, which is worth saying plainly: the
behaviour D-24 asks for is already implemented, already tested, and already green.  The charter
budgeted a session and provided for disposing of throwaway code; **there is none to dispose of.**

### The evidence, in the order it convinced me

**1. The arbiter admits exactly ONE job to the air.**  `admit` (`app/director.go:312-327`) holds a
single `d.onAir`; everything else queues in `d.waiting`.  **Serialisation is by construction, not by
convention.**

**2. A higher class suspends a lower one, and the lower resumes.**  Same function, lines 317-324: an
arriving higher-class job sets `on.suspended = true`, pushes it onto a stack, and calls
`mc.holdLine()`.  `awaitAir` (`:233-241`) is the condition-variable wait that resumes it.

**3. The code already distinguishes a suspension from a listener's hold** — `suspended` resumes on its
own; `paused` waits for a person (MVS-D-74, `director.go:106-110`).  That distinction is exactly what
an operator-paused main track needs, and it is already drawn.

**4. THE KNOWN CASE — and this is what answers the unknown one (INST-4).**  The severe-event read
(`app/severe_read.go:274`) **already runs through this arbiter as the lower class** and is already
suspended and resumed by a takeover.  **Eight tests pin it, all green:**

```
--- PASS: TestDirectorBreakingSuspendsAndResumesARead        (0.71s)
--- PASS: TestEventReadIsSuspendedByABreakingTakeover        (0.01s)
--- PASS: TestReadCancelledWhileSuspendedDoesNotWedgeTheDirector
--- PASS: TestReadCancelledWhileParkedSuspendedReturnsAtOnce
--- PASS: TestAPausedReadReleasesTheBedAndResumesOnRequest
--- PASS: TestATakeoverEndingDoesNotResumeAPausedRead
--- PASS: TestNothingOutranksATakeover
--- PASS: TestTheResumeTransitionDiffersOnlyByWhetherTheProgrammeKeptRunning
```

**The last one is D-24's own distinction, already tested** — the resume transition differs by whether
the programme kept running, which is the relay-rejoined versus synth-resumed split the HUM LEAD
described and `transition/resume.txt`'s `.InProgress` encodes.

**5. Adding a rotation class is safe by construction.**  `narrationClass` has two members and a
sentinel, and `TestNothingOutranksATakeover` **walks the type** rather than a hand-written list —
its own comment records that an earlier version iterated a list and stayed green when a class was
added.  **A third class below `narrateRead` is caught by the existing guard.**

## Output 2 — the double-speak window, characterised

**It is not where I expected, and it is not inherent.**

| | |
|---|---|
| **Does the merged end-state have a window?** | **No.**  One arbiter, one `onAir`, lower classes suspended.  Two narration jobs cannot speak at once |
| **Where does the window actually exist?** | **Only while BOTH paths are live** — the direct `startSynth` and the card path wired at the same time |
| **What closes it?** | **Retiring the direct path in the SAME change that adds the card path.**  Not a guard; a deletion |

**Today's coexistence is not double-speak.**  The arbiter *ducks or holds* the broadcast beneath a
narration — that is the designed give-way of D-24, not two speakers.

## Output 3 — the revised P3 estimate

**Bigger than "wiring", smaller than "a re-shape".  The reason is a number I did not have before.**

**`startSynth` has THREE production callers, not one** (`app/radio.go:182, 789, 974`):

| Caller | What it is |
|---|---|
| `:182` | The rotation's tune — **the expected one** |
| `:789` | **"relay unavailable"** — a fallback when the relay fails |
| `:974` | **"the relay was silent"** — a fallback when the relay goes quiet |

**Two of the three are relay-failure fallbacks that bypass the schedule entirely.**  Under the merge, a
relay dying mid-broadcast must produce a *card* rather than starting synth directly — and that is a
genuine behaviour change on a failure path, not wiring.

**Revised estimate:**

| | |
|---|---|
| The rotation's own tune becoming a card | **Wiring.**  The arbiter, the classes, the suspend/resume and the transition all exist |
| The two relay-failure fallbacks | **Real work.**  A failure path that currently sidesteps the schedule has to enter it, and it is the path that runs when something is already going wrong |
| The double-speak guard FR-2.5 demands | **Cheap** — the arbiter provides it; the test asserts it rather than builds it |

**The HUM LEAD's read was right** — *"we should have most of this infra built"* — and the part that is
not built is the failure path, which is the usual place.

## What P3 must now do that the plan did not say

1. **Retire `startSynth`'s direct path in the same change** that adds the card path.  The window is
   transitional and this is what closes it.
2. **Route both relay-failure fallbacks through the schedule**, so a dying relay produces a card.
3. **Add the third narration class below `narrateRead`**, and let the existing sentinel-walking guard
   cover it.
4. **Assert, do not build, the no-double-speak property** — the arbiter already provides it.

## Disposal

**No spike code was written.**  The question was answered by reading the arbiter and running eight
tests that already existed.  The worktree is removed; nothing was committed to the feature branch from
it.

**INST-5 — what this method could not see:** the spike proved the arbiter's *structure* serialises and
that the known case is green.  **It did not run the merged path**, because that path does not exist
yet.  A structural answer plus a passing known case is strong evidence and is not a measurement of the
merged system.  **The timing property test P3 carries is still owed and is not made redundant by this
result.**
