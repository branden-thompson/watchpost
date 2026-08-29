# BUILD report — severe-alerts-modals (0.13.0)

**Phase:** BUILD → REVIEW gate · **SEV-0 · HUMAN LEAD** · **Date:** 2026-08-28 · **Branch:** `feature/severe-alerts-modals`
**Commits:** `5eb88c7` (P1 domain) · `eb5a912` (P2 app) · `00da243` (P3 window) · `31674bf` (P4 verify) · the
round-3 remediation commit (see §5)

## 1. What was built

The Severe Weather / Disaster Events window: `w` (or `ctrl+s`) opens a modal over the dashboard listing every
active event in six categories — Warnings · Watches · Advisories · Spec. Statements · Sig. Quakes · Tropical —
browsed with `←` `→` and `↑` `↓`; `enter` replaces the table with the event's full record in the Alert Details
shape; `esc` backs out; `esc` `esc` closes. The index is the **union** of the ticker's global feeds (USGS, NHC,
NWS) and the tracked locations' alerts, one row per normalised event id, the location record winning; the
category paints the window in its fixed tint mixed onto the modal substrate. The severe-alert narration now
ends "Press W in Watchpost for the full report on this event".

Architecture as ratified (plan §8): AX-1 A — a pure `domains/severe` index joined in `app/severe.go`
(`severeDeck`); AX-2 per-class detail structs; AX-3 `severeDetail bool`; AX-4 a modal memo for the whole
window family; AX-5 go-studs DataTable behind `platform/render.SevereTable` + `Railify`; AX-6 the parse memo
keyed on the httpx cache hit with a sha256 fallback.

| Layer | New / changed | Tests |
|---|---|---|
| `domains/globalfeed` | detail structs, parse memo, guarded supersede, storm names, bounded fields | detail/cov/memo/supersede/fetch tests (COV 100 % of the record fields) |
| `domains/severe` | `NormalizeID`, `Classify`, `Union`, `Sort`, `Cap`, `ByTab`, `RecordOf` | full table tests, invariants |
| `app` | `severeDeck` (serialised publish-on-change), publisher hooks, narration, seen-store hardening | deck/cycle/narration/escape tests, `BenchmarkSevereDeckTrigger` |
| `modes/tty` | the window (state, nav, browse, record), the modal memo, [S] gauge, help row, esc-fusion guard | 30 window tests, memo invalidation table (30 rows), exclusivity, 6 goldens, budgets |
| `platform/render` | `EventCat*` tokens, `CategoryTone`, `SevereTable`, `Railify` | independence guard with planted control, AA contrast, declset |
| verification | `make pty-severe`, escape fixture end to end, README/CHANGELOG, gates ledger | — |

## 2. Gate results (07-readiness/gates.md)

`make verify` ALL GATES GREEN · `make p10` 0 live, 0 unmatched · `make pty-severe` ok · budgets within pins ·
goldens recorded with width invariants · schema regenerated at P1.

## 3. Measurements

| Path | allocs / View() | pin |
|---|---|---|
| severe window, memo hit (after round 3) | 2 401 | 2 521 |
| severe window, memo miss | 7 561 | 7 939 |
| 80×24 dashboard hit / miss (new pin, NFR-2 floor) | 996 / 3 370 | 1 046 / 3 539 *(round-3 figures — superseded; `modes/tty/bench_test.go` owns the pins: 962 / 3 598 at REVIEW)* |
| dashboard 133×44 hit / miss (unchanged) | 568 / 7 141 | 6 000 / 10 546 |
| deck trigger, unchanged index (580 rows) | 2 020 allocs · 0.58 ms | — |

## 4. Deviations from the plan (all documented in the batch logs)

P2: UI types and category tokens built before the app join (dependency order); the poke test asserts the
poke, not the publisher's count (a never-run `tea.Program` blocks `Send`). P3: no tick for the severe window
(nothing time-relative in it); the rail keeps the header pinned; the table fills the panel budget; the empty
state keeps `[enter]` muted; the clock test swaps the snapshot pointer; `toggleSevere` split for P10-04.
P4: none.

## 5. Red-team round 3 (BUILD exit, compiled code)

Three sectioned reviewers (domain+app · TTY+render · plan-conformance+docs), full lens set and the four
personas; 57 findings + 1 of my own from the PTY re-run. **51 fixed, 4 accepted with reasons, 1 deferred
(coverage of tie-break branches), 2 HUM LEAD rulings, 0 declined.** The four Highs were real and reproduced:
a zone-list truncation that dropped > 50-zone alerts from tracked locations (a 0.12.0 regression), the window
running one publish behind the tables, the ratified `⚠⚠`/`!!` severity glyph missing, and `[A]` module lines
rendering provider text raw while the docs claimed otherwise. Every reproduction was lifted into the repo as a
test. The PTY re-run surfaced a pre-existing input-layer behaviour (a lone `esc` fuses with the next key as
alt+key on a real terminal; 0.12.0 too) that FR-4's `esc esc → w` walks into — fixed in the model with a
test and the extended `make pty-severe` journey. Ledger: `08-reports/red-team-build.md`.

**Measurements after the round:** severe window hit 2 401 / miss 7 561 allocs (was 3 067 / 8 061 — the row
copies went); 80×24 hit 996 / miss 3 370 (new pin); dashboard 133×44 hit 568 / miss 7 141 (unchanged within
noise); COV 11/11 · 13/13 · 11/11 on the reconciled render list.

## 6. Open items carried to REVIEW / UAT

- HUM LEAD colour pass on the four tints (`EventCat*BG`, blend 0.6), the table's tokens (white-family for the
  gate — the muted greys and the alert red fail AA on the tints) and whether the severity glyph gets a per-tier
  tint back; plus the two `--ascii` rulings (spread headers; the `>` pointer/mark collision) — see the ledger.
- Ratify the two P10-05 package rows (`domains/globalfeed`, `domains/severe`).
- Linux halves: `make pty-severe` and the race suite on Linux CI with the release PR.
- Compositor lever: the memo-hit cost of any open window is the lipgloss overlay (~3 000 allocs); a cached
  composite while both halves hit is the next perf lever (follow-up, not 0.13.0).

## Addendum — UAT and red-team round 4 (2026-08-28)

UAT ran 25 commits on the branch (the facelifts, the narrator, scripts and pronunciation rules as files, DETECTION,
the tints — each recorded as an amendment in `01-objectives/objectives.md`). A fourth red-team round on the UAT diff
found 47 items, 37 fixed (`08-reports/red-team-build-round4.md`); the gate ledger (`07-readiness/gates.md`) was
re-run and rewritten at this exit. R6 is a blocking VALIDATE item.

**Rulings (HUM LEAD, 2026-08-28):** B-05 — the alert tones are lifted to AA on their tints at theme registration,
hue kept (`b12cb3b`); C-07 — `.a2dh.yml` and the ledger stay local; C-06 — the 14 rows were presented for
ratification (open, see `07-readiness/gates.md`). Two UAT items after the round: the Lookup modal opens on the
looked-up location from its first frame (`b12cb3b`), and the table's column titles are a painted row of touching
segments in a dip of their group's tint (`0123464`). The Vista tail is considered resolved by A-01.

## BUILD exit — APPROVED

**HUM LEAD approval 2026-08-28** at `0123464` ("this looks much better … I am ready to approve BUILD EXIT"; the
visual fix taken without a further red-team pass by the HUM LEAD's decision). Gates at approval: `make verify`
ALL GATES GREEN · `make p10` 0 live, 0 unmatched · `make pty-severe` ok · budgets within pins. Next phase: REVIEW
(FINAL) — the owed items ride with it: the R6 relay + audio smoke (blocking at VALIDATE), the P10 ledger rows'
ratification, A-08 / A-16c/e follow-ups, the Linux halves.

**REVIEW note (2026-08-29):** A-08 (a `[space]` read cannot be cancelled), deferred above, closed at `e3a1d58`
(`newEventReader(ctx…)`, `eventReader.End`, waited for by `stopAll`); A-16c/e closed in the same commit. One
UAT commit landed after the recorded approval (`8183f5f`, WX STN / ZIP cells centred) — recorded in the
objectives and the REVIEW report.

