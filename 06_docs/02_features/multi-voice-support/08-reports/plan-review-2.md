# Plan review 2 — multi-voice-support (P1 · P2 · P3, delta) — disposition ledger

**Reviewed:** `265319c` as tasked, with the HEAD delta to `b4950e6` noted per row. The reviewer verified every
round-1 fix against the real tree (PR-1..10, 13–16, 18, 19, 23 fixed; PR-11/12/17/20/21/22 were not cited by any
task at the time — `plan-review-1.md` now records them). **Verdict:** REWORK (P2/P3; P0/P1 may start).
**Remediation commit:** `426eb1e`, then the round-1 red-team remediation (this ledger's "Superseded" rows).

| ID | Sev | Finding (short) | Disposition |
|---|---|---|---|
| PR2-1 | High | Task 2.11 has no code for `resident_test.go` / `app/resident.go` ("as v1") | **Fixed** (`c13f0a7` blocks restored) → **Superseded**: Task 2.11 is cut from 0.14.0 (red-team round 1, four lenses); the task records the ruling |
| PR2-2 | High | `app/radio_test.go` pins break after 2.7/2.10 (`voiceID`, `piperSpec`, `defaultVoice`) | **Fixed** — both tests rewritten in Task 2.7's "existing pins" block |
| PR2-3 | High | `PreviewVoice` still calls `piperSpec` | **Fixed** — `VoiceByName(cast.Resolve(cast.All, …).Spoken)`; `defaultVoice` deleted |
| PR2-4 | High | Duplicate `"From"` map key in `script_test.go` | **Fixed** — only `"To"` is added |
| PR2-5 | High | The parity test in `modes/tty` imports `synth` (`make lint-imports`) | **Fixed** — the test lives in `app/marine_test.go` |
| PR2-6 | Medium | `forecastPeriods` after `Normalize` never matches | **Fixed** — `ForecastPeriods` cuts the raw filtered text before `Normalize` (a 4-period test expects 3) |
| PR2-7 | Medium | `MarineZoneFor` keeps one id per header; ranges/lists lost; duplicates `ugc.go` | **Fixed** — `synth.UGCCodes` (wraps the existing splitter/expander); `MarineZoneFor(ctx, codes, lat, lon)` |
| PR2-8 | Medium | `installOnce` nil map; the mutex test has no temp `voiceDir` | **Fixed** — lazy init in `installOnce`; the test uses a temp `voiceDir` |
| PR2-9 | Medium | The maritime hook's `attachRadio` signature not shown | **Fixed** — the signature and the `dashboard.go:86` call in Task 3.4 |
| PR2-10 | Medium | Three tests exist only as prose (3.4's extension, 3.1's parity, 3.7's soak case) | **Fixed** — written as code; the soak's coastal `Compose` named in 3.7 |
| PR2-11 | Medium | `[M]` inert between P2 and P4 | **Fixed** (2.10's hook re-cast) → **Superseded**: the hook calls `deck.setTones` (no reload, no re-cast — round-1 Perf #8) |
| PR2-12 | Low | 2.9's dead placeholders (`_ = d`, `_ = muted`) | **Fixed** — deleted (round-1 Junior #15 swept the rest) |
| PR2-13 | Low | `Reports.Voice` dead after 2.1 | **Fixed** — the field and the parameter deleted in 2.1 (seven call sites) |
| PR2-14 | Low | The old `const maxCached = 40` undeleted; `handOver`'s dead `pw` | **Fixed** — deleted; `handOver` has no `pw` |
| PR2-15 | Low | Imports not stated (`tone.go`, `radio_test.go`, `marinezone_test.go`) | **Fixed** — stated per task; `ToneMemo`'s imports moot |
| PR2-16 | Low | "your correspondent" has no owner | **Fixed** — `spokenName` returns it for a nameless voice |
| PR2-17 | Low | 3.2 nits (`f` vs `f64`, `trimFloat`, spelling) | **Fixed** — `f64`; trailing zeros trimmed; one spelling |
| PR2-18 | Low | File maps / index drift | **Fixed** — the index regenerated from the batch maps (this remediation) |
| PR2-19 | Low | 1.5's row splice "to document" | **Fixed** — the row and the splice dropped |
| PR2-20 | Low | `castConfig` both a func and a method | **Fixed** — the method is `currentCast()` |

**Reviewer's P10 hygiene and coverage notes** stand: every function within 60 lines / 15 decisions at the time
(`play` at the edge — round 1 later reduced it by resolving once); no new mutable package state; FR/NFR coverage
traced; the by-design gaps (NFR-4 at UAT, FR-9's uninstalled-alert-voice PTY smoke at VALIDATE) carried to
`07-readiness/`.
