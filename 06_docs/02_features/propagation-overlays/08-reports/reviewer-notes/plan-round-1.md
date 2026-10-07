---
title: "0.19.0 PLAN exit red team, round 1 - condensed reviewer notes"
date: 2026-10-07
phase: PLAN exit
sev: SEV-0
authority: HUM LEAD
status: "RECORD - each reviewer's findings, condensed by the agent from the full report (the full reports are kept outside the tree with the PLAN evidence)"
---

# PLAN exit red team, round 1: reviewer notes

Eight blind reviewers (D-90), briefed verbatim from `06_docs/red-team-brief.md` at watchpost `90499ae2`, go-ionomaps `d33e139`, go-tuiMaps `2a6452b`. IDs are prefixed by reviewer in the ledger (`../red-team-plan.md`): B- Business, A- Accessibility, I- InfoSec, D- Docs, L- PLAN lens, P- Performance, H- Hygiene, C- Code.

## Business Quality — verdict: DO NOT EXIT PLAN. Fix first: F1.
F1 Critical: D-103 blend needs yesterday's field; D-39/D-76/D-78 mean it never exists unless open 24 h; week.py scored with whole week on disk; options (a) climatology alone labelled, (b) yesterday = B on C from one 24-h GIRO request per station, (c) cut under D-75.
F2 Important: forecast floor re-check in BUILD on same data that chose the method; re-check on days after PLAN; R-1.3 quote worst margin.
F3 Important: confidence label is foF2 error (architecture.md:161) but band answers depend on MUF; "as measured" undefined without recording.
F4 Important: FR-3 preamble (requirements.md:101-102) still promises D-80's device Setting cut by D-85 (F-211).
F5 Important: M1/M4 (+5 pts, D-99) no consequence on miss; no check baseline beatable.
F6 Important: go-tuiMaps M1 no target/instrument/protocol; M6 count unstated (project-brief.md:103,108).
F7 Minor: go-ionomaps README:11-12, requirements:7,104 call climatology "the fallback"; R-5.6 names retired readout/recorder; watchpost requirements.md:4,7 header says revised to D-76.
F8 Important: GloTEC compresses ~9x (2,502,606 -> 272,030 gzip); D-94/D-95/A-26 costs assume uncompressed; measure with Accept-Encoding gzip in BUILD.
F9 Important: A-26 Setting is UI (D-13 excludes); "on demand" has no refresh action in FR-1.14; D-95 targets the hourly setting A-26 created.
F10 Minor: storm modelling (D-88) and idle 3% redraw have no F-rows.
F11 Minor: NOTICE lacks IGRF-14 and Apex sample (R-4.3 allows); F-208 evidence stale (one day, hmF2 M3000).
Cheaper: climatology alone gives floor within 0.00-0.05; measure rate blend vs climatology give different band answers.

## Accessibility persona — verdict: DO NOT EXIT PLAN until A11y-1. Fix first: A11y-1.
A11y-1 Critical: terminal cursor never set (view.go:41-42, broadcaster.go:564-565; entries draw glyph modal_location.go:304, setup_form.go:66); 4 new focus points (FR-1.14 target, FR-1.10 find, FR-1.16 frequency, FR-1.13 hour); only ack places cursor (FR-11.4); fix is F-205/AX-3 in O2; M5b graded by a sighted reader of --ascii. Action: O1 row cursor on focus/insertion/changed words; "cursor" in FR-1.12 closed set; AX-3 before W6; HUM choice: M5b with VoiceOver + covered display, or VALIDATE says sighted reader.
A11y-2 Important: "on demand" (A-26) and "retry" (architecture.md:226) have no key; add "refresh now" to FR-1.14.
A11y-3 Important: Broadcaster has no in-app words: "Instead of the map" Observer-only (setup_rows.go:258); D-58 shared, D-77 retired its words; FR-2 no words row; requirements.md:14,21 promise both; new Settings' mode scope unstated. Ask HUM whether D-58's shared Setting stands; FR-2.3.
A11y-4 Important: requirements.md:101-102 promises withdrawn device Setting (D-85) (= Business F4).
A11y-5 Minor: mode state (layer, hour, entry) not required in words; extend FR-1.14.
A11y-6 Important: reach field overwrites propagation fill (field.go:148-155); at 16/none colours drawn as contours (L-4.4 forbids for terminator); add L-row: reach edge non-colour mark, never a contour; TestTheReachIsMarkedWithoutColour + P0 specimen.
A11y-7 Important: P7 optimises per-cell contrast rule (contrast.go; frame.go:940) guarded only by benchmark; add whole-globe contrast test at every depth, both grounds, before P7.
A11y-8 Minor: words one block ~80 statuses; one line per target, name first, open bands first.
A11y-9 Minor: BandStatus.Limit/Reach.Limit free strings; typed value + watchpost words, or FR-3.2 test covers it.
A11y-10 Minor: refresh can move text being read; keep scroll/focus, say once it updated; TestARefreshKeepsTheReadersPlace.
Ordering: W8 no slot before UAT; three new Maps-tab Settings (AX-7 two-per-line).

## InfoSec persona — verdict: DO NOT EXIT YET (IS-1, IS-2 shape fixes before gate). Fix first: IS-1.
IS-1 Important: G1.2 bans `net` imports but Response.Header is http.Header (net/http) and url.Values (net/url); give Response its own type (rate headers + validators only, enforces FR-4.5); allow net/url only; ban net, net/http, crypto/tls, os/exec.
IS-2 Important: propagation client has no host allowlist (A-12 struck it); library can reach any HTTPS host; MAP STATUS test can't fail; fixed host set exported by go-ionomaps with terms (R-4.1), client refuses others. HUM (A-12 vetoable).
IS-3 Important: public API trusts caller: Options.Grid (0 / 0.01 -> memory), Bands/Path/Reach args unbounded, NaN mhz -> NaN SkipKm; R-3.5 + FuzzAnswersNeverReturnNaN; FR-1.16 state range.
IS-4 Important: FR-9.2 orphan say sweep: no ownership proof; TestAnUnrelatedSayIsNeverEnded.
IS-5 Important: UAT record (public repo) would publish origin, tower, callsigns; protocol states redaction (region / 4-char locator, no callsigns), raw logs outside tree.
IS-6 Important: tools/tables must read netCDF-4/HDF5 (magic \x89HDF) - stdlib can't; unruled dependency or Python in build chain; rule route: one-time export from pinned PyIRI env to committed plain text + checksums, or a ruled Go dep in a separate tools module.
IS-7 Minor: "no error quotes typed text" untested; quoted in resolver.go:108, geocode.go:56,59; httpx isSecretParam lacks name; TestNoLookupErrorQuotesTheQuery.
IS-8 Minor: httpx StatusError.Reason keeps 256 B of body (GIRO echoes IP); propagation client keeps no Reason or test covers it.
IS-9 Minor: FR-4.3 "no request names origin" untested; counting fetcher: answers make zero fetches.
IS-10 Minor: go-ionomaps NFR-1 govulncheck "from first dependency" vs G0.1 from start; align.

## Docs Quality — verdict: DO NOT EXIT PLAN. Fix first: F4 (=Business F1). week.py sections 1-7 reproduce exactly.
D-F1 Important: 00-REQUIRED-READING.md:17-33 lists none of PLAN's docs; add rows + phase state.
D-F2 Important: plan-questions.md status IN PROGRESS; Q-3..Q-6 no sections; Q-7 no options.
D-F3 Important: go-tuiMaps M1 (N, protocol, task) and M6 (run count) unset (=Business F6).
D-F4 Critical: D-103 forecast unbuildable (no yesterday field; 24 grids ~60 MB breaks D-39/D-95); G-G1 spike excluded forecast; options: staged (clim until 24h held), costed backfill, another method; re-check G1, G-G1.
D-F5 Important: FR-3 preamble device Setting (=Business F4, A11y-4).
D-F6 Important: dry-run.md:286 "floor met every day" vs :252 10-03 US foF2 0.93 vs 0.84.
D-F7 Important: Snapshot lacks F10.7 used (R-9.5), extrapolated flag (R-9.7), measured mask (R-3.2); Options lacks offset-correction ask (R-9.6).
D-F8 Important: plans drop required test names (watchpost: TestEachModeDrawsOnlyItsLayers, TestTheOriginIsTheSelectedPlaceOrTheTower, TestThePropagationModePansAndZooms, TestTheCentreAnswerIsSaid, TestAFailedFetchIsSaidWithItsPath, TestClosingTheAcknowledgementRecordsItSeen, TestTheAcknowledgementReadsWithoutThePicture, FR-10.3/10.4 tasks; go-ionomaps 13 incl FuzzNoNaNOrInfLeavesTheLibrary, TestEachBandsStatusForAPath, TestShortPathsUseFoF2, TestErrorsNeverQuoteInput, TestLowConfidenceReadingsAreDropped; go-tuiMaps TestAHostTypesLegendHasColoursAndUnit); mechanical trace check.
D-F9 Important: G10.4 oracle loo_hybrid.py (DISCOVER) -> week.py sections 2-4, 6.
D-F10 Important: NOTICE/credits lack IGRF-14 and Apex sample (=Business F11).
D-F11 Minor: dry-run kernel 4000/1.0 (:128) vs 2000/0.5 (:212); "not measured climatology" (:131) stale; RK-11/RK-12/:226 stale; R-5.6 readout/recorder; L-2.2 three colours vs one per class; go-ionomaps README "in discovery"/fallback; watchpost rulings.md:16 "go-giro-data (new)".
D-F12 Important: week.py sections 8/9 need undocumented 3rd arg; section 9 mislabelled quiet without it; document/derive disturbed days, fail if absent.
D-F13 Minor: D-103 margin "0.00-0.05" but +12h US MUF margin 0.06.
D-F14 Minor (HUM fork): dry-run opens with engineering table; plain verdict first?
D-F15 Important: targets table lacks blind spots: M1/M4 baseline never measured; G-M3 near-station from whole week incl tuning; Pacific hybrid MUF worse than clim (5.17 vs 5.09); one week/4 US stations. "measured on" notes.
D-F16 Important: forecast error source at run time unspecified; storm worse (=Business F3).
D-F17 Minor: FR-1.17 4.4 ms from placeholder formulas; "30-day mean" was 23-29 days (dsi starts 09-06).

## PLAN lens (Principal Architect) — verdict: DO NOT EXIT PLAN. Fix first: F8 (forecast's yesterday). week.py forecast tables reproduce.
P-F1 Important: M1/M4 (accuracy vs WSPR) have no instrument task in any plan (scenario set, +13 dB normalisation, climatology-baseline answers); add W10/G10 task.
P-F2 Important: Pacific hybrid MUF 5.17 vs clim 5.09 vs B on C 4.97 (test); Hawaii/Alaska are regions; HUM: regional-confidence statement, Pacific G-M3 row, or recorded acceptance.
P-F3 Minor: storm forecast worse than clim; offer climatology while G>=1.
P-F4 Important: one-week constants (offset seed -0.5, 2-SD 0.45, forecast typical error) never re-learned; record provenance, expiry test, say in words.
P-F5 Important: D-107 oracle tolerance set from measured agreement -> can't fail; derive from 0.1 MHz display via M3000 sensitivity, before measuring.
P-F6 Minor: climatology now main path but called fallback; failure rows for stale F10.7 and extrapolated coords.
P-F7 Important: API shape lacks point reading (R-2.1), many-points (R-2.3), measured mask (R-3.2), early/stale reason (R-5.4), F10.7 (R-9.5), extrapolated (R-9.7), reject counts (R-3.3); Limit free string -> typed.
P-F8 Critical: forecast unbuildable (= Business F1, Docs F4); also empties FR-1.15 open hours on first open; G-G1 cold open excludes 24 yesterday assimilations; options (a) yesterday from GIRO over climatology, (b) amend D-39 staged fetch, (c) persist (reopens D-78).
P-F9 Important: Update inside per-ask command that keypresses cancel (map_work.go:77-80, 30 s map_pane.go:53); 39-request burst ~20 s; separate update (one owner, own context) from pure answers; name what drives 10-min refresh.
P-F10 Minor: target misses (non-floor) have no route; O2 tied to libraries: ask HUM whether O2 can ship ahead.
P-F11 Important: G10.4 oracle loo_hybrid.py -> week.py; week data + PyIRI env only in the session's temp directory; evidence README says Py3.9/PyIRI 0.0.4; durable uncommitted location + env recipe.
P-F12 Important: dropped tests (= Docs F8).
P-F13 Minor: NOTICE IGRF/Apex; no NOAA-feeds risk row.
P-F14 Important: no sizing; go-tuiMaps M1/M6 unruled; batch count per WP vs RK-5.
P-F15 Minor: stale FR-3 preamble, RK-11/3/12, :226, R-5.6, plan-questions status.

## Performance persona — verdict: DO NOT EXIT PLAN. Fix first: P-1 (forecast inputs); re-rule D-95/D-98 same sitting with P-2, P-3.
P-1 Critical: history at open: 23 grids x 2.5 MB = 57.5 MB, ~3h50 at 6/h; ~0.9 s CPU vs D-98's 0.5 s; or no forecast for 24 h after every restart. (= Business F1, Docs F4, Lens F8)
P-2 Important: GloTEC names timestamped -> never a 304; newest grid needs the 505 KB index or guessed names with 20-40 min publication lag; 6/h decoded = 15.01 MB already over D-95 before GIRO/D-RAP/index; hourly ~3.1 MB > 3 MB; NOAA sends vary: Accept-Encoding, gzip 272,030 B (9.2x), Go asks gzip by default -> wire ~1.7 MB/h; state wire vs decoded, MB vs MiB; plan newest-grid discovery; measure gzip in BUILD's first batch; re-rule D-95.
P-3 Important: memory doesn't add up: library ~16 MB live (snapshot 3.3, clim 3.3, yesterday 3.2, body+decode 3.2, old snapshot 3.3) + host float64 conversion (x2, or 6.6 MB for 25 h) + GOGC 2x; D-95 "12 MB" underived; define "peak" (live heap), measure RSS in G10.3, split budget, hand go-tuiMaps only the hour shown.
P-4 Important: answers wait behind Update (GIRO burst sequential, 39 took 36 s); A-27 removed bound; journey has no threshold; workers on context.Background so close doesn't cancel; Update own command, answers from last snapshot, time-to-answer and time-to-first-picture targets, cancel on close.
P-5 Important: answer memo key lacks snapshot Computed; memo Fresh 1 h vs 10-min refresh; view in key -> reach recomputed on every pan; key on (snapshot, origin, hour, freq); add to FR-8.2 guard.
P-6 Important: Snapshot shares slices; concurrent updates; state immutable (fresh slices), -race test reader holds N while N+1 runs.
P-7 Important: all baselines on M5 Pro; ships linux/arm64, windows/amd64; M5 lacks terminator/night/reach; G1 harness presses no keys; measure on slower machine, scripted G1 workload, reach in M5 bench.
P-8 Minor: D-107 table size and lookup cost unmeasured; size in G3.5, re-check G-G1.
P-9 Minor: climatology cache invalidation key (UTC hour, F10.7 mean, month).
P-10 Minor: 1 deg unbounded; drop from v0.1.0 or bound.
P-11 Minor: "last readings held" unbounded; last per station.
Aside: blend's prediction same at every lead; +3/+12 floor is one forecast on two subsets.

## Project Hygiene — verdict: DO NOT EXIT PLAN (~one docs/evidence batch). Fix first: 5.1 preserve evidence. week.py/offset.py reproduce with right args; gate-runs rows match; plancode passes; atlas stale.
H-1.1 Minor: day.sh dead (never run); delete.
H-2.1 Important: week.py usage lacks 3rd arg; section 9 "quiet" includes storm without it (n=119 vs 63); README env Py3.9/PyIRI 0.0.4, "D-38 to D-40"; fix usage, refuse quiet without arg, record exact command.
H-3.1 Minor: atlas stale (pre-D-101, no scales); A-24 self-issued exemption; regenerate under full gate; docs-lane test diffing atlas.
H-3.2 Minor: plancode only watchpost; go-ionomaps no gate; go-tuiMaps v0.1.0 design docs have 4 blocks; run plancode over siblings or state gap.
H-4.1 Important: M1/M4 instrument unplanned (= Lens F1).
H-4.2 Important: 7 dropped tests + FR-1.12, FR-10.4 tasks; docs-reading test requirements names ⊆ plan.
H-4.3 Important: FR-3 preamble device Setting.
H-4.4 Important: G10.4 wrong script, inputs no durable home.
H-4.5 Minor: NOTICE IGRF/Apex; extend G0.3 to committed data.
H-4.6 Minor: RK-11, F-208, go-ionomaps plan :35 "only third-party data".
H-5.1 Important: evidence for ~10 rulings only in the session's temp directory (G1 harness, spikes, tmbench, SNR extracts, WSPR SQL, #27 measurement, PLAN request log 95 lines); commit to go-ionomaps evidence (no data).
H-5.2 Minor: floor "every day" overclaim (10-03).
H-5.3 Minor: spike kernel 4000/1.0 vs tuned 2000/0.5.
H-5.4 Minor: 30-day mean was 23-29 days.
H-5.5 Minor: plan-questions unfinished; D-100, D-104..D-106 no counter-argument recorded.
H-5.6 Minor: E-14 cited, error log stops at E-13.
H-5.7 Minor: requirements.md phase: DISCOVER; go-ionomaps requirements status B on D; README "in discovery"; dry-run "Summary so far"; REQUIRED-READING lacks PLAN docs.


## Code Quality — verdict: DO NOT EXIT PLAN. Fix first: C-M1 (forecast cold start). week.py reproduces every figure (39 s); lint-plan-code and lint-authoring pass.
C-U1 Important: dry-run.md:128 kernel 4000/1.0 vs tuned 2000/0.5 (:212); record shipped kernel in go-ionomaps (design.md:102 still undecided).
C-U2 Minor: plan-questions Q-7 body stale; Q-3..Q-6 no sections.
C-U3 Minor: architecture package list lacks scales, forecast (vs design.md:27,31).
C-U4 Important (SN-01): loo_hybrid.py named after wrong method; docstring says climatology, code GloTEC; rename, fix docstring.
C-M1 Critical: D-103 no data path on cold start (= B-F1).
C-M2 Important: third map mode on a boolean: radarMode() 28 call sites, 15 negations (map_prefs.go:633, map_pane.go:817); W2.0 mode enum + audit.
C-M3 Important: Snapshot contract incomplete (F10.7, extrapolated, measured mask, early reason, GIRO paused, D-RAP data for answers; no point reading, no many-points, no no-readings input; one Background for two fields; OpenHours "today" no time zone, past hours).
C-M4 Important: newest GloTEC grid discovery undefined (timestamped URLs, 505,115 B index uncosted).
C-C1 Minor: three point types; Field float32 vs tuimaps float64 conversion every request.
C-C2 Minor: offset.py re-runs week.py; factor loaders.
C-N (necessity): delete Snapshot.Age, Snapshot.Valid; ErrorMHz duplicates Forecast and blend error lead-independent -> one value, say which field; Options.Reference single ruled value; Options.Grid typed choice; R-7.3 SILSO/R12 cap dead under F10.7 (Important) delete or restate; yesterday(h) dead h; FastChar asks hmF2, MUF(D) unused after D-101; R-5.6 retired callers.
C-S Minor: Limit string -> enum; Type.Colours one per class vs L-2.2 three colours.
C-D Important: Field.Values "NaN for no data" vs R-3.3 no NaN leaves; mask or amend. Minor: blend docstring; NOTICE hmF2/IGRF/Apex; README "in discovery"; F-208 day-one numbers.
C-T1 Important: dropped instruments (watchpost 11 incl TestEverySourceIsCreditedOnce, TestTheMemoKeyCoversEverythingTheFrameShows; go-ionomaps 13 incl TestA429IsSeenByTheLibraryNotRetried; R-1.1 no WP; go-tuiMaps L-2.5); docs-reading diff test.
C-T2 Important: M1/M4 no instrument task; UAT prompter no task.
C-T3 Minor: G1 15 MB/h unit undefined (design spends ~15.26 MB).
C-P10: history.go:1478 discards readAs ok, make p10 doesn't see discarded bool -> add rule in W1.1; app.go:190-200 parseLatLon accepts NaN -> W0.3 states it's deleted.
C-E1 Important: week.py section 9 quiet includes storm without 3rd arg; make it required.
C-E2 Important: "30-day" F10.7 averages 23-29 days, no minimum; exit below N, re-score.
C-E3 Important: sections 2-5 not scored on common pairs (errors() drops keys); D-75 floor read from section 2; common pairs + assert.
C-E4 Minor: climatology chosen on whole week incl test days -> leaks into G-M3, floor; choose on tuning days.
C-E5 Minor: tuning on empty set returns first grid point (NaN compare); :235 prints unchecked cause.
C-E6 Important: dry_fetch.py continues to next GIRO station after giving up on 429 (D-39 says stop source); break.
C-E7 Important: G10.4 oracle loo_hybrid.py (one day, foF2 only, B on D, absolute noise; exits 0 on empty) -> week.py.
C-E8 Important: W8.2 SHIP gate has nothing to check (FR-7 instruments "as F-191"; follow-ups name no tests); name each row's test now.
C-E9 Minor: plancode scans watchpost only.
C-10 Important (evasion): wall-clock targets (M5 16 ms, G-G1 0.5 s/50 ms) "failing above target" on CI runners at -benchtime 1x -> machine checks or flakes; gate on allocations/work per cell; reference-machine timing a release step failing when record missing.
