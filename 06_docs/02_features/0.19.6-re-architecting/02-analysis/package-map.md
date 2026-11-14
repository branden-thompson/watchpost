---
title: "0.19.6 DISCOVER — the package map, reconciled from the nine lenses"
date: 2026-11-14
phase: DISCOVER
sev: SEV-0
authority: HUM LEAD
status: "PROPOSED — the agent's reconciliation (D-7); each fork below is put to the HUM LEAD one at a time"
---

# The 0.19.6 package map

This page is the one settled document for 0.19.6's shape (D-6, D-7; 0.19.0 D-311's rule: a later
finding is judged against this page, never lens against lens). It reconciles the nine lenses in
`lens-*.md`. Where they disagreed, the disposition and its reason are written here; only the forks
that survive go to the HUM LEAD.

## 1. What the survey found, in five lines

1. **Nothing in the record forbids the split** (RC). It forbids splitting the shared deck (K-3),
   copying a window or rule into a second package (K-8, 0.16.0 D-56), and re-implementing Charm
   (D-3). RC's K-1 to K-31 (`lens-RC-the-record.md`, "The constraints") bind every row below.
2. **The hard part is one type, not file count.** `Dashboard` carries 752 methods in 68 files
   (TS-1, JN-1, GP-4); `livePipelines` carries 108 in 21 (AS-2). Go keeps a method in its type's
   package, so an area leaves only after its logic moves onto a type of its own.
3. **The net is whole-model tests.** 705 of 1,097 `modes/tty` tests build the whole `Dashboard`
   (RX-2); 362 of 363 test files are white-box. Tests cannot simply move with their code.
4. **Moving code costs no performance by itself** (PB-3): Go inlines across packages and there is
   no PGO profile. The cost is what a cut forces: a boxed model (+6.5 KB a beat, PB-2), func-field
   contexts that stop 421 inlined calls (PB-3), or new message hops (GP-2).
5. **Many guards would go quiet, not red, after a move** (RX-9, TG-2, RC-3, GP-7, AS-8, TS-12): 22
   directory-scoped scans, two `TestMain` invariants (TG-3, RX-6), the file-map test's fixed list,
   four Makefile targets, and four checks that switch off when a name is exported (TG-6).

**Corrected figures** (A-6): 692 mutants anchor in the two packages, 539 in `modes/tty` and 153 in
`app`; within `modes/tty`, 289 in `map_*.go` and 80 in `broadcaster*.go` (PB-6, RX-1). TS's "18 map
and 66 console" and AS's "110 app" counted double-quoted anchors only. Under AS's Map I, the `app`
mutants fall 112 in the deck, 12 in the maps package and 29 in the root (recounted 2026-11-14 from
the anchors).

## 2. The proposed map

### `modes/tty` (TS Map II, built in two steps)

| Package | What it provides | Size (approx.) |
|---|---|---|
| `modes/tty` | the Router (the one `tea.Model`), the Observer core (`Dashboard`), Settings, the windows, the key routing and the window stack | ~16,900 lines |
| `modes/tty/console` | the Broadcaster console (`Broadcaster`, its 92 methods) | ~3,900 |
| map package (name an A-row) | the map and Propagation, behind a concrete sub-model | ~10,500 |
| kit (name an A-row) | the 41 declarations both surfaces draw: masthead, memo stats, tick, cursor slot, row helpers (TS-4) | ~300-600 |

- **Step 1, the console and the kit:** a move. No interface, no cycle (TS-3, TS-4). 80 mutants.
- **Step 2, the map:** two halves (GP-4, P-d).
  - **2a, in place:** the map's state and logic move onto a map model type *inside* `modes/tty`.
    Every whole-model test still compiles in its package and pins what it pins; tests change only
    identifiers (RX-13's prep commit). This is where principle 2's largest gain is (JN-1, TS-5).
  - **2b, the package move:** only after 2a, and only if the straddling tests can be served by
    production API (§3, C-3). TS-13's strongest counter-argument applies here: 122 straddling test
    files. 2b is the fork in question 5.

### `app` (AS Map I)

| Package | What it provides | Size (approx.) |
|---|---|---|
| `app` | the composition root: `RunDashboard`, the pipelines, the pool, `commit`, the diagnostics | 20 files, 4,054 lines |
| `app/internal/deck` | **the shared deck, whole**: the radio deck, the arbiter, MasterControl, the air scope and fences, the bed, the Director's executors, the Producer and Composer | 30 files, 9,297 |
| `app/internal/maps` | the map layers and the history | 22 files, 6,616 |
| `app/internal/kit` | leaf helpers (`everyTick`, `lazyMemo`, `savePreference` and the rest) | 4 files, 217 |

- Imports one way: `app` → `deck`, `maps`, `kit`; `deck` and `maps` → `kit` only (AS-9).
- The root's deck wiring in `app/dashboard.go` moves into the deck as one entry point, so the
  boundary is about 12 symbols, not 90 exports (AS-4).
- The deck's exported surface is `tty.Radio`'s method set plus the entry point, held by a test
  (AS-7). That makes "only the deck reaches the duck and the band writers" a compiler rule.

### What does not move

- The key routing, the `modal` enum, the window stack and the window-key table (TS-8, K-13).
- The post-update steps (`modes/tty/dashboard.go:1156-1171`), at the root, in today's order (GP-3).
- The deck's surface in the Router: `Station`, `canSwap`, the gain mirror, one `Config` (TS-10).
- `engine.giveWayLocked` and the disposition sheet's Instruct rows (K-4).
- `app/testdata/maps/m1` and every golden (RX-7).

## 3. Disagreements between the lenses, and their dispositions (D-7)

| # | The disagreement | Disposition | Why |
|---|---|---|---|
| C-1 | **The map's package move.** TS recommends it (Map II); RX asks whether it belongs in 0.19.6 at all (201 of 203 map-Propagation killers straddle). | **Step 2a in; 2b put to the HUM LEAD (question 5)** with a recommendation. | 2a gives the ownership gain with the net untouched, because the tests stay in-package. 2b's cost is the tests, and only the HUM LEAD can weigh rewritten pins against a smaller package. |
| C-2 | **Mutant counts.** TS 18/66, AS 110, brief 255, against RX/PB/TG 692. | **692** (A-6), with the per-package figures in §1. | Recounted against the sweep's own attribution, which reads both quote styles. |
| C-3 | **Where tests go.** GP: tests move with their code; RX: straddlers stay at the top, unchanged; RX-4e and TS-3: exported test hooks are possible. | **RX's two-layer net, with GP's helper package for pure helpers only, and no exported test hook.** A seam that cannot be served without a test-only export is not taken in 0.19.6. | Principle 4 is proved only if a pin's assertions do not change (RX-4). A test-only export widens the API against principle 2 (JN P2-5, target 0). |
| C-4 | **Aliases.** TS: the root could keep aliases for `app`. GP: none may survive the release. | **`app` imports the new paths. An alias is a bridge inside one batch, gone before the batch closes.** | GP-11, P-m. A surviving alias is a second name for one thing, against principle 2. |
| C-5 | **`internal/`.** AS puts `app`'s children under `app/internal/`; GP puts what `app` does not import under `modes/tty/internal/`; TG asks whether split-only exports live there. | **Every new package that nothing outside its parent imports goes under that parent's `internal/`.** The console and the map package stay importable if `app` names their types (TS-3, AS-9). | It makes "only the root wires them" a compiler rule (AS-9) and keeps split-only exports out of the module's API (TG-6). |
| C-6 | **A package size ceiling.** JN: no package over about 6,000 lines; TG asks for a ceiling. AS's deck is 9,297, TS's map ~10,500 and its root ~16,900. | **No 6,000 ceiling.** A ratchet: each package's size at 0.19.6's close is recorded and held by a test; a rise is an A-row with its reason. | A 6,000 ceiling forces the cuts AS-6 measured (57 unexported duck and band internals exported) and TS-7 dismissed. JN files P2-1 as a guard, not as evidence. The ratchet keeps D-1's reason for the release from quietly coming back (TG-14). |
| C-7 | **New locks for the bed and the history** (AS-5: two new mutexes, new lock orders). | **No new lock in 0.19.6.** The deck's and maps' structs hold today's `lp.mu` by pointer. | The lock order stays exactly as built, so there is no concurrency change to prove. A lock of their own is a later, measured change. |
| C-8 | **The pins' headroom.** PB recommends keeping ×1.05 and holding the measurement in the build log; PB's own counter-argument is that a rule held by discipline slips. | **A comparator holds it, not the log**: a test compares each pin's measurement with a recorded 0.19.6 baseline and fails on any rise (counts) or more than 0.1 % (bytes). The ×1.05 budgets stay for after the release. | 0.19.5's REFLECT: a control must run by itself, and L-rows that relied on memory failed. |
| C-9 | **A Charm upgrade after the closing sweep** (GP's question). | **Not in 0.19.6.** | It is a separate risk with its own frame comparison. D-3's aim, easy upgrades, is served by the split keeping Charm's surface narrow (GP-8, P-j). Bubbles' place in D-3 is recorded as owed when Bubbles is proposed. |
| C-10 | **D-2.** RM: D-2 proper cannot conform (every step changes what is heard, RM-3); AS: the Producer cut waits for D-2. | **D-2 proper is outside 0.19.6, by D-2's own condition**, put to the HUM LEAD to confirm (question 7). RM's slices 1-4 are in: F-158's owed test, the voice cluster moved whole, the vocabulary corrected, and RM-1's table recorded. | D-2 is in scope "only where it conforms". RM-3 shows Observer's duck would move from once per card to once per drain, which `TestNothingDucksWhenTheBedIsNotCarrying` and `TestTheBedIsDippedOnceForAWholeDrain` pin. |
| C-11 | **FULL TDD for a commit that only moves code** (RX's question). | **The first failing test is the batch-0 instrument**: on a half-done move, the test inventory or the union declaration check fails. Each move commit shows that failure on a deliberately partial move before it lands. | A move adds no behaviour to test first. The instruments are what a move can break, and INST-2 asks to see them fail. |
| C-12 | **Re-vetting at 692 mutants.** 0.19.5 A-14 says every re-pointed mutant is re-vetted by hand; RX-11 says that is not credible at this scale. | **The re-vet is a scoped sweep** (`MUTANT_ONLY` over the batch's mutants, `MUTANT_KILLERS` from the baseline), compared mutant by mutant with A-3's record by a comparator. | The scoped sweep requires the catch to repeat and checks attribution, which is stronger than a hand vet (RX-11). |
| C-13 | **P10 for new packages.** TG: inherit the parent's density row, or each its own; RC: ratify floors as one list at BUILD exit or per batch; TG-5: 16 file-keyed rows break `verify` the moment their file moves. | **Put to the HUM LEAD (question 6)**, with a recommendation: ratify the whole re-key table at PLAN exit (it is fixed by this map), and each new package's own density floor as one list at BUILD exit (0.19.0 D-264). | P10's ledger changes only by the HUM LEAD's ratification (D-4, K-21). Inheriting a parent's row would let a new package ride a floor nobody measured for it (TG-5). |

## 4. What must exist before any code moves (batch 0)

Every lens converged on this list. Each item is written test-first: seen failing, then passing.

1. **A-2:** the sweep falls back to the newest killer record.
2. **The baseline sweep record committed** (`since-v0.19.5.log`, A-3; RX-12), and **a comparator**
   of two verdict records, proven by a planted flipped verdict (RX-12).
3. **Every quiet guard made loud** (RX-9, TG-2, RC-3, GP-7, AS-8, TS-12): each directory-scoped scan
   walks its package tree with a floor taken from the tree, the reply-routing guard included; the
   file-map test derives its packages; `property`, `vet-tags`, `quality-bench` and `test-platforms`
   name `./...` trees or fail on no tests run; the band-writer check stops matching the qualifier
   `tty`; the `wires` sentinel and the other three export-sensitive checks (TG-6) are pinned.
4. **`TestMain` invariants that travel** (TG-3, RX-6): one test-support helper, and a check that
   every package under `app/` and `modes/tty/` carries its parent's invariants (the temporary config
   and state directories; UTC).
5. **Proof that a move lost nothing** (RX-10, TG-7, RC-8): a test inventory over both trees, with and
   without `-tags property` and `-race`; a union declaration check across the old and new packages;
   a `modes/tty` declaration set.
6. **The byte-identical traces** (RX-8): the router trace widened to the map, Propagation, Settings
   and console keys; an `app` trace of the deck and the Director step on fixed seeds; both recorded
   on code identical to `e57afeaf`.
7. **The performance instruments** (PB-1, PB-2, PB-7): the pin comparator (C-8); the three
   race-skipped pins brought under `make alloc-budget`, with a gate test that refuses one left
   outside; the model-escape and size probes recorded.
8. **The map layers' order pinned** (PB-4): `windowLayers()` returns today's order.
9. **The import rules as a gate** (TG-1, TG-14): `lint-imports.sh` extends to `modes`, `platform` and
   `domains` never importing `app`, and `platform` never importing `modes`, each with a planted
   specimen.
10. **The citation lint keyed on the package root, not the directory** (TG-4, RC-5), as an A-row.
11. **The tone-latency instrument reconciled** (PB-8): 15.8 µs today against 1.6 µs recorded; the
    test wiring inside its timed span accounted for. It is the yardstick for the deck's move.

## 5. How each batch proves the four principles

| Principle | Evidence, every batch |
|---|---|
| 1, performance | the pin comparator green; no new model escape on `Update` or `View`; model sizes unchanged; inlined calls on moved files not fewer without a stated reason; `verify`'s seconds recorded |
| 2, understandability | the graph re-run on the new layout: hub fan-in, each area's reach into the model, front-door exports, test-only exports (JN P2-2 to P2-5); the docs-to-code checks green with the file map derived; the atlas redrawn in the same commit |
| 3, best practice and P10 | `make p10` at the end of every batch (TG-5), the import gate green, `lint-authoring` green |
| 4, no regression | goldens and both traces byte-identical; the same test inventory; the union declaration set conserved; the scoped sweep compared with A-3's; `make verify` green, with MAKE_EXIT and "ALL GATES GREEN" read |

Each code batch is two commits (RX-13): **prep in place** (receivers re-homed, exports added, tests
changing only identifiers) and then **move only** (`git mv`, package clauses and imports, mutants
re-pointed by path, no comment edits). No commit is both.

The release closes with a full sweep compared with A-3's: 1,051 caught, the 7 equivalents surviving
or re-ruled, none with no evidence.

## 6. The order of work

1. Batch 0 (§4).
2. RM's slice 1 (F-158's test) and slice 3 (the vocabulary), which change no production behaviour.
3. `modes/tty`: the kit, then the console.
4. `app`: the kit, then the deck whole (RM slice 2), then the maps.
5. `modes/tty`: the map, 2a; then 2b if question 5 says so.
6. The closing sweep, the docs and the atlas, REVIEW.

The console and the deck go first because their nets support a move best (RX-3: the console's
killers sit in its own test files; the deck moved whole keeps 64 of its 101 killers with it).

## 7. The questions for the HUM LEAD, in order

Each is asked one at a time, with its evidence, options, recommendation and strongest
counter-argument.

5. **The map's package move (C-1):** 2a only, or 2a and 2b.
6. **P10 for the split (C-13):** when the re-keys and the new packages' floors are ratified.
7. **D-2 (C-10):** confirm that D-2 proper is outside 0.19.6 by its own condition, and that RM's
   questions (Observer's duck, the severe read's place, STOP and STANDBY) are owed to the release
   that builds F-73.
8. **0.19.5's REFLECT recommendation 3:** whether watchpost scans at its `go` floor and raises it to
   go 1.26.11 (it reverses 0.19.5 A-32's "floor stays").
9. **The brief:** its problem statement and metrics for the four principles (§5), its scope (§2,
   §6), and what is out (C-9, D-2 proper).

The agent's own choices (names, the alias rule, `internal/`, the lock decision, the comparator, the
ratchet) are A-rows, put for veto with the brief.

## 8. Recorded, not asked now

- RM's end-state notes (RM-1's table, RM-5: a pure Director step cannot hold the lock guarantees,
  so D-2's end state keeps a small air-gate effector), for the release that takes D-2.
- RC-7's six places where the docs and the code already disagree (for example
  `where-things-happen.md:15`, one window, against the stack at `modes/tty/dashboard.go:1591`), fixed
  in the batch that moves the code they describe.
- RM-8's and JN-5's misleading names, fixed as A-rows in their batches.
- GP: `docs/architecture.md:528` lists Bubbles as built, and `go.mod` has no Bubbles requirement.
