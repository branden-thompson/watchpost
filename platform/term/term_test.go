package term

import "testing"

// Spec: architecture.md §4/§6/§10 — report mode resolves width ONCE from this
// package (never from render); breakpoints 40/60/80/120 (PD-4: height <12 =
// compact); color gate honors NO_COLOR then TTY-ness; KeyMap is layered data
// with conflict detection at merge AND swap (D-15).

func TestWidthFallsBackTo80(t *testing.T) {
	t.Setenv("COLUMNS", "")
	// In tests stdout is not a TTY and /dev/tty may exist on a dev machine, so
	// exercise the pure resolver on explicit inputs instead of the process state.
	if got := resolveWidth(0, 0, ""); got != 80 {
		t.Fatalf("fallback width = %d, want 80", got)
	}
}

func TestWidthPrefersTTYThenEnv(t *testing.T) {
	if got := resolveWidth(132, 0, "100"); got != 132 {
		t.Fatalf("tty width must win, got %d", got)
	}
	if got := resolveWidth(0, 120, "100"); got != 120 {
		t.Fatalf("stdin width is second, got %d", got)
	}
	if got := resolveWidth(0, 0, "100"); got != 100 {
		t.Fatalf("COLUMNS is third, got %d", got)
	}
	if got := resolveWidth(0, 0, "garbage"); got != 80 {
		t.Fatalf("bad COLUMNS falls through to 80, got %d", got)
	}
}

// D-50 (HUM LEAD, 2026-09-10): four classes, and the lowest one is NOT A
// LAYOUT — it is a refusal to draw.
//
//	< 100    UNSUPPORTED   "resize your terminal to 100 x 25 or larger"
//	100-119  COMPACT       titles may truncate
//	120-150  OPTIMA        the range the reference mock is drawn at
//	> 150    LARGE         a later release: the right rail (D-51)
func TestBreakpoints(t *testing.T) {
	cases := []struct {
		w    int
		want Breakpoint
	}{
		{0, BreakUnsupported}, {40, BreakUnsupported}, {99, BreakUnsupported},
		{100, BreakCompact}, {119, BreakCompact},
		{120, BreakOptima},
		// 150 IS THE TOP OF OPTIMA, NOT THE BOTTOM OF LARGE. The ruling's two
		// ranges overlapped at exactly 150; the reference mock is 150 wide and
		// the ruling calls that Optima's range.
		{150, BreakOptima},
		{151, BreakLarge}, {400, BreakLarge},
	}
	for _, c := range cases {
		if got := BreakpointFor(c.w); got != c.want {
			t.Errorf("BreakpointFor(%d) = %v, want %v", c.w, got, c.want)
		}
	}
}

// A NEGATIVE WIDTH IS NOT A WIDE TERMINAL. Nothing produces one, which is
// exactly why it is worth pinning: the ordered comparisons below rely on
// Unsupported being the bottom, and a signed value that fell through would
// classify a garbage size as drawable.
func TestABadWidthIsUnsupportedRatherThanLarge(t *testing.T) {
	if got := BreakpointFor(-1); got != BreakUnsupported {
		t.Errorf("BreakpointFor(-1) = %v, want BreakUnsupported", got)
	}
}

func TestEveryBreakpointIsNamed(t *testing.T) {
	for _, b := range []Breakpoint{BreakUnsupported, BreakCompact, BreakOptima, BreakLarge} {
		if b.String() == "" {
			t.Errorf("breakpoint %d has no name; the log and the notice both read it", int(b))
		}
	}
	if got := Breakpoint(99).String(); got != "" {
		t.Errorf("a class outside the set names nothing; got %q", got)
	}
}

func TestColorGateNoColorWins(t *testing.T) {
	if colorEnabled(true, "1") {
		t.Fatal("NO_COLOR set must disable color even on a TTY")
	}
	if colorEnabled(false, "") {
		t.Fatal("non-TTY must disable color")
	}
	if !colorEnabled(true, "") {
		t.Fatal("TTY without NO_COLOR must enable color")
	}
}

// --- KeyMap (D-15) ---

func TestKeyMapMergeLayers(t *testing.T) {
	global := KeyMap{"quit": {Keys: []string{"q"}, Help: "quit"}, "help": {Keys: []string{"?"}, Help: "help"}}
	view := KeyMap{"dive-in": {Keys: []string{"enter"}, Help: "dive in"}, "quit": {Keys: []string{"x"}, Help: "quit"}}
	m, _, err := Merge(global, view) // later layers win per Action
	if err != nil {
		t.Fatal(err)
	}
	if m["quit"].Keys[0] != "x" {
		t.Fatal("view layer must override global per Action")
	}
	if m["help"].Keys[0] != "?" {
		t.Fatal("unoverridden global bindings must survive")
	}
}

func TestKeyMapMergeDetectsConflicts(t *testing.T) {
	a := KeyMap{"quit": {Keys: []string{"q"}}, "search": {Keys: []string{"q"}}}
	if _, _, err := Merge(a); err == nil {
		t.Fatal("two Actions on one key in the same scope must be rejected")
	}
}

func TestKeyMapLookup(t *testing.T) {
	m, _, err := Merge(KeyMap{"help": {Keys: []string{"?"}}, "quit": {Keys: []string{"q", "ctrl+c"}}})
	if err != nil {
		t.Fatal(err)
	}
	if act, ok := m.Lookup("ctrl+c"); !ok || act != "quit" {
		t.Fatalf("Lookup(ctrl+c) = %q,%v", act, ok)
	}
	if _, ok := m.Lookup("z"); ok {
		t.Fatal("unbound key must not resolve")
	}
}

func TestHelpReservedForHelp(t *testing.T) {
	// R-3/D-15: '?' is the ONLY locked binding — a merge that maps '?' to any
	// Action other than "help" is rejected.
	if _, _, err := Merge(KeyMap{"search": {Keys: []string{"?"}}}); err == nil {
		t.Fatal("'?' must be reserved for the help Action")
	}
	if _, _, err := Merge(KeyMap{"help": {Keys: []string{"?"}}}); err != nil {
		t.Fatalf("'?' bound to help must be valid: %v", err)
	}
}
