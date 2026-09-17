package gateoracle

import (
	"fmt"
	"runtime"
)

// Reporter is the slice of testing.T the assertions use, so a specimen table can
// hand them a Recorder and read the verdict instead of failing the parent.
type Reporter interface {
	Helper()
	Logf(format string, args ...any)
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
}

// Recorder captures what an assertion would have reported. Fatalf ends the
// assertion the way testing.T's does — the goroutine exits — and VerdictOf runs
// the assertion on a goroutine of its own for that reason, reading the record
// only after that goroutine is done.
type Recorder struct{ errs []string }

func (r *Recorder) Helper()                   {}
func (r *Recorder) Logf(string, ...any)       {}
func (r *Recorder) Errorf(f string, a ...any) { r.errs = append(r.errs, fmt.Sprintf(f, a...)) }
func (r *Recorder) Fatalf(f string, a ...any) {
	r.Errorf(f, a...)
	runtime.Goexit()
}

// VerdictOf runs an assertion against a Recorder and says whether it fired.
func VerdictOf(fn func(Reporter)) (fired bool, said []string) {
	r := &Recorder{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn(r)
	}()
	<-done
	return len(r.errs) > 0, r.errs
}
