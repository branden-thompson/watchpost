package app

// mapproblems.go — what went wrong with the map that the listener cannot
// act on (0.18.0 D-124, UAT-2 U2-29): "We should never show error messages
// to the end user unless we give them a path to resolve it." Such a thing is
// kept here, the last fifty, for the diagnostic dump - never shown.

import (
	"sync"
	"time"
)

// mapProblemsKept is how many problems the diagnostics hold.
const mapProblemsKept = 50

// mapProblems are the map's problems, the newest last.
type mapProblems struct {
	mu   sync.Mutex
	held []string
}

// note keeps a problem, stamped with when it happened.
func (p *mapProblems) note(what string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.held = append(p.held, time.Now().UTC().Format(time.RFC3339)+" "+what)
	if len(p.held) > mapProblemsKept {
		p.held = append([]string(nil), p.held[len(p.held)-mapProblemsKept:]...)
	}
}

// last is a copy of the problems held.
func (p *mapProblems) last() []string { return lockedCopy(&p.mu, &p.held) }
