# Goldens reviewed against the mocks — 0.14.0

**Delegated by the HUM LEAD, 2026-09-06** ("Approved for you to do this"), against
`02-analysis/mocks/setup.md` (MVS-D-10, *"reproduce exactly, colour is the HUM LEAD's pass"*).
Colour is out of scope by that instruction; this is layout, glyphs, labels and controls.

**Reviewed:** `setup-133x44.golden`, `setup-80x24.golden`, `setup-133x44-ascii.golden`.

---

## Divergences that are RULED, and correct

Every one of these differs from the mock because a later ruling changed it. Listed so the next
reader does not re-open them.

| Mock | Shipped | Authority |
|---|---|---|
| Title `Setup / Configs` | `Settings` | the window was renamed this release |
| Two groups (DATA, CORRESPONDENTS) | five (DATA, WATCHPOST UI, ALERTS-EVENTS, ALERTS-TONE, CORRESPONDENTS, RELAY REPLAY) | 0.14.0 added the theme row, Show Units in, Radio Convention, Relay Replay |
| `Significant Quakes & Disasters` | `Disaster Events` | `Sig. Quakes` → `Disasters` |
| `Maritme Report`, `Fire/Hostpots` | `Marine Report`, `Fire/Hotspots` | the mock's own note: *"spellings reproduced as drawn — the HUM LEAD's to correct"* |
| `○ All Tones On` / `○ Mute:` + checkboxes | a row per class, `←  Enabled  →` | MVS-D-26, per-class mute |
| chips `enter Next`, `↑↓ Pick`, `esc Cancel` | `enter Save`, `↑↓ Move`, `esc Close`, `←→ Voice`, `p Preview` | MVS-D-27 (preview); `Save`/`Close` state the two-exits rule truthfully |
| header `ALERTS - TONE  ( [M] toggles )` | `ALERTS - TONE` | `[M]` now *opens* Settings at the tones; the old hint would be false |

**One that looks like a defect and is not.** The tone group says **Maritime** while the
correspondents group says **Marine Report**. This is deliberate: `domains/radio/cast/tone.go:112`
records **MVS-D-33** — the storm class keeps the mock's word, and it is a different thing from the
marine *report*. Flagged here because it is the first thing a reviewer will challenge.

## Fidelity confirmed

- **133×44** — two columns, the Help modal's collapse rule; scroll rail `▲ █ ▼` present at the right.
- **80×24** — collapses to one column as the mock's rule requires; rail intact; the footer chips
  **wrap to two lines rather than clip**, which is the stated narrow-window behaviour.
- **Pickers** — the mock's `│ name │ ▾ │` dropdown ships as an `←  name  →` cycler. Consistent
  everywhere, matched by the `←→ Voice` chip, and the same control the theme row uses.

## Findings

**F-46 — the mock's *All Reports* picker is missing, and the role is config-only.** The cast tree
still has the level (`All → Standard → {Weather, Maritime, Fire, Seismic, Station}`) and
`cast.Groups()` still returns the mock's two group pickers, but Settings draws five rows and none is
Standard. It was specified in four places — the mock, PLAN §224, FR-8, `p4-ui.md:152` — and the
residue shows it was dropped rather than redesigned: **`roleStandard` is declared and referenced
nowhere**, and `Groups()` has no production caller. Consequence: the CHANGELOG's *"a different voice
for the alerts than for the reports"* takes four separate rows, and **Station still inherits Standard,
so the station reads keep the root voice** while the four reports changed. **Needs a ruling** —
restore the row, or ratify the flattened design and remove the residue and the wording it breaks.

**F-47 — `--ascii` survivors in the radio panel.** `▶`, `■`, `█` and `░` are hardcoded in
`radio_panel.go` rather than taken from the glyph set, in the same function whose *failed* branch
does use it. `■` has no glyph-set entry at all. It hides because the Setup ASCII test's forbidden
list is scoped to that window's marks while the golden it checks is the whole frame — so panel
glyphs are captured and never scanned. Small; the scan should widen with the fix.

## Dispositions — HUM LEAD, 2026-09-06

**F-46 is not a defect. The shipped window is the record; the spec is the stale artefact.**
*"That's residual spec that wasn't changed as a result of UAT; the settings record reflects the
truth, which was discovered after trying the 'inheritance' model — it was too complicated and the
same result could be had by just making the controls easier and giving the user fine-grained
controls outright, which is what we have now."*

Done under that ruling: `roleStandard` and `cast.Groups()` deleted — the residue that made a decision
look like an omission; the CHANGELOG and README rewritten, because both still promised the retired
alerts-vs-reports split; `mocks/setup.md` and `p4-ui.md` Task 4.6 marked SUPERSEDED at the head, the
latter because its line *"an override inherits from All Reports, not from the root"* described
behaviour the window no longer has. `Standard` stays in the registry; `[radio.voices.standard]` stays
a config key.

**F-47 fixed — and widening the scan found a third instance.** `Stop` added to the glyph set; the
volume bar and every branch of `radioStateLabel` now take their marks from it.
`TestASCIIFramesCarryNothingButASCII` replaces a list of forbidden glyphs with the question that
needs no list — *is anything in an `--ascii` frame outside ASCII?* — allowing `°` alone. Its first
run caught `·` on the tape, folded at `tapeLine`. Only the two `--ascii` goldens moved (3 lines);
every unicode golden is byte-identical, which is the check that the fix touched only the ASCII path.

## Verdict

**The Setup goldens reproduce the mock faithfully** once the ruled changes are set aside, and the
two collapse rules and the rail behave as specified. **Two findings, neither cosmetic**: one missing
control that the mock, the plan and the requirement all specify (F-46), and one `--ascii` defect
(F-47). Neither was visible from the goldens passing — the goldens pin *what is drawn*, and the
question this review asked was *what should be drawn and is not*.

**Both are now closed** (see Dispositions): one was the spec being stale rather than the window being
wrong, and one was a real `--ascii` defect that a widened scan turned into three. **The review's own
lesson is that neither finding needed new information** — the mock, the registry, the glyph set and
`radio_panel.go` all said this plainly, and it took reading them side by side rather than reading a
green test result.
