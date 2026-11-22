// Package ttytest is the test support of the modes/tty tree: what every
// package's tests under modes/tty share and none may copy (0.19.6 D-15, A-29).
// Only test files import it; scripts/lint-imports.sh refuses a non-test
// importer, so nothing here reaches the binary.
//
// It holds the TestMain every package under modes/tty calls, the memo
// guards' engine, and the pure helpers more than one package's tests need.
//
// The files, by what they own. TestEveryFileIsInItsPackagesMap holds every
// non-test file in this package to a row.
//
//	`main.go`      Main, the TestMain every package under modes/tty calls
//	`perturb.go`   PerturbEach, the memo guards' engine, and Field, one field's write
//	`helpers.go`   stripping ANSI, a frame's first differing line, a card, the masthead's
//	               snapshot and check, the loaded pool's size
package ttytest
