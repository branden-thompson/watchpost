package rendering

import (
	"os"
	"sync/atomic"

	"golang.org/x/term"
)

// R10 (app-status-multi-port-support T16): a single process-wide answer to
// "may we emit ANSI escapes?". Honors the NO_COLOR convention
// (https://no-color.org — any non-empty value disables color) and requires
// stdout to be a terminal, so piped/CI output receives plain text.
//
// Resettable tri-state, lock-free on the read path (BUILD red-team #13):
// 0 = uninitialized, 1 = enabled, 2 = disabled. Detection runs at most a
// handful of times under racing first-calls (idempotent, same result);
// every subsequent emission is a single atomic load.
var colorState atomic.Int32

func colorEnabled() bool {
	switch colorState.Load() {
	case 1:
		return true
	case 2:
		return false
	}
	on := os.Getenv("NO_COLOR") == "" && term.IsTerminal(int(os.Stdout.Fd()))
	if on {
		colorState.CompareAndSwap(0, 1)
	} else {
		colorState.CompareAndSwap(0, 2)
	}
	return colorState.Load() == 1
}

// SetColorEnabledForTest overrides color emission (test use).
func SetColorEnabledForTest(on bool) {
	if on {
		colorState.Store(1)
	} else {
		colorState.Store(2)
	}
}

// ResetColorEnabledForTest clears the override so the next call re-detects.
func ResetColorEnabledForTest() {
	colorState.Store(0)
}

// ColorsEnabled reports whether ANSI emission is currently allowed (the R10
// gate). Exposed for emitters whose escapes WrapSGR cannot express —
// truecolor gradients (SGR classifies each ";"-split element, which would
// mangle 38;2;r;g;b params). Prefer WrapSGR/ApplyColor wherever possible.
func ColorsEnabled() bool {
	return colorEnabled()
}

// WrapSGR wraps text in an SGR escape built from codes plus the reset —
// the gated form of the `SGR(codes...) + text + reset` idiom. With color
// disabled it returns the text untouched (no escapes, no reset — gating
// inside SGR alone would leak trailing resets at call sites).
func WrapSGR(text string, codes ...string) string {
	if len(codes) == 0 || !colorEnabled() {
		return text
	}
	return SGR(codes...) + text + "\033[0m"
}
