> **Reconstructed 2026-09-13**, from the commits, the tests and a re-run of the corpus.  It was
> written a month after the work, and that is the finding this document opens with rather than hides.

# P4 — operator intent

**Plan:** `03-architecture-design/implementation-plan.md`, batch P4 — *"FR-3, B1 — the `Moved` event,
the reorder mutator, the card modal, mis-action recovery."*

## The record ran a month behind the code, and that is a defect in its own right

P0 through P3 each closed with a build log and a readiness-gate table, the last of them on
**2026-09-09**.  Everything after it — **118 commits**, rulings **D-32 through D-123** — was gated,
mutant-covered and committed with its reasoning, and **none of it was written into the release
record**.

**Every commit is individually sound; the record is what was missing.**  That distinction matters
because BUILD exit is judged on the record, not on the tree.  On 2026-09-09 a ratified batch was built
and reverted the same day over four blockers, every one already answered in 0.14.0's documents — the
record was complete and was not read.  This is the same failure with the halves swapped, and it is the
more dangerous order: a record that does not exist cannot be read by anyone.

**Why it happened, stated plainly.**  The work after P3 was driven by HUM LEAD UAT rounds rather than
by the batch table, and each round closed with a commit rather than a document.  The batches did not
stop being real; nobody wrote them down.

## The batches as PLANNED and as BUILT

The plan's eight-batch table survived in substance and not in sequence.  The record follows the PLAN's
names, so exit can check one against the other, and says where reality diverged.

| Plan | What actually happened |
|---|---|
| P4 — operator intent | **Split in two, five weeks apart.**  The pure half (the events, the mutator, the Producer/Director split) landed 09-09..09-10 as ruled.  The OPERATOR-FACING half — the controls that emit those events — waited for the layout to settle and landed at **D-118..D-121** on 09-12/13, because a control needs a table to sit in |
| P4.5 — the mock's layout | **Ran twice.**  D-59..D-92 built the layout from mock v2; the HUM LEAD then supplied **mock v3**, and D-93..D-114 rebuilt the two main regions as go-studs tables.  The second pass was not rework of a mistake — v3 is a different design |
| P5 — the bed | Landed across D-76/D-77/D-78 (controls, fence) and completed at **D-117** (relay resolution), which is where it became true rather than plausible |
| P6 — settings | D-92 (a settings row belongs to a surface) and **D-115** (transmitter and service radius exposed, F-87) |
| P7 — instruments | Ran throughout, as planned, and closed with **D-120/D-122/D-123** and `make mutant-anchors` |

## What landed — the pure half (2026-09-09/10)

| Piece | Where | What it does |
|---|---|---|
| `Moved`, `Dropped`, `Restored` | `platform/lineup/` | the operator's three acts, as EVENTS.  Members of the closed set, and `wires` held them exempt until a key pressed them |
| the reorder mutator | `platform/lineup/` | the only thing that reorders.  **`Lineup.Set` refuses to reorder BY DESIGN** — FR-3.3's named trap, and the reason a promote routed through it would update a display while `Next()` answered the old order |
| the discard pile | `platform/lineup/discard.go` | D-35.  **Outside `tracks` on purpose**: `held()` counts tracks, and a pile that counted would mean the schedule never reads as stopped |
| the Producer / Director split | `platform/lineup/` | D-40 — the Producer proposes, the Director chooses.  Two of MVS-D-77's five roles, separated so that "who may put a card on the air" has one answer |
| the line-up as a projection | `platform/lineup/projection.go` | D-44, ruled 09-05 and built here: the line-up is DERIVED from the schedule, never stored beside it |

## What landed — the operator-facing half (2026-09-12/13)

| Piece | Where | What it does |
|---|---|---|
| `[P]` change position, `[k]` drop | `modes/tty/broadcaster_manage.go` | D-118.  Two windows: a number 2-15, and an ARE-YOU-SURE that says where the card goes |
| the move's arithmetic | `modes/tty/router.go` | D-119.  **The typed number is a SLOT; `Reorder` takes a LINE-UP INDEX**, and `liveOffset` differs by one on standby (D-84).  A move landed one slot low until this was written down |
| the card window owns the keyboard | `modes/tty/router.go` | **D-121, and it is the batch's most instructive defect** — see below |

## The defect worth keeping: a rule stated in the wrong place is a comment

D-118 said *"an open question owns the keyboard"* and put the handler BELOW the keymap switch — where
`enter` is bound to `actQueueOpen`, whose case closes the card window and returns.  So the operator
typed a position, pressed enter, both windows closed, **no move was sent**, and the typed number was
never cleared.

HUM LEAD, UAT 2026-09-13: *"Both Modals close / Table does not update / redraw … my entry is
remembered, but it doesnt seem to propogate correctly."*

**AND TWELVE TESTS PASSED AGAINST THAT BUILD.**  The helper `press` called `cardWindowKey` DIRECTLY,
so every one of them measured the handler while the operator drives the ROUTER.  P-1 says it in one
line: a seam a test can drive is not a covered seam unless it is the seam the KEY takes.  `press` goes
through `Update` now, and was run RED against the old placement before it was trusted — three
failures, including the redraw test written specifically for this defect and passing anyway.
