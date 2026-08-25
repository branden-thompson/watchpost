// Package term owns terminal capabilities and the keybinding data model.
//
// Contract (architecture.md §4, §6, §10, PD-4, D-15):
//   - Report/stdout mode resolves width ONCE here (TTY ioctl → stdin → $COLUMNS
//     → 80); TTY mode uses bubbletea WindowSizeMsg and never calls Width().
//   - Color gate: NO_COLOR (any value) wins, then TTY-ness of stdout.
//   - Width breakpoints 40/60/80/120; height <12 rows renders compact.
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

// --- breakpoints (PD-4) ---

// Breakpoint names the responsive layout class for a width.
type Breakpoint int

const (
	BreakTooNarrow Breakpoint = iota // <40: print "terminal too narrow"
	BreakMini                        // 40–59: mini/player layouts
	BreakSingle                      // 60–79: single column
	BreakStandard                    // 80–119: standard two-column
	BreakWide                        // >=120: three panels + charts
)

// BreakpointFor classifies a width.
func BreakpointFor(w int) Breakpoint {
	switch {
	case w < 40:
		return BreakTooNarrow
	case w < 60:
		return BreakMini
	case w < 80:
		return BreakSingle
	case w < 120:
		return BreakStandard
	default:
		return BreakWide
	}
}

// HeightCompact reports whether a terminal height renders the compact layout.
func HeightCompact(rows int) bool { return rows < 12 }

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
func Merge(layers ...KeyMap) (KeyMap, error) {
	if err := invariant.Check(len(layers) >= 1, "Merge requires at least one layer"); err != nil {
		return nil, err
	}
	out := KeyMap{}
	for _, layer := range layers {
		maps.Copy(out, layer)
	}
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
				return nil, err
			}
			if k == "?" && act != HelpAction {
				return nil, fmt.Errorf("key '?' is reserved for help (R-3); %q tried to claim it", act)
			}
			if prev, dup := seen[k]; dup {
				return nil, fmt.Errorf("key %q bound to both %q and %q in the same scope — rebind one in [keys]", k, prev, act)
			}
			seen[k] = act
		}
	}
	return out, nil
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
