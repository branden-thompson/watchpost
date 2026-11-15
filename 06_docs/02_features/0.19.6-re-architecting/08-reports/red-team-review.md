---
title: "0.19.6 DISCOVER exit red team"
date: 2026-11-15
phase: DISCOVER
sev: SEV-0
status: "Round 1: every finding dispositioned (D-7); five forks put to the HUM LEAD one at a time"
---

# 0.19.6 DISCOVER exit red team, round 1

DISCOVER's exit red team (D-4) was five blind, read-only reviewers. Each was sent
`06_docs/red-team-brief.md`'s template, with the record at `66d8a350` as its scope and the code at
`e57afeaf` (`git merge-base main HEAD`). Their reports are filed whole in `reviewer-notes/`:

| Axis | Report | Verdict | Critical | Important | Minor |
|---|---|---|---|---|---|
| CQ, code quality (the architecture) | `discover-CQ-code-quality.md` | DO NOT EXIT | 1 | 9 | 11 |
| RR, regression and performance risk | `discover-RR-regression-performance.md` | DO NOT EXIT | 1 | 12 | 3 |
| PH, project hygiene | `discover-PH-project-hygiene.md` | EXIT | 0 | 6 | 9 |
| DQ, docs quality | `discover-DQ-docs-quality.md` | DO NOT EXIT | 0 | 10 | 10 |
| BQ, business quality | `discover-BQ-business-quality.md` | DO NOT EXIT | 0 | 11 | 6 |

**What they agree on.** The figures hold: every reviewer re-derived the package sizes, the
`Dashboard` and `livePipelines` method counts, the 692 mutants and the baseline sweep, and found
them as recorded. The cuts are the right ones: CQ found that Map II and Map I are the simplest cuts
that meet D-1, and no reviewer proposed a different map. No new survey is needed.

**What they found.** The package map dropped some things its own lenses had found. The rules that
bind every batch contradict each other for the first code batch. Several batch-0 instruments
would pass by default. And the record's own status and A-rows are not what it says they are.

## The claims the agent checked before ruling

These claims decide a fork or a disposition. Every one holds.

- **The memo guard needs `unsafe`, and the console's guard needs console internals.**
  `perturbEach` imports `unsafe` (`modes/tty/memo_completeness_test.go:9`, `:206`, `:237`). Its
  comment rules out a second copy (`:115-120`). The console's guard calls `buildSpans` and
  `consoleKeyFor` (`modes/tty/broadcaster_memo_completeness_test.go:85-93`), so after the move it
  has to live in the console's package. P10-09 bans `unsafe` in P10 scope
  (`06_docs/code-standards.md:337`). The ledger has two P10-09 rows, neither for `unsafe`
  (`06_docs/p10-ledger.md:215-220`).
- **Router tests read the console's unexported fields.** Examples are
  `modes/tty/router_test.go:67` (`r.broadcaster.width`) and `modes/tty/router_bedplay_test.go:35`
  (`.broadcaster.statusNote`).
- **The router trace keys on Go type names.** It prints `%T`
  (`modes/tty/router_trace_test.go:61-62`), and its golden names `tty.tickMsg` 38 times. It also
  sets `r.broadcaster.now` directly (`:55`).
- **The sweep runs the last killer first** (`06_docs/mutants/run.sh:93`, `-failfast` and `-run`).
  So the brief's "every mutant re-runs its package's whole suite" is wrong.
- **Each promotion overwrites the release's verdict record** (`Makefile:376-378`, `:417`). A
  missing killer record falls quietly to the next step (`scripts/quality/mutant-verdicts.sh:154`).
- **The deck and the maps import `modes/tty`** (for example `app/mastercontrol.go:467`).
  **`app` already imports the standard library's `maps`** (`app/cast.go:10`;
  `modes/tty/help_about.go:9`).
- **The GOOS seam is unexported and marked "Tests only"** (`app/cast.go:51-52`).
- **F-22's guard exists** (`modes/tty/one_band_writer_test.go:19`), although
  `06_docs/follow-ups.md:167` says nothing enforces it.

## How the findings were judged (D-7)

Each finding was judged against the settled package map and the D-rows, not against another
reviewer (0.19.0 D-311). Where reviewers disagreed, the agent chose and recorded why. A finding
against a D-row, or one that settles a question a lens put to the HUM LEAD, goes to the HUM LEAD.
Everything else is the agent's, as an A-row for veto or a record fix.

### Disagreements between reviewers

| # | The disagreement | Disposition | Why |
|---|---|---|---|
| X-1 | **Exit now (PH) or not (the other four).** | **Not yet.** | PH judged hygiene only. RR's F1 and CQ's F-1 make the first code batch unplannable under today's rules. |
| X-2 | **The package-size ratchet (C-6).** CQ: drop it, keep the file rule. BQ: keep it, with a rise as a D-row. | **Put to the HUM LEAD (fork 4).** | C-6 settled a question TG put to the HUM LEAD (DQ-3), so it was never the agent's to settle. |
| X-3 | **The order.** BQ: 2a earlier, to size 2b sooner. RR: the order is sound, but split 2b into three commits. | **Both (A-13).** 2a follows the console. 2b is last, in three commits. | They do not conflict. 2a is in place and low risk, and RR's split is what makes 2b's net strongest when it lands. |
| X-4 | **The tone-latency instrument.** CQ: defer it to 0.19.7. RR: hold its 185 allocations as a pin. | **Both (A-17).** The allocation bound becomes a pin now; the wall-time reconciliation goes to 0.19.7 with D-2. | PB-8 makes it D-2's yardstick, and D-2 is 0.19.7. The deck still moves in 0.19.6, and an allocation bound is deterministic. |
| X-5 | **The graph tool.** PH: in the main module, with its dependency change ruled. CQ, DQ and BQ: commit it, somewhere. | **Committed in batch 0 as its own module under `tools/`, run by a make target, with its `e57afeaf` outputs committed (A-12).** | In watchpost's module it would raise `x/sync`, which the product uses (PH H-3). A dependency change has no place in a release that promises no behaviour change. |
| X-6 | **The partial move in each move commit (C-11).** CQ: ceremony, delete it. | **Kept (A-17).** | The theme of this survey is guards that go quiet after a move. A batch-0 failure proves an instrument works on today's layout. Only a partial move proves it sees the new package. |

## Put to the HUM LEAD, one at a time

1. **The test rule** (RR F1, CQ F-15, BQ 4.1; PH H-6 is a case of it). **Ruled D-14: A.** C-3, D-9 and M7 cannot all
   hold for the console batch: assertions unchanged, no test-only export, and the seam taken.
2. **Where the memo guard engine lives, and P10-09** (CQ F-1).
3. **The shared lock (C-7)** (CQ F-4, RR F9). AS asked for it to be ruled, not taken as an A-row.
4. **The package-size ratchet (C-6)** (X-2).
5. **The brief's amendment** (BQ, RR, DQ and PH, below). The brief is approved (D-13), so changing
   it is a ruling.

## The brief's amendment (fork 5), as proposed

| # | Amendment | From |
|---|---|---|
| B-1 | The problem's fourth bullet states the gate as measured: the sweep runs the last killer first, so the split saves about 9 % of the sweep, `race` about 10 %, and `verify` barely moves. The 57 % and 23 % are shares of 14,101 mutant-seconds summed over 4 workers. | BQ 1.1, DQ-15 |
| B-2 | M1 gains wall time: a benchstat comparison of the beat and frame benchmarks, paired back to back at `-count 10`, each code batch, mandatory at 2a and 2b. Startup (`inittrace`) and binary size are recorded at open and close. | BQ 1.2, RR F10 |
| B-3 | M1's baseline is the maximum over N runs on `e57afeaf`, recorded with N. The window pins' one-allocation flicker is stated beside it. Fewer inlined calls, counted per function, needs a HUM LEAD ruling, not a stated reason. Escapes are gated by function and type, not by line. | DQ-13, RR F6 and F6b, H-14 |
| B-4 | R8: no new `tea.Msg` or command where the code calls directly today, held by a pinned message-type set. R9: no func field read on the frame path, and the map's hooks held once. | BQ 1.5, RR F7, CQ F-10 |
| B-5 | M3 counts each area's reach by type, across packages the same way as within one, including the fields reached through `Env` and the hooks. The hub is measured by type, not by a file ratio. | BQ 1.3, DQ-14, CQ F-17 |
| B-6 | M4 has a before: the same four tasks, blind, on `e57afeaf` at PLAN, with JN's leave-one-out key. M4 is named as the one metric not measured each batch. | BQ 1.4, DQ-12 |
| B-7 | M3, M4 and M5 each name the property of principle 2 they prove: maintainability, traceability or understandability. | BQ 1.6 |
| B-8 | M6 says `make p10` is a local gate, and the build log carries each batch's summary. | H-8 |
| B-9 | M7's closing sweep: every A-3 mutant keeps its class or is re-ruled; every new mutant is caught; none has no evidence. | H-12 |
| B-10 | R3 lists what a move commit may change: package qualifiers, the `doc.go` file maps, ledger and baseline re-keys, doc path rows, the atlas. A body-hash union check proves nothing else changed. | RR F4, H-9, CQ F-14 |
| B-11 | R4 restated: the program's model is always the Router (`app/router_owner_test.go:17`). | CQ F-13 |
| B-12 | The brief states the release's cost and its end state: the batch count, the expected gate hours, what 0.19.7 waits on, and the root `modes/tty` left at about 16,900 lines with `Dashboard` at about 371 methods. A short opening says what users notice (nothing), what is ruled at PLAN, and what is out. | BQ 2.1, 2.2, DQ-11 |

## The agent's own (A-9 to A-18, for veto)

Written in `../02-analysis/rulings.md`:

- **A-9:** the baseline record is committed, its commit pinned, and comparators read it from that
  commit (H-1, H-5).
- **A-10:** C-4, C-5 and C-8 written down, with C-8's comparator specified.
- **A-11:** the re-vet's list is derived, not hand-built (RR F11, CQ F-19).
- **A-12:** batch 0's additions.
- **A-13:** the order of work.
- **A-14:** the naming constraints; the names come at PLAN.
- **A-15:** the record's figures corrected.
- **A-16:** F-22 closed; a follow-up row for every 0.19.7 deferral.
- **A-17:** what is kept, dropped and deferred among the reviewers' deletions.
- **A-18:** the package map SETTLED, with §7 renumbered and a §9 for what the moves invalidate.

## Owed to PLAN, recorded so they cannot be lost

- **The Settings and controls seam for 2b** (CQ F-2): where the D-147 controls and the map's
  Settings rows live, and which way the import runs. The PLAN pilot settles it.
- **The GOOS seam after the deck moves** (H-6): its options go to the HUM LEAD at PLAN, under
  fork 1's rule.
- **The lock inventory, if fork 3 keeps the shared lock** (CQ F-4, RR F9). Every `lp.mu` section
  that touches a moving field, by file and package. AS-5 missed `lp.land`, `lp.path` and
  `inject.go`.
- **F-158's owed test:** its assertion and path, written against the deck's entry point (RR F13).
- **The `app` trace:** specified against the deck's planned exported entry point, with stable keys
  and executed effects (RR F8).
- **Why `app/internal/maps` and the residue of `app/internal/kit` repay their cost**, or they fold
  back (BQ 4.3, CQ F-6).
- **The P10 re-key table by destination:** 14 file-keyed rows plus the two density rows, with the
  seven re-proofs (H-10, DQ-8).
- **A-7's pilot gains a same-name constructor and a same-name method pair** (P10-01), and measures
  a move with qualifiers (CQ F-16, H-9).
- **Each batch's comment pass:** comments that go false after a move, such as
  `app/air_boundary.go:18`, are fixed in that batch (CQ F-14).

## Not taken, with the reason

- **H-2, the local refs and `geodata.test`.** Neither the leftover refs nor the test binary is
  pushed or tracked. Deleting them is the HUM LEAD's call; the agent offers to.
- **DQ-19, line citations in the settled map.** A document under 0.19.6's own release record cites
  lines, as its rulings do (`TestEveryRulingNamingCodeCitesItsLine`). Symbol citations are for
  living documents, which the map stops being at SHIP.
