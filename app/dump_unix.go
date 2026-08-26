//go:build !windows

package app

// SIGUSR1 → diagnostic dump (quality pass Q0, C1). Unix only: Windows has
// no user signals, so it keeps the env hook (/debug/dump on the opt-in
// loopback server — see dump_windows.go). Trust boundary: a signal comes
// from the same UID; the dump holds no secrets (profiles carry stacks and
// counts, never string bytes — red-team InfoSec verified).

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// maxDumpsPerRun bounds the trigger loop (P10-02): one dump a minute for
// a week is 10,080 — a process that has been signalled that often has a
// louder problem than a missing profile.
const maxDumpsPerRun = 10_000

// startDumpTrigger listens for SIGUSR1 until ctx ends.
func startDumpTrigger(ctx context.Context, d *dumper) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGUSR1)
	go func() {
		defer signal.Stop(ch)
		for i := 0; i < maxDumpsPerRun; i++ {
			select {
			case <-ctx.Done():
				return
			case <-ch:
				_, _ = d.Dump(time.Now()) // the outcome lands in the [S] modal; nothing to do with an error here
			}
		}
	}()
}

// dumpHint is the [S] modal's one-line instruction.
func dumpHint(pid int, dir string) string {
	return fmt.Sprintf("kill -USR1 %d → %s", pid, dir)
}
