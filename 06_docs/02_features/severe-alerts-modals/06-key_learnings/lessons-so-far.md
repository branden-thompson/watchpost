# Key learnings — severe-alerts-modals (0.13.0), through REVIEW

Captured as the phases closed; REFLECT (the DEBRIEF) completes this folder with the after-action report.

1. **A fix that passes against the fake is not a fix.** The takeover-pauses-a-read behaviour was UAT-approved
   against the voice fake; the real engine closed the paused line the moment the takeover's tone took the one
   preview slot (round 4 A-01). Every narrator/engine contract now has a test on the recording output as well as
   the fake — and the R6 audio smoke is blocking at VALIDATE because the radio path changed.
2. **Write the arbiter's protocol as one owner.** Two protocol holes (a cancelled suspended job; a line starting
   under a takeover) came from "who has the air when it frees" living in two places. `settle` is the one owner;
   the collision loop and the cancel-while-suspended case are pinned.
3. **Search and slice the same string.** `ToUpper` can change byte lengths; a panic from feed prose reached
   the publish goroutine (A-05). Every index into provider text now comes from the string it indexes.
4. **Bare SGR numbers are ambiguous** — `250` is a 256 index, `97` is bright white. Tokens carry full parameters;
   a box with no tone of its own carries no SGR at all (B-01, D-9).
5. **Colour is never the only carrier** (R-12a): the alert class in text with colour off / `--ascii`; the
   severity glyph by count; the class token pins restored when the module was redesigned (B-04).
6. **Pins should say what the behaviour is, not what the output was.** Fixed label offsets broke on centring;
   "centred over its column" survives the facelift. The same for the header ladder (order, not widths) and the
   80×24 frame (never exceeds the terminal, not "23 lines").
7. **Real time in a concurrency test is a coin.** Gate the sequence (a channel the test opens), poll a condition,
   never sleep a fixed interval (D-8, A-16c).
8. **Budgets catch design regressions.** The eager chip forms (+170), the header's compositor cost (+740) and the
   thin-bands double render (−350 when fixed) were all found by the alloc pins, not by eye.
9. **The ledger is a document, not a checkbox.** Four P10 rows' reasons predated the code they absorbed; the rows are
   presented with their reasons for ratification, and the gate ledger is re-run at every exit (C-02/C-06).
10. **Mocks are exact, colours are the HUM LEAD's pass** — and an AA floor can be a render-time fact
    (`LiftToAA`) rather than a hand-tuned value per theme, keeping the hue's intention.
