# Red-team round 3 — BUILD exit, on compiled code (0.13.0 severe-alerts-modals)

**Date:** 2026-08-28 · **SEV-0 · HUMAN LEAD** · **Scope:** commits `5eb88c7` `eb5a912` `00da243` `31674bf` on
`feature/severe-alerts-modals` · **Dispatch:** three sectioned reviewers — A: domain + app (correctness,
concurrency, perf, InfoSec, junior-dev, test quality) · B: TTY window + render (+ A11y persona, modal-memo
audit) · C: plan conformance + documentation truth. Every finding was **verified before acceptance**: the
reviewers' throwaway demonstrations were re-run (on a scratchpad copy of the tree, or as a temporary probe
deleted before the commit), each cited site was read, and the demonstrations that held were lifted into the
repo as the positive controls that were missing.

Disposition codes: **Fixed** · **Accepted-as-is** (with reason) · **Declined** (with reason) · **Deferred** (to a
named owner/phase) · **HUM LEAD ruling** (presented at this gate).

Totals: **57 findings** — 51 Fixed · 4 Accepted-as-is · 1 Deferred · 2 HUM LEAD rulings · 0 Declined. Plus one
finding of my own from the PTY re-run (P-1), Fixed.

## Section A — domain + app (reviewer verdict: HOLD → fix-then-ship)

| ID | Sev | Finding | Verified | Disposition |
|---|---|---|---|---|
| R3-A-01 | High | `domains/weather/nws/alerts.go` clamped the zone list to 50 **before** matching tracked locations — an alert spanning > 50 zones (the fixture's Hydrologic Outlook has 81) never reached a location whose zone was the 51st+. A regression of the shipped `[A]` path (RS-10). | demo: `0 attached` for zone 60 of 80 | **Fixed**: match on the full list, bound only the retained copy; `TestMapAlertAttachesWhenTheZoneIsBeyondTheListCap` |
| R3-A-02 | High | The deck read the publishers' `last` snapshot from inside the publish hook, which runs **before** `pb.last` is stored — the window was one publish behind the tables (0-row first publish; ≤ 20 s / ≤ 2 min lag). | demo: a 1-row publish arrived as 0 rows | **Fixed**: the hooks hand the snapshot in (`SetLocations(slot, snap)`); `lastSnapshots`/`Trigger` and the `lp.mu` re-entry removed; `TestDeckSeesTheSnapshotBeingPublished` |
| R3-A-03 | Medium | `indexKey` hashed only key + `Sent`: a same-id USGS revision or a row flipping untied → tied did not republish. | demo: 1 publish, want 2 | **Fixed**: the key folds `At`, `Until`, `Location`, `Source`, `Severity`, the tie, quake `UpdatedAt`+`Mag`, tropical advisory+wind; `TestSevereDeckRepublishesOnContentChange` |
| R3-A-04 | Medium | `addFeed` ignored the Guard set: a replaced alert resurfaced untied via the national feed — two rows for one warning. | demo: 2 rows | **Fixed**: `addFeed` honours `superseded[key]`; `TestFeedPathHonoursTheLocationGuard` |
| R3-A-05 | Low/Med | One product, three tiers by path (Tornado Watch yellow untied, orange tied). | demo | **Fixed**: `globalfeed.CuratedSeverity` is the one authority; `TestSeverityIsTheSameByEveryPath` |
| R3-A-06 | Low | `toSevereRow` formatted a zero `At` as `01/01 00:00 UTC`. | read | **Fixed**: blank |
| R3-B-01 (A) | Medium | (= A-02) hook-before-store ordering. | — | Fixed with A-02 |
| R3-B-02 (A) | Low | `publishMu` held across `p.Send` before the program runs. | read | **Accepted-as-is**: both callers wait for the loop; documented |
| R3-C-01 (A) | Low | "Publishes only on change" is really "at most once per ticker cycle". | read | **Accepted-as-is**: doc block corrected; cadence bounded and benchmarked |
| R3-D-01 | Low | `os.WriteFile` keeps a 0.12.0 `seen.json` at 0644 (NFR-13). | demo `644` | **Fixed**: `Chmod` after the write; `TestSeenStoreUpgradeTightensAnOldMode` |
| R3-D-02 | Low | Feed ids unbounded on the national path. | read | **Fixed**: `clampField` in all three parsers |
| R3-E-01..05 | — | Five misleading comments. | read | **Fixed** |
| R3-F-01..06 | — | Test gaps (zone index 1; poke test blind; no content-change control; no guard-vs-feed case; fallback branches; seen-store fresh path only). | — | F-01/02/03/04/06 **Fixed** by the lifted tests; F-05 (`Sort` tie-breaks, `RecordOf` fallback) **Deferred** to REVIEW (coverage, not behaviour) |

`TestPublisherCountsPublishesAndFoldedTriggers` failed once under load and passed 8/8 with and without the
round-3 changes — pre-existing timing flake, **Deferred** to follow-ups (wait on the pending flag).

## Section B — TTY window + render (reviewer verdict: SHIP-WITH-CONDITIONS)

| ID | Sev | Finding | Verified | Disposition |
|---|---|---|---|---|
| R3-B-01 | Medium | At ≤ 20 rows the browse body (9 fixed lines) exceeded the panel budget (8), so the panel re-wrapped the railed table: two rails, collapsed columns, chips below the fold. | probe at 120×20: body 9 > budget 8, three ▲ | **Fixed**: the chrome gives up its blanks, then the total line; `TestSevereShortTerminalKeepsTheTableIntact` at 24/20/18/17 rows; golden `severe-120x20` |
| R3-B-02 | High | The plan's Red-tier `⚠⚠` / `!!` glyph (M-5, FR-15) was not built — severity within a tab was colour-only. | plan §5.1 text; goldens | **Fixed**: `severeSevGlyph` (two cells, always); goldens re-recorded; `!!` asserted once in the ASCII fixture |
| R3-B-03 | Medium | The AA test proved white body text only; the tokens the table painted (`TableMuted` 3.09:1, `TableHeader` 3.00, `FocusCell` 4.26, `AlertWarnFG` 2.53 worst-case) fail on the tints. | measured across every theme × substrate × tint | **Fixed for the gate**: the table paints `SevereTableTokens()` (white-family; headers/focus bold white) and `TestCategoryToneContrastAA` iterates exactly those; the severity glyph's per-tier tint is gone (the count carries the tier). **The hues are the HUM LEAD colour pass** — see the rulings |
| R3-B-04 | Medium | `Plain` keeps `\n`/`\t`: a newline in a single-line field split a row into three lines. | probe | **Fixed**: `render.PlainLine` on every one-line field; the escape test now covers OSC, CSI, C1 CSI/DCS and newline, and pins the window's line count |
| R3-B-05 | Low | ▼ drawn over the thumb at the bottom of the list. | read | **Fixed**: thumb ranges over the rows above the last; `TestSevereRailThumbShowsAtTheBottom`, `TestRailifyThumbTracksTheScroll` |
| R3-B-06 | Low | `enter` inside the record reset its scroll. | read | **Fixed**; `TestSevereRecordScrollsAndEnterIsInertInside` |
| R3-B-07 | Low | The 80-col `--ascii` chip row (72 cells) re-flowed with collapsed gaps. | read | **Fixed**: a three-step label ladder (`Event Details` → `Details` → `Rows/Tabs`); golden `severe-80x44-ascii` |
| R3-B-08 | Low | Two `--ascii` questions the documents disagree on. | — | **HUM LEAD ruling** (below) |
| R3-B-09 | Low | The memo's minute bucket was unconditional (every window missed once a minute); comments said otherwise. | probe | **Fixed**: projected under Details only; the ≤ 59 s label lag documented and tested |
| R3-B-10 | Medium | `severe_table.go` 0 % covered in its own package; ladder floor unpinned; `severeTabOf` unreached; record scroll uncovered; `TestSevereDownSourceIsStated` mutated state behind the memo; memo table missed six fields; escape test OSC-only; a test name over-claims. | `go test -cover` | **Fixed**: `severe_table_test.go` (ladder incl. the floor, brackets, marks, rail), `TestBreakingEventOpensTheWindowOnItsCategory`, record-scroll test, the source test through `Update`, six memo rows, the widened escape test. The name (`…MatchTheMocksAtEveryWidth`) **Accepted-as-is** — the goldens are the pin; the test guards the invariants |
| R3-B-11 | Low | `severeRows()` copied the tab on every key/miss; `Sprintf` twice; `Stats()` twice per frame while `[S]` is open. | read | **Fixed** (indexed access, one `Sprintf`; hit 3 067 → 2 401 allocs); the second `Stats()` **Accepted-as-is** (keying needs it; `[S]` only) |
| R3-B-12 | Low | Comments (never-scrolls claim, Railify contract, minute, byte-length brackets, focus-vanish). | read | **Fixed**; the "clamp instead of row 0 on a vanished focus" suggestion **Deferred** to UAT (spec-compliant either way) |

## Section C — plan conformance + documentation truth (reviewer verdict: SHIP-WITH-CONDITIONS)

| ID | Sev | Finding | Verified | Disposition |
|---|---|---|---|---|
| R3-C-01 | High | `[A]` module body and compact line rendered `Headline`/`Severity` raw (and, found while fixing, `Event` in both titles) while three documents claimed "every field" (NFR-6/RS-17). | read; probe | **Fixed**: `Plain`/`PlainLine` at all five sites; `TestAlertModuleAndCompactLineStripEscapes` |
| R3-C-02 | Medium | = B-02. | — | Fixed |
| R3-C-03 | Medium | The open tab lacked its category tint (FR-2, SAM-D-17, round-2 Y4 marked Fixed). | read | **Fixed**: bold white on the unmixed hue; `TestSevereOpenTabWearsItsCategoryTint` |
| R3-C-04 | Medium | `gates.md` incomplete (R6, PPROF soak, COV, `GOOS=linux go vet`, golden review), wrong branch, `-cover` conflated with COV. | read | **Fixed**: rewritten with real results; the two runs that cannot happen at BUILD are stated as owed with an owner |
| R3-C-05 | Medium | The COV denominator was neither §4's list nor what `RecordOf` renders. | read | **Fixed**: data-shape §4 amended to the ratified record; `cov_test.go` iterates it — 11/11 · 13/13 · 11/11 |
| R3-C-06 | Medium | The P10 exemption ledger is gitignored; "0 live, 0 unmatched" not reproducible from a clone. | `.gitignore` | **Fixed**: `07-readiness/p10-build.json` committed; the two rows quoted verbatim in `gates.md` |
| R3-C-07 | Medium | The watchlist hint showed with a populated watchlist. | read | **Fixed**: gated on an empty watchlist; both cases tested |
| R3-C-08 | Medium | No 80×24 budget row. | — | **Fixed**: measured hit 996 / miss 3 370, pinned × 1.05 |
| R3-C-09 | Low-Med | PTY journey lacked `enter → esc → esc`; script not `+x`; tmux/`ixon` limitation undocumented. | — | **Fixed** (see also P-1) |
| R3-C-10 | Low-Med | RECENT rail not `--ascii`-swapped (round-2 CQ #19). | read | **Fixed**: `recentSection` takes `RailGlyphsFor(o.ASCII)`; the ASCII frame golden re-recorded |
| R3-C-11 | Low | CHANGELOG "declared" (storms are "reported"); "every alert field". | read | **Fixed** |
| R3-C-12 | Low | Upstream candidate unlogged. | — | **Fixed**: `LOCAL_CHANGES.md` row |
| R3-C-13 | Low | Nine wrong `SAM-D`/`FR` cites. | read | **Fixed** |
| R3-C-14 | Low | Detail title dropped the window name (M-6); `len([]rune)` + magic 10 (CQ #17). | read | **Fixed**: name + crumb; `render.Width` + `severeTitleChrome` |
| R3-C-15 | Low | Round-2 ledger rows Y4, Y5, CQ #14, #17, #19 said Fixed while the code disagreed. | read | **Fixed**: Y4/Y5/#17/#19 restored in code and annotated; #14 corrected (the flag exists in the vendored kit and is load-bearing) |
| R3-C-16 | Low | `BenchmarkOverlayOnly` measured a memo-hit View, not the compositor. | read | **Fixed**: `render.Overlay` alone |
| R3-C-17 | Low | A personal path in `gates.md`; the build report untracked at dispatch. | — | **Fixed**: placeholder; committed with this exit |
| R3-C-18 | hyp. | = A-03. | — | Fixed |
| (objectives) | — | FR-10 (`tickNeeded`) and FR-13 (`E V E N T`) texts not amended for the documented deviation / the open ruling. | — | **Fixed**: amendment notes in `objectives.md` |

## P-1 — found in this round's PTY re-run (owner: me)

| ID | Sev | Finding | Verified | Disposition |
|---|---|---|---|---|
| R3-P-1 | Medium | On a real pty a lone `esc` followed by a letter reaches the model **fused as alt+letter** — the terminal sends `ESC` then the byte and the input layer has no ESC timeout — so `esc` then `w` (or `a`, or `ctrl+s`) did nothing. The shipped 0.12.0 binary behaves the same (`esc` → `a` never opens About): pre-existing since the bubbletea v2 adoption, but FR-4's `esc esc → w` walks straight into it. | probed step by step on both binaries; the model handles esc-then-w correctly | **Fixed** in the model: `splitEscFusion` reads an alt+key (no binding uses alt — asserted) as `esc` then the key; `TestLoneEscThenKeyIsNotLost`; `make pty-severe` drives the full `enter/esc/esc/w` journey green |

## HUM LEAD rulings requested at this gate

1. **Colour pass (B-03).** For the gate the table paints white-family tokens on the tints (AA everywhere) and the
   severity glyph carries its tier by count, not colour. Your call at UAT: the four tint values (`EventCat*BG`,
   blend 0.6), whether the `⚠`/`⚠⚠` glyph gets a per-tier tint back (it would need one that clears 4.5:1 on every
   tint — the current alert red does not), and the header/focus tones.
2. **`--ascii` headers (B-08a).** Objectives FR-13 asks for plain `EVENT` under `--ascii` (screen readers read
   `E V E N T` as five letters); the ratified mock §5.4 keeps the spread. Code follows the mock. Which wins?
3. **`--ascii` marks (B-08b).** Under `--ascii` the pointer and the play mark are both `>` (`>  > !!`) — a
   pre-existing glyph-set collision (`TestGlyphsSwapAsOneSetUnderASCII` pins Play = `>`). Keep, or give the mark
   its own ASCII form (mock.py used `*`)?
4. **P10 ledger rows** (`domains/globalfeed`, `domains/severe` — P10-05-INVARIANT-DENSITY): ratify.

## Round verdict

**SHIP-WITH-CONDITIONS → conditions met.** Two Highs in the data path (a shipped-feature regression and a
one-publish lag), one High in the UI (the ratified severity glyph) and one High in the docs (an over-claimed
security boundary) were all real, all reproduced, and all closed with the reproductions kept as tests. After
the round: `make verify` ALL GATES GREEN · `make p10` 0 live, 0 unmatched · `make pty-severe` ok on the
extended journey · budgets within pins · seven goldens re-recorded deliberately (glyph, tokens, chips, rail).
What remains is the HUM LEAD colour pass and the two `--ascii` rulings, none of which block REVIEW.
