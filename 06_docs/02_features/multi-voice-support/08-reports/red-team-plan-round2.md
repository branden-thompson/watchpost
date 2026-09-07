# Red-team — PLAN exit, round 2 (fresh lenses on `3fcd822`) — the findings, one row per ID

Every round-2 finding as the lens stated it (condensed to the claim and the action), with its disposition; §12.1 of
`red-team-plan.md` is the summary, this file is the audit trail (the 0.13.0 precedent). Dispositions are the
round-2 remediation's (`1d87831`) unless a round-3 row (`red-team-plan.md` §13) re-opened them — those are marked.

## Business Quality — BQ2 — round-2 verdict: ESCALATE (BQ2-1/2/3 driving)

| ID | Finding (as stated) → action | Disposition |
|---|---|---|
| BQ2-1 | M4 baseline unproducible: no fixture injects a takeover; no `breaking` debugLog entry exists (radio.go:277,535,543 only); PreviewAside debug line unspecified in 2.7; Linux release binary has neither → unit-level timing (Director → engine fake, time.Since) on both trees, or a debug-only injection hook; amend 0.1, Linux 1.1, perf §1 (drop "or a live one") | Fixed in `1d87831` (see §13 for re-openings) |
| BQ2-2 | M3 contradictory: brief's anti-solution clause "audio starts after the tone" + fresh-install arm "tones only" = chance by construction; 0.13.0 already ~10/10 with tone kept; no comparative pass rule → two trials: fresh arm "name the class from the tone" (0.14.0 vs 0.13.0 single tone); cast arm "alert vs report with tone stripped"; pass = ≥9/10 AND > 0.13.0; log AM-17 | Fixed in `1d87831` (see §13 for re-openings) |
| BQ2-3 | E-2 split option incoherent (pickers ARE P4; MVS-D-3 couples [V] retirement + panel); only clean cut is maritime 3.1–3.5 → restate E-2 as (a) whole (b) minus maritime (c) minus per-class mute (re-rules MVS-D-26); UI-less split needs re-ruling MVS-D-3; give task counts (P1 13 · P2 13 · P3 7 · P4 14) | Fixed in `1d87831` (see §13 for re-openings) |
| BQ2-4 | phase-B ceiling has no number; app RSS excludes piper processes → app RSS ≤ 115 MB×1.10 no trend; summed piper RSS during overlap ≤ 2× §3 number; name ps columns; Linux 2.11 say which RSS | Fixed in `1d87831` (see §13 for re-openings) |
| BQ2-5 | RAT-3/5/6 "not objected — proceeding" at SEV-0 → E-7 explicit rulings | ESCALATE — **E-7** raised in `1d87831`; open until the HUM LEAD rules (not "Fixed": an escalation is closed by a ruling, not by being written down) |
| BQ2-6 | AX-7 (A) exceeds "for free" (new NWS endpoint) → ESCALATE with (B)-only recommendation | ESCALATE — **E-8** raised in `1d87831` with the (B)-only recommendation; open until ruled |
| BQ2-7 | E-6 accept-or-nothing → add options: map legacy true → mute tones+words for one release with [S] note; one-time in-app note | ESCALATE — **E-6** options added in `1d87831` ((a) accept + CHANGELOG, (b) map legacy, (c) one-time note); open until ruled |
| BQ2-8 | M2 bound set after path known; brief row still "≤ 5 actions" → log honestly (≤ 11 pin, AM row) or direct-Save chord | Fixed in `1d87831` (see §13 for re-openings) |
| BQ2-9 | brief M5 "0 silent reads" vs FR-7's legitimately silent row → AM row | Fixed in `1d87831` (see §13 for re-openings) |
| BQ2-10 | Tasks 4.3 and 2.10 untraced → "by ruling MVS-D-3/23" rows | Fixed in `1d87831` (see §13 for re-openings) |

**Round-1 rows re-verified by this lens:** BQ-1 Partial(2-2) · -2/-3/-6/-7/-8/-13 ESCALATE open · -4 Verified (p2:7 stale) · -5 Verified · -9 Verified · -10 Verified(2-8) · -11 Partial · -12 NOT FIXED (2-1) · -14 Verified · -15 Partial(2-1)

## Code Quality — CQ2 — round-2 verdict: NO-GO

| ID | Finding (as stated) → action | Disposition |
|---|---|---|
| CQ2-1 | Recast double render (= PERF2-1/PA2-1) → voiceFor | Fixed in `1d87831` (see §13 for re-openings) |
| CQ2-2 | CastMode:"cast" in 2.7 tests + UAT note | Fixed in `1d87831` (see §13 for re-openings) |
| CQ2-3 | voice(): cfg.Voice=="" off darwin → error; fall back to synth.DefaultVoice().Key (UAT 118) | Fixed in `1d87831` (see §13 for re-openings) |
| CQ2-4 | cast.Validate never called → wire in recast/attachRadio → ConfigNotes; drop stale P1 RED assertion | Fixed in `1d87831` (see §13 for re-openings) |
| CQ2-5 | Names nil map panic; cycleVoice("",→) returns list[0] but tests expect Rishi | Fixed in `1d87831` (see §13 for re-openings) |
| CQ2-6 | tests assert Say order impossible under render-ahead → membership | Fixed in `1d87831` (see §13 for re-openings) |
| CQ2-7 | compile: invariant import; e.Key(); brokenVoice collides with synth_test.go:497; Segments(string); slices/maps imports; 2.1 RED builds Reports{Voice:}; 1.8 unused slices; tone.go const block gofmt | Fixed in `1d87831` (see §13 for re-openings) |
| CQ2-8 | phantoms: Class.Key(), synth.VoiceByKey, render.TextDim, cast.AllClassKeys, three installed-check names; castKeyAll "all" vs Key() "voice"; parity 7 vs 9 | Fixed in `1d87831` (see §13 for re-openings) |
| CQ2-9 | mark spacing "› " vs mock "›  ○"; tests expect › on every class row | Fixed in `1d87831` (see §13 for re-openings) |
| CQ2-10 | castGen dead; castRows not reset on listVoices | Fixed in `1d87831` (see §13 for re-openings) |
| CQ2-11 | render's resGen==gen store guard dead → store unconditionally | Fixed in `1d87831` (see §13 for re-openings) |
| CQ2-12 | 2.7 render keeps 2-min wrap; voices.go:142 is PreviewVoice's timeout; 1 s acquire note unobservable | Fixed in `1d87831` (see §13 for re-openings) |
| CQ2-13 | stale refs: plan §2.4/P3 3.6 d.tones.Get; §2.1/§3.3 Reports{VoiceFor}; §1.3 60 s/20 s vs 90 s; §2.6 MarineZoneFor(ctx,lat,lon,office)/"≤8"; §2.2 ToneName(class,tones); cast.Validate doc "Piper mode"; toneClass comment "quake is a warning"; UGCCodes "in order" then sort | Fixed in `1d87831` (see §13 for re-openings) |
| CQ2-14 | ticker_muted mirror in Save AND saveTones → keep Save's | Fixed in `1d87831` (see §13 for re-openings) |
| CQ2-15 | Link.String/ToneName unguarded; Validate message from Classes() | Fixed in `1d87831` (see §13 for re-openings) |
| CQ2-16 | setupPlan 3–4 builds/key | Fixed in `1d87831` (see §13 for re-openings) |
| CQ2-17 | handoff func executed under s.mu → copy out | Fixed in `1d87831` (see §13 for re-openings) |
| CQ2-18 | TestDeckVoicesAreLimited tautology; pressSetup(t,'V') misnamed | Fixed in `1d87831` (see §13 for re-openings) |

**Round-1 rows re-verified by this lens:** CQ-1 compile err · -2 Partial(2-4) · -3 Partial(2-8) · -4 Partial(2-8) · -8 Partial(2-1) · -10 Partial(2-1) · -12 Partial(2-8) · -14 Partial(2-13) · -15 Partial(2-7f) · -18 Partial(2-7c,2-12) · -19 Partial: mergeMarine only a comment; MarineFor re-implements harmonizeMarine loop · rest Verified

## Docs Quality — DQ2 — round-2 verdict: SHIP-WITH-CONDITIONS (0 High / 4 Important)

| ID | Finding (as stated) → action | Disposition |
|---|---|---|
| DQ2-1 | resident cut half-propagated (plan.md:44,72-78 live knob sentence; perf-protocol §3 step 6; p2 Goal block; RS-23 Open) | Fixed in `1d87831` (see §13 for re-openings) |
| DQ2-2 | preview-install: FR-4/NFR-4/§3.5 "ask once" vs P4 "downloads on Save" — rule once (recommend P4's; guard p with VoiceInstalled) | Fixed in `1d87831` (see §13 for re-openings) |
| DQ2-3 | cache bound voice-scaled in NFR-3/perf §4/RS-6 vs flat 40 MB in P2 | Fixed in `1d87831` (see §13 for re-openings) |
| DQ2-4 | objectives §0 "chosen by you from five presets" false; "Radio / Correspondents" old group name | Fixed in `1d87831` (see §13 for re-openings) |
| DQ2-5 | plan.md §0 "exactly as 0.13.0 (classic tone)" self-contradictory | Fixed in `1d87831` (see §13 for re-openings) |
| DQ2-6 | ToneName(class, tones) at plan.md:113 | Fixed in `1d87831` (see §13 for re-openings) |
| DQ2-7 | plan.md §2.1 stale file map (drop names, cite index) | Fixed in `1d87831` (see §13 for re-openings) |
| DQ2-8 | plan.md:32 objectives v1.1.0 → v1.2.0 | Fixed in `1d87831` (see §13 for re-openings) |
| DQ2-9 | two classifier rules of record (plan.md:190 data-shape §4b vs tones.md §2) | Fixed in `1d87831` (see §13 for re-openings) |
| DQ2-10 | NFR-8 grep pattern triplicated (objectives:62, p4:11, p4:1388) | Fixed in `1d87831` (see §13 for re-openings) |
| DQ2-11 | declset list: P4 modifies render+term → add render P4 · term P4 in gates.md | Fixed in `1d87831` (see §13 for re-openings) |
| DQ2-12 | piper-and-platform.md:163 "and the chip" | Fixed in `1d87831` (see §13 for re-openings) |
| DQ2-13 | setup.md:62 recommends cast="single" (Validate rejects) → append ruled values at :59 | Fixed in `1d87831` (see §13 for re-openings) |
| DQ2-14 | brief amendment log: v1.3.0 PLAN rulings heading; AM-12 order | Fixed in `1d87831` (see §13 for re-openings) |
| DQ2-15 | P4 file map line 60 omits Glyphs.Rail; OP order | Fixed in `1d87831` (see §13 for re-openings) |

**Round-1 rows re-verified by this lens:** all Verified except DQ-4 Partial(2-12), DQ-13 Partial(2-15), DQ-16 content wrong (2-4/2-5), DQ-21 regressed (2-1), DQ-22 regressed (2-9)

## Project Hygiene — PH2 — round-2 verdict: SHIP-WITH-CONDITIONS (3 Important)

| ID | Finding (as stated) → action | Disposition |
|---|---|---|
| PH2-1 | P0 "PTY smoke's fixture event" does not exist (severe-modal.expect opens empty window; 0.13.0 protocol used a live event or M-unmute with one pending) → name real method or add a debug-only fixture hook (Task 0.1 + 2.7; 4 docs) | Fixed in `1d87831` (see §13 for re-openings) |
| PH2-2 | literal employer path (a `$HOME/…/dist/a2dh` under an employer-named tree) in gates.md:5, p1:1765, p2:1769, p3:1064, p4:1432 → `A2DH=<framework build>`; add grep to release-checklist:20 | Fixed in `1d87831` (see §13 for re-openings) |
| PH2-3 | p4:1435 still lists ./platform/term; render P4 unnecessary → `go test ./modes/tty ./app -run DeclarationSet -update-declset`; gates.md owns list | Fixed in `1d87831` (see §13 for re-openings) |
| PH2-4 | missing imports: resolve.go/validate.go invariant (p1:704-707,765,905-908,923), marinezone.go encoding/json (p3:823-836), setup_tones slices (p4:797-803,912), setup_cast maps (p4:1039-1045,1203); toneClasses c.Key() → c.String() (p4:1277 vs p1:571-586) | Fixed in `1d87831` (see §13 for re-openings) |
| PH2-5 | WATCHPOST_DEBUG_RADIO is a path not bool (impl:60, perf:20, linux:11) | Fixed in `1d87831` (see §13 for re-openings) |
| PH2-6 | Setup goldens need TestSetupGoldens (golden_test.go:35-52); severe goldens also re-record | Fixed in `1d87831` (see §13 for re-openings) |
| PH2-7 | perf-protocol.md:40-42 "adopt resident in 0.14.0" | Fixed in `1d87831` (see §13 for re-openings) |
| PH2-8 | linux 3.1 make p10 on Arch (no framework build) → drop or say how | Fixed in `1d87831` (see §13 for re-openings) |
| PH2-9 | gates vs checklists drift: -count=2 vs 1; relative cp path in gates.md:13; synth cycle golden -update flag → -update-golden and add to Goldens gate | Fixed in `1d87831` (see §13 for re-openings) |
| PH2-10 | release steps: commit-tree prints hash (git branch release/v0.14.0 $(...)); main-publish has no upstream (git pull origin main) | Fixed in `1d87831` (see §13 for re-openings) |
| PH2-11 | stray worktree rt-perf/wt at 426eb1e → git worktree remove/prune; checklist rule | Fixed in `1d87831` (see §13 for re-openings) |
| PH2-12 | `_ = tones` (p4:1250); toneClassCount=6 + focusToneClass0..5 duplicate the class count → assert len(st.classes)==toneClassCount at openSetup | Fixed in `1d87831` (see §13 for re-openings) |

**Round-1 rows re-verified by this lens:** PH-3 Partial(2-3) · PH-9 Partial: VoiceName not deleted (p4 4.2 delete list; p2:1526; app/voices.go:161-180) · PH-14 Partial: whole map omits modes/tty/detail_marine.go; P4 map omits contrast.go, body.go, docs/extending.md, modal_theme_test.go, app/cast_test.go · rest Verified

## Principal Architect — PA2 — round-2 verdict: NO-GO (PA2-1, PA2-2) → SHIP-WITH-CONDITIONS once both are rewritten in P2

| ID | Finding (as stated) → action | Disposition |
|---|---|---|
| PA2-1 | hand-over is a blocking uncached Say on the writer (announce before first write; Recast renders whole segment then line+rest) → render hand-over in renderLoop: renderedSeg.handoff keyed cacheKey(to,"handoff:"+from); play only writes; Recast check uses voiceFor (resolve, no render); keep remainder rendering parallel with the line as 0.13.0; test Say count per Recast = 1 and no pipe gap > inter-segment gap | Fixed in `1d87831` (see §13 for re-openings) |
| PA2-2 | Director starvation: narrateRead has no priority; on Linux N=2 render-ahead + hand-over + preview hold all ordinary slots; read waits ≤90 s AFTER admit ducked → Director tier outranks the Source (two-tier limiter or reserved=2 with WithPriority for both classes); state in §1.3 | Fixed in `1d87831` (see §13 for re-openings) |
| PA2-3 | listVoices landing bumps no castGen/castRows/Invalidate → one d.castChanged() helper used by install landing + discovery; test | Fixed in `1d87831` (see §13 for re-openings) |
| PA2-4 | [M] persistence broken P2–P3: Save writes TickerMuted from Tones.Mode AFTER tickerMuteState's savePreference(TickerMuted) → 2.10 hook writes cfg.Radio.Tones.Mode + setTones; delete TickerMuted write from tickerMuteState; cast.AllClassKeys() doesn't exist (Muted:nil = all) | Fixed in `1d87831` (see §13 for re-openings) |
| PA2-5 | setupFocusRow misses second-cell class ids; questionRows yields 4 not 5 → []setupFocus; key as tab stop by table flag; test walks all 19 ids at 80×24 asserting on-screen | Fixed in `1d87831` (see §13 for re-openings) |
| PA2-6 | openSetupAt never scrolls (V opens below the fold) → end with scrollSetupToFocus; extend test | Fixed in `1d87831` (see §13 for re-openings) |
| PA2-7 | tone/cast rows call setupFinishCmd("") → FIRMS key dropped, ref==nil fallback lost → one setupSave() owner | Fixed in `1d87831` (see §13 for re-openings) |
| PA2-8 | group pickers can't express inherit → "(inherits …)" entry or checkbox | Fixed in `1d87831` (see §13 for re-openings) |
| PA2-9 | breakpoint off-by-one: 84 cols → inner 69 → narrow; state breakpoints in outer cols via one conversion; pin ±1 | Fixed in `1d87831` (see §13 for re-openings) |
| PA2-10 | phantom symbols: cast.AllClassKeys, Class.Key(), d.installedPiper, synth.VoiceByKey (VoiceByName accepts keys), shortFor "(P2)", d.tones.Get, shipped PreviewAside debug line absent from 2.7 code, slices import → reconciliation pass | Fixed in `1d87831` (see §13 for re-openings) |
| PA2-11 | install retry storm (= SEC2-2) | Fixed in `1d87831` (see §13 for re-openings) |
| PA2-12 | plan.md §0 row 0.3, AX-1 row, §1.1 :74-78, p2 header still ship resident → rewrite | Fixed in `1d87831` (see §13 for re-openings) |
| PA2-13 | setupFocusRow indexes unwrapped body vs wrapModal → cap every Setup line to inner width | Fixed in `1d87831` (see §13 for re-openings) |
| PA2-14 | radioLines inside layoutWith runs twice per frame → keep layoutRows{full, compact} pre-build | Fixed in `1d87831` (see §13 for re-openings) |
| PA2-15 | keepUnknown drops unknown keys inside [[recent]] → document | Fixed in `1d87831` (see §13 for re-openings) |

**Round-1 rows re-verified by this lens:** PA-1 Partial(2-12) · -3 Verified but 2-1 · -6 Partial(2-9) · -8 Partial: "[S] notes unparsable old file" has no task · -10 Partial(2-12) · rest Verified/moot

## Accessibility — A11Y2 — round-2 verdict: SHIP-WITH-CONDITIONS

| ID | Finding (as stated) → action | Disposition |
|---|---|---|
| A11Y2-1 | chips scrolled off (pin footer; OP) | Fixed in `1d87831` (see §13 for re-openings) |
| A11Y2-2 | cast notes off-screen (draw under focused row; keep focusRow+1) | Fixed in `1d87831` (see §13 for re-openings) |
| A11Y2-3 | tab stops 4 not 5 (groupKey; ↑↓ in DATA) | Fixed in `1d87831` (see §13 for re-openings) |
| A11Y2-4 | second-cell tone focus index -1 (tag []setupFocus / normalise) | Fixed in `1d87831` (see §13 for re-openings) |
| A11Y2-5 | ↑↓ Pick false; EVENTS toggles vs TONE moves (own focus id for Within; OP-4) | Fixed in `1d87831` (see §13 for re-openings) |
| A11Y2-6 | [S] Reason never printed; → literal | Fixed in `1d87831` (see §13 for re-openings) |
| A11Y2-7 | report --verbose has no deck; the dump is a different surface (name the real seam) | Fixed in `1d87831` (see §13 for re-openings) |
| A11Y2-8 | [M] label with partial set; [M] from On restores old set | Fixed in `1d87831` (see §13 for re-openings) |
| A11Y2-9 | TextDim token does not exist | Fixed in `1d87831` (see §13 for re-openings) |
| A11Y2-10 | › ⚠ ✔ literals (Glyphs) | Fixed in `1d87831` (see §13 for re-openings) |
| A11Y2-11 | rail literal (RailGlyphs) | Fixed in `1d87831` (see §13 for re-openings) |
| A11Y2-12 | empty voice list no note | Fixed in `1d87831` (see §13 for re-openings) |
| A11Y2-13 | preview downloads without asking (press p again) | Fixed in `1d87831` (see §13 for re-openings) |
| A11Y2-14 | stale setupNote | Fixed in `1d87831` (see §13 for re-openings) |
| A11Y2-15 | Save only on row 19 (ctrl+s chip; OP-4) | Fixed in `1d87831` (see §13 for re-openings) |

**Round-1 rows re-verified by this lens:** A11Y-1 Partial(2-1) · -2 Not fixed(2-7) · -3 Partial(2-8,2-9) · -4 Verified (body.go not in file map) · -5 Partial(2-3/4/5) · -6 Partial(2-12) · -7 Verified variant · -8 Verified · -9 Partial(2-6/10/11) · -10 Verified · -11 Partial(2-9) · -12 not verifiable (no M3 note recorded)

## InfoSec — SEC2 — round-2 verdict: SHIP-WITH-CONDITIONS (2 Important)

| ID | Finding (as stated) → action | Disposition |
|---|---|---|
| SEC2-1 | go-toml v2.2.4 (go.mod pin) PANICS on strict decode when an unknown quoted key carries an escape → bump to v2.4.3 AND recover() in unknownKeys (diagnostic never takes Load down) + fixture quoted-escape-key.toml | Fixed in `1d87831` (see §13 for re-openings) |
| SEC2-2 | install-retry storm: installInBackground deletes installing[key] on failure; resolveVoice per segment → new 63 MB download per segment when offline/removed → installFailed map[string]time.Time + installRetryAfter 10 min; one detail line per epoch; [S] "install failed — retries at hh:mm"; test 3 resolves → 1 attempt | Fixed in `1d87831` (see §13 for re-openings) |
| SEC2-3 | delete Host.Discovering() / the trust window: Discovered() returns macVoices() until listVoices lands (curated ⊇ intersected); removes the matrix row + SEC-4 special case; also: a bad name → render error ENDS the broadcast (source.go s.fail) | Fixed in `1d87831` (see §13 for re-openings) |
| SEC2-4 | SetupNoteMsg not PlainLine'd (dashboard.go:444) → PlainLine on receipt; fixture | Fixed in `1d87831` (see §13 for re-openings) |
| SEC2-5 | keepUnknown: (a) top-level quoted "radio.mode" aliases the known path → resurrects a cleared known key → skip paths colliding with a static knownPaths set; (b) unknown keys under [[locations]] dropped → document as accepted limitation; DecodeError.Key() is a METHOD (strings.Join(e.Key,".") won't compile) | Fixed in `1d87831` (see §13 for re-openings) |
| SEC2-6 | cfg.Unknown PlainLine lives in no code block → PlainLine in app/stats.go ttyStats; move hostile [S] assertion to an app/tty test rendering statusLines() | Fixed in `1d87831` (see §13 for re-openings) |

**Round-1 rows re-verified by this lens:** SEC-1 Partial(2-4,2-6) · SEC-2 Partial: `Segments(body[0])` passes string to Segments(paras []string) — won't compile; add TestForecastIsSegmented · SEC-3 Verified (nit: UGCCodes sorts → cap keeps alphabetical-first 32) · SEC-4 Verified prose, superseded by 2-3 · SEC-5 moot · SEC-6 Verified (edges → 2-5)

## Junior Developer — JD2 — round-2 verdict: NO-GO (JD2-1, JD2-9 Critical compile breaks; 5 Important RED/GREEN contradictions)

| ID | Finding (as stated) → action | Disposition |
|---|---|---|
| JD2-1 | keep.go: DecodeError.Key is a METHOD → e.Key() | Fixed in `1d87831` (see §13 for re-openings) |
| JD2-2 | resolve.go/validate.go import invariant | Fixed in `1d87831` (see §13 for re-openings) |
| JD2-3 | 1.8 RED demands [radio.tones] muted from Radio.Validate; GREEN returns nil → delete that RED assertion | Fixed in `1d87831` (see §13 for re-openings) |
| JD2-4 | 1.8 prose "all six classes" vs empty set; unused slices remark | Fixed in `1d87831` (see §13 for re-openings) |
| JD2-5 | 2.7 tests need CastMode:"cast" (else single-voice ignores pairs) | Fixed in `1d87831` (see §13 for re-openings) |
| JD2-6 | cast.AllClassKeys() undefined → Muted:nil | Fixed in `1d87831` (see §13 for re-openings) |
| JD2-7 | 3.6 d.tones.Get sentence; -run 'Tone\|Preset\|Memo' | Fixed in `1d87831` (see §13 for re-openings) |
| JD2-8 | 3.2 marine_test imports slices; drop "contains exists" claim | Fixed in `1d87831` (see §13 for re-openings) |
| JD2-9 | tea.KeyCode is not a type (special keys are runes) → pressSetup single `case rune` | Fixed in `1d87831` (see §13 for re-openings) |
| JD2-10 | questionRows 4 vs RED 5 → split groupData into location/key groups (matches a11y/architect) | Fixed in `1d87831` (see §13 for re-openings) |
| JD2-11 | 4.5 RED strings vs GREEN: "›  " two spaces vs "› "; › on unfocused checkboxes; PadTo(…,24) spacing → regenerate RED from GREEN/mock | Fixed in `1d87831` (see §13 for re-openings) |
| JD2-12 | cycleVoice("",true) returns list[0] but RED expects second entry → "" = index 0 semantics (first → advances to list[1]? No: treat "" as index 0 then forward → list[1]); Names nil map panic → seed non-nil; "›  Alerts" spacing | Fixed in `1d87831` (see §13 for re-openings) |
| JD2-13 | setup_cast imports maps (strings unused); setup_tones slices | Fixed in `1d87831` (see §13 for re-openings) |
| JD2-14 | c.Key() → c.String(); synth.VoiceByKey → VoiceByName; installed-check name unified (Installed) | Fixed in `1d87831` (see §13 for re-openings) |
| JD2-15 | parity test: CastView 7 keys with "all" vs Assignable 9 keys, root key "voice" → state real relation | Fixed in `1d87831` (see §13 for re-openings) |
| JD2-16 | NFR-8 grep 'Voice:' matches SayVoice{Voice:} → drop Voice: from pattern | Fixed in `1d87831` (see §13 for re-openings) |
| JD2-17 | 4.13 ./platform/term declset | Fixed in `1d87831` (see §13 for re-openings) |
| JD2-18 | p2:7 header "ResidentPiper behind the knob" | Fixed in `1d87831` (see §13 for re-openings) |
| JD2-19 | 1.1 GREEN: const ToneRate must be inside the block | Fixed in `1d87831` (see §13 for re-openings) |

**Round-1 rows re-verified by this lens:** JD-3 Partial(2-18) · -4 Partial(2-9; pressSetup cmd-run is prose) · -5 Partial(2-9/13/14) · -7 Partial(2-17) · -9 Partial: placeholders p2:724 "… the at checks …", p2:1690 "… unchanged", p2:939 "/* today's body */", p2:1093 · -11 Partial(2-7, 2-17) · -15 Partial(p2:1690) · rest Verified

## Performance — PERF2 — round-2 verdict: SHIP-WITH-CONDITIONS (3 Important). Writer slack ≈ 0.7 s (oto 0.5 s + 200 ms).

| ID | Finding (as stated) → action | Disposition |
|---|---|---|
| PERF2-1 | Recast path: play calls s.render (full uncached say of the new voice) THEN handOver renders line+rest → two writer renders; TestRecastHandsOverMidSegmentAsOneSay fails as written → in the chunk loop use voiceFor only, compare Name(), handOver is the single Say | Fixed in `1d87831` (see §13 for re-openings) |
| PERF2-2 | announce = uncached Say on the writer per voice change per cycle (2–5/cycle) → render/cache the hand-over line ahead (synthetic Segment in renderLoop, key handoff:from→to under the incoming voice; or cache in announce by cacheKey(voice,"handoff:"+from)) | Fixed in `1d87831` (see §13 for re-openings) |
| PERF2-3 | R6 not instrumented: add a Source test with markVoice{msPerChar:20} over a 7-role cycle asserting max inter-write gap ≤ 700 ms after the first segment; record 0.7 s writer budget in perf-protocol §1 | Fixed in `1d87831` (see §13 for re-openings) |
| PERF2-4 | store double-counts duplicate keys (dup guard + test) | Fixed in `1d87831` (see §13 for re-openings) |
| PERF2-5 | Setup builds columns 6×/key → build once (setupBody) then re-pin | Fixed in `1d87831` (see §13 for re-openings) |
| PERF2-6 | [S] memo key hashes cfg.Stats() per frame incl. Resolutions copy + castRowOf → key on castGen + stats gen; build rows on miss; add BenchmarkFrame_133x44_Status | Fixed in `1d87831` (see §13 for re-openings) |
| PERF2-7 | castRows memo not invalidated by listVoices/tune-path install/preview install → bump castGen there | Fixed in `1d87831` (see §13 for re-openings) |
| PERF2-8 | P3 3.6 says tone goes through d.tones.Get (stale) → align; AlertTone temp slices out[:len+n] | Fixed in `1d87831` (see §13 for re-openings) |
| PERF2-9 | Task 0.2 vs 4.11 contradict (in-tree measure vs formula) → P0 informational row; P4 spike-then-pin ×1.05 | Fixed in `1d87831` (see §13 for re-openings) |

**Round-1 rows re-verified by this lens:** PERF-1 Partial(2-6,2-7) · -2 Partial(2-1) · -3 Verified · -4 Partial: 40 MB not derived (+38% over 29 MB); suggest 32 MB or justify; 2-4 · -5 moot · -6 Verified · -7 Partial(2-9; not yet in tree) · -8 Verified · -9 Verified

## Safety-critical — SC2 — round-2 verdict: SHIP-WITH-CONDITIONS (2 Important)

| ID | Finding (as stated) → action | Disposition |
|---|---|---|
| SC2-1 | store's eviction loop is condition-only (P10-02 flag) → counter form `for n := len(s.order); n > 1 && s.bytes > maxCachedBytes; n--` | Fixed in `1d87831` (see §13 for re-openings) |
| SC2-2 | re-keying narrate.go rows to director.go FAILS the gate: git rename detection (R099) → unchanged admit/awaitAir hunks not emitted; p10-unmatched.sh treats director.go as new → re-keyed rows unmatched → exit 1 → make in_scope rename-aware (git diff -M --name-status) or rule rows on renamed files dormant; record in P2 build log | Fixed in `1d87831` (see §13 for re-openings) |
| SC2-3 | handOver fails to silence (s.fail) contradicting "never fatal" → on error onSeg handoff:failed, set r.gen = nr.gen, keep streaming old pcm; flakyVoice test | Fixed in `1d87831` (see §13 for re-openings) |
| SC2-4 | Link.String and ToneName unchecked array index → size + guard; out-of-range ToneName → PresetDualTone; extend test | Fixed in `1d87831` (see §13 for re-openings) |
| SC2-5 | gocyclo setupCastKey=16, Resolve=15 → split setupPickerKey unconditionally; note Resolve ceiling | Fixed in `1d87831` (see §13 for re-openings) |
| SC2-6 | phantom predicted rows: limit.go:Say (anchors at voice.go:207, filtered) would be UNMATCHED → exit 1; MarineZoneFor P10-02 (all range); setupCastLines P10-04 (26 lines) → predict only: store, setupCastKey, cast density, reason refreshes on existing package rows (config absorbs keep.go etc.) | Fixed in `1d87831` (see §13 for re-openings) |
| SC2-7 | AlertTone negative PulseDur/GapDur panic → guard | Fixed in `1d87831` (see §13 for re-openings) |
| SC2-8 | toneClassCount=6 vs classes → parity test in app/cast_test.go or clamp | Fixed in `1d87831` (see §13 for re-openings) |
| SC2-9 | 2.7 render block keeps 2-minute WithTimeout while note says it goes; 3.6 cites d.tones.Get | Fixed in `1d87831` (see §13 for re-openings) |
| SC2-10 | fetchCentroid trusts id → invariant.Check(marineZoneID) inside | Fixed in `1d87831` (see §13 for re-openings) |

**Round-1 rows re-verified by this lens:** SC-4 Partial(2-4) · SC-5 Partial(2-2) · SC-6 Partial(2-6) · SC-11 Partial(2-5) · rest Verified. Machine: p10 check at HEAD RAN 7 tools, findings null, tree_hash 04ca8069… (vacuous, no Go diff).
