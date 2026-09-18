# MVS-D-92 — a settings row belongs to a surface

**Status:** BUILT, 2026-09-12.  D-91 stage B.
**Ruled by:** HUM LEAD, 2026-09-12.

> "Settings that are truly shared should be (units / time / theme / etc).  Settings that are common,
> but can or need different values depending on the mode should be able to support that (Default
> location vs. Transmitter / Alert fence/radius / etc).  Settings that are unique and specific to
> their mode should only appear in the settings modal of their mode, and should not be able to leak
> into the mode."

---

## The ruling was already made — D-18, and he approved it

`02-analysis/config-field-table.md`, **approved 2026-09-09**: 52 persisted paths, 25 ruling units,
codes **S / O / B / SPLIT**.  The three tiers above are that table's three codes.  And metric **M4 —
"settings bleed", target 0** — has existed since DISCOVER.

**So this was never a design question; it was a build-state question.**  `setup.go` and
`setup_layout.go` contained **no reference to `Surface` at all**: `[s]` is forwarded to Observer from
the console, so every listener setting was editable while looking at a station.

## What it draws now

| Group | D-18 | Console |
|---|---|---|
| DATA — default location | row 1, **O** | **hidden** |
| DATA — provider key | row 3, **S** | shown |
| WATCHPOST UI — theme, units, clock | rows 19, 21, 22, **S** | shown |
| ALERTS - EVENTS | row 25, **O** (per D-20) | **whole group hidden** |
| ALERTS - TONE | rows 8, 9, **SPLIT** | shown |
| RADIO - CORRESPONDENTS | rows 6, 7, **S** (settled by D-11) | shown |
| RADIO - RELAY REPLAY | not persisted; the MONITOR's rotation pacing | **whole group hidden** |

**DATA is the one mixed group**, so it is half-drawn rather than skipped.

**RELAY REPLAY is not in D-18's table** because neither value is persisted — but the rotation it paces
is `advancesMonitor()`'s, which cannot advance at all while the console holds the air (D-74).  A pacing
control for something that cannot happen is a control that lies.

## The seam was already there

`rowVisible` existed, returned `true` for everything, and `stepRow` already walked only visible rows —
its own comment said *"a future rowVisible could"* hide them.  The build is one field on `setupRow`,
one predicate, and the group-level skip.

**The scope is a FIELD on the row, so the compiler enforces completeness**: positional struct literals
mean a new row cannot be added without declaring one.  The zero value is `scopeUnruled`, which the gate
rejects — but which still RENDERS, because a settings row that vanishes silently is worse than one that
appears where it should not: the first is invisible, the second is reportable.

## Four things the gates changed

**`dupes`** reported `firstVisibleOfGroup` and `groupHasAVisibleRow` as twins at 34 nodes — correctly:
one walk, two returns.  Collapsed into `visibleRowOfGroup` returning `(id, ok)`.  The bool is not
redundant with the id: `rowLocation` is both a real row and the zero value.

**`lint`** found `nextGroup`/`prevGroup` unused once tab moved to `stepGroup`.  Deleted rather than
kept — two ways to walk groups is the thing this codebase removes.

**`wires`** reported every `setupScope` member as NO WRITER.  `noteComposite` counts a member listed in
a table literal as a READ, deliberately — so a STATIC CLASSIFICATION never has a writer, because nothing
in production decides to produce one.  The sentinel `numSetupScopes` is what enrolled the set; the two
closest analogues in the same file (`setupRowKind`, `setupGroupID`) are static classifications with no
sentinel, so the sentinel went and the convention held.  **Not a dodge: the completeness property that
matters is `scopeUnruled`, and the gate asks for that directly.**

**The memo-completeness gate** caught a half-applied edit within seconds: the surface reached the key
struct and not `modalKeyFor`, and F-30's property reported it before any test of mine ran.

**And an existing mutant went stale** — `mM4` anchors on the relay row's literal, which gained a field.
Re-anchored; the rule it guards is unchanged.

## Not built

- **SPLIT storage.**  Tones render on both surfaces and share one value today.  D-18's own migration
  note describes the shim — *"a one-shot shim shaped like `withToneCompat` COPIES — never moves — the
  legacy value once"* — and it is additive.
- **The B rows.**  `broadcaster.transmitter`, `service_radius_mi` and `bed_radius_mi` are persisted and
  nothing writes them from the UI: that is **F-87**, and this is the mechanism it plugs into.
- **A conflict worth a ruling: D-18 row 29** recommends `broadcaster.gain_pct` as **B**, persisted,
  reasoning that "sharing them means an operator's on-air level changes because someone moved the
  listening volume".  The HUM LEAD ruled on 2026-09-12 that there is **one volume for the app**.  Row 29
  is unbuilt, so today matches the newer ruling — but the newer ruling was given against a survey that
  did not cite row 29, so it may not have been an informed override.  **Flagged, not resolved.**
