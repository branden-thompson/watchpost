// Command watchpost is the terminal-native live weather station.
//
// Invocation surface (architecture.md §6, T-L):
//
//	watchpost                 live TTY dashboard (B3)
//	watchpost <location>      location detail (B3)
//	watchpost report <loc>    machine/stdout mode: --json | --report-only | --every (B1a)
//	watchpost setup           the dashboard with the Setup window open (UAT 100)
//	watchpost schema          print JSON schema (B1a)
//
// main stays thin: it builds the cobra tree and delegates to app.Run (Option C).
package main

import (
	"errors"
	"fmt"
	"os"
	_ "time/tzdata" // zone data travels with the binary (red-team 0.9.0 F16): sun times and alert clocks must not depend on the host's zoneinfo

	"github.com/branden-thompson/watchpost/platform/invariant"
)

// version is stamped by the release matrix via -ldflags at SHIP.
var version = "0.0.0-dev"

func main() {
	err := run(os.Args)
	if err == nil {
		return
	}
	// Typed exit codes (§10.2): data printed, code carries the caveat.
	var ec interface{ ExitCode() int }
	if errors.As(err, &ec) {
		code := ec.ExitCode()
		// A typed exit error carrying 0 would silently mask a failure (§10.2).
		if invariant.Check(code >= 1 && code <= 3, "typed exit codes must be 1..3") != nil {
			code = 1
		}
		os.Exit(code)
	}
	// Framework error rule: every failure surfaced to a human must be actionable.
	if invariant.Check(err.Error() != "", "run errors must carry an actionable message") != nil {
		err = fmt.Errorf("watchpost failed with an empty error — please report this bug")
	}
	fmt.Fprintln(os.Stderr, "watchpost:", err)
	os.Exit(1)
}

// run is the testable entry seam wrapping the cobra tree.
func run(args []string) error {
	if err := invariant.Check(version != "", "version must be stamped or default"); err != nil {
		return err
	}
	if err := invariant.Check(len(args) >= 1, "argv must carry the program name"); err != nil {
		return err
	}
	if err := invariant.Check(args[0] != "", "argv[0] must be the program name"); err != nil {
		return err
	}
	root := newRootCmd()
	if err := invariant.Check(root != nil && root.Runnable() || root.HasSubCommands(), "cobra tree must be runnable"); err != nil {
		return err
	}
	root.SetArgs(args[1:])
	return root.Execute()
}
