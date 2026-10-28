---
title: "0.19.0 Propagation overlays, with go-ionomaps and go-tuiMaps v0.3.0 — REVIEW report"
date: 2026-10-27
phase: REVIEW exit
sev: SEV-0
authority: HUM LEAD
status: "APPROVED by the HUM LEAD 2026-10-27 (D-322). REVIEW exited on watchpost b434e021 (batch 76), go-ionomaps v0.1.0-rc.16 and go-tuiMaps v0.3.0-rc.10: its last round (D-320) says exit, no Important; every finding of rounds 1 to 10 is closed or ruled. VALIDATE is open."
---

# REVIEW report — 0.19.0, go-ionomaps v0.1.0, go-tuiMaps v0.3.0

Up: [red-team review, the record](red-team-review.md) · [rulings](../02-analysis/rulings.md) · [VALIDATE's agenda](../07-readiness/validate-agenda.md) · [BUILD report](build-report.md)

## What REVIEW reviewed

BUILD exited on `e889e72a` (D-233). REVIEW ran ten red-team rounds over the release as a whole and each round's remediation, every reviewer blind and fresh, each report filed verbatim (`reviewer-notes/review-round-1.md` to `-10.md`), every finding closed in the record against a commit, a tag or a ruling.

| Round | Ruling | Reviewers | Verdict | What it found, and how it closed |
|---|---|---|---|---|
| 1 | D-245 | 7 | 4 do not exit, no Critical | 17 forks ruled (D-246 to D-262), remediated in watchpost batches 59 to 65 and the libraries' candidates; measurements recorded (D-248, D-261, D-265, D-266) |
| 2 | D-269 | 7 | 5 exit, 2 do not | D-270 to D-282; batch 66 |
| 3 | D-284 | 3 | none exit | four Importants, two where round 2's fixes met the host's deadline; D-285 to D-289 |
| 4 | D-290 | 3 | 1 of 3 exits | D-286 caught a GIRO hang only early in a burst; D-291 to D-293; batch 68 |
| 5 | D-294 | 3 | 1 of 3 exits | two Importants in D-291's deadline rule; D-295 to D-299; batch 69 |
| 6 | D-300 | 4 | 2 of 4 exit | a quit during GIRO's burst kept no state; `MinDeadline` refused itself; D-301 to D-307; batches 70 and 71 (and rc.11's amd64 fuzz find, rc.12) |
| 7 | D-308 | 4 | none exit | D-307's own order called a healthy GIRO down; the HUM LEAD directed one settled timing model (D-311, D-312); batch 72 |
| 8 | D-313 | 4 | none exit | the model wrong in one place, changed whole (D-314); batches 73 and 74 (and batch 72's hosted race find) |
| 9 | D-315 | 4 | none exit | three more Importants in the timing model, two in round 8's own fixes; **the HUM LEAD replaced the model with six plain points (D-316 to D-319)**; batch 75 |
| 10 | D-320 | 1 | **exit** | one light review of the new model, weighed by what happens: no Important, seven Minors; D-321; batch 76 |

## The candidates REVIEW exits on

- **watchpost** `feature/propagation-overlays`, batch 76 (`b434e021`): `make verify` green; batch 75's hosted CI green on both runners, batch 76's running on its push.
- **go-ionomaps** v0.1.0-rc.16 (`74efaaf`), `feature/discover`: its full gate green; the full hosted gate on both architectures (run 40608474873).
- **go-tuiMaps** v0.3.0-rc.10 (`5f6bf07`), `feature/propagation-fields`: no code change since round 6; full hosted gate green at rc.10.

## The one area that did not settle, and what ended it

Rounds 3 to 9 each found an Important in the update's timing - how the library spends its deadline and treats GIRO's ionosonde stations. Each round's fix was right for its finding and broke an earlier one; rounds 7, 8 and 9 each found theirs in the previous round's own fix, even after the HUM LEAD fixed one written model to judge against (D-311, D-312). In round 9 the HUM LEAD named the cause - "overindexing on the what 'might bes' over the 'what happens'" - and replaced the model with six points (D-316): try the stations; note one with no data and move on; flag one missing three asks in a row and ask it every 6 hours until it returns (D-317); pause GIRO only for its own refusals (D-318); and say plainly, as kc2g's banner does, when fewer than half the stations reported in 3 hours or the newest reading is older (D-319). Round 10's reviewer, told to weigh findings by real use, found no Important and replayed the PLAN week's 96 hours through the notice's rule: it would have shown in none.

The model is go-ionomaps `06_docs/02_features/go-ionomaps/03-architecture-design/update-timing-model.md`. A later finding in that area is judged against it and weighed by what happens: a case only a probe can construct is a stated limit, not a new mechanism (D-316).

## What VALIDATE takes

Every item is on [VALIDATE's agenda](../07-readiness/validate-agenda.md), each with its ruling:
1. **M1 and M4 short of their targets** (53.7 % against 57.4 %, 68.4 % against 69.4 %).
2. **PS-1's claim** - still a hypothesis (D-217).
3. **G1, both comparisons**: met against the map open in Forecast mode (-0.17 % of a core, -10.4 MB), one 48-hour run on one machine; missed against the map never opened. **The Broadcaster arm** missed G1 on `a444b000` (+1.20 % of a core, +57.5 MB RSS); its re-run on the tip (`c8a99e48`, D-293) ends about 2026-10-29 15:52 UTC, and its verdict words item 7's known limit (D-274). Round 10 noted it measures go-ionomaps rc.13, not rc.16; rc.14 to rc.16 change only GIRO's hourly burst, which the measurement's verdict will say.
4. **The default refresh.**
5. **The Pacific worse than the climatology.**
6. **The memory growth's fix shown over days** (D-242).
7. **The changelog's known limits**, among them the timing model's accepted limits and, if it misses, the Broadcaster arm.
8. **M5 at 800 by 220.**

UAT walks the [protocol](../07-readiness/uat-protocol.md), the best-approximation notice and its words among what the HUM LEAD sees.

## For the HUM LEAD's veto, batched (D-13)

- **A-37 (3)**: `internal/stations`' P10 floor renewed by the agent at 1.29 (round 7). Since D-316 the package is rewritten; its floor check passes as it stands.
- **A-39**: the build of the six points - one place in the 40 kept for a flagged station due; stations that reported before ahead of codes never heard; a reply with no reading a miss.
- **A-99, A-100**: the notice's words and place - "BEST APPROXIMATION: … rely on this map warily", first on the status line - and "N stations gave no data".

## Still the HUM LEAD's to do

- **go-tuiMaps' nested modules' v0.2.2 tags** (R2H-5): the push was refused to the agent and is the HUM LEAD's.

## What the release learned here (for REFLECT)

- A red team can manufacture complexity: a probe can always construct the next edge. Weigh findings by what happens, and when a mechanism needs a fix in consecutive rounds, simplify it rather than patch it (D-316).
- The hosted gates found what no local run could: an amd64-only NaN panic (rc.11) and a race-detector bound past 2 s on a slower runner (batch 72). The full hosted gate ran before each later round.
- Two of the agent's own gate runs failed on checks it had not run first (a record test, and the whole mutant set); both are now in its pre-gate list.
