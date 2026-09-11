# D-79 — ANYTHING THAT TALKS BACK TO THE PROGRAM MUST NOT RUN ON THE UPDATE LOOP

**HUM LEAD, UAT 2026-09-11:** *"./dist/watchpost-diag is majorly broken: complete freeze/hang once
ctrl+b or B is pressed — to the point I have to kill the terminal."*

**Mine, and shipped.**

---

## The dump said it in one frame

```
1 runtime.gopark
  charm.land/bubbletea/v2.(*Program).Send (inline)
  app.(*radioDeck).onStatus
  player.(*Engine).set / halt / Halt
  app.(*radioDeck).Stop
  app.(*livePipelines).silenceMonitor
  app.(*livePipelines).takeTheAir
  modes/tty.Router.swapTo / update / Update
  app.RunDashboard
```

Taking the air stops the monitor's audio.  Halting the player calls back through `onStatus`, which
`Send`s to the program — **and the program is inside `Update`, which is where this call came from.**
It cannot receive.  Nine more goroutines were piled up behind it on the same `Send`.

## The answer was already in the codebase

```go
return d.withCmd(func() tea.Msg { radio.Stop(); return nil }), true
```

Observer has stopped the radio through a **tea.Cmd** since it had a radio — Bubble Tea runs commands
on their own goroutines, off the loop.  I called `Stop()` directly from a callback that runs inside
`Update`, which is the one place it cannot be called from.

`OnSurface` and `StepBedRelay` return a `tea.Cmd` now, and the swap carries it out through `swapTo`.

**What stays inline is deliberate**: recording the owner, declaring the air and nudging the rail are
non-blocking and must be TRUE by the time the next frame draws.  Only the two things that TALK are
deferred.

## It was TWO freezes, and the second was the same defect one function along

Fixing `silenceMonitor` left `ctrl+b` working and **the first arrow press froze it again** —
`stepBedRelay` tunes a relay and publishes the row, both of which reach the program.  Isolated by
bisecting the key sequence on the real binary:

```
no keys        : QUIT-OK 4 ms
ctrl+b         : QUIT-OK 8 ms
ctrl+b, b      : QUIT-OK 14 ms
b/o round trip : QUIT-OK 5 ms
arrows on obs  : QUIT-OK 6 ms
console+arrows : FROZEN          ← the second one
```

## And a third defect the same commit introduced

The bed's keys are looked up by the ROUTER, before either surface sees a message — so the cases ran
on **every** surface.  They returned unconditionally, which **swallowed `left` and `right` on
Observer**, where they walk the alerts.  The listener's navigation simply stopped working, and no
test saw it because every one of them pressed the arrows on the console.

Not returning IS the fall-through: execution continues past the switch to the active surface, which
is exactly what an unbound key does.

## What this says about the gates

**Every unit test passed throughout all three.**  They call the handler directly, where there is no
event loop to deadlock against and no Router to swallow anything — the seam was driven and the
DELIVERY was not.  That is this release's recurring finding, and here it cost the operator his
terminal rather than a wrong pixel.

`verify` runs no pty at all.  `journey` runs one, and drives Observer.  **Nothing drives the
console's keys**, and any future console control that talks back to the program can reintroduce this
exactly — filed as **F-90**, with the recipe that works and the reason a fresh HOME does not.

**The half-written gate was deliberately NOT left in the tree.**  A gate that fails for harness
reasons trains people to ignore it, which is worse than not having one.
