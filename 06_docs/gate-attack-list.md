# The gate attack list

**Written BEFORE the consolidation it holds to account, and committed first on purpose.** Three
remediation rounds on this layer each verified a fix against the spellings its author happened to
think of, and each was defeated by a cheaper spelling within hours. The order that failed was
*fix → think of an attack → verify*. This inverts it.

**Every row below must become an executable specimen.** A row that is only prose is a row that gets
verified against itself. The model under test therefore parses **strings, not paths** — production
gates hand it the real files, and the specimen table hands it synthetic ones — which is the same
shape `tools/authoring`'s 17-specimen self-test already uses and the reason that detector has never
been defeated.

**The control direction is half the table.** A gate that flags everything passes every attack here
and is useless. Every CAUGHT specimen has a PASSES twin that must stay green.

---

## A. The Makefile model

| # | Attack | Must |
|---|---|---|
| A1 | A gate-shaped target on none of the three lists | CAUGHT |
| A2 | A gate spelled `@scripts/x.sh` — no `./` (the Makefile already spells two targets this way) | CAUGHT |
| A3 | A gate spelled `@python3 scripts/x.py` | CAUGHT |
| A4 | A gate spelled `@bash scripts/x.sh` | CAUGHT |
| A5 | A control commented out with a **tab-indented** `#` (a comment to make, a live command to a naive parser) | CAUGHT |
| A6 | A control neutered with `|| true` | CAUGHT |
| A7 | A control made unreachable: `@true \|\| ./scripts/x.sh --self-test` | CAUGHT |
| A8 | A recipe line prefixed `-` so make ignores its exit code | CAUGHT |
| A9 | A control moved out of a gate recipe into a target that runs nowhere | CAUGHT |
| A10 | A control deleted outright | CAUGHT |
| A11 | A gate running `go test` without `-count=1` | CAUGHT |
| A12 | A build line with **no** `-o` — drops an untrimmed binary in the working directory | CAUGHT |
| A13 | A build line with `-o=path` (no space) | CAUGHT |
| A14 | A build line whose package is a variable, `$(PKG)` | CAUGHT |
| A15 | A build line missing `$(TRIMPATH)` | CAUGHT |
| A16 | `alloc-budget`'s `-run` pattern selecting **nothing** | CAUGHT |
| A17 | A **second** `verify-gates:` rule appending prerequisites (make accumulates; a first-match parser reads one list while make runs another) | CAUGHT |
| A18 | A required gate whose recipe is emptied to `@true` | CAUGHT |
| **A-ok1** | A legitimate tab-indented comment *beside* a live control on the next line | PASSES |
| **A-ok2** | `scripts/lint.sh` genuinely exempt, with its reason | PASSES |
| **A-ok3** | A build line that is correct: `-o`, `$(TRIMPATH)`, `./cmd/watchpost` | PASSES |

## B. The CI model

| # | Attack | Must |
|---|---|---|
| A19 | `- run: make X` with `if: false` on the following line | CAUGHT |
| A20 | `- if: false` as the step's **first** key — the dash sits between indent and key (**this is the spelling that defeated three rounds**) | CAUGHT |
| A21 | `- continue-on-error: true` as the step's first key | CAUGHT |
| A22 | Job-level `if: false` — silences every gate in the job at once | CAUGHT |
| A23 | Job-level `continue-on-error: true` | CAUGHT |
| A24 | `continue-on-error: ${{ true }}` — the expression form, not the literal | CAUGHT |
| A25 | `- run: make X \|\| true` — no condition at all, and the gate cannot fail | CAUGHT |
| A26 | A required gate removed from `ci.yml` only | CAUGHT |
| A27 | A required gate removed from **all three** lists in one commit | CAUGHT |
| **B-ok1** | The two legitimately conditional steps, with their declared reasons | PASSES |
| **B-ok2** | A non-required step carrying `if:` | PASSES |

## C. The exemption registry

| # | Attack | Must |
|---|---|---|
| A28 | A row with an empty reason | CAUGHT |
| A29 | A row with a shrug for a reason — `"n/a"`, `"ok"`, `"todo"` | CAUGHT |
| A30 | A row whose subject no longer exists | CAUGHT |
| A31 | A row whose subject now **satisfies** the rule it is exempt from (stale in the other direction) | CAUGHT |
| A32 | A whole table dropped from the meta-check | CAUGHT |
| A33 | A **new** table added and never registered — the case a hand-written list can never notice | CAUGHT |
| **C-ok1** | Every row in the tree today, with its real reason | PASSES |

## D. The model's own silence (FR-11.3)

`not-applicable`, `could-not-run` and `not-covered` must never render as passed.

| # | Attack | Must |
|---|---|---|
| A34 | A Makefile the model finds **zero** gate-shaped targets in | COULD-NOT-RUN, never pass |
| A35 | A `ci.yml` the model finds **zero** steps in | COULD-NOT-RUN, never pass |
| A36 | An empty `required-gates.txt` | COULD-NOT-RUN, never pass |

---

## What this list does not cover, stated so nobody reads it as wider than it is

- **`app/segments_completeness_test.go`** (F-129, the `:=` shadow) is an AST gate over a different
  artefact. It is a point fix and a fixture, not part of this model.
- **`scripts/quality/lint-ledger.sh`** (F-133, a private copy of its own rule list) is shell, and its
  fix is to count its own rules. Point fix.
- **`tools/authoring`** (F-136, a banner claiming a scope it does not have) is a separate tool.
- **The CI matrix** (F-139, a P10 arm no runner can execute). Point fix.
- This list attacks **parsing and semantics**. It cannot tell whether a gate asserts the *right*
  property — only whether it can be silenced.

## The rule this list exists to enforce

**An attack that is thought of after the fix does not count as a verification.** If an attack occurs
to anyone during or after implementation, it is added here **and to the specimen table**, and the
list is re-run — it is never verified ad hoc and declared closed.
