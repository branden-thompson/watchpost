package main

// The self-test: prove this instrument can FAIL before anyone quotes a number
// from it.
//
// THE STANDING RULE IS "VALIDATE THE INSTRUMENT" and it exists because this
// project has shipped three measurements that could not fail. A completeness
// check is especially prone to it: if declarations were counted as uses, every
// member would look wired, the tool would print "0 NOT" on any tree, and that
// zero is exactly what a reader would take for good news.
//
// So each scenario below is a tree with a KNOWN answer, and the check is that
// the tool reports that answer and no other. Two of them exist purely to fail
// if the walk stops discriminating: `neither` must be reported (the
// declarations-count-as-uses bug) and `wired` must NOT be (the everything-is-
// broken bug).

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// scenario is one synthetic tree and the members that must come back unwired.
type scenario struct {
	name string
	src  string
	want []string // "Set.Member", sorted
}

func selfTestScenarios() []scenario {
	return []scenario{
		{
			name: "a sentinel enum, one member of each kind",
			src: `package probe

type Colour int

const (
	Red Colour = iota
	Green
	Blue
	Grey

	numColours
)

// Red is WIRED: made here, discriminated on below.
func makeRed() Colour { return Red }

func describe(c Colour) string {
	switch c {
	case Red:
		return "red"
	case Green: // Green is READ and never made
		return "green"
	}
	if c == Grey {
		return "grey" // Grey is READ and never made
	}
	return "other"
}

// Blue is MADE and nothing ever looks at it.
func makeBlue() Colour { return Blue }
`,
			// Green and Grey have no writer; Blue has no reader. Red is whole.
			want: []string{"Colour.Blue", "Colour.Green", "Colour.Grey"},
		},
		{
			name: "the sentinel is not a member",
			src: `package probe

type Lane int

const (
	Fast Lane = iota

	numLanes
)

func pick() Lane { return Fast }

func fast(l Lane) bool { return l == Fast }
`,
			// numLanes is the bound, not a member, so nothing is reported.
			want: nil,
		},
		{
			name: "a marker set with a member nothing constructs",
			src: `package probe

type Effect interface{ effect() }

type isEffect struct{}

func (isEffect) effect() {}

type Speak struct{ isEffect }

type Duck struct{ isEffect }

func emit() Effect { return Speak{} }

func perform(e Effect) string {
	switch e.(type) {
	case Speak:
		return "speak"
	case Duck:
		return "duck" // handled, and nothing in production makes one
	}
	return ""
}
`,
			// This is the Duck/Restore shape exactly: an executor and no producer.
			want: []string{"Effect.Duck"},
		},
		{
			name: "a mention inside another declaration is not a use",
			src: `package probe

type Flag int

const (
	Alpha Flag = 1 << iota
	Beta

	numFlags
)

// Beta is MADE in production below, and the only thing that ever mentions it
// again is this constant's expression. Nothing in production logic reads it.
const Both = Alpha | Beta

func makeAlpha() Flag { return Alpha }

func makeBeta() Flag { return Beta }

func isAlpha(f Flag) bool { return f == Alpha }
`,
			// THE SCENARIO THAT MAKES THE DECLARATION SKIP LOAD-BEARING, and it
			// took two attempts to write. The first gave Beta no writer either,
			// so it was reported as unwired whether or not the mention counted —
			// an assertion that held for the wrong reason. Beta now has a WRITER
			// and its only other mention is inside a declaration: count that as
			// a read and Beta looks wired, which is exactly the false pass the
			// guard exists to refuse (D-2 wants the input that makes it fail).
			want: []string{"Flag.Beta"},
		},
		{
			name: "the zero value is written by every zero construction",
			src: `package probe

type State int

const (
	Proposed State = iota
	Admitted

	numStates
)

type Card struct {
	State State
	Name  string
}

// NOTHING NAMES Proposed. A Card built without a State field is one, and no
// walk of the syntax can see that — which is why the zero value is exempt from
// the writer half and only from that half.
func propose(name string) Card { return Card{Name: name} }

func check(c Card) bool {
	switch c.State {
	case Proposed:
		return true
	case Admitted:
		return false
	}
	return false
}

func admit(c Card) Card { c.State = Admitted; return c }
`,
			// Proposed: no writer, but it is the zero value — NOT reported.
			// Admitted: written by admit, read by check — NOT reported.
			want: nil,
		},
		{
			name: "the zero value is still reported when nothing READS it",
			src: `package probe

type Mode int

const (
	Quiet Mode = iota
	Loud

	numModes
)

// Quiet is the zero value AND nothing discriminates on it. The exemption
// covers the writer half only, so this is still a finding.
func loud(m Mode) bool { return m == Loud }

func set() Mode { return Loud }
`,
			want: []string{"Mode.Quiet"},
		},
		{
			name: "a const block with no sentinel is not a closed set",
			src: `package probe

type Loose int

const (
	One Loose = iota
	Two
)

func only() Loose { return One }
`,
			// Two is never made and never read, but the set is not declared
			// closed, so this tool says nothing about it.
			want: nil,
		},
	}
}

// runSelfTest builds each scenario, runs the real scan over it, and compares.
// Returns a process exit code.
func runSelfTest() int {
	fail := 0
	for _, sc := range selfTestScenarios() {
		got, err := scanScenario(sc.src)
		if err != nil {
			fmt.Printf("  FAIL  %s: %v\n", sc.name, err)
			fail++
			continue
		}
		if !sameSet(got, sc.want) {
			fmt.Printf("  FAIL  %s\n        want %v\n         got %v\n", sc.name, sc.want, got)
			fail++
			continue
		}
		fmt.Printf("  ok    %s (%d unwired)\n", sc.name, len(got))
	}
	// THE CONTROL ON THE CONTROLS. A scenario list that ran zero scenarios
	// would print nothing and exit 0, which is the same false pass this whole
	// file exists to refuse.
	if len(selfTestScenarios()) == 0 {
		fmt.Println("wires self-test: no scenarios — the control did not run")
		return 2
	}
	if fail > 0 {
		fmt.Printf("wires self-test: %d scenario(s) FAILED\n", fail)
		return 1
	}
	fmt.Printf("wires self-test: %d scenarios passed; the instrument discriminates\n", len(selfTestScenarios()))
	return 0
}

// scanScenario writes one source file to a temp tree and returns the unwired
// members the real scanner finds in it.
func scanScenario(src string) ([]string, error) {
	dir, err := os.MkdirTemp("", "wires-selftest")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	if err := os.WriteFile(filepath.Join(dir, "probe.go"), []byte(src), 0o644); err != nil {
		return nil, err
	}
	members, err := scan(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, m := range members {
		if m.unwired() {
			out = append(out, m.Set+"."+m.Name)
		}
	}
	sort.Strings(out)
	return out, nil
}

func sameSet(a, b []string) bool {
	return strings.Join(a, ",") == strings.Join(b, ",")
}
