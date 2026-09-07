---
name: implementation/defect-classes
display_name: "Defect Classes — D-1, D-2, D-5"
description: "Three habits that stop the defect classes measured on watchpost 0.14.0: one carrier per rule, every invariant must have an input that makes it fail, and every ordering guarantee names the scope it holds within"
version: "1.0.0"
category: implementation
author: "watchpost 0.14.0, T2.2"
attribution: "Extracted from measured defects; sibling to the p10 family, which it does not replace"
triggers:
  - "two carriers"
  - "vacuous invariant"
  - "assertion that cannot fail"
  - "ordering guarantee"
  - "happens before"
  - "one carrier per rule"
requires_tools: []
requires_skills: []
agent_roles:
  - executor
  - tester
  - conductor
family: implementation
family_context: _context.yml
---

# Defect Classes — D-1, D-2, D-5

Three rules, extracted from a single release where each was measured rather than
supposed. They are **habits, not tooling**: nothing here is enforced by a gate, which is
stated plainly because the two rules from the same analysis that COULD be mechanised were
mechanised instead (D-3, and the withdrawal of D-4, both in `06_docs/mutants`).

**The list is meant to shrink.** A rule here that cannot be made mechanical and has
prevented nothing in two releases should be retired rather than kept for completeness.

---

## D-1 — One carrier per rule

**A rule is enforced in exactly one place. Every other site calls that place.**

Two carriers of one rule will disagree, and the disagreement is silent — both copies look
correct in isolation, and the reader who checks one is satisfied.

**How to apply.** When writing a condition, ask whether this rule is already stated
somewhere. If it is, call it. If the existing statement is in the wrong place to be
called, move it rather than copying it. When a registry already answers the question, ask
the registry.

**Watch for the shapes it takes:** a stored flag beside the state it is derived from; a
boolean beside the value it describes; the same predicate written out in two functions;
an exported and an unexported spelling of one operation; a check at one door when there
are two doors.

**Measured (watchpost 0.14.0, four occurrences plus one historical):**

| Occurrence | What the two carriers cost |
|---|---|
| `Tune` lifted the alert duck and `tune` did not | A listener heard the next location at full volume over a breaking alert still reading. *"A rule that lives in the case of an identifier is a rule waiting to be missed."* |
| The wordless-structural-card rule in `Propose` AND in `check` | Deleting either changed nothing, so the mutant for it survived and the rule was untestable |
| `Slot.textAtStandby` written out literally in three places | A slot added to the registry and to one copy but not the other passed the whole suite, and wedged the schedule behind it |
| `Settings.Fenced` beside `Settings.Fence` | A boolean that could disagree with the radius it described |

**The exception, stated so it is not discovered by accident.** A deliberate second carrier
is legitimate when the first cannot be reached from where the check must happen — but it
is then DEFENCE IN DEPTH, it is documented as such at both sites, and its mutant is
expected to survive. If nobody can say why the copy exists, it is not defence in depth.

---

## D-2 — Every invariant must have an input that makes it fail

**Before writing a check, name the input that would make it fail. If none exists, do not
write it.**

A check whose condition is implied by its own enclosing branch, or by the construction of
the code above it, passes for ever. It reads as protection, it is counted as protection,
and it protects nothing — **and no gate can see this.** Mutation testing finds it only if
somebody thought to write that exact mutant.

**How to apply.** Two questions, both cheap:

1. *What input makes this fail?* If the answer is "none", delete it.
2. *Could this check stand in for the rule it is checking?* If deleting the rule leaves
   the check quietly returning the same answer, the rule is written twice (see D-1) and
   the check hides the first copy going missing.

**And the corollary: a rule proved by a probe keeps the probe.** Manual verification that
is not committed as a test is verification that expires when the terminal closes.

**Measured (watchpost 0.14.0, five occurrences):**

| Vacuous check | Why it could not fail |
|---|---|
| `invariant.Check(c.State == Admitted)` inside `if c.State == Admitted` | It asserts the branch condition it sits in |
| `len(runs) <= len(fx)` after a loop appending at most one run per effect | True by construction; the property that mattered was that every effect appears exactly ONCE |
| `cmp` of a file against the source it had just been copied from | Compares a thing to itself |
| A step "never describes two reads" | The one function that emits a read returns at most one, and the second call is guarded by the first |
| An invariant standing in for a magnitude guard | Deleting the guard left the invariant returning the same value in its place |

**Three of the five were added to satisfy a density gate.** That is the moment to be most
suspicious: reaching a number is not the same as stating a property.

---

## D-5 — Every ordering guarantee names its scope, and is tested at that boundary

**"A happens before B" is meaningless without "within what". State the scope in the
comment, and put a test at its edge.**

An ordering that holds within one batch, one step, one lock or one call is not an
ordering that holds. The defect always arrives from just outside the boundary, which is
precisely where nobody looked, because inside it the guarantee is real and demonstrable.

**How to apply.** For every ordering claim, write the sentence with its scope in it —
*"the cue precedes the words, within one step"* — and then ask what happens in the next
step, on the next goroutine, in the next batch. Test THAT. If the honest scope is smaller
than the requirement, the requirement is not met yet, however green the tests are.

**Measured (watchpost 0.14.0, two occurrences, both serious):**

| Guarantee as written | Scope it actually held within | The failure just outside it |
|---|---|---|
| The band's release precedes the next card's cue | One `Step`'s effect list | On the ordinary path the two fall in DIFFERENT steps, because a card finishes while the next card's build is still out. The release then clears a callout that has just gone up |
| The duck precedes the words it brackets | Effects naming the same card | The duck names NO card, so it was dispatched concurrently with the read, and a test asserted that concurrency as correct |

**Both were closed by making the scope real rather than by widening a test:** one lane
that all ordered work passes through, fed from a serial point.

---

## D-11 — A pin is not evidence until it has been seen failing on its own defect

**Run the new test against the BROKEN code and watch it fail, in every environment the gate will run
it in, before calling the defect closed.**

A vacuous invariant cannot fail at all, and that is D-2. This is different and worse to find: a test
that CAN fail, just not on the defect it was written for. It is green, it is counted as a pin, and
the report says "fixed".

**How to apply.** Revert the fix, run the pin, watch it fail, restore the fix. If it does not fail,
the pin is not measuring and the defect is not closed. **A pin for an interleaving sets its own
`GOMAXPROCS`** rather than inheriting it — one processor gives many operations no preemption point,
so the window is never observable and the defective code passes.

**And size a probabilistic probe from a MEASURED rate, stating the measurement.** "It caught it"
is not a number.

**Measured (watchpost 0.14.0, T2.3, two occurrences in one task):**

| The pin | Why it was green on the defective code |
|---|---|
| Waited to OBSERVE the state it raced, then acted | The goroutine was not scheduled until the operation under test had already finished |
| Kept the racing call in flight, 300 rounds | Caught it 4 runs in 5 — a coin toss. Re-sized from the measured rate to 5000 |
| The same, at `GOMAXPROCS=1` | No preemption point between the two halves, so the window never opens. **A one-core CI runner would have reported a live concurrency bug as green** |

**A "tidier" fix cost detection:** a `runtime.Gosched()` in the spin loop cured a timeout and took the
hit rate from 5-in-5 to 1-in-5, buying safety by rarely landing in the window at all. Fixing the
parallelism kept both. Final: the defect fails 13 of 13 across default P, `GOMAXPROCS=1` and `-race`.

**Mechanised half:** a mutant that reintroduces the defect, run to CAUGHT, IS this observation. That
is why pins live beside mutants in `06_docs/mutants/`.

---

## What this skill deliberately does not contain

- **D-3** (a mutant deletes a rule and leaves the tree compiling) and the corpus checks
  are MECHANISED, in `06_docs/mutants`. A rule that a compiler can decide does not belong
  in a document a person has to remember.
- **D-4** (never mutate an assertion) is TRUE AND NOT MECHANISABLE, and was withdrawn
  after three static predicates each flagged legitimate mutants. Weakening an invariant is
  a genuine rule deletion when the code under it can violate the weakened form, and
  vacuous when it cannot; that is semantic. Its decidable form is a triage rule at sweep
  time: a mutant that SURVIVES while editing only an invariant's condition means either an
  unpinned rule or a vacuous check, and both need a person.
- **D-9** (verification logic lives in the project's typed, tested language) is a
  convention, recorded in `06_docs/defect-classes.md`, and is the reason the guards for
  this family are Go tests rather than shell scripts.
