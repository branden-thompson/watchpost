# Debugging ledger — B3/B4

The investigations that took more than a look. Each entry: symptom → how it was found → cause →
fix → what pins it. Full context lives in the UAT log session cited.

| # | Symptom | Found by | Cause | Fix | Pinned by |
|---|---------|----------|-------|-----|-----------|
| D1 (UAT 73–74) | 177 threads / 95 MB in `btop` at launch; "82–91 threads, not the 15 the tests say" | A headless probe *pre-warmed its own cache* and hid the problem; instrumenting the **real** TUI process with pprof (`WATCHPOST_DEBUG_PPROF=1`, `127.0.0.1:6060`) showed the truth | `time.LoadLocation` under the assembler lock on every publish (200 publishes at launch) + 200 simultaneous cache-file syscalls | `platform/tz` memo; publishes coalesced to one snapshot per 50 ms; disk reads bounded to 4; recent schedulers staggered 10 ms apart | infra ledger baselines; 144 → 20 threads; pty measurement 23 threads / 78 MB (2026-08-25) |
| D2 (UAT 85) | `[V]` chooser froze the UI | The chooser ran `say -v ?` on every render | Subprocess on the update loop | Background discovery at startup; the modal snapshots the list once when opened | `TestVoiceChooserUAT84` + the "never on a key press, never on a render" rule at the call site |
| D3 (UAT 83) | Marquee "stuck"; synth auto-repeated | Reading `Source` | Marquee was static truncation; the source looped unconditionally | Speech-paced marquee (`Spoken` duration rides `RadioStatusMsg`); `Source.Loop` | `TestMarqueeFollowsTheVoiceAndRepeatWiresThrough`, `TestSourceEndsAfterOneCycleUnlessRepeating` |
| D4 (UAT 81–82) | "IN EFFECT" read as "Indiana effect"; "442 PM" as "four hundred forty-two"; LAT…LON coordinates narrated | Listening | State expansion without context; no clock rule; polygon blocks kept | Context rules + ambiguous-state set; `clockTime`; polygon rows dropped | `TestNarrationRulesUAT81`, normalize goldens |
| D5 (UAT 94) | Tail played in the **old** voice ~1 run in 10 after a hand-over | `go test -count=40` on the new hand-over test | A render that started before `SetVoice` finished after it and stored old-voice audio in the freshly cleared cache | Renders capture voice + generation; cache only if unchanged; callers tag audio with the producing generation | `TestSetVoiceHandsOverMidSegmentAtTheSameSpot` (40× clean) |
| D6 (UAT 76–77) | `GOOS=linux CGO_ENABLED=0` build broke once oto was imported; Piper macOS archive had no dylibs | Cross-compile in the gate | oto v3.4.1 needs cgo on Linux | Pin oto v3.5.0-alpha.11 (pure Go on Linux); macOS uses `say` | `make release-matrix` in the exit gate |
| D7 (UAT 59) | Carlsbad `UNKNOWN n/a` for minutes | Live API probe | A transient NWS 5xx waited for the next 30-min cadence | Retry-before-cadence (10/20/40 s) per unserved location | `sched` tests |
| D8 (B3, gates) | Three commits landed with a failing test or live P10 finding | Reading the commit outputs | `a && b; c` / `\|\|` shell chains kept going past a red gate | Procedural: each gate as its own step, read, then commit | key learnings §5 |

| D9 (UAT 98) | Wide-terminal synth "pauses + CPU ramp" report; the first pty runs showed CONNECTING forever at 0 % CPU | Goroutine dump via `WATCHPOST_DEBUG_PPROF=1` | The harness: `expect` drains the pty only inside `expect`; a `sleep` while the radio redraws fills the buffer and the app blocks in bubbletea's renderer on the terminal write | Wait with `expect -timeout N` (drains) instead of `sleep`; the re-run showed PLAYING and a few percent of CPU at 400 cols | `synth2.expect` pattern kept in this ledger; UAT 98 |
| D10 (UAT 98) | RSS +35 MB over 90 s of synth | The corrected measurement | 64 cached **stereo** segments (~1.3 MB each) | Cache mono, widen at write, cap 24 | re-measurement in the infra ledger |

How to reproduce the measurement in D1 without a human at the keyboard: `expect` drives the real
binary in a pty at 133×44 (`stty rows 44 cols 133 < $spawn_out(slave,name)`), samples
`ps -o rss= -p PID` and `ps -M -p PID | wc -l` every 5 s, then sends `q`; the M1 line prints on exit
with `WATCHPOST_DEBUG_TIMING=1`.
