# P4 build log — the Setup window, the Radio panel, the retirements (0.14.0)

```
Batch:   P4 — Setup per the mock, the panel per breakpoint, [S], the docs, the goldens
Date:    2026-08-30
Branch:  feature/multi-voice-support
Gate:    07-readiness/gates.md §1
CLI:     a2dh v1.17.0 (commit 7960f0e)
```

**The first batch a listener can judge.** Everything before this was reachable only by hand-editing a file
(RN-4). The UAT sheet is `07-readiness/uat-p4.md`.

## The open points, as taken

| OP | Ruling taken | Why |
|---|---|---|
| OP-1 | **`←→ Voice` and `p Preview` chips added** when a picker is focused | The mock's chip row names no key for changing a voice, and a control nobody can find is not a control |
| OP-2 | **The mock's DATA wording ships** | It is also what makes the two columns fit: the old labels were too wide for the left column |
| OP-3 | **A support line: "Nothing ticked: every class is muted."** | The rule is not guessable, and what the screen says must be what the ear gets |
| OP-4 | **The chips name the keys that operate the focused row** — `space Select`/`space Toggle` where it applies | With one keyboard rule `space` runs most of the window and the mock named it nowhere. Save stays on the last row; no `ctrl+s` chord, so M2's 11 keypresses stand |
| OP-5 | **Confirmed: a pinned footer** (`render.ScrollPanelFooter`) | At 80×24 a scrolling chip row is invisible exactly when a lost reader needs it |

**The mock's two misspellings ship corrected** ("Maritime Report", "Fire/Hotspots") — recorded here as a
deviation rather than made silently.

## What the batch is built on

**One row table** (`setup_rows.go`). The window went from three questions to twenty rows across four groups;
the old focus enum was a small state machine, and four separate ideas of the order — the keyboard, the ›
mark, the scroll, the save — would have drifted. The table is the order, and everything reads it.

**One keyboard rule.** `tab` between the five groups · `↑↓` within one · `space` operates · `←→` cycle a
picker · `p` previews · `enter` advances, and **saves on the last row of its group**. That last part is a
change from the plan's letter and matters: a listener changing only their location must not press enter
through twenty rows to commit it, and every group ending in a Save is also how the three-question form
behaved before this batch.

## Three defects the work found

1. **The scroll must keep the whole row, not its first line.** Keeping only the head on screen let a failed
   save's *reason* scroll off — a reason nobody can act on. It now tracks each row's full span.
2. **Width and height are different questions.** "Compact ⇒ narrow player" cost a short-but-wide terminal
   its labelled controls for no gain. Caught in the golden review. Width decides the control style; height
   decides the row count.
3. **The Setup save never persisted the cast.** Found by `make p10` reporting `castForSave` and
   `mutedClassKeys` as *unused* — a dead-code finding that was really a missing wire. The window would have
   drawn a cast a listener could edit and then thrown it away on Save.

## Gate

| Gate | Result |
|---|---|
| Unit + race | `go test ./... -race` green |
| Verify | **ALL GATES GREEN** |
| Harness | `a2dh validate` 17/17 |
| Alloc pins | green; the Setup window **re-pinned** (Task 4.11) |
| P10 | **0 live, 0 unmatched** (111 dormant) · `07-readiness/p10-p4.json` · **no new exemptions** |
| Declsets | `app`, `modes/tty`, `platform/render`, `platform/snapshot` re-captured |
| PTY | `make pty-severe` green |
| Goldens | six re-recorded + **three new Setup frames** (133×44, 80×24, 133×44 `--ascii`) |
| Docs | `where-things-happen` green; **NFR-8 grep = 0** |
| Build | `make build` → `./dist/watchpost` |

### P10: six live findings, all FIXED, none exempted

`setupColumn` (42 statements) and `handleSetupKey` (complexity 17) were split — the split also separates two
genuinely different questions ("which keys move the focus" from "what this row does with text").
`castNote`'s unused parameter went. `spokenPickerName` was deleted and its job moved to `pickerCell`, where
a name is actually drawn. And the two "unused" findings were the missing save wire, above.

### Alloc pins, re-measured (Task 4.11)

Against the **new** window with the correspondents focused — the heaviest state (pickers drawn, the focused
row's note built and wrapped, the two-column decision run):

| | hit | miss |
|---|---|---|
| 133×44 | **2,114** (was 3,046) | **3,895** (was 3,476 — ×1.12) |
| 80×24 | **1,799** (was 1,978) | **3,474** (was 2,478 — ×1.40) |

The hit path got *cheaper*. The miss grew, well inside the plan's rule that beyond **twice** the baseline the
builders are fixed rather than the pin raised — and it buys twenty rows where there were three. 80×24 grew
most because that is where the window **stacks**: it builds both columns and then lays them one after the
other.

### Goldens, reviewed row by row

The six existing frames changed by **exactly one line each**: the retirement of `[V] Voice` and `[T] Size`
from the player's control row (and, at 80 cols, the row that wrapping used to need). Nothing else moved.

The three new Setup goldens use a **complete** fixture — a voice list, a preview hook and an
installed-check. Incomplete, every picker reads "—" and every row grows a "No voices available" note, and
the golden would pin a window no listener will ever see.

**The ASCII golden earned its place immediately**: it caught the scroll rail (`▲ █ ▼`) and the focus mark
(`›`) still hard-coded rather than drawn from the glyph set. Both now go through `Glyphs`, and the test
asserts no Unicode mark survives `--ascii`.

## Deviations from the plan

1. **`enter` saves at the end of every GROUP**, not only at the end of the window (above).
2. **`Glyphs` gained `Up`/`DropDown` as separate marks.** The picker's `▾` and the scroll rail's `▼` are
   different marks with the same name in the plan; conflating them put a rail arrow in the pickers.
3. **`render.ScrollPanelFooter` is new.** OP-5's pinned footer needs the panel to reserve rows inside its own
   frame; drawing the footer after the panel put it outside the box.
4. **`setupWidth` mirrors `helpWidth`.** The plan assumed the fixed 78-column window; two columns need the
   content to decide the width, exactly as Help does.

## Notes carried to the release

1. **M3's two listening trials are the HUM LEAD's** (`gates.md` §2), recorded in `07-readiness/validate/m3.md`.
2. **The journey's cast step counts its own keypresses** and fails above 11 (AM-18/MVS-D-40). It runs on a
   fresh HOME, in the Linux protocol and at release.
3. **Every preset's sound is the HUM LEAD's pass** (`tones.md` §1) and is expected to be revisited at UAT.
