# Key learnings — severe-alerts-modals (0.13.0)

Captured as the phases closed (1–10 through REVIEW), completed at REFLECT (11–16). The after-action report is
`08-reports/debrief.md`; this file is the reusable part — patterns to keep and anti-patterns to avoid.

## Patterns to keep

1. **A fix that passes against the fake is not a fix.** The takeover-pauses-a-read behaviour was UAT-approved
   against the voice fake; the real engine closed the paused line the moment the takeover's tone took the one
   preview slot (round 4 A-01). Every narrator/engine contract now has a test on the recording output as well as
   the fake — and the R6 audio smoke is blocking at VALIDATE whenever the radio path changes.
2. **Write the arbiter's protocol as one owner.** Two protocol holes (a cancelled suspended job; a line starting
   under a takeover) came from "who has the air when it frees" living in two places. `settle` is the one owner;
   the collision loop and the cancel-while-suspended case are pinned.
3. **Search and slice the same string.** `ToUpper` can change byte lengths; a panic from feed prose reached
   the publish goroutine (A-05). Every index into provider text now comes from the string it indexes.
4. **Bare SGR numbers are ambiguous** — `250` is a 256 index, `97` is bright white, a truecolour token is
   neither. Tokens carry full parameters; a box with no tone of its own carries no SGR at all (B-01, D-9); one
   function (`render.FgSGR`) composes a foreground — the missing one was the root cause of Watchpost Light's
   dark grounds, and fixing it lifted every theme.
5. **Colour is never the only carrier** (R-12a): the alert class in text with colour off / `--ascii`; the
   severity by tab and tint; the class token pins restored when the module was redesigned (B-04).
6. **Pins should say what the behaviour is, not what the output was.** Fixed label offsets broke on centring;
   "centred over its column" survives the facelift. The same for the header ladder (order, not widths) and the
   80×24 frame (never exceeds the terminal, not "23 lines").
7. **Real time in a concurrency test is a coin.** Gate the sequence (a channel the test opens), poll a condition,
   never sleep a fixed interval (D-8, A-16c) — and **never wait for the first `done` when there can be two**:
   the publisher counter test read after one completion while a second publish was in flight; green on macOS
   for a month, red on the loaded ubuntu runner (PR #4). Wait for the condition, bounded (`waitUntil`).
8. **Budgets catch design regressions.** The eager chip forms (+170), the header's compositor cost (+740) and the
   thin-bands double render (−350 when fixed) were all found by the alloc pins, not by eye.
9. **The ledger is a document, not a checkbox.** Four P10 rows' reasons predated the code they absorbed; the rows
   are presented with their reasons for ratification, and the gate ledger is re-run at every exit (C-02/C-06).
10. **Mocks are exact, colours are the HUM LEAD's pass** — and an AA floor can be a render-time fact
    (`LiftToAA` / `withAA` over a register of painted pairs) rather than a hand-tuned value per theme, keeping
    the hue's intention.
11. **Decide at the press, not at wiring time** (VALIDATE V-2). A guard evaluated at configuration
    (`deck != nil`) was permanently false because the deck attaches later; the hook now reads the state when it
    fires. Late-bound dependencies are read late.
12. **The real input layer is part of the product** (V-1, D-2). A pty delivers `esc` fused with the next key and
    a shifted letter as a modifier with no text; two P1s came from that seam. The fresh-HOME journey script
    drives every key through the real binary on every VALIDATE — keep it the first thing run after a fix.
13. **The ship shape: squash onto the publish branch, PR, tag the merge.** One `commit-tree` squash of the
    feature tree with the publish branch as parent keeps private history private and gives the PR one commit
    to review; CI runs on the push and on the PR; the tag goes on the merged commit; rollback is tested
    *before* the tag with the previous release's pinned install.

## Anti-patterns to avoid

14. **Claiming a gate the Makefile does not run.** The PR template's lint box surfaced five nits at SHIP; a
    box is a claim. Fold the linters into `make verify` (`make lint`), restore `go.sum` after them (they resolve
    with `-mod=mod` and the tidy gate is strict).
15. **A credential checked for login, not for permission.** The personal fine-grained token could push and
    read but not create a PR; two 403s at the ship. The pre-ship check reads the token's permissions
    (*Pull requests: write*, *Contents: write*) against what the ship will do.
16. **Copying a gate's report after the gate has run.** `make p10` → `cp` the report → `a2dh validate` marks the
    run record stale because the copy changed the tree. Commit first, run the gate on the clean tree, keep the
    committed copy from the *clean* run. (Two P10 quirks to report upstream: the density row toggles between
    identical clean-tree runs; on the trunk with a docs-only change the ledger's rows read "unmatched".)
