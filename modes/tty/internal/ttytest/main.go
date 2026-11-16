package ttytest

import (
	"os"
	"testing"
	"time"
)

// Main pins the zone to UTC once, before any goroutine exists, then runs the
// package's tests and exits with their code. Every package under modes/tty
// calls it from its TestMain, so a package a split creates keeps the ground
// modes/tty's tests stand on (0.19.6 TG-3): a goldens' stamp reads the same on
// every machine, and no test writes time.Local while a timer left by an
// earlier one reads it (W14).
func Main(m *testing.M) {
	time.Local = time.UTC
	os.Exit(m.Run())
}
