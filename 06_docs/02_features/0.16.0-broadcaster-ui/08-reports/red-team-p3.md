---
title: "0.16.0 P3 — red team, and the disposition of every finding"
date: 2026-09-09
phase: BUILD
sev: SEV-0
authority: HUM LEAD
status: "ALL TWELVE DISPOSITIONED.  One BLOCKER fixed, three MAJORs fixed, two MAJORs recorded against the open ruling, the rest fixed or recorded with reasons."
---

# P3's own red team

**The plan gave this batch its own red team because it replaces the working path**, on the 0.14.0
precedent for the takeover swap.  It ran against `7cccb50` and `8dfbd48` — the producer and the property
test — in a disposable worktree, and it found twelve things.

**It also found the thing the whole staging existed to prevent, and the staging could not have shown
it.**

## The blocker

> **The `live` stage can never start the station.  Nothing ever plays, and `dark` structurally cannot
> reveal it.**

**Reproduced independently before acting on it**, driving the deck's own emit seam into a real Director:
`power=STOPPED cards=0` after three needs.

The Director begins `Stopped` on purpose — a station comes up silent — and the ONLY thing that ever told
it otherwise was `setMode`'s transition edge, which on the synthesised path is reached only from
`startSynth`.  **`live` does not call `startSynth`.**  So the first need arrived at a stopped Director,
`advances(MainTrack)` refused the card, no mode changed, the Director was never powered, and every
subsequent need was refused identically: **a permanently silent station with a permanently empty lineup
and no fault raised**, because nothing failed and nothing was ever admitted.

**Why `dark` is green on it.**  In dark the order is *tell, then `startSynth`* — so dark loses only the
FIRST report of a session, and `setMode` powers the Director for every turn after.  The staging's central
claim, that the producer's decisions are observable and comparable against the live path's, **is false
for the very first decision**, and dark would have reported success.

**And it is the second time this wire has broken.**  `setMode`'s own comment records the first.  The pin
written then drove `setMode` directly, so it passed throughout: **a pin on the CARRIER rather than on the
RULE cannot see the carrier become the wrong one.**

**The fix is the asymmetry, not the symptom.**  Stop is reported from `Stop`, where the listener acts.
Start is now reported from `tune`, for the same reason and in the same terms, **before the relay/synth
fork** so which medium wins cannot change whether the station is on.  `setMode` no longer reports power
at all, and the transition variable that existed only to guard that send went with it.

**Three behavioural gates and one structural one.**  No offline fixture has a live relay, so "before the
fork" is a POSITION — and a position is what a walk can assert: `TestThePowerReportIsNotInsideABranch`
finds the send in `tune`'s own body and fails if it is nested in anything.  A plant moving it into the
`!live` branch is CAUGHT by that gate and by nothing else.

## Disposition of all twelve

| # | Severity | Finding | Disposition |
|---|---|---|---|
| 1 | **BLOCKER** | the live stage can never start the station | **FIXED.**  Power reported from `tune`, before the fork; `setMode` no longer carries it.  4 gates, 4 plants CAUGHT |
| 2 | MAJOR | both seam AST walks skip package-level declarations — the exact defect they exist to catch | **FIXED.**  Both walk the whole file and name the enclosing declaration whatever kind it is.  **The red team's own escape re-planted and CAUGHT**, naming `var secondSpeaker` |
| 3 | MAJOR | the property test's power dimension is inert — zero events ever stepped against a stopped Director | **FIXED.**  A stop is a state with duration now, and **property 6** was added (a stopped programme does not read), asked of the Director's own power.  **The red team's own plant — the PD-1 guard neutralised — is CAUGHT.**  Reach: 0 → 4,986 events on a stopped station |
| 4 | MAJOR | the property-3 reach gate is not derived, and **the published figure was wrong** | **FIXED AND CORRECTED.**  35 of the 38 "locations read twice" were hazards.  The counter uses the same `isRead` the report counter does, the floor is 20 rather than 1, and the driver was rebalanced until the case is routine: **104**, not 3 |
| 5 | MAJOR | `composeFor`'s comment claims the card and the direct path cannot differ; `Role`, `SelfIntro` and `Pause` are dropped | **COMMENT CORRECTED, GAP RECORDED as G-7** in the flip design.  It is the gap that decides between Shape A and Shape B |
| 6 | MAJOR | `NeedsRead` carries no reason, so `why` has no route to the listener | **RECORDED as part of G-7, NOT built.**  Adding a field nothing consumes is the shape this codebase refuses (AP-DEAD-01), and where the reason should GO depends on the open ruling.  PL-6 governs the event SET, which has grown once and stays once |
| 7 | MINOR | a rotation report issues a band RELEASE for a CUE that never happened, every turn | **COMMENT CORRECTED — it asserted the opposite and was wrong — and filed as F-71, which P4 must close.**  Benign only because the band has one writer today, and P4 is what gives it a second |
| 8 | MINOR | in dark, a card that will never speak briefly holds the air ahead of a hazard | **RECORDED in the dark-run protocol.**  One pump round-trip with no network in it; an observer who sees a report cue and vanish should know it is the stage |
| 9 | MINOR | the stage is read twice per read, at two moments, unpinned | **RECORDED, NOT GUARDED.**  Only `t.Setenv` can make them disagree, and the guard would take the stage away from the tests that must switch it, on a file deleted at P3(d).  The comment that asserted stability now says what is actually true |
| 10 | NIT | a duplicated statement in the property snapshot | FIXED |
| 11 | NIT | property 5's track lookup defaults to `MainTrack` for an absent card | **FIXED.**  Both lookups ask with `ok` now; a card the schedule no longer holds is ABSENT, never main-track |
| 12 | NIT | the dark-decline test's `off` arm proves nothing about `off` | **FIXED.**  The arm is gone; what `off` guarantees is pinned where it is true, at the seam |

## What the red team confirmed

It re-ran three of the twenty-five claimed plants (p1b, n8, n4) **and all three held**, which is evidence
about the log's care rather than proof of the other twenty-two — its own words, and the right framing.

It found **nothing** on double-speak in `off` or `dark` (all three sites traced through the seam, the
goroutine, the lock discipline and the pump), nothing on accumulation or spinning while dark, and
nothing on the rail being caught by the dark decline.  Each of those is a verdict, not an absence: it
said what it looked at.

## What it could not see, in its own words

It ran nothing under `-race`, ran no station, re-checked 3 of 25 plants, and did not review the four
commits that landed while it worked.  **One of its blind spots is now a task**: whether the property
test's unserviced `Tune`/`CueTicker`/`Publish` backlog grows without bound and biases the random choice
away from servicing builds late in a run — which would make the delayed-build interleaving RARER as a
run goes on, exactly opposite to what the test is for.

**That one is now CLOSED, and it was right in substance by a different mechanism.**  The backlog does not
grow without bound — `service` removes whatever it draws, actionable or not — but a `Publish` is emitted
on EVERY settle, so the queue filled with effects the driver could only discard.  **Measured before
fixing: 29,798 service calls, 4,680 of them actionable — 15.7% — with `pending` reaching 39.**  So the
delayed-build interleaving the test exists for was five times rarer than the design intends, and got
rarer through a run, exactly as suspected.

**Fixed by queueing only what the driver can answer.**  The others are dropped because that is what they
are: a publish is a send to a console, a cue and a release are the band's, a tune is the deck's; none of
them answers the Director back.  **The reach roughly quintupled at the same cost:**

| | before | after |
|---|---|---|
| steps | 35,182 | **60,300** |
| readings | 2,073 | **10,207** |
| reports aired | 612 | **6,671** |
| locations read a second time | 104 | **899** |
| events on a stopped station | 4,986 | **8,116** |

**Every plant re-run against the rebalanced driver, and all six still CAUGHT** — including the red
team's own PD-1 plant, both precedence sites, the busy guard, duplicate identities, and the producer
made a no-op.
