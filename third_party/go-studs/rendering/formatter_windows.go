//go:build windows
// +build windows

package rendering

import (
	"os"
	"strconv"
	"syscall"
	"unsafe"
)

var (
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleMode = kernel32.NewProc("GetConsoleScreenBufferInfo")
)

type coord struct {
	X int16
	Y int16
}

type smallRect struct {
	Left   int16
	Top    int16
	Right  int16
	Bottom int16
}

type consoleScreenBufferInfo struct {
	Size              coord
	CursorPosition    coord
	Attributes        uint16
	Window            smallRect
	MaximumWindowSize coord
}

// GetTerminalSize returns the current terminal width and height.
// This is the Windows implementation using Windows Console API.
func GetTerminalSize() (width, height int) {
	// Try to get console screen buffer info
	handle := syscall.Handle(os.Stdout.Fd())

	var info consoleScreenBufferInfo
	ret, _, _ := procGetConsoleMode.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&info)),
	)

	if ret != 0 {
		width = int(info.Window.Right - info.Window.Left + 1)
		height = int(info.Window.Bottom - info.Window.Top + 1)

		if width > 0 && height > 0 {
			return width, height
		}
	}

	// Check environment variables as fallback
	width, height = 80, 24 // defaults

	if cols := os.Getenv("COLUMNS"); cols != "" {
		if parsedWidth, err := strconv.Atoi(cols); err == nil && parsedWidth > 0 {
			width = parsedWidth
		}
	}
	if rows := os.Getenv("LINES"); rows != "" {
		if parsedHeight, err := strconv.Atoi(rows); err == nil && parsedHeight > 0 {
			height = parsedHeight
		}
	}

	return width, height
}
