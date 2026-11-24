package console

// broadcaster_cursor.go — where the console's cursor stands, and the slot its
// memo holds it in. The placing rule and the slot are the kit's (cursor.go in
// internal/frame); the Observer's half is the root's cursor.go (0.18.0 F-205
// AX-3, 0.19.0 D-362).

import "github.com/branden-thompson/watchpost/modes/tty/internal/frame"

// Cursor is where the console's cursor stands: on its focused row's
// pointer, the line-up's or the pool's.
func (b Broadcaster) Cursor(content string) (x, y int, ok bool) {
	r, c, found := frame.SurfaceMark(content, b.opts().Glyphs().Pointer)
	return c, r, found
}

// HeldCursor is the console memo's cursor slot, nil without a memo.
func (b Broadcaster) HeldCursor() *frame.CursorSlot {
	if b.memo == nil {
		return nil
	}
	return &b.memo.cursor
}
