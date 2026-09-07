# Project Brief — Multi-Voice Support (correspondent roles)

| Field | Value |
|---|---|
| Report | project-brief v1.1.0 — **RATIFIED** (HUM LEAD, 2026-08-29); v1.0.0 was the draft; **amended v1.2.0 at the DISCOVER red-team, v1.3.0 at the PLAN rulings/red-team (2026-08-29), and v1.4.0 at PLAN exit (2026-08-30) — see the Amendment Log**; every escalation E-1…E-11 is ruled (MVS-D-29…46) |
| Phase | PLAN (ratified brief; amended at the DISCOVER and PLAN red-teams) |
| Date | 2026-08-29 |
| Author of record | Branden Thompson (HUM LEAD) |
| Feature | `multi-voice-support` · target **0.14.0** |
| Branch | `feature/multi-voice-support` (off `main` @ `186d97c`, the 0.13.0 DEBRIEF) |
| Directives | LEVEL-1 · SEV-0 · HUM LEAD · FULL GIT · FULL DOCS · FULL REPORTS · FULL RCC · FULL PLAN · FULL DIAGRAMS · FULL TDD |
| Rulings | **MVS-D-1** problem statement A · **MVS-D-2** 0.14.0 · **MVS-D-3** assignment lives in Setup; the `[V]` chooser and the player's `Voice: System Voice` control are removed and the Radio UI is redesigned in the same release · **MVS-D-4** full maritime report (buoy observations + tides + currents) · **MVS-D-5** per-product tones in scope · **MVS-D-6** scripted hand-over · **MVS-D-7** the two-voice minimum dropped · **MVS-D-8** fresh install = one voice everywhere, roles opt-in · **MVS-D-9** (2026-08-29) tones: every category is configurable to either share the *Warnings / Disasters* tone or use one specific to that category; presets follow the *style* of known alert tones for the category where one exists (not exact reproductions); candidates presented by the agent for the HUM LEAD's audition · **MVS-D-10** the Radio UI / Setup mocks come from the HUM LEAD when PLAN is ready for them · **MVS-D-11** (2026-08-29, after audition) tone presets ratified — *Dual-tone* (EAS-style 853+960 Hz) for **Warnings & significant disasters** (tornado warnings, earthquakes, tsunamis…); *1050 Hz* (WAT-style) for **Watches**; *Classic* (today's 1000 Hz pulses) for **Advisories**; *Soft chime* for **Special Weather Statements**; *Low sweep* for **Tropical / Winter Storm — named maritime storms**; the rising two-tone, descending two-tone and quake chime dropped ("too happy / like a phone ring") · **MVS-D-12** (OQ-13) a burst mixing classes sounds ONE tone, by its highest-severity event · **MVS-D-13** (OQ-14) a *Station* role for the lead and the sign-off; the lead is always "This is the <report> for Watchpost Radio." and the tail follows today's rules until UAT · **MVS-D-14** (OQ-15) the coastal-waters forecast is IN scope bounded to what the existing plumbing gives for free (coastal cities already carry marine data) · **MVS-D-15** (OQ-16) storm class wins over warning/watch; *Winter Weather Advisory* stays advisory · **MVS-D-16** (OQ-17) a Setup preview ducks the live broadcast, the arbiter owning the duck — and the arbiter is renamed the **Station Director**, in concept and in code (the radio-station metaphor) · **MVS-D-17** (OQ-18) the resident-Piper decision is deferred; HUM LEAD measures RSS at UAT on a real build (Linux has so far out-performed macOS in threads and memory) |
| Standing lens | **Performance & resource** (carried from 0.13.0) and **R6 — radio is sacred**: this feature lives *on* the radio path, so the relay/audio smokes are blocking at VALIDATE on both platforms |
| Status | Brief APPROVED — HUM LEAD, 2026-08-29 ("1. A · 3. Both · 4. Full · 5. In scope · 6. scripted · 7. Merge the voice options into Setup and remove `[V]`; update the Radio UI at the same time · 8. Drop it · 9. Approach approved"); DISCOVER opens next |
| Decision namespace | Intake decisions are **MVS-D-n**; bare `D-n` in source comments remain watchpost-cli rulings |

---

## Executive Summary

Watchpost Radio speaks with one voice: whichever the listener picked under `[V]` reads everything — the
breaking takeover, the severe-event read, the location's weather, fire and seismic reports and the sign-off.
This feature lets the listener assign voices to **correspondent roles** — a voice for alerts, another for the
routine reports, or one per report — so the station sounds like theirs and an alert is recognisable by ear
before its words register. It adds the missing **maritime report** so every dashboard section has a
correspondent, gives each class of alert its own tone, and moves every voice choice into Setup, retiring the
`[V]` chooser and redesigning the Radio UI with it. The brief was drafted from the HUM LEAD's intent and a
code-level probe of the voice path, and ratified on 2026-08-29; DISCOVER opens next.

## Summary & Intent (raw input, captured verbatim in spirit)

> Now that we've got a lot of the core features wired, I want to enable using more than one voice
> simultaneously for Watchpost radiocasts, where the user can either choose a single voice, or assign
> available voice profiles to a specific "correspondent role":
> *Alerts & Notification Reads* → BREAKING ALERTS (takeover announcements over the ticker) · SEVERE WEATHER /
> DISASTER ALERT READS (from the `w` modal). *Normal Weather & Event Reads* → LOCATION WEATHER (the NWS report
> for a location) · LOCATION MARITIME (new — like FIRE and SEISMIC) · LOCATION FIRE & HOTSPOTS · LOCATION
> SEISMIC REPORTS. Purely a "user delight" feature: distinctiveness for certain reads ("Use Rishi for
> Breaking Alerts, and System Voice for everything else"). Constraint: if only two voices are feasible for
> performance reasons, the minimum is one voice for Alerts & Takeovers and one for everything else.
> Second/third-order: multiple/configurable tones per alert/takeover type (Special Weather Statements get a
> different tone than Warnings).

- **What we are making:** a role → voice assignment with inheritance (everything → group → role), the UI to
  set and preview it, the narration and broadcast paths honouring it, and a maritime report.
- **Why now:** the correspondents exist (six Piper voices, ten macOS voices), the reads exist (takeover,
  severe read, weather, fire, seismic), and the arbiter that orders them shipped in 0.13.0 — the station has
  a cast but only one seat.
- **Who benefits:** the listener — especially the one who is *not* looking at the screen when the radio
  changes what it is saying.
- **If we do nothing:** every read keeps sounding the same; an alert is only as recognisable as its opening
  words; the station cannot be made one's own.

## Problem statement — refinement trace

**Raw (as scored):** *"Enable using more than one voice simultaneously for Watchpost radiocasts, where the
user can assign voice profiles to correspondent roles."*

| # | Criterion | Score | Why |
|---|---|---|---|
| 1 | Bad Outcome | ✗ | a desired state ("enable…"), not what goes wrong today |
| 2 | Affected Humans | ~ | "the user" — which listener, in what moment? |
| 3 | Tech Agnostic | ✗ | "voice profiles", "radiocasts", the app's own role names |
| 4 | Non-prescriptive | ✗ | prescribes the mechanism (assign profiles to roles) |
| 5 | Verifiable | ~ | "distinctiveness" is not observable as stated |

Score **0/5 (+2 partial)**. One pass names the affected human (the listener away from the screen), removes the
mechanism, and states the observable: by ear, one read is indistinguishable from another.

### Candidate A (recommended) — the observable outcome

> **"A Watchpost listener who is not looking at the screen cannot tell, by ear, that what has just started
> speaking is an alert rather than one of the routine reports."**

| # | Criterion | Score | Evidence |
|---|---|---|---|
| 1 | Bad Outcome | ✓ | an inability — cannot tell an alert from a routine read |
| 2 | Affected Humans | ✓ | the listener not watching the screen (radio on in the next room, SSH session minimised) |
| 3 | Tech Agnostic | ✓ | no voice engine, profile, ticker or modal named |
| 4 | Non-prescriptive | ✓ | a second voice, a distinct tone, a spoken station ident, a chime would each address it |
| 5 | Verifiable | ✓ | hide the screen, play a takeover and a weather report, ask which was which — today both come in the same voice with the same opening cadence; the only cue is the words |

**5/5 — LOCKED.** Ratified by HUM LEAD 2026-08-29 ("1. A"). Note the honest reading of criterion 4: a *tone* already precedes a takeover (`domains/radio/synth/tone.go`),
so "by ear" today means "by the tone, if you know it, or by the words" — the statement is about the *voice*
that follows, and a listener who joins mid-line has neither cue. The HUM LEAD's "delight" framing is carried
as the **intent** (Summary above) and by metric M2; the problem statement is what a new listener can verify.

### Candidate B (the delight reading, for comparison)

> *"A Watchpost listener cannot give the station a character of their own beyond one voice for everything."*

Scores 4/5 — criterion 5 is weak ("character" is not observable without asking the listener). Recorded so the
ratification was a choice, not a default; not chosen (MVS-D-1).

## Metrics of Success

| ID | Metric | Symbol | Type | Definition (direction) | Measured in |
|---|---|---|---|---|---|
| M1 | Role fidelity | `ROLE` | Primary | Share of reads spoken in the voice resolved for their role: **100 %**, higher is better — with distinct voices assigned to *Alerts* and *Standard*, a breaking takeover, a severe read, and each section of the location broadcast render with the voice their role resolves to (inheritance included) | table-driven resolution test + narration/broadcast render tests on the recording output (`go test ./app -run Role`, `./domains/radio/synth -run Role`) — acceptance criteria, no post-ship telemetry by design |
| M2 | Actions to assign | `A2V` | Secondary | Distinct in-app actions from the dashboard to "role X speaks in voice Y", persisted, **no typing**: target **≤ 5** (open Setup/chooser, pick the role, pick the voice, confirm, close), lower is better | scripted-PTY journey (fresh HOME) |
| M3 | Ear test | `EAR` | Tertiary | With the screen hidden and two distinct voices assigned, a listener names "alert" vs "report" from the first two seconds of audio: **≥ 9/10** trials, higher is better | HUM LEAD UAT (the only human-in-the-loop metric; the app-level proxy is M1) |
| M4 | Radio untouched + resource | `R6·PERF` | Maintenance | No regression: relay/audio smokes green on both platforms; time-to-first-audio of a takeover **not worse** than 0.13.0 with a pre-installed voice; peak RSS during a takeover-over-read overlap bounded and pinned; closed-frame allocations unchanged | `WATCHPOST_LIVE=1 go test ./app -run LiveRelay`; `WATCHPOST_DEBUG_PPROF` soak; the alloc pins in `modes/tty/bench_test.go` |
| M5 | Never silent | `FALL` | Tertiary | A role whose assigned voice cannot be resolved (uninstalled, renamed, other OS) speaks in the next voice up its inheritance chain, never in silence and never in the wrong role's voice: **0** silent reads in the fallback matrix | table-driven test over the fallback matrix |

**Anti-solution check (Step 5).** M1 can be satisfied by hard-wiring two voices with no user choice — M2
(a persisted, in-app assignment) closes that. M2 can be satisfied by a config-file key nobody can find — the
"from the dashboard, no typing" clause closes that. M3 can be gamed by a longer tone — M3 measures the
*voice* after the tone (trial audio starts after the tone). M4 can be met by shipping without pre-installing
role voices (nothing changes until first use) — M4's latency clause and R-9 close that. M5 can be met by
silently using one voice for everything — M1 catches it. Read together.

## Requirements

Verifiable form; stated preferences in brackets are the HUM LEAD's default binding for PLAN.

| ID | Requirement |
|---|---|
| R-1 | The listener can assign a voice at each of these levels, and a level without its own assignment **inherits** the next level up: *All reads* → {*Alerts & Notification reads* → Breaking alerts/takeovers · Severe weather / disaster event reads} · {*Standard Watchpost Radio reports* → Location weather · Location maritime · Location fire & hotspots · Location seismic} |
| R-2 | With nothing assigned below *All reads*, behaviour is byte-for-byte today's: one voice, chosen under `[V]`, persisted in `voice` |
| R-3 | Every read is spoken in the voice its role resolves to (M1) — the takeover and its tone, the `[space]` severe read, and each section of a location broadcast, including when sections in one broadcast resolve to different voices |
| R-4 | Assignment is made in the **Setup window** without typing (MVS-D-3: the voice options merge into Setup as a group; the `[V]` chooser is removed and its key freed; the player's `Voice: System Voice` control goes with it and the Radio UI is redesigned in the same release — mock from the HUM LEAD), previewed per role before it is saved (a preview is never pausable by a takeover), and persisted; it survives restart and an upgrade/downgrade of the binary |
| R-5 | The station's identity lines name the voice that is actually speaking (lead, sign-off, `{{voice}}`), and a change of correspondent within one broadcast is a **scripted hand-over** (MVS-D-6) — a user-overridable script part, wording in PLAN — not a splice |
| R-6 | A new **maritime report** joins the location broadcast with the same shape as fire and seismic (head · content · absence line · script parts users can override), spoken only when the location has marine data; its role is *Location maritime*. **Full content (MVS-D-4):** buoy observations, tides and currents |
| R-7 | A role whose voice cannot be resolved falls back up the chain (M5); the fallback is stated in the diagnostics (`[S]`) and never in silence |
| R-8 | Both platforms: macOS system voices and the Piper catalogue on Linux/Windows; a config written on one OS loads on the other with unresolvable roles falling back (R-7) |
| R-9 | The breaking path never waits on a cold voice install: voices assigned to *Alerts* roles are installed ahead of need (at assignment or at start), and until they are, the takeover speaks in the fallback voice |
| R-10 | The dashboard shows which voice is on the air for the current read (the voice chip), consistent with R-5 |
| R-11 | Performance envelope (M4): a takeover with a distinct voice starts no later than today's; the peak during a takeover-over-read overlap is pinned; no relay change |
| R-12 | **Per-class tones (MVS-D-5/9/11, in scope):** the tone that precedes a takeover or a severe read is chosen by the product's class from five ratified presets — **Warnings & significant disasters** → dual-tone (853+960 Hz) · **Watches** → 1050 Hz · **Advisories** → classic (today's 1000 Hz pulses) · **Special Weather Statements** → soft chime · **Tropical / Winter Storm (named maritime storms)** → low sweep — each class switchable in Setup to *share the Warnings / Disasters tone*, each previewable there; one owner in code (`synth/tone.go` generalised); the presets' parameters are recorded in `02-analysis/tones.md` and their sound remains the HUM LEAD's pass |

## Technical Constraints (from the code probe, 2026-08-29)

- **Feasibility — more than two voices is fine.** The audio engine has no notion of voice: every clip carries
  its own sample rate and is resampled per clip (`domains/radio/player/engine.go:272-273`), and narrations
  already resolve a voice per utterance (`app/radio.go:398`, `:406`). The **two-voice minimum is not forced
  by performance**; the cost is disk (~63 MB per Piper voice) and the per-utterance model load that exists
  today regardless of how many voices are assigned. The HUM LEAD's "minimum" fallback is therefore a scope
  choice, not a technical one.
- **The broadcast stream is one voice at a time.** The location broadcast renders through one
  `synth.Source` with one `Voice` (`domains/radio/synth/source.go:32`); `SetVoice` refuses a different sample
  rate (`source.go:87-89`) and the cache is keyed by segment only, protected by a generation counter
  (`source.go:36`, `:93-96`, `:303`). Per-section voices inside one broadcast (R-3) need the voice in the
  cache key and a per-segment voice at render; every voice today is 22 050 Hz (`synth/install.go:73`,
  `synth/voice.go:91`) so the rate lock is latent, not active. A mid-broadcast hand-over already exists
  (`Handoff`, `source.go:101-103`, `:238-264`) — R-5's hand-over builds on it rather than a splice.
- **Piper loads its ~63 MB model on every utterance** (`synth/voice.go:183`; acknowledged in the UI at
  `app/voices.go:140`); there is no daemon, warm-up or model cache. A takeover-over-read overlap with two
  Piper voices means two concurrent loads — the latency and peak-RSS spike M4 pins. A persistent Piper process
  per assigned voice is an area to investigate, not a given.
- **Install serialisation is deck-wide** (`app/radio.go:49`, `app/voices.go:53-57`): an alerts-role voice
  that is not installed would queue behind another download — R-9. `PreviewVoice` bypasses the deck mutex
  (`app/voices.go:132-137`) and must keep using `Audition` (`app/voices.go:150`, R5-B-03).
- **Voice identity is a bare string in two namespaces** — macOS names incl. the `"System Voice"` sentinel
  (`app/voices.go:218-230`) vs the Piper catalogue `Key`/`Name` (`synth/install.go:64-104`); resolution falls
  back silently (`app/voices.go:67-78`). Per-role fields need explicit resolution and R-7's stated fallback.
- **Config has no version or migration** (`platform/config/config.go:136-137`, `:64-82`): new fields must be
  additive, zero-value-means-inherit; the single `voice` key stays as the *All reads* level (R-2, R-8).
- **The narrator's seam**: `narrationVoice.render/tone` (`app/narrate.go:57-66`, `app/radio.go:372-407`)
  call `d.voice()` with no notion of who is asking; the classes are numeric (`narrate.go:30-33`). Threading
  the role through the speaker is the minimal change; a new class (a maritime read on its own) renumbers.
- **The tone is hard-coded** (`synth/tone.go:14-23`) and takes its rate from the current voice
  (`app/radio.go:377`) — R-12's "one owner" is nearly free; per-product tones are new design.
- **No maritime script exists** — `synth.Compose` takes fire and seismic hooks only (`synth/compose.go:48`);
  buoys/tides/currents are dashboard-only (`modes/tty/detail_marine.go`, providers at `app/dashboard.go:74`).
- Radio is sacred (R6): the relay path (`Handoff` aside) is not touched; the Linux half of the 0.13.0 R6 gate
  is still owed and folds into this feature's VALIDATE.
- UI primitives from `third_party/go-studs` where they exist; Setup's settings-group pattern is the home for a
  new group (`modes/tty/setup.go:252-285`, "More groups join as more configurability is added").

## Other Considerations

- **Second- and third-order effects (HUM LEAD):** per-product tones (R-12, OQ-5); a hand-over line between
  correspondents (R-5, OQ-6); the `{{voice}}` token and the sign-off naming the broadcast voice; the voice
  chip's meaning when two voices alternate; the `[V]` chooser's meaning once roles exist (OQ-7); the
  diagnostics page listing the cast.
- **Prior art:** the `[V]` chooser (`modes/tty/modal_chooser.go`); the 0.12.0 P4 install serialisation; the
  0.13.0 narrator arbiter and the engine's held/aside/audition lines; the fire and seismic report shapes
  (`synth/fire.go`, `synth/seismic.go`, `scripts/fire-report`, `scripts/seismic-report`); the settings-group
  Setup (0.12.0); the pronunciation rule tables (0.13.0) — a voice-only spelling may need to be per voice.
- **Non-goals (proposed, DISCOVER to confirm):** no new voice engines or cloud TTS; no per-location voices;
  no voice cloning; no tone editor beyond presets; no change to the relay; no more than the six Piper voices
  in the catalogue unless a role needs a seventh.
- **Stakeholders:** HUM LEAD (approver; UAT ear test; Linux validation; the tones' sound is the HUM LEAD's
  pass, as colours are); the listener; junior developers and future agents (the record); upstream Piper
  (a persistent-process mode, if investigated).

## Discovery Handoff Package

### Areas to Investigate

| # | Area | Concrete artefacts |
|---|---|---|
| A-1 | **Role model + resolution** — the role tree, inheritance, the config shape (additive keys under `[radio]`; `voice` stays the root), resolution table with fallback (R-1, R-7, R-8) | `platform/config/config.go:41-43,147`; `app/voices.go:29-78`; `synth/install.go:64-104` |
| A-2 | **Per-section voices in one broadcast** — voice in the cache key, per-segment voice at render, the `Handoff` line as the hand-over, `{{voice}}`/tail naming, the rate lock | `synth/source.go:32-103,200-264,291-331`; `synth/compose.go:48-140` |
| A-3 | **Narration seam** — role on the speaker; `render`/`tone` resolving by role; the tone's rate from the role's voice; class numbering if a new class is needed | `app/narrate.go:30-66,218`; `app/radio.go:363-418`; `app/ticker.go:248-272`; `app/severe_read.go:61-103` |
| A-4 | **Piper cost (perf lens)** — pre-install of assigned voices (R-9); whether a long-lived `piper` process per voice removes the per-utterance load; the overlap's peak RSS; disk budget per voice | `synth/voice.go:144-194`; `app/voices.go:79-151`; `06_docs/…/quality-baseline.md:75`; `soak-profile.md:23-37` |
| A-5 | **UI** — the Setup group ("Radio / Correspondents": the role tree, the tones); removing the `[V]` chooser; the Radio UI redesign (control row, chip) from the HUM LEAD's mock; go-studs fit; preview via `Audition` (OQ-10) | `modes/tty/setup.go:31-90,220-376`; `modal_chooser.go:52-151`; `dashboard.go:266,835-837` |
| A-6 | **Maritime report** — content from the marine snapshot (buoy obs, tides, currents), script parts, absence line, its place in the composition order (R-6) | `modes/tty/detail_marine.go`; `platform/snapshot` marine types; `synth/fire.go:38`, `synth/seismic.go:32` as the shape |
| A-7 | **Tones** — one owner; per-class presets (Warnings · Watches · Advisories/Statements · Quakes · Tropical), their preview in Setup, the HUM LEAD's sound pass (R-12, OQ-11) | `synth/tone.go`; `app/radio.go:372-380` |
| A-8 | **Platform parity** — the macOS `say` voice set vs the Piper catalogue; portability of per-role names; the ear test on both | `app/voices.go:218-258`; `synth/install.go:76-91` |

### Stakeholders to Consider

HUM LEAD (every gate; the ear test; the tones' sound; Linux R6) · the listener away from the screen · junior
developers and future agents · upstream Piper.

### Risk Signals

| # | Risk | Mitigation |
|---|---|---|
| RS-1 | **Latency on the breaking path** — a cold or concurrent Piper load delays the most urgent read | R-9 pre-install; M4 pins time-to-first-audio; fallback voice until installed; A-4 investigates a warm process |
| RS-2 | **Wrong voice in the wrong role** silently (string identity, two namespaces) | R-7 explicit resolution + `[S]` line; M5 fallback matrix; the recording-output tests (0.13.0 lesson 1) |
| RS-3 | **A splice mid-broadcast** — sections in different voices sounding like a bug | R-5 hand-over built on `Handoff`; cache key carries the voice; the HUM LEAD's ear at UAT |
| RS-4 | **Config drift across OS/versions** with no migration | additive keys, zero = inherit, `voice` stays the root; a downgrade reads exactly today's key |
| RS-5 | **Scope** — tones, hand-over scripts, a full maritime report and a Radio UI redesign are all in (MVS-D-3/4/5/6) | PLAN batches them DATA FIRST (role model → broadcast seams → maritime → tones → UI) so each lands on a stable spine; the non-goals bound the rest |
| RS-6 | **Peak RSS** during takeover-over-read with two models resident | M4 pin; soak phase B; cap concurrent Piper processes at 2 (today's budget) |
| RS-7 | **R6** — the broadcast path changes (per-segment voices) | relay untouched; both platforms' smokes blocking at VALIDATE; the Linux half owed from 0.13.0 folds in |

### Open Questions (HUM LEAD)

| ID | Question | Intake recommendation |
|---|---|---|
| OQ-1 | Ratify the problem statement — Candidate A (by ear) or B (character) | **Resolved — A** (MVS-D-1) |
| OQ-2 | Version | **Resolved — 0.14.0** (MVS-D-2) |
| OQ-3 | Where assignment lives | **Resolved — Setup** ("3. Both" was superseded by 7: the voice options merge into Setup; `[V]` is removed) (MVS-D-3) |
| OQ-4 | Maritime report scope | **Resolved — full** (buoy observations + tides + currents) (MVS-D-4) |
| OQ-5 | Per-product tones | **Resolved — in scope** (MVS-D-5); R-12 |
| OQ-6 | Hand-over | **Resolved — scripted** (MVS-D-6); wording in PLAN |
| OQ-7 | `[V]` once roles exist | **Resolved — neither: remove `[V]`**, merge the voice options into Setup, redesign the Radio UI in the same release (MVS-D-3). New for DISCOVER: **OQ-10** what the freed `V` key and the player's control row become, and the Radio UI mock (HUM LEAD) |
| OQ-8 | Two-voice minimum | **Resolved — dropped** (MVS-D-7) |
| OQ-9 | Fresh install default | **Resolved — approved:** one voice everywhere; roles opt-in (MVS-D-8) |
| OQ-10 | Radio UI redesign — the player's control row without `[V]`; the voice chip (R-10); the freed key | **Open — HUM LEAD's mock at PLAN (MVS-D-10)** |
| OQ-13 | A burst mixing classes | **Resolved — one tone, highest severity (MVS-D-12)** |
| OQ-14 | Lead and sign-off | **Resolved — Station role; lead "This is the <report> for Watchpost Radio."; tail as today until UAT (MVS-D-13)** |
| OQ-15 | Spoken coastal-waters forecast | **Resolved — in scope, what the existing plumbing gives for free (MVS-D-14)**; DISCOVER A-6 addendum: the NWS products endpoint already fetched for HWO/SPS/NOW/ZFP also serves CWF (Coastal Waters Forecast) — the free candidate |
| OQ-16 | Two-class products | **Resolved — storm wins (MVS-D-15)** |
| OQ-17 | Preview while on air | **Resolved — duck; the arbiter renamed Station Director (MVS-D-16)** |
| OQ-18 | Resident Piper process | **Deferred — HUM LEAD measures at UAT (MVS-D-17)**; PLAN designs the `Voice` seam so either backend fits |
| OQ-11 | Tone presets | **Resolved (MVS-D-9 + MVS-D-11):** five presets ratified after audition (see R-12); each category can still be switched to *share Warnings / Disasters*. Open detail for DISCOVER → **OQ-16**: classifier precedence for products that belong to two classes (a *Winter Storm Warning* — storm tone per the ruling's wording, or warning tone?; a *Winter Weather Advisory* → advisory) |
| OQ-12 | Maritime report order in the broadcast (after weather, before fire?) and its absence line | **Open — for DISCOVER (A-6)** |

### Problem statement status
Refined via `refine-problem-statement` (0/5 → 5/5, Candidate A) and **locked** by HUM LEAD 2026-08-29 (MVS-D-1);
measurements validated for anti-solutions (Step 5).

### Sharpening observations
- SH-1: the raw requirement listed *levels* (checkboxes); the brief states the underlying need — inheritance
  (R-1) — so PLAN owns the UI.
- SH-2: "purely user delight" and a verifiable problem are both kept: the problem is what a listener can
  observe (A); the delight is the intent and M2/M3.
- SH-3: the "two voices if performance forces it" constraint was tested against the code and found not to
  bind (Technical Constraints §1) — recorded so PLAN does not design to a limit that is not there.
- SH-4: the maritime report is a feature in its own right riding on this one; OQ-4 scopes it so RS-5 stays
  bounded.
- SH-5: the tone is the HUM LEAD's pass (as colours are); R-12's minimum keeps the door open without
  designing the presets here.


---

## Amendment Log

### v1.2.0 — DISCOVER red-team round 1 (2026-08-29; reviewers A requirements · B technical · C hostile input/UX · D documentation + performance)

The ratified text above is frozen; these entries correct or supersede it. The objectives (`01-objectives/objectives.md`)
and the data shape of record (`02-analysis/data-shape.md`) carry the amended wording.

| # | Where | Was | Now |
|---|---|---|---|
| AM-1 | R-2 | "chosen under `[V]`" | the root voice is chosen in **Setup** (MVS-D-3); the `[V]` chooser is removed. Objectives FR-2 |
| AM-2 | R-12 / A-7 | the pre-audition preset set; "the tone that precedes … a severe read" (implying one exists) | five ratified presets (MVS-D-11, `02-analysis/tones.md`); the severe read has **no tone today** — FR-11 adds one, suppressed under `[M]` (red-team A-5) |
| AM-3 | RS-6 | "cap concurrent Piper processes at 2 (today's budget)" | **no cap exists in the code**; the worst case today is 4–5 (`02-analysis/piper-and-platform.md` §2). The cap is FR-12, PLAN batch 1 |
| AM-4 | SH-3 | "the two-voice constraint … found not to bind" | it does not bind at the audio engine; **it binds on Linux until FR-12's cap and (if adopted) a resident Piper process exist** — MVS-D-7 stands, on that understanding |
| AM-5 | Technical Constraints | "config has no version or migration (`config.go:136-137`, `:64-82`)" | the `:64-82` range is `Fire.WithDefaults` — the correct citations are `config.go:136-137` (unknown keys ignored) and `:197` (Load validates only Fire) |
| AM-6 | R-1 | the role tree without station identity | a **Station** role (MVS-D-13); the tree of record is in `data-shape.md` |
| AM-7 | R-6 | "buoy observations, tides and currents" | + the **Coastal Waters Forecast**, served for free by the existing products endpoint (MVS-D-14, `maritime-report.md` §10) |
| AM-8 | M2 | "≤ 5 actions" counted as steps | counted as **keypresses** from the dashboard, fresh HOME — the bound is set in the objectives (red-team A-7/D-14) |
| AM-9 | M3 | measured only "with two distinct voices assigned" | + a **fresh-install arm** (tone-only): the locked problem's default-on answer is FR-11 (red-team A-8) |
| AM-10 | M4 / RS-6 | "peak RSS … bounded and pinned" | there is no RSS pin mechanism; M4 uses the soak phase-B pass criterion, the 0.13.0 baseline (RSS 116 MB launch / 86–115 MB plateau, `severe-alerts-modals/08-reports/validate-report.md` §4) and the measurement protocol in `07-readiness/perf-protocol.md` (red-team D-2) |
| AM-11 | RS-4 | "`voice` stays the root" as a single string | the root stays `voice` for compatibility, but its **terminal fallback** (unresolvable → platform default, stated in `[S]`) is FR-7's last row (red-team A-6) |
| AM-12 | Non-goals | "no widening of the macOS curation" | unchanged — and the `voice` **action** is kept, redefined as "open Setup at Correspondents" so a rebound `V` never breaks startup (OQ-19, red-team A-3/C-5) |
| AM-13 | Exit rulings (2026-08-29, DISCOVER exit — "Approved") | — | **MVS-D-18** (OQ-12) maritime order: products → maritime → fire → seismic → tail · **MVS-D-19** (OQ-19) the `voice` action stays, default `V`, opens Setup at Correspondents · **MVS-D-20** (OQ-20) Setup's save may drop hand-written comments in 0.14.0 (keys preserved), documented · **MVS-D-21** (OQ-21) maritime wording as recommended *for now*; **future preferences noted for the backlog**: 12/24-hour time for display and narration; a time-zone preference (default system/local); a "transmitting on my own FRS channel" mode that changes the time wording and certain heads/tails · **MVS-D-22** (B-1) the downgrade wording accepted — may be revisited on catastrophic issues (fewer than four humans use pre-1.0 builds) |

### v1.3.0 — PLAN rulings and PLAN red-team amendments (2026-08-29)

| # | Where | Was | Now |
|---|---|---|---|
| AM-14 | PLAN rulings (2026-08-29) | — | **MVS-D-23** AX-1…AX-8 ratified as recommended ("AX-all approved"); RAT-4 hand-over wording OK; the Radio panel mock supplied (`02-analysis/mocks/radio-panel.md`): **both `[V] Voice` and `[T] Size` retired — a standard vertical size per breakpoint**; three breakpoints (wide/medium/narrow) with a header volume bar |
| AM-15 | PLAN rulings (2026-08-29) | R-10 the on-air chip | **MVS-D-24**: the on-air correspondent is **dropped from the Radio panel for now** (FR-10 → the `[S]` cast table only); the volume stays on the `-`/`+` keys as today, the mock's header bar displays it; the Setup mock is in progress (MVS-D-10) |
| AM-16 | Setup mock rulings (2026-08-29) | R-4, R-11/R-12, MVS-D-9 | **MVS-D-25** Single Voice ↔ Correspondent Cast is a *mode* (`[radio] cast`), assignments kept when switching · **MVS-D-26** the tone section is a **per-class mute**; the "share the Warnings tone" switch (MVS-D-9) is **dropped** — every class always sounds its ratified preset; `[M]` toggles *All Tones On ↔ Mute* and mutes **tones only** — the words always read (supersedes 0.13.0's "Mute Severe Alerts" muting the narration) · **MVS-D-27** a `p  Preview` chip appears when the focus is on a voice picker · **MVS-D-28** the classes as drawn: *Significant Quakes & Disasters* and *Warnings* separately mutable (both the dual-tone), *Maritime* = the Tropical / Winter Storm class |
| AM-17 | M3 | one trial, "alert vs report from the first two seconds", ≥ 9/10; the fresh-install arm "tones only" (AM-9) | **two trials, A/B against the 0.13.0 binary, randomised** (`07-readiness/gates.md` §2): the fresh-install arm asks the listener to *name the class from the tone* (0.14.0's six-class tones vs 0.13.0's one tone); the cast arm keeps the brief's clause — *alert vs report with the tone stripped* (the voice carries it). Pass: 0.14.0 ≥ 9/10 **and** better than 0.13.0 on the same trial. If 0.13.0 already scores ≥ 9/10 on the cast trial, E-3 rules on the problem statement's standing (PLAN red-team round 2) |
| AM-18 | M2 | "≤ 5 actions" (AM-8: keypresses; the bound set in the objectives as ≤ 12) | the measured path is **11 keypresses** (`V ↓ space ↓ → enter×6`); the journey script pins **≤ 11** — a regression pin, not a target (PLAN red-team round 2) |
| AM-19 | M5 | "0 silent reads in the fallback matrix" | 0 silent reads **except the one legitimately silent row** — a Linux host with nothing installed and nothing installable (FR-7), which `[S]` explains |
| AM-20 | RAT-3 / RAT-5 / RAT-6 | "not objected — proceeding" | at SEV-0 silence is not a ruling: E-7 asks for explicit rulings on the Blizzard-in-storm class, the maritime wording as written, and the three-period cap |

### v1.4.0 — PLAN exit: GO for BUILD, and the escalation rulings (2026-08-30)

**"PLAN approved; GO 4 BUILD."** The Plan of Record (`por-report.md`) is approved and BUILD is authorised. The
eleven escalations are ruled **as recommended in the PoR**; each becomes a ruling below. Where the PoR offered no
recommendation (E-5) the default taken is recorded as such and stays reversible until its batch.

| AM | Ruling | Was | Now |
|---|---|---|---|
| AM-21 | **MVS-D-42** (E-9) | M4: one instrument, ambiguous between a code-path measure and a wall-clock one | **two parts, one instrument**: `BenchmarkTimeToToneStart` as the code-path pin on both trees, plus an absolute live budget observed on the `breaking:`/`tone:` debug lines (PLAN red-team rounds 3-4) |
| AM-22 | **MVS-D-29** (E-1) | AX-1 "build the seam for all three backends" | the **resident-Piper path is cut from 0.14.0**, confirmed. The seam AX-1 asked for is the existing `synth.Voice` interface, so nothing is lost; the measurement is 0.15.0 input on the HUM LEAD's UAT list |
| AM-23 | **MVS-D-30** (E-2) | scope open for cutting | **(a) the whole plan ships.** Each batch is UAT-able alone, so the (b) maritime / (c) per-class-mute-UI / (d) Radio-panel cuts remain available at the P2 and P3 gates without re-planning |
| AM-24 | **MVS-D-31** (E-3) | objectives 0 / AM-9 claimed the tones are the "default-on answer" | **Candidate A stands as the locked problem; the "default-on answer" wording is dropped.** The tones are the fresh-install *cue* (they say *which class*); the **cast** is the answer to alert-vs-report, and it is opt-in by MVS-D-8. M3 still runs; it informs 0.15.0, it does not re-open this |
| AM-25 | **MVS-D-32** (E-4) | - | **MVS-D-8 stands**: a fresh install has one voice. No default alert voice is installed unasked (macOS free, Linux +63 MB) |
| AM-26 | **MVS-D-33** (E-5) | the mock's word "Maritime" for the storm class | **the mock's word is kept.** *No recommendation was offered; this is the standing mock-fidelity default, not a considered ruling* - the HUM LEAD may substitute "Tropical / Winter Storms" any time before the P4 gate at the cost of one label and one golden |
| AM-27 | **MVS-D-34** (E-6) | `[M]`'s meaning changes silently for a 0.13.0 listener | **(a) accept, with a CHANGELOG line.** Someone who muted alerts in 0.13.0 will hear the words again; no legacy mapping and no in-app note |
| AM-28 | **MVS-D-35/36/37** (E-7) | AM-20: silence is not a ruling | **all three ratified explicitly**: RAT-3 Blizzard Warnings sit in the storm class - RAT-5 the maritime wording ships **as written** in `maritime-report.md` 7 as amended by MVS-D-21 - RAT-6 the coastal forecast is cut to **three periods** (`SpokenPeriodsCap = 3`) |
| AM-29 | **MVS-D-38** (E-8) | AX-7 open between (A) nearest-zone-by-geometry and (B) first nearshore block | **(B) only for 0.14.0.** MVS-D-14 ruled the CWF came "for free"; (A) is a new NWS endpoint and is not free. (A) returns in a later release if UAT reads the wrong stretch of water. P3 Task 3.5 builds (B); (A)'s sibling code is not written |
| AM-30 | **MVS-D-39/40/41** (E-9) | AM-17 / AM-18 / AM-19 proposed by the red-team against ratified metrics | **ratified**: M3 is two trials with the comparative rule - M2's target becomes the measured **11 keypresses pinned <= 11**, a regression pin rather than a target (OP-4's Save chord is the only lever on the number) - M5 admits the one legitimately silent Linux row |
| AM-31 | **MVS-D-43** (E-9) | AX-4: N ordinary slots, **one** reserved | **two reserved slots** ratified (D-R3-1): a Station Director job must never queue behind the broadcast's read-ahead, and the hand-over line is a second Director-initiated `Say` |
| AM-32 | **MVS-D-44** (E-10) | `ensureRoleVoices` installs every named-and-missing Piper voice on launch, unasked | **(b) a per-session cap**: two background installs per session, then `[S]` reads "install on next Save". A hand-edited config may not pull ~380 MB unattended |
| AM-33 | **MVS-D-45** (E-11) | NFR-7 promised `golangci-lint run ./...` and `staticcheck ./...` "at every exit" | **NFR-7 is amended to the gates that actually run** (`make verify` = fmt vet tidy vuln race lint-imports lint-watermark gate-controls; plus `make p10` 0/0, `a2dh validate`, `make pty-severe`, `make alloc-budget`). Neither linter has ever run in `make verify` or CI and `golangci-lint` is red on pre-existing style outside this feature; **no new linter is adopted mid-feature.** "Add a `make lint` target" stays on the carried follow-ups |
| AM-34 | **MVS-D-46** (OP-1..5) | five open points on the Setup mock | **as written in `p4-ui.md`** - the standing mock-fidelity default. Like MVS-D-33 these are label-and-layout decisions the HUM LEAD may revisit before the P4 gate; OP-4's Save chord is the one with a measurable consequence (M2's 11 keypresses) |
