# Prior art — the PLAN-phase code sketches, kept for BUILD

**Status: NOT the plan. NOT reviewed as the plan. NOT a specification.** These four files are a verbatim
snapshot (at `7296874`) of the PLAN batch documents *before* they were stripped to task shape — file · symbol ·
contract · test intent · verify command. The stripped documents in `04-development/` are the artefacts of
record; anything here that disagrees with them loses.

They are kept because the work was already done and a good part of it was **executed and proven** by the
round-4 compile-first red-team lenses. BUILD should use them as **a starting point and a comparator**, not as
copy-in source: re-derive each task against the compiler, then diff your result against the sketch here and ask
why they differ. Where they differ, the sketch is often wrong (see the defect list below) — but sometimes the
sketch encodes a fact a later reviewer measured, and that is worth knowing before re-inventing it.

Why they are not in the plan: `AP-PLANCODE-01` (`../../06-key_learnings/retro-notes.md` RN-1). Written code in a
plan artefact cannot be compiled, linted or run, so review spends itself on defects a compiler catches free
while the design goes unexamined; three red-team rounds here bore that out. The rule going forward is *we build
during BUILD* — and this folder is the one-off carve-out for a phase whose code had already been written.

## What was actually executed, and what was not

The round-4 Code Quality lens assembled these blocks into a throwaway worktree over the real packages and ran
them (Go 1.27.0, darwin/arm64, `-race`). Trust the sketches in proportion to this table.

| Area | Blocks | Verification | Result |
|---|---|---|---|
| `domains/radio/cast` | 1.3, 1.4, 1.5, 1.6 | built, vetted, gofmt'd, tests run | **3/3 pass** — registry, fallback matrix, classifier |
| `synth.Limiter` | 1.1, 1.13 | built, `-race -count=3` | **all pass**, incl. bound-covers-the-wait, fail-closed wiring |
| `synth` tone/presets | 2.0, 3.6 | built, tests run | **all pass** |
| `synth.Source` (the whole seam) | 2.2, 2.3 | assembled over the real package, `-race -count=2` | **17/19 pass** — both hand-over paths, writer-starvation (0 writer Says), non-fatal announce, every pre-existing synth test. Two failures, both real: see D-1, D-2 |
| `platform/config` (+ `keep.go`, fixtures) | 1.8–1.11 | built, gofmt'd, tests run | **fails on the pinned go-toml v2.2.4** (panic); **entire suite passes on v2.4.3** — see D-3 |
| `app` deck / Director | 2.6, 2.7, 2.10 | static review only (needs the rename) | lock order verified by reading; arities checked against HEAD |
| `platform/snapshot`, `nws`, marine | 3.2, 3.4, 3.5 | symbol shapes checked against HEAD | no compile |
| `modes/tty` Setup / panel | 4.0, 4.3–4.8 | 4.0 compiled; the rest **hand-computed only** | the 4.5/4.6 expected strings match their builders character-for-character; the rest is unverified — see D-4, D-5 |

## Known defects in this prior art (found after it was written; never remediated here)

Fix these when you re-derive; do not copy them forward.

- **D-1 — `TestInvalidateNeverHandsOverMidSegment` cannot pass as written.** With one segment of look-ahead both
  segments render in the old voice before `Invalidate()`, and the soft path plays them as rendered, so the new
  voice says nothing. *The contract this discovered is worth keeping:* a background change takes effect at the
  next segment **rendered**, which with look-ahead 1 is two segments later — not the next one played.
- **D-2 — two assertions in `synth_test.go` contradict.** 2.2 asks that the mid-broadcast render-failure test
  still end the stream, while 2.3 makes `handOver` failure non-fatal (`Err() == nil`). Pick one contract.
- **D-3 — the go-toml bump is required, not optional.** On the pinned v2.2.4 the strict decode of an escaped
  quoted key panics; the scoped `recover` masks it, so `unknownKeys` returns nil and NFR-5 silently drops
  unknown keys. v2.4.3 passes the whole suite. (`scripts/third-party-licenses.sh` + tidy/vuln come with it.)
- **D-4 — `render.WrapLines` takes `[]string`, not `string`;** and at `noteWidth = 44` the note strings wrap
  differently from the expected strings written beside them.
- **D-5 — `PadTo(role, 18)` cannot hold the registry's own labels** ("Alerts & Notification reads" is 27 cells);
  size the `[S]` ROLE column from `widest(labels)+2`.
- **D-6 — `runtimeGOOS()` is defined in a test file** but production code calls it; it needs a production seam.
- **D-7 — `warmHandoffs` has no caller** (and its shape is wrong for Linux: V(V−1) serial renders on one
  ordinary slot). Decide at BUILD whether a pre-warm ships at all; the Linux first-cycle row is the gate.
- **D-8 — `discoverMacVoices`, `tonesLine`, `capNotes`, `pairOf`, `newTestTicker`, `fixtureWarning` are named
  but never defined** here.
- **D-9 — smaller:** prose inside a GREEN fence (2.7); the `Source` struct block is not gofmt-clean; the P1
  held-slot test builds its limiter twice and can panic on a scheduler-latency race; `picker()` pads but never
  truncates; `openSetup`'s root seed has no empty-list guard; two stale comments.

The full finding sets (rounds 1–4, with dispositions) are in `../../08-reports/red-team-plan.md` §12–§13 and
`red-team-plan-round2.md`. Rounds 3 and 4 found many more items in this material than the design itself
contained — which is the evidence behind `AP-PLANCODE-01`.

## How to use this in BUILD

1. Read the stripped task in `04-development/p{n}-*.md` first: it carries the contract, the test intent and the
   verify command. That is the specification.
2. Write the code against the compiler. Run the task's verify line.
3. *Then* open the matching section here and diff. Where they differ, decide which is right — and if the sketch
   is right, you have just saved yourself a discovery.
4. Never resolve a disagreement between this folder and the plan in this folder's favour without saying so in
   the batch build log.
