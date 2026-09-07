# P4 — the Setup window, the Radio panel, the retirements (multi-voice-support, 0.14.0)

```
Goal:         The listener assigns the cast and mutes tone classes in Setup, per the HUM LEAD's mock; [V] and [T]
              leave the Radio panel and V opens Setup at the correspondents; [S] says who speaks for whom and
              why; the same answer is readable without the TUI.
Architecture: plan.md §2.7; 02-analysis/mocks/{setup,radio-panel}.md (reproduced exactly — colour is the HUM LEAD's pass)
Branch:       feature/multi-voice-support
Gate:         07-readiness/gates.md §1
```

Task shape only; code at BUILD (`AP-PLANCODE-01`). The PLAN sketches are in `prior-art/p4-ui-code.md`; of this
batch only the test scaffolding was ever compiled — the Setup rows were hand-computed and matched their
builders, everything else is unverified. Treat it as a comparator, not a source.

## Constraints every task honours

- **`modes/tty` imports no `domains/` package** (`make lint-imports`). The cast reaches the TUI as strings: role
  keys are the registry's own words, supplied by `app`; the tone classes arrive as a key/label list; a parity
  test in `app` pins the words to the registry.
- **The mock is the design.** Rows, spacings, group titles and chip text are the mock's; the two misspellings
  ship corrected. A deviation is an open point for the HUM LEAD (OP-1…OP-5), never a silent choice.
- **One keyboard rule for the whole window.** `tab`/`shift+tab` move between *questions* (the first row of each
  group: location · key · events · tone · correspondents); `↑↓` move among a group's rows; `space` selects a
  radio or toggles a checkbox; `←→` cycle the focused picker; `enter` accepts and moves on, and on the last row
  **saves**; `esc` cancels. Note for BUILD: bubbletea v2 names the space key `"space"` — a `" "` case never
  fires, including the one in today's `setup.go`.
- **Notes belong to the row that raised them**, are wrapped to the note column (never truncated, never wider
  than the picker rows so focusing one cannot flip the two-column layout), and the scroll keeps the row **and
  every line of its note** on screen. An empty note draws nothing.
- Reuse, never re-roll: the Help modal's two-column helpers, the existing radio-mark and key-cap renderers, the
  text wrap/pad/truncate helpers, the modal memo, the scroll rail. Marks come from the glyph set so `--ascii`
  is automatic.

## File map

```
CREATE: modes/tty/setup_rows.go            — the row table: focus ids, groups, kinds, the keyboard helpers
CREATE: modes/tty/setup_tones.go           — the ALERTS - TONE group
CREATE: modes/tty/setup_cast.go            — the WATCHPOST RADIO - CORRESPONDENTS group
CREATE: modes/tty/setup_helpers_test.go, setup_{layout,tones,cast,golden}_test.go
CREATE: modes/tty/modal_theme.go (+ test)  — the theme chooser, moved whole
DELETE: modes/tty/modal_chooser.go (+ test) — the voice chooser
MODIFY: modes/tty/{setup,view,dashboard,memo,radio_panel,layout,help_about,status,body}.go (+ their tests)
MODIFY: platform/render/{units,panel,contrast,text}.go — the new glyphs; the rail; the AA pairs
MODIFY: platform/term/term.go (+ test)     — Merge tolerates an override for a retired action
CREATE: app/hostfacts.go (+ test)          — one cast.Host for the deck and the report; CastReport
MODIFY: app/{cast,dashboard,stats,dump,voices}.go (+ tests) — the view mapping, the save hooks, the [S] rows
MODIFY: cmd/watchpost/root.go (+ report_test.go) — the cast block under --verbose
MODIFY: scripts/quality/validate-journey.expect  — the cast step, counted
MODIFY: README.md, CHANGELOG.md, docs/{where-things-happen,extending}.md, docs/img
MODIFY: modes/tty/testdata/*.golden        — re-recorded once, plus three Setup frames
```

---

### Task 4.0 — the test scaffolding

**Files:** `modes/tty/setup_helpers_test.go`.

**Contract:** the helpers every later RED test uses — a model constructor, a plain-text line reader, line
lookups, `openSetupAt`, and a key-press driver that runs any command the update returns and feeds the message
back (the shape the existing Setup tests already use). Record two probes in the build log because they decide
strings elsewhere: the space key's name, and how a key cap renders with colour off.

**Verify:** `go vet ./modes/tty`

---

### Task 4.1 — `term.Merge` tolerates an override for a retired action (FR-14; RS-21)

**Contract:** an override in `[keys]` naming an action this build no longer has is **dropped with a note**, never
an error — a rebound `T` must not break startup. The notes reach `[S]`; the action name is config text, so it is
rendered plain.

**Test intent:** a known override applies, an unknown one is dropped, and exactly one note names it.

**Verify:** `go test ./platform/term ./modes/tty -run 'Merge|Keys' -count=1`

---

### Task 4.2 — `V` opens Setup at the correspondents; `[T]` retires; the choosers part ways

**Contract:** `voice` keeps its binding and its place in Help's RADIO group but now opens Setup **scrolled to**
the Correspondents group; `radio-size` and the min-player state are deleted. The **theme** chooser moves whole to
its own file *before* the voice chooser's file is deleted — the two share a file today and only one is retiring.
The deck's voice-note message gains the voice it is about (Task 4.6 needs it) and is rendered plain on receipt.

**Existing pins this changes:** the player tests that assert the size control, the modal-marker test, the memo
key's voice fields.

**Verify:** `go build ./... && go test ./modes/tty -run 'VoiceKey|Theme|Help|Keys' -count=1`

---

### Task 4.3 — the Radio panel per breakpoint (MVS-D-23/24)

**Contract:** three fixed layouts chosen by **terminal columns** as the mock states them (the mock's numbers are
outer columns, not inner width): wide, medium, narrow — each with a standard vertical size, so `[T]` is not
needed. A height-compact frame takes the narrow player whatever the width (today's rule, kept). The header is the
station name (wide only), the on-air detail, the volume bar and the state — **no voice name** (MVS-D-24). The
visualizer area exists at wide and medium; the `v` control appears only where that area does. One builder feeds
both the rendering and the height budget, and the frame builds the player once per frame, not once per layout
pass.

**Test intent:** each breakpoint's row count, controls and header at its boundary column and one column either
side; the compact rule; no retired control anywhere.

**Verify:** `go test ./modes/tty -run 'RadioPanel|Layout' -count=1`

---

### Task 4.4 — the row table, the body built once, two columns, focus-following scroll (RS-19)

**Contract:** one table describes every focusable row — its group, its kind (input · radio · checkbox · picker)
and the role key it edits. The keyboard rule, the focus order, the mark and the scroll all read that table; the
focus enum becomes an ordering, not a state machine. The window's body is **built once per change** and memoised
(keyed on a generation counter plus the terminal size), and each line records which rows it draws — a class row
holding two checkboxes carries both, so the scroll can find either. Two columns when the Help rule says they
fit, stacked otherwise. **Every writer of the Setup state bumps the generation** — key handling, opening, a note
arriving, a resolved location, a save failure, a snapshot arriving, a resize — or the window renders one
keystroke late. The chips are a **footer pinned under the scroll window** (they must not scroll away at 80×24;
OP-5 confirms that reading of the mock). One `setupSave` owns the save rules for every group's last row,
including the typed FIRMS key and the bounce when no location is chosen.

**Test intent:** the table's shape and the five tab stops; two columns at 133 and stacked at 80; **every** row,
when focused, sits inside the scroll window at 80×24 — including its note's last line; the frame's height fits
80×24 with the widest footer.

**Verify:** `go test ./modes/tty -run Setup -count=1`

---

### Task 4.5 — the ALERTS - TONE group (FR-11; MVS-D-26/28)

**Contract:** the mock's rows — `All Tones On (Default)` / `Mute:` and a checkbox per class, one on the first
row then two per row. `space` selects a mode or toggles a class (ticking one implies Mute); `←→` move between
the two cells of a paired row; `↑↓` walk the group in reading order. When Mute is chosen with nothing ticked,
a support line says the rule the ear will follow (OP-3). The class list and its order come from the app.

**Test intent:** the rendered rows against the mock, character for character; the toggle and the mode rules; the
support line only in that state.

**Verify:** `go test ./modes/tty -run SetupTones -count=1`

---

### Task 4.6 — the WATCHPOST RADIO - CORRESPONDENTS group (FR-4; MVS-D-25/27)

> **THE CONTRACT BELOW WAS SUPERSEDED AT UAT** (HUM LEAD, 2026-09-06): the inheritance model *"was
> too complicated and the same result could be had by just making the controls easier and giving the
> user fine-grained controls outright."* The mode radio, the **All Reports** picker and the enabling
> checkboxes are retired; five plain rows ship, each with its own picker, each naming the voice it
> inherits when it has none of its own. **In particular the sentence "an override inherits from *All
> Reports*, not from the root" no longer describes the window** — with no All Reports row, an unset
> report row shows the root voice, which is what the golden draws ("Inherits System Voice."). The
> `Standard` role stays in the registry and `[radio.voices.standard]` stays a config key. See
> `07-readiness/goldens-vs-mocks.md`.

**Contract:** the mock's two-way radio — `Single Voice` with the root's picker, or `Correspondent Cast` with the
Alerts and All Reports pickers and four optional per-report overrides. A picker shows the voice that will
actually speak: its own name, else **the name it inherits** — and an override inherits from *All Reports*, not
from the root (the registry's parent), or the row lies about what the listener will hear. `←→` cycle **by the
entry the picker holds** (an "inherit" entry sits at the head of a group list, so an explicit root voice stays
reachable and `←` is never a dead key); choosing a voice for a role turns the cast on, because a pick under
Single Voice would otherwise be silent. `p` previews through the deck (ducked by the Director); on a voice that
is not installed the first `p` **offers** the download and the second proceeds (FR-4's "asks once"). The deck
owns the progress and failure words — one owner — and a failure's reason is not overwritten by a clear. Notes
under the focused row: not-installed (with the size), no voices, no audio device, inherits. A name too long for
the picker column is truncated there and given in full in the note. The root's display seed is **display only**:
a save writes what the listener chose, never a name they never picked.

**Test intent:** the rendered rows against the mock (with a complete fixture: a voice list *and* a preview hook,
or a note shifts every row); cycling from an inherited name; the ask-once flow; the four notes; a save writing
exactly the cast the rows showed, with unticked overrides cleared.

**Verify:** `go test ./modes/tty -run 'SetupCast|Picker|Notes' -count=1`

---

### Task 4.7 — the hooks; `[M]` = tones only (MVS-D-26)

**Contract:** the TUI gains the cast, the tones, the class list and four hooks (save cast, save tones, list
voices, preview); it loses the old voice-chooser hooks. `[M]` flips the tone mode and persists it — the words
always read. The masthead states the tone truth **in every form of its width ladder** (on · muted, all · muted,
n of N), and where the ladder drops the label the state is still readable in `[S]` (Task 4.8); the chip's verb
matches the state it will produce. The app maps the config to the view and back: this platform's half of each
pair, catalogue names on screen and catalogue keys on disk, the other OS's half untouched (FR-8). The role keys
come from the registry, not from literals; `runtimeGOOS` is the production seam P1 declared.

**Test intent:** a save writes the pairs and the mode and re-casts the deck; a Linux round-trip shows a name and
stores a key; the class list matches the registry; `[M]` writes the mode and nothing else.

**Verify:** `go test ./app ./modes/tty -run 'Cast|Tones|Setup' -count=1 && make lint-imports`

---

### Task 4.8 — `[S]`, and the same answer without the TUI (FR-7; NFR-5)

**Contract:** `[S]` gains **CAST** (a header row and one row per assignable role: role · requested · spoken ·
which link won · the install state · why the requested one lost — columns sized from the widest label the
registry can produce, padded by display width), **TONES** (the mute state in words, with the muted classes), and
**CONFIG** (the ignored `[keys]` overrides, the unknown `[radio.*]` keys, and `cast.Validate`'s problems —
de-duplicated and capped). Every string is plain: these are config-controlled. `watchpost report --verbose`
prints the same cast and tone answers for a shell, built from config and host facts **without a deck** — one
`cast.Host` implementation shared with the deck (and one discovery, under one ceiling), not a second one. The
diagnostic dump carries the same rows.

**Test intent:** the rendered `[S]` blocks including a reason and the header; the hostile fixtures produce no
escape or newline in any rendered line; `report --verbose` lists the cast and the tone state.

**Verify:** `go test ./modes/tty ./app ./cmd/... -run 'Status|Cast|Dump|Report' -count=1`

---

### Task 4.9 — help, README, CHANGELOG, docs, glyph parity, NFR-8

**Contract:** Help's RADIO group and the two changed key labels; the masthead label. Glyph parity: the focus
mark, the warning and health marks, the new dropdown/rail/arrow/ellipsis/bullet marks and the scroll rail all go
through the glyph set, so `--ascii` needs no special cases. Contrast: register the Setup window's foreground
tokens against the modal ground — **not** the key-cap chip, which carries its own background and would drive the
AA pass to darken every theme; give composite chips their own self-check instead. README: the Correspondents
group, the tone mutes, the maritime report, the panel per breakpoint, the new config keys with the
comments-dropped-on-save note, the overridable script folders, and the HUM LEAD's fresh captures; the retired
screenshots go. CHANGELOG: Added · Changed — including plainly that **`[M]` now mutes tones only, so a listener
who muted alerts in 0.13.0 hears the words again** · Fixed. `where-things-happen` gains the rows this batch's
symbols own. **NFR-8:** one grep pattern, owned by this task, over code, help, README and docs → zero.

**Verify:** the grep; `go vet ./...`; `go test ./cmd/... -run WhereThingsHappen`

---

### Task 4.10 — the PTY journey (M2) and the M3 arms

**Contract:** the journey gains a cast step from the dashboard: open Setup at the correspondents, turn the cast
on, pick a voice for Alerts, save — asserting the group on screen and, after the save, `[S]` showing the chosen
name against the Breaking role. The script **counts its own keypresses** and fails above the measured path
(11 — AM-18, pending E-9). Invoke it as the script expects: a fresh HOME and its log path. M3's two listening
trials are the HUM LEAD's UAT (`gates.md` §2), recorded in `07-readiness/validate/m3.md`.

**Verify:** `make pty-severe`, then the journey on a fresh HOME.

---

### Task 4.11 — the Setup-open allocation pin (NFR-3)

**Contract:** P0 recorded today's Setup window as an informational baseline; this task **measures the new
window** (both sizes, hit and miss, correspondents focused) and pins it spike-then-pin. The body-built-once rule
is what keeps the miss path near the baseline; if the miss exceeds twice it, the builders are fixed rather than
the pin raised.

**Verify:** `go test ./modes/tty -run AllocBudget -v`

---

### Task 4.12 — goldens, once

**Contract:** three new Setup frames (80×24, 133×44, and 133×44 `--ascii`) on the same fixture the row tests use,
plus a re-record of every frame golden carrying the Radio panel and the severe window. Review the diff row by
row against the two mocks before committing. The ASCII frame is where the glyph parity of Task 4.9 is proven.

**Verify:** `go test ./modes/tty -run Golden -count=1`

---

### Task 4.13 — P4 gate

Run `07-readiness/gates.md` §1 over the whole tree (declsets before the test line; stage before `make p10`), plus
`make pty-severe` and the journey.

- **P10 ledger:** rows only from the run; delete the retired file's rows.
- **Build log** `p4-build-log.md`: the OP rulings as taken, the golden review, ledger decisions, deviations.
- The three readiness documents are checked off here before SHIP.

**UAT:** Setup at 80×24 and 133×44; the three Radio breakpoints; `V`; `[M]`; the cast on the air.

## Open points for the HUM LEAD (at the PLAN exit)

- **OP-1 — the `←→ Voice` chip.** The mock's chip row does not name the key that changes a voice; this plan adds
  `←→ Voice` beside `p Preview` while a picker is focused. Keep, or hold to the mock?
- **OP-2 — the DATA row wording.** The mock re-words today's two questions; this plan ships the mock's words.
- **OP-3 — "Mute:" with nothing ticked** means every class; the mock has no words for it, so this plan adds a
  support line and says so on the masthead. Keep, or leave the rule to the README?
- **OP-4 — the chip wording, the EVENTS rule, and M2's number.** With one keyboard rule, the mock's `↑↓ Pick`
  chip is true only of the location hints; `space` — the key that operates 14 of the 20 rows — is named on no
  chip. Reword, or hold to the mock? And Save lives on the last row (11 keypresses; a `ctrl+s` chord would
  lower it and is the only lever on M2 — see E-9).
- **OP-5 — the chips as a pinned footer** rather than the frame's last scrollable row. Confirm that reading.
