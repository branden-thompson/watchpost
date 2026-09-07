---
title: "Red-Team Verdict — multi-voice-support (BUILD exit)"
date: 2026-08-31
scope: "feature: multi-voice-support — git diff e31f8a5..HEAD (51 commits, 136 files)"
mode: multi-agent
---

# Red-Team Verdict — multi-voice-support, BUILD exit

**Ten lenses, dispatched blind to one another** (skill anti-pattern: anchoring). The four always-on
axes — Code Quality, Project Hygiene, Docs Quality, Business Quality — the BUILD phase lens
(Distinguished Engineer, all nine challenge areas), and five persona masks overlaid on the axes:
InfoSec, A11y, Perf, Junior-Dev, Safety-Critical. **~110 findings.**

Scope and persona set were confirmed by the HUM LEAD. Findings were verified by the orchestrator
before disposition — nine checked independently, eight upheld, **one refuted**.

## Verdict: SHIP-WITH-CONDITIONS

Round 1 was **NO-GO** on two lenses and ESCALATE on a third. Every Critical is now Fixed and proven,
and the four escalations were ruled by the HUM LEAD. What remains is a **round-2 convergence pass**
over the surface these fixes created — the skill's own stopping rule (Step 9) is a low find-rate, and
this round's rate was not low.

## Convergence — where independent lenses met

This is the part a single reviewer cannot produce, and it is the strongest signal in the set.

| Converged finding | Lenses that found it independently |
|---|---|
| The hourly GitHub check: no requirement, no opt-out, no disclosure, and the raw tag rendered into the update prompt | **Docs, Business, Hygiene, Junior-Dev, InfoSec** (5) |
| **The UAT rulings were never written down** — `"HUM LEAD, UAT <date>"` appears 98× as the governing authority and resolves to nothing | **Business, Hygiene, Junior-Dev, Docs** (4) |
| The `scope()` data race on the alert path, invisible to the `-race` gate | **Code Quality, Safety-Critical** (2, different proofs) |
| The CHANGELOG advertises a feature UAT deleted | **Docs, Business, Hygiene** (3) |
| The dip-vs-hold decision is made once and the source outlives it | **Code Quality, Safety-Critical** (2) |
| The ISSUES table clips its last column instead of dropping it | **A11y**, orchestrator round-0 (2) |
| Dead hooks left by the retired `[t]`/`[V]` modals | **Code Quality, Distinguished Engineer** (2) |

The second row is the batch's root cause: four lenses reached it from four directions, and it is why
`objectives.md` contradicted the build while four legitimate rulings looked like unauthorised scope.

## Findings — Critical

| # | Axis / Lens | Finding | Sev | Evidence | Simplify/Delete? | Action / Status |
|---|---|---|---|---|---|---|
| 1 | CQ + SC | `scope()` read `s.locs` unlocked on the alert path (3 sites) while `SetLocations` wrote it; the `-race` gate could not see it because every test that set a radius called `scope` directly | Critical | `app/severe.go:103,109,119` | N | **Fixed** `a51c34d` — `publish` resolves the default in the lock it already holds; the regression test enters the branch *through publish* |
| 2 | SC | An alert past ~cell 421 of a busy lane was never rendered; the scroll reset every rotation. 5 of 12 warnings shown over 8 rotations | Critical | `modes/tty/ticker.go:306` | Y — delete the reset | **Fixed** `a51c34d` — each lane parks its offset and resumes |
| 3 | SC | Setting an alert radius silently deleted zone-only NWS alerts (watches, most flood warnings, heat advisories) from the severe window | Critical | `domains/globalfeed/stack.go:36`, `app/severe.go` | N | **Fixed** `a51c34d` — an alert tied to a location in scope is kept; unknown distance ≠ out of range |
| 4 | DE | `enter` in Settings saved four groups of five and discarded theme/units/clock with the window state | Critical | `modes/tty/setup.go` — `uiApplyCmd` had one caller | Y — one exit path | **Fixed** `df8ebaa` — both exits write the same groups; a test asserts they agree |
| 5 | DE | `tea.Batch` ran two config writers concurrently; both read the file before either wrote it | Critical | `modes/tty/setup.go:164` | Y — sequence | **Fixed** `df8ebaa` — `tea.Sequence`; the test harness extended to drain it |
| 6 | SEC | `watchpost report` printed a provider's error text raw to stdout — an OSC 52 from a weather API reaches the user's clipboard | Critical | `modes/report/report.go:95` | Y — sanitize once at the source | **Fixed** `df8ebaa` — cleaned in the assembler, so no render site has to remember |
| 7 | PERF | Every allocation pin measured a dashboard with an **empty** marquee — the one state the app does not enter. All four hit pins fail against a live tape | Critical | `modes/tty/bench_test.go:benchDash` | No — add work | **Fixed** `e387db8` — fixture carries 30 alerts; every pin re-taken |
| 8 | PERF | The `[S]` modal memo could never hit: its key fingerprints a nanosecond uptime. 0 hits/20 frames, 223 MB/min while open | Critical | `modes/tty/memo.go:205` | Y | **Fixed** `e387db8` — 20/20 hits, 6,537 → 2,931 allocs/frame |
| 9 | PERF | The marquee tape was rebuilt twice per tick from an **uncapped** input: 4.1 allocs per item per frame | Critical | `app/severe.go:setLaneRows` | Y — bound it | **Fixed** `e387db8` — capped at `globalfeed.MaxEvents`, after the sort so the worst survive |
| 10 | A11y | Ticker lane identity was colour-only across six lanes: byte-identical with colour off, 1.16:1 on Monochrome | Critical | `modes/tty/ticker.go:153` | Y — no new mechanism | **Fixed** `e387db8` — the lane says its name |
| 11 | A11y | The `[S]` tables paint tokens on the **modal** ground; `aaPairs()` registered them against the window only. 14 pairs below AA, incl. the default theme | Critical | `platform/render/contrast.go:279` | Y — one line | **Fixed** `e387db8` — 14 → 0, verified |
| 12 | BQ | The `[S]` cast table, deleted at UAT, is the named surface of FR-7/10/13/14, NFR-5 and the M5 gate row | Critical | `objectives.md:45,48,51,52,62`; `gates.md` §2 | N | **Fixed** — **MVS-D-47** amends them to `report --verbose`; ruled by HUM LEAD |
| 13 | DQ | README taught the retired `T` key and shipped the `radio-min` screenshot — through a gate P4 recorded as passing at zero | Critical | `README.md:160,167` vs `p4-build-log.md:62` | Y — delete | **Fixed** `b24d330`; NFR-8 re-run to zero |
| 14 | DQ | README and CHANGELOG taught `M` as a mute; the in-app help string said so too | Critical | `README.md:107`, `dashboard.go:281` | N | **Fixed** `b24d330` — **MVS-D-48** records the behaviour change |
| 15 | DQ + PH + BQ | CHANGELOG advertised the deleted `[S]` blocks and described none of the 50 post-P4 commits | Critical | `CHANGELOG.md:17-19` | Y — delete the bullet | **Fixed** `b24d330` — rewritten against the shipped build |
| 16 | DQ + BQ + PH + JD + SEC | The hourly GitHub check: no requirement, no opt-out, no disclosure; the raw tag rendered unfiltered into the update prompt | Critical | `app/release.go:31,37,93` | Y | **Fixed** `df8ebaa` + `b24d330` — **MVS-D-52**: opt-in, disclosed, parsed numbers only |
| 17 | PH | 44 post-P4 commits had no batch row, no build log and no p10 run | Critical | `04-development/` stopped at P4 | N | **Fixed** `b24d330` — `p5-build-log.md`, batch row, `p10-p5.json` |

## Findings — Important (fixed)

| # | Axis / Lens | Finding | Evidence | Status |
|---|---|---|---|---|
| 18 | SC | The 30-event cap sorted recency-first, evicting the most severe; whole lanes vanished while live | `globalfeed/stack.go:68` | **Fixed** `a51c34d` — survival is severity, display order stays recency |
| 19 | CQ + SC | The dip-vs-hold choice was fixed at admit time; a relay tuned mid-alert was paused at full volume, a synth fallback played dipped for life | `app/radio.go:418`, `engine.go:403` | **Fixed** `ef088ef` — the engine decides, re-read every watch tick |
| 20 | CQ | `[S]` issue order was nondeterministic (map + tying sort): 6 distinct orderings in 200 identical calls, and the 8-row cut hid a different class each render | `modes/tty/status.go:735` | **Fixed** `67d92c5` — first-seen order, stable sort, total tiebreak |
| 21 | CQ | A folded issue mixed the first occurrence's status/blame with the last one's message — the blame column can name the wrong side | `modes/tty/status.go:721` | **Fixed** `67d92c5` |
| 22 | JD | Rows were styled by cell **index** while the column set varies by form: muting "the second cell" muted STATUS at 80 columns | `modes/tty/status.go:299` | **Fixed** `67d92c5` — cells carry their tone |
| 23 | A11y + orchestrator | The ISSUES table had no width ladder; `SEEN` was clipped mid-token at the 80×24 floor | `modes/tty/status.go:645` | **Fixed** `a51c34d` — the ladder its siblings have |
| 24 | A11y | No reduced-motion escape; the 0.9.0 deferral rationale ("no animation exists") was made false by the ticker | `red-team-build.md:38` (0.9.0) | **Re-dispositioned at round 2 — MVS-D-55**: `--no-animation` shipped in `a51c34d` froze the band at its first alert, so it was withdrawn before release. The reduced-motion surface is `[w]`, a navigable table of the same alerts, which the spoken reports already name. |
| 25 | SEC | `[S]` rendered the full failure message with no plaintext boundary — OSC 8 and bidi reached the terminal | `modes/tty/status.go:728` | **Fixed** `df8ebaa` — same single boundary fix as #6 |
| 26 | SEC | `release.go` claimed it sends no version while sending `watchpost/0.9` — stale for five releases | `app/app.go:29` | **Fixed** `df8ebaa` — the version is gone from the UA |
| 27 | JD | `extending.md`'s add-a-colour rule was wrong in four ways on the path the next contributor walks | `docs/extending.md:69-74` | **Fixed** `b24d330` |
| 28 | JD + DQ | The flow map covered no surface this release added, and its one new row named a deleted one | `docs/where-things-happen.md:37` | **Fixed** `b24d330` — four rows added |
| 29 | DQ + JD | Four file headers described designs the code no longer has | `status.go:3`, `columns.go:4`, `theme.go:3`, `list.go:5` | **Fixed** `b24d330` |
| 30 | PH | No UAT disposition record; 34 findings and 7 scope rulings lived only in commit subjects | `grep "UAT #" 06_docs/` → nothing | **Fixed** `b24d330` — `p5-build-log.md` §1–2 |
| 31 | PH | The p10 evidence chain is broken: `gates.md` records a `tree_hash` that is not a git object | `gates.md:28` vs `p10-p1.json` | **Fixed** `b24d330` — column deleted; the JSON is the evidence |
| 32 | PH | Carried follow-ups lived in an agent memory file outside version control, stale since before BUILD | — | **Fixed** `b24d330` — `06_docs/follow-ups.md` |
| 33 | A11y | `FETCHED` overflowed its 7-cell column past 100 h, losing the unit | `modes/tty/status.go:273` | **Fixed** `a51c34d` |
| 34 | A11y | `warnGlyph` was a second private owner of the alert mark | `modes/tty/ticker.go:61` | **Fixed** `e387db8` — deleted; the glyph set owns it |
| 35 | CQ | 33 % comment density with 106 `HUM LEAD` / 97 `UAT <date>` stamps and 62 history-narrating comments (`AP-HIST-01`) | measured over the diff | **Fixed (partial)** `a51c34d` — 102 stamps swept from the 0.14.0 surface; the rest is **F-12** |

## Findings — Declined or Deferred, with reasons

| # | Axis / Lens | Finding | Disposition |
|---|---|---|---|
| 36 | SC | `a2dh` on `$PATH` has no `p10` subcommand, so `make p10` cannot pass | **Declined — refuted.** The framework build has it; `gates.md:5-6` already documents `A2DH=<framework build>`, and every result in this release was produced that way. The PATH install being a month stale is a developer-environment note |
| 37 | SEC | `StatusError`/`ReachError` format `e.URL` rawly | **Declined — verified holding.** Both construct from `req.safe` (`RedactURL`'d at `httpx.go:488`), confirmed independently by the InfoSec lens and the orchestrator |
| 38 | BQ | Per-class tones answer "which class", not the locked "alert vs report by ear" | **Declined by HUM LEAD** — the taxonomy stands on its own merits (MVS-D-31/E-3 already ruled it) |
| 39 | BQ | `WATCHPOST Observer` brands a product line against a product that does not exist | **Declined by HUM LEAD** — a deliberate delineation so the Broadcaster edition arrives as a word, not a rename. The lens was right that no ruling was *recorded*; **MVS-D-50** and the P5 log fix that |
| 40 | BQ | `units`/`clock` were ruled BACKLOG (MVS-D-21) and built anyway | **Declined — ruled, not unauthorised.** The ticker's times were wrong without a preference to follow; **MVS-D-49** records the overturn |
| 41 | BQ | `ticker_radius_mi` silently changed meaning | **Fixed as a record item** — **MVS-D-53**; the CHANGELOG names the surprise |
| 42 | DE | Six independent `Load`→mutate→`Save` config owners | **Deferred — F-1**, named THE architectural risk for 0.15.0. Both live defects it caused are fixed |
| 43 | DE | Three product classifiers over the same NWS strings; divergence already live | **Deferred — F-2** |
| 44 | DE | `app/release.go` is a network feed outside the provider seam | **Deferred — F-3** |
| 45 | DE + PERF | `[S]` builds its tables 3–4× per render, and has no pin | **Deferred — F-4**; the memo fix reduced the rebuild rate ~3× |
| 46 | DE | **No `before-you-write-code` gate was recorded for this feature**, while two other features in this repo carry full READY verdicts. Its library-safety item, applied to `tea.Batch`, is exactly the check that would have caught #5 | **Accepted, not retro-fitted.** Recording a verdict after the fact would be a lie; the gate runs at 0.15.0 entry. `AP-ASSUME-01` |
| 47 | BQ + SC | M3's two listening trials are unrun, and n=10 cannot separate 9 from 8 | **ESCALATE — HUM LEAD.** The only measure of the locked problem, and the only evidence that will exist |
| 48 | SEC | The public branch carries the developer's home path in committed profiler dumps; internal framework naming in published docs | **Deferred to SHIP** — a publish-branch scrub, not a code change |
| 49 | SEC | Same-origin redirects compare `Hostname()` not `Host` | **Deferred — F-8** |
| 50 | PERF | A degraded host is no longer negative-cached | **Deferred — F-11**, recorded rather than left undocumented |
| 51 | — | ~60 further Minor items (dead hooks, unreachable branches, `StripSGRForTest`, `º` vs `°`, the seen-store cap, `pipeRow`'s dead parameter) | **Deferred — F-5..F-15**, or folded into the fixes above |

## Axis coverage

- **Code Quality** — clean on `clock.go`'s layouts (verified across all three conventions and midnight), `StripSGR`'s differential pin, `numRecent`'s equivalence, and the director's suspend/resume balance. Findings: the race, the hold lifetime, issue order and fold consistency, dead code.
- **Project Hygiene** — the eleven PLAN escalations are exemplary and traceable (E-1…E-11 → MVS-D-29…46); no attribution trailers; the p10 ledger is sound. The BUILD-phase UAT round was the sole gap.
- **Docs Quality** — `docs/accepted-costs.md` verified accurate against the code. Everything user-facing had drifted.
- **Business Quality** — the deleted `[S]` surface against five locked requirements; four scope changes needing recorded rulings. All ruled.
- **Distinguished Engineer (phase lens)** — all nine areas answered. **The layering invariants held under 51 commits**: no `platform/` imports `domains/`, `snapshot` does not import `httpx`, `render` is the only go-studs consumer. The seams did not erode.
- **InfoSec** — proved holding: the FIRMS `map_key` never leaks across four failure modes including the full `[S]` render and the disk cache; the SGR scanner is byte-identical to the regexp over **400,000** random strings; disk permissions; body bounds; no new dependencies; the go-studs scrub.
- **A11y** — proved holding: the Monochrome guard is real, Settings selection is glyph-borne, blame is a word column, endpoint health carries a word beside its glyph.
- **Perf** — proved holding: the ticker lane fix is free (pin identical pre/post), `release.go` cannot leak, `status_table.go` is flat in width.
- **Junior-Dev** — the code explains itself well *inside* the files; the three pages that route a newcomer *to* them had lagged.
- **Safety-Critical** — proved holding: lane rotation is fair over 24 h simulated across 4 seeds; the expiry boundary is inclusive; superseded alerts cannot resurface; blame never errs toward "healthy". P10: 7/7 tools ran, 0 live.

## Summary

**Critical 17 · Important 18 · Minor ~75.** All 17 Criticals **Fixed and verified**. Of the
Importants, 18 Fixed, the remainder Deferred to `06_docs/follow-ups.md` with a stated trigger. Two
Declined as refuted, three Declined by HUM LEAD ruling, one **ESCALATE** open (M3).

**multi-voice-support can exit BUILD.** It cannot SHIP until M3's trials are run, the Linux protocol
is completed, and the publish-branch scrub (#48) is done — all HUM LEAD-owned — and a round-2
convergence pass has attacked the surface these fixes created.
