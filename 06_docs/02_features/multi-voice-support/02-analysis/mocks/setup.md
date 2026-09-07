# Setup mock — HUM LEAD, 2026-08-29 (MVS-D-10; reproduce exactly, colour is the HUM LEAD's pass)

> **SUPERSEDED AT UAT — the shipped window is the record, not this mock.** HUM LEAD, 2026-09-06:
> *"That's residual spec that wasn't changed as a result of UAT; the settings record reflects the
> truth, which was discovered after trying the 'inheritance' model — it was too complicated and the
> same result could be had by just making the controls easier and giving the user fine-grained
> controls outright, which is what we have now."*
>
> So the two-way **Single Voice / Correspondent Cast** mode radio, the **All Reports** group picker
> and the per-report **enabling checkboxes** below are all retired. The window ships **five plain
> rows** — Alerts / Takeovers, Location Report, Marine Report, Fire/Hotspots, Seismic Reports — each
> with its own picker, and a row with no assignment of its own shows the voice it INHERITS, so no
> listener has to combine two controls to learn one answer. `[radio.voices.standard]` remains a
> config key; it simply has no row. Read the rest of this file as the design that was tried, and
> `07-readiness/goldens-vs-mocks.md` for what shipped and why.

"Follow the same column-group collapse / rollover rules as the Help modal" — two columns when the width
allows (the Help modal's rule), the groups rolling into one column below that; a scroll rail on the right.

```
╭─── Setup / Configs ──────────────────────────────────────────────────────────────────────────────────────────────╮
│                                                                                                                ▲ │
│    DATA                                                 WATCHPOST RADIO - CORRESPONDENTS                       │ │
│                                                                                                                █ │
│    ›  Default location: │Oceanside, CA 92057            ›  ○ Single Voice │ System Voice     │ ▾ │ (Default)   │ │
│       City Name, "City", or Zip                         ›  ○ Correspondent Cast:                               │ │
│                                                            ›  Alerts / Takeovers - │ System Voice     │ ▾ │    │ │
│    ›  NASA FIRMS key: stored (…d9a6) — ✔ working           ›  All Reports        - │ System Voice     │ ▾ │    │ │
│       Paste new key to replace — empty keeps                                                                   │ │
│       Key:                                                 ...except when there's a:                           │ │
│       ctrl+r  Reveal key                                   › [ ] Location Report - │ System Voice     │ ▾ │    │ │
│                                                            › [ ] Maritme Report  - │ System Voice     │ ▾ │    │ │
│    ALERTS - EVENTS                                         › [ ] Fire/Hostpots   - │ System Voice     │ ▾ │    │ │
│                                                            › [ ] Seismic         - │ System Voice     │ ▾ │    │ │
│    ›  ○ All locations (Default)                                                                                │ │
│    ›  ○ Within │    mi of Default Location                                                                     │ │
│                                                                                                                │ │
│    ALERTS - TONE  ( [M] toggles )                                                                              │ │
│                                                                                                                │ │
│    ›  ○ All Tones On (Default)                                                                                 │ │
│    ›  ○ Mute:                                                                                                  │ │
│       › [ ] Significant Quakes & Disasters                                                                     │ │
│       › [ ] Warnings    › [ ] Watches                                                                          │ │
│       › [ ] Advisories  › [ ] Special Statements                                                               │ │
│       › [ ] Maritime                                                                                           │ │
│                                                                                                                │ │
│                                                                                                                │ │
│   tab  Next question    enter  Next    ↑↓  Pick   esc  Cancel                                                  ▼ │
╰──────────────────────────────────────────────────────────────────────────────────────────────────────────────────╯
```

## Reading of the mock (for PLAN P4)

- **Frame:** the rounded modal (`╭ ╮ ╰ ╯`), title `Setup / Configs`, a scroll rail (`▲ █ ▼`) at the right — the
  window scrolls (RS-19 closed by design), the focused row always on screen.
- **Left column — DATA** (today's two questions, re-labelled: `Default location: │…`, `NASA FIRMS key: stored (…d9a6) — ✔ working`
  with the paste/reveal lines) · **ALERTS - EVENTS** (today's radius: `○ All locations (Default)` / `○ Within │    mi of Default Location`)
  · **ALERTS - TONE ( [M] toggles )**: `○ All Tones On (Default)` / `○ Mute:` with a checkbox per class —
  *Significant Quakes & Disasters* · *Warnings* · *Watches* · *Advisories* · *Special Statements* · *Maritime*
  (two per row where they fit).
- **Right column — WATCHPOST RADIO - CORRESPONDENTS:** a two-way radio: `○ Single Voice │ System Voice │ ▾ │ (Default)`
  (one picker: the root) **or** `○ Correspondent Cast:` with two group pickers — `Alerts / Takeovers` and
  `All Reports` — then `...except when there's a:` and four optional per-report overrides, each a checkbox that
  enables its picker: `[ ] Location Report` · `[ ] Maritme Report` · `[ ] Fire/Hostpots` · `[ ] Seismic`
  (spellings reproduced as drawn — the HUM LEAD's to correct).
- **Pickers** are the `│ <name> │ ▾ │` control: a dropdown over the voice list (the old chooser's list, inline).
- **Footer chips:** `tab  Next question    enter  Next    ↑↓  Pick   esc  Cancel` (bare keys, as the Radio mock).
- **Mapping to the cast tree (data-shape.md §1) — a subset, the model unchanged:** Single Voice → `voice` (the
  root); Alerts / Takeovers → `alerts` (breaking and severe_read inherit it); All Reports → `standard`; the four
  overrides → `weather` · `maritime` · `fire` · `seismic`; **Station has no picker** (it inherits *All Reports*);
  Breaking and Severe read have no separate pickers (they inherit *Alerts / Takeovers*).

## Open points — RULED by the HUM LEAD 2026-08-29 (MVS-D-25 a mode, assignments kept — values of record: `[radio] cast = ""` (single voice, the default) | `"cast"` (`data-shape.md` §2) · MVS-D-26 per-class mute, `[M]` tones only · MVS-D-27 the `p  Preview` chip · MVS-D-28 the classes as drawn)

1. **Single Voice vs Correspondent Cast** — a *mode*: switching back to Single Voice keeps the cast's assignments
   (a `[radio] cast = "single" | "cast"` key, default single, the pairs kept), or clears them? *(recommend: a mode key;
   assignments kept)*
2. **The tone section is per-class MUTE, with no "share the Warnings tone" switch.** Reading: MVS-D-9's share
   switch is **dropped** (each class always sounds its ratified preset), a per-class mute is **added**, and `[M]`
   toggles *All Tones On ↔ Mute* (the mute set). Does `[M]` still also mute the takeover's *narration* as today
   ("Mute Severe Alerts"), or tones only now? *(recommend: `[M]` = tones only; the words always read)*
3. **Preview** — the mock shows no preview key; FR-4 asks for a per-role preview. Add a `p  Preview` chip that
   appears when the focus is on a picker (recommend), or drop previews from Setup?
4. The classes: *Significant Quakes & Disasters* and *Warnings* are separate checkboxes though they share the
   dual-tone preset (MVS-D-11); *Maritime* = the Tropical / Winter Storm class (the low sweep). Confirm.
