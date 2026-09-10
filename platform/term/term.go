// Package term owns terminal capabilities and the keybinding data model.
//
// Contract (architecture.md §4, §6, §10, PD-4, D-15):
//   - Report/stdout mode resolves width ONCE here (TTY ioctl → stdin → $COLUMNS
//     → 80); TTY mode uses bubbletea WindowSizeMsg and never calls Width().
//   - Color gate: NO_COLOR (any value) wins, then TTY-ness of stdout.
//   - Width breakpoints 100/120/150 (D-50); below 100 the console refuses to
//     draw rather than clamping a frame that does not fit.
//   - KeyMap is layered DATA (view → mode → global). Only "?" = help is locked
//     (R-3). Merge validates conflicts; runtime swaps revalidate (§10.7).
package term

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"sort"
	"strconv"

	"golang.org/x/term"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

// --- width ---

// Width resolves the terminal width once for stdout/report mode.
func Width() int {
	tty, stdin := 0, 0
	if f, err := os.Open("/dev/tty"); err == nil {
		if w, _, err := term.GetSize(int(f.Fd())); err == nil {
			tty = w
		}
		_ = f.Close()
	}
	if w, _, err := term.GetSize(int(os.Stdin.Fd())); err == nil {
		stdin = w
	}
	return resolveWidth(tty, stdin, os.Getenv("COLUMNS"))
}

// resolveWidth is the pure precedence rule: tty > stdin > $COLUMNS > 80.
// Non-positive inputs (failed ioctl leaves 0) fall through to the next source.
func resolveWidth(tty, stdin int, columnsEnv string) int {
	if tty > 0 {
		return tty
	}
	if stdin > 0 {
		return stdin
	}
	if n, err := strconv.Atoi(columnsEnv); err == nil && n > 0 {
		return n
	}
	return 80
}

// --- breakpoints (PD-4, D-50) ---

// Breakpoint names the responsive layout class for a width.
//
// THE LOWEST CLASS IS NOT A LAYOUT. It is a refusal to draw: below 100 columns
// the console says so and renders nothing else, btop-style, because a frame
// clamped into a terminal that cannot hold it is the F-55 defect — 57 cells
// rendered into a 20-cell terminal, silently.
//
// FOUR CLASSES, RULED (D-50, HUM LEAD 2026-09-10). They REPLACE the 40/60/80/120
// scale rather than joining it: `broadcaster.go` was the only production caller
// of this enum, and Observer keeps its own `radioBP` at 84/146, so redefining
// the boundaries here is F-68's "wire the platform one" answer.
type Breakpoint int

const (
	// BreakUnsupported is below the floor: say so, draw nothing else.
	BreakUnsupported Breakpoint = iota
	// BreakCompact is 100-119, where titles may truncate.
	BreakCompact
	// BreakOptima is 120-150, the range the reference mock is drawn at.
	BreakOptima
	// BreakLarge is above 150 and is a LATER RELEASE — the right rail, once
	// there are terminal maps to put in it (D-51). It is classified now so the
	// seam exists; nothing lays out differently for it yet.
	BreakLarge
)

// String names the class, and empty for anything outside the set — a
// Breakpoint crosses package boundaries and a hand-built value must not print
// as a lie.
func (b Breakpoint) String() string {
	switch b {
	case BreakUnsupported:
		return "UNSUPPORTED"
	case BreakCompact:
		return "COMPACT"
	case BreakOptima:
		return "OPTIMA"
	case BreakLarge:
		return "LARGE"
	}
	return ""
}

// BreakpointFor classifies a width.
//
// 150 IS THE TOP OF OPTIMA, NOT THE BOTTOM OF LARGE. The ruling's two ranges
// overlapped at exactly 150; the reference mock is 150 wide and the ruling calls
// that Optima's range, so that is the reading.
func BreakpointFor(w int) Breakpoint {
	switch {
	case w < 100:
		return BreakUnsupported
	case w < 120:
		return BreakCompact
	case w <= 150:
		return BreakOptima
	default:
		return BreakLarge
	}
}

// --- color ---

// colorEnabled is the pure rule: NO_COLOR (any value) wins, then TTY-ness.
func colorEnabled(stdoutIsTTY bool, noColorEnv string) bool {
	if noColorEnv != "" {
		return false
	}
	return stdoutIsTTY
}

// --- keybindings (D-15) ---

// Action names something a view or the app can do ("help", "quit", "dive-in").
type Action string

// HelpAction is the one Action with a locked key: "?" (R-3).
const HelpAction Action = "help"

// Binding is the data for one Action's keys; Help feeds the live help view.
type Binding struct {
	Keys []string
	Help string
}

// KeyMap maps Actions to Bindings for one scope (view, mode, or global).
type KeyMap map[Action]Binding

// Merge folds layers left to right (later layers win per Action), then
// validates: no key may serve two Actions, and "?" may serve only help.
// It is used at registration AND inside runtime swaps (§10.7) — a conflicting
// swap is rejected with an actionable error, never applied.
func Merge(layers ...KeyMap) (KeyMap, []string, error) {
	if err := invariant.Check(len(layers) >= 1, "Merge requires at least one layer"); err != nil {
		return nil, nil, err
	}
	// The FIRST layer is the build's own actions; the rest are the user's
	// [keys] overrides. An override naming an action this build no longer has
	// is DROPPED WITH A NOTE, never an error (FR-14, RS-21).
	//
	// A listener who rebound T for the player size in 0.13.0 must not find
	// 0.14.0 refusing to start because that action retired. Losing a binding
	// is a nuisance; refusing to launch over one is a broken upgrade.
	out := KeyMap{}
	maps.Copy(out, layers[0])
	var dropped []string
	for _, layer := range layers[1:] {
		for act, b := range layer {
			if _, known := out[act]; !known {
				dropped = append(dropped, string(act))
				continue
			}
			out[act] = b
		}
	}
	sort.Strings(dropped)
	seen := map[string]Action{}
	acts := make([]string, 0, len(out))
	for act := range out {
		acts = append(acts, string(act))
	}
	sort.Strings(acts) // deterministic conflict reporting
	for _, a := range acts {
		act := Action(a)
		for _, k := range out[act].Keys {
			if err := invariant.Check(k != "", "bindings must not contain empty keys"); err != nil {
				return nil, nil, err
			}
			if k == "?" && act != HelpAction {
				return nil, nil, fmt.Errorf("key '?' is reserved for help (R-3); %q tried to claim it", act)
			}
			if prev, dup := seen[k]; dup {
				return nil, nil, fmt.Errorf("key %q bound to both %q and %q in the same scope — rebind one in [keys]", k, prev, act)
			}
			seen[k] = act
		}
	}
	return out, dropped, nil
}

// Lookup resolves a pressed key to its Action in this merged map.
func (m KeyMap) Lookup(key string) (Action, bool) {
	if err := invariant.Check(key != "", "cannot look up an empty key"); err != nil {
		return "", false
	}
	for act, b := range m {
		if slices.Contains(b.Keys, key) {
			return act, true
		}
	}
	return "", false
}
