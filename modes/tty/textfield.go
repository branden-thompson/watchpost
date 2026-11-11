package tty

// textfield.go — what a window's text field does with backspace.

import "unicode/utf8"

// eraseRune is backspace on a text field: the last letter goes whole, never
// one byte of it, and an empty field stays empty (0.19.5 D-5). Every window
// that takes typed text erases through it.
func eraseRune(s string) string {
	_, size := utf8.DecodeLastRuneInString(s)
	return s[:len(s)-size]
}
