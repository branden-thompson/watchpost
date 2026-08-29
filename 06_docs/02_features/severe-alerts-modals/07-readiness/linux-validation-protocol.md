# Linux validation protocol — 0.13.0 (second machine)

Purpose: prove the release on a Linux machine with audio (Arch/CachyOS as for 0.9.0–0.12.0) — the halves
owed since 0.12.0 plus what 0.13.0 changed: the radio path (NFR-7 superseded). Budget ~30 min. Record every
step's outcome in the VALIDATE report.

## 0. Environment
`uname -a; echo $TERM; locale | grep LANG; pactl info | head -3` — glibc Linux, a Unicode font, ALSA/PipeWire.

## 1. Build and gates on the machine
```
make verify                      # the race suite on Linux — the half macOS cannot prove
make pty-severe                  # w → tabs → enter/esc/esc → w → esc → ctrl+s → esc → q, on the real binary
```

## 2. R6 — the radio (BLOCKING)
1. Tune a location on **Synth**: the cycle plays to the sign-off ("This is <voice> for Watchpost Weather
   Radio…") — with `WATCHPOST_DEBUG_RADIO=~/radio.log`, the log ends `segment key="tail:…"` then
   `cycle-end source-err=""`.
2. **Nearest Relay**: audio within 90 s; the marquee carries the stream title.
3. Open `w`, focus an event, `space`: the record is read; the panel shows `EVENT · …`; the ▶ marks the row.
4. During a read, force a takeover (a breaking event, or `M` unmute with one pending): the read PAUSES, the
   takeover speaks, the read RESUMES where it stopped; `+`/`-` during the read move the volume.
5. `V`: audition a voice while a read plays — the read keeps playing, the sample plays over it, a takeover
   pauses the read (not the sample).
6. Quit during a read: the app exits promptly (the read is cancelled and waited for).

## 3. The window
`w` on each tab; `enter` on a record; `esc` `esc`; `--ascii`: the chip row holds one line at 80 cols; colour
off (`NO_COLOR=1`): the alert module shows `[severity]`, the bands and titles in brackets.

## 4. Soak (RS-8)
`WATCHPOST_DEBUG_PPROF=1 watchpost` for 1 h with Repeat: Watchlist; `/debug/counters` every 10 min — RSS
without trend, goroutines flat, `severe index` rows ≤ 500.
