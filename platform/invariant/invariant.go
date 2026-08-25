// Package invariant provides side-effect-free invariant checks with explicit
// error-return recovery — the ratified Go idiom for P10 Rule 5 (P10-05-INVARIANT-
// DENSITY, ruling D-2). Never panics: the caller's `if err := invariant.Check(...);
// err != nil { return ..., err }` IS the recovery Rule 5 demands.
//
// VENDORABLE (ruling A-1 b): downstream repos copy this file (any import path —
// the density analyzer counts any imported package NAMED `invariant` with these
// signatures); the guard-clause form needs no import at all.
package invariant

import (
	"errors"
	"fmt"
)

// Check returns nil when the invariant holds, or an error naming the violation.
// The condition expression must be side-effect free (the density analyzer screens
// call arguments; see the p10-invariant-density skill for the documented limits).
func Check(cond bool, msg string) error {
	if cond {
		return nil
	}
	// An unnamed violation defeats the recovery contract: the caller's error
	// path is the diagnosis, so the violation must say what broke.
	if msg == "" {
		return errors.New("invariant violated: unnamed violation (Check called with empty msg)")
	}
	return fmt.Errorf("invariant violated: %s", msg)
}
