// Package allocpin measures an allocation pin and reports it in the one line
// the pin comparator reads (0.19.6 B0.7, A-10). Every pin `make alloc-budget`
// selects reports through it, so `tools/pins` can hold each measurement to its
// baseline, the maximum over 10 runs on code identical to `e57afeaf`.
//
// Only test files import it; scripts/lint-imports.sh refuses a non-test
// importer, so nothing here reaches the binary. It sits at the module's root
// because the pins are in every tree: `modes/tty`, `app`, `platform` and
// `domains`.
//
// The files, by what they own:
//
//	`allocpin.go`   Cost, Pin, MeasureEach, PinBenchmark, Report and ReportCount
package allocpin
