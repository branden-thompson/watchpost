---
title: "Red-Team Verdict — severe-alerts-modals (DISCOVER)"
date: 2026-08-28
scope: "feature: severe-alerts-modals (docs on feature/global-ticker..HEAD; code read for evidence)"
mode: multi-agent (4 sectioned dispatches, 7 independent verdicts)
---

# Red-Team Verdict — severe-alerts-modals (DISCOVER)

> red-team: **SHIP-WITH-CONDITIONS (post-remediation)** · multi-agent · scope:feature ·
> personas:[Perf, A11y, InfoSec, JuniorDev] · lenses: Code Quality (solo) · Project Hygiene + Docs Quality
> (sectioned) · Business Quality + DISCOVER phase lens (sectioned) · 4 personas (sectioned) · HUM LEAD
> confirmed scope + persona set ("GO", SAM-D-23) · every lens recorded `git status` (clean at `e3cda5a`)

## Verdict: SHIP-WITH-CONDITIONS

Pre-remediation the round returned **four NO-GO verdicts** (Code Quality, DISCOVER phase lens, Perf,
InfoSec) and three SHIP-WITH-CONDITIONS; 85 findings, 13 Critical. Every finding was verified against the
code before acceptance (calibration: Verify-Before-Accept); **77 are Fixed in the documents**, 5 Declined
with rationale, 3 Escalated to HUM LEAD (rulings touched), 0 dropped. The three Criticals that would have
falsified PLAN's foundation — the memo target off the render path (C-1/P1), the pre-cap set with no
`Location` (C-2), the storm-name grammar break (C-3) — are corrected in FR-10/FR-11/FR-12. The exit is
conditional on HUM LEAD's rulings on E-1..E-3 and on three BUILD-time fixes to the reused `[A]` path (NFR-6,
NFR-12, NFR-13) that the feature must ship with.

## Findings (consolidated ledger, most severe first)

Disposition: **Fixed** (where), **Declined** (rationale), **Escalated** (E-n), **Deferred** (to which phase).

| # | Axis / Lens | Finding | Severity | Evidence (file:line) | Simplify/Delete? | Action / Status |
|---|---|---|---|---|---|---|
| C-1 · P1 | Code · Perf (convergence ×3 with B-F11) | FR-10 memoised `modalLines()`, whose only caller is the scroll clamp; the render path is `View→modalView→floatModalToned→wrapModal` — zero per-tick saving, RS-5 "MITIGATED" false | Critical | `modes/tty/view.go:100`; caller `nav.go:92`; path `view.go:29-38`, `:160-166` | Re-target | **Fixed** — FR-10 memoises `modalView(o)`'s result; RS-5 → HELD |
| C-2 | Code | The pre-cap set at `app/ticker.go:153` has no `Location`; `Locate` runs at `:168-169` after `Active` (`:155`) and the radius branch — the modal's designated source cannot produce the labels FR-9 needs | Critical | `app/ticker.go:153-169`; `event.go:81` | N | **Fixed** — FR-11: publish path runs `Locate` on the pre-radius set before filtering |
| C-3 | Code | FR-12 "one `Sentence()` owner" hides a grammar break: `Article()` keys on `Type[:1]` → "A Tropical Storm Dolly" | Critical | `domains/globalfeed/event.go:80-92`; `nhc.go:34` | N | **Fixed** — FR-12: `Name` field + name-aware `Sentence()`; golden written failing-first |
| A-F1 | Business | K2D baseline "∞ for an untracked-location event" is false for NWS events — `l` lookup opens Details for any location, then `A` renders the full record (≈ 5 actions + typing) | Critical | `modal_location.go:95-98`; `dashboard.go:218`; `alerts.go:131-236` | Y | **Fixed** — problem statement criterion 5, brief M1 v1.2.0, objectives §1: ∞ holds only for quakes/cyclones |
| A-F3 · S6 · S7 · C-18 | Business · InfoSec · Code (convergence) | `COV` = 100 % of *supplied* fields mandates rendering GIS/kmz URLs, seismic telemetry and attacker-named `parameters` keys the non-goals forbid | Critical | `data-shape.md §4`, `§2.2-2.3`; non-goals | **Y — delete** | **Fixed** — M2 redefined against a frozen render list; §4 carries Render/Keep/v2; URLs never rendered; `parameters` allowlist. Reading of SAM-D-14 **Escalated (E-2)** |
| B-F1 | DISCOVER lens | The modal is a symptom fix for 0.12.0's D5 place-tying; event-named narration would solve most of the pain without a modal | Critical | `global-ticker/objectives.md:85-93`; `app/ticker.go:381,:387` | Y | **Declined** — event-named narration is done *as well* (FR-12) and gives quakes/cyclones no record (§1); the modal is HUM LEAD's ratified answer (SAM-D-1..13). Recorded as a considered negative (objectives §10) |
| B-F2 | DISCOVER lens | The glyph-dense mock has no `--ascii` form though README:91-92 ships that affordance for screen readers | Critical | README:91-92; brief mock; no FR mentioned ascii | N | **Fixed** — FR-13, RS-12 |
| B-F9 | DISCOVER lens | The discovery probe found zero national severe events; the all-empty six-tab state was undesigned | Critical | `samples/nws_active_severe.json` (`features: []`) | Y | **Fixed** — FR-14, RS-13 |
| P2 · C-6 | Perf · Code | NFR-2's "miss ≤ 15 000" is a +42 % raise over the pinned 10 546 stated as parity; "hit ≤ 6 000" is the closed-frame constant copied, unmeasured | Critical | `bench_test.go:34-45` | — | **Fixed** — NFR-2: budgets measured by a PLAN spike, then pinned; modal-open row added; 80×24 worst case |
| S1 | InfoSec | NFR-6 is false today: `render.Plain` covers `Event`/`Description` only — `AreaDesc`, `Instruction`, `Headline` reach the terminal raw on the reused `[A]` path | Critical | `alerts.go:218`, `:226`, `:306` vs `:201` | — | **Fixed (spec)** — NFR-6 rewritten; BUILD fixes at one choke point + escape-injection test; RS-17 |
| S2 | InfoSec | §4 bounds cover only globalfeed structs; `snapshot.Alert` from `mapAlert` is unbounded (prose verbatim, slices uncapped) | Critical | `domains/weather/nws/alerts.go:88-110` | — | **Fixed (spec)** — NFR-5 covers both paths |
| S3 | InfoSec | The superseded set honours `references` from *any* feature; extending it to the location path would let a low-grade alert suppress a live warning | Critical | `nws.go:131-138`; `data-shape.md §5.3` | — | **Fixed (spec)** — NFR-12 guard (same sender + product + newer `sent`) on both paths; RS-16 |
| C-4 | Code | Enum + breadcrumb (A-3's pick) touches 5 switches; a bool touches 2 — chosen on discipline, not cost | Important | `dashboard.go:485-495`, `nav.go:20-23`, `view.go:41-60,:81-95,:102-113` | Delete the enum value | **Deferred to PLAN** as an approach comparison with the switch-count evidence recorded (FR-4, RS-9) |
| C-5 | Code | `bodyMemo` cite points at the tail of `bodyKey` | Important | `memo.go:29-41` vs `:45` | N | **Fixed** |
| C-7 | Code | "Reuse `railify`" is new code — table-shaped signature; `ScrollPanel` already draws the rail | Important | `body.go:290-293`; `panel.go:172`; `view.go:168` | Y | **Fixed** — FR-3 drops `railify` |
| C-8 | Code | RS-11 misrated L/L — `ModalTone(dark=false)` already serves light terminals today | Important | `sgr.go:189-194`; `dashboard.go:417` | N | **Fixed** — RS-11 M/M; `CategoryTone` substrate mandatory |
| C-9 | Code | Proposed `Quake.Mag float64` drops the parser's `*float64` | Important | `usgs.go:50` | N | **Fixed** — data-shape §4 |
| C-10 · J1 · J2 | Code · JuniorDev | Wrong cites: seam "`ticker.go:9-10`" (imports); memo `view.go:99` (comment); `toggleModal` "`:485-494`" (that is `handleKey`'s capture) | Important | `app/ticker.go:9-10`; `view.go:100`; `dashboard.go:484`, `:618` | N | **Fixed** — seam `dashboard.go:33-35`; `:100`; `:618` + capture `:485-494` named as such |
| C-15 | Code | Vacuous guards: token-independence passes on an empty set; the hit-rate test cannot fail; the `.expect` cannot exercise outer-tmux `ixon` | Important | `quattro.go` (no Ticker tokens); `soak-phases.expect` | N | **Fixed** — positive controls required (FR-6, FR-10); NFR-9 states the limitation |
| C-16 | Code | Untestable as written: FR-5 denominator is prose; FR-2 10-min rule has no clock injection; NFR-1 has no fetch assertion | Important | `data-shape.md §4`; `dashboard.go:302` (`now:`) | N | **Fixed** — table-driven fixture; `d.now`; httpx round-trip counter |
| C-17 | Code | Deletable now: new text token (`AlertModalText` exists); 6th Yellow token; new message type; new parse memo (`Memo[T]` is generic) | Important | `theme.go:89`; `dashboard.go:36`; `fire/memo.go:13-30` | **Y ×4** | **Fixed** ×3 (text token, message type, memo); 6th token **Escalated (E-3)** |
| C-18 | Code | Per-class structs (SAM-D-21) vs flat `Event` should be re-opened now that FR-5 is a generic per-field render | Important | `data-shape.md §7` vs FR-5 | Y | **Escalated (E-1)** |
| C-19 | Code | A 7th category touches 8 places with no registry; two competing classifiers | Important | `nws.go:26-37`; `data-shape.md §5.4` | Y | **Fixed** — FR-2 `severeTabs` data table (proposed) |
| H-A1 | Hygiene | 0.12.0 follow-up "multi-alert circle viz" carried no disposition | Important | `global-ticker/debrief.md:73` | N | **Fixed** — brief v1.2.0: Deferred |
| H-A2 | Hygiene | Decision-ID collision: feature D-15/D-19 vs watchpost-cli D-15/D-19 in source comments | Important | `dashboard.go:9-10`, `:207` | N | **Fixed** — `SAM-D-n` namespace |
| D-B1 | Docs | M2's fixture base `domains/globalfeed/testdata` does not exist | Critical (per lens) → Important (verified: fixture absent, metric unmeasurable *as written*) | `ls domains/globalfeed` | N | **Fixed** — samples committed under `domains/globalfeed/testdata/`; M2 cites them; BUILD promotes |
| D-B2 · C-11 | Docs · Code | 300 ms tick cited at `:341`/`:339` — `tea.Tick` is at `:330` | Important | `dashboard.go:330`, `:339-341` | N | **Partially declined** — `:339-341` is the predicate that *pins* the tick (the claim being made); **Fixed** by citing both |
| D-B3 | Docs | `:562` cite is the Details hydrate, unrelated to modal rebuild | Important | `dashboard.go:558-565` | N | **Fixed** |
| D-B4 | Docs | R-1 still says `ctrl+s` with no amendment | Important | brief R-1 vs objectives FR-1 | N | **Fixed** — R-1 v1.2.0 |
| D-B5 | Docs | Blue-token error left standing in the brief | Important | brief A-5 vs `theme.go:74` | N | **Fixed** — struck with pointer |
| D-B6 | Docs | Requirement-ID drift: R-12 / FR-12 / watchpost-cli R-12a | Important | three docs | N | **Fixed** — FR-12; borrowed id qualified |
| D-B8 · J5 | Docs · JuniorDev | Live evidence lives in an ephemeral session scratchpad | Important | `data-shape.md:4` | N | **Fixed** — samples committed |
| A-F2 · B-F3 | Business · DISCOVER lens | Problem rests on maintainer assertion; stakeholder set is n=1 | Important | brief §Stakeholders | N | **Fixed** — recorded honestly (objectives §5 Organisational) |
| A-F4 | Business | The parse memo exists only to service over-retention | Important | `data-shape.md §3, §6` | Y | **Partially declined** — prose retention is HUM LEAD's ruling (OQ-13/SAM-D-18); the memo is now a flag on httpx's existing not-modified fact (P7), not new machinery |
| A-F5 | Business | 3 of 4 metrics have no post-ship signal | Important | brief §Metrics | N | **Fixed** — relabelled acceptance criteria; `R6·PERF` the only post-ship metric |
| A-F6 | Business | Inaction cost unexamined; FR-8+FR-12 alone capture much of the value | Important | brief §If we do nothing | Y | **Declined** — see B-F1; costed as a considered negative (§10) |
| B-F5 | DISCOVER lens | Six-tab taxonomy locked (SAM-D-10) before population was measured; the live mix is marine/heat/air-quality | Important | `data-shape.md §2.3`; SAM-D-10 predates OQ-10 | Y | **Declined for v1** — taxonomy is HUM LEAD's ruling; accepted limitations documented (objectives §5) and FR-14 designs the sparse case; re-opened at REFLECT with a second season's sample |
| B-F6 | DISCOVER lens | No minimum-width requirement; the mock needs ≥ 117 cols | Important | `view.go:77-96` | N | **Fixed** — NFR-11, RS-14 |
| B-F7 | DISCOVER lens | No time-zone rule for rows without a tracked location | Important | FR-9; `alerts.go:186-194` | N | **Fixed** — FR-9 zone rule |
| B-F10 | DISCOVER lens | Seasonal variance unexamined; the 500 cap may never engage | Important | single probe | Y | **Fixed** — RS-13; cap kept as the P10 bound only |
| B-F11 | DISCOVER lens | RS-1/RS-5 "MITIGATED" by unwritten intent | Important | objectives §6 | N | **Fixed** — both HELD |
| B-F12 | DISCOVER lens | Missing risk categories: a11y, empty-state, narrow terminal, wrong baseline | Important | RS-1..11 | N | **Fixed** — RS-12..RS-15 |
| P3 | Perf | No open-modal allocation budget exists today | Important | `bench_test.go:111-118`, `:123-145` | — | **Fixed** — NFR-2 adds the row |
| P4 | Perf | Copying `bodyMemo` verbatim imports `shimmer` → 0 % hit while any row loads | Important | `memo.go:38`, `:62-64` | Drop `shimmer` | **Fixed** — FR-10 |
| P5 | Perf | `WrapLines` expands 4 000-rune prose × N rows for the whole body before clipping | Important | `text.go:98-114`; `view.go:104-115` | Y | **Fixed** — FR-10 wraps the visible window only |
| P6 | Perf | 133×44 is not the worst case; no 80-col budget | Important | `view.go:75-80` | — | **Fixed** — NFR-2 pins 80×24 |
| P7 · S8 | Perf · InfoSec | Hashing 1.1 MB per cycle is redundant — httpx already knows the body was not modified; the hash was unnamed | Important / Minor | `httpx.go:511-512`; `cache.go:222-232` | **Simplify** | **Fixed** — NFR-3 keyed on the httpx flag; sha256 fallback; error memo TTL-bound |
| Y1 | A11y | esc/esc undiscoverable — no detail-view chip set | Important | brief mock; FR-4 | — | **Fixed** — FR-4 chips `[esc] Back [esc esc] Close` |
| Y2 | A11y | No AA gate can see background tokens | Important | `theme_test.go:199-217` | — | **Fixed** — NFR-14 |
| Y3 · J4 | A11y · JuniorDev | FR-6 froze the Quattro `mix(p.red, p.darkBg…)` recipe, which is theme-dependent by construction | Important | `quattro.go:151-153` | — | **Fixed** — fixed RGB hues + `CategoryTone` substrate |
| Y4 | A11y | The same key opens a different tab by a 10-min window with nothing announcing it | Important | FR-2 | — | **Fixed** — first body line names the category; opt-out **declined** for v1 (naming suffices) |
| S4 | InfoSec | Bare `urn:oid:` suffix extraction with no grammar check → id collision + overwrite | Important | `data-shape.md §5.2`; `nws.go:161`; `alerts.go:90` | — | **Fixed** — grammar validation; trusted-source wins, merge never overwrites |
| S5 | InfoSec | `seen.json` is `0644`/`0755` with no size cap vs the `0600`/`0700` config precedent | Important | `app/ticker.go:478-479`; `config.go:217,:232` | — | **Fixed (spec)** — NFR-13 |
| J3 | JuniorDev | Proposed symbols read as existing (`handleSevereNav`, `modalSevere`, `CategoryTone`, `EventCat*`) | Important | zero grep hits | — | **Fixed** — every unbuilt symbol marked (proposed) |
| C-12 · C-13 · C-14 | Code | Off-by-one cites (`nws.go` 73→74 etc.; `alertNarration` 382→381; `helpGroups` 63→62; `clampField` 62-75→69; `types.go` 34-35→35-36; seen proof `:476`→`:477`); bare `ticker.go` names two files | Minor | as listed | N | **Fixed** — sweep applied; paths qualified |
| C-20 | Code | House style stamps release numbers into comments; new tokens would inherit it | Minor | `themes.go:60`, `theme.go:74` | N | **Deferred to PLAN** (comment intent, not release) |
| H-A3 | Hygiene | FULL DOCS folders are untracked-empty in a clone | Minor | `git ls-files` = 3 files | N | **Declined** — git tracks files, not folders; siblings follow the same pattern; folders fill as phases produce artefacts |
| H-A4 | Hygiene | Sibling 08-reports naming drift (`debrief.md` vs `reflect-report.md`) | Minor | four listings | Y | **Declined** — this feature follows the watchpost-cli lineage (template names); no fifth variant introduced |
| D-B7 | Docs | 15 000 miss budget silently exceeds the pinned 10 546 | Minor | `bench_test.go:34` | N | **Fixed** — with P2 |
| D-B9 | Docs | Perf baseline sourced to an agent memory | Minor | brief M4 | N | **Fixed** — cites `global-ticker/debrief.md:32-36` + the procedure doc |
| D-B10 | Docs | Brief appendix is superseded duplication hosting 2 of 3 bad cites | Minor | brief appendix | **Y — delete** | **Fixed** — retired with pointer |
| D-B11 | Docs | Objectives front-load identifiers before the "why" | Minor | objectives §2-3 | Y | **Fixed** — §0 plain-language block |
| B-F4 | DISCOVER lens | FR-10 specified mechanism while §8 routed memo scope to PLAN | Minor | FR-10 vs §8 | Y | **Fixed** — requirement column states the need; scope (family vs new) stays a PLAN approach |
| B-F8 | DISCOVER lens | English substring classification; 12-h clock; unsourced 10-min constant | Minor | `data-shape.md §5.4`; `ticker.go:390` | N | **Fixed** — documented as accepted limitations (objectives §5) |
| B-F13 | DISCOVER lens | Missing negatives — no rejected alternatives recorded | Minor | `data-shape.md §7` only | N | **Fixed** — objectives §10 |
| P8 | Perf | `tickNeeded()` does not list the new modal — an open modal with an empty ticker never repaints | Minor | `dashboard.go:336-341` | — | **Fixed** — FR-10 |
| P9 | Perf | Sort location unspecified; first `tz.Location` does disk I/O in `View()` | Minor | `platform/tz/tz.go:36-42` | — | **Fixed** — FR-3/FR-9: sort and zone resolution at publish |
| Y5 | A11y | NAR measured as a string count; "press w" assumes the listener is at the keyboard | Minor | brief M3 | — | **Fixed** — FR-8 spoken confirmation; M3 relabelled acceptance criterion |
| Y6 | A11y | Three tabs share Yellow; severity within a tab conveyed by nothing | Minor | brief R-6 | — | **Fixed** — FR-15 severity glyph |
| Y7 | A11y | No `helpGroups()` section named | Minor | `help_about.go:61-71` | — | **Fixed** — NAVIGATE |
| J6 · J7 | JuniorDev | Token names/registration sites unlisted; `TickerCategory` (4 lanes) presented as the six-tab enum | Minor | `ticker.go:31-42` | — | **Fixed** — FR-6 names tokens + sites; §5 states "new enum" |
| J8 | JuniorDev | One-probe facts justify design prose | Minor | `data-shape.md §2.3` | — | **Fixed** — labelled single-probe; seasonal sampling noted |

## Axis coverage

| Lens | Examined | Result |
|---|---|---|
| Code Quality (solo) | 12+ citations verified against source; necessity of 7 design commitments; maintainability; testability of FR/NFR; evidence soundness | 20 findings (3 Critical) — clean: TTLs, caps, `Merge`/`Within`, `Memo[T]`, `LocationTable`, `ModalTone`, `floatModalToned`, seen-store ids-only, the USGS declared/kept table |
| Project Hygiene | watermark scan of 2 commits + 3 docs; folder structure; stray state; naming; prior-finding disposition | 4 findings — **zero AI attribution**; clean tree; `.gitignore` covers logs/artefacts/`.DS_Store`; go-studs scrub holds |
| Docs Quality | 21 factual claims cross-checked; cross-doc consistency; audience; staleness; redundancy | 11 findings — 18/21 claims held; key/Blue/cap/OQ/metric definitions consistent after v1.2.0 |
| Business Quality | problem defence; traceability; observability; inaction | 6 findings — NFR-1 sound; 0.13.0 SemVer correct; verbatim decisions throughout |
| DISCOVER phase lens (Distinguished PM) | all 7 challenge areas + anti-patterns | 13 findings — research "unusually honest": contradicted its own brief twice in public |
| Perf persona | budgets, caps, memo target, hashing, per-frame I/O, worst-case width | 9 findings — `Memo[T]` reusable; NFR-1 holds; memo-key discipline sound |
| A11y persona | colour-alone, keyboard, contrast gate, timing, narration | 7 findings — category conveyed by label+glyph+bold; every control keyboard-reachable; `--ascii` fallbacks exist in panel primitives |
| InfoSec persona | input bounds on both paths, id collision, suppression, persisted state, memo hash | 8 findings — no credentials/PII; no shell construction from feed data; httpx validators bounded; no new dependencies |
| JuniorDev persona | empirical trace "add a seventh tab" (12 files, 4 mis-pointed) | 8 findings — live-vs-durable labelled; rulings verbatim; parser claims accurate |

## Summary

**85 findings: 13 Critical · 49 Important · 23 Minor.** Dispositions: 77 Fixed (in the three documents,
this round) · 5 Declined with rationale (B-F1, A-F6, B-F5, H-A3, H-A4) · 3 Escalated (E-1 per-class vs
flat, E-2 SAM-D-14 reading, E-3 Yellow token) · 2 Deferred to PLAN (C-4 design choice, C-20 comment style)
· 0 dropped. Partial: D-B2 (cite kept, both lines cited), A-F4 (memo kept per SAM-D-18, simplified).

**Convergences:** memo-target (Code · Perf · PM lens); COV over-reach (Business · InfoSec · Code);
wrong-baseline (Business) with the PM lens's "wrong problem" framing. **Contradictions surfaced:** A-3's
enum recommendation vs C-4's cost argument (to PLAN); SAM-D-21 vs C-18 (E-1).

**Multi-round decision (Step 9):** find-rate was material (13 Critical) and remediation added significant
new *specification* surface (FR-13..15, NFR-11..14) — a second whole-base round is warranted **at PLAN
exit** on the plan-of-record, where those specifications become designs; a second DISCOVER round on the
same documents would re-read prose the first round already exhausted.

**DISCOVER can exit** on HUM LEAD's approval of the discover-report, with conditions: rulings on E-1..E-3;
BUILD carries NFR-6/NFR-12/NFR-13 as mandatory fixes to the reused paths; PLAN opens with the perf spike
(NFR-2) before any budget is pinned.
