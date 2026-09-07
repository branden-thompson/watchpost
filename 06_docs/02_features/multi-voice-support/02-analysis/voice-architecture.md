# Voice architecture — DISCOVER analysis (multi-voice-support, 0.14.0)

Read-only investigation of the code as of `main @ 186d97c` (2026-08-29), organised by the brief's handoff
areas. Every claim carries a file:line. Sections A-4 / A-6 / A-8 are appended as their investigations land.

## 0. The one-paragraph picture

Text becomes audio on two paths. The **broadcast** (a location's report) is composed into `Segment{Key, Text,
Pause}` (`domains/radio/synth/compose.go:16-20`) and rendered by one `synth.Source` holding **one** `Voice`
(`source.go:32`), with a one-segment look-ahead (`source.go:136`) and a 40-segment mono PCM cache keyed by
segment only (`source.go:36,303`) that is wiped wholesale on `SetVoice` (`source.go:93-96`). The
**narrations** (breaking takeover, severe `[space]` read) go through the arbiter (`app/narrate.go`) whose sole
`narrationVoice` implementer, `*radioDeck`, resolves `d.voice()` **per line** (`app/radio.go:398`) and hands
the engine a clip with its own rate (`app/radio.go:406`); the engine resamples per clip (`player/engine.go:272-273`)
and knows nothing about voices. Every voice on every platform speaks at 22 050 Hz today (`synth/voice.go:91,108`,
`synth/install.go:72-79`). Consequence: per-role voices are **cheap on the narration path and structural on
the broadcast path**.

## A-2 Per-section voices inside one broadcast

### A-2.1 Where the voice attaches

Producers: `Composer.Compose` (`compose.go:48-80`), `FireSegments` (`fire.go:38-75`), `SeismicSegments`
(`seismic.go:32-61`); the app's provider closure `app/radio.go:275` → `radioDeck.segments` (`:294-324`).
Consumer: `renderLoop` (`source.go:160-189`, one goroutine) → `render` (`:301-324`) → `renderedSeg{seg, pcm, gen}`
(`:153-157`) on a buffer-1 channel (`:136`) → `play` (`:198-234`).

**Recommendation:** a **role tag on `Segment`** (`Role` — weather · maritime · fire · seismic · station),
resolved at render by a `resolve func(role) Voice` injected into `Source` at `NewSource` (`source.go:53-61`),
defaulting to "always the one voice" (R-2 byte-for-byte). Not a resolved `Voice` on the segment: `compose.go`
is pure text by design (`compose.go:28-38`) and must not learn installs/config, and a resolved voice on the
segment breaks the generation model at `source.go:314`.

**Cache key sites that need the voice name** (`voiceName + "\x00" + seg.Key`): lookup `source.go:303`, store
`:315`, LRU order `:316`, eviction `:317-320`, the wholesale wipe `:95`, the field comment `:36`. Fire/seismic
keys are already content-hashed and voice-independent (`fire.go:44,72`; `seismic.go:38,58`).

**Latent bug found:** `compose.go:78` keys the tail as `"tail:" + voiceName`, but production passes
`synth.VoiceToken` (`app/radio.go:275`), so the key is the literal `"tail:{{voice}}"` for every voice while
`render` substitutes the real name into the *text* (`source.go:309`). The wholesale wipe masks it today; with
a voice-keyed cache it becomes correct; without one, a role change would replay the previous correspondent's
sign-off. (`synth_test.go:48` asserts `tail:Samantha` — a path production never takes.)

### A-2.2 Generation counter, rate lock, `StartSource`

`gen` is Source-global: `SetVoice` (`source.go:81-97`) sets `prev`/`voice`, bumps `gen`, wipes the cache;
`play` re-renders when `r.gen != gen` (`:201-206`); `render` caches only under the current gen (`:314`). With
per-segment voices `gen` answers the wrong question — `renderedSeg` must carry **which voice rendered it** and
`play` compares against the segment's resolved voice; `SetVoice`'s wipe becomes role-scoped; `s.prev` (`:33`)
loses meaning. Every `s.current()` consumer becomes role-aware: `Rate()` `:65`, `play` `:199,213`, `handOver`
`:244`, `spoken` `:292`.

The rate lock (`source.go:87-89`) is **latent**: all voices are 22 050 Hz; only `TestSetVoiceRefusesADifferentRate`
(`synth_test.go:542`) exercises it. The invariant must move to **assignment time** (Setup validation) because a
mid-stream mismatch has no re-tune escape (`app/voices.go:112-116` serves only the `[V]` path).

`Engine.StartSource` (`player/engine.go:391-399`) halts and re-arms, then fixes **the sample rate** in one
resampler (`:399`; `resample.go:24-29`, `OutputRate = 44100`) from `src.Rate()` (`app/radio.go:288`) — the real
lock — and bakes the **stream name** `"Watchpost Synth (<voice>)"` (`app/radio.go:288`) for the stream's life,
a lie from the second section onward under roles. The narration path builds a resampler per clip
(`engine.go:272-273`) and is already rate-safe.

### A-2.3 `Handoff` today, and the scripted hand-over

`Handoff(new, old)` = `"This is %s, taking over for %s."` (`source.go:101-103`), triggered **only** by a
mid-segment voice change in `play`'s chunk loop (`source.go:212-225`): `Remainder` (`:281-288`, next word
boundary, `""` under three words) then `handOver` (`:238-264`), which speaks the line **in the new voice**
(`:244,250`), renders the remainder concurrently (`:245-248`), writes straight to the pipe (`:253`), never caches
(`:329-331`), and marquees under the interrupted segment's key (`:252,262`).

**For R-5 (MVS-D-6) — compose-time hand-over segments (SUPERSEDED by red-team B-3: a compose-time segment double-fires when a role changes mid-segment and goes stale under Repeat; the hand-over is a Source-time decision — `data-shape.md` §5). The original reasoning:** when consecutive sections resolve to
different voices, `Compose` inserts `Segment{Key: "handoff:"+from+":"+to+":"+hash, Role: incoming}` whose text
comes from the script tree (`scripts/handover/line.txt`, inheriting from `global` — `script/script.go:9,47`):
user-overridable (R-5), testable without audio, cached like any segment, riding `pauseLast(reportPause)`
(`compose.go:82-93`). Cautions: `synth_test.go:400-402` pins the existing wording; and the *existing* gen-based
hand-over (`source.go:213`) must not double-fire when a user changes only the fire role while weather plays.

### A-2.4 Every place a voice name is spoken or displayed

Spoken: the tail (`compose.go:78,135-140`; `scripts/weather-radio/tail.txt` — *"This is {{.Voice}} for Watchpost
Weather Radio. You can change your correspondent voice in your Watchpost CLI application settings."*, fallback
"your correspondent" `:136-138`); `{{voice}}` late binding (`source.go:18`; audio `:309` from the render voice,
marquee `:291-294` from `s.current()` — **these diverge under per-segment voices**); the hand-over line; the
chooser sample (`compose.go:187-199`, `scripts/voice-preview/sample.txt`, `app/voices.go:144`). The **lead does
not name a voice** (`weather-radio/head.txt` names the station; `live.txt`/`span.txt`, `compose.go:117-132`;
same for `fire-report/head`, `seismic-report/head`, `global/head`) — the natural anchor for a per-role ident.

Displayed: stream name `app/radio.go:288`; `VoiceName()` `app/voices.go:163-171`, `Voices()` `:175-185`; the
sentinel `"System Voice"` (`app/voices.go:218`) vs the spoken "the System Voice" (`synth/voice.go:60-68`);
`voiceChip` `modes/tty/modal_chooser.go:69-74`, ✔ `:139-141`, footnote `:150`; the control
`[V] Voice: <chip>` `modes/tty/radio_panel.go:432`; `radioVoice` `modes/tty/dashboard.go:266,318,836`; the
failure hint *"check the voice in [V], or reinstall it"* `app/radio.go:505`; help `modes/tty/help_about.go:164`.

### A-2.5 Render-ahead and concurrency

Depth: one segment (`source.go:136`). From one Source **up to 3** concurrent `say`/`piper` (measured, red-team B-2: the hand-over pair `:245-250` beside the play goroutine's re-voice `:203` or the render loop `:174`). Process-wide today up to **6** (the suspended read still rendering counts too — `narrate.go:129` renders before `awaitAir` at `:134`): Source (2) + the narrator's `render` (`app/radio.go:395-407`) + a
preview (`app/voices.go:144`). **No semaphore caps subprocesses anywhere**; `installMu` (`app/radio.go:49`,
`app/voices.go:53-57`) serialises installs only. The brief's RS-6 "cap at 2 — today's budget" names a budget the
code does not enforce. With a composite cache key, a re-voice back to a used voice becomes a cache hit.

### A-2.6 Composition order, and where maritime slots (OQ-12)

lead notice (`compose.go:51`, `leadPause` 2 s `:23`) → lead span (`:52`) → conditions (`:53-55`) → alerts
(`:56-61`) → products HWO · SPS · NOW · ZFP (`:62-66`; `products.go:18-22`) → **fire** (`:69-72`, "UAT 114:
after the forecast, before the tail") → **seismic** (`:73-76`, "P4: after the fire report") → tail (`:77-78`,
`tailPause` 1 s). Air: `reportPause` 2 s, `tailPause` 1 s (`:82-93`). The comments are chronological, not a
hazard hierarchy. **Recommendation: maritime first of the three** — products → maritime → fire → seismic →
tail (`compose.go:69`) — the only order with precedent (the detail view reads marine and tides before FIRE and
SEISMIC, `README.md:60`; `modes/tty/detail_marine.go` precedes `detail_fire.go`) and marine reads as forecast,
not hazard. Plumbing mirrors fire/seismic: a hook field (`app/radio.go:43-44`), set in `attachRadio`
(`app/dashboard.go:256`), read at `app/radio.go:319-322`. `Compose` already takes **8 positional parameters**
(`compose.go:48`); maritime + a role resolver makes 10 → fold into a `Reports{Fire, Seismic, Maritime, Roles}`
struct in batch 1.

## A-3 The narration seam

### A-3.1 Call chains and the minimal seam

Breaking: `tickerDeck.cycle` (`app/ticker.go:219-222`) → `breaking` (`:248`, `audible` at `:254`) →
`narrator.Run` (`:258`, class `narrateBreaking`) → `admit` (`narrate.go:213-245`, duck `:240-243`) → `speaker`
(`:207`) → `s.attention()` (`ticker.go:259` → `narrate.go:113-118` → `radioDeck.tone` `app/radio.go:372-380`:
**`d.voice()` `:373`**, `AlertTone(v.Rate())` `:377`, `PreviewAside` `:378`) → `readBreaking` (`ticker.go:278-303`)
→ `s.line` (`:283` → `narrate.go:125-138` → `radioDeck.render` `app/radio.go:395-407`: **`d.voice()` `:398`**,
`AlertNarration` `:402`; `vizFor(class)` `narrate.go:133,49`; `awaitAir(play)` → `app/radio.go:409-415`).

Severe read: `eventReader.Read` (`app/severe_read.go:61-72`) → `run` (`:91`) → `r.row(key)` (`:98`) →
`eventScript` (`:102`, `:128-157`) → `narrator.Run` (`:103`, class `narrateRead`) → `s.line` (`:106`) → the same
render/play path; status overlay `:110-113`; `s.hold` `:114`. **The read sounds no tone today** (no
`attention()`), so R-12's tone before a severe read is new behaviour.

Seam options: (a) widen `narrationVoice` only (`tone(role)`, `render(ctx, role, text)`) — every `s.line` site
passes a role; **(b) role on the job** — `Run(ctx, class, role, audible, seq)`, `narrationJob.role`, the
`speaker` forwards it into `tone`/`render` — callers state the role once and `narrate.go` stays domain-free
(imports `narrate.go:22-26`) — **recommended**; (c) a per-job `synth.Voice` resolver closure — rejected, it
drags `synth` into `narrate.go`. Pair (b) with `radioDeck.voiceFor(role)` wrapping `voice()` (`app/voices.go:29-63`)
with the inheritance walk — one owner of resolution, which R-7's `[S]` line also reads.

Callers to touch: `app/ticker.go:258`; `app/severe_read.go:103`; `app/narrate.go:191` (nil-narrator speaker),
`:207`, `:113-118`, `:125-138`; `app/radio.go:372,395`; fakes `app/narrate_test.go:12-33,314-318`,
`app/ticker_test.go:67`.

### A-3.2 The tone

Constants `synth/tone.go:14-23` (1000 Hz · 3 pulses · 200 ms · 100 ms gaps · 2 s tail · 0.45 · 8 ms envelope);
`AlertTone(rate)` (`:29-57`) emits **stereo** with the 2 s pre-narration pause **inside the buffer** (`:56`) —
`s.hold(s.attention())` (`ticker.go:259`) relies on it, so a preset with a different tail changes the takeover's
pacing. Invoked **once per job, before the first line, breaking only** (`ticker.go:259`); a burst of eight events
gets one tone. Rate from `d.voice()` (`app/radio.go:373-379`) — **which takes the install path on a fresh Linux host (red-team B-4)**; the tone's rate becomes a constant (every voice is 22 050 Hz) and the tone never resolves a voice (FR-9).

The product class is available at `Run` time on both paths: breaking holds `[]globalfeed.Event` with `Class`,
`Severity`, `Type` (`domains/globalfeed/event.go:36-38,51-53`) and `tickerCategory` (`ticker.go:360-372`) is already
the Watch/Warning classifier; the severe read holds `tty.SevereRow{Product, Severity, Record}`
(`modes/tty/severe.go:65-75`) and `severe.ProductCode` (`domains/severe/codes.go:16-28`, used at
`severe_read.go:112`). **A shared classifier must take the product string** to serve both. Design question the
brief did not raise: the breaking tone fires before any event is identified and `byBreakingPriority`
(`ticker.go:329-336`) can mix classes in one burst — tone by the highest-severity event's class (recommended,
no pacing change) or move `attention()` into `readBreaking`'s loop (~2.7 s per event). → **OQ-13**.

### A-3.3 The arbiter is unchanged by roles

`narrationClass` is a 2-value iota (`narrate.go:28-33`), consumed at `:49,218,285,321`; `narrationJob{class,
audible, ctx, seq, suspended}` (`:68-74`); `speaker{n, job, ctx, sleep}` (`:102-107`). Nothing in
`admit/settle/release/first/unsuspend` reads a voice — a role is inert data consumed only by `render`/`tone`.
Preserve: `audible` is the caller's (silence + mute, `ticker.go:254`, `narrate.go:96`) — a resolution failure
must **fall back**, never flip `audible` (R-7); a maritime read is a broadcast section, not a narration —
**classes stay at 2**. `clip.rate` (`narrate.go:38-44`) + per-clip resampling already make two narration roles
at two rates work.

### A-3.4 Everything that assumes one voice — the `[V]` removal inventory (MVS-D-3)

Control row `modes/tty/radio_panel.go:432` (one of six segments in `radioControlLines` `:406-445`, wrapped by
`render.WrapSegments` `:435`, assembled at `:100` — removing a segment re-flows the wrap and moves every golden
frame). Chooser `modal_chooser.go:52-152`. Dashboard plumbing `modes/tty/dashboard.go:73-77` (hooks),
`:193-197,444` (`VoiceNoteMsg`), **`:229` the `V` binding** (the key OQ-10 frees), `:260-266` state, `:318`,
`:452-453`, `:573`, `:674`, `:722-723`, `:834-837`. Frame memo keys `modes/tty/memo.go:147-148,175-176`. View
`view.go:64-65,93`. Help `help_about.go:164`. App: `app/dashboard.go:86,256,258-261`; `app/voices.go:100-117`
(`SetVoice`), `:121-151` (`PreviewVoice` — keep `Audition` `:150`), `:163-185`; `app/radio.go:68-69,263,288,505`.
Config `platform/config/config.go:147`. Diagnostics `app/stats.go:101-102,108,190,228` — **no per-voice or
per-role row exists**; R-7's `[S]` line is new UI and the `Gauge{Name, Len, Bytes}` shape does not fit it.
Setup `modes/tty/setup.go:31-38,40-51,55-63,69-93,253-255,259-285,365-376` — **every question is on one screen
with a flat focus iota**; eight assignments plus five tones need a nested list or sub-focus (→ OQ-10 mock).

## Proposed data model — SUPERSEDED by `02-analysis/data-shape.md` (red-team A-1/D-1: two shapes were on the record; the typed per-namespace model is the one of record)

*Kept for the reasoning; do not build from this block.*

```toml
voice = "Samantha"              # unchanged — All reads
[radio]
mode = "synth"
[radio.voices]                  # "" = inherit the level above
alerts = "Rishi"    standard = ""
breaking = ""       severe_read = ""
weather = ""        maritime = ""      fire = ""      seismic = ""
[radio.tones]                   # MVS-D-26: mode = "" | "mute"; muted = [class keys] (empty under mute = every class); every class always sounds its own preset
warning = ""  watch = ""  advisory = ""  statement = ""  storm = ""
```

A **struct**, not a map (`VoiceRoles{Alerts, Standard, Breaking, SevereRead, Weather, Maritime, Fire, Seismic}`):
deterministic TOML round-trip for `savePreference` (`app/dashboard.go:273+`); a typo'd key is ignored per the
config contract (`config.go:136-137`) rather than minting a role; the field list *is* the role registry.
Resolution: one owner `Resolve(role) []string` → `[role, group, root]`, walked until a voice exists on this host
(`VoiceByName`/installed, `app/voices.go:67-78`), **recording which link won** (R-7's `[S]` line, M5's matrix).
Gap: `Load` validates only `cfg.Fire` (`config.go:197`, `:121-134`) — add `Radio.Validate()` at the same site.

## Tests that pin voice behaviour today

`domains/radio/synth/{synth,seismic,fire,soak,fuzz,composer,tone,install,normalize}_test.go` (incl. Compose
order/keys, `tail:Samantha`, the mid-segment hand-over, the rate refusal, `Remainder`, the `Handoff` wording,
render failure, stereo widening, voice-gone stop, lead/report pauses, hostile text, `Pronounce`);
`domains/radio/script/script_test.go`; `domains/radio/player/{player,alert,redteam,fuzz}_test.go`;
`app/{radio,narrate,ticker,severe_read,wiring,dump,radio_debuglog}_test.go`;
`modes/tty/{modal_chooser,radio_panel,dashboard,modal,uat_0827,golden,memo,bench}_test.go`.

## Risks the brief missed (fed into the risk register)

1. **`maxCached = 40` was sized for one voice** (`source.go:348`); voice-keyed, a four-role broadcast under
   Repeat: One thrashes — a CPU/latency regression, not the RSS one M4 pins. Scale by assigned-voice count or
   evict per voice.
2. **No role for the lead and the tail** — station identity; unassigned, the sign-off falls to the last
   section's correspondent. Add a `station` role (default: inherits *standard*) → **OQ-14**.
3. **The tail script's words go stale** — "change your correspondent voice in … settings" (singular; the
   settings move). Ship the built-in updated.
4. **Marquee/audio name divergence** for `{{voice}}` (`source.go:291-294` vs `:309`).
5. **No concurrency cap exists** (RS-6 assumed one); render-ahead + hand-over + narration + preview reach 4
   model loads today. A bounded worker before roles multiply the odds.
6. **Pre-install (R-9) has no hook and an ordering hazard** — installs are lazy inside `voice()`
   (`app/voices.go:58`) under `installMu`; `startSynth` must never run under `tuneMu` (`app/radio.go:263-265`);
   the progress channel `voiceNote`/`VoiceNoteMsg` (`app/voices.go:155-159`, `dashboard.go:193-197`) belongs to
   the chooser being deleted — Setup needs its own progress line or a 63 MB download goes invisible.
7. **Silent fallback already exists** (`piperSpec`, `app/voices.go:71-77` → `installed[0]` → catalogue
   default, no signal) and undermines M5 until made explicit.
8. **Maritime data is not in the deck's fetch set** — `radioDeck.segments` fetches only `KindObs` and
   `KindAlerts` (`app/radio.go:295-300`); fire/seismic arrive by injected closure; the landlocked case must skip
   silently (`seismic.go:33`); `Compose` carries one `imperial` bool and already has a unit exception
   (`seismic.go:72`, depth in km) — tides (feet) and currents (knots) bring their own.
9. **`Compose`'s signature** (8 params → 10) — the `Reports{}` refactor goes in batch 1.
10. **`Segment` is on the diagnostic path** (`app/radio.go:276-277`, `WATCHPOST_DEBUG_RADIO` line) — a role
    field changes the log shape; `app/radio_debuglog_test.go` may pin it.
11. **The engine stream name is fixed at start** and embeds the voice (`app/radio.go:288`) — the Radio UI
    (OQ-10) should take the on-air name from the segment in flight (`onSeg`), not from `StartSource`.
12. **Setup's flat single-screen form** does not fit a role tree plus tones — a UI-model change, not "a new
    group joins".
