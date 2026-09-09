---
title: "0.15.0 — Pre-Broadcaster UI Improvements — PROJECT BRIEF"
date: 2026-09-07
phase: pre-DISCOVER
report_template: project-brief v1.1.0
level: LEVEL-1
sev: SEV-0
authority: HUM LEAD
directives: FULL GIT; FULL REPORTS; FULL DIAGRAMS; FULL RCC; FULL PLAN; FULL TDD
status: "Assembled and locked at intake 2026-09-07 — handed off to DISCOVER"
---

# New Major Feature | `0.15.0-pre-Broadcaster-ui-improvements`

**LEVEL-1; SEV-0; FULL GIT; FULL REPORTS; FULL DIAGRAMS; FULL RCC; FULL PLAN; FULL TDD**

## Summary & Intent

Before I build the Broadcaster UI, I want to close the design, performance and structure items that
Broadcaster would otherwise inherit and multiply. These land on both surfaces — Observer today, the
operator's console next — so doing them now is cheaper than doing them twice.

The reason this release exists in the form it does is issue #7. Alerts were never spoken on Linux.
That shipped in 0.14.0 through four red-team rounds, 171 mutants and a 127-row exemption ledger, and
it survived eight days. Nothing caught it. I caught it, on my own hardware, by waiting for a real
alert to arrive and noticing that the tone played and nobody said anything.

That is the shape of the risk in this product. Watchpost does not usually fail loudly. It fails by
not saying something, and a station that has stopped speaking looks exactly like an afternoon with
no hazards in it.

**If we do nothing:** Broadcaster adds a second writer to the config, a second caller into the audio
arbiter, a second writer of the band, and a screen full of new windows with new memo keys — on top
of code where each of those already has more than one owner and no check that the owners agree.
Every class of defect this release is meant to close gets a second instance, in a surface that
broadcasts over the air to people who are not looking at a screen.

### Locked Problem Statement

> **A person relying on Watchpost for hazard awareness cannot tell a quiet day from a station that
> has silently stopped telling them things.**

| # | Criterion | Score | Confirmation |
|---|-----------|-------|--------------|
| 1 | Bad Outcome | ✓ | Two states are indistinguishable and one of them is dangerous |
| 2 | Affected Humans | ✓ | The listener today; the operator and the channel listener at Broadcaster |
| 3 | Tech Agnostic | ✓ | "Station", "telling", "quiet day" are domain terms; no technology named |
| 4 | Non-prescriptive | ✓ | Says nothing about diagnostics, single owners, or memo keys |
| 5 | Verifiable | ✓ | Issue #7 — eight days silent on Linux, found only when a real alert arrived |

**Score: 5/5 — LOCKED.**

Everything in scope traces to that sentence, and the trace is what makes this a release rather than
a grab-bag. Two implementations of one rule is *how* a station silently stops telling you things —
that is literally #7's mechanism. A memo key carrying position instead of identity is *how* a window
silently shows a stale frame — #11, and two frozen windows before it. An AA register that shrinks
when a token is added, and `--ascii` scans that miss four windows, are *how* a surface silently
degrades. The diagnostics surface is the only item that attacks the sentence head-on: it makes the
difference observable on demand instead of waiting for weather.

**The scope test that follows from it: if an item's failure mode is loud, it is probably not 0.15.0.**

## Metrics of Success

| Name | Symbol | Type | Definition | Measured in |
|---|---|---|---|---|
| Time to confirm the station works | **T** | Primary | Wall-clock from operator intent to a verified end-to-end alert — tone, words, ticker and window — on their own machine. Today unbounded (you wait for real weather). Target ≤ 2 min. Lower is better | Operator stopwatch during UAT; the diagnostics surface's own bounded test event |
| Unexplained duplicate implementations | **D** | Primary | Count of operations implemented more than once with no written, ratified reason. Target 0. Lower is better | The duplicate-body / near-duplicate detection script from 0.14.1, run as a gate |
| Memo keys with no completeness guard | **K** | Secondary | Count of memo keys lacking a test that fails when the struct they key gains a field. Target 0. Lower is better | `modes/tty` completeness tests, extended per issue #12 |
| Gates never observed failing | **G** | Maintenance | Count of quality gates that have never been watched failing against a deliberately broken input. Target 0. Lower is better | `07-readiness/gates.md`, one evidence line per gate |

**Anti-solution checks run on each (refine-problem-statement Step 5):**

- **T** — the obvious anti-solution is a test event that runs on its own path and passes while the
  real alert path is broken. That is not a hypothetical; it is exactly what #7 was. **Hardened:** the
  test event must travel the same path a real hazard travels. A diagnostic with its own code path
  does not count as satisfying this metric.
- **D** — could be satisfied by writing a reason for every duplicate. **Hardened:** reasons are
  ratified by the HUM LEAD, not self-issued, on the same footing as a P10 exemption.
- **K** — a *fraction* of keys guarded rises when you delete memos. **Hardened:** stated as an
  absolute count with a target of zero.
- **G** — this is the standing "validate the instrument" rule expressed as a number. It has no
  anti-solution I can find, because the evidence required is a watched failure.

## Requirements

Six themes. Each row names its source so the trace back to a real defect stays visible.

### R1 — One owner per rule

Broadcaster adds a second writer to every one of these.

- **R1.1** Persisted config has exactly one write path. Six independent `Load` → mutate → `Save`
  owners write `config.toml` today; `savePreference` calls itself "the one path" and four siblings
  bypass it. *(F-1 — named THE architectural risk to carry into 0.15.0 at 0.14.0 BUILD exit)*
- **R1.2** One product classifier. Three independent substring classifiers run over the same NWS
  product strings, held together only by prose, and the divergence is already live —
  "Coastal Flood Statement" is `TabNone` but `ClassStatement`. *(F-2)*
- **R1.3** The band's single-writer rule is enforced by something other than a comment. *(F-22)*
- **R1.4** The audio arbiter's hold/resume/drop path is serialised against duck and restore. Those
  three functions touch the voice with no lock at all today. *(F-34)*

### R2 — Memo keys carry identity, not position

- **R2.1** Every memo key audited against the state its window actually reads; anything deliberately
  left out carries a written reason. *(GitHub #12)*
- **R2.2** A guard that fails when a struct gains a field its key does not mention — same idea as
  the declset. *(#12, F-30)*
- **R2.3** A cursor fixture for the completeness test. Broadcaster's operator surface is all
  cursors, and there is no fixture for one today. *(F-30)*
- **R2.4** `tileMemo` and `boxMemo` — byte-identical stats and the same shape around them —
  consolidated, **with the frame cost measured rather than assumed.** *(F-53)*

### R3 — The station can be tested on demand

This is the requirement that answers the locked problem statement directly.

- **R3.1** A diagnostics surface at `ctrl+d` that lets a human inject event types and exercise the
  real machinery — including audio out, which the Broadcaster operator needs before going to air.
  *(GitHub #9, F-21)*
- **R3.2** Diagnostic head and tail on audio events, on the American emergency-broadcast pattern —
  an opening that says this is a test, and a closing that says the test has ended. *(#9)*
- **R3.3** `*** TEST EVENT ***` treatment in the news ticker and in `[w]` event reports. *(#9)*
- **R3.4** Test events expire on their own in no more than 2 minutes — long enough to check every
  surface, short enough not to foul real data. *(#9)*
- **R3.5** The `ctrl+d` window's tail is reachable at 24 rows at every width, and its prose stops
  double-wrapping into fragments at 80 columns. This gates R3.1: the window becomes user-facing.
  *(F-35)*
- **R3.6** An artifact-level check that the injector is absent from every shipped binary — not a
  source-text assertion, a check against the built artifact. This gates R3.1 in the other direction.
  *(F-38)*
- **R3.7** A STOP ALL control returning Observer to no active reads or relays. Needs a key-binding
  decision. *(F-26)*
- **R3.8** `/debug/dump` checks its method. Wanted before the debug server is documented for users,
  which R3.1 effectively does. *(F-9)*

### R4 — Every gate can fail

- **R4.1** A `make lint` target wiring `golangci-lint` and `staticcheck`. Neither has ever run as a
  gate here. *(F-15)*
- **R4.2** One completeness test covering both the AA contrast register and `--ascii`. The AA gate
  shrinks silently whenever a token is added, and non-ASCII furniture survives `--ascii` in four
  windows with no scan — including Location Details, the most-opened modal. These are the same test
  written twice. *(F-18, F-37, F-47)*
- **R4.3** The PTY journey's read step establishes its own precondition. Today it depends on its tab
  happening to have a focused event, so the one honest step in that block is the one that fails.
  *(F-44)*
- **R4.4** Every gate in `07-readiness/gates.md` carries an evidence line recording a watched
  failure. *(metric G)*

### R5 — Silent user-visible defects

All fail quietly; all cheap.

- **R5.1** A looked-up location that never returns data is recognisable as such, instead of reading
  as "still loading" forever. *(GitHub #13)*
- **R5.2** The *All Reports* picker appears in Settings. It is absent from the shipped window and
  the role it sets is reachable only by editing the config file — and it is the CHANGELOG's own
  promise. *(F-46)*
- **R5.3** The three config-only cast roles are reachable, or documented as deliberately not.
  *(F-5)*
- **R5.4** The relay-fault window's 10-second auto-close is fixed. It is a hard timeout on the app's
  only station-tuning control and **fails WCAG 2.2.1 (Level A) outright.** How long, and whether a
  key resets it, is a UX ruling. *(F-36)*
- **R5.5** The relay-fault window is audible. Today `onSilence` and `escalate` send a message and
  nothing else — the reason string never reaches a voice. *(F-33)*
- **R5.6** F-43 chased: a tone sounded three times with no words on a real burst during UAT, and it
  has never been reproduced. A tone is a promise of words. The diagnostics surface from R3 is the
  most likely tool for reproducing it on demand, which is why the two are in the same release.
  *(F-43)*

### R6 — Rulings, not code

Decisions that six new feeds will copy. **The decision is in scope; the migration work is not.**

- **R6.1** Rule on `app/release.go`: either state in `extending.md` that app-lifecycle checks are not
  providers and why, or move it behind the seam. It is the precedent six new feeds will follow.
  *(F-3)*
- **R6.2** Rule on the cache root — `os.UserCacheDir()` as today, or a single `~/.watchpost/`. Any
  move strands existing caches including ~63 MB Piper models per voice, so the migration is a
  separate piece of work from the ruling. *(F-49)*
- **R6.3** The 23 P10 ledger rows still naming gates that passed releases ago are presented for
  ratification. They were deliberately kept out of the 2026-09-06 ratification. *(F-45)*
- **R6.4** The ~200 ephemeral attribution stamps in comments outside the swept surface. *(F-12)*
- **R6.5** Pre-discovery only on moving pure documentation to GitHub Pages: survey what is prose,
  what is evidence, and what is code-adjacent; report the split; rule on where each belongs. **No
  files move in this release.**

## Technical Constraints

1. **The injector must never ship.** Build-tagged out, not runtime-gated. R3 makes diagnostics
   user-facing, which promotes F-38 from "if it is cheap" to mandatory.
2. **A test event travels the production path.** Same code path a real hazard takes, unmistakably
   marked in every surface, bounded at 2 minutes. A diagnostic on its own path would pass while the
   real path is broken — which is what #7 was.
3. **Both platforms exercised in CI while the work is being done.** The single biggest finding of
   0.14.0 was that the release ran on Linux for the first time in its own release PR. Push the
   branch early.
4. **The frame path is allocation-sensitive.** Memo changes are measured, not assumed. Read
   `docs/accepted-costs.md` before optimising anything.
5. **A cache move must not silently re-download ~63 MB per Piper voice.** *(F-49)*
6. **Nothing in 0.15.0 may require Broadcaster to exist.** Every item stands alone in Observer.
7. Go pinned at 1.27.0 with `GOTOOLCHAIN=local` in CI · any table is a go-studs table
   (`data_table_row.go`, not `table.go`) · no reimplementing lipgloss or go-studs — patch narrowly,
   go upstream, or accept and record the cost · P10 exemptions are presented for ratification, never
   self-approved · no AI attribution anywhere in commits, PRs or code · no usernames or absolute
   paths that reveal PII in shipped artifacts.

## Other Considerations

- **Broadcaster UI is 0.16.0**, retitled on issue #10 at intake.
- **F-40 gets its own release, before Broadcaster — so it takes a 0.15.x number.** A real evacuation
  order and its fire were both invisible on 2026-09-06 (the Brengel Fire, Vista CA). It is the
  purest instance of the locked problem statement in the whole ledger and the most expensive to fix
  — it needs its own DISCOVER against the fire feeds and APIs, and folding it in would swallow this
  release. **Sequence: 0.15.0 (this) → 0.15.x (fire) → 0.16.0 (Broadcaster).**
- **Deferred, with reasons already recorded:** F-16 / F-32 (ticker geometry — its own DISCOVER;
  every fixture written for it so far has been invalid) · F-19 · F-23 · F-24 · F-27 · F-31 (read
  order, deferred post-Broadcaster by MVS-D-79) · F-4 · F-7 · F-8 · F-10 · F-11 · F-13 · F-20 ·
  F-48 (a HUM LEAD decision, not a bug) · F-54 (an A2DH framework change, not watchpost code).
- **Prior art is unusually strong here.** 0.14.0's debrief already named this release's dominant
  shape — *"a rule implemented and pinned in one layer, not carried by the layer that would deliver
  it," nine instances in one release* — and proposed the producer/consumer completeness check over
  closed sets that R4 is a version of. `06_docs/quality-observations.md` carries the catches.
- **From 0.15.0 onward the finalized brief is the GitHub issue description** for the feature or
  release it covers.

## Discovery Handoff Package

### Areas to Investigate

1. **The config write path (R1.1).** Enumerate all six `Load` → mutate → `Save` owners and establish
   whether a single serialized `config.Mutate(func(*Config))` covers every one, including the two
   whose defects are already fixed (the enter-save drop, the concurrent write).
2. **The three classifiers and their taxonomies (R1.2).** Build the cross-table of all three
   six-member taxonomies over the same NWS product strings and enumerate every existing divergence,
   not just the known "Coastal Flood Statement" one. That table is the requirement.
3. **The memo key inventory (R2).** Every memo key in the tree against the state its window reads.
   Red team round 3 already counted `modalKey` at 27/34 and `bodyKey` at 15/22 — start there and
   find the rest.
4. **What a test event must touch to be honest (R3, constraint 2).** Trace the production path a
   real hazard takes from arrival to spoken word, and identify where a test event can be injected
   such that everything downstream is the real thing.
5. **`ctrl+d` at 24 rows (R3.5).** Reproduce the unreachable tail and the 80-column double-wrap
   before designing the fix.
6. **Frame cost of memo consolidation (R2.4).** Blocked on the Arch box measurement — see RS-2.
7. **The docs tree split (R6.5).** What in `06_docs/02_features/` is prose, what is evidence, what
   is code-adjacent. Note that eight run records in those trees were git-ignored until this
   morning's hygiene pass, so "pure docs vs artifacts" is not a clean split today.

### Stakeholders to Consider

- **HUM LEAD (Branden)** — SEV-0, HUMAN LEAD: approves all decisions. Owns every UX ruling in this
  brief (R5.4's timeout, R3.7's key binding, R6.1 and R6.2's rulings) and the P10 ratifications in
  R6.3.
- **The listener** — the person the locked problem statement names. Not consulted directly;
  represented by UAT on real hardware and by the read of a real alert burst.
- **The future Broadcaster operator** — a second modality, not yet built. Every R1 and R2 item
  exists because of them.
- **The channel listener over FRS/GMRS/CBR** — reached only by ear, with no screen at all. The
  strictest audience for anything in R3 and R5.5.

### Risk Signals

- **RS-1 — Scope.** R1–R5 plus R6's rulings is a large release. 0.14.0 was 790 commits and it also
  started as a bounded list. The mitigation is the scope test in the locked statement (loud failures
  are not 0.15.0) and a checkpoint after R1–R3, which are the Broadcaster gate proper.
- **RS-2 — A blocking dependency the HUM LEAD owns.** `perf-protocol.md` §3–5 on the Arch box.
  R2.4's consolidation and the resident-Piper decision (MVS-D-17 / OQ-18) have no input until it is
  measured. **DISCOVER cannot honestly exit without it.**
- **RS-3 — R3 raises the stakes on injection.** The diagnostics surface makes injection user-facing
  for the first time. F-38's artifact check stops being a nice-to-have; if it slips, R3.1 should not
  ship.
- **RS-4 — R5.6 is an unbounded hunt.** F-43 has never been reproduced. It needs a timebox and a
  written disposition if it is not caught, rather than an open-ended chase.
- **RS-5 — R1 and R2 are wide refactors under a SEV-0 release.** Both touch code paths that carry
  hazard information. TDD is directive-mandated here for a reason; the safety of these is the tests
  that exist before the change, not after.
- **RS-6 — RESOLVED at intake.** Broadcaster is 0.16.0 (issue #10 retitled by the HUM LEAD), so the
  fire release takes a 0.15.x number. Kept visible because it constrains F-40's scope: a 0.15.x
  cannot carry breaking changes, which shapes what its DISCOVER may propose.

### Open Questions

- **OQ-1** — ~~What number does the fire release take?~~ **RULED at intake:** Broadcaster is 0.16.0,
  so the fire release is 0.15.x. Remaining half: does F-40's fix fit inside a point release, or does
  adding a fire feed make it a minor and push Broadcaster? Its DISCOVER answers this.
- **OQ-2** — R5.4: how long should the relay-fault window stay open, and does a keypress reset it?
  UX ruling, HUM LEAD.
- **OQ-3** — R3.7: what key binding does STOP ALL take, and does it live in the diagnostics window or
  on the masthead?
- **OQ-4** — R6.1: is `app/release.go` a provider or is it not? The answer becomes the documented
  precedent for six new feeds.
- **OQ-5** — R6.2: `os.UserCacheDir()` or `~/.watchpost/`? And if the latter, does the migration
  land in the same release as the ruling or a later one?
- **OQ-6** — R5.6: what is F-43's timebox, and what is the written disposition if it is not
  reproduced inside it?
- **OQ-7** — R2.4: does consolidating `tileMemo` and `boxMemo` cost frame time, and is that cost
  acceptable against `docs/accepted-costs.md`? Blocked on RS-2.
- **OQ-8** — R6.5: after the survey, does documentation move at all, and does GitHub Pages carry
  evidence records or only prose?

## Brief Metadata

| Field | Value |
|---|---|
| Scope adjective | Major |
| Project type | Feature (release) |
| Project name | `0.15.0-pre-Broadcaster-ui-improvements` |
| LEVEL | LEVEL-1 |
| SEV | SEV-0 |
| Phase instructions | FULL GIT; FULL REPORTS; FULL DIAGRAMS; FULL RCC; FULL PLAN; FULL TDD |
| Completeness at handoff | 5/5 required sections complete |
| Problem statement | Refined and LOCKED at 5/5 (`refine-problem-statement` Step 6) |
| Metrics | 4, each anti-solution checked (`refine-problem-statement` Step 5) |
| Backlog review | No `backlog.yml` present; GitHub issues and `06_docs/follow-ups.md` reviewed in full instead |
| Branch | `feature/0.15.0-pre-broadcaster-ui-improvements` |

## Source documents

GitHub issues #9, #10, #12, #13 · `06_docs/follow-ups.md` (47 open rows reviewed) ·
`06_docs/quality-observations.md` · `06_docs/02_features/multi-voice-support/08-reports/debrief.md` ·
`CHANGELOG.md` §0.14.0, §0.14.1.
