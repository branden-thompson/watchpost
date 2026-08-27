# Q7 validate log — Proof and the baseline document

**Batch:** Q7 of the plan of record v3 (§3 Q7; VALIDATE phase).
**Approval:** "Go for 0.10.0 release, Q7 validate" (2026-08-27).
**Status:** IN PROGRESS — the 7-day macOS soak runs 2026-08-27 09:01 UTC → 2026-09-03 09:01 UTC; the
Arch 72 h run is HUM LEAD's; the baseline document fills as the numbers land.

## 1. What Q7 is

Q7 changes no product behaviour. It runs the proof the plan defined — a 7-day soak on macOS under the
phase schedule of `07-readiness/soak-profile.md` and a 72 h soak on Arch — and writes the two documents
that turn the pass into a corpus: `06-key_learnings/quality-baseline.md` (every §1 number with its
command and source) and `06-key_learnings/reading-profiles-and-soak-logs.md` (the junior walkthrough).

## 2. Done so far (2026-08-27)

| Item | State |
|---|---|
| `v0.10.0` released (Q6) | tag `v0.10.0`, release green, 8 assets |
| Warm launch × 10 (`WATCHPOST_DEBUG_TIMING=1`, 133×44, cache primed) | median **790 ms** (550, 600, 690, 780, 780, 800, 810, 920, 970, 1000) — above the ≤ 550 ms line; taken while the Q0 24 h soak shared the machine and the cache dir. **Re-taken after that soak ends** (§5). What the timer measures: the first *fully-populated* snapshot, which needs every favourite's observation and alerts from the network (alerts never cache) — network round trips, not render. |
| Q0 24 h soak (the Q0-era binary, launched 2026-08-26 15:55 UTC) | 17 h in: `tools/slope` → **GROWTH** (+25.5 MB/day, 95 % CI [+2.5, +48.5], plateau 49.8 MB). Attributed with `pprof -base` across its hourly dumps: the growth is `hms.Parse` (the inflated archive, the XML decoder's copies, one string per point) and the post-GC heap steps **56.0 → 57.1 → 61.0 MB** exactly when `hms.memo.points` steps 77k → 81k → 95k, flat between steps: a **data-driven plateau shift bounded by the archive**, not an unbounded term. With the warm-up moved past the last step: +0.4 MB/day, CI [−15, +16], UNCERTIFIABLE (one day cannot certify the 30-day bar — as R2-1 said). Q3 already retired every site named (streaming walk, no inflated copy, interned strings). **"No site rising over ≥ 4 consecutive dumps" (plan §1):** `pprof -base` from the 13.1 h dump to the 17.1 h dump (four hourly dumps after the last archive step) nets **−3.1 MB**; the largest mover either way is the disk-cache file read buffer (−2.4 MB), the largest riser `Assembler.Apply` at +1.0 MB (a snapshot's worth); post-GC heap 61.0 → 61.6 MB; goroutines 277 → 278; threads 23 flat; fds 17–19; every gauge flat (`nws.gridinfo` 56, `httpx.neg.entries` 5–7, disk 812–819 files). **Passes.** Final numbers at the run's end (§5). |
| **Instrument defect found and fixed** | `/debug/counters` read the memory rows *without* a GC; only the dump did. Every 5-minute sample the soaks took since Q0 was a pre-GC reading (the CSV showed 75–98 MB where the same hour's dump showed 57–61 MB); the hourly dumps were the post-GC truth. Fixed in `0109256` (`record()` collects first; test `TestCountersRecordRunsAGCFirst`); CHANGELOG `[0.10.1]`. **Release `0.10.1` approved by HUM LEAD 2026-08-27** ("0.10.1 release approved"); the 7-day soak's binary (`0109256`) is that tag's tree. |
| 7-day macOS soak | **running**: PID 39249, port 6064, `dist/watchpost` built from `0109256` (= `v0.10.0` + the counters fix — the profile says "the batch's tag"; deviation recorded, the tag follows on approval), 133×44, 5-minute samples, hourly dumps, phases A–F driven by `scripts/quality/soak-phases.expect` (A idle 0–2 h → B synth Repeat: Watchlist volume 5 2–26 h → C viz 26–30 h → D Nearest Relay 30–54 h → E storm: radio off, `http/` emptied at 54 h → F settle 56–72 h → idle to day 7). `soak-phases.log` beside the CSV. First attempt (08:52 UTC) discarded after 9 min: the sampler had not attached and the binary predated the fix. |
| `scripts/quality/soak-phases.expect` | the driver, committed, platform-neutral (cache dir argument) — the Arch run uses the same script |

## 3. HUM LEAD's half (Arch, 72 h) — the exact steps

```
git checkout v0.10.1
make build
WATCHPOST_LIVE=1 go test ./app -run LiveRelay -v      # the Linux half of the relay proof (RP-10)
scripts/quality/soak-phases.expect soak-phases.log "$HOME/.cache/watchpost/http" &   # 133×44 terminal, TERM=xterm-256color
sleep 15; pid=$(lsof -nP -iTCP:6064 -sTCP:LISTEN -t | head -1)
WATCHPOST_DEBUG_PPROF_ADDR=127.0.0.1:6064 scripts/quality/soak.sh "$pid" 72 soak.csv 300
go run ./tools/slope -in soak.csv            # after 72 h
```

Send back `soak.csv`, `soak-phases.log` and the `profiles/` dumps; they go into `02-analysis/q7-arch/`.

## 4. Freshness table (§0.3)

None — Q7 changes no cadence or lifetime.

## 5. Pending

- [ ] Q0 24 h soak ends 2026-08-27 ~15:55 UTC → final `slope` verdict, `pprof -base` over ≥ 4
  consecutive dumps ("no site rising"), the per-counter table at 24 h → baseline document.
- [ ] Warm launch re-taken × 10 with the machine quiet.
- [ ] 7-day soak: phase reports at 24 / 48 / 72 h (threads must return to phase A's count after the
  storm; every counter flat against its bound), the 30-day extrapolation table, the `slope` verdict
  on day 7 (floor ≈ 3.8 σ).
- [ ] Arch 72 h + relay proof (HUM LEAD).
- [ ] `quality-baseline.md` complete; VALIDATE report (`08-reports/validate-report.md`); DEBRIEF.
- [x] Release `0.10.1` (approved 2026-08-27; published).

## 6. Deviations recorded

1. The 7-day soak runs a binary built from `0109256`, one commit past `v0.10.0` (the counters fix),
   because the statistic's input was wrong without it. The tag catches up on approval.
2. The Q0 24 h soak's CSV series is pre-GC; its verdicts are read from the hourly dumps (post-GC)
   and the CSV is kept as the record of the defect.
