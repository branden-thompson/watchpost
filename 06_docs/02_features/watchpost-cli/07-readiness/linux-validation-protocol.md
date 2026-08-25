# Linux validation protocol (second machine)

Purpose: prove the 0.9.0 release on a machine that has never seen the source — install, first run,
dashboard, radio with audio, the fire report, and the numbers. Record every step's outcome in the
VALIDATE report. Budget: ~30 minutes. Bring: the laptop, a terminal with a Unicode-capable font
(Nerd Font not required — the glyphs are plain Unicode), speakers or headphones, network.

## 0. Before you start (capture the environment)

```
uname -a; head -2 /etc/os-release; echo $TERM; locale | grep LANG; fc-list | wc -l
pactl info 2>/dev/null | head -3 || echo "no pulse/pipewire"; aplay -l 2>/dev/null | head -3 || echo "no alsa"
ldd --version | head -1
# Arch / CachyOS: the binary needs glibc (present) and ALSA for audio — pipewire-alsa routes it through PipeWire
pacman -Q glibc alsa-lib pipewire-alsa 2>/dev/null || dpkg -l libasound2 2>/dev/null | tail -1
```
Supported: any glibc Linux on amd64/arm64 — Debian/Ubuntu, Fedora, Arch/CachyOS. Not Alpine/musl.
Audio: the player loads `libasound.so.2` at runtime (ALSA; PipeWire/Pulse via their ALSA plugin) —
without it the player reads FAILED with the reason and the dashboard keeps running.

## 1. Install (the thing that must work)

```
curl -fsSL https://raw.githubusercontent.com/branden-thompson/watchpost/main/scripts/install.sh | sh
watchpost --version                       # → watchpost version 0.9.0
```
Expect: asset picked for `linux-amd64|arm64`, `checksums.txt` fetched, SHA-256 verified, installed
to `~/.local/bin` (or `/usr/local/bin` when writable and `~/.local/bin` is off PATH), PATH advice
if needed. Then the pinned form once:

```
WATCHPOST_VERSION=v0.9.0 sh -c "$(curl -fsSL https://raw.githubusercontent.com/branden-thompson/watchpost/main/scripts/install.sh)"
```
Record any deviation verbatim (the script prints `install: …` on every failure).

## 2. First run — the Setup window

- `watchpost` with no config → the dashboard opens with the **Setup** window over it (both
  questions on screen, `›` on the focused one).
- Type a city (type-ahead hints; `↑↓` pick), `enter` → the location reads `Chosen: …` and the
  focus moves to the FIRMS key line; leave it empty, `enter` saves. The row appears in the
  watchlist and fills within seconds.
- `s` reopens it: `Current: <your city> — [enter] keeps it`; `esc` cancels.
- Record: `ls -la ~/.config/watchpost` (dir 0700, `config.toml` 0600) and `cat` it.

## 3. Dashboard (10 minutes)

Launch `WATCHPOST_DEBUG_TIMING=1 watchpost`; on quit it prints the launch→full-view time. Check:
- The two-line masthead: title · centred `Updated:` stamp · `API: ✔n ⚠n ✘n / n  [S] Status`, then
  `[s] Setup [a] About [t] Theme [?] Help [q] Quit`. Resize to 80×24 and 200×60 (`resize` or the
  window): no horizontal overflow; the stamp shortens before anything wraps.
- Rows fill in seconds; `n/a` clears after rehydration; a location near a wildfire shows the
  orange `n◆` mark (try `l` → `Mineral Wells, TX` or `Grand View, ID`).
- Glyphs render: `› ▶ ∞ ◆ ⚠ ♪ ━ ░ █ ✔ ✘ ↗ ↘ ▲ ▼ º`. Note any tofu (□) and the font in use.
  There is no ASCII fallback in 0.9.0 — a font without these glyphs is a finding, not a setting.
- `enter` → Location Details: forecast, marine (coastal only), the **FIRE** section, alerts; the
  chip row `[↑↓] Scroll [esc] Close [ctrl+a] + Watchlist [shift+del] − Watchlist`.
- `l` lookup `92020`, `q`, relaunch → El Cajon is the top RECENT row (persistence). `ctrl+a` on
  it → it moves to the watchlist and leaves RECENT (never in both).
- `t` theme swap (try Synthwave '84 — the title gradient follows), `?` help (row-marks legend),
  `a` About, `A` alert details, `S` API status.
- Threads/memory while it runs (another terminal): `ps -o nlwp,rss,pcpu -p $(pgrep -x watchpost)`
  at 10 s and 60 s.

## 4. Radio (the Linux unknowns)

- `space` on a favourite with Mode: Synth. **First tune-in installs Piper** (~90 MB: 26 MB piper
  + 63 MB `en_US-lessac-medium`; 15-minute ceiling): the player shows `installing the Piper
  voice… NN%`, then the voice speaks within a few seconds. Record the install time and
  `ls -la ~/.cache/watchpost/piper/piper ~/.cache/watchpost/piper/voices`.
- The lead names the covering NWR transmitter, its frequency read digit by digit, then the
  forecast, then (when fire data is known) the **Fire and Hotspot report**, then the sign-off;
  two seconds of air between reports, one before the sign-off. The marquee follows the voice;
  `v` shows the bars; `+`/`-` work.
- `V` lists six correspondents (Lessac, Amy, Ryan, Joe, Alan, Alba) whether or not they are
  installed; `p` previews, `enter` picks. Picking Amy downloads her (~63 MB) with
  `installing Amy voice… NN%` in the player, then the broadcast hands over mid-sentence. Record
  `ls ~/.cache/watchpost/piper/voices` afterwards (two files per voice).
- `m` → Nearest Relay → a live relay plays (LIVE RADIO), or Synth with the reason line if none.
- `r` twice → Watchlist; wait for a cycle to end → the next favourite tunes (row mark moves).
- Stop, then `q` → the process exits at once (no lingering audio thread).
- Failure modes to try: no output device → the player reads FAILED with
  `audio output: no audio output device — check the system sound settings (…)` and the dashboard
  keeps running (a device plugged in later needs a restart); kill the network mid-broadcast →
  Synth continues from cache, a relay reconnects with backoff and fails over to Synth.

## 5. Report

```
watchpost report "El Cajon, CA"; echo "exit $?"          # plain text; 0, or 2 if a provider is degraded
watchpost report 92020 --json | head -20; echo "exit $?"
watchpost schema | head; echo "exit $?"
```
Expect the `fire:` line (`no hotspots nearby` or a count) and `provider firms: off` without a key.

## 6. Record

Fill the VALIDATE report: environment, install transcript, timing number, thread/RSS samples,
every glyph or audio deviation, and a verdict per section (PASS / FAIL + evidence). Anything that
fails is a `v0.9.x` fix-forward, not a reason to skip the report.
