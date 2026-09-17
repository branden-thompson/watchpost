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
| A37 | A **column-0 comment between two recipe lines** — make ignores it and the recipe continues; a parser that resets on it drops every line after (found by the real `mutant-check`, which carries four paragraphs between its `mkdir` and its `go test`) | the second line is still a live check — PASSES |
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

---

# Round two: attacks the parser cannot see, and the oracle that can

**Written and committed before the oracle it holds to account.** A blind reviewer defeated the
consolidated model six ways in one sitting, and named the cause exactly: *every Critical is the model
deciding a semantic question — does this line run, can it fail, is this a check — by pattern on a
fragment, in a language whose global constructs it never reads. The attack list enumerates spellings;
the parser enumerates the same spellings back.*

**So the Makefile half stops parsing and starts EXECUTING.** Every project checker is replaced by a
stub that exits non-zero, every toolchain command on PATH likewise, and `make <gate>` is run for every
required gate. **If make exits zero, the gate cannot fail, whatever the spelling.** Make and sh are
the oracle; the list below no longer has to imagine every way a status can be discarded, because the
shell decides. Target discovery comes from `make -pn` — make's own expanded database — so a target
named through a variable, an include or a conditional is found because make found it.

**The CI half has no oracle.** Nothing here executes GitHub Actions. Its checks are a **ratchet on
spellings** and are labelled as one; the guard for CI is that CI runs and its per-gate results are
read, which the workflow's one-step-per-gate shape already provides. Section G states the ceiling.

## E. Make semantics — executed, never parsed

Every row: plant the edit in a scratch copy, stub every checker red, run `make <gate>`, assert
**non-zero**. `CAUGHT` means the oracle reports the gate cannot fail.

| # | Attack | Must |
|---|---|---|
| E1 | `.IGNORE:` at the top of the Makefile — every recipe's status ignored | CAUGHT |
| E2 | `MAKEFLAGS += -i` | CAUGHT |
| E3 | `SHELL := /usr/bin/true` — every recipe "succeeds" | CAUGHT |
| E4 | `.SHELLFLAGS := -c :` — **the reviewer's spelling, `-c true; #`, was inert** on 3.81 AND 4.4.1, probed directly; `-c :`, `-c "true ;"` and `-c true \#` all silence the gate | CAUGHT. **The oracle requires GNU Make ≥ 3.82** (macOS ships 3.81; `brew install make`) so its verdict is the one CI would give |
| E5 | `-` prefix on the check line | CAUGHT |
| E6 | `\|\| exit 0` | CAUGHT |
| E7 | `\|\| echo skipped` — any word that succeeds | CAUGHT |
| E8 | `; exit 0` | CAUGHT |
| E9 | `\| tee log` with no `pipefail` — the status is tee's | CAUGHT |
| E10 | The check replaced by `echo "…scripts/x.sh…"` — the path is text, nothing runs | CAUGHT |
| E11 | The gate's target named through a variable, `$(GATE):`, with a neutered recipe | CAUGHT — `make -pn` resolves the name; the oracle executes it |
| E12 | The gate defined in an `include`d file, neutered there | CAUGHT — `make -pn` follows the include |
| E13 | `ifeq` with a real branch and a `@true` branch, the `@true` branch active | CAUGHT — `make -pn` shows the active branch; the oracle executes it |
| E14 | A `define`/`endef` block holding a control that looks live | not a control — the database lists no recipe for it |
| E15 | `-@go run ./tools/treelock … $(MAKE) verify-gates` — `make verify` itself exits 0 on a red gate | CAUGHT |
| **E-ok** | The real Makefile, every checker stubbed red: every required gate exits non-zero | PASSES — this is the control, and the whole oracle is void if it does not hold |

## F. The registry's own trust (FR-11.6)

An instrument answers a KNOWN case before it is believed about an unknown one. Every table must
prove its two functions CAN return false.

| # | Attack | Must |
|---|---|---|
| F1 | A table whose `exists` always returns true | CAUGHT — the table's declared `absent` subject must resolve to false |
| F2 | A table whose `stillNeeded` always returns true | CAUGHT — the table's declared `satisfied` subject must resolve to false |
| F3 | A table declared `var x map[string]string` and filled in `init()` | discovered |
| F4 | A table declared in a non-`_test.go` file of the package | discovered |
| F5 | A name in `notATable` that no longer exists | CAUGHT |
| F6 | A28–A33 from round one, each now an executable specimen | CAUGHT, and C-ok1 PASSES |
| **F-ceiling** | A table registered under a type alias (`type rowsT = map[string]string`) | **NOT caught** — no type-checker is loaded, and one will not be taken for a test. Declared |

## G. CI — the ceiling, declared

| # | Attack | Verdict |
|---|---|---|
| G1 | `if : false` — a space before the colon, which YAML strips | CAUGHT (a one-character fix, and the fourth spelling of the class that defeated three rounds) |
| G2 | A required job made to `needs:` a job that never runs | **NOT catchable** without workflow semantics. Declared |
| G3 | `on.push.branches: [never-exists]` | **NOT catchable**. Declared |
| G4 | `strategy.matrix: ${{ fromJSON('{"os":[]}') }}` | **NOT catchable**. Declared |
| G5 | `run: \|` block scalar with `make x` on the second line | reads as ABSENT — fires, wrong reason; the false-positive direction only |

**The CI checks' banner must say this.** A check that claims to close a class it cannot see is the
defect this project names most often.

---

# Round three: what execution introduces that parsing did not have

**Written and committed before the fixes.** A blind adversary found four Criticals in the executed
oracle, three of them one class — **the scratch tree is not the real tree**, so a gate can be red in
the oracle for a reason unrelated to its check and green forever in the tree it judges — and named
the missing instrument: **the oracle had no positive control.** Nothing proved a gate goes GREEN when
every stub is green. FR-11.6: an instrument answers a KNOWN case before it is believed about an
unknown one. Every executed assertion now runs the green case first, and a gate that is red under
green stubs is **UNJUDGEABLE — COULD-NOT-RUN, by name** — never "sound".

**Why this round is different from the last two.** Round one found six spellings and needed six
regexes. Round two found four Criticals with **one mechanism** behind three of them. The count fell
and the fixes changed kind. That is the convergence signal, and it is why a third batch is worth
building rather than stopping.

## H. Scratch-tree divergence — closed by the green control

| # | Attack | Must |
|---|---|---|
| H1 | `@test -f go.mod \|\| exit 1` as a preflight, then `./scripts/x.sh \|\| exit 0` — red in scratch (no go.mod), green in the tree | **UNJUDGEABLE** under green stubs (the preflight is red) — reported by name, never "can fail" |
| H2 | `@go version >/dev/null \|\| exit 1` as the preflight | UNJUDGEABLE, by name |
| H3 | `@git diff --quiet \|\| exit 1` — the dirty-tree refusal this project already writes | UNJUDGEABLE, by name |
| H4 | A required gate whose PREREQUISITE is red under stubs (`install-test: release-matrix`), its own line `\|\| exit 0` | UNJUDGEABLE, by name — forces a decision instead of a false "sound" |
| H5 | `verify:` with a preflight, then `-@go run ./tools/treelock … $(MAKE) verify-gates` | CAUGHT — with ONE checker red and everything else green, `make verify` must go red; the `-` makes it exit 0 |
| **H-ok** | The real Makefile: every required gate exits 0 with every stub green, and non-zero with every stub red | PASSES — both halves, or the oracle is void |

## J. Discovery is `.PHONY`-dependent — closed by the phony audit

| # | Attack | Must |
|---|---|---|
| J1 | A required gate not in `.PHONY`: `touch <gate>` in the tree → "is up to date", exit 0, nothing ran. **Four required gates were in this state in the real Makefile with no edit at all** | CAUGHT — every required make target must be `#  Phony target` in make's database |
| J2 | Two gates consolidated into a pattern rule `%-tags:` with a neutered recipe — absent from the database, still run by make | CAUGHT — a required gate absent from the database is an error unless DECLARED CI-only; "missing" no longer means "fine" |
| J3 | A gate's rule deleted and `.DEFAULT:` supplying a green recipe for any name | CAUGHT — same rule as J2 |
| J4 | A prerequisite that is red under green stubs (e.g. `release-matrix` running `shasum` over nothing) | UNJUDGEABLE, by name |

## K. The control proof paints the wrong thing — closed by stubs that answer by argument

| # | Attack | Must |
|---|---|---|
| K1 | The carrier runs the checker AND its `--self-test \|\| exit 0`: painting the checker red makes the carrier red for the checker's sake, and the neutered control passes | CAUGHT — the stub answers `--self-test` with the CONTROL status and anything else with the CHECKER status; the proof paints the control status alone |
| K2 | Sibling `_test.sh \|\| exit 0` in one gate and the flag behind `\|\| exit 0` in another; `controlOf` is last-gate-wins | CAUGHT — every carrier is painted and tried |
| K3 | A checker with no extension (`scripts/lint-x`) or `.bash` — not a checker to the regex, so no stub and no control proof | CAUGHT — any `scripts/…` token is a checker |
| **K-ok** | Two controls joined by `;` in one carrier | PASSES (each is reached) |

## L. The registry chooses its own control — narrowed

| # | Attack | Verdict |
|---|---|---|
| L1 | `exists: func(_, s) bool { return s != "absent-thing" }` — honest for the declared control, dishonest for everything else | **CLOSED for `exists`:** the REGISTRY generates the absent subject (a nonce) so a table cannot choose it |
| L-ceiling | The same shape on `stillNeeded`, whose `satisfied` subject the table must name because only the table knows what satisfies its rule | **Declared.** A table can be honest for one subject and dishonest elsewhere; the registry cannot see inside a function |

## What round three still cannot see, declared

- **A real tool that succeeds on an empty tree** (`/usr/local/go/bin/gofmt -l .` by absolute path, `find … -exec gofmt`) makes the oracle refuse a gate that CAN fail — the false-positive direction, loud, never silent.
- **A double-colon rule** whose second part is empty overwrites the first in the database parse — loud, never silent.
- **The CI half** remains a ratchet; G2–G4 stand.
