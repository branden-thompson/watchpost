// Package gateoracle judges a Makefile's gates by RUNNING them in a copy of the
// tree as CI has it, with every checker and toolchain command replaced by a stub
// that records its invocation and answers with a painted status.
//
// WHY EXECUTION. "Can this gate fail?" is a question about what make and sh DO,
// and a parser answers it only for the spellings its author imagined. Six blind
// adversarial rounds each defeated a text-reading layer; none defeated the
// executed one. So make and sh are the oracle and nothing here reads a recipe.
//
// WHY OBSERVATION. "What does this gate reach?" is the same kind of question. The
// stubs RECORD every invocation, and the green run's record IS the reach —
// through variables, substitutions, absolute paths, wrappers and recursion.
//
// WHY THE TREE, AS CI HAS IT. An empty scratch answers every predicate a recipe
// puts to the tree the opposite way the repository does, and a developer's tree
// answers differently from CI's (HEAD attached, tags present, ignored files
// present). The scratch is a shared clone at the commit, HEAD detached, no tags,
// with the source's tracked and unignored files over it; the stubs interpose by
// PATH alone, a script is answered through its `#!/usr/bin/env` shebang, and not
// one byte of the tree is rewritten.
//
// WHY ONE INVOCATION AT A TIME. Red under RED proves only that the gate is red
// when everything is; red under one KEY proves only that it is red when every
// `go test` is. Every gate runs GREEN first — red under green is UNJUDGEABLE by
// name (FR-11.6) — then each recorded invocation (`go:test#2`) is painted red
// alone and the gate must go red.
//
// WHY THE STUBS ARE A GO PROGRAM. Three rounds slipped shell into the stubs and
// the next adversary's Criticals were bugs in that shell. The stub is
// tools/gateoracle/stub, built once per test process and uninstrumented (under
// the race detector an instrumented stub cost ten times more per exec, and make
// execs it thousands of times); bin/go, bin/sh and the rest are symlinks to it.
// Every decision it makes is a Go function in this package with a unit test.
//
// WHAT IT ENFORCES AGAINST. Drift: a recipe an author could plausibly write
// without meaning to neuter a gate. Not evasion: a recipe written knowing the
// oracle exists. The ceiling every passing run prints names the difference
// (FR-11.5); the red-team brief carries the evasion lens.
package gateoracle
