---
title: "0.16.0 — the console is unreachable, and the door it will open is one-way"
date: 2026-09-09
phase: BUILD
sev: SEV-0
authority: HUM LEAD
status: "FOUND 2026-09-09 by the HUM LEAD's question: \"how will I as an operator run the radio?\"  NOT FIXED — the fix needs a ruling."
---

# There is no operator UI yet, and the one that is half-built has a trap in it

**Found by a question, not by a gate**, which is worth recording on its own: *"how will I as an operator
run the radio?"*  The answer is **you don't — you run it from Observer, exactly as in 0.15.0** — and
finding out why turned up a defect nothing was watching for.

## The three facts, verified

| # | Fact | Where |
|---|---|---|
| **1** | **`ctrl+o` and `ctrl+b` do nothing in a running build.**  `broadcasterKeyMap()` is defined and **never called**; `Router.keys` is never assigned, and `Update`'s swap branch is guarded by `r.keys != nil` | `modes/tty/router.go:53`, `:76`, `:156` |
| **2** | **The console accepts no input at all.**  `Broadcaster.Update` handles four messages — window size, background colour, lineup, station power — and no key presses | `modes/tty/broadcaster.go` |
| **3** | **Nothing in the tree ever produces `lineup.OffAir`.**  The only two power changes are `radioDeck.tune` → RUNNING and `radioDeck.Stop` → STOPPED, both driven from Observer | `app/radio.go:204`, `:819` |

## The trap, which is latent only because of fact 1

`canSwap` refuses to LEAVE the console while the station is running:

> `"the station is ON AIR — go to STANDBY before leaving the console"`

**That is the HUM LEAD's own ruling (QQ-1, "Require STANDBY before swap") and it is correctly
implemented.**  The problem is the other half:

1. Tune a location in Observer → the console's power becomes RUNNING.
2. Press `ctrl+b` → **arriving is allowed**, deliberately: *"leaving a LIVE station is the hazard"*.
3. Press `ctrl+o` → **refused**.
4. Press `shift+enter` to reach STANDBY → **nothing is listening**, and nothing can produce `OffAir`
   anyway.

**The operator is on a surface with no controls, and no way off it but quitting the process.**

**It cannot happen today**, because step 2 cannot happen: no key reaches the router.  **It happens on
the first commit that installs the keymap** — which is P4's or P4.5's — and it will look like a
different batch's bug.

## Why no gate caught it

**Every gate asked the right question about the wrong subject.**  `canSwap` has thorough tests, and they
are correct: they construct a `Router` directly and assert the precondition.  **Nothing asserted that a
`Router` a PROGRAM builds can swap at all** — `NewRouter` is the seam, and it leaves `keys` nil.

**This is the CARRIER-not-RULE shape again**, the third instance this release: a test that drives the
function under test directly cannot see that nothing in production reaches it.  The router-owner walk
asserts the program's model IS a `Router`; it says nothing about whether that Router can act.

## The ruling needed before this is fixed

**Wiring the keymap is one line.  What `shift+enter` DOES is not.**

| Question | Why it is yours |
|---|---|
| **What does going to STANDBY do to a read already on the air?**  Cut it dead, let it finish, or fade? | It is a pacing and read-order call, and the operator hears it |
| **Does STANDBY stop the rotation, or only take it off the air?**  `OffAir` already means "dead air holds everything, the rail included" in `advances` — so a takeover would be held too | An evacuation order held because the operator stepped away is a safety decision, not an implementation one |
| **Should `ctrl+b` be refused while ON AIR as well**, until there is something on the console that can act? | Refusing to ARRIVE contradicts the ratified rule; leaving the trap open contradicts common sense.  A third option — allow arrival and offer the STANDBY control on the console — is what the mock draws |

**The mock already answers the third one:** it shows `[ SHIFT + ENTER ] GO TO STANDBY` on the console
itself, which is the affordance that makes arriving safe.  **So the fix is to build the control, not to
narrow the door** — but the first two questions have to be answered before it can be built.

## Recommendation

**Do not install the keymap until the STANDBY control exists in the same change.**  A commit that makes
the console reachable without one makes the trap live, and it would be the kind of defect that is
obvious in hindsight and invisible in review — the console would look finished.
