// Package apptest is the test support of the app tree: what every package's
// tests under app share and none may copy (0.19.6 A-24, A-29). Only test files
// import it; scripts/lint-imports.sh refuses a non-test importer, so nothing
// here reaches the binary.
//
// It holds the TestMain ground every package under app stands on. The pure
// test helpers arrive with the packages that need them (0.19.6 B5).
//
// The files, by what they own. TestEveryFileIsInItsPackagesMap holds every
// non-test file in this package to a row.
//
//	`main.go`   Main, the TestMain ground every package under app calls
package apptest
