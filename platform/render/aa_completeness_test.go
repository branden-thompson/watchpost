package render

// aa_completeness_test.go — FR-8: the AA register covers every token, or says
// why it cannot.
//
// THE AA GATE WAS A TAUTOLOGY. withAA LIFTS every pair in aaPairs when a theme
// registers, and TestEveryPaintedPairReadsAAInEveryTheme ITERATES the same
// list — so it can only fail if the lifter fails to converge. A token painted
// somewhere and absent from the register is invisible to it: not failing, not
// passing, not measured. ConfirmBG was exactly that, and it was genuinely
// failing at 4.17:1 in Tokyo Night when someone went looking by hand (F-18).
//
// THE PRODUCER HAS TO BE SOMETHING OTHER THAN THE REGISTER, and the honest one
// available is the TOKEN VOCABULARY: every token this package declares is in
// the register, or is declared here with the reason it cannot be. That is
// weaker than "every pair painted at a call site" — it counts tokens, not
// pairs — and the difference is stated rather than glossed: a token registered
// against one ground and painted on another still passes this, which is the
// hole aaPairs' own `onBoth` list exists to plug by hand.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/platform/closedset"
)

// unmeasurable are the tokens the register cannot carry, and WHY. Every reason
// here is a mechanism that was checked, not a judgement that was made.
var unmeasurable = map[Token]string{
	// SGR 49 is the terminal's OWN default background. There is no colour here
	// to measure: what it resolves to is the user's terminal setting, which the
	// app neither controls nor can read. RadioFG is measured against the window
	// ground, which the app does control.
	RadioBG: "SGR 49, the terminal's default background: no colour to measure",

	// chip() builds ONE value carrying a foreground, a background and a weight,
	// and sgrRaw paints it in a single escape. The register's model is
	// fg x [bg tokens]; a composite is neither, and splitting one to measure it
	// is a different check. F-57.
	KeyChip:       "a composite fg+bg value; the register's fg x bg model cannot express it (F-57)",
	KeyChipMuted:  "a composite fg+bg value (F-57)",
	ChipFlashUp:   "a composite fg+bg value (F-57)",
	ChipFlashDown: "a composite fg+bg value (F-57)",

	// TitleGradient INTERPOLATES between the three stops, so what is painted is
	// never any one of them. The wordmark is also large display text, whose AA
	// threshold is 3:1 rather than the register's 4.5:1 — two reasons the
	// register would measure the wrong thing at the wrong bar. F-57.
	GradStart: "a gradient stop: what is painted is interpolated between the three, at large-text 3:1 (F-57)",
	GradMid:   "a gradient stop (F-57)",
	GradEnd:   "a gradient stop (F-57)",

	// The visualizer's bars are painted on the window ground and ARE
	// expressible — but they carry no information the frame does not also give
	// in words (the play mark, the volume), so the bar is decoration and its
	// threshold would be 3:1. Whether it owes anything is a colour ruling, and
	// colour rulings are the HUM LEAD's. F-57.
	SpectrumLow:  "decorative bars, redundant with the play mark and the volume; threshold is a HUM LEAD ruling (F-57)",
	SpectrumMid:  "decorative bars (F-57)",
	SpectrumHigh: "decorative bars (F-57)",
}

func TestEveryTokenIsMeasuredOrExcused(t *testing.T) {
	registered := map[Token]bool{}
	for _, p := range aaPairs() {
		registered[p.fg] = true
		for _, bg := range p.on {
			registered[bg] = true
		}
	}
	closedset.EachMember(t, "AA register", declaredTokens(t), unmeasurable, func(tk Token) bool {
		return registered[tk]
	})
}

// declaredTokens is the vocabulary, read from the source rather than listed:
// a token added later is a member here without anyone remembering, which is
// the whole point — F-18's defect was a token that shipped "passing AA" having
// never been measured.
func declaredTokens(t *testing.T) []Token {
	t.Helper()
	// THE WHOLE PACKAGE, NOT ONE FILE (red team, 2026-09-08). This parsed
	// "theme.go" alone, so a token declared in any of the package's other
	// 29 files was invisible to the gate — and a token the gate cannot see is a
	// token nothing holds to WCAG AA. Planted: `const PlantedBG Token =
	// "planted.bg"` in themes.go passed silently.
	// EVERY .go FILE IN THE DIRECTORY, parsed one at a time. parser.ParseDir is
	// deprecated (SA1019) precisely because it does not consider build tags when
	// grouping files into packages — and "every file that might declare a token,
	// tags or not" is exactly what this needs, so globbing is both simpler and
	// more correct than the API that was deprecated for getting it wrong.
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	var parsed []*ast.File
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		parsed = append(parsed, f)
	}
	if len(parsed) == 0 {
		t.Fatal("no source files parsed: this gate would pass having measured nothing")
	}
	var out []Token
	inspect := func(n ast.Node) bool {
		vs, ok := n.(*ast.ValueSpec)
		if !ok || vs.Type == nil {
			return true
		}
		if id, ok := vs.Type.(*ast.Ident); !ok || id.Name != "Token" {
			return true
		}
		for i := range vs.Names {
			if i >= len(vs.Values) {
				continue
			}
			lit, ok := vs.Values[i].(*ast.BasicLit)
			if !ok {
				continue
			}
			out = append(out, Token(lit.Value[1:len(lit.Value)-1]))
		}
		return true
	}
	for _, f := range parsed {
		ast.Inspect(f, inspect)
	}
	if len(out) == 0 {
		t.Fatal("no tokens found in the package: this gate would pass having measured nothing")
	}
	return out
}

// TOKENS ARE DECLARED ONE WAY, and this is what makes the parse above
// sufficient rather than merely wide (red team, 2026-09-08).
//
// declaredTokens reads `Name Token = "value"`. A token written as a CONVERSION —
// `Name = Token("value")` — has no ast.Ident type and is invisible to it;
// planted in theme.go's own const block, it passed. Rather than teach the parser
// every spelling, this forbids the other spellings: one shape to parse, and a
// gate that fails the day someone invents a second.
func TestEveryTokenIsDeclaredInTheOneShapeTheGateCanRead(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(b), "\n") {
			if strings.Contains(line, "Token(\"") {
				t.Errorf("%s:%d declares a token by CONVERSION, which declaredTokens cannot see;\n"+
					"  write `Name Token = \"value\"` instead:\n  %s", f, i+1, strings.TrimSpace(line))
			}
			if strings.Contains(line, "Token = \"") {
				found++
			}
		}
	}
	if found == 0 {
		t.Fatal("no token declarations found at all: this guard is looking at the wrong place")
	}
	t.Logf("%d token declarations, all in the readable shape", found)
}
