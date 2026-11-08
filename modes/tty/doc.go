// Package tty is the live terminal program: the Router and the two surfaces
// it holds, the Observer and the Broadcaster console.
//
// The Router (router.go) is the Bubble Tea model the program runs. It owns
// the keys and messages that belong to the program, swaps between the two
// surfaces, and mirrors what both must agree on. The Observer is the
// listener's dashboard, the Dashboard type; the Broadcaster is the station
// operator's console. Both read the published snapshot (platform/snapshot) and
// the hooks app hands in through Config. No file here imports a domain
// (make lint-imports), and only platform/render draws with go-studs.
//
// Where a particular thing happens, event by event, is
// docs/where-things-happen.md; the program's shape is docs/architecture.md.
//
// The files, by what they own. TestEveryFileIsInItsPackagesMap holds every
// non-test file in this package to a row.
//
//	`router.go`                                the Router: the program's model, the swap, the shared keys
//	`dashboard*`, `nav.go`, `window_keys.go`   the Observer's model, its Config and messages, its keys
//	`masthead.go`, `body.go`, `layout.go`,     the Observer's frame: masthead, tables, layout, view,
//	`view.go`, `columns.go`, `memo.go`,        the frame's memos, the cursor, Reduce motion,
//	`cursor.go`, `motion.go`, `timing.go`      and the timing instrument
//	`ticker.go`, `alerts.go`                   the ticker band and the alert module
//	`radio_panel.go`, `relayfault.go`          the radio player panel and the relay-fault window
//	`detail*`, `modal_location.go`,            Location Details, the lookup and its answer,
//	`locate.go`, `textfield.go`, `poolnote.go` the text field, and the station-pool note
//	`severe.go`                                the severe events window
//	`status*`, `help_about.go`,                Status, Help, About and its credits,
//	`credit_row.go`, `debug.go`                and the ctrl+d diagnostics window
//	`setup*`                                   Settings: the row table, the tabs, the layout, each group
//	`map_*`                                    the map pane, its modes, layers, Settings rows, and the
//	                                           Propagation mode (`map_prop_*`)
//	`broadcaster*`, `request.go`               the Broadcaster console and its Line-Up Request window
package tty
