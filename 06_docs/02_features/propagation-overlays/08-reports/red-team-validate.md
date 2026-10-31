---
title: "0.19.0 — VALIDATE exit red team: the record"
date: 2026-10-30
phase: VALIDATE exit
sev: SEV-0
authority: HUM LEAD
status: "Round 1 (D-338): two fresh reviewers - code exits, release does not; every finding closed below against a ruling or batch 81."
---

# VALIDATE exit red team

Up: [rulings](../02-analysis/rulings.md) · [VALIDATE's agenda](../07-readiness/validate-agenda.md) · [the REVIEW record](red-team-review.md)

**Round 1 (D-338), 2026-10-29.** Two fresh, blind reviewers over VALIDATE's changes (watchpost batches 77 to 80, go-ionomaps v0.1.0-rc.17, the README fixes) and the release as it will ship, D-323 to D-337 standing, findings weighed by real use (D-316). Briefs kept with the evidence outside the tree; reports verbatim in `reviewer-notes/validate-round-1-code.md` and `reviewer-notes/validate-round-1-release.md`.

| Reviewer | Verdict | Importants |
|---|---|---|
| Code (VC) | exit VALIDATE | none; five Minors |
| Release (VR) | do not exit | VR-1, VR-2, VR-11; nine Minors |

## The Importants, each confirmed in the code first

| Finding | Closed by |
|---|---|
| VR-1: no refresh with the chart (or any window) over the map; on Hourly a chart left open never refreshed | D-339: the tick asks under a window, never under the acknowledgement (batch 81; `TestTheModeRefreshesUnderAWindow`, m598) |
| VR-2: the first run's esc saves, its chip said "Cancel" | A-105 (1): "esc Save" (batch 81; m599) |
| VR-11: each station's data provider not credited (D-41, FR-4.1) | D-340: acknowledged in About as GIRO lists them, with GIRO's address (batch 81; `TestEveryPropagationSourceIsCredited`, m601) |

## The Minors

| Finding | Closed by |
|---|---|
| VC-1: a first-run esc with a mistyped FIRMS key lost the pick | A-105 (2): the form kept for the reopen (batch 81; `TestAFirstRunSaveRefusedKeepsThePick`, m600) |
| VC-2: requests queued on a hung host each tried it once | A-105 (3): the memo read again with the slot held (batch 81; `TestRequestsQueuedOnAHungHostFailFast`, m597) |
| VC-3: Hourly's STALE margin about 5 minutes | A-105 (8): D-332's live check on the shipped 2 h bound, before SHIP |
| VC-4: two of D-334's figure sets were the probe's, its box hard-edged | re-scored with gm3 at rc.17 (go-ionomaps `37b7d42`); the rulings' figures labelled by source |
| VC-5: the interactive lane's 4 below the zone fetcher's 6 | A-105 (8): a stated limit, A-101 as ruled |
| VR-3: the README's Setup keys | A-105 (6) |
| VR-4: "GIRO is asked at most once an hour" | A-105 (5): "in one burst an hour" |
| VR-5: "the Lowell Global Ionospheric Radio Observatory" | A-105 (4): "the Global Ionospheric Radio Observatory", the acknowledgement's version 3 |
| VR-6: the heap fix's result read as still to come | A-105 (5): its 3 days stated |
| VR-7: two of the timing model's limits missing from Known | A-105 (5): carried in Known |
| VR-8: `B`, `L`, `U`, `F`, `/` newly the map's, unnamed | A-105 (5): in Changed |
| VR-9: figures without their basis | A-105 (5): the build, the refresh setting and the days, a clause each |
| VR-10: the agenda - item 5's independent week, item 4's bound not yet seen live, its status | the agenda, batch 81 |
| VR-12: OpenStreetMap's credit without its address | A-105 (7) |
