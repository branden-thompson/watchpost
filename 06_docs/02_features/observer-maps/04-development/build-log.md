---
title: "0.18.0 Observer maps — BUILD log"
date: 2026-09-25
phase: BUILD
sev: SEV-0
authority: HUM LEAD
status: "LIVE — one entry per batch: what landed, the tests that hold it, the mutation verdicts, and what the gates found."
---

# BUILD log

## Batch 1 — the map window opens and draws in Update (2026-09-25)

**Tasks:** W1.1, W1.3, W1.7, W2.1, the first half of W1.2 and of W2.2, on go-tuiMaps
`v0.2.0-rc.8` (D-60). **Ruled during it:** D-60 (P1-a on the release candidate), D-61 (the map
window owns the keyboard).

**What landed.** `g` (`map.toggle`, in Help's NAVIGATE group) opens and closes the map window; esc
closes it. The composition root hands the window `Config.NewMap(size)` (`app/maps.go`), which the
window calls on the first `g` with the window's size — the library moves only a sized map. It names
no source: the embedded tiles draw the basemap and nothing is reached (FR-3.2). The map opens at
zoom 6 on the selection. It is drawn in `Update` into `mapPane` and `View` only prints (D-41);
`Work` runs as a command and its `mapWorkedMsg` is drawn by the `Update` it reaches. With no
selection the window says so and builds nothing (FR-1.3). Under `--ascii` it says the picture is
braille and names the remedy (FR-1.7, FR-1.8) until W1.4's description replaces that line.

**The closed-set guards found four real holes the moment the window joined the enum (W1.7):**
1. **A stale frame.** The window's memo key did not include the pane, so a frame drawn before the
   tiles landed was replayed after they landed — the failure D-45 exists to prevent. The pane now
   raises a generation on every draw and the key carries it.
2. The body ran into the border: the map is now drawn three cells inside each side.
3. The map was not redrawn on resize: `tea.WindowSizeMsg` redraws it (D-45's size row). The
   reachability guard used to assign the size fields directly; it now reaches 80×24 through the
   resize message, as a terminal does.
4. `mapWorkedMsg` had no route: it is an answer owed to Observer's window (`observerScoped`).

**Mutation verdicts** (each restored and compared):

| # | Mutation | Verdict |
|---|---|---|
| M1 | the memo key without the draw generation | caught — `TestTheMapIsDrawnInUpdate` |
| M2 | resize not redrawn | caught — reachability at 80×24 |
| M3 | a landing not drawn | caught — `TestTheMapIsDrawnInUpdate` |
| M4 | View draws the map | caught — `View called the library: [Render]` |
| M5 | a map built with no selection | caught (a crash in the no-selection test) |
| M6 | braille under `--ascii` | caught — `TestASCIIFramesCarryNothingButASCII` |
| M7 | the reply not routed to Observer | caught — `TestEveryWindowReplyIsRoutedBackToTheWindow` |
| M8 | not centred on the selection | caught — `TestTheMapIsCentredOnTheSelection` |
| M9 | no key bound | caught — `TestTheMapKeyOpensAndClosesTheWindow` |
| M10 | no inset | caught — the margin survey |
| M11 | the generation not raised | caught — `TestTheMapIsDrawnInUpdate` |

**What the gates found.** `TestDeclarationSetUnchanged` (a new top-level name in `app`, written
into its golden); `mutant-anchors` (mCA1's anchor, re-pointed at the case `mapWorkedMsg` joined).

**For UAT.** The library refuses to move a map that has no size yet (its P-56), so a host must
build the map at a size before it can place the view. Watchpost does; whether the library should
place a view given before the first size is a question for the paired release.

**Owed from these tasks:** W2.2's goroutine record and the `NextCall` tick; W2.6's join on close
(`closeMap` exists, the app does not call it yet).
