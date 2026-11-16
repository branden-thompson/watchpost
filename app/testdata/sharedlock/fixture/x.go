package fixture

import "sync"

// livePipelines is the planted owner of the shared lock.
type livePipelines struct {
	mu sync.Mutex
	f  func()
}

func (lp *livePipelines) locks() { lp.mu.Lock(); defer lp.mu.Unlock() }

// again takes the lock, then calls a method that takes it again.
func (lp *livePipelines) again() {
	lp.mu.Lock()
	lp.locks()
	lp.mu.Unlock()
}

// across calls, under the lock, a method planned for another package.
func (lp *livePipelines) across() {
	lp.mu.Lock()
	lp.far()
	lp.mu.Unlock()
}

func (lp *livePipelines) far() {}

// value calls, under the lock, through a function value nobody recorded.
func (lp *livePipelines) value() {
	lp.mu.Lock()
	lp.f()
	lp.mu.Unlock()
}

// peekLocked is named for a caller that holds the lock, and takes it.
func (lp *livePipelines) peekLocked() { lp.locks() }
