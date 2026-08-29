# Implementation plan — severe-alerts-modals (0.13.0)

```
Goal:         Press w anywhere → a six-tab browsable window of every active severe event with a full
              record one keypress away; storms named on the ticker; narration re-pointed at the window.
Architecture: domains/severe (pure index: normalise · classify · guard · union · sort · cap · record)
              fed by widened globalfeed parsers + bounded location alerts; app/severe.go severeDeck joins
              the ticker's pre-radius events with the publishers' last snapshots and sends SevereMsg on
              change; modes/tty renders UI rows only (browse table on go-studs DataTable + rail, in-place
              detail, memoised modalView); fixed-hue category tokens on the active modal substrate.
Tech Stack:   Go 1.27 · bubbletea v2 · lipgloss v2 · go-studs (vendored) · expect (PTY journeys)
Branch:       feature/severe-alerts-modals  (0.13.0; SemVer minor — SAM-D-12)
Rulings:      SAM-D-1..26 (brief, objectives §9, plan.md §1/§8)
```

The plan is split into four batches (one subsystem each, dependency-ordered); every task is 2–5 minutes,
names its exact file, carries the RED test first, complete code, and a verify command. Nothing in these
files is a placeholder; where a value is HUM LEAD's to tune (tints), the task says so and ships a value.
Two ordering notes carry across batches: P1 Task 1.7 (the parse memo) executes before 1.3–1.5; P3 Tasks
3.5–3.7 share one compile gate (the fixture 3.5 defines is what 3.7 tests).

| Batch | File | Scope | Gate |
|---|---|---|---|
| P1 | `p1-domain.md` | `domains/globalfeed` detail structs, widened parsers, name-aware `Sentence()`, parse memo, bounded `mapAlert` + `SenderName`, **`domains/severe`** package, fixtures already in `domains/globalfeed/testdata` | `go test ./domains/... ./platform/snapshot/...` green; COV table tests 100 % |
| P2 | `p2-app.md` | `Locate` before radius, `severeDeck` join + `SevereMsg` mapping (zone rule), triggers, narration strings, seen-store hardening | `go test ./app/...`; `-run Narration`; fetch-count assertion |
| P3 | `p3-tty.md` | tokens + `CategoryTone` + guards, UI types + tab registry, state/keys/help, nav, browse + detail renderers, `--ascii`, empty state, **modal memo (family)**, budgets, goldens | `go test ./modes/tty/...`; `make alloc-budget` with the new rows; goldens at 80/100/120 |
| P4 | `p4-verify.md` | `render.Plain` at every field on the reused `[A]` path (NFR-6), escape-injection fixtures, PTY `.expect` journeys ×2, memo hit-rate positive control, README/CHANGELOG, Linux gate notes | full `make verify`; PTY journeys pass on macOS; Linux half recorded as owed |

## File map (whole feature)

```
CREATE: domains/globalfeed/detail.go              — QuakeDetail / TropicalDetail / SevereDetail + bounds (clampProse, clampInt, clampSlice)
MODIFY: domains/globalfeed/event.go               — Event.Name + per-class detail pointers; name-aware Sentence(); Title()
MODIFY: domains/globalfeed/usgs.go                — decode the render list + Keep fields; detail populated
MODIFY: domains/globalfeed/nhc.go                 — Name, intensity, pressure, movement, advisory numbers; detail populated
MODIFY: domains/globalfeed/nws.go                 — headline/description/instruction/CAP fields/sender/parameters allowlist; guarded superseded (Supersedes)
CREATE: domains/globalfeed/memo.go                — sourceMemo: skip decode when the body is unchanged (cache hit / equal hash)
EXISTS: domains/globalfeed/testdata/{usgs_significant_week,nhc_currentstorms,nws_active_severe,nws_active_unfiltered_trimmed}.json — the DISCOVER probe samples, promoted at PLAN exit (one home)
CREATE: domains/globalfeed/detail_test.go         — COV per source (render list present), bounds, name-aware Sentence, memo skip
MODIFY: platform/snapshot/types.go                — Alert.SenderName
MODIFY: domains/weather/nws/alerts.go             — bounds on every field; SenderName decoded
CREATE: domains/weather/nws/alerts_test.go        — bounds + SenderName (the package has no alerts test file yet)
CREATE: domains/severe/severe.go                  — Tab, Row, Detail, NormalizeID, Classify, Guard, Union, Sort, Cap, ByTab
CREATE: domains/severe/record.go                  — RecordOf (the one class switch) + Plain on every field
CREATE: domains/severe/severe_test.go             — id forms, union/de-dup, guard, cap, classify, sort
CREATE: domains/severe/record_test.go             — per-class records; escape injection
MODIFY: app/ticker.go                             — Locate before Within; deck.SetFeed; narration strings
CREATE: app/severe.go                             — severeDeck + toSevereMsg (zone rule, formatting)
CREATE: app/severe_test.go                        — deck publishes on change only; zone formatting; fetch count unchanged
MODIFY: app/pipelines.go                          — onPublish / recent run → deck.Trigger
MODIFY: app/dashboard.go                          — construct the deck; pass to ticker + pipelines
MODIFY: app/ticker_test.go                        — narration strings (NAR); Locate-before-radius
MODIFY: platform/render/theme.go                  — EventCat*BG tokens (defaultTheme)
MODIFY: platform/render/themes.go                 — Monochrome overrides
MODIFY: platform/render/sgr.go                    — CategoryTone
MODIFY: platform/render/theme_test.go             — independence guard (+ positive control); fg×bg AA on both substrates
CREATE: modes/tty/severe.go                       — SevereMsg/Row/Record/Tab, severeTabs, state helpers, browse + detail renderers
CREATE: modes/tty/severe_test.go                  — nav, tabs, esc/esc, empty state, --ascii, goldens 80/100/120
MODIFY: modes/tty/dashboard.go                    — state fields, modalSevere, key "severe", toggleModal, dispatch SevereMsg, tickNeeded
MODIFY: modes/tty/nav.go                          — route modalSevere → handleSevereNav
MODIFY: modes/tty/view.go                         — modalView memo + case, modalWidth case, modalLines case
MODIFY: modes/tty/memo.go                         — modalKey / modalMemo
MODIFY: modes/tty/memo_test.go                    — modal memo invalidation table; loading-row positive control
MODIFY: modes/tty/help_about.go                   — "severe" in NAVIGATE
MODIFY: modes/tty/modal_test.go                   — "severe" opener + marker
MODIFY: modes/tty/bench_test.go                   — Severe hit/miss budgets; 80x24 row
CREATE: platform/render/severe_table.go           — SevereTable (go-studs DataTable), Railify + RailGlyphs, bracketSpread (the render layer stays the only kit consumer)
MODIFY: modes/tty/body.go                         — railify → render.Railify with the options' rail glyphs (RECENT gains --ascii rail)
MODIFY: modes/tty/status.go                       — "severe index: N rows / 500" gauge row (NFR-4)
MODIFY: third_party/go-studs/LOCAL_CHANGES.md     — upstream candidate: per-row prefix column + cursor index on DataTable
MODIFY: modes/tty/alerts.go                       — Plain on AreaDesc/Instruction/Headline (NFR-6)
MODIFY: modes/tty/alerts_test.go                  — escape-injection on every [A] field
CREATE: scripts/quality/severe-modal.expect       — PTY journeys: w → SEVERE → esc esc; ctrl+s alias
MODIFY: README.md, CHANGELOG.md                   — key table, 0.13.0 notes
MODIFY: 06_docs/02_features/severe-alerts-modals/07-readiness/  — Linux gate checklist
```

## Requirement → task trace

(`P1-8` = `p1-domain.md` Task 1.8, and so on.)

| Req | Tasks |
|---|---|
| FR-1 keys | P3-3, P3-8, P4-3 |
| FR-2 tabs, highlight, opening rule, first line | P3-2, P3-4, P3-5 |
| FR-3 rows + rail | P3-5 |
| FR-4 in-place detail, esc/esc, chips | P3-3, P3-4, P3-6, P3-8 |
| FR-5 record list, COV, Plain | P1-1..5, P1-9, P1-10, P4-2 |
| FR-6 tokens/tone/guards | P3-1 |
| FR-7 `[A]` language | P3-5, P3-6 |
| FR-8 narration | P2-4 |
| FR-9 columns, count, stamp, zone rule, sort at publish | P2-2, P3-5 |
| FR-10 memo family | P3-7 |
| FR-11 publish path, normalise, guard, cap | P1-8, P2-1, P2-2 (P2-3 is the wiring) |
| FR-12 storm names | P1-2, P1-4, P2-4 |
| FR-13 `--ascii` | P3-5, P3-6, P3-8 |
| FR-14 empty state | P3-5, P3-8 |
| FR-15 severity glyph | P3-5 |
| NFR-1 zero fetches | P2-5 (by construction — the deck has no client; deck benchmark) |
| NFR-2 budgets | P3-9 |
| NFR-3 parse memo | P1-7 |
| NFR-4 cap 500 + gauge in [S] | P1-8, P2-2, P3-10 |
| NFR-5 bounds both paths | P1-1, P1-6 |
| NFR-6 Plain everywhere | P1-9, P4-1, P4-2 |
| NFR-7 radio untouched | P2-4 (strings only), P4-5 (R6 gate) |
| NFR-8 help | P3-3 |
| NFR-10 go-studs first + upstream candidate | P3-5 (`render.SevereTable`; LOCAL_CHANGES note) |
| NFR-9 PTY | P4-3 |
| NFR-11 narrow layout | P3-5, P3-8 |
| NFR-12 superseded guard | P1-5, P1-8 |
| NFR-13 seen.json | P2-6 |
| NFR-14 contrast | P3-1 |

## Order of execution

P1 (1.1 → 1.10) → P2 (2.1 → 2.6) → P3 (3.1 → 3.10) → P4 (4.1 → 4.5). Every batch ends with the full
enforcement surface (`make verify` **and `make p10`** — the P10 gate is not in `make verify` today, so it is
run explicitly; the largest new functions, `severeBrowseLines`/`SevereTable`, must stay within
P10-04's statement budget) — dogfood is per commit, not per milestone.
