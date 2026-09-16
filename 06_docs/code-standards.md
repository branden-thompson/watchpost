# Code standards this repository enforces

**Every rule below is checked by a tool in this repository, and every tool runs without anything
outside it.** `go run ./tools/authoring` needs only the Go toolchain — no plugin, no CLI, no
configuration. If you can build Watchpost you can run its checks.

**Why this file exists.** The rules originate in a design harness that is deliberately NOT part of
this repository — it is a private toolchain, and `.gitignore` keeps it out. That is right for the
harness and wrong for the rules: a contributor who reads `AP-HIST-01` in a failure message needs
somewhere to look it up, and a reference they cannot reach makes every other reference less
trustworthy. So the IDs stay, and their meanings live here.

---

## Running the checks

```
make lint-authoring          # the rules below, over the whole tree
go run ./tools/authoring     # the same, directly
go run ./tools/authoring -self-test    # prove the checker can fail
go run ./tools/authoring -json         # machine-readable
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

A counterfactual is fine when it carries the reason — *"carried on `BedMsg` it would have three
publishers of which one sets it"* explains the design without narrating a past.

**If a symbol is genuinely deprecated**, use Go's own directive (`// Deprecated:`), not prose.

**When this rule is broken, it is almost always during a REMEDIATION.** A review finds a comment
that asserts something untrue, and the rewrite gets written as an account of the correction, because
the correction is what the author has in mind. Commit-message prose migrates into the source. The
check to run before finishing such a pass: *read this comment as someone who has never seen the
previous version — is any sentence about a state of the code they will never encounter?*

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

---

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
