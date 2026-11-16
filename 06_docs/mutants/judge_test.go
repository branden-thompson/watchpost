//go:build mutants

package mutants

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// judgeModule is a throwaway module the judge can be pointed at: one package,
// one function with a rule in it, and a test that compiles against it.
func judgeModule(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"go.mod":         "module example.invalid/judged\n\ngo 1.26\n",
		"p/p.go":         "package p\n\n// Allowed is the rule a mutant deletes.\nfunc Allowed(n int) bool {\n\treturn n > 0\n}\n",
		"p/p_test.go":    "package p\n\nimport \"testing\"\n\nfunc TestAllowed(t *testing.T) {\n\tif !Allowed(1) {\n\t\tt.Fatal(\"1 is allowed\")\n\t}\n}\n",
		"06_docs/README": "the corpus lives below\n",
	}
	for rel, body := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// plantMutant writes a mutant script into the module's corpus.
func plantMutant(t *testing.T, root, name, body string) string {
	t.Helper()
	p := filepath.Join(root, "06_docs", "mutants", name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// replaceIn is a mutant's body: replace old with new in the target, once.
func replaceIn(target, old, new string) string {
	return "import pathlib\np = pathlib.Path('" + target + "'); s = p.read_text()\n" +
		"old = " + pyQuote(old) + "\nassert s.count(old) == 1, 'planted'\n" +
		"p.write_text(s.replace(old, " + pyQuote(new) + ", 1))\n"
}

func pyQuote(s string) string { return "'" + strings.ReplaceAll(s, "\n", "\\n") + "'" }

// TestTheJudgeGivesEachVerdict is the judge held to its five verdicts on a
// throwaway module (0.19.6 D-37): a mutant that compiles applies; one whose
// text is gone is stale; one that changes nothing is inert; one that breaks
// the build, or only its tests, is invalid; and one that writes into the tree
// it judges, or names a target outside it, escaped. The real tree's file is
// unchanged after every case but the planted write.
func TestTheJudgeGivesEachVerdict(t *testing.T) {
	root := judgeModule(t)
	real := filepath.Join(root, "p", "p.go")
	escapeBody := "import pathlib\np = pathlib.Path('p/p.go'); s = p.read_text()\n" +
		"open(" + pyQuote(real) + ", 'a').write('// escaped\\n')\n" +
		"p.write_text(s.replace('n > 0', 'n >= 0', 1))\n"
	absoluteBody := "import pathlib\np = pathlib.Path(" + pyQuote(real) + "); s = p.read_text()\n" +
		"p.write_text(s.replace('n > 0', 'n >= 0', 1))\n"
	for _, c := range []struct {
		name, body string
		want       verdict
	}{
		{"m1_applies.py", replaceIn("p/p.go", "n > 0", "n >= 0"), applies},
		{"m2_stale.py", replaceIn("p/p.go", "n > 9", "n >= 9"), stale},
		{"m3_inert.py", replaceIn("p/p.go", "n > 0", "n > 0"), inert},
		{"m4_invalid.py", replaceIn("p/p.go", "return n > 0", "return undefinedName"), broken},
		{"m5_invalid_in_a_test.py", replaceIn("p/p.go", "func Allowed(n int) bool", "func Allowed(n string) bool"), broken},
		{"m6_escaped.py", escapeBody, escaped},
		{"m7_names_the_real_tree.py", absoluteBody, escaped},
	} {
		t.Run(c.name, func(t *testing.T) {
			was, err := os.ReadFile(real)
			if err != nil {
				t.Fatal(err)
			}
			got, detail := judge(t, root, plantMutant(t, root, c.name, c.body), t.TempDir())
			if got != c.want {
				t.Errorf("%s judged %s (%s); want %s", c.name, got, detail, c.want)
			}
			if c.name == "m6_escaped.py" {
				if err := os.WriteFile(real, was, 0o644); err != nil { // the planted escape's damage, undone for the next case
					t.Fatal(err)
				}
				return
			}
			if now, _ := os.ReadFile(real); string(now) != string(was) {
				t.Errorf("%s changed the real tree's p/p.go", c.name)
			}
		})
	}
}
