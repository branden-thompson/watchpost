package rendering

import (
	"os"
	"strconv"

	"golang.org/x/term"
)

// GetTerminalSize returns the current terminal dimensions with robust
// fallback logic — one implementation for every platform on
// golang.org/x/term (no ioctl, no unsafe, no build tags).
//
// Detection order:
// 1. /dev/tty (works even when stdout is piped or redirected)
// 2. stdin, then stdout
// 3. the COLUMNS and LINES environment variables
// 4. 80x24
func GetTerminalSize() (width, height int) {
	if tty, err := os.Open("/dev/tty"); err == nil {
		w, h, err := term.GetSize(int(tty.Fd()))
		_ = tty.Close()
		if err == nil && w > 0 && h > 0 {
			return w, h
		}
	}
	for _, f := range []*os.File{os.Stdin, os.Stdout} {
		if w, h, err := term.GetSize(int(f.Fd())); err == nil && w > 0 && h > 0 {
			return w, h
		}
	}
	width, height = 80, 24
	if cols, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil && cols > 0 {
		width = cols
	}
	if rows, err := strconv.Atoi(os.Getenv("LINES")); err == nil && rows > 0 {
		height = rows
	}
	return width, height
}
