# Implementation plan — multi-voice-support (0.14.0)

*Edited in place. **PLAN approved 2026-08-30 — GO for BUILD.** Rulings applied through **MVS-D-46**: the eleven
PLAN-exit escalations were ruled as recommended in `08-reports/por-report.md` and recorded in the brief's
amendment log v1.4.0 (AM-21…34). Red-team rounds 1–4 remediated (`08-reports/red-team-plan.md` §12–§14).*

**What the rulings changed for BUILD:** the resident-Piper path is not built (MVS-D-29) · the whole plan ships,
with the scope cuts still available at the P2/P3 gates (MVS-D-30) · the tones are the *cue*, the cast is the
answer (MVS-D-31) · a fresh install still has one voice (MVS-D-32) · the storm class keeps the mock's label
"Maritime" (MVS-D-33, revisable before P4) · `[M]`'s change of meaning is accepted with a CHANGELOG line
(MVS-D-34) · RAT-3/5/6 ratified (MVS-D-35/36/37) · **AX-7 is (B) only — no zone geometry is built** (MVS-D-38) ·
M3/M2/M5/M4 and AX-4's second reserved slot ratified (MVS-D-39…43) · **background installs are capped at two per
session** (MVS-D-44) · the lint gate is `make verify`'s own steps, no new linter (MVS-D-45) · OP-1…5 as drawn
(MVS-D-46).

## 0. In plain words

- **What the listener gets.** Every alert opens with a tone that says what kind of alert it is; the station can
  have a *cast* — one voice for alerts, one for the reports, optional correspondents for the local weather, the
  sea, the fires and the quakes — who hand over to each other by name; coastal listeners get a maritime report.
- **What changes on screen.** Setup gains ALERTS - TONE and WATCHPOST RADIO - CORRESPONDENTS; `[V]` and `[T]`
  leave the Radio panel (`V` opens Setup at the correspondents); the panel takes one of three fixed layouts by
  width; `[S]` shows who speaks for whom and why; `[M]` mutes the tones only.
- **What stays the same.** A fresh install sounds as 0.13.0 does apart from the class tones; the broadcast never
  stalls (R6) — a cap bounds how many voices render at once.

```
Goal:         A listener away from the screen hears an alert before its words: every alert class has its own
              tone; the station has a cast — a voice per correspondent role with inheritance, assigned in Setup
              with a preview, hand-overs between correspondents, a per-class tone mute — and a full maritime
              report; the [V] and [T] controls retire and Setup and the Radio panel follow the HUM LEAD's mocks.
Architecture: domains/radio/cast (registry · inheritance · resolution · validation · classifier) ← config pairs;
              synth.Limiter around every Say; Segment.Role + a resolver in one Source with a voice-keyed,
              byte-bounded cache and a Source-time hand-over; the Station Director (the renamed arbiter)
              forwarding role and class; the tone as constants on a find-only alert path; MarineSegments + the
              CWF via FilterUGC; five parameterised presets; Setup's two new groups; the Radio panel per breakpoint.
Tech Stack:   Go 1.27 · bubbletea v2 · go-toml v2 · go-studs (vendored) · expect (PTY journeys) · Piper 2023.11.14
Branch:       feature/multi-voice-support  (0.14.0; SemVer minor)
Rulings:      MVS-D-1..46 (brief v1.4.0 incl. the amendment log AM-1..34; plan.md §1/§6; mocks/{radio-panel,setup}.md)
```

The plan is five batches (dependency-ordered, each UAT-able alone). Every task is **task shape** — file · symbol
· contract · test intent · verify command — and carries no implementation code: that is written at BUILD against
the compiler (`AP-PLANCODE-01`; `06-key_learnings/retro-notes.md` RN-1). Where a value is the HUM LEAD's (script
wording, the Setup rows), the task says so.

**Prior art.** The PLAN-phase code sketches are kept verbatim in `prior-art/`, with a README recording which of
them the round-4 red-team lenses actually **compiled and ran** (the `cast` package, the Limiter and the whole
`Source` seam ran green under `-race`; the `modes/tty` work never compiled) and nine known defects not to copy
forward. Use them **after** writing a task against the compiler, as a comparator — never as source.

## Legend (codes used in the batch files)

| Code | Meaning |
|---|---|
| `FR-n` / `NFR-n` | a requirement in `01-objectives/objectives.md` |
| `RS-n` | a risk in `02-analysis/risk-register.md` |
| `MVS-D-n` | a HUM LEAD ruling recorded in `08-reports/project-brief.md`'s amendment log |
| `AX-n` / `RAT-n` | an architecture axis / ratification item in `03-architecture-design/plan.md` |
| `P10-nn` | a rule of the harness's safety-critical code gate (`make p10`): 01 no recursion · 02 bounded loops · 04 ≤ 60 lines / ≤ 15 decisions · 05 invariant density (package average) · 06 no mutable package globals · 07 checked returns · 09 no `reflect`. Findings are dispositioned in the local ledger `.a2dh-p10-exemptions.yml` (untracked), ratified by the HUM LEAD at each gate |
| `UAT nn` | a numbered observation from an earlier release's user-acceptance log (the behaviour it pinned still holds) |
| `R6` | the radio's sacred rule: the broadcast never stalls or goes silent because of something else the app does |
| `OP-n` | an open point for the HUM LEAD, listed at the end of a batch file |

| Batch | File | Scope | Gate |
|---|---|---|---|
| P0 | §P0 below | the 0.13.0 time-to-tone-start and the Setup-open allocation pin recorded before any change | numbers in `07-readiness/perf-protocol.md`; `TestSetupAllocBudget` in the tree |
| P1 | `p1-foundation.md` | the P10 scope helper (Task 1.0 — the gate's own tooling, before the first run); `synth.Limiter` + bounded `Say` (fail-closed); `Compose(Reports{})`; the `cast` package (registry, bounded `Resolve`, `WantsInstall`, the M5 matrix, `Validate`, `Classify`/`ToneName`/`Muted`); config pairs/`cast`/tones + `Radio.Validate` (shapes) + save preserves unknown keys without `reflect` + fixtures; `castConfig`, the deck as `cast.Host`, the deck's limiter | `go test ./domains/radio/... ./platform/config ./app`; `make verify`; p10 with the predicted rows ratified |
| P2 | `p2-seams.md` | `Segment.Role` + tags; the Source under roles (voice-keyed byte-bounded cache, `renderedSeg` resolved once, soft/hard generations, Source-time hand-over as one `Say`, non-fatal `announce`, `spokenName`); the hand-over script part + built-in fallback; the tail script; the Station Director rename + `Run(role)` + `tone(class)`/`render(role)` + docs rows + ledger re-key; `resolveVoice` find-only, `ensureRoleVoices`, previews on the app ctx; the ticker's class tone + burst rule; the read's tone + mute rule; `setTones` | `go test ./domains/radio/... ./app -race`; `make verify`; `make pty-severe` |
| P3 | `p3-maritime-tones.md` | the `platform/render` marine lift; `MarineReport`/`MarineSegments` (forecast through `Segments`) + the word helpers; the eleven scripts; `Assembler.MarineFor` (shared `mergeMarine`) + `marineFor` + `Reports.Maritime` + the order; `Products.Marine` + `nws.MarineZoneFor` (capped, budgeted, centroids memoised) + `FilterUGC`; the five presets + `PresetByName` + `ToneRate` | script tests; `TestReportsAreSeparatedByAir` ×3; tone tests; the R6 soak on a coastal cycle; four declsets |
| P4 | `p4-ui.md` | the test scaffolding; `term.Merge` notes; `[V]`/`[T]` retired, `V` opens Setup at the cast; the theme chooser moved, the voice chooser deleted; the Radio panel per breakpoint (height-compact rule kept); Setup as a row table with the two new groups per the mock (two columns like Help, focus-following scroll, string-keyed `CastView`/`ToneClass`, pickers with `←→`/`p`, install/no-audio notes); the hooks (`SetCast`, `SetTones`, `ToneClasses`, `VoiceInstalled`) + `[M]` = tones only; the `[S]` cast table + config notes (+ `report --verbose`); help/README/CHANGELOG/`where-things-happen`; NFR-8 grep; the PTY journey (M2: 11 keypresses, pinned ≤ 11); the Setup alloc pin re-measured; goldens incl. three Setup frames | goldens; `make pty-severe` + the journey; alloc pins; `grep` zero; `make lint-imports` |

## P0 — measure first (perf-protocol.md §1–2)

**Task 0.1** — the tone path's code-path pin on the 0.13.0 tree (`perf-protocol.md` §1 item 1 is the one owner
of the method; this task only runs it). A worktree of the tag `v0.13.0` under the session scratchpad
(`git worktree add <scratchpad>/p0 v0.13.0`); copy in `app/tone_latency_test.go` exactly as P2 Task 2.9b lands it
in 0.14.0 (the file is written once, at 2.9b, and applied here first); `go test ./app -run '^$' -bench TimeToToneStart -count=10`;
median and p95 into `perf-protocol.md` §0. The Linux run of the same command is the HUM LEAD's
(`07-readiness/linux-validation-protocol.md` 1.1). End with `git worktree remove <scratchpad>/p0`. **Verify:** the row is filled.

**Task 0.2** — the Setup-open allocation numbers, **in the tree**: `modes/tty/bench_test.go` gains
`BenchmarkSetup_80x24` / `BenchmarkSetup_133x44` (Setup open, hit and miss, the `benchFrame` shape) and
`TestSetupAllocBudget` pinning today's window ×1.05 (`setupAllocBudget`, hit and miss; the probe's figures —
133×44 hit 3 046 / miss 3 476; 80×24 1 979 / 2 478 — are the expected order). Today's numbers are recorded in
`perf-protocol.md` §2 as the **informational** "before"; P4 Task 4.11 measures the new window and re-pins it
(spike-then-pin) — the pin is always a measurement, never a formula. **Verify:** `go test ./modes/tty -run SetupAllocBudget -v`

## File map (whole feature — regenerated from the batch maps)

```
CREATE  domains/radio/cast/{cast,resolve,validate,tone}.go + cast_test.go                    — P1
CREATE  domains/radio/synth/limit.go + limit_test.go                                          — P1
CREATE  app/tone_latency_test.go (BenchmarkTimeToToneStart — both trees)                      — P2 (2.9b); run at P0
CREATE  app/hostfacts.go + hostfacts_test.go (one cast.Host for the deck and the report)      — P4
MODIFY  domains/radio/synth/compose.go (Reports, Segment.Role, HandoffLine, tail key)         — P1/P2/P3
MODIFY  domains/radio/synth/source.go (resolver, cache key + byte bound, hand-over, Recast)   — P2
MODIFY  domains/radio/synth/{fire,seismic}.go (tagged)                                        — P2
MODIFY  domains/radio/synth/tone.go + tone_test.go (Preset, ToneRate, five presets)           — P2 (2.0) / P3 (3.6)
MODIFY  domains/radio/synth/{synth,seismic,soak}_test.go (pins, Recast tests, coastal soak)   — P2/P3
CREATE  domains/radio/synth/testdata/cycle.golden                                             — P2
CREATE  domains/radio/synth/marine.go + marine_test.go                                        — P3
MODIFY  domains/radio/synth/products.go (Marine), ugc.go (UGCCodes; FilterUGC param)          — P3
CREATE  domains/radio/script/scripts/handover/line.txt                                        — P2
CREATE  domains/radio/script/scripts/maritime-report/*.txt (11)                               — P3
MODIFY  domains/radio/script/scripts/weather-radio/tail.txt; script_test.go                   — P2/P3
CREATE  domains/weather/nws/marinezone.go + marinezone_test.go                                — P3
MODIFY  platform/config/config.go (RoleVoice, Voices, Tones, Radio, Validate, Save, Unknown)  — P1
CREATE  platform/config/keep.go + keep_test.go; testdata/*.toml (8 fixtures)                  — P1
CREATE  platform/render/marine.go + marine_test.go                                            — P3
MODIFY  platform/render/units.go (Glyphs.Down/Rail/Arrow), panel.go (rail glyphs), contrast.go — P4
MODIFY  platform/snapshot/harmonize.go (mergeMarine shared with MarineFor)                      — P3
MODIFY  scripts/quality/p10-unmatched.sh (+ p10-unmatched_test.sh) — Go-free-diff dormancy, rename scope — P1 (Task 1.0)
MODIFY  platform/snapshot/assembler.go (MarineFor, mergeMarine)                               — P3
MODIFY  platform/term/term.go + term_test.go (Merge notes)                                    — P4
CREATE  app/cast.go + cast_test.go (castConfig, castViewOf, saveCast, saveTones, parity)      — P1/P4
CREATE  app/marine.go + marine_test.go (marineFor, coastalForecast, the parity test)          — P3
RENAME  app/narrate.go → app/director.go (+ narrate_test.go → director_test.go)               — P2
MODIFY  app/{radio,voices,ticker,severe_read,dashboard,stats,dump}.go (+ tests)               — P1–P4
MODIFY  cmd/watchpost/root.go (+ report_test.go) — the cast block and tones line under --verbose — P4
MODIFY  docs/where-things-happen.md (rows at the P2 and P4 gates)                              — P2/P4
MODIFY  go.mod, go.sum, THIRD_PARTY_LICENSES.md — only if the go-toml spike reproduces (P1 1.9) — P1
MODIFY  modes/tty/{detail_marine,body}.go                                                       — P3/P4
CREATE  modes/tty/setup_rows.go, setup_cast.go, setup_tones.go (+ tests), setup_helpers_test.go — P4
CREATE  modes/tty/modal_theme.go (+ test)  ·  DELETE modes/tty/modal_chooser.go (+ test)      — P4
MODIFY  modes/tty/{setup,view,dashboard,memo,radio_panel,layout,help_about,status,body}.go    — P4 (nav.go is untouched: modalLines' Setup case lives in view.go)
MODIFY  modes/tty/bench_test.go (Setup pin), radio_panel_test.go, status_test.go, dashboard_test.go — P0/P4
MODIFY  modes/tty/testdata/*.golden (+ frame-setup-{80x24,133x44,133x44-ascii}.golden)         — P4
MODIFY  scripts/quality/validate-journey.expect (the cast step, counted)                       — P4
MODIFY  README.md, CHANGELOG.md, docs/where-things-happen.md, docs/extending.md, docs/img/*    — P2 (rows) / P4
CREATE  07-readiness/p10-p{1,2,3,4}.json; 04-development/p{1,2,3,4}-build-log.md               — each gate
        (declset pins re-captured: app P1–P4 · render P3 · snapshot P3 · nws P3 · tty P4)
```

## Requirement → task trace

| Requirement | Tasks |
|---|---|
| FR-1 roles + inheritance (incl. Station) | 1.3, 1.4, 2.1 (Station tags), 2.7 |
| FR-2 zero value = today | 1.2 (`Reports{}` test), 2.1 (the tail key), 2.5 (the tail script delta), 2.5b (the cycle golden), 3.6 (classic = zero preset) |
| FR-3 every read in its resolved voice | 2.2, 2.6, 2.7, 2.8 (M1 test), 2.9 |
| FR-4 assignment in Setup; 80×24; previews; no-audio; install notes | 4.4, 4.6, 4.7; 2.7 (previews on the app ctx, the Director's duck, the slot note) |
| FR-5 identity lines; hand-over never silence | 2.1 (`{{voice}}` one owner), 2.3 (non-fatal `announce`), 2.4 |
| FR-6 maritime report | 3.1–3.5 |
| FR-7 fallback explicit, never silent; display validation | 1.4 (matrix), 1.5, 2.7 (`Resolutions`, `render` records the reason), 4.8 (`[S]`, `report --verbose`) |
| FR-8 typed pair per role; validate at load; config-only pairs | 1.8, 1.10, 1.12; 4.7 (`displayName`/`storedName`) |
| FR-9 find-only alert path incl. the tone; background install | 2.7 |
| FR-10 on-air correspondent — not on the panel (MVS-D-24) | 2.7 (`onAir` kept for `[S]`), 2.8 (stream name), 4.8 |
| FR-11 per-class tones; per-class mute (empty = all); burst; storm precedence | 1.6, 2.7 (`tone` honours the mute), 2.9, 3.6, 4.5, 4.7 (`[M]`) |
| FR-12 bounded concurrency (one bound owner); Station Director | 1.1, 1.13, 2.3 (one `Say`), 2.6, 2.7 |
| FR-13 ascii/colour-off parity | 4.5 (`[ ]`/`[x]`), 4.6 (`Glyphs.Down/Rail`), 4.12 (the ASCII goldens) |
| FR-14 the `voice` action lives on; retired actions never break startup | 4.1, 4.2 |
| NFR-1 R6 | 3.7 (the soak); VALIDATE both platforms (`07-readiness/linux-validation-protocol.md`) |
| NFR-2 time-to-tone-start | 0.1; 2.7 (the tone from constants; the shipped debug line) |
| NFR-3 memory; the Setup pin | 0.2; 2.2 (`maxCachedBytes`); 4.11 |
| NFR-4 disk | perf-protocol §5 at UAT |
| NFR-5 no migration; unknown keys; save preserves | 1.8–1.11 |
| NFR-6 hostile text | 2.2 (`spokenName`), 3.2 (`stationWords` PlainLine + cap), 4.6 (`picker` PlainLine), 4.8 (`castRowOf`, notes), 4.1 (Merge notes) |
| NFR-8 no trace of `[V]`/`[T]` | 4.9 |
| by ruling MVS-D-23/24 — the Radio panel per breakpoint | 4.3 |
| M2 — the journey (AM-18, E-9) | 4.10 |
| FR-3/FR-4 — a save re-casts the deck | 2.10 |
| the test scaffolding every P4 RED test uses | 4.0 |

## Order of execution

P0 → P1 (1.0 … 1.14) → P2 (2.0 … 2.12) → P3 (3.1 … 3.7) → P4 (4.0 … 4.13; both mocks are in). Batch UAT: P2 alone
needs `[radio] cast = "cast"` in the file (the pairs are read only in cast mode, MVS-D-25); `[M]` persists through
`[radio.tones] mode` from P2 on (`Save` mirrors `ticker_muted`). Cross-batch notes:
P2 Task 2.0 lands `Preset`/`ToneRate`/`PresetByName`/`Classic()` in full and P3 Task 3.6 adds the other four
presets; P3 Task 3.4 changes `Compose`'s report order — `TestReportsAreSeparatedByAir` is rewritten there, not
in P1; the declset pins are re-captured at each batch's gate (the packages per batch are in the file map), never
mid-batch; the `where-things-happen` rows for the Director and the resolver land at the P2 gate (the docs test is
in `make verify`), the Setup/cast rows at P4. Every gate follows the batch-exit checklist (P1 Task 1.14): tests ·
`make verify` · `make p10` with the framework build and the predicted ledger rows presented for ratification ·
`p10-pN.json` with its `tree_hash` recorded in `07-readiness/gates.md` · declsets · the build log · a commit with
no attribution trailers.

## Pre-code spikes (recorded in `p1-build-log.md` before Task 1.1)

- go-toml v2 `omitempty` on a zero nested struct omits the table (probed by the plan review: yes) — the P1 fixture
  assertions depend on it.
- `toml.Decoder.DisallowUnknownFields` + `StrictMissingError` yield dotted key paths for every unknown key
  (Task 1.9's `unknownKeys`).
- `tea.KeyPressMsg{Code: 'V', Text: "V"}` reaches the `voice` binding without `Mod` (the existing tests' shape);
  special keys (`tea.KeyTab`, `tea.KeyUp`, …) are runes with no `Text`.
- The go-toml strict-decoder panic on an escaped quoted key: two probes disagreed — run `quoted-escape-key.toml`
  on the pinned v2.2.4 first; bump to v2.4.3 (+ `scripts/third-party-licenses.sh`) only if it reproduces (P1
  Task 1.9 keeps the scoped `recover` regardless).
- `git diff -M --name-status` reports the `narrate.go → director.go` move as `R0xx` — `p10-unmatched.sh` must map
  it before the P2 gate (P2 Task 2.6).
