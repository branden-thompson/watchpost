// Package tz memoizes time-zone loading (B3 UAT 74). time.LoadLocation
// opens and parses a zoneinfo file on every call; called for every
// location on every published snapshot — under the assembler's lock — it
// blocked a syscall per location and spawned an OS thread each time (the
// 140-thread launch). Zones never change during a run: load each name once.
package tz

import (
	"sync"
	"time"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

var cache sync.Map // name -> result (process-wide memo; P10-06 exemption recorded)

type result struct {
	loc *time.Location
	err error
}

// Location is a drop-in for time.LoadLocation, memoized by name (errors
// too, so a bad name is not re-parsed every cycle).
func Location(name string) (*time.Location, error) {
	if err := invariant.Check(name != "", "tz: empty zone name — callers decide their own fallback"); err != nil {
		return nil, err
	}
	if v, ok := cache.Load(name); ok {
		r := v.(result)
		return r.loc, r.err
	}
	loc, err := time.LoadLocation(name)
	cache.Store(name, result{loc: loc, err: err})
	return loc, err
}
