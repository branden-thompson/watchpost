package gateoracle

// ceiling is what every passing run says it did NOT judge (FR-11.5): the drift
// it enforces against is bounded; evasion is named, not closed.
func ceiling(t Reporter, did string) {
	t.Helper()
	t.Logf("%s. The oracle enforces against DRIFT and not EVASION. NOT judged: a script sourced or inlined "+
		"rather than executed, a hop through another makefile (-f, -C), a compensating invocation that makes "+
		"counts agree, PATH re-exported for a target, a predicate on the CI environment or on an untracked "+
		"file, a toolchain reached by absolute path, a recipe under `env -i` or with ORACLE_* reassigned, a "+
		"discard inside a script, a binary `go build` writes without `-o`, a third-party `go run` tool's "+
		"control, which of two PARALLEL invocations takes #1, and any command not in %v. Git metadata is judged "+
		"as CI has it: HEAD detached at the commit, no tags.", did, tools)
}
