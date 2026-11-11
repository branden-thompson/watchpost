---
title: "0.19.5 DISCOVER — what the five lenses found, checked"
date: 2026-11-06
phase: DISCOVER
sev: SEV-0
status: "DISCOVER exited (D-19); every ruling it asked for is in rulings.md"
---

# What the five lenses found

Five read-only lens agents surveyed watchpost at `4c803d99` (v0.19.3), one per goal (D-3). Their reports
are filed whole beside this page:

| Lens | File | Findings |
|---|---|---|
| RN, the regression net | `lens-RN-regression-net.md` | 13, and a minor list |
| PF, performance | `lens-PF-performance.md` | 11, and a minor list |
| DR, DRY and consolidation | `lens-DR-dry-consolidation.md` | 16, minor, rejected candidates |
| DC, docs and comments | `lens-DC-docs-comments.md` | 17, and a minor list |
| JR, junior readability | `lens-JR-junior-readability.md` | 13, and a minor list |

The scratch tools behind them (coverage profiles, AST scanners, the PF benchmarks and prototypes) are
kept outside the tree, in the evidence folder named in the PLAN report.

## How the findings were checked

D-3 says no finding counts until it has been checked against the code. Two passes do that:

1. **At DISCOVER**, the agent checked every finding that is a defect or would need a ruling, and a
   sample of the rest, by reading the code at the lens's `file:line` (and for PF, by re-running its
   benchmarks). Each held:

   | Finding | What was checked | Result |
   |---|---|---|
   | PF-1 to PF-5 | the lens's benchmarks re-run on its pristine copy (equal to the tree) and on its prototype copy | reproduced exactly: frame 89,227 → 67,635 B; Router tick 60,384 → 37,216 B; console frame 503 → 355 allocs |
   | RN-3 | `app/severe_test.go:176-185` loads with a one-hour window | the cap is never reached |
   | RN-4 | `modes/tty/request.go:449` slices bytes; the other seven fields slice runes | confirmed defect |
   | DR-2 | `synth/compose.go:290` says "degrees Celsius", `synth/marine.go:163` "degrees" | confirmed |
   | DR-3 | `app/cast.go:304` `spokenName` has no caller outside its test | confirmed |
   | DR-4 | `app/setup.go:63` compares ZIP or exact coordinates, not `snapshot.PlaceID` | confirmed |
   | DR-6 | `app/severe_read.go:332` parses `"(~…)"` out of the display string | confirmed |
   | DC-2 | `platform/config/config.go:118` names a `radioModes` set that does not exist; `[radio] mode` is not validated | confirmed |
   | DC-4 | `modes/tty/broadcaster.go:25` imports go-studs | confirmed |
   | DC-8 | `Makefile:382` `VERDICTS_RECORD` is the 0.16.0 folder's log | confirmed |
   | JR-2 / DC-1 | `go doc -u ./app startSchedule` prints no doc; its doc sits on `bedSeams` (`app/schedule.go:73-85`) | confirmed |
   | JR-3 | `go doc ./modes/tty` still describes "the B3 build of mock rev2" | confirmed |
   | JR-7 | `modes/tty/dashboard.go:1000` is one 1,410-character line | confirmed |
   | JR-9 | `app/executors.go:57` says `read` returns "whether the words ran out" | confirmed |
   | DR aside | the Broadcaster's `render.Opts` sets no Units (`broadcaster.go:578`), and its pool table writes temperatures and distances through it (`platform/render/table.go:497`, `:520`) | confirmed defect (DEF-2 below) |

2. **At BUILD**, every remaining finding is checked again where it is changed: each batch writes the pin
   first and sees it fail on a mutant before the change (the rule in the required reading). A finding
   that does not hold there is recorded as not holding, not changed.

## Defects found (user-visible or record-damaging)

| # | Finding | What happens |
|---|---|---|
| DEF-1 | RN-4 | Backspace in the Broadcaster's Line-Up Request window cuts a byte: "Peñ" then backspace leaves invalid UTF-8 in the query. Every other text field erases a letter. |
| DEF-2 | DR aside | The Broadcaster's location pool writes temperatures in °F and distances in miles whatever the units Setting, against 0.16.0 FR-6.2 ("shared theme, units, clock"). |
| DEF-3 | DR-2 | Under metric, the radio says "23 degrees Celsius" for the air and "23 degrees" for the water; the comment says they match. Which wording is right is a ruling. |
| DEF-4 | DC-8 | `make mutant-verdicts` writes into 0.16.0's shipped record, since its target was never moved. |
| DEF-5 | DC-2 | `[radio] mode` is not validated at load, though the comment says it is. Whether to validate is a ruling (it changes what a bad config does). |

## The work, by stream

- **A. Pins first.** RN-1 to RN-13, PF-6's beat pin (bytes as well as allocations), RN-11's goldens for
  the console, Details, Help, Status and Diagnostics. Nothing in B to E touches code a pin does not
  cover.
- **B. Performance.** PF-1 to PF-5 are small and preserve behaviour by construction or by a tested
  equivalence; together they cut the bytes of an idle beat by 33 %, a radio beat by 34 % and a console
  beat by 48 % (measured on the lens's prototypes). PF-7 to PF-11 are medium: memos and copies. D-4's targets need PF-7 to PF-10 as well.
- **C. Consolidation.** Behaviour-identical: DR-1, DR-5, DR-7a, DR-8 to DR-14, DR-16, the test helpers.
  Each needs a ruling first: DR-2, DR-3, DR-4, DR-6, DR-7b, DR-15.
- **D. Docs and artifacts.** DC-1 to DC-17: wrong-bound docs, names that do not exist, follow-ups.md
  (about a third closed rows, twelve "open" rows done), the as-built maps, the atlas's plan-era
  diagrams, the Makefile, README's Settings and screenshot.
- **E. Readability.** JR-1 to JR-13: package front doors (`modes/tty`, `app`), vocabulary, an
  "add a Setting" walkthrough, keyed literals, moving essays out of function bodies, onboarding.
- **F. The checks that let this drift.** Widen SN-02 to types, values and fields (DC-1, JR-2); a stale-row
  check in `tools/dupes` (DC-12); AP-HIST-01 over the Makefile and scripts (DC-9); a symbol check over
  `docs/extending.md` (DC-14, JR-9); an enforced go-studs import rule (DC-4).

## The rulings this asks for

Put one at a time, in this order:

1. The brief: problem, metrics and targets (`../01-objectives/project-brief.md`). **Ruled: D-4**, M2 stretched.
2. DEF-1 and DEF-2: fixed in 0.19.5, or a 0.19.4 patch first.
3. DEF-3 (DR-2): the spoken temperature wording under metric.
4. DR-3: the voice's spoken name, with or without its language suffix.
5. DR-4: which "same place" rule first-run Setup uses.
6. DEF-5 (DC-2): validate `[radio] mode` at load, or say it is not.
7. DC-4: keep go-studs behind `platform/render` (move the console's rows), or restate the rule.
8. JR-1: ruling IDs: a generated index, and naming the release in new citations.
9. JR-12: the all-caps comment style.
10. JR-5(c): rename the narration arbiter (`app/director.go`) so it is not a second "Director".
11. JR-10 and JR-7: P10's rules written into `code-standards.md`, and whether a keyed literal counts as one line.
12. DC-7: the atlas's plan-era diagrams: labelled, moved to a history section, or left out.
13. DC-8: the Makefile's release-bound paths: derived from the release in flight, or fixed.
14. DR-6, DR-7(b), DR-15: the alert window as a value; one compass table; the card box's rule.

Minor choices (names, wording, file layout) are A-rows for veto.
