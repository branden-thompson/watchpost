// Package console is the Broadcaster console: the station operator's surface,
// the second of the two the Router in modes/tty holds (0.16.0 P1, FR-2;
// 0.19.6 design CK-2).
//
// Its inputs are what the Router hands it (design CK-2, CK-9): New's Opening
// at start, then the messages Scoped names, the Mirror of what the Observer
// owns and the console only draws, the listener's mute, and the snapshots
// through WithSnapshot and WithRecent. It reads no Observer state and holds no
// Router field, which the compiler holds, since modes/tty imports it and not
// the other way. What both surfaces draw the same way is the kit's
// (internal/frame); like every package under modes/tty it imports no domain
// (make lint-imports).
//
// The files, by what they own. TestEveryFileIsInItsPackagesMap holds every
// non-test file in this package to a row.
//
//	`broadcaster.go`            the console: its model, Opening and Mirror, its messages, Update and View
//	`broadcaster_air.go`        the air box: what is on the air, and the bed under it
//	`broadcaster_upnext.go`     UP NEXT and the takeover beside it
//	`broadcaster_slots.go`      the ten slots, drawn always, and the card in each
//	`broadcaster_manifest.go`   a card as a manifest: what it is doing, how fresh, what is in it
//	`broadcaster_detail.go`     a card's own window, the drill-down the manifest defers to
//	`broadcaster_lineup.go`     the running order as the table's rows
//	`broadcaster_pool.go`       the station's location pool, with its weather
//	`broadcaster_rail.go`       the left rail, the frame's chrome, and the widths the layout measures
//	`broadcaster_header.go`     the console's masthead
//	`broadcaster_memo.go`       the tables' memo and its key
//	`broadcaster_cursor.go`     where the console's cursor stands, and its slot
package console
