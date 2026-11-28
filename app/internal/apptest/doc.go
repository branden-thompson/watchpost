// Package apptest is the test support of the app tree: what every package's
// tests under app share and none may copy (0.19.6 A-24, A-29). Only test files
// import it; scripts/lint-imports.sh refuses a non-test importer, so nothing
// here reaches the binary.
//
// It holds the TestMain ground every package under app stands on, and the
// pure helpers their tests share (0.19.6 design AP-9).
//
// The files, by what they own. TestEveryFileIsInItsPackagesMap holds every
// non-test file in this package to a row.
//
//	`main.go`                  Main, the TestMain ground every package under app calls
//	`helpers.go`               the config file, the poll, the places, the offline client, the NWS fixtures
//	`race.go`, `norace.go`     RaceSlowdown and RaceEnabled, with the race detector and without
package apptest
