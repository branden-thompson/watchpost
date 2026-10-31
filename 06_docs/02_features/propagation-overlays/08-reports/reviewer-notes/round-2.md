---
title: "0.19.0 DISCOVER exit red team, round 2: reviewer notes (condensed)"
date: 2026-10-06
phase: DISCOVER exit
sev: SEV-0
authority: HUM LEAD
status: "CONDENSED by the agent from the four reviewers' reports as they arrived; the full texts were returned as messages and were not saved."
---

# Round 2 reports (condensed)

## Personas (A11y/InfoSec/Perf) — PROCEED once N-1..N-7 dispositioned; no Criticals; fix N-1 first
Closure: A11y CLOSED A,B,D,F,G,H,K,L,M,N; PARTLY AX-C (no 'no selected place' case N-9), AX-E (watchpost words never say day/night; FR-3.1), AX-I (N-1); AX-J UNVERIFIABLE (cited in requirements :60 but not in round-1 AX list). InfoSec CLOSED IS-3,5,7,8; PARTLY IS-1 (N-5), IS-2 (N-11), IS-4 (N-2,N-3), IS-6 ('allowed hosts' still promised in brief GR-5 project-brief.md:140). Perf CLOSED F2,F3,F6,F9,F11, typed decode; PARTLY F1/F10 (N-5), F4/F5 (N-4), F7 (N-8), F8 (N-10).
- N-1 Imp A11y: screen reader (VoiceOver) speaks through default output device = broadcast audio (domains/radio/player/output.go:27-52 oto default device) -> notice reaches air for blind operator; FR-2.9 test can't see it. FORK: output-device Setting / state it / accept.
- N-2 Imp Perf (Y delete): watchpost ALREADY embeds GeoNames cities15000 (geodata/index.go:1-13; 34,106 rows; 13 MB RSS; 50 ms; +1.3 MB), loaded at dashboard start (app/dashboard.go:208); non-US refused by coverage gate (resolver.go:101-102). D-61 premise wrong (AGENT ERROR E-8). Restate FR-1.10 as mode-aware coverage gate; correct IS-4 text.
- N-3 Imp InfoSec: typed coordinates already go to Open-Meteo via data client (app/resolve.go:46 -> resolver.go:90-113 -> geocode.go:56); query quoted in errors (geocode.go:58,61; resolver.go:108); isSecretParam doesn't redact `name` (httpx.go:48-58); report parser accepts NaN,NaN (app/app.go:169). Shared finite coordinate parser before every resolver; coords never online; redact name; no error quotes typed text; Observer fallback NoCache.
- N-4 Imp Perf: read cap is decompressed (history.go:879); month roll-up 2° 17.5 MB, 1° 70 MB > 32 MB; 1° day doc 18 MB disk / ~50 MB []*float64; forecast hours recorded? (x25). Size split from grid step; name cap; say if forecast hours recorded.
- N-5 Imp: httpx Retry-After holds pacing on every lane up to 30 s (memo.go:138-140,:45); 5xx arms host avoidance (memo.go:148-149); non-200 returns no headers (httpx.go:838-847; StatusError no header field 301-311) -> R-5.1/R-5.3 can't hold; GIRO 429 pauses weather if shared client. FR-4.5: own client instance, no memo/pacing hold, status+headers every outcome; test headers on 429.
- N-6 Imp: backfill unbounded (range, budget share, 30 days ~120 h/1.8 GB), GIRO multi-day reply vs body cap, recorded hours lack provenance (background), FR-6.1 vs FR-6.6 contradiction. Max range, live first, cancellable, progress in words, provenance per hour, reword FR-6.1.
- N-7 Imp A11y: acknowledgement fires on every Broadcaster's first start (D-29 on) swallowing first keystroke; no cursor placement. FORK: trigger, closing key, cursor.
- N-8 Minor: per-pan answers (10 bands × P.533+absorption + continents) on UI path unbounded -> bound or off UI goroutine.
- N-9 Minor: no selected place -> description nil (map_describe.go:58-61); 4th case.
- N-10 Minor: Claim losers have no read-back; crashed claimant holds 10 min (history.go:1074); D-RAP not claimed.
- N-11 Minor: evidence commits X-Amz-Cf-Pop "MCI50-P4" (coarse location) 24x -> strip; rule runtime header logging.
- N-12 Minor: uncached D-RAP drops conditional GETs (144 full downloads/day; GloTEC ETag unused) -> validators in memory.
- N-13 Minor: FR-2.7 expand key, FR-2.9 acknowledge key not console actions; no console Help-fits; focused target not named in words.
- N-14 Minor: GIRO learns IP before acknowledgement (D-29 default on); FORK: window names hosts + Setting.
- N-15 Minor: problem-statement.md:55 G1 still "over one hour" with fixed targets vs D-49.

## PM (DISCOVER lens) round 2 — DO NOT EXIT; fix N2 first
Closure: CLOSED P1 (under ruling; weak discovery -> N1), P2, R1 (threshold -> N2), R2, R3, K1, K3 (PLAN commitment); PARTLY P3 (go-ionomaps brief :91 "bind whoever redistributes" vs D-69), E2 (space-weather scales/geomagnetic storms in no requirement), IR1 (fleet total unbounded; RK-2 mitigations per copy), K4 (go-ionomaps brief :72 still KC2G for G-M3), S1 (D-68's VALIDATE note not carried in M2/M5 rows); NOT CLOSED I1 do-nothing, E3 competitor comparison (status line "every HUM fork ruled" false); CANNOT AUDIT round-1 reports not kept -> commit them.
- N1 Imp: PS-1 user unevidenced (#25 by HUM, 0 comments); D-45 answered whether watchpost transmits; 97.113 still bars regular retransmission; D-29/D-48/M3/M1 built for this user. FORK: keep / aim at operator's own lawful two-way HF / readout default off.
- N2 Imp (FIRST): "open" undefined: WSPR spot (~-28 dB SNR) counts as reaches vs voice ~+10 dB (reviewer's general knowledge); D-RAP 1 dB threshold counted as closed. Define reference circuit (mode, power, SNR, dB threshold); count WSPR spots by SNR.
- N3 Imp: M3a can't fail (no false-alarm term); hours-ahead no metric; D-49 lets targets fit results; no kill/floor criterion. M3 precision term; forecast metric; pre-registered floor + consequence (fork).
- N4 Imp: consensus with GIRO assumed; fleet as "educational/non-commercial research"? fork with fleet scenarios.
- N5 Minor: intake docs pre-ruling: project-brief :13, :31-32, :141, :198-202, :226; problem-statement :7; REQUIRED-READING :24; go-ionomaps brief :12,:62,:76; F-174 "Likely 0.19.0". Docs pass or "superseded by requirements.md" banner.
- N6 Minor: backfill has no reader; scope creep; fork defer FR-6.6.

## Business round 2 — PROCEED after N1-N4, N7 dispositioned; fix N1 first
Closure: CLOSED F1,F5,F6,F8,F9,F10,F12; PARTLY F2 (M3a), F3 (wording N6), F4 (text not in tree), F7 (default never re-asked); NOT CLOSED F11 (not in any file), PM-I1/PM-E3.
- N1 Imp: D-29 default-on predates throttle, legal text, D-45, D-48; every Broadcaster incl FRS/GMRS-only gets HF readout, acknowledgement window on first launch, ~38 GIRO/h. FORK: keep on / off until a band picked / on with My HF bands empty.
- N2 Imp: M3a self-scoring; no forecast target; 80 m default -> daily sunrise sticky notice. Score M3a vs WSPR observed + false alarms; forecast target+cutoff; UX fork: predicted routine closures as schedule in readout, sticky notice only M3b/M3c.
- N3 Imp: scope +~1/3 (watchpost 54->72 rows; go-ionomaps 26->35; go-tuimaps 21->26); RK-5 mitigation thin; O2 waits; PM-I1 do-less fork now (N4/N5 candidates to cut); status line false.
- N4 Imp: global-grid recording + backfill have no reader (no trend view); hits #27 cap. FORK: record tower values now; grids+backfill when a trends view exists; FR-6.5 stays.
- N5 Minor: Maidenhead locators not offered (local parse, free). UX fork.
- N6 Minor: "targets set in DISCOVER" lingering; GR-5 allowed hosts; issues of record omit #27; F-174 "Likely 0.19.0"; no "Not in 0.19.0" list.
- N7 Imp: round-1 reports not kept (go-tuimaps kept reviewer-reports/). Add round-1+2 reports or state not kept and why.

## Docs round 2 — DO NOT PROCEED yet; no Critical; 8/9 Importants are doc fixes; I-2 needs ruling; round 3 not needed if fixes checked. All numbers reproduce; citations hold.
Closure: CLOSED F6,F11,F12,F13,F14,F16,F18,F19,F20; PARTLY F4(I-1),F5(I-2),F7(I-3,I-9),F8(I-5),F9(I-6),F10 (go-ionomaps brief :72,:128 KC2G); NOT TRACEABLE F1,F2,F3,F15,F17 (red-team-discover cites 15/20) (I-8). Record: wave2:176 "OQ-G2 to be ruled"; F-180 follow-ups:106 still D-256; go-ionomaps brief :91 RK-2 wording; bare D-40 (brief :95), D-41 (ps :39); "about 38" remains wave2:115,:172, reqs :186, follow-ups :92; wave2:209 "GIRO 125" sums 127; red-team-discover :78 R-9.3, :154 R-9.2 PLAN question (now M3000 row), :151 R-2.2 -> R-2.7.
- I-1 Imp: "targets set in DISCOVER" in 8 places: project-brief :13,:98,:259; problem-statement :7; REQUIRED-READING :24; go-ionomaps brief :12,:76,:157.
- I-2 Imp (FIRST): D-48 set M3b<=15, M3c<=90; D-49 (agent-authored text) listed M3b/M3c as unset -> contradiction; FR-2.6 hard-codes; problem-statement:54 "Proposed at intake". HUM FORK.
- I-3 Imp: G1 row problem-statement:55 "over one hour" % CPU MB vs :61/D-49/NFR-1.
- I-4 Imp: US scores WRONG: "1.06-1.28" are all-station bins; per station B-on-D AL945 0.52, EG931 0.81, IF843 0.51, MHJ45 0.53; "four US stations" omits Adak EA653 1.23, Kauai LL721 1.35, Guam GU513 1.76, Wake WA619 1.37. Fix reqs :187 + go-ionomaps :128; define "US".
- I-5 Imp: reqs:37 readout "picked bands" wrong (shows all; notice picked only); go-tuimaps brief :30 lists L-5 delivered, omits L-7; C-7 :97 "L-5's constraint".
- I-6 Imp: go-ionomaps has no backfill requirement (watchpost FR-6.6 needs library recompute past hours); OQ-G4 says last good field only.
- I-7 Imp: go-ionomaps GC-2 brief:91 "bind whoever redistributes" vs D-69.
- I-8 Imp: red-team-discover :7,:30 "Every Fix is applied" overstated; list all 20 DQ.
- I-9 Imp: F-208 "about 0.6 MHz worse" wrong: 0.26 (0.63 vs 0.37) or 0.39 on clim; limits.
- M-1 figures w/o committed script (wave2 biases/quality subsets/62%; 3.97/3.22) -> commit script; fix evidence README "every figure", ":32 B-on-D MUF is the dry run's".
- M-2 throttle inconsistently described (wave1:228 D-RAP under D-39 6/h vs D-48; REQUIRED-READING:40 omits D-48/D-51, "measured limit").
- M-3 R-8.2 benchmark no machine/script/n. M-4 F-174 "Likely 0.19.0"; F-187/F-188 not marked O3; F-190 CLOSED under Open. M-5 D-71 uncommitted at 393aeaf7 (now committed 6fc5c994). M-6 old name in headings go-ionomaps rulings :10,:14; go-tuimaps rulings :14. M-7 allowed hosts (brief:140, go-ionomaps brief:62); GC-1 "DISCOVER rules on protocol" (D-53 ruled). M-8 wave2:93 "16 in flight" is httpx.go:96; D-61 geocode.go:65 -> :62-63. M-9 39 live + 1 probe = 40 = cap; at 40 live -> 41 (PLAN question).
- Q4 fork: listener section first + gloss terms (A) or keep (B); other repos have no designer section. Q1 wave2 title "D-18 budget".
