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
| **D-9** | **Verification logic lives in the project's typed, tested language.** Shell invokes things; it does not decide them. | **Yes, by convention** | 17 |

Not adopted, and why: **D-6** folds into D-2. **D-7** (tests assert the requirement, not the observed
behaviour) is a good habit but the list is kept short deliberately. **D-8** (no number without a
re-derivation) is already an A2DH calibration. **D-10** mostly disappears once D-9 holds.

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
