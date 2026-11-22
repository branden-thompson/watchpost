// Package frame is the kit: the parts of a frame both surfaces under
// modes/tty draw the same way, the Observer and the Broadcaster console
// (0.19.6 A-21, design CK-1). What one surface alone draws stays with it; what
// both draw is here once, so the two cannot come to draw it differently.
//
// Its declared inputs (design CK-9, M3) are its parameters and two packages:
// the published snapshot (platform/snapshot), which every masthead, row and
// count is drawn from, and the drawing layer (platform/render), with Bubble
// Tea for the shimmer's command and the cursor. It reads no surface's model,
// which the compiler holds, since both surfaces import it; like every package
// under modes/tty it imports no domain (make lint-imports).
//
// The files, by what they own. TestEveryFileIsInItsPackagesMap holds every
// non-test file in this package to a row.
//
//	`masthead.go`   the masthead: its title, stamp and API summary, its key and its slot
//	`row.go`        the location row: its day cells, trend, loading state and marks
//	`cursor.go`     the cursor: where the focus sits, how it is marked, its slot
//	`tick.go`       the shimmer's clock: its message, its command and its arming rule
//	`memo.go`       the counters and the lock every memo slot carries
//	`text.go`       centring and clipping a line, the test event's mark, a plural
package frame
