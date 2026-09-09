---
title: "0.16.0 — P0 build log: the router"
date: 2026-09-09
phase: BUILD
sev: SEV-0
authority: HUM LEAD
status: "P0 COMPLETE.  Two commits, whole tree green under -race, PTY smoke green."
---

# P0 — the router

**Commits:** `e90e533` the seam · `3ef9f9d` the wiring.

## What it delivered

| Requirement | Status |
|---|---|
| **FR-1.1** the Router is the program's model | **DONE.**  Both `tea.NewProgram` sites |
| **FR-1.2** the Router fans the program-scoped messages | **DEFERRED TO P1, deliberately — see below** |
| **FR-1.3** senders take the Router's send | **A NON-TASK.  Zero changes.  See below** |
| **F-67** the five unhandled program messages | **DISPOSITIONED** — written at the seam, not closed |
| Observer behaviour identical | **DONE**, and tested where it can be: the PTY smoke on the real binary |

## The estimate was wrong in the cheap direction, twice

**FR-1.3 was a non-task.**  The plan said ~9 senders would take the Router's send; its red-team audit
refined that to 11 sites across 6 holders.  **The real number is ZERO.**  `p.Send` sends to the
PROGRAM, and the program delivers to whichever model it holds — it was never model-scoped.

**Verified before claiming it:** nothing type-asserts the model back, and `p.Run()`'s returned model is
discarded.  The reason is now written at the seam in `app/dashboard.go` so it is not re-derived.

**This is the third time this release that a cost or a risk shrank once someone measured it** — after
the request budget read from a library default, and the FIRMS quota.  **The pattern is worth naming: I
am reliably wrong when I reason about a system's shape and reliably right when I measure it.**

## FR-1.2 is deferred, and the reason is not "we ran out of time"

**Fanning a message to one surface is the same thing as delegating it.**  With only Observer in the
tree, a fan-out is unobservable — there is no second surface to receive the copy, so a test could not
tell a fan-out from a delegation.  **A mechanism nothing can observe is a claim.**  It lands in P1,
with the surface that makes it testable.

## What P0 could NOT prove, and how it was covered

**The unit tests cannot see the Router.**  Every frame golden calls `Dashboard.View()` directly, so a
Router that rendered the wrong surface — or nothing at all — would leave all three green.  That is why
P0 built its own pin, and why the real evidence is `make pty-severe` driving the shipped binary through
a pseudo-terminal.

## The bad plant, kept because it is the useful one

The first plant against the byte-for-byte pin **survived**, and the rule says a surviving plant
indicts the plant before the gate.  It was right to: the plant replaced `"Watchpost"` and the frame
renders `WATCHPOST` in capitals.  **The gate was never blind; my plant never arrived.**  Re-planted, it
caught a one-character difference at byte 22 **with the byte count unchanged** — proof it catches a
subtly wrong frame rather than only an absent one.

## Gates at P0 exit

`gofmt` clean · `go vet` clean · **whole tree green under `-race`** · `make pty-severe` green ·
`a2dh validate` 100% (18/18) · `a2dh p10 check --base main` zero findings · branch CI green both
platforms.

## Next

**P1 — the console shell.**  It carries FR-1.2 (now testable), the three lanes read-only from
`Publish`, the breakpoints, the notice at 44 lines, the `--ascii` render, and NFR-3's frame
measurement.
