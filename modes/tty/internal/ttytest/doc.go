// Package ttytest is the test support of the modes/tty tree: what every
// package's tests under modes/tty share and none may copy (0.19.6 D-15, A-29).
// Only test files import it; scripts/lint-imports.sh refuses a non-test
// importer, so nothing here reaches the binary.
//
// It holds the TestMain every package under modes/tty calls. The memo guard's
// engine and the pure helpers arrive with the packages that need them (0.19.6
// B2).
//
// The files, by what they own. TestEveryFileIsInItsPackagesMap holds every
// non-test file in this package to a row.
//
//	`main.go`   Main, the TestMain every package under modes/tty calls
package ttytest
