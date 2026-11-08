// Package app is the composition root: the one place that wires domains,
// platform and the live surfaces together. cmd/watchpost stays thin: it
// parses the command line and calls in here, and imports only the one-shot
// report's renderer (modes/report) and the schema generator (pkg/schema)
// itself.
//
// What app builds and runs: the PRIORITY and RECENT pipelines that publish
// the snapshot; the ticker, the producer of breaking alerts; the radio deck,
// the one deck both surfaces read through; the arbiter, the one owner of the
// correspondent's voice, and MasterControl, which gates the deck from the air;
// the Director's schedule, its executors and the pump; the map's builders and
// feeds; and the history recorder. The program's shape is
// docs/architecture.md, and the air-and-deck picture as built is
// 06_docs/02_features/0.16.0-broadcaster-ui/03-architecture-design/as-built-air-and-deck.md.
//
// The files, by what they own. TestEveryFileIsInItsPackagesMap holds every
// non-test file in this package to a row.
//
//	`app.go`, `dashboard.go`, `refs.go`,        the one-shot report, RunDashboard's wiring, location
//	`resolve.go`, `setup.go`, `themes.go`,      refs and the resolver, Setup's save, themes,
//	`shared.go`, `every.go`, `lazymemo.go`,     the shared owners and the one clock-driven loop,
//	`hangup.go`, `hostfacts.go`, `release.go`   shutdown, the host's facts and the release check
//	`pipelines.go`, `stationrecord.go`,         the pipelines and the publisher, the station's
//	`eventrecord.go`, `history.go`, `uvknown.go` feeds, and what history records
//	`fire.go`, `marine.go`, `seismic.go`,       the providers' wiring by family, and where each
//	`endpoints.go`, `credits.go`                provider talks and how it is credited
//	`ticker.go`, `seen_store.go`, `alert_store.go`, the producer: what arrived, what was announced,
//	`burst_words.go`, `compose_takeover.go`,    and the words and Composer for a takeover;
//	`severe.go`, `severe_read.go`, `metro.go`   the severe index and its reads
//	`radio*`, `livesource.go`, `cast.go`,       the radio deck: relays, the watchlist queue, synth,
//	`voices.go`, `saychildren.go`, `bed*`,      the cast and its voices, say's children, the bed
//	`rotation_script.go`, `read_script.go`,     the rotation's script, the Reader, and the main track
//	`mainread.go`, `maintrack.go`
//	`arbiter.go`, `mastercontrol.go`,           the voice's one owner and the air's effector,
//	`air_boundary.go`, `airscope.go`            who may reach the air, and what the rail is scoped to
//	`schedule.go`, `executors.go`, `pump.go`,   the Director's schedule, its executors and the pump,
//	`pool.go`, `inject.go`                      the station's pool, and the test-alert injection seam
//	`map*`, `framelog.go`                       the map's builders, layers, feeds and frame recorder
//	`debug.go`, `dump*`, `stats.go`, `timing.go` diagnostics: the debug server, the dump, the counters
package app
