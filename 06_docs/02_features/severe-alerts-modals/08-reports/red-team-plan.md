---
title: "Red-Team Verdict — severe-alerts-modals (PLAN, round 2)"
date: 2026-08-28
scope: "feature: severe-alerts-modals — 03-architecture-design/plan.md + 04-development/* (the plan's Go code attacked as a PR); DISCOVER docs as context"
mode: multi-agent (4 sectioned dispatches, 8 independent verdicts) + the writing-plans document reviewer
---

# Red-Team Verdict — severe-alerts-modals (PLAN, round 2)

> red-team: **SHIP-WITH-CONDITIONS (post-remediation)** · multi-agent · scope:feature ·
> personas:[Perf, A11y, InfoSec, JuniorDev] · lenses: Code Quality (solo, dispatched last against the
> remediated plan) · Project Hygiene + Docs Quality · Business Quality + PLAN phase lens (Principal
> Architect) · 4 personas · plus the `writing-plans` plan-document reviewer · HUM LEAD confirmed scope +
> personas ("GO") · every lens recorded `git status` (the plan was untracked at dispatch — Hygiene A-1 —
> and is committed now)

## Verdict: SHIP-WITH-CONDITIONS

Pre-remediation: the plan-document reviewer returned **22 fixes**; the four red-team dispatches returned
**5 NO-GO/BLOCK verdicts** (Architect, Hygiene, Docs, Perf, JuniorDev), 3 PROCEED-WITH-CHANGES (Business,
A11y, InfoSec), and the Code Quality lens — run *after* those were remediated — still returned **NO-GO**
with 24 findings, 3 of them "won't compile / will panic" in code the previous pass had introduced.
**All 100 findings are dispositioned: 88 Fixed in the plan, 9 Declined with rationale, 3 accepted as
documented limits, 0 dropped.** Every Fixed item is in the committed plan (`ed7f3af`, `5488f31`, `e3a01ee`
and this exit's commit). The exit is conditional on the BUILD-time measurements the plan itself demands
(budgets pinned after the spike, goldens recorded after the rail test) and on the P10 gate being run
explicitly per batch.

**The find-rate signal.** Round 1 (DISCOVER) found 85; round 2 found 100, of which ~40 were in code the
plan wrote — i.e. the plan *created* the surface round 2 attacked, exactly the Step 9 pattern. The
Code Quality lens ran last on the remediated text and still found three compile-level defects introduced by
that very remediation. **A third round is therefore due — at BUILD exit, on the code that compiles**, not
on this document again: the remaining risk is in execution, and the tests the plan carries (rail column,
renderModal purity, serialised publish under `-race`, positive-control guards) are the round-3 instruments.

## Findings ledger (by lens; dispositions)

### Plan-document reviewer (writing-plans Step 4) — 22 fixes, all applied

Missing test-file package clauses/imports (1.6, 1.5, 1.7, 1.10) · duplicate test names across parsers (one
table-driven read-only guard) · splice-marker placeholders (parsers restructured as `parse(body)` +
memo-calling `Fetch`) · `Duration.String` "15m0s" (`shortDur`) · `newSevereDeck` in tests (nil `now`
panic) · `fakeSource` (not `stubSource`) · nil `recentPipeline.pub` guard · one publish hook (the existing
`onPublish` closures, not a new `after`) · `RequestStats()` per host · `UnregisterTheme` shipped ·
unused imports in Task 3.2 · `railify` semantics (callers draw ▲/▼) · `--ascii` forms for rule/rail glyphs
· `setupState.hints` slice → `setupKey` projection · `nvoices` in the key · `-update-golden` flag + golden
test code · `severeBench` code · full `alertRecordLines` · the go-studs layering rule (composer moved to
`platform/render`) · NFR-4 gauge task (3.10) · NFR-7/NFR-10 trace rows.

### Business Quality (Distinguished PM) — SHIP-WITH-CONDITIONS

| # | Finding | Sev | Disposition |
|---|---|---|---|
| A-1 | `Updated` = publish time; a dead source republishes with a fresh stamp; a calm day freezes it | Critical | **Fixed** — `SourceHealth` per feed; `Updated` = newest OK fetch (dataAsOf); fetch-minute in the index key; category line states "NWS unavailable" (§5.9) |
| A-2 | NFR-4 gauge / NFR-7 / NFR-10 untraced | Important | **Fixed** — Task 3.10 ([S] gauge); trace rows |
| A-3 | NFR-1 fetch-count test vacuous | Important | **Fixed** — deleted; NFR-1 holds by construction (the deck has no client), stated in `severe.go`; deck cost benchmarked instead |
| A-4 | Parse memo + `Keep` fields are ruling-driven, deletable | Important | **Declined** — SAM-D-14/18 (retain prose; memo) are HUM LEAD rulings; the memo is ~40 lines keyed on httpx's own cache-hit fact |
| A-5 | 30 tasks vs a 13-task two-tab slice | Important | **Declined** — SAM-D-10 ruled six tabs; recorded as the considered smaller slice for a future re-scope |
| A-6 | Empty tabs still render on a calm day | Minor | **Declined** — SAM-D-16: tabs never disappear (stable ←/→); FR-14 text + the source line carry the state |

### PLAN phase lens (Principal Architect) — NO-GO → remediated

| # | Finding | Sev | Disposition |
|---|---|---|---|
| B-1 | `modes/tty` importing go-studs — a second kit consumer the lint cannot catch | Critical | **Fixed** — `render.SevereTable` + `render.Railify` in `platform/render` (the only consumer stays the only consumer) |
| B-2 | `publish()` computes outside the mutex; concurrent triggers can land a stale index | Critical | **Fixed** — `publishMu` held across read→compute→compare→send; `TestSevereDeckPublishIsSerialised` under `-race` asserting the final key |
| B-3 | `railify` reintroduced against FR-3's wording | Important | **Fixed** — FR-3 amended (objectives); the glyph-aware `Railify` is shared by RECENT and the window |
| B-4 | 27-field `modalKey` couples every modal; measured gain ~1 ms/300 ms | Important | **Declined** — AX-4 = A is HUM LEAD's ruling; mitigations kept: invalidation table with one row per field, `renderModal` purity test, `opts` keyed as a whole |
| B-5 | `RecordOf` × 500 per trigger asserted "microseconds" | Important | **Partially declined / Fixed** — records compose only on a *changed* index (the guard precedes them; `ByTab`/`Cap` moved below the guard too); `BenchmarkSevereDeckTrigger` measures the steady state |
| B-6 | P10 gate absent from `make verify` | Important | **Fixed** — `make p10` run explicitly per batch; folding it into `verify` logged as a 0.13.x chore |
| B-7 | Source outage / stall invisible; republish swaps the open record | Important | **Fixed** — source line; `applySevere` pins the focus by `Key` and closes a vanished record |
| B-8 | Window/rail arithmetic under-specified | Important | **Fixed** — worked 120×44 and 80×24 line budget in Task 3.5 |
| B-9 | Blend 0.6 vs 0.30; `CategoryTone(int)` re-splits the registry | Minor | **Fixed** — one `categoryBlend` const, documented; `CategoryTone(Token, dark)` |

### Project Hygiene — BLOCK → remediated

| # | Finding | Sev | Disposition |
|---|---|---|---|
| A-1 | The plan was untracked at dispatch | Critical | **Fixed** — committed (`ed7f3af`+) |
| A-2 | Employer name in `plan.md §4` and in the `97ac60d` commit body (public-tree scrub) | Critical | **Fixed** — heading reworded; the unpushed commit reworded (`ef1ea69`); tree scan clean |
| A-3 | Mock generator only in the session temp dir | Critical | **Fixed** — `02-analysis/mocks/mock.py` committed; §5 repointed |
| A-4 | Empty FULL DOCS folders untracked | Minor | **Accepted** — sibling convention; folders fill as phases produce artefacts |
| A-5 | Fixture triplication | Important | **Fixed** — one home, `domains/globalfeed/testdata/` (moved at PLAN exit); every cite updated |
| A-6 | DISCOVER C-20 undispositioned | Important | **Fixed** — ruled Declined-with-rationale in `plan.md §7b` |
| A-7 | NFR-7/NFR-10 trace rows | Important | **Fixed** |

### Docs Quality — BLOCK → remediated

| # | Finding | Sev | Disposition |
|---|---|---|---|
| B-1..B-5, B-10 | `plan.md §2.2–2.4` sketches drifted from the task code (Detail union, `Guard` signature, `ByTab` type, the deck struct, the `Empty` field) | Critical ×3 / Important ×2 | **Fixed** — §2.2–2.4 replaced by intent + pointers; the batch files are the single owner of code |
| B-6 | Source-health line absent from the mocks | Important | **Fixed** — §5.9 specimen |
| B-7 | Task-id style / FR-11 trace | Minor | **Fixed** |
| B-8 | Task 3.5 unexecutable without the renderer | Important | **Fixed** — with A-3 |
| B-9 | Budgets stated in three places | Important | **Fixed** — `plan.md §0` is the owner; objectives and Task 3.9 cite it, no numbers |

### Perf persona — BLOCK → remediated

| # | Finding | Sev | Disposition |
|---|---|---|---|
| P1 | `modalKey` non-comparable (`setupState.hints`); `voiceList` missing | High | **Fixed** — `setupKey` projection; `nvoices` |
| P2 | `detail.go` calls `time.Now()` directly — a clock the key cannot see | High | **Fixed** — routed through `d.now()` (Task 3.7) |
| P3 | `statsGen` per tick → [S] 100 % miss | Medium | **Fixed** — bumps only when the counters' fingerprint changes |
| P4 | `ByTab`/`Cap` above the change guard | Medium | **Fixed** |
| P5 | `severeRows` filtered 3× per frame | Medium | **Fixed** — bucketed once per `SevereMsg` |
| P6 | Budgets not derivable; Overlay unmeasured | Medium | **Fixed** — `BenchmarkOverlayOnly`; budgets zero until measured |

### A11y persona — PROCEED WITH CHANGES

| # | Finding | Sev | Disposition |
|---|---|---|---|
| Y1 | Category line below the tab row | High | **Declined** — HUM LEAD's mock (M-2/M-3) puts the tabs first; the tab row itself reads "› Warnings" first for a linear reader |
| Y2 | Blend documented two ways | Medium | **Fixed** (one const) |
| Y3 | `[enter]` chip live in the empty state | Medium | **Fixed** — `KeyCapIf` mutes it |
| Y4 | Selected-tab tint == panel bg (no-op) | Low | **Fixed** — the chip wears the unmixed tint. *BUILD note (R3-C-03): the first P3 cut tinted the chip bold-white only; restored at BUILD exit.* |
| Y5 | `[lt/rt]` opaque | Low | **Fixed** — `[left/right]`, `[up/down]`. *BUILD note (R3-C-15): P3 followed mock.py's `[up/dn]`/`[lt/rt]`; restored at BUILD exit with a label ladder so the row fits the 80-col floor.* |
| Y6 | Storm-name pronunciation untested | Low | **Fixed** — `synth.Pronounce` pass-through assertion |

### InfoSec persona — PROCEED WITH CHANGES

| # | Finding | Sev | Disposition |
|---|---|---|---|
| S1 | Speech path un-`Plain`'d; 0.13.0 routes provider names into it | High | **Fixed** — `render.Plain` in `eventNarration` + an OSC-52 fixture |
| S2 | Feed `Superseded` and `Guard(locs)` never unioned | Medium | **Fixed** — merged before the location loop; test |
| S3 | Sender field asymmetry (`sender` email vs `senderName`) | Medium | **Fixed** — `SenderName` on both paths |
| S4 | `Totals` array length unlinked across packages | Medium | **Fixed** — compile-time array conversion assert |
| S5 | seen.json oldest-first eviction could re-fire a tone | Low | **Accepted** — bound documented in `07-readiness/gates.md` |
| — | OID grammar, `Sent` DoS, allowlist, no new deps, no TTS injection | — | verified clean |

### JuniorDev persona — BLOCK → remediated

| # | Finding | Sev | Disposition |
|---|---|---|---|
| J1 | "platform/* only" vs a go-studs import | High | **Fixed** — composer in `platform/render` |
| J2 | `setupState` slice resolution left conditional | High | **Fixed** — `setupKey` stated outright |
| J3 | `severeBench` described, not written | Medium | **Fixed** — code shipped |
| J4 | `make pty-severe` cited before it exists | Low | **Fixed** — marked "(new, Task 4.3)" |
| J5 | Alias note pointing at a lint that does not check aliases | Low | **Fixed** — removed with the composer move |

### Code Quality (dispatched last, against the remediated plan) — NO-GO → remediated

| # | Finding | Sev | Disposition |
|---|---|---|---|
| 1 | `synth.NormalizeForSpeech` does not exist | Critical | **Fixed** — `synth.Pronounce` (`normalize.go:304`) |
| 2 | `UnregisterTheme` used invented identifiers (`themes`, `active`) | Critical | **Fixed** — `themeTable`/`themeName` |
| 3 | `t.lastOK` nil-map write on the first cycle | Critical | **Fixed** — field deleted; `fetchedAt` alone serves `SourceHealth` |
| 4 | Rail off-by-one (caps one column right of the bars); goldens would freeze it | Important | **Fixed** — one `railCol`; `TestSevereRailIsOneColumn` |
| 5 | `Union` discarded the national CAP record when the location record won — gusts/hail/VTEC lost | Important | **Fixed** — `Detail.Severe` carried beside `Alert`; `capExtras` shared by both records |
| 6 | `indexKey` ignored `FetchedAt` → "Updated" frozen on identical rows | Important | **Fixed** — fetch minute in the key; test |
| 7 | `lastSnapshots` races the `lp.priority` assignment (immediate first publish) | Important | **Fixed** — assignments and reads under `lp.mu` |
| 8 | Byte-identical-twice test cannot fail once memoised | Important | **Fixed** — asserts on `renderModal` |
| 9 | 80-col DECLARED assertion true at every width | Important | **Fixed** — spread-header strings; 100-col arm strengthened |
| 10 | "same rows" republish test changed the rows | Important | **Fixed** — same feed, new health / new fetch minute |
| 11 | Verify order (1.7 before 1.3–1.5; 3.5→3.7 gate) | Important | **Fixed** — order notes in P1 and the index |
| 12 | `Classify`: `Statement` exact-match; `0,false` sentinel | Important | **Partially declined** — exact match kept (HUM LEAD named *Special Weather Statements*); `TabNone = -1` adopted |
| 13 | `SevereColumnNames` dead export | Minor | **Fixed** — deleted |
| 14 | `NoLeadingGutter` no-op; width math relies on a kit rule silently | Minor | **Disposition corrected at BUILD (R3-C-15)** — the flag exists in the vendored kit (`data_table_row.go` `gutterBefore`) and is what folds the marks/number/EVENT zone together; it is kept and named in the column spec |
| 15 | Two degrade mechanisms; comment contract false | Minor | **Fixed** — one ladder; `severeMinEventFloor` named |
| 16 | `hz()` unused where ASCII mattered; whitelist compensating | Minor | **Fixed** — `dash(o)`; `—` removed from the whitelist |
| 17 | `len([]rune(stamp))` + magic 10 | Minor | **Fixed** — `render.Width`; `severeTitleChrome`. *BUILD note (R3-C-14): regressed in the first P3 cut, restored at BUILD exit.* |
| 18 | Vestigial `i` param + history comment reprinted | Minor | **Fixed** — dropped |
| 19 | `--ascii` rail sweep missed `body.go:247` | Minor | **Fixed** — *delivered at BUILD exit (R3-C-10): `recentSection` takes its rail glyphs from `render.RailGlyphsFor(o.ASCII)`.* |
| 20 | Task 4.2 instructed instead of shipping `severeDetailLines` | Minor | **Fixed** — Plain at use shipped in 3.6 |
| 21 | Serialisation test proves monotonic gens only; no `-race` | Minor | **Fixed** — final-key assertion; `-race` in the verify |
| 22 | `modalKey` re-derives what `render.Opts` carries | Minor | **Fixed** — `opts` keyed whole; `frame` case added |
| 23 | `t.Cleanup` inside the loop leaks Monochrome | Minor | **Fixed** |
| 24 | File map MODIFY vs CREATE | Minor | **Fixed** |

## Axis / lens coverage

| Lens | Result |
|---|---|
| Plan-document reviewer | 22 fixes; API claims verified (httpx raw-body path, go-studs fields, `railify`, `publisher`, `snapshot.Alert`, test helpers) |
| Business Quality | 6 findings — traceability and honesty of the record clean; K2D acceptance-criterion framing held |
| PLAN phase lens (Architect) | 9 findings — the A-vs-B evaluation judged genuine; round-1 C-1/C-2/C-3/P4/P8/P9/S1/S3/S4/S5/C-17 re-verified in the plan's code |
| Project Hygiene | 7 findings — **zero AI attribution** across all commits and files (re-verified); no stray artefacts |
| Docs Quality | 10 findings — 12/12 line cites held; column numbers consistent across plan/mocks/code |
| Perf | 6 findings — memo target, window-only wrap, no-shimmer key, closed-frame path all clean |
| A11y | 6 findings — every state conveyed without colour; contrast ≈ 13.5:1 dark / 10.7:1 light; no WCAG failure |
| InfoSec | 5 findings + 7 clean — no exploitable path; no TTS injection (argv + tempfile/stdin, no shell) |
| JuniorDev | 5 findings — every Task 3.5 symbol resolves; live-vs-simulated data labelled |
| Code Quality | 24 findings — every cited signature/field verified; domain logic (`Supersedes`, `Cap`/`ByTab`, `Union` tie order, `severeMarks` width, degrade sets) clean |

## Summary

**100 findings: 18 Critical/High · 38 Important/Medium · 22 Minor/Low · 22 reviewer fixes.** 88 Fixed ·
9 Declined with rationale (A-4, A-5, A-6, B-4, Y1, #12-partial, and the three lens items that asked to
undo HUM LEAD rulings) · 3 Accepted limits (H-A4, S5, B-5-partial) · 0 dropped.

**Multi-round decision (Step 9):** round 3 runs at **BUILD exit** on compiled code — the tests the plan
carries are its instruments; a further pass over this document would re-read prose the two rounds have
exhausted.

**PLAN exited** — HUM LEAD approved the Plan of Record 2026-08-28 ("APPROVED; GO 4 BUILD"), with conditions: budgets pinned only after
the P3 spike; goldens recorded only after `TestSevereRailIsOneColumn` passes; `make p10` per batch;
the Linux halves (PTY journeys, R6) recorded as owed at BUILD exit if not yet run.
