---
title: "0.16.0 — a SEQUENCING constraint on the swap gate, not a new finding"
date: 2026-09-09
phase: BUILD
sev: SEV-0
authority: HUM LEAD
status: "RE-GRADED 2026-09-09.  The first draft reported as new what 01-objectives had already ruled.  What survives is one sequencing constraint."
---

# Most of this was already ruled, and I should have read it first

**The first version of this document reported three findings.  Two of them are ratified requirements
and a recorded risk**, and re-deriving them from the code was the error, not the code.

| What I "found" | Where it was already written |
|---|---|
| ON AIR / STANDBY does not really assert a carrier | **C-4** measured it: *"ON AIR as a real continuous state needs a primitive the codebase does not have — which is exactly why R-4.1 is scoped to mock."*  **FR-5.2** states it |
| the swap gate depends on a state the console cannot yet change | **RS-3**, MED, already on the risk register: *"a mocked ON AIR drifts from the real audio state while the swap gate depends on it."*  Mitigation recorded: **FR-5.1** reads the Director's power, never a UI flag — which is what the console does |
| there is no control to reach STANDBY | **FR-5.4** requires one, and says in as many words that it *"is a requirement rather than a mock detail"* — added by the business lens at the DISCOVER red team, precisely because promote and demote had FRs and the state toggle had none.  **It is not built yet.  That is a batch not yet run, not a defect** |

## What survives, and it is one sentence

**`Router.keys` is never assigned**, so `ctrl+o` and `ctrl+b` reach nothing in a running build
(`modes/tty/router.go:76` builds the Router; `:156` guards the swap on `r.keys != nil`;
`broadcasterKeyMap()` at `:53` has no caller).

**That is the only thing standing between today's build and RS-3 becoming live.**  `canSwap` correctly
enforces the ratified STANDBY-first rule (**FR-1.4**, **D-1**) against the Director's REAL power — which
Observer sets to `Running` on every tune — while **FR-5.4**'s control, the release valve, does not exist
yet.  Install the keymap first and the console is a one-way door: arriving is allowed by design
(*"leaving a LIVE station is the hazard"*), leaving is refused, and nothing on the surface can act.

## The constraint

**The keymap and FR-5.4's control land in the SAME change, never the keymap alone.**

It is a sequencing rule between two batches that already own their halves — P4/P4.5 own the console's
input, P5 owns the station state — so it belongs nowhere except written down, which is what this
document is for.

**The mock already draws the affordance that makes arriving safe:**
`[ SHIFT + ENTER ] GO TO STANDBY`, on the console itself.

## And one thing the brief already owes that this touches

**FR-5.5** requires the ON AIR boundary to be stated where the operator reads it — *"Watchpost has no
radio path: it produces audio, and a separate transmitter the application cannot observe puts it over
the air."*  **The console's help and About text do not exist yet**, so that exit criterion is open, and
it is the same sentence that makes the whole surface honest: the banner confirms **the software is
putting programme out**, never that the antenna is radiating.
