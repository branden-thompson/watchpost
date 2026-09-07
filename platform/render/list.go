package render

// list.go — how a LIST shows which row has the focus (HUM LEAD, UAT
// 2026-08-30). One owner, so every list-shaped surface marks focus the same
// way: the Settings window's rows, and whatever comes
// next.
//
// # Why this is not the table's focus, and should not become it
//
// The dashboard table marks focus with THREE things: a bold white pointer
// (FocusPointer), the name cell in bold yellow (FocusName), and every other
// grey cell shifted to light blue (FocusCell). A list uses TWO: a bold yellow
// pointer and the label in the same yellow, not bold.
//
// That difference is earned, not accidental:
//
//   - A TABLE ROW IS MANY CELLS. Its focus has two jobs — separate the row from
//     its neighbours (FocusCell does that across the whole row) and separate
//     the identity within the row (FocusName, bold, so the name stands out
//     from the numbers beside it). Both jobs exist because there is competing
//     content on the same line.
//   - A LIST ROW IS ONE LABEL. There is nothing on the line to separate it
//     from, so the second job does not exist. Bolding the label as well as the
//     pointer would be emphasis with nothing to out-rank — every focused row
//     would shout, which is how a list stops reading as a list.
//
// Consolidating them would mean forcing one of those wrong: either the table
// loses the bold name it has carried since UAT 50, or every list row shouts.
// So they share a VOCABULARY (a pointer token and a focus-label token) and
// differ in the values, and this comment exists so the inconsistency is not
// "tidied up" later by someone who sees only the two token names.

// ListMark is the pointer column for one row of a list: the focused row's
// pointer in bold yellow, two spaces otherwise.
//
// It returns a fixed two cells wide, so a list's rows align whether or not
// anything is focused, and it takes its glyph from the set so --ascii needs no
// special case.
func (o Opts) ListMark(focused bool) string {
	if !focused {
		return "  "
	}
	return Tint(o.Glyphs().Pointer, Tok(ListPointer)) + " "
}

// ListLabel is a row's own words: bright normally, the focus yellow when the
// row has it — so the pointer and the thing it points at read as one.
func ListLabel(text string, focused bool) string {
	if focused {
		return Tint(text, Tok(ListFocus))
	}
	return Tint(text, Tok(TextBright))
}
