# Q6 build log — Seams (`v0.10.0`)

**Batch:** Q6 of the plan of record v3 (§3 Q6; L3-F8..F15, A11-6).
**Approval:** "Approved; GO for 0.9.8; GO 4 Q6" (2026-08-26).
**Branch:** `feature/watchpost-performance-quality-pass`, commits `7c962dc`, `6e5c6f7` (+ docs, this log).
**Status:** APPROVED 2026-08-27 ("Go for 0.10.0 release, Q7 validate") — `v0.10.0` cut from this
tree; Q7 (VALIDATE) opens.

## 1. What changed, and why (junior-first)

Q6 removes the places where the same thing was written twice, and one place where a fact was kept
ten times.

- **One open window.** The dashboard kept ten booleans (`showHelp`, `showDetails`, …) and kept them
  exclusive by hand at ten reset sites. The red team read the sites and found them inconsistent
  (L3-F15); the test written first — for every window and every way of opening one, render the
  frame and count the window titles — was **red on eight cases**: Help opened over Alerts, About or
  Status; Remove over Help or Details; a failed voice change reopened the Voice chooser over Help,
  Details or Status. Now there is one `modal` value: `open(kind)` closes whatever was open,
  `toggle(kind)` closes the same kind, `View` draws the open window through a single switch, and
  the exclusivity test asserts it on the rendered frame (A11-6: `esc` never moves the focus).
- **Single owners.** `render.Thousands` (the acreage format was byte-identical in the FIRE rows and
  the broadcast), `geo.CompassIndex` (the arithmetic behind three word tables — the tables stay;
  8 vs 16 points for the spoken wind stays the excluded §0.9 decision), `render.DisplayCondition`
  (the modal's copy deleted), `tty.WatchCap` / `tty.RecentCap` (the app reads the tty's numbers
  instead of its own 50 and a literal 10), `Opts.Controls` (the modal footers' shape),
  `WrapSegments` returning rows (three callers re-split on `\n`).
- **Nothing on screen changed.** Every golden (plain, ASCII, colour-on) and every synth compose
  test is byte-identical.

## 2. Files touched

| Area | Files |
|---|---|
| `modes/tty` | `dashboard.go` (`type modal`, `open`/`close`/`toggle`, `toggleModal`, the key router), `view.go` (`modalView`, `modalWidth`, `modalLines`), `nav.go`, `modal_chooser.go`, `modal_location.go` (`WatchCap`, `RecentCap`, footers), `setup.go`, `detail.go`, `detail_fire.go`, `detail_marine.go`, `help_about.go`, `status.go`, `body.go`, `radio_panel.go`; `modal_test.go` (new: the exclusivity table), every test that read a flag |
| `platform/render` | `text.go` (`Thousands`, `Control`/`Ctl`/`CtlIf`/`Controls`, `WrapSegments` rows), `table.go` (`DisplayCondition`) |
| `platform/geo` | `CompassIndex` (+ test against the three old formulas) |
| `domains/radio/synth` | `fire.go` (`Thousands`, `CompassIndex`), `compose.go` (`CompassIndex`) — R6: smokes + soak in §7 |
| `app` | `dashboard.go` (`tty.RecentCap`), `pipelines.go` (the copy deleted) |
| docs | `CHANGELOG.md`, `docs/where-things-happen.md` (+1 row, +1 vocabulary), `docs/extending.md` (a new window = one constant), `architecture.md` §6 |
| records | `07-readiness/p10-q6.json`, `02-analysis/q6-soak-1h.csv`, `02-analysis/q5-counters/counters-q6-1h.json` |

## 3. Tests first (the pins)

| Pin | Guards |
|---|---|
| `TestExactlyOneModalRendersWhateverOpensOverWhat` (12 openers × 12 states) | at most one window title in the rendered frame; `esc` leaves `selected` unchanged — **red on 8 rows before the refactor** |
| `TestEveryModalHasItsMarker` | each window's frame marker is real (the test cannot pass vacuously) |
| `TestCompassIndexMatchesTheThreeOldFormulas` | every half-degree, 16- and 8-point, equals the arithmetic it replaced |
| `TestThousandsControlsAndWrapSegmentsRows` | the format, the footer shape (muted control included), the rows |
| goldens (plain, ASCII, colour-on); synth compose tests; declsets re-captured (tty, render, app — intentional) | byte-identical output |

## 4. Before / after

| Measure | Before Q6 | After |
|---|---|---|
| window-state fields on the model | 10 booleans, 10 reset sites, 4 parallel switch chains | 1 `modal`, 3 methods, 1 draw switch |
| latent stacking cases (rendered-frame test) | 8 | 0 |
| copies of the acreage format / compass arithmetic / condition vocabulary / list caps | 2 / 3 / 2 / 3 | 1 / 1 / 1 / 1 |
| modal footer rows hand-assembled | 6 | 0 (through `Controls`; 3 richer rows keep their own shape) |

## 5. Bounds stated (§0.8)

- `modal` has eleven values; a new window is one constant and three cases (documented in
  `docs/extending.md`); the exclusivity test's table must list it.
- `CompassIndex(deg, 0)` is 0, never a division by zero.

## 6. Decisions and non-decisions

1. **C3 re-check (plan §2.4 threshold: > 5 MB live or publish-coalescing failures attributable to
   the 50 schedulers).** From the Q5 soak: RECENT publishes 35/h (one per tier wave), goroutines
   273 flat, live heap 36–51 MB in total with no term attributable to the schedulers (~1 MB of
   parked goroutines). **The threshold did not trip; C3 = A stands** — recorded here as the plan asked.
2. **Compass 8 → 16 for the spoken wind** — untouched (excluded by HUM LEAD); the arithmetic is
   shared, the 8-word table stays.
3. **Control rows.** `Controls` owns the six simple footers; the three richer rows (Details' chip
   line, the voice chooser's, Setup's wrapped chips) keep their bespoke assembly — the OQ-6 ordering
   nit was not changed, since a reordering is a visible change under §0.9.
4. **Release `0.10.0`** per §0.7: user-visible seams (one window at a time is a behaviour people
   can notice — nothing stacks any more).

## 7. Gate

| Check | Result |
|---|---|
| `make verify` | ALL GATES GREEN |
| `make p10` | 0 live · 0 unmatched · `07-readiness/p10-q6.json` |
| `a2dh validate` | 18/18 |
| goldens, alloc budgets | byte-identical / green |
| the rendered-frame exclusivity test | green (was red on 8 rows) |
| R6: Synth / Relay smokes + `LiveRelay` (the synth's helpers moved to shared owners) | synth PLAYING 4 s; relay PLAYING 21 s; `LiveRelay` ok (§8) |
| R6: 1 h soak | heap flat 38–43 MB, goroutines flat, RECENT 33 publishes/h (§8) |

## 8. R6 smokes and the 1-hour soak (`dist/watchpost` at `6e5c6f7`)

- **Synth:** first data → **PLAYING in 4 s** → `[m]` Nearest Relay **PLAYING in 21 s** → clean quit.
- **`LiveRelay`:** ok (both relays play through the player, 9.5 s).
- **1 h idle soak** (60 s samples, `q6-soak-1h.csv`, `counters-q6-1h.json`): heap after GC 38–43 MB
  flat; RSS 99–138 MB; goroutines 277 → 273; disk files 814 flat; RECENT publishes 33/h (Q5: 35).
  The hour's requests: NWS 710 attempts / 596 net / **112 renewals**, FIRMS 216 (tiles), CO-OPS 128,
  NDBC 104 / 26 renewals — the Q5 shape; TLS handshakes NWS 4, FIRMS 19, CO-OPS 11, NDBC 13. No
  fast-fails, no negative hits.

## 9. Carried forward

- Q7 next (VALIDATE): the 7-day macOS soak and Arch 72 h (HUM LEAD), `tools/slope` verdicts,
  `pprof -base` over ≥ 4 dumps, the baseline document, the pass's DEBRIEF.
- The 24 h Q0 soak ends ~2026-08-27 15:55 UTC — its σ result opens Q7.
- Follow-ups F-1, F-2 after the pass.
