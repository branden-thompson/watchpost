---
title: "0.19.6 DISCOVER exit red team — CQ, code quality"
date: 2026-11-15
phase: DISCOVER
sev: SEV-0
status: "Filed whole as returned; dispositioned in ../red-team-review.md"
---

## Red team: Code Quality axis, 0.19.6 DISCOVER record (`feature/re-architecting` @ `66d8a350`)

I checked the record against the code at `e57afeaf`. `git diff e57afeaf HEAD` outside `06_docs` is empty. I reproduced the headline figures by `wc` and `grep`: `modes/tty` is 31,346 lines in 88 files, `app` is 20,184 in 76, `Dashboard` has 752 methods in 68 files, and the map is 10,528 lines in 31 files. I made no tree edits and wrote nothing to `<scratch>/`. I ran no tests.

### 1. Understandability

**F-1 (Critical): The memo-completeness guard has no home after the first move, and the obvious home breaks P10-09.** `perturbEach` is the F-30 frozen-frame guard engine. It is a `_test.go` generic shared by `memo_completeness_test.go`, `broadcaster_memo_completeness_test.go`, `masthead_memo_test.go` and `projection_cache_test.go`. Its comment rules out a second copy (`modes/tty/memo_completeness_test.go:115-120`). The lenses say it must become importable *before the console moves* (TS-9, `lens-TS-tty-seams.md:336-337`, `:509`), and RX put "the internal test-support package" in batch 0 (`lens-RX-regression-net.md:427`). The package map's batch 0 dropped it: item 4 is only the `TestMain` helper (`package-map.md:113-114`). It also dropped TS-9's constraint that the map sub-model be embedded or flat, or that the walk be made recursive and proven with a planted two-deep field (`lens-TS-tty-seams.md:332-333`). The walker only goes one level into named structs (`memo_completeness_test.go:181-189`), so getting this wrong passes silently. Worse, an importable support package is non-test code, and `perturbEach` needs `unsafe` (`memo_completeness_test.go:9`, `:237`). P10-09 bans `unsafe` in that scope (`06_docs/code-standards.md:337`). No non-test file imports `unsafe` today, and the ledger has only two P10-09 rows (`06_docs/p10-ledger.md:215-220`). So the console move in §6 step 3 cannot happen without a P10 ratification that nobody has asked for.
- Simplify/Delete? N.
- Action: add a batch-0 item for a memo-guard support package. Put the P10-09 exemption (or the alternative) to the HUM LEAD as a fork. Write TS-9's depth rule into §2.

**F-2 (Important): Step 2b's seam is designed for windows only. The map's need for Settings and the shared controls has no destination.** The map reaches 122 root symbols, 37 of them from `setup_rows.go`: the `rowMap*` ids and the picker helpers (`lens-TS-tty-seams.md:205-210`). The D-147 controls (`checkMark`, `arrowChips`, `cycleIn` and others) are declared in `modes/tty/setup_rows.go:496-618` and used by five `map_*` files. JN-2 said they need a home before any move (`lens-JN-junior-navigation.md:124-125`). That recommendation does not appear in §2, §4 or §6. The kit is sized as "exactly this closure and nothing speculative" for the console (`lens-TS-tty-seams.md:180`; `package-map.md:49`). TS-13's API (`Env`/`Want`/`Key`/`Lines`, `:443-455`) has nothing for Settings rows. Yet M4's boundary-crossing task is "a map Setting" (`project-brief.md:49`).
- Simplify/Delete? N.
- Action: before PLAN, state where the controls and the map's Settings rows live, and the import direction between them.

**F-3 (Minor): The record contradicts itself on its own status.** The required reading calls the package map "the settled document" (`00-REQUIRED-READING.md:26`). Its frontmatter still says "PROPOSED" (`package-map.md:7`), and §7 starts at item 5 (`package-map.md:168`).
- Action: set the status and renumber §7.

### 2. Maintainability

**F-4 (Important): One mutex, owned by the root, would guard fields in three packages.** C-7 has the deck's and the maps' structs "hold today's `lp.mu` by pointer" (`package-map.md:93`). A reader of the deck then takes a lock whose guarded set and lock order (`commit`'s "LOCK ORDER") live in another package. AS-5's inventory of what `lp.mu` guards is also incomplete:
- it misses `lp.land` and `lp.path` in `app/mapwaves.go:102-139`, which go to `maps`;
- it misses `inject.go`'s reads of the root's `watchRefs` and `ticker` (`app/inject.go:219-221`, `:247-249`), which go to the deck.

C-7 also overrides AS's explicit request that the lock question be ruled rather than taken as an A-row (`lens-AS-app-seams.md:542-544`). It is recorded as neither a ruling nor an A-row (see F-11).
- Simplify/Delete? N.
- Action: re-inventory every `lp.mu` section by file and package. Put C-7 to the HUM LEAD. If it stands, require each child's struct doc to name the lock's owner and the full guarded set.

**F-5 (Minor): The C-6 size ratchet taxes every future change.** It holds each package's closing size by a test, and "a rise is an A-row" (`package-map.md:92`). From then on, every feature that adds a line to any of seven packages needs a ruling row. M5 already has the 1,000-line file rule (`project-brief.md:50`).
- Simplify/Delete? Y.
- Recommendation to the HUM LEAD: drop the package ratchet and keep the file rule. Alternatively, report sizes at each REFLECT instead of gating them.

### 3. Complexity

**F-6 (Minor): `app/internal/kit` is a grab-bag that the graph does not justify.** Its members' users, from `symbols.tsv`:
- `lazyMemo` and `keptCopy`: maps and the root only;
- `savePreference`: deck only (`bedrelay.go`, `ticker.go`);
- `everyTick` (`app/every.go:27`) and `lockedCopy` (`app/severe.go:442`): genuinely shared.

`everyTick` must stay one function for its P10-02 row (`p10-ledger.md:63`).
- Simplify/Delete? Y.
- Action: put `lazyMemo`/`keptCopy` in `maps` and `savePreference` in the deck. Name the residue for what it provides (a poll/clock package), or drop it.

Otherwise: Map II for `modes/tty` and Map I for `app` are the simplest cuts that meet D-1. TS-6/TS-7 and AS-6 measured the alternatives soundly. Nothing simpler survives.

### 4. Necessity (primary lens)

These elements can be deleted and the requirements still met:

**F-7 (Minor): C-11's "deliberately partial move" in each move commit is ceremony.** See `package-map.md:97` and `project-brief.md:61-62`. Each batch-0 instrument is already seen failing when it is made (R2, `project-brief.md:58-59`). Re-staging a failure per move commit leaves no evidence in the tree.
- Delete? Y.

**F-8 (Minor): The new `modes/tty` declaration-set pin adds nothing during 0.19.6.** It is §4 item 5 (`package-map.md:118`). It would be re-recorded in every move commit, so it proves nothing in the commit that changes it. The union declaration check is the actual proof.
- Delete? Y. Add the pin at the close if wanted.

**F-9 (Minor): The tone-latency reconciliation can wait for 0.19.7.** It is §4 item 11 (`package-map.md:130-131`; M1, `project-brief.md:46`). PB-8 calls it D-2's yardstick (`lens-PB-performance-baseline.md:366`, `:430`), and D-2 is out (D-11). For a deck move, PB-3 shows the package boundary costs nothing, and the pins plus the deterministic 185-alloc bound suffice.
- Delete? Y, or keep only the alloc bound.

**F-10 (Minor): `Env` should not carry the map hooks.** TS-13 has the hooks both embedded in `Config` and passed again per call in `Env` (`lens-TS-tty-seams.md:445-447`, `:462`). PB-3 forbids func fields on the frame path anyway (`lens-PB-performance-baseline.md:207-208`).
- Delete? Y. Hold the hooks once.

Also see F-5 (ratchet) and F-6 (app kit).

### 5. Name semanticism (SN-01)

**F-12 (Important): The proposed names fail the brief's own name criterion.** M5 says every new package takes "the user's or the role's word" and is not named for a mechanism (`project-brief.md:50`; P2-10, `lens-JN-junior-navigation.md:330`). The proposal breaks this in four places:
- **Two packages named `kit`** (`package-map.md:49`, `:67`). GP-5 says a `util`-style name means the boundary is wrong.
- **`app` would import both kits.** `app/pipelines.go` uses `SnapshotMsg` and `RecentSnapshotMsg` (`cross-package-edges.tsv`), which TS-4 puts in the tty kit (`lens-TS-tty-seams.md:170`, `:181`). So `app` imports `modes/tty/kit` and `app/internal/kit` together, a forced import rename, and reads `kit.SnapshotMsg`.
- **`app/internal/maps` shadows the standard library `maps`**, which `app/cast.go` and the map tests already import (`app/mapfeed_test.go`, `app/maps_basemap_test.go`). It also holds the history (`history.go`, `eventrecord.go`, `stationrecord.go`), and `platform/history` already exists.
- **The tty map package is unnamed** ("name an A-row", `package-map.md:48`), and `map` is a keyword. No A-row ever named it (A-1 to A-8, `rulings.md:45-52`).
- Action: name all four by responsibility, as A-rows, before PLAN.

**F-13 (Minor): R4 states an invariant the code does not hold.** R4 says "the Router stays the one `tea.Model`" (`project-brief.md:63-64`; `package-map.md:46`). `Dashboard` implements `tea.Model` too: `Init` at `modes/tty/dashboard.go:1141`, `Update` returning `tea.Model` at `:1151`, `View` at `modes/tty/view.go:18`. The real invariant is "the program's model is always the Router" (`app/router_owner_test.go:17`).
- Action: restate R4 that way.

### 6. Code documentation

**F-14 (Minor): R3's "no comment edits" in a move commit contradicts the docs gate.** See `project-brief.md:60-62` and `package-map.md:144-146`.
- R6 moves doc rows in the same commit.
- `doc.go`'s file table is a comment, and `TestEveryFileIsInItsPackagesMap` fails when a row names no existing file (`cmd/watchpost/docs_test.go:347-375`; `modes/tty/doc.go:20-33`).
- A new package needs a `// Package` comment.
- Some comments go false after a move with no step assigned to fix them: "Nineteen functions in this package can change what is audible" (`app/air_boundary.go:18`) and "Every sender in this package" (`app/dashboard.go:601`).
- Action: exempt `doc.go` tables and package docs from the rule, and add a docs-reflect-current-reality step per batch.

AP-HIST-01 itself: the record proposes nothing that would write history into comments. Clean.

### 7. Testability

**F-15 (Important): D-9, C-3 and M7 together have no answer for a test that reaches both sides.**
- D-9 commits step 2b whatever the straddle count (`rulings.md:32`).
- C-3 forbids test-only exports (`package-map.md:89`).
- M7 forbids changing a pin's assertions (`project-brief.md:52`).

Of the 84 tty test files that reach map internals, 81 also reach non-map root internals (my count over `test-reach.tsv`, pre-2a, so an upper bound). TS measured 122 files and 512 references under Map II (`lens-TS-tty-seams.md:216`). A test whose assertion reads map internals and whose setup needs root internals cannot move, cannot stay, and cannot be served under these rules. Even 2a's "tests change only identifiers" (`package-map.md:54-55`) is optimistic: methods that take an `Env` need an `Env` built in setup.
- Simplify/Delete? N.
- Action: state the fallback now (rewrite against production API with a re-proof rule, or a ruled exception), and put it to the HUM LEAD.

### 8. P10 conformance

- **P10-09:** see F-1. The support package needs `unsafe`.
- **P10-05:** D-10 and A-7 handle density for moved and changed functions properly (`rulings.md:33`, `:51`).
- **F-16 (Minor): A-7's pilot does not test the recursion rule.** P10-01 finds recursion by bare name (`code-standards.md:329`), and GP left open whether `Dashboard.update` → `Model.Update` or `New` → `console.New` trips it (`lens-GP-go-practice.md:529-531`). The pilot (`rulings.md:51`) settles only the changed-function question.
- Action: add one same-name constructor and one same-name method pair to the pilot.

### 9. Evidence soundness (fail closed)

**F-17 (Important): M3 and M5 rest on a scratch tool that cannot see what they claim to measure.** M3 and M5 are measured "at the end of every batch" (`project-brief.md:41`, `:48`, `:50`) by A-4's tool, which "stays out of the tree" (`rulings.md:48`). The tool and its baseline TSVs sit in a temporary scratch directory (`<scratchpad>/q196/`), not in git.
- **Within-package counts.** It records only same-package cross-file uses, plus exported uses between packages named on its command line (`pkggraph/main.go:68-79`; README: "uses between the two packages (in practice `app` -> `modes/tty`)"). Once the map is its own package, "the map's reach into the model" falls by construction.
- **`Env` reach.** JN's guard, which counts fields reached through the parameter, is not implemented (`lens-JN-junior-navigation.md:323`).
- **Projection.** "The graph projected on the planned layout" (`project-brief.md:48`) needs the lenses' `cut.py` scripts, which are also scratch.
- Action: commit the tool and the baselines in batch 0. Define every M3 figure so it counts across packages the same way it counts within one, and show it failing on a planted `Env` pass-through.

**F-18 (Important): Several batch-0 instruments pass by default.**
- **Floors.** "a floor taken from the tree" (`package-map.md:109`; `lens-RX-regression-net.md:338`) is tautological: the floor shrinks with what it scans. Floors should be fixed numbers recorded at batch 0.
- **Pin baseline.** The pin comparator's baseline is a file the same commit can re-record (C-8, `package-map.md:94`; "re-pinned down", `lens-PB-performance-baseline.md:153`). It should compare against `e57afeaf`'s record, or only ever allow the baseline to fall.
- **M6 without instruments.** M6 claims "no surviving alias", "`internal/` for every…" and "no new lock" (`project-brief.md:51`), and none of the twelve batch-0 items checks them. GP-11's `TypeSpec.Assign` gate (`lens-GP-go-practice.md:483-484`) was dropped.
- **Deck surface test.** It covers only `radioDeck`'s methods (`package-map.md:72-73`). `pool.go` reaches MasterControl's `DropCard`/`MoveCard`/`RequestCard`/`Refence` through `.arbiter.mc` (`file-edges.tsv`), and that exported surface is unheld. Hold the deck package's whole exported API.

**F-19 (Important): Mutant anchors drift in the prep commit, and nothing re-anchors them there.** Anchors are exact text with `assert old in s` (for example `06_docs/mutants/m100_*.py`). 131 of the 289 map mutants have anchor text containing `d.` (my grep), and 2a rewrites that receiver. RX's "Mutants are re-pointed by text" in the prep commit (`lens-RX-regression-net.md:432`) is missing from R3 and §5 (`project-brief.md:60-62`; `package-map.md:144-146`), which re-point "by path" only. The assert fails loudly, which is good. But a hand-re-anchored mutant judged by C-12's comparator only on class and killer name (`package-map.md:98`; `lens-RX-regression-net.md:408-410`) can mutate a weaker site and still read "caught".
- Action: restore prep-commit re-anchoring, and require the comparator to report each mutant whose diff changed beyond a declared receiver rename.

### 10. Go and Bubble Tea idiom

The concrete-child composition, the absence of nested `tea.Model`s, value contexts and the post-update chain held at the root are idiomatic and sound (GP-1 to GP-4). The nine map arms run after every message, between `resume` and `armTick` (`modes/tty/dashboard.go:1156-1171`). TS's `After(Env)` keeps that order only if it runs on every message, not only on the map's (`lens-TS-tty-seams.md:458`, ambiguous).

**F-20 (Important): Embedding the map model in `Dashboard` leaks its methods into `tty.Dashboard`'s API.** TS-13 embeds `mapview.Model` in `Dashboard` (`lens-TS-tty-seams.md:434`), which `app` uses. That promotes `After`, `Key`, `Lines` and `MemoKey`, plus an exported field `Model`, into `tty.Dashboard`'s exported API. It also shadows `Model.Update(msg, Env)` with `Dashboard.Update(msg)`. That breaks M5's front-door limit (`project-brief.md:50`) and confuses a reader.
- Action: use a named, unexported field and make the walker recursive, proven with a planted two-deep field (TS-9's option b). Settle this in §2.

**F-21 (Minor): GP's `*tea.Program` rule (P-j) is cited, not adopted.** P-j says only the root names `*tea.Program` (`lens-GP-go-practice.md:504`). The map cites it only for C-9 (`package-map.md:95`). `app/radio.go` and `app/ticker.go`, both bound for the deck, take `*tea.Program`.
- Action: adopt P-j for the deck in §2.

### Not found, by question

- **F-73 and F-158** (`06_docs/follow-ups.md:90`, `:145`) agree with D-11 and A-8. Clean.
- **F-11 (Important, traceability):** §7 says names, aliases, `internal/`, the lock, the comparator and the ratchet are "A-rows, put for veto with the brief" (`package-map.md:176-177`). A-1 to A-8 contain none of them, and D-13 approved "A-1 to A-8" (`rulings.md:36`). C-4 to C-8 therefore have no ratification trail.
- **Action for F-11:** write them as A-rows, and put C-7 as a fork.

### Verdict: DO NOT EXIT DISCOVER

The figures are right and the cuts are well chosen. But the package map has dropped three things its own lenses found: the memo-guard home (F-1), the controls and Settings seam for step 2b (F-2), and the fallback for tests that reach both sides (F-15). Two of them need HUM LEAD forks that belong to DISCOVER, not PLAN.

**Fix first: F-1.** It blocks the first code batch, the console, and it needs a P10-09 ruling nobody has asked for.
