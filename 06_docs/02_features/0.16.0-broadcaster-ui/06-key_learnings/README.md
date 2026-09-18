---
title: "0.16.0 — key learnings"
status: "Opened at BUILD exit 2026-09-15.  REFLECT fills this; what is here now is what BUILD already knows."
---

# Key learnings — 0.16.0 Broadcaster UI

**This folder is a REFLECT artifact and is opened, not finished, at BUILD exit.**  It exists now
because SEV-0 requires all seven folders and red team found it absent — and because the release
already knows things worth writing down before the retrospective reorganises them.

## What BUILD already knows

**1. A fix is new code, and it arrives with less scrutiny than the code it replaces.**  Remediating
nine red-team findings introduced three new defects — a duplicate found by `dupes`, a corpus anchor
found by `mutant-anchors`, and a shared-primitive "fix" that broke the invariant it was meant to
serve.  All three were caught by gates, none by the author.  The probes are in
`06_docs/quality-observations.md`.

**2. A rule taught to one caller and not its neighbours is this codebase's dominant defect shape.**
Round 1's two criticals were one root — a fence predicate taught to the function that AIRS a card and
the one that DRAWS it, not the one that PREPARES it or the one that ducks the bed.  **Round 2 then
found a fourth site the remediation had missed.**  A policy with two askers and three non-askers is
the shape; the tell is a comment on one call site explaining why the rule is needed *there*.

**3. A test that cannot fail is worse than no test.**  Seven instances in one release.  The counter is
one command — re-apply the defect and watch the new test fail — and it caught three of them.

**4. A citation can hide a hole.**  `gates.md` named a successor for a retired safety gate; the
successor asserts something else, and underneath it FR-5.5 had **no test and no implementation**
(F-109).  A roster is evidence, and evidence rots at the rate the code changes.

**5. Blind agents find what self-review cannot, and the AXES are load-bearing.**  The docs/hygiene
agent's most valuable finding in round 1 was that the tree was uncommitted — no code-quality axis
would ever have found it.  In round 2 the same agent audited the BUILD report and found three
over-claims in it.

## What REFLECT still owes

- Whether the 0.16.0 corpus grew in the right places, or only where it was easy.
- Whether `platform/debounce` earns being a package, or wants a second caller first.
- The cost-per-defect of the two red-team rounds against the gate set they ran past.
