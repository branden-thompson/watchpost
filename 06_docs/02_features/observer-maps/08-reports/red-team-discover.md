---
title: "0.18.0 — DISCOVER exit red team, rounds 1 and 2"
date: 2026-09-22
phase: DISCOVER (RCC) exit
sev: SEV-0
authority: HUM LEAD
status: "ROUNDS 1 AND 2 COMPLETE — every finding dispositioned; round 3 pending the HUM LEAD (find-rate still material)"
---

# DISCOVER exit — red team

**Five reviewers, none with prior context, each dispatched verbatim from `06_docs/red-team-brief.md`
in its own scratch directory.** The full axis set plus the DISCOVER lens plus the two personas the
HUM LEAD confirmed (`rulings.md` D-24 onward). Sectioned dispatch with independent verdicts, per
calibration; the code axis ran solo, as it always does.

| Reviewer | Verdict |
|---|---|
| Code Quality (solo) | **Do not ship** |
| Project Hygiene | Ship |
| Docs Quality | **Do not ship as the DISCOVER record** |
| Business Quality | **Do not proceed to PLAN** |
| DISCOVER lens (Distinguished PM) | **Do not exit** |
| InfoSec persona | **Do not proceed to PLAN** |
| Accessibility persona | **Do not proceed to PLAN** |

Six of seven verdicts were negative. Every finding below was **re-checked against the code before
remediation** (Verify-Before-Accept): two were downgraded on that check, and one was over-stated.

## Round 1 — disposition ledger

**Fixed in code.**

| # | Finding | Disposition |
|---|---|---|
| Code 9a | Fail-open: an unreadable `$HOME` returned `nil, nil`, so gates ran with half the rule and **printed a pass** — the exact class the package exists to close | **Fixed.** Returns the error; positive control asserts the failure, plus a control that an *empty* home still derives nothing without error (`trees.go`, `trees_test.go`) |
| Code 9b / Hygiene H3 | `make lint-identity` ran `-run PublishedTreeNames`, which does not match the new one-definition test: the gate named as enforcing the rule never ran it | **Fixed.** Recipe widened; the selection guard now requires the pattern to select **every** test declared in `identity_test.go`, discovered from the file rather than listed (`Makefile:123`, `gates_test.go`) |
| Code 9c | The one-definition test asserted a *substring*: deleting the live call and keeping its comment left it green | **Fixed.** It now asserts an execution with comment lines stripped, plus a control proving a commented-out call fails the check (`identity_test.go`) |
| Code 4 | Four of six guards added the same day cannot fire — `if expr == ""` is provably unreachable, which P10 Rule 5 forbids ("an assertion a static checking tool can prove can never fail violates this rule") | **Fixed by deletion (D-30).** Both unreachable guards and their tests removed; one **reachable** check added (an explicitly empty `HOME` argument) and P10 re-run: 0 findings. Recorded in the code comment as what it was — checks written to satisfy a density count |
| Code, unlisted | `panic` in a package-level initializer took down every test in `cmd/watchpost` on a plain permissions error | **Fixed.** Built once on first use (`sync.OnceValues`); the test that needs it fails with the reason (`identity_test.go`) |
| Code 3 | Two adjacent filesystem failures had opposite policies (one silent, one fatal) with no explanation | **Fixed** by the 9a fix: both are errors now |
| Code 1 | The retired workspace names were written as two concatenated string halves — a dodge around the identity scanner, and a pointless one, since that file is its own exemption row | **Fixed.** The literal is written plainly, with the reason. *(This row states the names as placeholders: this report is tracked and public, and the gate refuses the real ones here — see F-179.)* |
| Code 5 / Hygiene H3b | The self-test printed "8 leak class(es) fired" while counting probes, and probes only the static half | **Fixed (honest half).** It now says "8 static probe(s) … the derived half is NOT probed — F-177". The fixture probe is F-177 |

**Fixed in the record.**

| # | Finding | Disposition |
|---|---|---|
| InfoSec F6 | `seedZoneShapes` fetches zone geometry at station start, contradicting D-21's "no cache warming, of any source"; FR-3.2's instrument passed it by counting only basemap requests | **D-25.** Seed gated on the maps setting and first open; FR-3.2 and its instrument widened to every source |
| A11y A-1 / A-5 | A screen-reader listener had no path in any surface; `Describe` appeared in zero requirements | **D-26.** FR-7.4: the text description ships in macro phase 1, scored by M1's own question; discharges go-tuiMaps D-122's redirect |
| A11y A-4 | R-7.1's "readable without colour" was false: the library's hatch runs only at `NoColour` depth | **D-27.** FR-7.1 amended (depth hinted from `NO_COLOR` *and* theme) + **HR-7** for a depth-independent pattern |
| A11y A-2 | Colour-vision deficiency named nowhere, though the library ships a host-runnable check | **D-27.** FR-7.5: every palette passes `VisionSafe` in watchpost's own tests |
| A11y A-7 | The library ships `ReduceMotion`; the record never mentioned it, so stopping the loop cost the radar | **D-27.** FR-5.8: Settings row, rate ceiling, still form keeping the newest frame and its age |
| A11y A-3 | No-braille listeners get a notice only if they already knew to relaunch | **Fixed.** FR-1.8: the window says it draws in braille and names the remedy |
| A11y A-6 | "Passes the floor" is not "legible": a county view at 69×12 rendered entirely blank | **Fixed.** FR-1.9: a stated legibility tier |
| A11y A-9 | Map colour tokens would sit outside the existing contrast register | **Fixed.** FR-7.6 |
| Business B1 / PM | The bound drifted from "the United States" to "contiguous US"; Puerto Rico, Anchorage and marine zones are live in the tree | **D-28.** FR-2.1 is per-region over every US region and territory the APIs cover; FR-2.5 covers a location in none |
| PM D2 / Docs D3 | M1 had no grader, N or protocol; M5's targets were the library's, never derived for this host | **D-29.** M1: HUM LEAD as grader, recorded-scenario protocol; M5: target set at PLAN, marked as such |
| InfoSec F1 | No threat model: nine risks, none security or privacy | **D-31.** "What leaves the machine" table in the brief; RK-10, RK-11, RK-12 added |
| InfoSec F5 | Whether a source address is user-settable was unstated — a free-form URL would allow a planted map | **D-31.** FR-3.8: closed list in 0.18.0 |
| InfoSec F3 | No watchpost-side radar body cap (httpx would accept 32 MB, twelve times a loop) | **D-31.** FR-5.7: ≤ 1 MiB, never cached over cap |
| InfoSec F4 / F7 / F8 | Caches are an unexpirable record of where the listener looked, with no purge path | **D-31.** FR-3.9: stated retention and a clear path |
| InfoSec F2 | The library opens its own HTTP client, so NFR-4's User-Agent is unachievable | **HR-6** on v0.2.0; NFR-4 states the exemption until it lands |
| Docs H4 | Stale "D-1..D-16" tick; a decided question still written as open; a commit citation that is not the tag | **Fixed** in the brief |
| Docs D5 | "Four alerts in five" published bare in the brief and problem statement; wave 1 measured nine in ten, marine-inflated | **Fixed** at both sites, with the caveat |
| Docs D1 | The required-reading file was linked from nowhere | **Fixed.** Linked from `handoff-0.18.0.md` and the F-174 row |
| Hygiene H5 | Wave 1's numbers rest on live requests with no committed fixtures | **FR-8.6:** the recorded responses are committed before FR-5.3's tests depend on them |
| Docs D4 (fork) | No designer-facing artifact for a release whose output is a picture | **D-32.** The look belongs to PLAN, ratified from rendered specimens; the window follows the existing modal pattern with control commands along the bottom |

**Deferred, with a row.**

| # | Finding | Disposition |
|---|---|---|
| Hygiene H3b | The derived half of the identity rule has no probe | **F-177** — needs a fixture workspace; the fail-open path it would have caught is already fixed |
| Code 8, Hygiene H2 | `P10-02` "bounded by" annotations that are inputs rather than bounds; the mirror script shells out at import time | **F-178** |
| Hygiene H1 | `feature/map-ready-data` is a merged local branch still present | **Deferred:** branch hygiene at SHIP, with the spike branch |

**Declined, with a reason.**

| # | Finding | Why declined |
|---|---|---|
| Docs D3 "dangling D-30" *(the docs axis's fix-first item)* | **Over-stated, downgraded to Minor.** The citation reads "go-tuiMaps NFR-5, D-30" and D-30 **is** a go-tuiMaps ruling (`rulings-discover.md:26`, the time-to-placed-view numbers). The real defect is ambiguity — both repositories number rulings `D-n` — so cross-repo citations are now prefixed, as the v0.2.0 log already does. The record's "every ruling lands here" promise was never broken |
| Business B4 | A bearing-and-distance sentence as a cheaper answer than a map | **Adopted in part, not instead.** FR-7.4 makes the text answer a committed part of the release and the fallback for every failure mode — but not a replacement: the locked problem is about *seeing* where an alert is, and D-3 ruled the Observer map first |
| PM D3 | Radar traces to a preference ruling (D-7), not to the locked problem, and is separable | **Declined, with the rationale recorded.** The HUM LEAD ruled radar the most common map view (D-7) and required loops (D-10); the problem statement classes radar as context for the same question — where is it, relative to me. Recorded so the trace is explicit rather than assumed |
| Code 4 (partial) | Delete `Expr`'s empty-root guard as duplicating `Derived`'s, and `resolve`'s | **Declined at round 1 — REVERSED at round 2 (see the round-2 ledger).** A mutation test showed `Expr`'s guard is dead: `Derived`'s fires first and no test notices the deletion. `resolve`'s guard stands |
| Code 4 (partial) | Delete the `regexp.Compile` check as unable to fail | **Declined at round 1 — REVERSED at round 2.** Also mutation-tested: every consumer compiles the expression anyway, and nothing notices its removal. Deleted in `e1cc972` |
| Code 2 | The derived half differs per machine and is empty on CI, so the merge-blocking gate runs the weakest form | **Declined for now, stated instead.** That is the design (D-1's shape rule): CI has no workspace to derive from, and the static half is what holds there. Documented in the package header rather than changed |

## What the round changed, in one line

A consent promise contradicted by code already on the branch; a bound that excluded Alaska, Hawaii
and the territories; three classes of listener with no path; a metric with no grader; a gate that did
not run its own invariant; and four checks written to satisfy a counter rather than to catch a defect.

## Round 2

The multi-round rule (calibration: iterate on scope and find-rate) says another whole-base round is
due when the scope is foundational **or** the find-rate was material. Both hold: this round produced
Criticals, and remediation added requirements (FR-1.8/1.9, FR-3.8/3.9, FR-5.7/5.8, FR-7.4/7.5/7.6,
FR-2.1/2.5) plus HR-6 and HR-7 that have had no adversarial pass. **A round 2 with fresh reviewers is
recommended before the phase gate**, told the round-1 findings and fixes, asked to re-verify the fixes
hold, to attack what remediation introduced, and to say what they verified clean.


# Round 2

**Five fresh reviewers, same lens set**, each given round 1's findings and dispositions and asked to
(a) verify the fixes hold, (b) attack what remediation introduced, (c) re-test the declines, (d) find
what round 1 missed. Each was told to say what it verified **clean**, because convergence is only
readable if silence is distinguishable from not looking.

| Reviewer | Verdict |
|---|---|
| Code Quality (solo) | *(pending at the time of writing)* |
| Project Hygiene | **Do not ship** |
| Docs Quality | **Do not ship as the DISCOVER record** |
| Business Quality | **Do not proceed** |
| DISCOVER lens | **Do not exit** |
| InfoSec persona | **Do not proceed** |
| Accessibility persona | **Do not proceed** |

**The round-1 fixes were verified to hold**, individually, by the reviewers who re-checked them: the
fail-open path, the gate recipe and its discovery-based guard, the execution-not-mention test, the
deleted unreachable guards, the lazy rule build, the honest self-test count, and every record fix
except those listed below. That is the part of convergence this round did establish.

**What it also established: most of round 2's findings are in round 1's remediation.**

## Round 2 — disposition ledger

**Fixed.**

| # | Finding | Disposition |
|---|---|---|
| Hygiene 1 | **HEAD was red.** The round-1 report quoted the internal-tree names verbatim in a tracked public file; `make lint-identity` — a required gate, in `verify` and CI — failed at HEAD. `make verify` had passed **before** the commit because the gate scans `git ls-files` and the file was untracked | **Fixed** (names stated as a description). The instrument gap is **F-179** |
| Docs 2 / Business 1 / PM | The **brief was never fully amended**: it still stated the superseded bound in two places, claimed `--ascii` produces a readable map, ticked "HR-1..HR-5", cited the wrong commit in its completeness block, and defined M5 by the target D-29 disowned | **Fixed.** R-2.1 and the summary carry D-28's per-region bound; R-7.1 corrected and R-7.4/7.5/7.6 added; ticks and the tag corrected; M5 restated; the status line records the round-1/2 amendments |
| PM "what round 1 missed" | **The phase boundary no longer matched the requirements**: D-12 put `Describe`, settings and layers in phase 2 while FR-7.4, FR-9.1, FR-5.8 and FR-1.9 committed them, and only two of ~40 requirements carried a phase | **D-33.** The boundary is re-ruled and **every** requirement is tagged; phase 2's surface is named and currently empty by design |
| A11y F-1 | FR-7.4's acceptance ("from the description alone") is contradicted by M1's own anti-gaming clause ("the answer must come from the picture") — the release's only screen-reader requirement could be neither passed nor failed | **Fixed. M1b** added: same recorded scenarios, picture hidden, description shown, same grader |
| A11y F-4 | FR-7.5 cited `CheckRamp(..., VisionSafe)`, which does not compile, and named no ground or depth | **Fixed.** Stated as a palette × theme-ground × colour-depth matrix against the real signature |
| A11y F-5 / PM D1 | FR-5.8's "rate carried from NFR-21" cited a **flash** ceiling for a **loop** rate, and stated no number | **Fixed.** Split: FR-5.8 is the switch and the still form; **FR-5.9** is the rate, marked NO INSTRUMENT YET, with **HR-9** asking v0.2.0 for loop-rate control |
| A11y F-6 | A Settings row labelled *motion* reaching only the map, while the Observer animates at 300 ms and 50 ms | **Fixed** in wording (the row is the map's) + **F-180** for an app-wide switch |
| InfoSec F-1 | FR-3.8's "closed list" had no members and a negative-only instrument | **Fixed.** The list is enumerated as a table (source, scheme+host, what it feeds) with a **positive** instrument |
| InfoSec F-3 | FR-5.7's "never cached over cap" cannot hold through `httpx`, which caches before returning and has no per-request cap | **Fixed.** The cap is enforced inside the client as it reads; the instrument asserts the **cache is empty** after an over-cap response |
| InfoSec F-4 | FR-3.9's retention had no number, and the library's `CacheRoot` takes bytes only — no age, no purge | **Fixed.** Numbers stated (7 days map, 2 hours radar) + **FR-3.10** recording that the map half is unenforceable until **HR-8** lands |
| InfoSec F-5 | NFR-4's exemption cited the wrong cause and was wider than the defect | **Fixed.** The cause is the internal type inside the exported alias; the exemption is scoped to basemap tiles; the interim user-agent is recorded |
| Docs 3 / Hygiene 4 | M1's protocol depends on alert fixtures that FR-8.6 did not cover; the required-reading file still taught the superseded bound | **Fixed.** FR-8.6 widened to M1's scenario set; required reading rewritten for D-25, D-26, D-28, D-31, D-33 |
| Docs 2 | `wave1-findings.md` still raised three questions as open that had been ruled | **Fixed**, each marked with its ruling |
| Business 3 | D-20 superseded by D-26 with no supersession row; no user-facing disclosure of egress | **D-34** records the supersession; **FR-9.4** commits to telling the listener what a map open sends, and to whom |
| Business 2 / PM 6 | RK-5 mitigated a single-source basemap with "the source is a setting", true only for radar | **Fixed.** Split: RK-5 is now the single-basemap risk, with a fallback named as PLAN's first network question |
| PM 3 | RK-4 carried no drop-dead rule for a v0.2.0 slip | **Fixed.** A ship-without-radar criterion: alert areas + the description satisfy the locked problem; radar follows in 0.18.1 if v0.2.0 is not tagged in time |
| Hygiene 5 | M1's protocol had gained a scenario D-29's verbatim ruling did not contain | **Fixed.** Extensions are marked as such, and the subject and count are stated with the honest limit — the grader authored the scenarios |

**Deferred, with a row.** F-179 (the gate cannot see an about-to-be-committed file), F-180 (an
app-wide motion switch).

**Declined, with a reason.**

| # | Finding | Why |
|---|---|---|
| A11y F-2 | A screen-reader listener still cannot reach the description in-session, because `--ascii` is a restart flag | **Accepted as a finding, not declined** — FR-9.1's Settings rows now include the description's mode. Recorded here because the reviewer offered it as a fork and it was taken |
| A11y F-3 | The too-small notice should carry the description | **Accepted**, folded into FR-1.4/FR-1.9's wording |
| A11y F-7 | The depth hint could override the library's own terminal detection over SSH | **Deferred to PLAN**, named in FR-7.1's instrument: the hint must not claim colour where the terminal cannot be asked. PLAN owns the detection order |
| A11y F-8 | `Describe` returns nothing when no places are registered | **Deferred to PLAN** as a wiring condition, not a requirement: FR-7.4's instrument fails on an empty description where an alert is in view |
| A11y F-9 | FR-7.4 names no owner for the description's wording | **Deferred to PLAN**, which owns composition — the requirement names the facts, not the sentence |
| Business 4 | Radar remains the cheapest severable block | **Declined again**, on the same ratified ground (D-7, D-10), now with a stated drop-dead rule (RK-4) rather than an open-ended dependency |

## Convergence

Round 2's find-rate was **material**, and most of it landed on round 1's remediation rather than on
the original record — which is the signal the multi-round calibration names: fixes create surface.
Round 3 is therefore due on the same rule, scoped to what round 2's remediation introduced (M1b, the
phase tags, the source table, FR-3.10, FR-5.9, FR-9.4, HR-8, HR-9, the brief's amendments) and to
re-verifying round 2's fixes. **The exit condition remains a round that returns only polish.**


# Round 3 — the convergence round

**Three reviewers** (code solo; hygiene + docs + business sectioned; accessibility + InfoSec
sectioned), each told round 1 and round 2's findings and dispositions, each asked to attack what the
remediations introduced — and each told explicitly that **saying "only polish" is the exit signal**,
because a reviewer who does not know the exit condition cannot give it.

| Reviewer | Verdict | On convergence |
|---|---|---|
| Code Quality (solo) | **Ship**, after one fix | "This round is convergence, with one exception" — 6 of 8 mutations killed |
| Hygiene | Ship, after one fix | not yet |
| Docs | Do not ship | not yet |
| Business | Do not proceed | not yet |
| Accessibility | Do not proceed | "not yet — but this is the last substantive round" |
| InfoSec | Do not proceed | same |

## Round 3 — disposition ledger

**Fixed — the two that mattered.**

| # | Finding | Disposition |
|---|---|---|
| InfoSec S1 | **`FR-9.4` did not exist.** The round-2 ledger claimed it. The edit matched a string that phase-tagging had already changed, and that one replacement carried no assertion while its siblings did | **Written**, with an instrument. The lesson is recorded in the method, not just the file: every scripted edit asserts its anchor, and a ledger claim is checked against the tree |
| Code F-1 | **A live false negative.** The private-copy check read the *stripped* text, so a retired pattern hidden behind a `#` passed — reproduced by the reviewer | **Fixed and re-verified**: the "is it run" question reads stripped code, the "does a copy survive" question reads the file whole |

**Fixed — the rest.**

| # | Finding | Disposition |
|---|---|---|
| Code F-2 | Marker discovery split files on blank lines, so a detached marker left the gate silently | Parses the syntax tree, **and** requires every marker written in a comment to be attached to a test — which immediately caught an unattached one in the prose above it |
| Code F-3 | An empty derived rule passed silently — the fail-open shape the package refuses | The command says so on stderr, distinguishing "a runner with no workspace" from "a workspace that does not match the convention" |
| Code F-4..F-7 | Polish: a built-up string delimiter, an unused parameter, a stranded comment, comments explaining absent code | All removed |
| Docs D-1 | M2 still defined by the superseded US-national bound in three places | Restated per-region at all three |
| Docs D-2 | The brief still carried D-12's phase boundary and "whichever macro phase PLAN puts `Describe` in" | Replaced with D-33's boundary; the `Describe` debt is scheduled, not floating |
| Docs D-3 | M1b existed in one file; the brief's metric list and tick omitted it | Added, with the count corrected |
| Docs D-5, D-6, D-7 | Stale rulings tick, a malformed table, out-of-order rows | Fixed |
| Hygiene H-1 | Two round-1 declines the tree now refutes, with no reversal row | **Reversal rows added above** — a decline that changes becomes a row, not an edit |
| Hygiene H-3, H-4 | F-178(b) no longer true; a comment naming the pre-fix recipe | Closed and corrected |
| A11y A-1 | FR-7.4's acceptance still cited M1, whose own rule forbids scoring text | Cites **M1b** |
| A11y A-2 | M1b's protocol shares M1's grader and sitting, so it would measure recall | Order rule stated |
| A11y A-3 | The spoken path was promised and never exercised | FR-7.3's speech guard extended over the description |
| A11y A-4 | A description re-derived per frame is a talking surface that never stops | FR-5.8's still form freezes the description too |
| A11y A-6 | The ship-without-radar path silently drops HR-7 and HR-3 | RK-4 states what that path costs beyond radar |
| A11y A-7, A-8 | No degradation order at the small end; the vision check omitted two axes | Both stated |
| InfoSec S2 | Retention numbers incoherent for radar; a third copy of the viewed-rectangle record unnamed | FR-3.10 widened to both halves and names the HTTP cache |
| InfoSec S3 | The effective tile host comes from the TileJSON document, not the configured address | Stated, with **HR-10** asking the library to keep that confinement |
| InfoSec S4 | A byte cap is not a decode cap | FR-5.7 bounds decoded dimensions too |
| InfoSec S5 | Remote attribution text reaches the listener's chrome | FR-3.7 bounds and neutralises it |
| InfoSec S6, S7 | The transport policy was asserted, not stated; the proxy is a fourth party | Stated in RK-11; the proxy noted |

**Ruled rather than fixed.**

| # | Finding | Disposition |
|---|---|---|
| A11y A-5 / Business B-1 | D-27's three-state motion row was narrowed to a boolean in place — self-issued, not ratified | **D-35.** The three states stand and **HR-9 widens**: the host supplies the frames, so the library exposes loop playback control (off / slow / normal) for every looped overlay. Watchpost does not throttle on its own side |
| Business B-2 | M1b was added by a review, not a ruling | **D-36.** Adopted as a primary metric, with "for now" recorded |

**Declined, with a reason.** Business B-4's suggestion — make `requirements.md` the one normative
site and have the brief cite rather than restate the bound, the metrics and the phase boundary — is
**accepted in principle and deferred**: it is the right cure for the drift class every round has
found, and it is a restructuring of an approved artifact that belongs at PLAN entry, not in a
remediation pass. Recorded here so it is not lost.

## Convergence

Round 3 found **two real defects** (a requirement claimed and never written; a live false negative)
and a long tail of one-sentence record edits. The code axis called it convergence with one exception;
the personas called it the last substantive round. Nothing reopened a class, no new excluded listener
was named, and no new egress party appeared.

**The exit condition is a verification-only pass** confined to what round 3 changed — not another
whole-base round. If that pass returns only polish, DISCOVER exits.
