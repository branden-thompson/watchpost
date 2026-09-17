package gateoracle

import "fmt"

// Reporter is the slice of testing.T the assertions use, so a specimen table can
// hand them a Recorder and read the verdict instead of failing the parent.
type Reporter interface {
	Helper()
	Logf(format string, args ...any)
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
}

// Recorder captures what an assertion would have reported. Fatalf aborts the
// assertion the way testing.T's does, and VerdictOf absorbs the abort.
type Recorder struct{ Errs []string }

type fatal struct{}

func (r *Recorder) Helper()                   {}
func (r *Recorder) Logf(string, ...any)       {}
func (r *Recorder) Errorf(f string, a ...any) { r.Errs = append(r.Errs, fmt.Sprintf(f, a...)) }
func (r *Recorder) Fatalf(f string, a ...any) {
	r.Errs = append(r.Errs, fmt.Sprintf(f, a...))
	panic(fatal{})
}

// VerdictOf runs an assertion against a Recorder and says whether it fired.
func VerdictOf(fn func(Reporter)) (fired bool, said []string) {
	r := &Recorder{}
	func() {
		defer func() {
			if x := recover(); x != nil {
				if _, ok := x.(fatal); !ok {
					panic(x)
				}
			}
		}()
		fn(r)
	}()
	return len(r.Errs) > 0, r.Errs
}
