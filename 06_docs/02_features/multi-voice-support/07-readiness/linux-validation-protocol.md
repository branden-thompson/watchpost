# Linux validation protocol — multi-voice-support (0.14.0) — HUM LEAD, the Arch box

Linux is where Piper lives, so every Piper-shaped behaviour is validated here by the HUM LEAD; the agent's
macOS runs are the other half. R6 (the broadcast never stalls) is blocking on both. Fill the *Result* column;
anything not PASS is a VALIDATE finding.

## 1. Before 0.14.0 — the 0.13.0 baselines (once, on the tag `v0.13.0` — the one identifier for "the 0.13.0 tree")

| # | Measure | Method | Result |
|---|---|---|---|
| 1.1 | The tone path's code-path pin | `go test ./app -run '^$' -bench TimeToToneStart -count=10` on `v0.13.0` with `app/tone_latency_test.go` copied in (`perf-protocol.md` §1 item 1 — the release binary carries no instrument); median / p95 | |
| 1.2 | Whole-app RSS over an hour | `scripts/quality/soak.sh` (phase B) | |
| 1.3 | Summed `piper` RSS during a takeover-over-read, one voice (the 0.13.0 denominator of §4's comparison) | `ps -o rss= -C piper` sampled every second through one takeover over a read; sum and max | |

## 2. 0.14.0 — the batch builds

Precondition for every row that names a pair: `[radio] cast = "cast"` in the file — the pairs are read only in
cast mode (MVS-D-25).

| # | Check | Batch | Result |
|---|---|---|---|
| 2.0 | Per-Piper-process RSS (MVS-D-17; 0.15.0's resident decision input) | `perf-protocol.md` §3 steps 1–6, on this build | P2 | |
| 2.1 | Fresh HOME, no voice installed: tune → the root voice installs with progress as 0.13.0 did (UAT 118); the broadcast starts | P2 | |
| 2.2 | Assign `[radio.voices.alerts] piper = "en_GB-alan-medium"` (not installed): a takeover sounds its tone **at once** and reads in the fallback voice; `[S]` says *installing*; the install lands in the background; the next takeover reads in Alan | P2 | |
| 2.3 | A cast with two voices, **first cycle after a tune**: the fire report hands over by name and back with no audible gap at either boundary (the lines are pre-warmed — `perf-protocol.md` §1's inequality); the sign-off names whoever reaches it; the `synth.pcm.cache` gauge stays under 40 MB | P2 | |
| 2.4 | Recast while a segment plays (Setup save): the running segment hands over at the spot reached — one render, no repeat | P2 | |
| 2.5 | Both ordinary slots busy + a takeover: the takeover renders in the reserved slot within the bound; the marquee explains a preview that waits | P2 | |
| 2.6 | Coastal tune (Oceanside): the maritime report reads between the forecast and the fire report; the coastal forecast is three periods (RAT-6, E-7 pending); the one-hour soak on this cycle holds R6 | P3 | |
| 2.7 | Each class tone once per class (`[space]` on a Warning / Watch / Advisory / Statement / Storm / a significant quake); `[M]` silences the tones and the words still read | P3 | |
| 2.8 | Setup at 80×24 and 133×44 in a Linux terminal: the pickers list Piper voices by **name**; a first `p` on an uninstalled voice offers the download, the second previews (ducked); the "not installed — downloads on Save" note sits under the picker; save round-trips to `piper = "<key>"` | P4 | |
| 2.9 | `--ascii`: the Setup window and the three Radio breakpoints draw with `[ ]`/`[x]`, `o`/`*`, `>`, `v`, `\|` | P4 | |
| 2.10 | The tone path: the benchmark on the 0.14.0 tree ≤ row 1.1 + 50 % (the pin); the live budget — `WATCHPOST_DEBUG_RADIO=<path>`, a takeover, `breaking:` → `tone:` ≤ 250 ms with the root installed **and** on a fresh HOME with nothing installed (`perf-protocol.md` §1 item 2) | P4 (release build) | |
| 2.11 | Over an hour with two distinct Piper voices assigned and a takeover-over-read overlap: the **app's** RSS (`ps -o rss= -C watchpost`) ≤ 126 MB, no trend; the **summed `piper`** RSS during the overlap ≤ row 1.3 × 1.10 and ≤ (N + 2) × row 2.0's per-process number; ≤ N + 2 `piper` processes at once (`perf-protocol.md` §4) | P4 (release build) | |
| 2.12 | Disk: `~/.cache/watchpost/piper/voices` after two assignments — one model per assigned voice, nothing duplicated (NFR-4) | P4 | |
| 2.13 | The one legitimately silent row (M5, AM-19): fresh HOME, network **off**, `cast = "cast"` with a Piper pair, a takeover → the tone sounds, no words, `[S]` reads *installing* then *install failed — not before hh:mm*, the marquee shows the words | P4 | |

## 3. Release build

| # | Check | Result |
|---|---|---|
| 3.1 | `make verify` green on the Arch box (the ubuntu CI leg is the second witness; `make p10` runs where the framework build lives) | |
| 3.2 | `HOME=$(mktemp -d) expect scripts/quality/validate-journey.expect dist/journey.log`: the cast step ≤ 11 keypresses (AM-18) | |
| 3.3 | M3 ear test, both arms, vs the 0.13.0 binary (`gates.md` §2) — recorded in `07-readiness/validate/m3.md` | |
