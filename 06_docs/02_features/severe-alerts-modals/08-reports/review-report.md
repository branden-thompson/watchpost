# REVIEW report — severe-alerts-modals (0.13.0)

**Phase:** REVIEW → VALIDATE gate · **SEV-0 · HUMAN LEAD** · **Date:** 2026-08-29 · **Branch:**
`feature/severe-alerts-modals` · **Reviewed at:** `8183f5f` (BUILD exit `0123464` + the WX STN/ZIP nit) ·
**Remediation:** `e3a1d58` (A-08, A-16 c/e, the SEV-0 folders), `34dc40e` (round-5 fixes), this commit (docs)

## 1. Gates (Step 1)

Every gate in `07-readiness/gates.md` was re-run at this exit: `a2dh validate` 18/18 · `make verify` ALL GATES
GREEN · `make p10` 0 live / 0 unmatched (17 exempted findings, all listed for ratification) · `GOOS=linux go vet`
clean · allocation budgets within pins (80×24 memo-hit 964, severe window 2 031) · COV 11/11 · 13/13 · 11/11 ·
coverage globalfeed 92.5 % / severe 92.0 % / tty 91.6 % / render 85.9 % / app 53.0 % · `-race ×2` green on every
touched package · `make pty-severe` ok. Owed to VALIDATE with owners: the R6 relay + audio smoke (**blocking**,
both platforms), the 1-hour soak, the Linux halves, the live-feeds smoke, the README screenshots.

## 2. Requirements verification (Step 2 — reviewer A)

118 items traced from the objectives (FR-1..15, NFR-1..14, RS-1..17 mitigations, decisions, non-goals, every
UAT amendment) to the code and the test that pins each: **104 MET · 7 PARTIAL · 0 NOT MET · 7 SUPERSEDED**
(FR-15 waived at UAT; NFR-7, RS-11 superseded; three plan-era items). The PARTIALs: NFR-1 (no counter test —
now a boundary pin), FR-13 (the `--ascii` rulings — open), RS-8 (soak owed), RS-9 (fixed at REVIEW), RS-12
(the dark pairs outside the gate — ruling), the COV predicate and the "showing N of M" pin (fixed).

## 3. Stress and edge testing (Step 3 — reviewers B and C)

B (adversarial correctness, concurrency, lifecycle) and C (hostile input, accessibility, geometry, keys) drove
the whole 0.13.0 surface with throwaway probes: lock order and every `go` statement's owner; every message in
every modal state (34 keys × 18 states × 4 sequences — no double window, the read mark never touched);
0 / 1 / 500 rows and a vanished focus; the full escape corpus (every ESC/CSI/OSC/DCS/C0/C1 form) through the
severe cells, the record, the tape, the breaking line, the radio status, `[A]`; 10 000 hostile features;
widths 20–240 × heights 20–100 for the dashboard, the window at 60–240 × 20–100 with 0/1/13/500 rows, Help
and Status in both layouts; colour off, `--ascii`, a light background. What held is listed in their reports
("verified correct"); what did not is §5.

## 4. Documentation review (Step 4 — reviewer D)

The README was audited section by section against the tree; the CHANGELOG against `git log main..HEAD`; the flow
map's every `path:Symbol` (test-pinned) and its prose; Help against the key map; 25 doc comments beside their
code; the feature's eight folders; commit hygiene (no attribution, no probe files, `gofmt`/`vet`/`tidy` clean).
Code-facing docs were sound; the README's Setup prose, the missing `M` row, the CHANGELOG's gaps and a
13-vs-14 contradiction are fixed in this commit; the screenshots are the HUM LEAD's capture (A4).

## 5. Critical analysis — round 5 findings and dispositions

Four reviewers, 60 findings: **A 10 · B 10 · C 15 · D 25**. Every finding was verified before acceptance;
every reproduced defect's probe is a test in the tree. Disposition codes as in the BUILD reports.

### A — requirements (verdict SHIP-WITH-CONDITIONS)

| ID | Sev | Finding | Disposition |
|---|---|---|---|
| R5-A-01 | Medium | NFR-1's httpx round-trip counter test never existed | **Fixed** as a structural pin: `TestSevereImportsNoNetwork` (the index imports no client); the counter form is superseded |
| R5-A-02 | Medium | FR-13 `--ascii` header form and the `>` mark collision still unruled (B-08a/b) | **HUM LEAD ruling** (release checklist A1) |
| R5-A-03 | Low | COV predicate pinned `MoveDirDeg` on `MoveSpeedKt`; `Effective` never renders | **Fixed** (predicate); **Accepted** — `Effective` is the Declared fallback |
| R5-A-04 | Low | `close()` did not reset `severeDetail` | **Fixed** |
| R5-A-05 | Low | "showing N of M" unpinned | **Fixed**: `TestSevereRendererClampsTheFocusAndSaysShowing` |
| R5-A-06 | Low | AA gate missed the column-title dip and marks/chips on tints | **Fixed** for the dip (`TestTableHeaderDipReadsAAInEveryTheme`); the rest → R5-C-04 |
| R5-A-07 | Info | RS-11 a no-op at blend 1.0 | **Fixed**: marked SUPERSEDED in the objectives |
| R5-A-08 | Info | `[A]`/module paging does not apply the superseded guard | **Deferred** (follow-ups): `severe.Guard` at the snapshot seams |
| R5-A-09 | Info | data-shape "enum-validate" overstated | **Fixed**: re-worded |
| R5-A-10 | Info | Owed VALIDATE items | carried (`gates.md`) |

### B — adversarial code (verdict SHIP-WITH-CONDITIONS)

| ID | Sev | Finding | Disposition |
|---|---|---|---|
| R5-B-01 | **High** | The severe deck and the tty shared one snapshot pointer; the tty sorts alerts in place, the deck re-reads on the ticker goroutine — a `-race` report on the main data path (introduced by the round-3 A-02 fix) | **Fixed**: the deck clones the locations and their alerts at `SetLocations`; `TestDeckKeepsItsOwnCopyOfTheLocations` (the race probe) |
| R5-B-02 | Medium | A cancelled job never woke the cond var; `settle` resumed a cancelled suspended read | **Fixed**: `context.AfterFunc` broadcasts; `settle` discards cancelled suspended jobs; `TestReadCancelledWhileParkedSuspendedReturnsAtOnce` |
| R5-B-03 | Medium | The `[V]` voice preview took the flight slot: a pause held the sample, the read kept playing under the takeover | **Fixed**: `Engine.Audition` (never the line in flight); `TestAuditionIsNeverTheLineInFlight` |
| R5-B-04 | Medium | No-audio build: `[space]` marked ▶ and held the reader busy for minutes of silence | **Fixed**: the hook is nil without a deck (the chip mutes); `TestNarrateEventIsNilWithoutADeck` |
| R5-B-05 | Low | Ids clamped to 120 runes before normalising — the URL and bare forms of a long OID disagreed | **Fixed**: `clampID` (200) on both paths; pinned |
| R5-B-06 | Low | `cap1` sliced bytes | **Fixed**: by rune; pinned |
| R5-B-07 | Low | The breaking goroutine sits outside the shutdown wait set (a render in flight may outlive teardown) | **Deferred** (follow-ups): derive the render's context from the job's; a done channel for `stop` |
| R5-B-08 | Low/Nit | esc + arrows/backspace not un-fused; alt overrides could never fire | **Fixed** with C-08 (every alt chord is esc then the key; no binding uses alt — asserted) |
| R5-B-09 | Nit | `lookupRef` survived a failed commit / another window | **Fixed**: `open()` clears it for any window but Details |
| R5-B-10 | Nit | One `-race` failure of `modes/tty` with reviewer C's probes present | **Accepted**: 6/6 and ×2 green on the clean tree |

### C — hostile input, accessibility, geometry (verdict SHIP-WITH-CONDITIONS)

| ID | Sev | Finding | Disposition |
|---|---|---|---|
| R5-C-01 | Medium | The tape windowed by rune (240 cells on 133 cols for wide runes); a feed `\n` added rows to the frame | **Fixed**: `tapeText` one line; `scrollWindow` by cells; `TestTapeTextIsOneLine` |
| R5-C-02 | Medium | Lookup with an empty RECENT: a priority publish reset the focus to the first favourite; `lookupRef` never cleared | **Fixed**: the focus is kept while a lookup waits; the rebuilt list is searched by identity; `TestLookupWithEmptyRecentSurvivesAPriorityPublish` |
| R5-C-03 | Medium (A11y) | On a light terminal every window-painted token fails AA in every theme (`WindowBGLight` is a placeholder) | **HUM LEAD ruling** (release checklist A2): support light terminals with a light token set, or state them unsupported |
| R5-C-04 | Low-Med (A11y) | Dark pairs outside the gate: `GroupText` on the bands (4 themes), `StateStopped` on the window (9), `NameWarning`/`ProviderDown` (2), the green ▶ on two tints, `TickerFG` on the yellow lane, `AlertDanger` on the modal | **HUM LEAD colour pass** — the numbers are in C's report; `LiftToAA` at registration is the one place to widen |
| R5-C-05 | Medium | NFR-6 not global: Details forwarded the station name, buoy id and tide/current station names raw; config labels raw in titles | **Fixed**: labels cleaned once at the assembler (`platform/plaintext`, the leaf both render and snapshot use); the Details sites through `PlainLine`; the fire field C could not locate is not in the tree |
| R5-C-06 | Low-Med | The 80×24 floor holds only with ≤ 1 favourite (32 lines with 10) | **Accepted + documented**: the floor is 22 rows + one per favourite (README); windowing the favourites table is a design change for a later release |
| R5-C-07 | Low | `severeBrowseLines` trusted a focus past the tab | **Fixed**: clamped; pinned |
| R5-C-08 | Low | esc + arrow/backspace/non-ASCII fused and lost | **Fixed**; `TestLoneEscThenKeyIsNotLost` extended |
| R5-C-09 | Low | Alerts shrinking under an open page: raw `alertIdx` in the chips | **Fixed**: the page as shown; pinned |
| R5-C-10 | Low | `truncateTo` cut runes: a wide-rune event overflowed the compact row | **Fixed**: `render.TruncateCells`; pinned |
| R5-C-11 | Low | A long label overflowed the Details / `[A]` titles | **Fixed**: `PanelColored` clamps its title; pinned |
| R5-C-12 | Low | One malformed JSON value silences a whole source | **Deferred** (follow-ups): decode features as `[]json.RawMessage` and skip the bad one |
| R5-C-13 | Low (A11y) | `—`/`·` and surrounding surfaces keep non-ASCII marks under `--ascii`; muted chips indistinguishable with `--ascii` + colour on | **Deferred** to the FR-13 ruling's scope (A1) |
| R5-C-14 | Low | Bidi controls, zero-width and unbounded combining runes survived `Plain` | **Fixed** for bidi/zero-width; the combining cap **Deferred** |
| R5-C-15 | Low | A 20 MB override part became a 20 MB spoken line | **Fixed**: 64 KB bound; pinned |

### D — documentation (verdict SHIP-WITH-CONDITIONS)

| ID | Sev | Finding | Disposition |
|---|---|---|---|
| R5-D-01 | High | All six README screenshots are 0.12.0 captures; no window shot | **HUM LEAD** capture before SHIP (A4) — the images are yours to take, as for 0.12.0 |
| R5-D-02 / 03 / 04 / 05 / 06 / 07 / 09 | Medium–Low | Setup prose (two-question model); `M` missing; "0.9.0" stamp; intro silent on the window; four of five pronounce tables; the verify list; the `S` gauge | **Fixed** (all) |
| R5-D-08 | Nit | `v0.9.0` example; "Go 1.25" floor vs CI 1.27 | **Declined** (optional; the floor is true) |
| R5-D-10 / 11 / 12 | Medium–Nit | CHANGELOG: the column-title row, the AA lift and a `### Fixed` section missing; "lighter tint"; "opens the window" reads as auto-open | **Fixed** |
| R5-D-13 | Nit | "Unreleased" date | at SHIP |
| R5-D-14 / 15 / 16 | Low–Nit | flow map "only package" (tests import the kit); no Lookup / diagnostic rows; `helpGap` comment | **Fixed** |
| R5-D-17 / 18 / 19 / 20 / 21 / 22 | Medium–Nit | "13 rows" vs 14; round-3 budget figures unlabelled; the ledger not re-run; the `Interrupt` claim; A-08's closure unrecorded; the round-4 tally | **Fixed** (the ledger rewritten at this exit) |
| R5-D-23 | Low | No risk document in 02, no release checklist / Linux protocol in 07 | **Fixed**: `02-analysis/risk-register.md`, `07-readiness/release-checklist.md`, `linux-validation-protocol.md` |
| R5-D-24 | Info | The bare harness CLI name in six tracked files | **Accepted** — the CLI's name is the gate command, not an employer or internal-product name; HUM LEAD to confirm |
| R5-D-25 | Info | A UAT commit after the recorded BUILD approval | **Fixed**: recorded (build-report note, objectives) |

## 6. Rulings requested (HUM LEAD)

1. **`--ascii` forms** (A1): keep the spread `E V E N T` (the ratified mock) and amend FR-13, or plain `EVENT`;
   give the play mark its own ASCII form (`*`) or keep `>`.
2. **Light terminals** (A2): support them (a light token set, the AA gate over `WindowBGLight`) or document
   "dark backgrounds" as a requirement. Today every text token fails AA on `#ECECEC` in every theme.
3. **The dark-theme pairs outside the AA gate** (R5-C-04): your colour pass, or widen `withAA` to those pairs.
4. **Ratify the 17 P10 ledger rows** (`gates.md`, incl. the new `platform/plaintext` row).
5. **Screenshots** (A4): dashboard, the window, Details on the 0.13.0 UI.
6. R5-D-24: confirm the CLI's bare name may stay in the six tracked files.

## 6a. Rulings given (HUM LEAD 2026-08-29) and applied — `3285462`, `74aed75`; Watchpost Light corrected on the HUM LEAD's screenshot at `df59528` (a valid frame base — `render.FgSGR` — and pale grounds throughout; the same fix lifted every theme's contrast)

1. plain `EVENT` under `--ascii`; the play mark `*`. 2. Watchpost Light, the one light theme; every theme paints
its own ground. 3. `withAA` widened to every painted pair (`aaPairs`). 4. The rows ratified (17 at the exit); the four reasons
refreshed. 5. Screenshots after the 0.13.0 stamp, before the public release. 6. The CLI's bare name stays as the
gate command; prose names the gate.

## 7. Follow-ups — all closed this phase

R5-A-08 (`dropSuperseded` at the publishers; `TestDropSupersededKeepsOnlyTheReplacement`) · R5-B-07 (the takeover
goroutine in `stop`'s wait set; `render(ctx, …)` bound to the job; `TestTickerStopWaitsForATakeoverInFlight`) ·
R5-C-12 (entry-by-entry decoding on USGS/NWS/NHC; `TestOneBadEntryDoesNotSilenceASource`) · R5-C-13 (the ASCII
sweep through `Glyphs`) · R5-C-14 (the combining cap) · the dead `table.header` token · the detail-modal memo (AX-4
already memoises Details — resolved) · the 0.12.0 ticker stage-2 duck (the arbiter ducks both paths — resolved).
Open by design: the multi-alert circle visualisation (awaits a HUM LEAD mock).

## 8. Verdict

**PROCEED to VALIDATE — conditions met, rulings applied, follow-ups closed.** The four reviewers converged on SHIP-WITH-CONDITIONS; every
condition that is code is closed and pinned (one High — the shared snapshot race — and eleven Mediums), the
documentation is true to the tree again, and the SEV-0 document set is complete. What remains is the
HUM LEAD's: six rulings above, and the VALIDATE gates the ledger owes — the R6 radio smoke on both platforms
foremost, blocking.
