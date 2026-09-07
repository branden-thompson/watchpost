# P0 build log — measure first (multi-voice-support, 0.14.0)

```
Batch:   P0 — the two "before" numbers, recorded before any 0.14.0 change
Date:    2026-08-30
Branch:  feature/multi-voice-support
Method:  07-readiness/perf-protocol.md §1 item 1 (time-to-tone-start) and §2 (Setup allocations) — the one owner
Host:    darwin/arm64, Apple M3 Pro (12 threads). The Linux twin of Task 0.1 is the HUM LEAD's
         (07-readiness/linux-validation-protocol.md row 1.1)
```

## Task 0.1 — the tone path's code-path pin on the 0.13.0 tree ✅

**The instrument.** `app/tone_latency_test.go` is written once, here, exactly as P2 Task 2.9b will land it in
0.14.0. It reports **`ns/tone`**: the wall-clock nanoseconds from `tickerDeck.breaking()`'s entry to the moment
the narrator asks for the attention tone. A fake `narrationVoice` (`toneStampVoice`) stamps `tone()` and cancels
the sequence's context at that instant, so nothing after the tone is timed and no iteration waits out a hold.
`ns/op` is **not** the number of record — it includes the teardown after the stamp.

The fake sits at the `narrationVoice` seam, not below the deck, because `radioDeck.engine` is a concrete
`*player.Engine`. That is what the protocol's "it measures Go — the Director's admission" means in practice.

**How it was run.**

```
git worktree add <scratchpad>/p0 v0.13.0        # baec8b0
cp app/tone_latency_test.go <scratchpad>/p0/app/
cd <scratchpad>/p0 && go build ./...            # compiles unchanged on the 0.13.0 tree
go test ./app -run '^$' -bench TimeToToneStart -count=10
git worktree remove --force <scratchpad>/p0     # tree left clean
```

**The ten `ns/tone` values** (in run order): 1035 · 1012 · 1021 · 1044 · 1034 · 1016 · 1024 · 995.4 · 1005 · 1008.

| Statistic | 0.13.0 (`v0.13.0` = `baec8b0`) |
|---|---|
| **median** | **1,018.5 ns** |
| **p95** (nearest-rank) | **1,044 ns** |
| mean | 1,019.4 ns |
| spread (max − min) | 48.6 ns (4.8 % of the median) |

**Reading it.** The 0.13.0 admission path costs about **one microsecond** of Go before the tone is asked for.
The spread across ten runs is under 5 %, so the instrument is stable enough for the M4 pass rule (0.14.0 median
≤ 0.13.0 median + 50 % → **≤ 1,527.75 ns**). That is a wide gate on a small number, which is deliberate: this
pin protects the *path*, not the audio. The audio budget is the separate absolute one — `breaking:` → `tone:`
≤ 250 ms — and it has no 0.13.0 counterpart because the release binary carries no such log line.

**What it does not measure.** `d.voice()` — 0.13.0's `tone()` resolves a voice and can therefore wait on an
install. That wait is real and is exactly what FR-9's find-only path removes, but it lives below the fake and
cannot appear in this number on either tree. It is the live budget's business, and the fresh-Linux-HOME row of
the Linux protocol is where it is observed.

## Task 0.2 — the Setup-open allocation numbers, in the tree ✅

**The instrument.** `modes/tty/bench_test.go` gains `setupBench` (the fixture dashboard with `openSetup`), the
benchmarks `BenchmarkSetup_133x44` / `BenchmarkSetup_80x24` / `BenchmarkSetup_133x44_Miss`, and
`TestSetupAllocBudget` with `setupAllocBudget` — all in the same shape as the existing frame and severe pins, in
the file that already owns them.

| Size | memo **hit** | memo **miss** | pin (×1.05) |
|---|---|---|---|
| 133×44 | **3,046** | **3,476** | 3,198 / 3,650 |
| 80×24 | **1,978** | **2,478** | 2,078 / 2,602 |

`go test ./modes/tty -run SetupAllocBudget -v` — **PASS**. The test is picked up by `make alloc-budget`
(`go test -count=1 -run 'AllocBudget$' ./...`), which is green, and by CI's alloc-budget step.

**Against the PLAN probe.** The probe predicted 133×44 hit 3 046 / miss 3 476 and 80×24 hit 1 979 / miss 2 478.
Three of the four matched exactly; the 80×24 hit measured **one allocation lower** (1,978). The pin is set from
the **measurement**, not the probe — the plan's own rule that a pin is always a measurement, never a formula.

**What these numbers are for.** They are the **informational "before"**, not the release pin. The window is
about to grow two groups (ALERTS - TONE and WATCHPOST RADIO - CORRESPONDENTS); P4 Task 4.11 re-measures the new
window and re-pins these four values. Recording them now is what makes that re-pin a comparison rather than a
guess — and the 133×44 miss (3,476) is the number to watch, because a keypress inside the form is the path the
new pickers add work to.

## Files touched

```
CREATE: app/tone_latency_test.go            — the time-to-tone-start instrument (also P2 Task 2.9b's file)
MODIFY: modes/tty/bench_test.go             — setupBench, three Setup benchmarks, setupAllocBudget, TestSetupAllocBudget
MODIFY: 07-readiness/perf-protocol.md       — §0 baselines and §2 filled with the measured numbers
CREATE: 04-development/p0-build-log.md      — this file
```

No production code changed in P0. `git status` clean at the commit; the `v0.13.0` worktree removed and pruned.

## Deviations from the plan

None. Both tasks ran as written.

## Notes carried to P1/P2

1. **`app/tone_latency_test.go` is now the shared file.** P2 Task 2.9b must keep its shape rather than rewrite
   it, so the two trees measure the same thing. When 0.14.0's `narrationVoice` changes (`tone(class)`,
   `render(role, text)` — the Station Director rename), the twin for the 0.13.0 tree is the version committed
   here, unchanged. Both are preserved by this commit.
2. **The instrument is blind to the install wait by construction** (above). Do not let the M4 pass rule be read
   as covering it; the live budget and the Linux fresh-HOME row are the only places that wait is visible.
3. **The 80×24 Setup form already overflows** (RS-19). These numbers describe the form as it renders today,
   overflow included; P4's re-measure is of the fixed form, so the two are not strictly like-for-like and the
   build log there should say so.
