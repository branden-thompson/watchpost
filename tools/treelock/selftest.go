package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// selfTest proves the lock EXCLUDES, in both directions.
//
// A LOCK THAT NEVER LOCKS PASSES EVERY OPTIMISTIC TEST. Delete the Flock call
// and `acquire` still returns a file, `run` still runs the command, and every
// gate still goes green — while two sweeps edit the tree at once. So the
// discriminating case is a SECOND PROCESS being refused while this one holds,
// and it is checked by actually starting one.
//
// AND THE OTHER DIRECTION MATTERS AS MUCH. A lock that never releases wedges
// the tree until someone deletes a file they have never heard of, so the run
// after the release must be admitted.
func selfTest() int {
	fail := func(f string, a ...any) int {
		fmt.Fprintf(os.Stderr, "treelock self-test: FAILED — "+f+"\n", a...)
		return 1
	}
	// ITS OWN LOCK, NOT THE TREE'S. This runs as a gate inside a `verify` that is
	// itself holding the real lock, so testing the real one would test whether
	// verify is running. A temporary path tests the MECHANISM, which is the thing
	// that can break.
	dir, err := os.MkdirTemp("", "treelock-selftest")
	if err != nil {
		return fail("could not make a scratch lock: %v", err)
	}
	defer func() { _ = os.RemoveAll(dir) }() // scratch, and the OS reclaims it either way
	if err := os.Setenv("WATCHPOST_TREELOCK_PATH", filepath.Join(dir, "lock")); err != nil {
		return fail("could not point the self-test at its own lock: %v", err)
	}

	if who, held := holder(); held {
		return fail("a lock nothing has taken reports itself held by %s", who)
	}

	f, who, err := acquire("self-test")
	if err != nil {
		return fail("could not take the lock: %v", err)
	}
	if f == nil {
		return fail("a free tree refused the lock (held by %s)", who)
	}
	if _, held := holder(); !held {
		return fail("the lock is taken and `holder` reports the tree free")
	}
	if code := childCheck(); code != 1 {
		_ = f.Close()
		return fail("a second process was ADMITTED while the lock was held (-check exited %d)", code)
	}

	// THE RELEASE IS PART OF WHAT IS BEING TESTED, so its error is checked where
	// the two above are not: a close that failed leaves the lock held, and every
	// assertion after this one would be measuring that instead of the mechanism.
	if err := f.Close(); err != nil {
		return fail("the lock could not be released: %v", err)
	}
	if who, held := holder(); held {
		return fail("the lock was released and `holder` still reports %s", who)
	}
	if code := childCheck(); code != 0 {
		return fail("a released tree still refuses a second process (-check exited %d)", code)
	}

	fmt.Println("treelock self-test: passed; the lock excludes a second process and releases")
	fmt.Println("  Both directions: a held tree refuses, a released tree admits. A lock that")
	fmt.Println("  never locks passes every optimistic test, so the refusal is the assertion.")
	return 0
}

// childCheck runs `-check` in a SEPARATE PROCESS and returns its exit code.
//
// IT HAS TO BE A REAL PROCESS. An advisory flock is held per open file
// description, so a second attempt inside this one would be a different test —
// and the thing being defended against is another shell, not another function.
//
// IT INHERITS THE ENVIRONMENT, which is how the child is pointed at the same
// scratch lock the parent took rather than at the tree's.
func childCheck() int {
	self, err := os.Executable()
	if err != nil {
		return -1
	}
	cmd := exec.Command(self, "-check")
	cmd.Stdout, cmd.Stderr = nil, nil
	if err := cmd.Run(); err != nil {
		var ex *exec.ExitError
		if errors.As(err, &ex) {
			return ex.ExitCode()
		}
		return -1
	}
	return 0
}
