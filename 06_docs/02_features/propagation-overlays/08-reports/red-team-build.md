---
title: "0.19.0 (with go-ionomaps and go-tuiMaps v0.3.0) — BUILD exit red team, round 1"
date: 2026-10-17
phase: BUILD exit
sev: SEV-0
authority: HUM LEAD
status: "ROUND 1 DISPOSITIONED (D-203; forks D-204 to D-217; remediated, batches 35 to 47). ROUND 2 REPORTED 2026-10-18 (D-225): four blind reviewers, all four do not exit BUILD, no Critical; its forks are put below"
---

# BUILD exit — red team, round 1

Up: [rulings](../02-analysis/rulings.md) · [build log](../04-development/build-log.md) · reviewer notes: [build-round-1](reviewer-notes/build-round-1.md), [build-round-2](reviewer-notes/build-round-2.md)

## Summary

- **Eight blind reviewers (D-203), over watchpost `5012fae7`, go-ionomaps `d7e7f3c` and go-tuiMaps `b50fc86`. All eight say do not exit BUILD.**
- **Five reviewers found the same first fix: watchpost does not run on the go-tuiMaps it will ship with.** Its `go.mod` pins `9dc4ff3`; P5, P6 and the checklist fix (`b50fc86`) were built and gated in go-tuiMaps alone, so their colour-depth change, shade key, marks and dialer have never run in watchpost or been through UAT. go-ionomaps has no tag and no release checklist; both feature branches are behind released fixes on `main` (go-tuiMaps' merges with conflicts).
- **Two Criticals:**
  - **Accessibility AX-1:** away from the US regions, the Observer's words name the wrong place for the origin - the region last drawn, or "the map" - so a listener who pans or finds a place abroad is told about somewhere they are not looking (`modes/tty/map_describe.go:150-153`, verified).
  - **Business B-F1:** a place found on the Observer becomes the origin, not a target, while FR-1.9 (`requirements.md:77`) and the UAT protocol's M5 (`uat-protocol.md:35`) make found places targets beside the continents (verified).
- **Wrong answers in front of a listener today** (verified): the hours ahead carry the live D-RAP disturbance (`map_prop_ahead.go:94-97`, L-F2); the chart, centre and reach answers stay keyed to a snapshot the no-readings correction has rebuilt (`map_prop_chart.go:44-50`, L-F3); a frequency's reach ignores the lower limit the chart applies (go-ionomaps `reach.go:26-29`, D-F3); found coordinates are spoken "33.2000 megahertz" (`map_prop_words.go:264-282`, AX-2); each hour ahead reads out a developer citation (go-ionomaps `ahead.go:48`, AX-3); a deadline on a slow link is said as "asked too soon" with a wrong cause and a midnight time (go-ionomaps `update.go:110-113,147-148`, L-F4); a mode reopened while its last update is still cancelling joins that update and gets no field (`domains/propagation/propagation.go:111-135`, C-2).
- **The record's gaps:** four tests the requirements name do not exist, one described as "written first" (B-F4, D-F2, verified); the requirements still state FR-1.8's origin and D-77's scope that D-193 and D-195 overturned (D-F1, B-F5); the architecture's library shape is stale (D-F4); go-ionomaps' README says "no code yet" (D-F6); file and line citations have drifted and the check reads only that a line is given (C-10, D-F5).
- **Unmeasured or missed targets:** go-tuiMaps M5 missed 2x on the UI goroutine with the night and the reach drawn (P-3, measured by the reviewer: 33.6 ms against 16 ms at 400x110); G1, R-8.1 and M1/M4 have no number at BUILD exit (P-4, B-F3); FR-1.17's figure is a spike's (P-4, D-F9).
- **Hosted CI, which the agent had not been reading:** batch 28 failed on macOS (`TestThePropagationRowsFollowTheMock`, a wall-clock fixture the agent wrote, at a minute boundary) and batch 32 failed on both systems (the shell hook, since replaced); the tip, batch 34, is green on both.

## The verdicts

| Reviewer | Verdict | Fix first |
|---|---|---|
| Code Quality | Do not exit | C-19: the go-tuiMaps pin |
| InfoSec | Do not exit | I-1: the go-tuiMaps pin; I-2: positions in unrecognised forms reach the online geocoder |
| Accessibility | Do not exit | AX-1: the origin named wrongly abroad (Critical) |
| Performance | Do not exit | P-3: the M5 frame missed 2x on the UI goroutine |
| Docs Quality | Do not exit | D-F3: the reach ignores the lower limit |
| Project Hygiene | Do not exit | H-1: the pin, with H-2's merge from `main` |
| Business Quality | Do not exit | B-F1: a found place is the origin, not a target (Critical) |
| BUILD lens | Do not exit | L-F2: the hours ahead carry today's disturbance |

## The forks for the HUM LEAD, in order

Each is put singly, with its evidence, options, a recommendation and the strongest counter-argument, and recorded verbatim as a D-row.

1. **A found place on the Observer: a target row, or the origin** (B-F1, FR-1.9, M5).
2. **The reach and the lower limit: apply it, or record the gap** (D-F3, R-2.9, FR-1.16).
3. **The library pins: tag go-ionomaps v0.1.0-rc.1 and go-tuiMaps v0.3.0-rc.2 at merged tips, and the visible colour-depth change (L-7.2) in watchpost** (I-1, H-1, H-2, H-7, AX-14, D-F8).
4. **The metrics at BUILD exit: amend D-118 and D-132, or build G10.1/G10.2 and measure G1 inside BUILD** (B-F3, P-4).
5. **The M5 frame: P7's blend memo and the edge pass now, or record the miss** (P-3).
6. **The words' read order: grouping identical targets, the station said once, the weather heading in this mode** (AX-7).
7. **A reader's place on a refresh: anchor the scroll, reserve the lines, or the age on change** (AX-5).
8. **The answers seam: build it, or ratify the departure and correct the architecture** (L-F1).
9. **GIRO's throttle across restarts: persist it, or ratify the risk A-20 left** (L-F7).
10. **A GIRO-free option for go-ionomaps' commercial users, or correct the README** (B-F6).
11. **The P10 workarounds: fix the analysers, or ratify the pattern as a class** (C-7); **go-ionomaps' root and `internal/limits` ledger rows** (C-8, H-3, H-4), with go-ionomaps' A-7 to A-17 put for veto.
12. **The libraries' API corrections before v0.3.0 and v0.1.0 freeze:** `Snapshot.Hours` (C-11, L-F14), go-tuiMaps' two-class edge mark (C-18), its colour-count error kind (C-17, L-F13), and `CheckedDialerFor` adopted or dropped (C-15, I-7).
13. **PS-1's population: evidence, or accepted as a hypothesis** (B-F2).

## The rulings

| Fork | Ruling |
|---|---|
| 1, a found place | D-204: a target row on the Observer; the map does not move |
| 2, the reach | D-205: the lower limit applied |
| 3, the pins | D-206: merge `main`, tag go-ionomaps v0.1.0-rc.1 and go-tuiMaps v0.3.0-rc.2, pin and gate |
| 3b, colour depth | D-207: a "Colour depth" Setting, Automatic by default |
| 4, the metrics | D-208: G10.1, G10.2 and the M1/M4 baseline in BUILD; the long runs beside REVIEW, D-118 and D-132 amended |
| 5, the M5 frame | D-209: fixed in go-tuiMaps before rc.2, the benchmark in M5's shape |
| 6, the read order | D-210: identical targets grouped, the station said once, the mode before the weather |
| 7, a reader's place | D-211: the scroll anchored to the sentence at its top |
| 8, the answers seam | D-212: built, the targets an input |
| 9, GIRO's throttle | D-213: its state persisted across restarts |
| 10, GIRO-free | D-214: a "no stations" option, and a watchpost Setting |
| 11, the P10 pattern | D-215: the gaps upstream; the code written to the rules' intent |
| 12, the APIs | D-216: all four corrections, and a holistic review of both libraries' APIs |
| 13, PS-1 | D-217: a hypothesis, said |

## The remediation batches, in order

1. **F-216 first** (due 2026-10-19): the CI actions re-pinned off Node 20 and the runner image chosen.
2. **go-ionomaps:** the API review and its corrections (`Snapshot.Now`, an ahead snapshot with no live disturbance, `Sources()` a package function, `Band.Name`), the reach's lower limit, "no stations", the throttle's state, L-F4 to L-F6, L-F12, I-5, C-20, C-22, C-24, P-2, G10.1 and G10.2, the P10 rework (D-215) with its ledger rows put for ratification, the README, NOTICE, the evidence logs, the gate's fixes (C-9, H-11), its release checklist; v0.1.0-rc.1.
3. **go-tuiMaps:** `main` merged (v0.2.1), the API review and its corrections (the explicit edge mark, `MalformedRamp`, the host-ramp warning once), the M5 frame (D-209), C-25, C-27, `socks5h`, C-16, the README and contract, the P10 rework; v0.3.0-rc.2.
4. **watchpost:** `main` merged (0.18.1), pinned to both release candidates; the answers seam with the found target row; the colour-depth and no-stations Settings; the throttle kept; then the words, the keys, the join, the session state and the remaining findings below; the record.

## The agent's own remediation, without a ruling

Batched after the forks, each batch through its gate with hosted CI read before the next:

- **The ship-together state:** merge `main` into both feature branches (H-2), the libraries tagged as ruled, watchpost pinned and gated on them (I-1, H-1, H-7); the libraries' full hosted gates dispatched (H-10).
- **Wrong answers and words:** L-F2, L-F3, L-F4, L-F5, L-F6, C-2, AX-1, AX-2, AX-3, AX-4 (hours said), AX-6 (Help's hour row), AX-8 to AX-13, P-6, L-F12, I-2 (positions refused locally), I-3, I-4, I-5, I-6.
- **The record:** B-F4 and D-F2 (the four tests mapped or written; the trace check made to find each test in the code), D-F1 and B-F5 (the requirements amended under one A-row), D-F4, D-F5 and C-10 (citations re-pointed and resolved), D-F6, D-F7, H-6 (the READMEs), D-F9 to D-F15, H-5, H-8, H-12, H-13, C-1, C-23.
- **The code:** C-3, C-4, C-5 and L-F10, C-6 (functions past 60 lines), C-9, C-12, C-16, C-20 to C-22, C-24 to C-27, L-F8, L-F9 (one horizon), L-F11, P-1, P-2, P-5, I-8, H-9, H-11, and the batch 28 wall-clock test pinned to its own clock.

## Round 2 (D-225)

**Four fresh blind reviewers** looked at watchpost `559f22db`, go-ionomaps `2436c8a` and go-tuiMaps `228d6f7`: the BUILD lens (BL2), Code (CN2), Docs (DN2), and the combined personas (PN2). They first checked round 1's closure finding by finding, then looked for anything new. **All four say do not exit BUILD. No Critical.** Their notes are in `reviewer-notes/build-round-2.md`.

**Round 1's closure.** Both Criticals are closed (AX-1, B-F1). So is most of the rest. These are not closed:
- **Not closed:** L-F1 in part (the reach and the hour ahead are still worked out in the window); L-F8 (bands labelled by position); C-7 in watchpost (D-215 not carried, BL2-2, CN2-3); C-16 and C-25 in go-tuiMaps; D-F15; I-2 (positions in common Unicode forms still go online, PN2-4).
- **Closed in the record only:** L-F5 and L-F6 for the host (the library's reasons are never said, BL2-3, CN2-5); C-10 and D-F5 (the citation check, CN2-7, DN2-8).

**What the remediation itself brought in, found by more than one reviewer:**
- **The reach's words ignore the limit go-ionomaps now returns** (BL2-1, CN2-4, DN2-1, PN2-2: all four). Whenever nothing is reached, the words say "above the MUF everywhere". When absorption or the circuit closes the band, that is false. D-205 and FR-1.16 require the limit.
- **A throttle-state file that Restore accepts crashes watchpost** every time the mode opens (CN2-2, PN2-1). go-ionomaps bounds one counter, not the other.
- **The reach now costs 0.4 to 0.9 s, not about 30 ms** (DN2-2, PN2-3). The lower limit (D-205) made it 15 to 30 times slower. watchpost's record still says otherwise. Hour steps neither settle nor cancel, so holding the key through 24 hours queues 24 reaches.
- **The library's honesty fields reach no listener:** GIRO paused, Unfinished, the refused feeds, and "no stations: by choice" (BL2-3, CN2-5).
- **D-215 was not carried into watchpost** (BL2-2, CN2-3). Batch 45 added a new instance: a rename and density guards in `tools/reachscore`.

**Single-reviewer Importants:** CN2-1 (the Observer's answers freeze after a pan followed within 600 ms by another window), CN2-7 and CN2-8 (the record checks), CN2-9 (the instrument's checks fail one way only), CN2-10 (go-tuiMaps' M5 is not enforced; an Edge refusal has the wrong error kind), CN2-11 (two caches of one climatology), DN2-3, DN2-4, DN2-6, DN2-9 (the BUILD report: floor 2 misstated, D-217's sentence missing, the as-built map stale, the hosted-CI claim unsupported), DN2-5 (D-22's RBN check dropped without a ruling), DN2-7 (the architecture's answers path), DN2-10 (the README says nothing of 0.19.0).

### The forks for the HUM LEAD, in order

Each is put singly, with its evidence, options, a recommendation and the strongest counter-argument.

1. **D-22's RBN cross-check:** build it into the instrument, or rule it out (DN2-5).
2. **The reach's cost:** accept about half a second to a second, off the UI goroutine, with the latest ask winning; or make the method cheaper in go-ionomaps (PN2-3, DN2-2).
3. **M1/M4's uncertainty:** report an interval beside each figure, and say whether the target is judged on it (BL2-6).
4. **The libraries' last API corrections before they freeze:** go-ionomaps' `Inputs.Refused` and `Kept`; go-tuiMaps' Edge refusal kind (CN2-15, CN2-10).

### The agent's remediation, without a ruling

Each batch goes through its gate, and hosted CI on the tip is read and recorded before the next.

- **go-ionomaps rc.3:**
  - the state's counter bounded, with a fuzz test that runs an update after each restore (CN2-2, PN2-1);
  - one cache for the climatology (CN2-11);
  - the 5xx counted, and the fault test made to fail when nothing is reported (CN2-15, CN2-17);
  - the status lines (DN2-12).
- **go-tuiMaps rc.3:**
  - the M5 benchmark run with its contours, and held by the release (CN2-17, CN2-10);
  - C-16, C-25, `Class.Shade`, and the documents' contradictions (DN2-13, DN2-12).
- **watchpost, the words and answers:**
  - the reach said by its limit (all four);
  - each reason said;
  - the answers re-armed when the map returns to the front (CN2-1);
  - positions refused by shape (PN2-4);
  - the hour ahead's moment one (BL2-4);
  - the words remembered when the place is kept (PN2-5, CN2-19);
  - the geocoder's text clamped, a cancelled find cancelled, and the errors in words (PN2-6, PN2-7, PN2-9, BL2-7);
  - bands by name (L-F8, CN2-14);
  - the reach and the hour ahead moved into `propanswers` (L-F1).
- **watchpost, the remaining code:**
  - D-215 carried through, with its exemptions put for ratification (BL2-2, CN2-3);
  - the state file's name made unique, and the state kept at the burst (BL2-5);
  - the instrument's checks failing closed (CN2-9);
  - `main` merged (0.18.1, BL2-8).
- **watchpost, the record:**
  - the BUILD report corrected (DN2-3, DN2-4, DN2-9, DN2-14, DN2-15);
  - the as-built map redrawn, with its test reading 0.19.0's log (DN2-6);
  - the architecture (DN2-7);
  - the citation check made to match a symbol (CN2-7, DN2-8);
  - the hook's list (CN2-8);
  - the README and CHANGELOG for 0.19.0 (DN2-10);
  - the instrument page (DN2-11);
  - the status lines (DN2-12).
