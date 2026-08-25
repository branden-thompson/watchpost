# Red-Team Report — PLAN Exit — watchpost-cli

`red-team: SHIP-WITH-CONDITIONS · multi-agent (4 sectioned dispatches) · scope:feature(PLAN corpus) · personas:[infosec, a11y, perf, junior-dev]`

| Field | Value |
|---|---|
| Phase / SEV | PLAN exit · SEV-0 (HUMAN LEAD) |
| Date | 2026-08-23 |
| Mode | Sectioned dispatch, independent verdicts; code axis solo; mutation freeze held |
| Lens set | 4 axes + PLAN phase lens (Distinguished Engineer) + 4 HUM-LEAD-confirmed personas (fresh scope-scan added junior-dev) |
| Scope | 03-architecture-design/** + cross-corpus consistency; all dispatches attributed clean tree @ `bcba2d1` |
| Watermark sweep | **Clean** (10 commit messages, 38 tracked files) |

## Section verdicts

| Lens | Verdict | Driving finding |
|---|---|---|
| Code Quality (solo) | SHIP-WITH-CONDITIONS | Diffs-field & Hotkey() contradictions vs PD-2/D-15; Fragment/merge core undefined |
| Project Hygiene | CONDITIONAL PASS | plan-report in-flight (H-1); spike evidence in ephemeral scratchpad (H-2) |
| Docs Quality | REVISE | "two-tier" label; M6 supersession; P-x identifier collision |
| Business Quality | SHIP-WITH-CONDITIONS | R-6 mock dependency without fallback; traceability orphans T-D/T-I/T-J |
| PLAN phase lens | SHIP-WITH-CONDITIONS | FULL PLAN bar unmet on payload types + exit/error taxonomy; schema v1 one-way door |
| InfoSec persona | CONDITIONAL | `--tts-cmd`/product-text exec safety untreated; IS-1 landed as log-redaction only |
| A11y persona | CONDITIONAL | R-12d screen-reader surface was promise-not-mechanism; R-12a/b/c verified landed |
| Perf persona | **PASS** | All Accepted→PLAN items verified landed; spikes close budgets; one measure-forward (25-loc bench) |

## Disposition ledger (consolidated; all verified against artifacts before fixing)

**Fixed in working tree (lands in this exit commit):**

| Finding | Fix |
|---|---|
| Code#1 + Docs D-5 (Diffs field vs PD-2) | `Diffs` deleted from `Location`; diff view derives from `ByProvider` (§2 note) |
| Code#2 (`Hotkey() rune` vs D-15) | Method deleted; View exposes `Actions()`+`DefaultKeyMap()` only |
| Code#5/#6 (silence detection; oto invariant; alert-resume; synth race) | §10.4 radio mechanics |
| Code#7 + F-1 + J·a/J·b (undefined types, merge rule, TTL dual-authority) | §10.1 type appendix; `Provider.TTL` deleted — scheduler tiers are the single freshness authority |
| Code#8 (`platform/audio` unowned) | Added to B4 Delivers |
| Code#9 (report↔render ambiguity) | Report width from `platform/term`; report never imports render (§4) |
| Code#10 + Biz B-2 (mock-slip fallback) | Placeholder-layout fallback + B5-data pull-forward buffer named; T-J → SHIP |
| F-2 (exit codes / Warning enum) | §10.2 |
| F-5 + IS·d (transmitter table hidden-80%; provenance) | §10.6 pipeline + checksums |
| F-6 (B1 oversized) | Split B1a/B1b |
| F-7 (schema one-way door) | §10.3: v1.0-rc at B1a; v1.0 ratified at B5 exit (HUM LEAD gate item) |
| F-8 (missing risks) | RS-19/20/21 added in §10.9 with P×I ratings; pxi register carries a pointer (single source, no duplication — R2#11 wording corrected) |
| F-9 + P·a/P·b (Clock/fixture recorder; 25-loc bench; M9 idle def; soft exit evidence) | §10.8; B1b bench; B3/B4 evidence restated pass/fail |
| IS·a (secret-in-output constraint + golden) | §10.5 last para |
| IS·b/c/e (`--tts-cmd`, product-text, PowerShell quoting) | §10.5 argv-only exec spec + hostile-fixture tests |
| A·a (R-12d mechanism) | §10.7 `report --every` line-oriented SR surface |
| A·b + Code#4 (swap announcement; swap-time conflict validation) | §10.7 |
| A·c (WAT startle) | 250 ms amplitude ramp (§10.4) |
| Deletion list (Marquee/Meter/Bars/Grid speculative) | §10.10.5 on-demand rule; B3 cell reworded |
| Hygiene H-2 (ephemeral spike evidence) | Code+logs+data copied to `04-development/spikes/` |
| Hygiene H-3 (.gitkeep) | Removed |
| Docs D-1 ("two-tier") | Relabeled 5-tier in architecture.md; round 2 caught the structure-proposal.md:90 remnant — also fixed (R2#2) |
| Docs D-2 (brief M6 stale) | Supersession note; brief → v1.3.0 |
| Docs D-3 (pxi DRAFT status) | Updated + measured risks closed |
| Docs D-4 (P-x collision) | Architecture decisions renamed PD-1..PD-4 |
| Docs D-6 (AI-9 breakpoints) | Supersession annotation |
| Biz B-1 (orphans) | T-D→B3, T-I→B5, T-J→SHIP |
| Biz B-4 (stale badges late) | Pulled into B3 |
| J·d (key→provider wiring) | §10.1 `FetchReq.Hint` + extending.md example spec (§10.10.1) |
| F-3 (named algorithms unspecified) | §10.1 merge rule + §10.2 taxonomy landed in round 1; **round 2 caught the ledger omission and the two unclosed halves** — config format (TOML) + harmonize fill_from algorithm now specified in §10.11 |
| Hygiene H-1 (plan-report absent) | plan-report.md authored this exit sequence |
| R2#1 (§2 TTL remnant) | Deleted from §2 (the ledger's own "Fixed" claim was premature — corrected) |
| R2#5 (stale scratchpad paths) | S2 artifacts line + s1-measure.sh DIR annotated to 04-development/spikes/ |
| R2#6/#7/#8/#9 (obs_stale status rule; remaining types; null parity; template tokenization) | §10.11 |

**Transferred (named owners):** J·c extending.md → B0 deliverable (BUILD-entry precondition §10.10); Code#3 bubbles/help adapter → 5-line spike at B3 entry; F-4 alternatives-considered notes → plan-report §Decisions; Biz B-3 M6 post-SHIP measurability → plan-report; F-10 mock contingency → recorded (see Fixed row). **Declined:** none.

## Cross-cutting synthesis

1. **Convergence (4 lenses):** "specified in research, never promoted to the approved spec" — `Conditions`/`Fragment` (code, phase, junior-dev), `--tts-cmd` (infosec), R-12d (a11y). Round 1's DISCOVER meta-finding recurred at PLAN; §10 is the structural fix, and the plan-report carries a standing rule: *a research recommendation is not real until it has an architecture line.*
2. **Convergence:** both empirical traces + phase lens agree Option-C answers *where* but B0 must deliver *how* (`extending.md`).
3. **Contradictions found were self-inflicted supersession misses** (Diffs vs PD-2, Hotkey vs D-15) — the same class as DISCOVER's D-10 propagation misses; propagation is now checked by grep before every gate commit.
4. **Risk deltas:** +RS-19/20/21; RS-8, RS-18 closed by measurement/verification.
5. **Round-2 decision (Step 9):** find-rate material but ~90% documentation-completion, 0 Critical, architecture unchanged in substance; remediation added no new mechanisms beyond paragraph specs. **One targeted round-2**: fresh single sectioned dispatch verifying the §10 addendum + fix propagation before the exit gate (cheaper than DISCOVER's full round 2, matching the smaller blast radius). Result appended below.

## Overall verdict

**SHIP-WITH-CONDITIONS** — conditions were the §10 specifications and hygiene fixes, all now landed in the working tree; residual conditions transfer to BUILD entry (§10.10) and the B5 schema-ratification gate. The human decides at the PLAN exit gate.


---

## Round 2 Addendum (fresh lens, 2026-08-23)

**Verdict: SHIP-WITH-CONDITIONS → converging.** 5 of 7 round-1 fix groups verified HOLDING with line evidence; spike numbers independently recomputed from retained artifacts (1.83% CPU mean reproduces exactly). Round 2 found 3 Majors — §2 TTL remnant (fixed), F-3 ledger omission + two unclosed halves (fixed, §10.11), and **`report --every` vs OQ-18** (scope expansion past a HUM LEAD ruling — escalated to the exit gate as G-9, not self-ratified) — plus 6 Minors and 2 polish items, all fixed in §10.11/doc edits. Remaining open item at gate: G-9 ruling. Convergence: achieved contingent on G-9 (no further round planned; HUM LEAD spot-check closes).
