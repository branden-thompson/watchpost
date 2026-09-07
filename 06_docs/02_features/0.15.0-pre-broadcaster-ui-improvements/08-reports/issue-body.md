# v. 0.15.0 — Pre-Broadcaster UI Improvements

## Bottom Line Up Front / BLUF

0.15.0 closes the design, performance, and structure items that the Broadcaster UI would otherwise inherit and duplicate.  Those items affect both surfaces — the Observer experience shipping today, and the operator console that follows it — so the work costs less once, now, than twice, later.  Broadcaster UI moves to 0.16.0.

## Antecedent

0.14.0 shipped with alerts silent on Linux (#7).  That defect passed four rounds of red-teaming, 171 mutation tests, and a 127-row exemption ledger, and it survived eight days inside a tagged release.  No gate caught it.  A human operator caught it, on real hardware, by waiting for a live alert and hearing the tone play with nothing behind it.

That is the characteristic failure of this product.  Watchpost rarely breaks loudly.  It breaks by omission, and a station that has stopped speaking is indistinguishable from an afternoon with no weather in it.

## The problem

> **A person relying on Watchpost for hazard awareness cannot tell a quiet day from a station that has silently stopped telling them things.**

Every item in scope traces back to that sentence, which is the difference between a release and a task list:

- Two implementations of one operation is *how* a station goes quiet.  #7 was precisely this: four call sites asked which voice to use, and one of them asked incorrectly.
- A window keyed on position rather than identity is *how* a stale screen persists.  #11 was this, and two frozen windows preceded it.
- Contrast and plain-text checks that do not cover every window are *how* a display degrades unobserved.

The statement also supplies a scope predicate: **an item whose failure mode is loud is probably not 0.15.0.**

## Measures of success

| Measure | Today | Target |
|---|---|---|
| Elapsed time to confirm the station functions, on the operator's own machine | Unbounded; contingent on live weather | Under 2 minutes |
| Operations implemented more than once, with no written and ratified reason | At least 9 identified during 0.14.0 | Zero |
| Window-memory keys with no guard against drift | Most of them | Zero |
| Quality gates never observed failing | Several | Zero |

The first measure carries an obvious anti-solution: a diagnostic that runs on its own path and reports success while the production path is broken.  That is an exact description of #7.  The constraint that follows is that a test event must traverse the same path a live alert traverses.

## In scope

**1. Single ownership of shared rules.**  Six independent call sites write the configuration file.  Three classifiers read the same weather product strings and already disagree; "Coastal Flood Statement" is classified two different ways today.  The audio arbiter exposes three functions that touch the voice under no lock at all.  Broadcaster adds a second writer to every one of them.  *(F-1, F-2, F-22, F-34)*

**2. Window memory keyed on identity, not position.**  A window decides whether to redraw by comparing a compact key.  Several of those keys encode "row 3" where they should encode "Miami," so two distinct subjects occupying the same slot compare equal, and neither redraws.  Three user-visible defects have come out of this one class.  The Broadcaster operator surface consists almost entirely of cursors and rows, which multiplies the exposure rather than holding it constant.  *(#12, F-30, F-53)*

**3. On-demand verification of the station.**  A `ctrl+d` diagnostic surface injects an event and exercises the real machinery: audio out, the news ticker, and the event reports.  Test audio carries an opening and a closing announcement on the American emergency-broadcast pattern.  Test events expire unattended inside two minutes, so they cannot contaminate live data for any meaningful duration.  This is the item that addresses the problem statement directly, and the Broadcaster operator requires it to verify audio out before going to air.  Two prerequisites ship alongside it: the window's lower region is unreachable on a 24-row terminal, and the built artifact needs a check proving the injector is absent from it.  *(#9, F-21, F-35, F-38, F-26, F-9)*

**4. Gates capable of failing.**  `golangci-lint` and `staticcheck` have never run as a gate in this repository.  The contrast register contracts silently whenever a token is added.  Four windows fall outside the plain-text scan, including the most frequently opened one.  One step of the automated journey cannot fail, which is why it is the step that keeps failing.  Every gate exits this release carrying a recorded instance of a deliberate failure.  *(F-15, F-18, F-37, F-47, F-44)*

**5. Silent defects with user-visible consequences.**  A location that never returns data remains in RECENT indefinitely, indistinguishable from one still loading.  The *All Reports* voice picker is absent from Settings, despite the changelog promising it.  Three cast roles are reachable only by hand-editing the configuration file.  The relay-fault window closes itself after ten seconds — a hard timeout on the only station-tuning control, which fails WCAG 2.2.1 Level A outright — and it produces no audio at all.  F-43 remains open: a tone played three times with no words behind it, never reproduced.  A tone is a promise of words.  *(#13, F-46, F-5, F-36, F-33, F-43)*

**6. Rulings, not migrations.**  Four determinations that the next six data feeds will copy: whether the release check constitutes a data provider, whether caches consolidate under `~/.watchpost/`, ratification of 23 stale ledger rows, and a survey separating prose from evidence in the documentation tree.  **No files move in this release.  These are the determinations, not the work.**  *(F-3, F-49, F-45, F-12)*

## Out of scope

- **F-40, the Brengel Fire blindness.**  A live evacuation order and its fire were both invisible on 6 September.  It is the most severe instance of the problem statement anywhere in the backlog, and it receives its own release ahead of Broadcaster: the fire feeds require their own investigation, and folding that investigation into 0.15.0 would consume the release.
- **Broadcaster UI itself.**  #10, now 0.16.0.  Sequence: 0.15.0, then 0.15.x for the fire work, then 0.16.0.
- **Sixteen further follow-ups**, deferred with reasons already on record.  Most of them fail loudly, which is the predicate.
- **Relocating documentation to GitHub Pages.**  Survey only.  The prose-versus-artifact distinction is not clean yet: eight run records inside those folders were excluded from git entirely until 7 September.

## Open questions

- Duration of the relay-fault window, and whether a keypress resets it.
- Key binding for STOP ALL, and whether it belongs in the diagnostics window or on the masthead.
- Whether the release check is a data provider.
- Whether caches relocate, and whether the migration accompanies the ruling or follows it.
- Timebox for F-43, and the written disposition if it is not reproduced within that timebox.
- Frame cost of consolidating the two memo types.  Blocked on the performance protocol running against Arch hardware; that measurement gates several decisions here.

## Notes

- SEV-0.  Full workflow: DISCOVER, PLAN, BUILD, REVIEW, VALIDATE, SHIP, and REFLECT, with full git, reports, diagrams, and TDD.
- The working brief — every requirement with its source, the discovery handoff, the risk register, and the anti-solution analysis behind each measure — sits in the repository at `06_docs/02_features/0.15.0-pre-broadcaster-ui-improvements/08-reports/project-brief.md`.
- This issue will be updated as items surface.

## Related

#9, #12, and #13 fold into this release.  #10 is the Broadcaster UI at 0.16.0.  #7 and #11 are the defects that shaped it.
