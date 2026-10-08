---
title: "0.19.0 PLAN exit red team, round 2 - condensed reviewer notes"
date: 2026-10-07
phase: PLAN exit
sev: SEV-0
authority: HUM LEAD
status: "RECORD - each reviewer's findings, condensed by the agent (the full reports are kept outside the tree with the PLAN evidence)"
---

# PLAN exit red team, round 2: reviewer notes

Four fresh blind reviewers (D-128) at watchpost `cb1ca501`, go-ionomaps `b7a7b19`, go-tuiMaps `0784649`, checking closure of round 1 and anything new. IDs: persona N (combined Accessibility, InfoSec, Performance), LN (PLAN lens), CN (Code), DN (Docs).

## Round 2 combined personas — verdict: DO NOT EXIT YET; no Critical; Importants N1-N6. Fix first: N1.
Closure: A11y-1..10 closed (A11y-5 untested N7); IS-1..10 closed (IS-2 scope N9, IS-3 counts N14, IS-10 stale N13); P-1 closed; P-2 closed in ruling (stale N12); P-3 NOT closed (N1, N2); P-4 partly (N3); P-5 partly (N4); P-6..P-11 closed (P-10 stale N13).
N1 Important: A-29 NoData + NearestKm per field per hour ~7.4 MB/snapshot (from 3.3); two held + clim + body ~21 MB > D-112 15 MB; share Hours[1..24] with climatology cache; one NearestKm per snapshot; NoData bitset; re-derive cap.
N2 Important: D-112 split leaves 6 MB headroom for 19 MB added live heap at GOGC=100 (~38 MB); HUM re-rule: gate on RSS, lower live, or memory limit.
N3 Important: P-4 fix prose only; shape has one seam MapPropagation(ctx, ask); sequence diagram still runs Update inside per-ask; give update and answers a seam each, redraw diagram.
N4 Important: memo key misses radius (D-100 Setting) and target set; extend FR-8.2 guard.
N5 Important: CorrectNoReadings only in Options at New; one object per process; toggling Setting needs new object -> fresh throttle (another burst outside D-39) and loses typical offset; make it a parameter of answers/Update or apply from Offset.TypicalMHz; test toggle makes no request.
N6 Important: RK-6 mitigations no longer hold for 0.19.0 (FR-7 gate moved to 0.19.1; M5b sighted); ships mode whose focus screen reader can't follow, nothing in app says so; FR-11.4 already places cursor for ack -> extending to 4 focus points cheap; HUM fork: Help states limit, or extend cursor.
N7 Minor: D-127(1) state said has no test; TestEachControlsStateIsSaid.
N8 Minor: FR-3.1 order doesn't place new sentences (typical label/error/basis, unusual offset, nearest station, updated); HUM read order/verbosity.
N9 Minor: D-113 allowlist from library's own export guards bugs only; state scope; optionally go list -deps check.
N10 Minor: Retry-After unbounded; bound, parse-check, fuzz.
N11 Minor: cancelled partial GIRO burst counted against hour? TestACancelledBurstCountsAgainstTheHour.
N12 Minor: NFR-1 "15 MB an hour" unqualified (requirements.md:199), architecture.md:237, design.md:53 "304 when unchanged", W10.2 wire-byte counting method (Go decompresses gzip transparently).
N13 Minor: go-ionomaps plan :18 govulncheck "from first dependency"; G10.3 :132 1 deg run.
N14 Minor: At/Bands/Path slices unbounded; cap counts in R-3.5.

## Round 2 PLAN lens — verdict: DO NOT EXIT YET; no Critical (both R1 Criticals closed); Importants N1-N4. Fix first: N1.
Closure: L-F1..F4, F6, F8, F10, F12 (trace 95/95, 74/74, 30/30), F13 closed; L-F5 NOT closed (N4); L-F7 closed except data time (N3); L-F9 partly (N1); L-F11 closed (folder row lands with report); L-F14 partly (N8); L-F15 closed, new stale N9.
LN1 Important: seam still single MapPropagation(ctx, ask) (architecture.md:69), diagram Update inside per-ask (:89-96); mapWorkLimit 30 s (map_pane.go:53) vs 36 s GIRO burst; cancelled burst budget unstated (no G9 test) -> open/close repeatedly breaks D-39; first picture waits whole burst, no target; split seam, deadline, TestACancelledBurstSpendsTheHoursBudget, GloTEC+climatology snapshot before GIRO?
LN2 Important: G10.4 scores on PLAN's week (same data set targets/kernels); D-118 floor can't fire; 15% target certain; score >=3 days fetched in BUILD after PLAN; cost to HUM.
LN3 Important: shape carries only Computed; data age (GloTEC valid time, oldest reading) lost (A-29 removed Valid); FR-5.1, R-5.7 can't be met; add input times to Inputs; say which FR-5.1 reports.
LN4 Important: D-123's safeguard (agreement reported in MHz beside tolerance) has no task; go-ionomaps rulings never restate D-123; carry into G3.5, R-9.7, restate.
LN5 Minor: R-9.8 12-month expiry in gate = time bomb (CI red on unrelated fixes, rebuild of tags); no owner/procedure; shape carries basis only for typical error; make it a release-step check, name re-measure.
LN6 Minor: one TypicalError globally; clim MUF error 2.28 (US quiet) to 4.29 (storm), 5.22 Pacific; HUM: one figure or by distance band.
LN7 Minor: OpenHours current hour source unspecified (climatology vs hybrid) -> contradiction now; state in R-2.8.
LN8 Minor: size estimate uncalibrated (0.18.0 174 tasks / 137 batches = 0.8; 90 tasks -> ~70); put in RK-5.
LN9 Minor stale: requirements.md:20 "labelled as forecasts"; dry-run.md:7, :36, :327 D-103 blend; dry-run.md:40 G10.4 as M1/M4 instrument (W10.3); architecture.md:96 and design.md:66 valid/age fields; design.md:81 SILSO/R12; watchpost project-brief.md:146, go-ionomaps project-brief.md:65 "allowed hosts struck A-12"; requirements.md:199 NFR-1 15 MB decoded; problem-statement.md:104 D-98 vs D-112.
LN10 Minor: dropped blend (69 vs 48) preserved nowhere; link to F-210.

## Round 2 Code — verdict: DO NOT EXIT YET; no Critical; Importants N1-N3. Fix first: N1. week.py all ten sections reproduce; lints pass; plancode on siblings: go-ionomaps clean, go-tuiMaps hits only v0.1.0 approach docs.
Closure: most C-* closed. C-M3 partly (N3); C-M4 closed, stale text N4; C-C1 deferred; C-C2 reason (coupling now broken N8); C-N not carried through: design.md:81 SILSO (N5), design.md:66 + architecture.md:96 valid/age; C-E4 NOT closed in substance: A-28 chose CCIR refit on same week's whole-week figures; C-E5 message also blames CS for missing coord header (minor); C-E6 residue N9; C-E7 no tolerance N13; C-E9 pending report; C-10 residue N14.
CN1 Important: D-109's figures (69/48, 49/49) don't reproduce from committed week.py (76/56, 73/55; 55/53, 53/51); correction row; cite section 10; add section 10 to dry-run.md.
CN2 Important: typical error value undefined (candidates: sec 2 clim test 1.14/4.04; sec 6 +3h 1.12/3.98; sec 8 quiet 1.07/3.69); name section/subset/lead, week.py prints it, G10.5 asserts. Minor: how 'at' between hours maps to an hour.
CN3 Important: Snapshot can't express D-RAP missing, GloTEC grid age, measured part (R-3.2); add Inputs.DRAP, GloTEC valid time; define measured as NearestKm within X or a mask.
CN4 Minor: GloTEC 304 text architecture.md:93, :237, design.md:53; rename TestAnUnchangedGridCostsA304 -> TestAnUnchangedFeedCostsA304.
CN5 Minor: design.md:81 SILSO/R12 vs R-7.3.
CN6 Minor: G10.3 1 deg record; plan:18 govulncheck "from first dependency".
CN7 Minor: week.py doesn't check DISTURBED days in week; exit if not in by_day or split empty.
CN8 Minor: offset.py usage DIR DSI now exits 1; document DIR DSI DISTURBED (offset.py:4, evidence/README.md:19).
CN9 Minor: dry_fetch gives up after 480 s, never 15 min; NOAA loop keeps asking after giving up.
CN10a Minor: internal/forecast + G8 now a cache lookup + flag; fold into G9/climatology cache.
CN-necessity Minor: Response.NotChanged = Status==304; Hour.Typical = index>0; R-5.6 merging duplicates watchpost's single owner; R-3.3 hmF2 range check; Options.Grid one value (HUM rec).
CN11 Minor: week.py docstring lacks hybrid, pinned clim, sections 9-10; :238 comment misleading.
CN12 Minor: go-tuiMaps M6 no task.
CN13 Minor: G10.4 replica "agrees" no tolerance.
CN14 Minor: D-119 release step check unplanned; record proves machine by own say-so.

## Round 2 Docs — verdict: DO NOT EXIT YET; no Critical; Importants N-1..N-7. Fix first: N-1. week.py reproduces; trace 95/95, 74/74, 30/32 (L-5 out); 13 code citations hold.
Closure: D-F1..F17 closed except D-F2 leftover (N-9), D-F4 two places missed (N-5, N-6), D-F15 figures wrong (N-1), D-F16 partly (N-7).
DN-1 Important: D-109, D-116, RK-3 figures don't reproduce; Pacific flips (hybrid 5.17 vs clim 5.09, B on C 4.97; <500 km clim 0.72|2.63; >2000 hybrid 4.09 vs clim 4.25); RK-3 clim foF2 1.14 not 1.15; D-109 76/56, 55/53 (+3h), 73/55, 53/51 (+12h); ledger "every published figure reproduces" false; correction rows; publish section 10; error row; D-116 to HUM.
DN-2 Important: architecture sequence still Update inside per-ask (:89-91); diagram Snapshot "valid, age" (:96); GloTEC "304 if unchanged" and "15 MB an hour" (:93, :237); design.md:53, :66; redraw, regenerate atlas.
DN-3 Important: O2 follow-up rows (F-205, F-191..F-194, F-196..F-198, F-180) still "0.19.0 ... SHIP blocked"; change to 0.19.1 citing D-122 (D-110 for F-205).
DN-4 Important: NFR-1 (requirements.md:199) fixed 15 MB (D-95) without D-111/D-112; requirements win -> superseded figure binds; restate.
DN-5 Important: requirements.md:20 hours ahead "labelled as forecasts (D-50)"; -> typical (D-109).
DN-6 Important: FR-1.15 "hours today expected open": future hours climatology unlabelled typical; past hours no source (no history); HUM fork on source and label.
DN-7 Important: typical error value/subset unstated; one global figure (foF2 0.60 US quiet to 1.38 Pacific; MUF 2.28 to 5.09; worse in storm); record subset, figure, blind spots.
DN-8 Minor: design.md:81 SILSO row delete.
DN-9 Minor stale: dry-run.md:7, :36, :327 D-103; dry-run.md:70, :101 spikes "never committed" vs D-124; REQUIRED-READING lacks D-128; go-tuimaps requirements.md:4 phase DISCOVER; plan-questions.md:13 options claim; :50 retired decay; go-ionomaps requirements.md:102 R-9 "B on D", :138-139 old floors; F-206 "fallback"; uat-protocol.md:35 FR-1.15 -> FR-1.9.
DN-10 Minor: plan :18 govulncheck; G10.3 1 deg; two "first in BUILD" (W0.0, W1); P7.1 reach overlay redefines D-96's target without ruling.
DN-11 Minor: RK-12 "all 24 hours" one day.
Also: dry-run.md:18 "costs fit" but G1 RSS/downloads unmeasured.

