# VALIDATE Report — Watchpost CLI 0.9.x

| Field | Value |
|---|---|
| Phase | VALIDATE (li-A2DH: REVIEW → **VALIDATE** → SHIP → REFLECT) · SEV-0 · HUM LEAD |
| Entry | REVIEW approved 2026-08-25; `main` published and `v0.9.0` released the same day |
| Scope | The public release as installed by a stranger: the Linux protocol on HUM LEAD's Arch/CachyOS laptop, the one-hour radio soak (M8) on the build machine, the release pipeline under real tags |
| Releases during VALIDATE | `v0.9.1` (Linux voice profiles), `v0.9.2` (chooser progress), `v0.9.3` (road abbreviations), `v0.9.4` (◆ in Location Details, README screenshots) — all fix-forward from validation findings |
| Verdict | **PASS** — two full hours of radio, the Linux protocol, and the release pipeline under real tags; the one open question (V6) did not reproduce over a second instrumented hour |

## Executive Summary

VALIDATE put 0.9.0 in front of the two things BUILD could not: a Linux machine that had never seen the
source, and an hour of continuous radio. The Linux run (Arch/CachyOS, UAT 122) installed from the
public `curl … | sh` line, ran the Setup window, played the synthesized broadcast with Piper, heard
the Fire and Hotspot report, and tuned a live relay — and surfaced three real gaps, each shipped the
same day: the voice chooser knew only the one installed voice (v0.9.1: a curated six-voice Piper
catalogue, downloaded on first use), the wait before a preview looked broken (v0.9.2: progress and
"loading" in the window), and road abbreviations were spelled out letter by letter (v0.9.3). The
release pipeline itself needed three CI-only fixes before the first tag published (a darwin-only test
expectation, `lint-watermark` on a detached tag checkout, a timing budget under the race detector) —
all on `main`, none affecting the binaries.

The soak (UAT 123) answers the question left open at BUILD (UAT 98): RSS oscillates between 142 and
221 MB over an hour of Synth on Repeat: Watchlist with **no trend**, CPU 1–6 %, ~30 threads — no
leak. The one loose end: the harness found the app still alive a minute after `q` at the end of that
hour (it left on SIGTERM); two direct reproductions (quit at 25 s and 45 s into the broadcast) exit
instantly, so it is not a plain mid-segment quit. A second full hour with pprof armed and a
dump-on-linger tail exited in 0 s on `q` — RSS 146–202 MB, no trend, CPU 2–8 %, 28–31 threads. V6 is
closed as a harness artefact of the first script's `expect eof` window.

## Evidence

### Linux protocol (UAT 122, Arch/CachyOS, HUM LEAD)

| Step | Result |
|---|---|
| Install (`curl -fsSL …/install.sh \| sh`) | PASS |
| First run → Setup window; watchlist fills | PASS |
| Radio Synth; Piper installs on first tune-in | PASS (install line brief on a fast link) |
| `[V]` voice profiles | v0.9.0 showed only the installed voice → v0.9.1 catalogue; other voices download and speak — PASS |
| Voice chooser wait explained | v0.9.2 — PASS |
| Fire and Hotspot report on air | PASS |
| `m` Nearest Relay (Oceanside, CA from Linux) | PASS |
| Narration | road abbreviations → v0.9.3 |
| Not exercised on the laptop | 80/200-col resize, `--json` exit codes, no-audio failure mode — covered by tests and the macOS run |

### M8 soak (UAT 123, macOS, pty 133×44, Synth on Repeat: Watchlist, 60 min)

RSS 144 → 221 (peak, 15 min) → 142 (55 min) → 194 MB; CPU 1.0–5.6 %; threads 21 → 29–32. Full table in
`04-development/b3-infra-ledger.md`. Verdict: no leak; the audio cache's 40-segment bound is the ceiling.

### Release pipeline

Four release runs for `v0.9.0` (three CI-only failures, then green), one each for `v0.9.1`–`v0.9.4`
(all green: verify → matrix → installer smoke test → publish). CI on `main` green on ubuntu and macos
after every push. The public install line verified end to end from a fresh directory on this Mac.

### Gates

`make verify` GREEN · P10 0 live · golangci-lint 0 · staticcheck 0 · `a2dh validate` 18/18 at every
tagged commit.

## Findings

| ID | Sev | Finding | Disposition |
|---|---|---|---|
| V1 | Important | Linux `[V]` listed only the installed voice; no way to add profiles | **Fixed** v0.9.1 (UAT 118) |
| V2 | Important | ~10 s silent wait between `p` and the preview on Linux | **Fixed** v0.9.2 (UAT 119) |
| V3 | Minor | "SWY", "Rd." read letter by letter | **Fixed** v0.9.3 (UAT 120) |
| V4 | Minor | FIRE rows in Location Details used `▲` while the row mark is `◆` | **Fixed** v0.9.4 |
| V5 | Minor | Release job failed on CI-only conditions (darwin test expectation, `lint-watermark` on a tag checkout, race-detector timing budget) | **Fixed** on `main` before `v0.9.0` published (UAT 117) |
| V6 | Minor | Process alive ~1 min after `q` at the end of the first hour soak; left on SIGTERM | **Not reproduced**: quits at 25 s / 45 s exit in 0 s; a second instrumented hour exits in 0 s (no dump). Closed as a harness artefact (UAT 123.2) |

## Conditions into SHIP

1. SHIP = the record: release checklist rows ticked (A4, B8, B9, C6 done at VALIDATE), this report
   approved, README/CHANGELOG already public.
2. Anything found after SHIP ships as `v0.9.x` fix-forward; `1.0.0` waits on the schema ratification
   and the 1.0 plan (§11.9).

## Source Documents

`04-development/b3-uat-log.md` sessions 117–123 · `04-development/b3-infra-ledger.md` (M8 soak) ·
`07-readiness/release-checklist.md` · `07-readiness/linux-validation-protocol.md` ·
`08-reports/review-report.md`.
