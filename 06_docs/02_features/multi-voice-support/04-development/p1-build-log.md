# P1 build log — the foundation (multi-voice-support, 0.14.0)

```
Batch:   P1 — the gate's scope helper, the render cap, Reports, the cast package, the config
Date:    2026-08-30
Branch:  feature/multi-voice-support
Gate:    07-readiness/gates.md §1
Host:    darwin/arm64, Apple M3 Pro
CLI:     a2dh v1.17.0 (commit 7960f0e) — one build for the whole release; the ledger's symbol keys depend on it
```

**Verifiable alone — but by an AGENT, not a listener** (HUM LEAD ruling, 2026-08-30): a 0.13.0 config still
sounds exactly as 0.13.0 (nothing below the root is read); a hand-written `[radio.voices.*]` pair is accepted,
validated against this host and reported. The checks are in `07-readiness/agent-uat-p1.md`. **P1 has no
listener-facing surface at all** — the first real end-user UAT is P4, when Setup can assign a voice.

## Tasks

| Task | What landed | State |
|---|---|---|
| 1.0 | `p10-unmatched.sh`: Go-free-diff dormancy, rename-aware symbol scope, four fixture cases in `gate-controls` | ✅ |
| 1.1 | `synth.Limiter` / `Limited`: N ordinary + 2 reserved, one bound over wait-and-render, fail-closed | ✅ |
| 1.2 | `Compose(…, Reports{})`; the zero value is 0.13.0 | ✅ |
| 1.3 | `cast`: the role tree, `Config`, `Pair` | ✅ |
| 1.4 | `cast.Resolve`, the M5 fallback matrix, `WantsInstall` / `WantedInstalls` | ✅ |
| 1.5 | `cast.Validate` | ✅ |
| 1.6 | `cast.Classify` / `ToneName` / `Muted` / `ClassKeys` | ✅ |
| 1.7 | withdrawn at the round-1 review (it re-implemented `make lint-imports`) | — |
| 1.8 | config: `[radio] cast`, `[radio.voices.<role>]`, `[radio.tones]`, `Radio.Validate`, the `ticker_muted` mirror | ✅ |
| 1.9 | config: `Save` preserves unknown keys; the go-toml bump | ✅ |
| 1.10 | nine cross-version fixtures, every one referenced by a case | ✅ |
| 1.11 | `Config.Unknown`: the `[radio.*]` keys this build does not know, rendered and capped | ✅ |
| 1.12 | `app`: `castConfig` / `castToConfig`, the deck as `cast.Host`, the `runtimeGOOS` seam, the lock discipline | ✅ |
| 1.13 | the deck owns one `Limiter`; `voice()` wraps every engine it returns | ✅ |
| 1.14 | this gate | ✅ |

## The dependency bump (Task 1.9)

`github.com/pelletier/go-toml/v2` **v2.2.4 → v2.4.3**, **required**, not optional.

The PLAN spike found that on v2.2.4 the strict decode *panics* on a quoted key carrying an escape
(`subslice address … is before data address`). The panic is swallowed by the scoped `recover`, the unknown-key
list comes back empty, and every future key is then silently dropped on save — NFR-5 failing **quietly**, which
is the worst shape a compatibility bug can take. `testdata/quoted-escape-key.toml` is that case, pinned by
`TestQuotedEscapeKeyNeitherPanicsNorAliasesAKnownKey`.

`scripts/third-party-licenses.sh` regenerated `THIRD_PARTY_LICENSES.md` — 45 modules, 0 without a file.
`go mod tidy -diff` and `go mod verify` clean.

The scoped `recover` is **kept** regardless of the version: it is scoped to the decode alone, so a panic in a
caller's loop can never be swallowed here and misread as "no unknown keys".

## Gate

| Gate | Result |
|---|---|
| Unit + race | `go test ./... -race -count=1` green; `./app` green at `-race -timeout 300s` |
| Verify | `make verify` — **ALL GATES GREEN** (fmt · vet · tidy · vuln · race · lint-imports · lint-watermark · gate-controls, now including the scope helper's own four cases) |
| Harness | `a2dh validate` **17/17** |
| Alloc pins | `make alloc-budget` green (41 packages), the P0 Setup pins included |
| Lint | as ruled (MVS-D-45): `make verify`'s own steps; no new linter |
| P10 | **0 live, 0 unmatched** (116 dormant) · `tree_hash 6b414514eb82783d…` · record at `07-readiness/p10-p1.json` |
| Declsets | `app` re-captured; the diff is 16 additions, all intended, no accidental exports |
| PTY | `make pty-severe` green |
| Build | `make build` → `./dist/watchpost` |

### P10 ledger rows added this batch — **ratified by the HUM LEAD, 2026-08-30**

| File | Symbol | Rule | Why |
|---|---|---|---|
| `domains/radio/cast/cast.go` | package | P10-06 | `registry` is an immutable `[numRoles]role` table with no run-time writer. Stronger than the ratified `themes.go` / `tz` rows, which guard mutable state under a mutex. |
| `domains/radio/cast/tone.go` | package | P10-06 | `classes` and `rules`, same shape. Keeping `rules` as ordered data is what makes MVS-D-15 reviewable against `tones.md` §2. |
| `domains/radio/cast` | package | P10-05 | Density 0.26 across 39 functions: mostly total accessors whose behaviour on bad input **is** the contract, exhaustively tested. The invariants sit where something can be wrong. |
| `app/cast.go` | package | P10-06 | `runtimeGOOS` — the seam Task 1.12 prescribes. Written only by tests, restored by `t.Cleanup`. Without it half the M5 matrix would never execute on this machine or in CI. |

Two findings were **fixed rather than exempted**: a `Config.Clone` that called a `Tones.Clone` read as recursion
(P10-01) — the deep copy now has one owner; and an empty critical section in a test (P10-10) — the goroutine now
reads real state under the lock, which is what the assertion was always about.

## Deviations from the plan

1. **`Reports` carries `Fire` and `Seismic`, not `Maritime`.** `MarineReport` is created at P3 Task 3.2; a field
   of a type that does not exist cannot compile. That the field can join at P3 with **no signature change and no
   call-site churn** is precisely the RS-12 mitigation working, and it is the property the zero-value test pins.
2. **`cast.Host` has a fourth method, `Default()`.** Task 1.4 named three. The platform's own last resort — the
   macOS sentinel, the first installed Piper voice — is a host fact of exactly the same kind as `Discovered` and
   `Installed`, and the fallback matrix's "root unresolvable → platform default" row needs it. Putting it in the
   interface keeps the rule in one place instead of making each call site re-derive it.
3. **`discoverVoices` now routes through `discoverMacVoices(ctx)`.** The plan said the shared function exists for
   the deck and (P4) the report. Leaving the deck on its own `exec.Command("say", "-v", "?")` would have made it
   an uncalled function until P4 — dead code, and two implementations of "which voices does this Mac have". The
   deck's path now carries the 30-second ceiling as a side effect, which it did not before.
4. **One unplanned fix outside the batch.** `TestSayVoiceOnDarwinNarratesHostileTextSafely` ran the real
   `/usr/bin/say` on `context.Background()`; it wedged for ten minutes and panicked the whole package, which is
   what an intermittent `make verify` failure turned out to be. Bounded at 30 s so a wedge fails that test
   legibly. Unrelated to this feature, but it would have destabilised every remaining batch gate.

## Notes carried to P2

1. **The lock discipline is now stated in code**, in `app/cast.go`'s header comment, and guarded by
   `TestValidatingTheCastDoesNotDeadlockAgainstTheDecksLock` — which **fails on a timer rather than hanging**, so
   a regression is a red test and not a wedged CI job. P2 Task 2.7's `setCast`/`castChanged` is bound by it:
   snapshot under the lock, resolve outside, store under the lock.
2. **`castChanged()` is deliberately empty**, with a comment saying so and why. P2 gives it its body. It is the
   one piece of scaffolding in this batch and it is named as such.
3. **`WantedInstalls` returns one de-duplicated, ordered list** rather than an answer per role, because
   MVS-D-44's per-session cap applies to the list, not per role. P2's `ensureRoleVoices` should take it whole and
   cap it once.
4. **`rawVoice` is unexported with exactly one caller** (`voice`), which is what makes "nothing in `app` renders
   uncapped" true rather than merely intended. Keep it that way when P2 adds `resolveVoice(role)`.
5. **The unknown-key merge re-marshals only when it kept something**, so an ordinary save stays byte-stable
   (`TestAnOrdinarySaveIsByteStable`). Anything P4 adds to `Save` must preserve that.
