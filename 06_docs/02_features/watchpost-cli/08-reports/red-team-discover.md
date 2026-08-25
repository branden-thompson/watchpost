# Red-Team Report — DISCOVER Exit — watchpost-cli

`red-team: SHIP-WITH-CONDITIONS · multi-agent (4 sectioned dispatches) · scope:feature · personas:[infosec, a11y, perf]`

| Field | Value |
|---|---|
| Phase / SEV | DISCOVER exit · SEV-0 (HUMAN LEAD) |
| Date | 2026-08-23 |
| Mode | Multi-agent, sectioned dispatch with independent verdicts (code axis solo) |
| Lens set | 4 axes + DISCOVER phase lens (Distinguished PM) + 3 HUM-LEAD-confirmed personas |
| Scope | `06_docs/02_features/watchpost-cli/**` (brief, 11 research docs, 2 syntheses, objectives) + `.a2dh.yml`, `.gitignore`, seam files |
| Worktree attribution | All 4 dispatches recorded `git status --porcelain` = clean at `8fd311c`; mutation freeze held for the round |
| Watermark sweep | **Clean** — zero AI-attribution hits in `git log --format=%B` and committed files (calibration self-matches excluded) |

## Section verdicts

| Lens | Verdict | Driving finding |
|---|---|---|
| Code Quality (solo) | SHIP-WITH-CONDITIONS | 4 Important: bubbletea version ambiguity, Snapshot concurrency contract, alias-prone parity verifier, missing M2/M3 test design |
| Project Hygiene | CONDITIONAL | H-1: Discovery Report not yet committed (this exit sequence produces it) |
| Docs Quality | REVISE | D-1/D-2: D-10 fire ruling not propagated to objectives.md and brief tables |
| Business Quality | CHALLENGE | B-1 alternatives never surveyed; B-3 radio descope option owed to HUM LEAD |
| DISCOVER phase lens | SHIP-WITH-CONDITIONS | F-1: "while running" alert limitation was agent-assumed, not human-ruled |
| InfoSec persona | CONDITIONAL | IS-1..IS-3: named risks dropped at synthesis (redaction, hostile stream, doc scrub) |
| A11y persona | **FAIL** | A-1/A-2: zero a11y requirements; no non-color severity channel for a life-safety alert tool |
| Perf persona | CONDITIONAL | P-1: M8 budget unproven at worst case (15–25 MB baseline + 7–10 MB geodata vs 40 MB) |

## Consolidated findings & disposition ledger

Severity: C=Critical, I=Important/High, M=Minor/Medium, L=Low. Disposition: **Fixed** (commit), **Accepted→PLAN** (PLAN-entry condition), **Accepted→BUILD** (BUILD rule), **Accepted→SHIP** (SHIP-gate item), **Gate** (HUM LEAD ruling requested at this exit), **Declined** (rationale).

| # | Lens | Finding | Sev | Disposition |
|---|---|---|---|---|
| 1 | Code | bubbletea "current" ambiguous (v1.3.x facts vs v2 existence) | I | **Ruled by HUM LEAD (G-5, D-12)**: latest bubbletea, overriding red-team recommendation; consequence accepted → PLAN condition: re-verify v1-derived frame/renderer claims on latest |
| 2 | Code | `Snapshot` ownership/concurrency unstated ("by value" shares slice backings) | I | **Accepted→PLAN**: contract rule — providers publish immutable snapshots via `tea.Msg`; `Update` swaps wholesale; no writer retains a reference |
| 3 | Code | Parity golden can false-pass on aliasing values | I | **Accepted→PLAN**: parity fixture uses pairwise-distinct sentinel values |
| 4 | Code | No test design for primary metrics M2/M3 (replay harness, loop tests, `-race`) | I | **Accepted→PLAN**: PLAN must specify alert replay harness, teatest-style loop tests, `-race` in verify |
| 5 | Code | `radio.status:"playing"` leaks view-state into data contract | M | **Accepted→PLAN**: schema field becomes availability (`available/none`) |
| 6 | Code | runewidth version mismatch (matrix on v0.0.28; go-studs pins v0.0.19) | M | **Accepted→PLAN**: align versions; re-run matrix on pinned version |
| 7 | Code | tier2 CPU figure mis-transcribed (2% vs source's ≤1% @10fps) | M | **Fixed** — tier2-synthesis.md corrected (working tree; lands in exit commit) |
| 8 | Code | `additionalProperties:false` envelope-only | M | **Accepted→PLAN**: apply to all object schemas |
| 9 | Code | Exit 2 on any warning → chronic `obs_stale` noise | M | **Accepted→PLAN**: exit 2 = provider failure only; warnings in-band |
| 10 | Code | "Stateless render layer" overstated; NextFrame() must be Update-driven | M | **Accepted→BUILD**: frame stepping in `Update` on tick only; `View()` pure |
| 11 | Code | Simplify: `diffs[]` derivable; oggvorbis 1/117; dual golden systems; premature compat/deprecation machinery; height breakpoints | M | **Accepted→PLAN** simplification list (PLAN decides each; default = cut from v0.1) |
| 12 | Hygiene | H-1 Discovery Report missing | I | **Fixed by this exit sequence** — discover-report.md is this phase's closing artifact |
| 13 | Hygiene | H-2 `.a2dh.yml` empty description; no `languages: [go]` | M | **Gate** → G-6 (config edits need HUM LEAD approval) |
| 14 | Hygiene | H-3 `08-reports/` not in 7-folder spec text | M | **Declined-with-rationale**: 08-reports/ is the framework's FULL REPORTS convention (template source `05_templates/reports/`, precedent in A2DH repo features); spec text lists core folders, reports folder is additive. Recorded here as the ruling of record. |
| 15 | Hygiene | H-4 `.gitignore` lacks Go artifacts | M | **Accepted→BUILD** entry task |
| 16 | Docs | D-1 objectives.md stale vs D-10 (FIRMS vs HMS; missing radio/private caveats) | I | **Fixed** — objectives.md regenerated (working tree; lands in exit commit) |
| 17 | Docs | D-2 brief OQ-11/R-9a stale vs D-10 | I | **Fixed** — supersession notes added (working tree; lands in exit commit) |
| 18 | Docs | D-3 RS-1 severity stated three ways | M | **Fixed** — brief RS-1 row now carries final disposition |
| 19 | Docs | D-4 OQ-19 ruling lacked question text | M | **Fixed** — tier2 §C.10 now tagged OQ-19 |
| 20 | Docs | D-5 AI-9 cross-terminal cells documented-not-validated | M | **Accepted→VALIDATE**: mark risky rows; validate on real terminals in VALIDATE phase |
| 21 | Business | B-1 alternatives never surveyed ("no single glanceable surface" unratified) | I | **Fixed (in progress)** — AI-12 alternatives survey dispatched; result lands in Discovery Report §Alternatives |
| 22 | Business | B-2 M6 unmeasurable if D-9 pivot lands | M | **Gate** → G-4a: proposed redefinition — M6 = "components accepted into whichever shared component home D-9 resolves to" |
| 23 | Business | B-3 radio v0.1 must carries un-dispositioned ToS exposure; descope never offered | I | **Ruled by HUM LEAD (G-2, D-12)**: descope rejected; radio critical; synthesized-voice avenue added (AI-13) — RS-12 partially mitigated by de-soleing the volunteer streams |
| 24 | Business | B-4 non-US users get data, no alerts — statement asymmetry | M | **Accepted→Report**: Discovery Report states v0.1 solves the problem for US users; alerts-abroad an acknowledged non-goal |
| 25 | Phase | F-1 "while running" limitation agent-assumed, never ruled | I | **Gate** → G-1 (the round's top item) |
| 26 | Phase | F-2 no end-user personas; no safety-disclaimer requirement | I | **Gate** → G-3b adds R-13 (disclaimer + stale-honesty); personas authored at PLAN entry |
| 27 | Phase | F-3 problem statement asserted, not evidenced (n=1 missing) | M | **Gate** → G-7: one paragraph of HUM LEAD's own workflow evidence |
| 28 | Phase | F-4 M6/T-D serve library roadmap; label as owner preference | M | **Accepted→Report**: recorded as HUM LEAD preference, legitimate for a personal project; D-9 contains blast radius |
| 29 | Phase | F-5 radio-required is preference elevated to must | M | Merged into #23 (G-2) |
| 30 | Phase | F-6 no inaction/alternatives analysis | M | Merged into #21 (AI-12) |
| 31 | Phase | F-7 risk register lacks probability × impact | M | **Accepted→PLAN entry**: P×I pass on full register; RS-15 re-rated; RS-11 moved to process checklist |
| 32 | Phase | F-8 AI-11 Q2/Q3 (employer IP, OSPO) deferred not answered | M | **Accepted→SHIP** gate (with OQ-17 private-repo holding state); NWS-roadmap check added to SHIP gate |
| 33 | InfoSec | IS-1 no secret-redaction rule in machine-mode output | M | **Accepted→PLAN**: schema-level constraint "no secret in any machine output" + golden test |
| 34 | InfoSec | IS-2 hostile-stream/supply-chain risk dropped at synthesis | M | **Fixed by registration**: **RS-16** "attacker-controlled audio input" (Medium) — ICY+MP3 fuzz is a gated BUILD requirement; go-mp3 vendored |
| 35 | InfoSec | IS-3 private hostnames in committed docs | M | **Accepted→SHIP** gate: "scrub/redact 06_docs before repo goes public" added as SHIP-gate item |
| 36 | InfoSec | IS-4 non-TLS stream path unvalidated | L | **Accepted→BUILD**: https-preferred; warning on http mounts/overrides |
| 37 | InfoSec | IS-5 0600 file vs dotfile-sync leak vector | L | **Accepted→BUILD**: separate keys file + README warning |
| 38 | InfoSec | IS-6 no integrity commitment for embedded datasets | L | **Accepted→BUILD**: checksums in CI |
| 39 | A11y | A-1 a11y absent from requirements entirely | I | **Gate** → G-3a: add requirement family R-12 (a11y) |
| 40 | A11y | A-2 no non-color severity channel | I | **Accepted→BUILD rule (within R-12)**: severity always rendered as text label + position, color additive |
| 41 | A11y | A-3 no reduced-motion option | M | **Accepted→R-12**: `--no-animation` flag alongside `--ascii` |
| 42 | A11y | A-4 no contrast floor; dark-default on light terminals | M | **Accepted→PLAN**: contrast minima per token pair; light-background test |
| 43 | A11y | A-5 screen-reader path unpositioned | M | **Accepted→R-12**: `--report-only` documented as the accessible surface |
| 44 | Perf | P-1 M8 unproven at worst case; S2 optional | I | **Accepted→PLAN entry gate**: spike S2 (geodata memory) gates PLAN exit; M8 re-derived from measured numbers (G-4b covers re-ratification authorization) |
| 45 | Perf | P-2 "idle" definition excludes polling cost | M | **Accepted→PLAN**: M9 idle window defined to include polling schedule; ingest costed in spike S1 |
| 46 | Perf | P-3 fire-history store unbounded | L | **Accepted→PLAN**: bounded, date-keyed eviction |
| 47 | Perf | P-4 geocode cache + alert log never rotated | L | **Accepted→PLAN**: size caps/rotation in cache subsystem |

## Cross-cutting synthesis (round 1)

1. **Convergence — "named then dropped."** InfoSec, Perf, and the phase lens independently found the same failure shape: risks named inside research docs (fuzzing, M8 arithmetic, color-stripping) never promoted to the register/gates where SEV-0 discipline lives. Remediation: RS-16 registered; S2 gated; R-12 proposed; this report is now the authoritative register carry-forward.
2. **Convergence — radio.** Code (#11 oggvorbis), Business (B-3), Phase (F-5), InfoSec (IS-4), and Tier-1 #10 all press on the same seam: radio is the highest-preference, highest-ToS-exposure v0.1 item. All routes converge on gate item G-2.
3. **Composition — A-2 × NO_COLOR.** The docs already commit to honoring `NO_COLOR`; composing that with color-coded severity means severity *already must* survive color-stripping — A-2 is not new scope, it is an unstated consequence of an existing commitment.
4. **Contradiction resolved — none open.** All lens contradictions route to gate items G-1..G-7; no lens contradicts another's facts.
5. **Risk register deltas:** +RS-16 (audio input, Medium); RS-15 re-rate queued for the P×I pass; RS-11 demoted to checklist.
6. **Round-2 decision (Step 9):** find-rate is material (2 High-class + ~8 Important) → a **round 2 with fresh lenses runs after remediations land** (gate rulings receive HUM LEAD review directly at the exit gate), targeted at: the remediated docs, the Discovery Report, and the new gate decisions. Convergence expected (round 1 verified-clean lists were long); round 2 is the check.

## Overall verdict

**SHIP-WITH-CONDITIONS** — DISCOVER's research base is strong (every lens recorded a substantial verified-clean list); the conditions are the seven gate items (G-1..G-7) and the Accepted→PLAN/BUILD/SHIP rows above, which transfer as entry conditions to their phases. A11y's sectional FAIL is resolved by G-3 (R-12) at this gate rather than blocking, per red-team's advisory role — the human decides.


---

## Round 2 Addendum (fresh lenses, 2026-08-23)

**Verdict: SHIP-WITH-CONDITIONS — near-converged.** No substantive analysis defects; all five round-1 fixes verified holding item-by-item; 47-row ledger confirmed complete; `a2dh validate` re-run live 18/18; AI-12 honesty confirmed carried into the Discovery Report. Findings (all record-integrity): R2-1 mislabeled disposition (row 22 → now Gate G-4a) — fixed; R2-2 bundled gate items (G-3, G-4 split into a/b) — fixed; R2-3 unversioned edit to approved brief (→ v1.2.0 + D-11) — fixed; R2-4 5th-cell table overflow (RS-1 note folded into cell 4) — fixed; R2-5 "this commit" wording + filename typo — fixed; R2-6 AI-12 counter-argument added to Discovery Report §2 — fixed; R2-7 Closed-row hygiene (RS-6 reclassified, RS-7/RS-11 split out) — fixed; R2-8 critical_analysis PASS marked conditional on gate rulings — fixed; R2-9 `a2dh` framework-upgrade FLAG MEDIUM recorded in Discovery Report §10 — fixed. Convergence: remaining risk is paperwork-level; round 3 is HUM LEAD spot-check at the exit gate.
