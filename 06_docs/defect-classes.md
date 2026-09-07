# Defect classes, and which of them a rule can catch

Written 2026-09-03, after a session that produced 22 commits, five adversarial reviews and five
rejections. The question it answers is the HUM LEAD's:

> *"I'm concerned that now in a negative feedback loop of having to build more and more protective
> infrastructure because the code quality is apparently low… buggy code is being written to address
> buggy code… All of this is preventing us from actually building the features for the product."*

The concern is correct in its observation and, I think, wrong in one of its premises. This document
separates the two, because the fix depends on which is true.

## 1. Where the session's effort and defects actually went

| | Lines added | Defects found in it |
|---|---|---|
| Product Go (non-test) | 614 | 10 |
| Test Go | 1,000 | 4 |
| **Instruments** (mutants, sweep wrappers, guards — all shell and Python) | **644** | **17** |
| Record (build log, learnings) | 377 | 4 false claims |

**The instruments have roughly the same volume as the product and nearly twice its defect count.**
That is the loop the HUM LEAD is describing, and it is real. But it is not evenly distributed, and
the distribution is the whole finding.

## 2. The defects are not uniformly spread — they cluster in two places

**Cluster A — ordering and state across time, in the impure half.** Every product defect this
session was in `app/pump.go` or in the parts of `platform/lineup` that decide what happens NEXT: the
band race, the cross-step ordering gap, the duck unordered against the read, the lane outliving the
loop, the queued release dropped on a coin flip, the rail wedge, the lookahead regression, the double
publish. Not one was in a pure value: the card model, the planner, the fence and the ladder were
right first time and their mutants caught everything.

**That is the architecture working exactly as designed and is the most encouraging fact here.**
Approach C exists to move decisions into a pure function so they can be asserted directly. The pure
half needed cheap mutants; the impure half needed expensive adversarial review. The remaining risk is
concentrated in a small, named place rather than spread through the release.

**Cluster B — the instruments, all of them written in shell.** Silent failure is bash's default: exit
codes vanish through pipes, counters die in subshells, an unmatched glob passes as an empty list,
`--exclude '.git/'` matches a directory but not a file, and nothing has a type. Seventeen defects,
including the only one this session that could have destroyed the HUM LEAD's work.

## 3. The reframe

The premise I think is wrong is *"the code quality is apparently low"*. Ten defects in 614 lines of
concurrent scheduling code, every one found before it reached a user and none of them shipped, is not
a quality collapse — it is what this class of code costs everywhere, and the release exists precisely
because the previous arrangement shipped ~30 of them past every gate.

**The loop is real, but its cause is narrower than "our code is buggy": the guards are being written
in the least safe medium available, so each guard needs its own guard.** That is the regress. It
stops if the guards stop being shell scripts.

**Recommendation R-1 — port the guards to Go tests, and delete the shell.** A `TestEveryMutantApplies`
that walks `06_docs/mutants/*.py`, applies each to a temp directory and asserts it applies and
compiles would be strictly better on every axis that failed this session:

- it runs under `go test ./...`, which is already gated, already parallel and already in CI, so no
  Makefile wiring and no `verify` cost decision;
- `t.Fatalf` cannot be lost the way an exit code can;
- copying is `os.CopyFS` rather than rsync semantics, so the `.git` file-versus-directory bug cannot
  exist;
- it is testable by ordinary means, so the controls stop being a parallel implementation of the thing
  they check — which is why nine of ten sabotages passed them;
- no bash version, no BSD-versus-GNU, no word splitting, no subshell counters.

This REMOVES infrastructure rather than adding to it. It is the one piece of build work I would argue
for doing before anything else, and it should take an hour, not a night.

## 4. The rules worth extracting

Numbered as candidates for a skill beside `p10`. **"Mechanical" means a script or a compiler can
decide it** — the rest are review prompts, and are marked as such honestly rather than dressed up.

| # | Rule | Mechanical? | Evidence this session |
|---|---|---|---|
| **D-1** | **One carrier per rule.** A rule is enforced in exactly one place; every other site calls that place. Two carriers WILL disagree. | Partly — repeated literal predicates over the same constants are greppable | 4×: the wordless-card rule in `Propose` and in `check`; `textAtStandby` written out in three places; the duck lifted by `Tune` and not `tune`; `Fenced` beside `Fence` |
| **D-2** | **Every invariant must have an input that makes it fail.** If none exists, delete it — a check implied by its own enclosing branch is decoration. | Partly — the narrow shape (`Check(X)` inside `if X`) is an AST match; the general case needs a mutant per invariant | 5×, three of them added to satisfy a density gate |
| **D-3** | **A mutant deletes a RULE and leaves the tree compiling.** Removing a *use* yields INVALID, which is not evidence either way. | **Yes** — apply every mutant to a copy and build it | 6× |
| **D-4** | **Never mutate an assertion.** A mutant that weakens a check measures the check; it can never fail. | **Yes** — reject a mutant whose diff touches only assertion lines | 1× |
| **D-5** | **Every ordering guarantee names its SCOPE, and is tested at the boundary of that scope.** "A before B" is meaningless without "within what". | No — a review prompt | 2×: the band ordered within a step and not across; the duck ordered against a card's own effects and not against effects naming no card |
| **D-6** | **A rule proved by a probe keeps the probe.** Manual verification becomes a test in the same commit, or the rule is unpinned the moment you close the terminal. | Partly — a commit claiming verification with no test added is flaggable | 1×: the lane's retirement, proved and then deleted |
| **D-7** | **A test asserts the REQUIREMENT, not the observed behaviour.** A test written by running the code and recording what it did will enshrine the defect. | No — a review prompt, but "was it watched to fail?" is the tell | 2×: a test asserting that a cue and its words may be split; a test asserting the duck runs concurrently with the read |
| **D-8** | **No number without a re-derivation in the commit that quotes it.** | No | 6 false claims, including a density figure measured before the code it described existed |
| **D-9** | **Verification logic lives in the project's typed, tested language.** Shell is for invoking things, not for deciding them. | **Yes, by convention** — no logic, loops or counters in a gate script | 17× (all of cluster B) |
| **D-10** | **A guard's controls exercise the guard's REAL path.** Controls that run their own copy of the logic prove nothing about selection, counting or exit plumbing. | Partly | 2×: `check.sh`'s controls ran beside its loop; `controls.sh` still passes 9 of 10 harness sabotages |

**D-1, D-2 and D-5 are the ones that would have prevented actual product defects.** D-3, D-4, D-9 and
D-10 are about the instruments and are where the volume is. If only three are adopted, adopt D-1, D-2
and D-9.

## 5. What I would NOT do

- **Do not add another review layer.** Five reviews returned five rejections, but three of those
  rejections were of fixes to instruments, not of product code. More review of the same material has
  a falling return; fewer, better-targeted instruments have a rising one.
- **Do not adopt cross-model review yet.** The evidence says the find-rate comes from the reviewer
  being FRESH and being required to RUN something — both already true within one model. Cross-model
  adds independence of training on top, which is real but unmeasured, at a context-handover cost the
  HUM LEAD has already named as the deciding factor. If it is tried, try it at phase exits only,
  where the record must be complete anyway.
- **Do not keep hardening the shell guards.** Port them (R-1) or park them. Every hour spent on them
  this session bought less than the hour before it.

## 6. The honest cost accounting

The reviews were not the expensive part. Each cost roughly twenty minutes and each found something no
gate did — including a data-loss bug in a guard, a freshness regression in the scheduler, and a
concurrency race in the pump. The expensive parts were **my own instrument churn** (three sweeps
started, two killed, one wrapper rewritten three times) and **the fix-review-fix cycles on the pump**,
where two defects took five attempts between them.

The second of those is the cost of the code being genuinely hard. The first was avoidable and is what
R-1 addresses.
