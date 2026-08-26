//go:build windows

package app

// Windows has no SIGUSR1; the diagnostic dump is reachable through the
// env hook only: launch with WATCHPOST_DEBUG_PPROF=1 and GET
// http://127.0.0.1:6060/debug/dump (quality pass Q0, C1).

import "context"

// startDumpTrigger is a no-op here; /debug/dump is the trigger.
func startDumpTrigger(context.Context, *dumper) {}

// dumpHint is the [S] modal's one-line instruction.
func dumpHint(_ int, dir string) string {
	return "WATCHPOST_DEBUG_PPROF=1 then GET http://" + debugAddr() + "/debug/dump → " + dir
}
