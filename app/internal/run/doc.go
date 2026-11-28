// Package run is what one run of watchpost shares across app's packages: how
// it names itself to providers, where it caches, how it saves a preference,
// the one clock-driven loop, the locked copy and the merged request counts
// (0.19.6 A-20, design AP-5). Each helper here has users in at least two of
// app's packages, the root, the deck and the map data; a helper one package
// alone uses lives with that package.
//
// It holds no state of its own and imports no other package under app, so any
// of them can import it.
//
// The files, by what they own. TestEveryFileIsInItsPackagesMap holds every
// non-test file in this package to a row.
//
//	`every.go`    EveryTick, the one clock-driven loop, and its P10-02 row
//	`shared.go`   the user agent, the cache directories, the preference save, the locked copy, the request counts
package run
