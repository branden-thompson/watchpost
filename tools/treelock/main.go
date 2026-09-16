// Command treelock serialises the operations that cannot share a working tree.
//
// THE FAILURE IT CLOSES. `make verify` cleans the build cache and then runs the
// gates; a `go test` started beside it races that clean, and the run reports
// exit 0 over a cache it no longer owns. A mutant sweep is worse: it edits the
// tree, runs a gate, and reverts — so an edit made while it is running is
// reverted with it, and the work is gone with no error anywhere.
//
// `pgrep` IS NOT THE ANSWER, and finding that out is what this exists for.
// Killing the parent of a sweep leaves the `go test` child running under the
// next edit, and a name-matched search for "make verify" does not see it. A
// lock is held by a PROCESS, so a child outliving its parent still holds it.
//
// IT REFUSES RATHER THAN QUEUES. Two gate runs that overlap are not slow, they
// are wrong, and a caller that waited would hide that from whoever started the
// second one.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// lockPath is under dist/ because dist/ is already the build-artefact directory
// and is already ignored — a lock file that showed up in `git status` would be
// noise in every run that takes one.
//
// THE ENVIRONMENT CAN POINT IT ELSEWHERE, and the self-test is the reason. A
// self-test that took the REAL lock could only run on an idle tree, which is
// exactly when nobody needs to ask whether the lock works; pointed at a path of
// its own it runs as a gate INSIDE a locked verify, where it belongs.
func lockPath() string {
	if p := os.Getenv("WATCHPOST_TREELOCK_PATH"); p != "" {
		return p
	}
	return filepath.Join("dist", ".treelock")
}

func main() {
	var (
		name   = flag.String("name", "", "what is being run, shown to whoever is refused")
		status = flag.Bool("status", false, "print the holder, or `free`")
		check  = flag.Bool("check", false, "exit 1 if the tree is held; print who holds it")
		self   = flag.Bool("self-test", false, "prove the lock actually excludes")
	)
	flag.Parse()

	switch {
	case *self:
		os.Exit(selfTest())
	case *status:
		h, held := holder()
		if !held {
			fmt.Println("treelock: free")
			return
		}
		fmt.Println("treelock: HELD by " + h)
	case *check:
		h, held := holder()
		if held {
			fmt.Fprintln(os.Stderr, "treelock: the tree is HELD by "+h)
			fmt.Fprintln(os.Stderr, "  Do not edit files until it is released: an edit made under a")
			fmt.Fprintln(os.Stderr, "  mutant sweep is reverted with the mutant and lost without an error.")
			os.Exit(1)
		}
		fmt.Println("treelock: free")
	default:
		if flag.NArg() == 0 {
			fmt.Fprintln(os.Stderr, "usage: treelock -name <what> -- <command> [args...]")
			os.Exit(2)
		}
		os.Exit(run(*name, flag.Args()))
	}
}

// acquire takes the lock, or reports who holds it.
//
// THE FLOCK IS THE LOCK; THE FILE'S CONTENTS ARE ONLY THE MESSAGE. An advisory
// flock is released by the kernel when the holder exits however it exits —
// killed, panicked, or its terminal closed — so a crashed run cannot wedge the
// tree. A pid written into the file would need a liveness check and would be
// wrong in the window between the check and the answer.
func acquire(name string) (*os.File, string, error) {
	if err := os.MkdirAll(filepath.Dir(lockPath()), 0o755); err != nil {
		return nil, "", err
	}
	f, err := os.OpenFile(lockPath(), os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, "", err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		who := read(f)
		f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, who, nil
		}
		return nil, who, err
	}
	if err := f.Truncate(0); err != nil {
		f.Close()
		return nil, "", err
	}
	if _, err := f.WriteAt([]byte(name+" (pid "+strconv.Itoa(os.Getpid())+", since "+
		time.Now().Format("15:04:05")+")\n"), 0); err != nil {
		f.Close()
		return nil, "", err
	}
	return f, "", nil
}

// holder reports whether anyone holds the lock, without taking it.
//
// IT TAKES AND IMMEDIATELY DROPS a lock rather than reading the file, because
// the file says who wrote it last and not whether they are still there.
func holder() (string, bool) {
	f, err := os.OpenFile(lockPath(), os.O_RDWR, 0o644)
	if err != nil {
		return "", false // no file at all is a tree nobody has locked
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return read(f), true
	}
	_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	return "", false
}

func read(f *os.File) string {
	b := make([]byte, 256)
	n, _ := f.ReadAt(b, 0)
	if s := strings.TrimSpace(string(b[:n])); s != "" {
		return s
	}
	return "an unnamed run"
}

// run holds the lock for the life of the command.
//
// SIGNALS ARE FORWARDED, NOT SWALLOWED. A ctrl-C that killed this wrapper and
// left the gate running would recreate the exact defect the lock is for: a
// child still editing the tree with nothing holding the lock that says so.
func run(name string, argv []string) int {
	if name == "" {
		name = argv[0]
	}
	f, who, err := acquire(name)
	if err != nil {
		fmt.Fprintln(os.Stderr, "treelock: "+err.Error())
		return 2
	}
	if f == nil {
		fmt.Fprintln(os.Stderr, "treelock: REFUSED — the tree is held by "+who)
		fmt.Fprintln(os.Stderr, "  These operations cannot share a working tree: one cleans the build")
		fmt.Fprintln(os.Stderr, "  cache the other is measuring, and a sweep reverts edits made beside it.")
		fmt.Fprintln(os.Stderr, "  Wait for it, or stop it, and run this again.")
		return 1
	}
	defer f.Close()

	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sig)
	if err := cmd.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "treelock: "+err.Error())
		return 2
	}
	go func() {
		for s := range sig { // bounded by the run (P10-02): the channel is stopped on return
			_ = cmd.Process.Signal(s)
		}
	}()
	if err := cmd.Wait(); err != nil {
		var ex *exec.ExitError
		if errors.As(err, &ex) {
			return ex.ExitCode()
		}
		fmt.Fprintln(os.Stderr, "treelock: "+err.Error())
		return 2
	}
	return 0
}
