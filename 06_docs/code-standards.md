# Code standards this repository enforces

**Every rule below is checked by a tool in this repository, to that tool's ceiling, and every tool
runs without anything outside it.** The ceiling is stated with each rule: `AP-HIST-01` is a phrase
list, and a narrating line none of its phrases match needs a reader. `go run ./tools/authoring` needs only the Go toolchain — no plugin, no CLI, no
configuration. If you can build Watchpost you can run its checks.

**Why this file exists.** The rules originate in a design harness that is deliberately NOT part of
this repository — it is a private toolchain, and `.gitignore` keeps it out. That is right for the
harness and wrong for the rules: a contributor who reads `AP-HIST-01` in a failure message needs
somewhere to look it up, and a reference they cannot reach makes every other reference less
trustworthy. So the IDs stay, and their meanings live here.

---

## What a comment is for

These four hold for every comment, Go or `#`. The rules below check what a syntax tree can; the rest is
a reader's, and a reviewer judges a comment against all four.

1. **It describes the code as it is now.** No history: the story of a change is in `git log` and the
   build logs (`AP-HIST-01`).
2. **It is not mechanical.** It does not restate what the line plainly does ("loop over the rows",
   "this is a switch"). The code says that already.
3. **It is about the problem the code solves.** It explains why the code is shaped this way: the
   requirement, the failure it prevents, the ruling it carries, cited by ID (`AP-CITE-01`).
4. **It warns about what is not obvious.** It names the caveats and gotchas a junior engineer would
   not see from the code: a lock order, a call that blocks, what nil means, a line that looks
   tidy-able and is not. A contract a caller must obey may lead in capitals (`AP-CAPS-01`).

A long rationale goes in the doc as a short "Why" list, with its IDs, and the body keeps one-line
comments only where a statement needs its own (0.19.5 D-34; `Broadcaster.stationLine` is the model).

---

## Running the checks

```
make lint-authoring          # the rules below, over the whole tree (SN-02 and SN-03 included), and new citations since the merge-base with main (AP-CITE-01)
go run ./tools/ids -write    # regenerate 06_docs/ids.md after a change to a rulings table; TestTheIndexIsCurrent fails while it is stale
go run ./tools/authoring     # the same, directly
go run ./tools/authoring -self-test    # prove the checker can fail
go run ./tools/authoring -json         # machine-readable
make dupes                   # metric D: a duplicate with no ratified row fails, and so does a ratified row that matches nothing
make lint-imports            # modes/ and platform/ import no domain, and only platform/render imports go-studs (D-10)
go test -run Extending ./cmd/watchpost   # every symbol and path docs/extending.md names exists, and its architecture citations name their file
```

**Every instrument here can prove it works.** `-self-test` runs planted specimens — the bad forms
AND the corrected forms beside them — and fails if any goes undetected or if a good comment is
flagged. A checker that reported nothing on any input would otherwise print a clean bill of health
for ever, and that zero is exactly what a reader would take for good news. `tools/dupes` and
`tools/wires` carry the same flag for the same reason.

---

## `AP-HIST-01` — comments describe the code, not its history

A comment says what the code does and **why it is that way now**. It does not say what it used to
be, what is "legacy", who found a defect, or when something was corrected. That history is already
recorded, accurately and version-scoped, in `git log` — and in this repository also in
`06_docs/follow-ups.md` and each release's `07-readiness/gates.md`.

**Why:** a comment narrating the past rots the moment the code moves on, and it grows without bound —
every fix adds a layer while the code it describes stays the same size. A reader then has to filter
all of it to find the one sentence saying what the function does now.

```go
// no
logger.SetOutput(io.Discard) // used to throw away extra output

// yes
logger.SetOutput(io.Discard) // discard extra log output during tests
```

**Keep the reasoning; drop the narration.** These are not history and must not be removed:

| Keep | Remove |
|---|---|
| `the boundary errs towards telling the listener` | `an earlier version claimed X` |
| `97 cells does not fit every width` | `corrected at D-160`, `found by red team round 3` |
| `the caller holds a.mu` | `this said the opposite for a fortnight` |
| `one key, one meaning per surface (D-56)` — a ruling cited as AUTHORITY | the same ID used to narrate a change |
| `EVERY, not ANY: a burst is one card carrying many hazards` | `the first fix did not work` |
| `a cell is blank only where all four of its points are` | `land beside an empty point was blank`, `batch 97 made the lattice denser` |
| `a box's eight asks one after another take tens of seconds` | `the asks took half a minute`, `under Notes they were never seen` |

A counterfactual is fine when it carries the reason — *"carried on `BedMsg` it would have three
publishers of which one sets it"* explains the design without narrating a past.

**If a symbol is genuinely deprecated**, use Go's own directive (`// Deprecated:`), not prose.

**When this rule is broken, it is almost always during a REMEDIATION.** A review finds a comment
that asserts something untrue, and the rewrite gets written as an account of the correction, because
the correction is what the author has in mind. Commit-message prose migrates into the source. The
check to run before finishing such a pass: *read this comment as someone who has never seen the
previous version — is any sentence about a state of the code they will never encounter?*

**It covers every comment that explains a gate, not only Go's.** `tools/authoring` checks the `#`
comments of every `Makefile`, `.sh` script and `.gitignore` for the same phrases. In those files the
reader is learning what each gate guarantees, and an account of the session that added it tells them
nothing about that. The tool decides only the phrases. A dated incident such as *"it had no target
until 2026-09-06"* still needs a reader, and its present form says what the target guarantees: *"it
is a target, not a command in a document"*. A ruling or a measurement keeps its date: it is the
evidence, not the story.

---

## `AP-DEAD-01` — delete dead code rather than suppressing the compiler

`_ = x` on its own line keeps an unused declaration alive. Delete the declaration, or use it.

**Why:** the assignment hides the code from every other check too — a static analyser reads it as a
use. In this repository a dead closure survived that way with a gate watching the file.

**If the discard is genuine, say why on the line** and the check accepts it:

```go
_ = err // truncated metadata is an error, never a panic
```

---

## `SN-02` — a doc comment belongs to one declaration

Go binds a contiguous comment block to whatever follows it. Two doc comments with **no blank line
between them** are one block: the declaration above loses its documentation, and the one below
acquires a contract describing something else.

```go
// no — Alpha is now undocumented and Beta's doc claims Alpha's contract
// Alpha does the first thing.
// Beta does the second thing.
func Beta() {}

// yes
// Beta does the second thing.
func Beta() {}
```

**Why it needs a tool:** both comments are present and both read well, so review does not catch it
and neither does any line-based check. The signal is structural — a declaration whose own name
arrives partway down its doc has lines above it belonging to something else.

**What it judges:** the doc of every function, type, constant, variable and struct field. An opener
is the subject's name followed by a present-tense verb (`is`, `has`, or any word ending in `s`).
- **Fused:** the subject's opener arrives after line 0.
- **Doubled:** the doc opens with the subject's opener and has a second one. A reader cannot tell
  which half is true, so keep the one that matches the code and delete the other.

A line that merely starts with the name is not always an opener. A later line counts when the line
above it ends a sentence and is not a paragraph break, or when the block opens by naming another
declaration. So prose that wraps onto the name (`… A nil` / `frameChecks checks every frame.`) passes,
and so does a new paragraph that opens with it, as godoc's own examples do.

```go
// no — two docs for one field, and they disagree about its return
// read returns whether the words ran out.
// read installs the card and returns its error.
read func() error

// yes — one doc; a later paragraph may open with the name
// read installs the card and returns its error.
//
// read blocks for the length of the read.
read func() error
```

---

## `SN-03` — one package comment per package

`go doc` joins every package comment in a package's files into one, in file order. A fragment above
`package` in a second file is printed as part of the package's front door.

```go
// no — in app/setup.go, beside app.go's "Package app is …"
// Setup: the first-run questions live in a …
package app

// yes — a file header, detached by a blank line
// Setup: the first-run questions live in a …

package app
```

**Why it needs a tool:** each fragment reads well in its own file, and nobody reads all of them
together except `go doc`. A test file's package comment does not count; `go doc` does not read it.

---

## `AP-SHELL-01` — everything is Go; shell needs a ruling

**Rule.** A Go file never carries a program in shell: no string literal that is a shebang line
followed by a body, and no `exec.Command("sh"|"bash", …, "-c", …)`. A shebang line ALONE is a file
header (the gate oracle's fixtures write one) and is allowed; a regex about shebangs is allowed;
running a program directly (`exec.Command("git", "-c", …)`) is allowed.

**Why.** Standing rule, HUM LEAD 2026-09-17: everything is Go unless absolutely necessary, and
"necessary" is a ruling to ask for, not a comment to write. The gate oracle is the worked example:
three rounds slipped ~70 lines of shell into it as stubs, and the next blind adversary's Criticals
were bugs in exactly that shell. A stub is a small Go program built at test time
(`tools/gateoracle/stub`); a checker is a `tools/` main.

**No exemption marker.** A lint that accepts `// shell: because` accepts every because. If shell
is genuinely necessary, stop and bring the lines and the reason to the HUM LEAD; a ruling adds a
row to the shell ledger (`cmd/watchpost/shell_test.go`), which is the same path every existing
`scripts/*.sh` is on — each row is a port candidate.

## `AP-CITE-01` — a new citation names its release

Every release numbers its rulings, requirements and findings from 1, so a bare `D-30` names a
different row in each release that used it. A comment written now says which one it means.

```go
// no — which release's D-30?
priorityRows int // favourites shown: fewer on a short terminal (D-30)

// yes
priorityRows int // favourites shown: fewer on a short terminal (0.19.5 D-30)
```

- **What counts as qualified.** A release number earlier in the same sentence qualifies every ID
  after it, across the lines of one comment, as in "is 0.19.5 Q7.4 (JM-3, DC-14)". A full stop ends
  the sentence. A release's own prefix (`MVS-D-80`) is qualified too.
- **What counts as new.** An ID the package's Go files did not carry at the merge-base with `main`,
  so a comment moved between files of one package keeps its citations. Older bare citations stay as
  they are; `06_docs/ids.md` resolves them.
- **Families.** They are `tools/ids/idset`'s list, the one `06_docs/ids.md` is built from.

**Why it needs a tool:** the reader of a citation is the one who pays for a missing release, often
releases later, and the author never sees the ambiguity, because to them the number is obvious.

---

## `AP-CAPS-01` — an all-caps headline is for a contract

An all-caps sentence opening a paragraph is a headline, and it is kept for **a contract a caller must
obey**. Examples are blocking, locking, never calling from render, and what nil means. Rationale,
history of a choice and the reason a thing is not something else are written in sentence case.
**No paragraph opens with an all-caps AND, BUT or SO:** such a paragraph means something only after
the one above it, and loses its sense when either moves.

```go
// no — a continuation, and a reason dressed as a contract
// AND THE REFUSAL IS CARRIED TO THE SURFACE THAT WAS REFUSED. A refusal
// recorded and drawn nowhere is indistinguishable from a broken control.

// yes — a contract as the headline, the reason in sentence case
// IT BLOCKS FOR THE LENGTH OF THE READ. The Speak effect comes home Finished
// when the card has actually been said.
//
// The refusal is carried to the surface that was refused: a refusal recorded
// and drawn nowhere is indistinguishable from a broken control.
```

- **What is judged, and when.** The rule is brought to code as it is edited (0.19.5 D-12). "Touched"
  means a declaration the branch changed, its doc included, plus the whole of a file a task rewrites.
- **What the tool decides.** `tools/authoring` reports a caps continuation in a changed declaration
  as a **soft warning**, printed and never failing the gate. Over the whole tree it has many standing
  hits, and a gate that fired on each would be ignored.
- **What it leaves to a reader.** Whether a headline is a contract. A sentence-case "And" that
  continues a sequence in the same function is allowed.

---

## `AP-OK-01` — a value's bool is checked, not thrown away

**Rule.** A call to one of the module's functions or methods that returns a value and a `bool` never
assigns the bool to `_`: `v, _ := f()` and `v, _ = x.f()` are findings. Handle the bool (`if v, ok :=
f(); ok { … }`), or give the callers that need only the value a function that returns only the value
(`NormalizeKey` beside `NormalizeID`; `chosenOn` beside `choiceOf`). A map lookup, a type assertion and
an `error` discard (P10-07's, `errcheck`) are not this rule's. Test files are not judged.

**Why.** #27 (0.19.0 D-142): `year, _ := readAs(...)` carried on with an empty year when the read
failed, and wrote it over a year of real roll-ups. P10-07 checks `error` returns only and treats `_` as
deliberate, so it cannot see a discarded bool. The zero value a failed call returns is a value the
code then trusts.

**A syntax tree has no types,** so the callee is known by name, and a name some declaration returns
otherwise is left alone rather than guessed at. The self-test plants both directions, including the
ambiguous name.

**No exemption marker,** for AP-SHELL-01's reason: a bool that truly does not matter to a caller is a
sign the caller wants a different function.

## The other gates

| Command | What it holds |
|---|---|
| `make verify` | every required gate; the list is `06_docs/required-gates.txt` |
| `make dupes` | duplicate function bodies, against a ratified ledger |
| `make wires` | closed-set members that nothing reads or writes |
| `make mutant-anchors` | every mutant still finds the line its rule lives on |
| `make mutant-check` | every mutant still applies and compiles |
| `make alloc-budget` | allocation pins on the paths that have them |

**A required gate appears in three lists** — the `verify` target, `.github/workflows/ci.yml`, and
`06_docs/required-gates.txt` — so a gate cannot silently appear or disappear. `cmd/watchpost/gates_test.go`
fails if they disagree.

**Some checks state their own blind spot** next to their result, and that is deliberate: a number
published without what it cannot see is a number that lies by omission. `dupes` says it is a floor;
`tools/authoring` says it cannot see a comment that is merely wrong, which is the larger class and
still needs a reader.

---

## P10 — the safety-critical rules

A comment that cites `P10-02` or `P10-05` is citing one of these. They are Holzmann's "Power of Ten"
rules for safety-critical code, as the A2DH harness applies them to Go. **The checker and its
exemptions ledger live in the harness, outside this public tree.** So the thresholds below are the
checker's own, taken from its pinned configuration, and this section is where the tree states them
(0.19.5 A-2).

**Who runs it.** `make p10` runs the harness's `a2dh p10 check`. It is not part of `make verify` or CI,
whose runners have no harness; it fails loud rather than skipping when the CLI is absent. It runs at
each batch and at BUILD and REVIEW exit, under `make quality`, and its figures go in the build log.
**An exemption is ratified by the HUM LEAD, never self-issued.** `make p10` regenerates the ledger's
public mirror, `06_docs/p10-ledger.md`, and fails when the committed one is stale.

**What it judges.** Production code only: tests and `testdata` are out of scope. P10-05 counts only
the functions a branch changed since its merge-base with `main`.

| Rule | It limits | Threshold | The accepted forms |
|---|---|---|---|
| `P10-00` build coverage | a changed `.go` file the analysed build configuration excludes | none may be unanalysed | run the check under the file's `GOOS`/`GOARCH` (`app/dump_windows.go` is checked on Windows), or a ratified exemption |
| `P10-01` control flow | `goto`, `panic`, recursion | none | an error return; a loop with a bound in place of recursion. Recursion is found by **bare name** across the package, so a `New` calling `other.New` reads as recursion; call a package-level constructor value instead |
| `P10-02` bounded loops | a loop with no visible bound | `for {}` and `for cond {}` are flagged | `range`, or a three-clause counter. This tree's convention adds `// bounded by <what> (P10-02)` to say the bound to a reader; the checker does not read it |
| `P10-03` dynamic memory | (allocation after start-up, in C) | not applicable to Go | ruled out by the harness, with its rationale |
| `P10-04` small functions | a function's length and branching | at most 60 lines and 40 statements; cyclomatic complexity under 15 | split into named steps. Comment lines are not counted against the 60, as the specimen below shows |
| `P10-05` invariant density | checks per changed function, per package | 2.0, or a floor the HUM LEAD ratified for the package (`FLOOR x.xx`, held by `tools/p10floor`) | `invariant.Check(ok, …)` whose arguments make no call (`len` and `cap` aside), so `ok := !t.Before(x)` comes first; and a guard, `if <call-free condition> { return … }`. A check is added where it protects a real state, never for the count (0.19.5 D-27) |
| `P10-06` minimal scope | package-level variables, and stores never read | none | a constructor function returning the value (`setupTable`, below); a variable declared where it is used |
| `P10-07` checked returns | an ignored error, an unused parameter or result | none | handle the error, or discard it explicitly with `_ =` and a reason on the line |
| `P10-08` directives | `//go:linkname` and cgo; `//go:build` and `//go:generate` | the first two banned; the second two listed | a ratified exemption per build or generate directive |
| `P10-09` pointer discipline | `unsafe`; `reflect`; dereference depth | `unsafe` banned, `reflect` banned unless justified, depth at most 1 | a value or a single pointer |
| `P10-10` zero warnings | any other analyser's warning | none | rewrite the code until the tool has nothing to say |

**A specimen: `setupTable` (`modes/tty/setup_rows.go`).**
- **P10-06.** It is a function returning the Settings table, not a package variable. A
  package-level table would be state any code could reassign, and each caller wants its own copy.
- **P10-04.** Its body is one statement, a keyed composite literal of 57 rows, one per line (0.19.5
  Q6.4). It runs 101 lines: 59 of code and 34 of comments. P10 reports nothing on it, although
  batch 19 changed it, so the comments are not counted against the 60 (measured 2026-11-07). A
  few more rows reach the limit.
- **The route at the limit.** Split the table by Settings group into named builders, or ask the HUM
  LEAD to ratify the literal as one table. Moving rows into a package variable is not a route: it
  trades P10-04 for P10-06.
