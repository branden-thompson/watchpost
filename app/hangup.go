package app

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/branden-thompson/watchpost/domains/radio/synth"
)

// endOnHangUp ends the say children when the terminal hangs up, then lets
// the hang-up end watchpost as it would have (FR-9.1): the TUI quits cleanly
// on an interrupt or a terminate, and its exit ends them; a hang-up runs no
// exit. The function returned takes the handler off.
func endOnHangUp() func() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGHUP)
	done := make(chan struct{})
	go func() {
		select {
		case <-ch:
			synth.EndChildren()
			signal.Reset(syscall.SIGHUP)
			if self, err := os.FindProcess(os.Getpid()); err == nil {
				_ = self.Signal(syscall.SIGHUP) // where there is no hang-up signal to send, nothing is
			}
		case <-done:
		}
	}()
	return func() {
		signal.Stop(ch)
		close(done)
	}
}
