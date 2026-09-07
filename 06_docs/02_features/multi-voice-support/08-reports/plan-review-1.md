# Plan review 1 — multi-voice-support (P1 · P2 · P3 + index) — disposition ledger

**Reviewed:** `d6c2f94` (P1 v1, P2 v1, P3 v1, the index) against `objectives.md`, `data-shape.md`, `plan.md` and the
real tree; go-toml and `render.Plain` behaviour probed in a throwaway worktree. **Verdict:** REWORK.
**Remediation commit:** `265319c` (P2 rewritten as v2; P1/P3 patched), verified by plan review 2 (`plan-review-2.md`).
**Disposition vocabulary:** Fixed · Declined (with the reason) · Deferred (with the owner) · Superseded (a later change removed the site).

| ID | Sev | Finding (short) | Disposition |
|---|---|---|---|
| PR-1 | Critical | P2 cannot build: `AlertTone` arity; `Preset` does not exist until P3 | **Fixed** — P2 Task 2.0 lands `Preset`, `ToneRate`, `PresetByName`, `AlertTone(p, rate)`, `Classic()`; P3 3.6 adds the other four |
| PR-2 | Critical | `Source.render` return arity contradicts itself | **Fixed** (v2: four values) → **Superseded** by round-1 red-team: `render` returns `renderedSeg` |
| PR-3 | Critical | The default hand-over names the wrong correspondent | **Fixed** (v2 closure) → **Superseded**: one order `(from, to)` everywhere, `builtinHandoff` |
| PR-4 | Critical | `ResidentPiper.Say` nil-dereference after cancel | **Fixed** (v2 `sc := r.paths`) → **Superseded**: the resident backend is cut from 0.14.0 (red-team round 1) |
| PR-5 | High | `recVoice` is not what the plan assumed; the one-Say test could not reach the mid-segment path | **Fixed** — `markVoice{msPerChar}` for the timing tests; `recVoice` gains `name`; helpers `spokeLine`/`count` written as code |
| PR-6 | High | Existing tests the tasks break but never list (a–i) | **Fixed** — every task carries an "existing pins to update" block with the new values as code |
| PR-7 | High | Helpers/identifiers assumed to exist do not (`newTestEngine`, `fakeVoice`, `contains`, `testClient`, `Fragment{Sections}`, `SevWatch`…) | **Fixed** — each helper defined in the task that first uses it; `firstOf` lifted to `platform/render` (3.1); 3.4's test rewritten on `PerLocation`; 2.9 on `SevRed`/`SevYellow` |
| PR-8 | High | Find-only over-applied to the broadcast path — Linux first-run regression | **Fixed** — the root's tune path keeps its blocking install; only the Director's paths and the per-role resolver are find-only; two tests added |
| PR-9 | High | Task 3.6 contradicts itself on the memo (P10-06) | **Fixed** (presets as functions; the memo on the deck) → the memo itself **Superseded**: dropped by round-1 red-team (Code #14), `AlertTone` is called per takeover |
| PR-10 | High | `TestSevereReadSoundsItsClassToneUnlessMuted` is a race | **Fixed** — `waitUntil` for the read to reach the air; `newTestReader` in full |
| PR-11 | Medium | Task 1.5 test and code disagree on the message | **Fixed** — the label form (`Role.String()`), one expectation |
| PR-12 | Medium | Task 3.2's tide-next string cannot be produced | **Fixed** — `tide-next` takes a bare height for the high, `levelWords` only for a below-mark low; the expectation matches |
| PR-13 | Medium | `MarineSegments` over the complexity budget | **Fixed** — `observationSentences` split out |
| PR-14 | Medium | Promised verifications missing (three fixtures, three tests, NFR-6 at `spokenFor`/`announce`) | **Fixed** — fixtures `0.14.0-macos-only`, `0.14.0-piper-only`, `hostile-name.toml`; the three tests; `spokenName` (PlainLine + 48 runes) at one seam |
| PR-15 | Medium | `resolveVoice` spawns an install goroutine per call | **Fixed** — `installOnce` with an in-flight set |
| PR-16 | Medium | Prose-only or placeholder code where full code is required | **Fixed** — each written as code (Director struct/constructor verbatim, `appCtx`, `attachRadio` signature, the fire/seismic edits, the rewritten `TestReportsAreSeparatedByAir`) |
| PR-17 | Medium | `MarineZoneFor` tie claim not implemented; MultiPolygon left open; `testClient` | **Fixed** — `firstRing` decodes Polygon and MultiPolygon; the tie claim dropped; `newProvider(t, base)` |
| PR-18 | Medium | plan.md §2.2 `ToneFor` in cast; `Classify` departs from §2.5 | **Fixed** — plan.md amended to `ToneName`; the HLS → Statement rule stated (RAT-3 area; revisit at UAT) |
| PR-19 | Medium | `PreviewVoice` outside the Limiter | **Fixed** — `d.limited(v)` for previews |
| PR-20 | Low | Limiter tests timing-based | **Fixed** — `slowVoice` signals on a channel; the 20 ms negative check documented |
| PR-21 | Low | `linkFor`'s unused `depth`; `LinkRoot` reported for a platform default | **Fixed** — `depth` dropped; `LinkDefault` when the platform default was substituted |
| PR-22 | Low | `TestCastImportsNothingAbove` checks one file | **Superseded** — the test is deleted (it re-implemented `make lint-imports`; round-1 Code #19) |
| PR-23 | Low | Trace gaps: the composed-cycle golden, per-preset envelope assertions, the sea-state parity test | **Fixed** — Task 2.5b (the golden); 3.6 envelope/length assertions; the parity test in `app/marine_test.go` (moved there by plan review 2, PR2-5) |

**Verified sound by the reviewer (no finding):** go-toml omits empty nested structs; `alerts = 3` → "corrupt"; `map[string]any`
round-trip keeps `[[locations]]`; `render.Plain` preserves `{{voice}}`; the M5 matrix walk; the marine assembler
fields; no import cycle from the `platform/render` lift; `synth → cast` allowed by the lint; the maritime order and
pause rule; the classifier precedence; function budgets.
