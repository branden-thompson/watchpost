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
| J2 | Two gates consolidated into a pattern rule `%-tags:` with a neutered recipe | CAUGHT — if the gate is NOT in `.PHONY` it is absent from the database (J1's audit); if it IS, make says "nothing to be done" and exits 0, which the green/red run catches. Either way |
| J3 | A gate's rule deleted and `.DEFAULT:` supplying a green recipe for any name | CAUGHT — same rule as J2 |
| J4 | A prerequisite that is red under green stubs (e.g. `release-matrix` running `shasum` over nothing) | UNJUDGEABLE, by name |

## K. The control proof paints the wrong thing — closed by stubs that answer by argument

| # | Attack | Must |
|---|---|---|
| K1 | The carrier runs the checker AND its `--self-test \|\| exit 0`: painting the checker red makes the carrier red for the checker's sake, and the neutered control passes | CAUGHT — the stub answers `--self-test` with the CONTROL status and anything else with the CHECKER status; the proof paints the control status alone |
| K2 | Sibling `_test.sh \|\| exit 0` in one gate and the flag behind `\|\| exit 0` in another; `controlOf` is last-gate-wins | CAUGHT — every carrier is painted and tried |
| K3 | A checker with no extension (`scripts/lint-x`) or `.bash` — not a checker to the regex, so no stub and no control proof | CAUGHT — any `scripts/…` token is a checker |
| K4 | Two controls joined by `;` in one carrier — the line's status is the LAST command's, so the first control's failure is discarded | CAUGHT (an earlier draft of this row said PASSES; the reviewer's R4c was right) |
| **K-ok** | Two controls joined by `&&` | PASSES — both reached |

## L. The registry chooses its own control — narrowed

| # | Attack | Verdict |
|---|---|---|
| L1 | `exists: func(_, s) bool { return s != "absent-thing" }` — honest for the declared control, dishonest for everything else | **CLOSED for `exists`:** the REGISTRY generates the absent subject (a nonce) so a table cannot choose it |
| L-ceiling | The same shape on `stillNeeded`, whose `satisfied` subject the table must name because only the table knows what satisfies its rule | **Declared.** A table can be honest for one subject and dishonest elsewhere; the registry cannot see inside a function |

## What round three still cannot see, declared

- **A real tool that succeeds on an empty tree** (`/usr/local/go/bin/gofmt -l .` by absolute path, `find … -exec gofmt`) makes the oracle refuse a gate that CAN fail — the false-positive direction, loud, never silent.
- **A double-colon rule** whose second part is empty overwrites the first in the database parse — loud, never silent.
- **The CI half** remains a ratchet; G2–G4 stand.

---

# Round four: paint one thing, not the world

**Written and committed before the fix.** A fourth blind adversary found three Criticals and named
them as one shape: *the oracle paints the WORLD, then asks one bit.* Green-under-green proves the
gate is judgeable; red-under-red proves it is red when EVERYTHING is red — and neither proves it is
red BECAUSE OF ITS OWN CHECK. `./scripts/x.sh || exit 0` followed by `go version` is green under
green, red under red, and neutered. In the real Makefile: `mutant-check`'s verdict replaced by
`exit 0` read as sound (the `$(MAKE) cache-clean` stub reds it); `lint-injector … || true` — the sole
debug-injector check before a release — read as sound (release-matrix's `go build` stub reds it).

**The fix is the control proof's own method, applied everywhere.** For each required gate, for each
stub its recipe REACHES — its own checkers, every toolchain sub-command, its prerequisites' recipes,
the targets it recurses into with `$(MAKE)` — paint ONLY that one red and require the gate red.
Toolchain stubs answer by sub-command (`go test` ≠ `go clean` ≠ `go build`), or `mutant-check`'s
two `go` calls are the same stub and the discard stays invisible. The count fell 6 → 4 → 3 across
rounds and each round's Criticals collapsed to one mechanism; this is convergence, and the reason a
fourth batch is built rather than stopped.

## M. Red for the wrong reason — closed by per-key painting

| # | Attack | Must |
|---|---|---|
| M1 | `./scripts/x.sh \|\| exit 0` then `go version` — red-under-red from the wrong stub | CAUGHT — with only `scripts/x.sh` red, the gate is green |
| M2 | `gate: other-gate` as a prerequisite, own line `\|\| exit 0` — the prerequisite reds it | CAUGHT |
| M3 | `OK := $(shell go version && echo yes)` at parse time, `test "$(OK)" = yes \|\| exit 1` before `\|\| exit 0` | CAUGHT |
| M4 | **Real tree:** `mutant-check`'s `exit $$rc` → `exit 0` | CAUGHT — paint `go:test` alone red; the gate must go red |
| M5 | **Real tree:** `install-test.sh \|\| true` behind `release-matrix`'s red `go build` | CAUGHT |
| M6 | **Real tree:** `lint-injector.sh … \|\| true` — the debug-injector check | CAUGHT |
| M7 | `verify: lint` as a prerequisite, `-@` on treelock — verify red for `lint`'s sake and green for every other gate's | CAUGHT — every invoked checker is painted in turn and verify must go red each time; not `invoked[0]` |
| **M-ok** | The real Makefile: every required gate goes red for EACH stub it reaches, painted alone | PASSES — or the recipe has a status discard, named by key |

## N. The escapes — closed by deletion and by walking

| # | Attack | Must |
|---|---|---|
| N1 | Delete a gate's rule, prepend `.DEFAULT:` green, add a 20-character `ciOnly` row — every executed check skipped, `make <gate>` prints "skipped" and exits 0 | CAUGHT — **a `ciOnly` row exempts a gate from `verify`, never from having a phony rule.** A required gate absent from the database is an error unconditionally |
| N2 | `gate: gate-run` with the recipe on `gate-run`, which is not phony — `touch gate-run` silences it while `gate` is phony | CAUGHT — the audit walks prerequisites; every non-phony node WITH a recipe is the finding |
| N3 | `exists: s != absentNonce` — the nonce is a package constant a table can name; or match by shape `HasPrefix("__registry")` | CAUGHT — a fresh random string per call, no fixed prefix, never a package identifier |
| N4 | `MAKEFLAGS=i` in the environment — the oracle runs inside `make race`, so a parent `-i` or `-n` reaches every child make | CAUGHT — `MAKEFLAGS` and `MAKELEVEL` are stripped from the child environment |

## Declared this round

- The `go` stub's treelock contract matches `go run ./tools/treelock` exactly. `./tools/treelock/`,
  `github.com/…/tools/treelock`, or another wrapper make `verify` UNJUDGEABLE — **the green run
  must reach the gates, and now says so when it does not** rather than diagnosing a `-` that is not
  there.
- A checker referenced nowhere by a `scripts/` token (`cd scripts && ./x.sh`) is red under green —
  loud.

---

# Round five: observe, don't guess

**Written and committed before the fix.** A fifth blind adversary found six Criticals, and every one
was the same defect: `reach()` — the function that decides which stubs a gate can touch — was a set
of regexes over recipe text. `$(A2DH)` is not the literal `a2dh`; `$$(./scripts/x.sh)` has `(`
before `scripts/`; `$(CURDIR)/scripts/x.sh` has `/`; `$(MAKE) -s target` has a short flag; two
`go run` calls collapse to one key; a pattern-rule prerequisite has no recipe in the database. In
the tree AS SHIPPED, `p10`'s check — `$(A2DH) p10 check` — had never once been painted red alone.

**That is round one's shape, in the one function where round one's lesson had not been applied.**
Parsing was replaced by execution for "does this gate fail?" and then quietly reintroduced for
"what does this gate reach?" — so the count rose back to six for round one's reason.

**So the stubs RECORD what make ran.** Every stub appends its key to a log when it is invoked; the
green run's log IS the reach — whatever make actually executed, through variables, substitutions,
absolute paths, `sh -c`, `xargs`, short-flag recursion, pattern rules — and the per-key proof paints
exactly what was recorded. There is no regex anywhere in the executed half. The same move replaces
the database-parsed phony audit: create a file named after each prerequisite, run green, and if the
recorded reach SHRINKS, that file silences the gate. Two text layers deleted, not patched.

**The upstream learning, stated once for li-A2DH:** a semantic question about a language — what
runs, what can fail, what is reached — is answered by executing the language and observing, never by
pattern-matching its source. A parser reaches the spellings its author imagined; an observer reaches
whatever happened. This held for the Makefile at every one of five rounds, and it is the shape of
every Critical this layer has ever had.

## O. Reach that text cannot see — closed by recording

| # | Attack | Must |
|---|---|---|
| O1 | Two `go run ./tools/X` calls in one gate, the check one `\|\| true`, the self-test one live — one `go:run` key hid the discard | CAUGHT — the go stub records `go:run:<pkg>` and `:ctl` when it sees a self-test flag; two calls, two keys |
| O2 | A tool named through a variable: `$(A2DH) p10 check … \|\| true` — **the shipped `p10`** | CAUGHT — make expands `$(A2DH)`, the stub runs and records itself |
| O3 | A checker inside command substitution: `out=$$(./scripts/x.sh) \|\| true` — **the shipped `fmt` shape** | CAUGHT — the stub records itself whatever surrounds it |
| O4 | A checker by absolute path: `$(CURDIR)/scripts/x.sh \|\| true`; `sh -c "./scripts/x.sh" \|\| true` | CAUGHT — same |
| O5 | `$(MAKE) -s target` recursion into a neutered target | CAUGHT — recursion is real make; the stub records |
| O6 | A pattern-rule prerequisite (`%-run:`) supplying a neutered recipe; `touch <prereq>` silences it | CAUGHT — the stub records through the pattern rule, and the touch test sees the reach shrink |
| O7 | `MAKEFILES=<file>` in the environment setting `.SHELLFLAGS := -ec` — every `;` discard becomes red, and the oracle certifies `exit 0` verdicts as sound (the QUIET direction) | CAUGHT — `MAKEFILES`, `GNUMAKEFLAGS`, `MAKE`, `MAKEOVERRIDES` are stripped; specimen M4 must still CAUGHT with `MAKEFILES` set in the parent |
| ~~O-ok~~ **O9** | `@go version \|\| true` above a live checker — a "tolerated diagnostic" | **Changed while building, before the fix was trusted: CAUGHT, not PASSES.** The relaxation as written would have read text to decide which key is a diagnostic — and `./scripts/x.sh \|\| true` is the same shape. There is no text in the executed half, so there is no relaxation: a diagnostic under `\|\| true` in a gate is refused, and the message says to move it out or let it fail. The real Makefile has none |
| O8 | `python3 -m lint_a \|\| true` — an interpreter run with no `scripts/` argument | CAUGHT — refused as UNJUDGEABLE, never unseen |
| **O10** | **Found by the first recorded run of the SHIPPED tree**, not by an adversary: `release-matrix`'s `(command -v sha256sum && sha256sum … \|\| shasum …)` — `A && B \|\| C` runs C when B fails, so a failed `sha256sum` fell through to a successful `shasum` and its failure was gone. `sha256sum` was not on the regex oracle's tool list, so it was never painted | CAUGHT — and fixed as an `if`. The regex oracle could not have seen it; the record did on its first run |
| **O-ok2** | The real Makefile, recorded rather than regex-reached: every required gate goes red for each recorded key | PASSES — and `p10` is judged for `a2dh` for the first time |

## Declared

- `python3 -m module` and `python3 - < script` are refused as UNJUDGEABLE (the interpreter stub
  execs a `scripts/` argument only). Loud. (O8)
- An order-only directory prerequisite (`gate: \| out`) is touched like any other node; creating a
  file where `mkdir -p` needs a directory makes the gate RED, which is not silence, so it passes
  (O-ok). Verified rather than declared.
- **The tool list is the one list left**: `go`, `gofmt`, `a2dh`, `python3`, `expect`,
  `golangci-lint`, `govulncheck`, `shasum`, `sha256sum` are stubbed; any other command a recipe runs
  is real. A real tool on an empty tree is red under green (loud), but a real tool that HAPPENS to
  exit 0 on an empty tree under `\|\| true` is not judged. Adding a checker that is neither a
  script nor `go run` means adding it to the list — and O10 is what forgetting looks like.
