# The quality plan — four steps, seven metrics, six rules

**HUM LEAD approved 2026-09-03**, in answer to the concern that the project had entered a negative
loop: buggy code met with guards that were themselves buggy, at the cost of building the product.
The evidence behind it is `defect-classes.md`; this is what to do about it.

**The premise, in one line.** The product's defect rate is what concurrent scheduling code costs and
every defect was caught before shipping. The instruments' defect rate is not, and it is the loop.
Seventeen of the session's thirty-four defects were in shell scripts.

## The four steps

| # | Step | Effort | Why in this order |
|---|---|---|---|
| 1 | Port both guards to Go tests; delete the shell | ~1 h | Removes the medium that produced 17 of 34 defects. Ends with **less** infrastructure than it started. **Done: `06_docs/mutants` is one Go package; check.sh, controls.sh and the four fixtures are deleted** |
| 2 | Fold D-3 and D-4 into that test | ~0 | Free once step 1 exists. **Done for D-3. D-4 WITHDRAWN** — three static predicates each flagged legitimate mutants, and the distinction it needs is semantic. Its decidable form is a triage rule at sweep time |
| 3 | Write D-1, D-2 and D-5 as a short skill beside `p10` | ~30 m | Three habits, no tooling. The ones that would have stopped real product defects. **Done: `06_docs/skills/defect-classes/SKILL.md`** |
| 4 | Resume T2.3 under the rules, measure, decide | — | The evaluation point |

**Step 4 is a real gate.** If the metrics have not moved by the T2.3 exit, the answer is not more
rules, and this plan should be abandoned rather than extended.

## The metrics

Each carries a number measured from the 2026-09-02/03 session, so the comparison is evidence rather
than impression.

| Metric | Baseline | Target at T2.3 exit | How it is measured |
|---|---|---|---|
| Fix attempts per defect | 2.5 | ≤ 1.5 | Commits per finding |
| Review rejection rate | 5 of 5 | < 50 % | Reviews returning ADDITIONAL FIXES REQUIRED |
| Instrument lines per product line | 1.05 | ≤ 0.2 | `git diff --numstat`, by path |
| Invalid or stale mutants authored | 12 | 0 | The guard reports it |
| Vacuous invariants shipped | 5 | 0 | Every new invariant has a mutant that fires |
| False claims per commit | 0.27 | 0 | Review sampling |
| **Tasks completed per session** | **1** | **≥ 2** | The one that actually matters |

**WHAT COUNTS AS A DEFECT, AND WHAT DOES NOT (HUM LEAD, 2026-09-03).** *"UAT bugs do not count
against our metrics — these are, as you correctly state, only possible to know from a human
exercising the functionality. These were CORRECT CODE, just not DESIRED behaviour. In complicated
cases that's generally not possible to predict at first; specifications have to be created and tests
built around them."*

| | Counts | Example from 0.14.0 |
|---|---|---|
| **Defect** — the code does not do what its own specification says | **Yes** | The tone taken from the first card after the rail changed what "first" means; `waitFor` subtracting silence that was not there; a stale read clearing the current one's state |
| **Specification gap** — the code does exactly what was specified, and the specification was incomplete | **No** | No way to pause a `[w]` read; `[esc]` not stopping one; the read stacking a human could produce and no written rule forbade |

**The line is drawn at the ruling, not at the report.** A UAT finding costs nothing against the
metrics. The IMPLEMENTATION that follows its ruling is ordinary work and is measured like any other —
a regression introduced while satisfying MVS-D-74 counts exactly as much as one introduced anywhere
else. Three did, this session, and they are counted.

**Why this matters to the programme rather than being bookkeeping:** without the distinction, the
metrics punish exactly the activity that finds the most valuable problems. A session with heavy UAT
would look worse than one with none, which would make the honest move — putting a build in front of a
person early and often — the one the numbers argue against.

**The last row is the point.** Every other number can improve while the product stands still, and if
that happens the programme has failed however green the rest looks.

## The rules

Adopted. Numbered for a skill beside `p10`. "Mechanical" means a compiler or a script decides it, not
a person remembering.

| Rule | Statement | Mechanical | Would have prevented |
|---|---|---|---|
| **D-1** | **One carrier per rule.** A rule is enforced in exactly one place; every other site calls that place. Two carriers will disagree, and the disagreement is silent. | Partly | 4, plus the historical duck-lift bug |
| **D-2** | **Every invariant must have an input that makes it fail.** If none exists, delete it. A check implied by its own enclosing branch is decoration. A rule proved by a throwaway probe keeps the probe as a test. | Partly | 5 |
| **D-3** | **A mutant deletes a RULE and leaves the tree compiling.** Removing a *use* yields INVALID, which is not evidence either way. | **Yes** | 6 |
| ~~D-4~~ | **WITHDRAWN 2026-09-03.** Never mutate an assertion is TRUE and not mechanisable: three static predicates each flagged legitimate mutants, the last being one that drops half a conjunction inside an invariant, which is a real rule deletion and is caught. Weakening an invariant is legitimate when the code under it can violate the weakened form and vacuous when it cannot, and that is semantic. Its decidable form is a triage rule at sweep time — a mutant that SURVIVES while editing only an invariant's condition means an unpinned rule or a vacuous check, and both need a person. | No, as it turned out | 1 |
| **D-5** | **Every ordering guarantee names its SCOPE and is tested at that boundary.** "A before B" means nothing without "within what". | No | 2, both serious |
| **D-11** | **A pin is not evidence until it has been seen failing on its own defect**, in every environment the gate runs it in. A pin for an interleaving fixes its own `GOMAXPROCS` rather than inheriting it. | Partly — a mutant run to CAUGHT is this observation mechanised | The two T2.3 pins that were green while the defect was live |
| **D-9** | **Verification logic lives in the project's typed, tested language — and shell that DOES decide must have Go tests that drive it.** *(Restated 2026-09-08, then corrected the same day. The absolute form contradicted the tree: `06_docs/mutants/run.sh` is 115 lines and is the entire verdict engine. **Only that one has a Go driver** — `harness_test.go` builds a probe repo and asserts all five verdicts with their exit codes. `p10-unmatched.sh` (131) and `ledger-ratified.sh` (115) decide things too and are covered by SHELL self-tests through `gate-controls`, not Go; `exposure-scan.py` (228) has neither. So the rule is **aspirational for three of the four**, and the restatement's own "~220 more" undercounted: it is 246, plus 228 of Python it omitted entirely. Named here rather than left as a rule the tree quietly fails.)* | Partly | 17 |

Not adopted, and why: **D-6** folds into D-2. **D-7** (tests assert the requirement, not the observed
behaviour) is a good habit but the list is kept short deliberately. **D-8** (no number without a
re-derivation) is already an A2DH calibration. **D-10** mostly disappears once D-9 holds.

## Instrument validation — the TDD dimension, INST-1 to INST-5 (added 2026-09-08, 0.15.0 B6/B7)

**HUM LEAD, 2026-09-08:** *"the testing / instrumentation methodology to drive feature development…
I argue is better than the standard TDD methodology on its own. I do not want that 'magic' lost."*
And, on where it belongs: *"this to me is another **dimension** of TDD — I expect to take these rules
and fold them formally into the A2DH framework so this becomes part and parcel to the 'FULL TDD'
directive when doing work."*

**The name was changed from "instrument-driven development" for accuracy** — nothing here drives
development; every rule is applied to an instrument, and a reader told "we use IDD not TDD" would
write no test first.

**A NEW PREFIX, DELIBERATELY.** `D-n` already means two different things — `D-9` is this plan's
"verification logic lives in a typed language" AND the architecture's render pivot seam
(`platform/render/render.go:1`). These are `INST-n` rather than deepening that collision.

**IT IS A DIMENSION OF TDD, NOT AN ALTERNATIVE TO IT** (HUM LEAD, 2026-09-08).
You still write the test first; these rules govern the INSTRUMENT that test
becomes. Read "on top of TDD", never "instead of". The destination is the A2DH
framework, where these fold into the **FULL TDD** directive rather than living
here as a local house rule.

**On independence, which this section does not claim to have discovered.** The
A2DH framework's **red-team** skill (`02_skills/critical-analysis/red-team/`, in the A2DH
install — NOT in this repository) already documents it, and 0.15.0's BUILD exit re-validated it the hard way: the author's
own red team found five things; three blind agents found roughly twenty more,
**including all three blockers**, and the sharpest was a defect *inside the fix
for one of their findings*. What follows may sharpen that skill; it does not
replace it.

### Why this is a dimension TDD does not cover

TDD proves the CODE does what the TEST says. It does not prove the test is asking anything. Every
defect below was found in 0.15.0 while the suite was **green**, and none of them is a TDD failure —
they are instrument failures, and TDD has no opinion about instruments:

| Green, and measuring nothing | What it actually was |
|---|---|
| the AA contrast gate | iterated the same list its lifter had just lifted — a tautology |
| the `--ascii` scan | 2 surfaces of 13, and 7 glyphs of 22, both hand-listed |
| `gate-controls` | the gate that proves other gates fire **could not fire on itself** |
| 5 P10 exemptions | in force, unratified, hiding behind *"the same RATIFIED pattern as platform/geo"* — and `platform/geo` was one of the five |
| the release-check pin | asserted a bound on `start` while calling `checkAt` |
| my own Settings test | asserted the setter FIRED, not that the value round-tripped |

**The premise this extends.** `D-11` says a pin is not evidence until seen failing on its own defect.
That is the seed. INST-1..5 are what the same idea costs when the subject is a GATE rather than a
function, and the reason it is worth paying: **an instrument that silently reports "clean" is
indistinguishable from a clean subject, and it is the only kind of bug that gets more confident the
longer it survives.**

### The rules

| Rule | Statement | Mechanical | The catch that earned it |
|---|---|---|---|
| **INST-1** | **The SUBJECT LIST is derived, never enumerated.** The set a gate ITERATES is computed from the thing itself — the enum, the AST, the registry, the type by reflection. A hand-written set is stale the day after it is written and its staleness is invisible. **Scope: the SET, never the thresholds.** A floor, a budget or a cadence is a parameter and is meant to be written down; the list of things being checked is not. | No — habit | `--ascii` (2/13 surfaces, 7/22 glyphs); the AA register (75 declared vs 64 registered); F-45's 23 rows found by ENUMERATING a file whose closing line said it was clean |
| **INST-2** | **Silence is a verdict and must be a DISTINCT one.** An instrument reports "passed", "not applicable", "could not run" and "not covered" as four different things. Any of the last three rendered as the first is worse than having no instrument. | No — habit | `git grep -E` treats inline `(?i)` as FATAL — four exposure categories printed a confident `0` from a crashed command; `m44` reported **UNAPPLIED** — the harness's own verdict word, which these documents narrate as "unmeasured" — rather than passing, when a collapse moved the rule it guarded |
| **INST-3** | **A plant that survives indicts the PLANT first.** Before reporting a hole, prove the plant reached the subject. | No — judgement | `_ = make([]byte, 64)` never escapes, so the compiler deleted the defect before `alloc-budget` could count it; re-planted so it escaped, CAUGHT |
| **INST-4** | **Believe an instrument about the unknown only after it has answered the KNOWN.** A/B it against a case whose answer you already have. | No — habit | The Lookup stall: a forced repaint "showed" an empty box, and the identical dump appeared in the warm-config run where the echo arrived 0.0 s later. Forty seconds of A/B falsified a finding I had already reported |
| **INST-5** | **State the instrument's blind spot with its number** — in the instrument's OWN OUTPUT, not only in a document that quotes it. A count from a method that can miss is a FLOOR, not a total. | No — habit | The duplicate detector reported two 3-copy groups as pairs: `nws.Fetch` differs from its twins by one statement, `Slot.String` by the field it returns |

### How to apply them

The five in the order you meet them when building a gate. Each has a "done when",
because a rule you cannot check you have followed is a rule nobody follows.

1. **Before writing the gate — INST-1.** Ask what set it iterates and where that
   set comes from. If the answer is a literal in the test file, stop and find the
   producer: the enum, `parser.ParseDir`, the registry, the struct by reflection.
   **Done when** adding a member to the real thing makes the gate cover it with
   no edit to the gate. Worked example: `modes/tty/golden_test.go`
   (`asciiSurfaces` walks the modal enum) against what it replaced (seven glyphs
   and one window, both typed by hand).
2. **While writing it — INST-2.** Enumerate the ways it can end: passed, failed,
   nothing to check, could not run. Give each a distinct message. **Done when**
   you can make it print each one on demand. Worked example: `06_docs/mutants`
   (`CAUGHT` / `SURVIVED` / `INVALID` / `UNAPPLIED` / `SKIPPED`).
3. **Before believing it — INST-3.** Plant the defect it claims to catch. If the
   plant survives, **suspect the plant first**: check it compiled, applied, and
   was reachable in the state the gate renders. **Done when** you have watched it
   fail for the right reason — an assertion with a message, never a compile
   error (see `build-methodology.md` item 1).
4. **Before quoting a number from it — INST-4.** Run it against a case whose
   answer you already know. **Done when** the known case comes out right. Forty
   seconds of this falsified a finding already written down (F-58).
5. **When publishing the number — INST-5.** Say what the method cannot see, in
   the output. **Done when** the blind spot appears next to the count.

**`plant`** is defined in `06_docs/02_features/0.15.0-pre-broadcaster-ui-improvements/07-readiness/gates.md` (the only release whose roster carries the definition); the short form
is: introduce the defect the gate claims to catch, run the gate, record CAUGHT or
SURVIVED. **`P10`** is an A2DH skill, indexed at `02_skills/implementation/p10/README.md` **inside the A2DH
install, not this repo** — read it with `a2dh show 02_skills/implementation/p10/README.md`; `make p10` needs the framework CLI (`A2DH=/path/to/a2dh`)
and is a LOCAL gate by design, since the exemption ledger lives outside the
public tree.

### Two corollaries that are not rules but keep being true

- **Assert the ROUND TRIP, not the call.** A setter that fired proves the write was *asked for*. The
  Settings defect survived because the tests asserted the hook, and the listener was watching the
  value fail to come back. *(See `TestClosingSettingsLeavesTheModelAgreeingWithTheWrite`.)*
- **Measure the surface before scoping the fix.** "139 files carry the demo location" was true and
  useless; exactly **one** of them shipped in the binary, and it was a comment. The measurement
  turned a re-record of 67 recorded API fixtures into a three-line change.

### Why five rules were added to a list that is meant to shrink

The plan closes with *"the list should shrink over time, not grow. A rule that
cannot be made mechanical and has not prevented anything in two releases should
be retired."* Five rules were then added, three of them self-describing as
judgement or habit. That needs answering rather than leaving as a contradiction
(junior-dev review, 2026-09-08).

**The admission price was paid: every INST rule carries a named catch from this
release, and none is speculative.** The shrink rule's own test is "has not
prevented anything in two releases" — so these are due for review at **0.17.0**,
and any that has not caught something by then should be retired the way `D-4`
was. That is the deal, written down where the rules are, not asserted afterwards.

### Two limits recorded rather than built around (red team, 2026-09-08)

**The exposure scan once followed whoever RAN it, and does not any more.** It took its identifiers
from `git config` and `$HOME`, so on another machine the `path` and `host` categories matched nothing
and reported a confident ZERO against a real 12 files and 81 occurrences. It now derives them from
`git log --all` and from the paths already in the tree, which is machine-independent and
self-maintaining for new contributors. **This paragraph previously recorded that as an accepted
limit, citing a HUM LEAD "leave it" — and stayed there after the adjacent commit closed it.** The
ruling was about not building for hypothetical collaborators; it was never a ruling against this fix.
A document rewritten to remove claims the tree contradicts had grown a fresh one within the same
diff, which is the whole reason this section exists. *(INST-5.)*

**The mutant corpus survives a collapse by FAILING, and that is already universal.** Measured
2026-09-08: **171** mutants across **32** distinct target files, and **zero without an `assert`** — so
every mutant can report UNAPPLIED, and the harness fails the gate on it rather than passing (proven
by `m44`, stranded when metric D moved `ByTab`'s guard into `platform/bucket`).

**The consequence is a procedure, not a tool.** As duplicates collapse n→1, mutants that guarded the
copies strand. Each is **re-pointed at the new owner, not deleted** — after which one mutant guards
every caller, which is the collapse paying twice: one place to fix, one place to measure. Deleting a
stranded mutant instead silently retires the rule it was written for.

### What it cost, and what it returned

Six instruments were built or repaired in B6/B7. **Four were wrong on their first run, and every one
of those failures looked like good news** — a zero, a clean bill, a green gate. All four were caught
by planting rather than by reading. The plant sweep for `gates.md` was ~20 minutes and found a real
hole in the gate whose entire job is finding that class of hole.

**The honest caveat, for whoever reads this next:** every one of those catches was made by the same
agent that wrote the instrument. That is the arrangement a red team exists to distrust, and it is why
the BUILD exit red team should point at the NEW INSTRUMENTS first. A gate is at its least trustworthy
on the day it is added.

## Step 4, measured — the evaluation point, 2026-09-03

Measured at the T2.3 exit on `92a3c21`, against the baselines above. Every figure is derived from
`git diff --numstat` over `6751c8f..HEAD` or from a gate's own output, not recalled.

| Metric | Baseline | Target | T2.3 | |
|---|---|---|---|---|
| Review rejection rate | 5 of 5 | < 50 % | 2 of 3 (67 %) | moved, missed |
| Instrument lines per product line | 1.05 | ≤ 0.2 | 0.43 (144 / 334) | moved, missed |
| Invalid or stale mutants authored | 12 | 0 | **0** | **met** |
| Vacuous invariants shipped | 5 | 0 | **0** | **met** |
| False claims per commit | 0.27 | 0 | 0.13 (2 in 15, both self-caught) | moved, missed |
| Fix attempts per defect | 2.5 | ≤ 1.5 | 1 on each product defect; 3 on one instrument | mixed |
| **Tasks completed per session** | **1** | **≥ 2** | **1** | **did not move** |

**RULING: CONTINUE (HUM LEAD, 2026-09-03).** *"This still was a much better loop experience than
before — where we were burning HOURS of time. So I think we should continue this process — keep
measuring metrics and look for programmatic improvements when gaps are discovered."*

The plan's own step 4 said the programme should be abandoned if the metrics had not moved. They
moved: the two classes that produced the bulk of the last session's defects — vacuous invariants and
stale mutants — went to zero and stayed there. The headline row did not move, and that is recorded
rather than explained away; the loop is still finding real product defects, and F-D5 was one that
would have reached a listener at T3.1.

## The gap T2.3 found, which no rule in this plan covers

**A pin can be green while the defect it names is live.** It happened twice in one task, and neither
time was caught by a gate:

| | The pin | Why it was green anyway |
|---|---|---|
| 1 | Waited to observe the dip, then lifted | The goroutine never ran before `hold` had finished |
| 2 | Kept a lift in flight, 5000 rounds | On ONE processor `hold` has no preemption point, so the window is never observable and the DEFECTIVE code passes |

Both were caught only by running the pin against the broken code first. **D-2 is the nearest rule and
does not reach this** — D-2 is about invariants that cannot fail, and these were tests that could
fail, just not on the defect they were written for.

### D-11 — a pin is not evidence until it has been seen failing on its own defect (RATIFIED, option A)

**Statement.** A test written to catch a specific defect must be OBSERVED failing against that defect,
in the environment the gate will run it in, before the defect is called closed.

**Mechanical: mostly, and already half-built.** The mutation harness IS this mechanism — a mutant
that reintroduces the defect, run to CAUGHT, is exactly the observation. T2.3's residual gap is that
a mutant is run in ONE environment, so a pin that false-greens on a single processor still reports
CAUGHT on a developer's machine.

**Two ways to close it, with costs, for ratification:**

| Option | Cost | Catches |
|---|---|---|
| **A — habit.** A pin for an interleaving fixes its own `GOMAXPROCS` rather than inheriting it | zero | The environment-dependence, at the point of writing |
| **B — tooling.** Every NEWLY ADDED mutant is also run once at `GOMAXPROCS=1` | ~143 s per new mutant, not per sweep — ~24 min across T2.3's ten | The same, mechanically, including pins nobody thought about |

**RATIFIED 2026-09-03 (HUM LEAD): OPTION A.** B is not adopted and stays priced above, to be revisited only if A is seen to fail.

**Recommended was: A now, B only if A is seen to fail.** A costs nothing and the list is meant to shrink,
not grow; B doubles the cost of authoring a mutant to catch a case that has arisen once. **Not
self-approved — this is a cost knob.**

## The caveat that was stated when this was approved

Only D-3 and D-9 actually reduce work — D-4 was expected to and could not be made mechanical. D-1, D-2 and D-5 are cheap, but they are still a person
being careful, which is the thing that has been failing. **The list should shrink over time, not
grow.** A rule that cannot be made mechanical and has not prevented anything in two releases should
be retired.
