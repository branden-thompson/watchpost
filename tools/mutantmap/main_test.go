package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// fixture is a throwaway repository with one Go file, its tests and two
// mutants, each mutant editing a function of its own.
func fixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	write(t, dir, "x/x.go", "package x\n\nfunc F() int { return 1 }\n\nfunc G() int { return 2 }\n")
	write(t, dir, "x/x_test.go", "package x\n\nimport \"testing\"\n\nfunc TestF(t *testing.T) { _ = F() }\n\nfunc TestG(t *testing.T) { _ = G() }\n")
	write(t, dir, "06_docs/mutants/mF.py", mutant("x/x.go", "return 1", "return 9"))
	write(t, dir, "06_docs/mutants/mG.py", mutant("x/x.go", "return 2", "return 8"))
	git(t, dir, "init", "-q")
	commit(t, dir, "base")
	return dir
}

func mutant(target, old, new string) string {
	return "import pathlib\np = pathlib.Path(\"" + target + "\"); s = p.read_text()\nold = \"" + old + "\"\nassert old in s\np.write_text(s.replace(old, \"" + new + "\", 1))\n"
}

func write(t *testing.T, dir, rel, body string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func commit(t *testing.T, dir, msg string) string {
	t.Helper()
	git(t, dir, "add", "-A")
	git(t, dir, "-c", "user.name=t", "-c", "user.email=t@example.invalid", "commit", "-q", "-m", msg)
	return git(t, dir, "rev-parse", "HEAD")
}

// record is a sweep's record over the fixture's two mutants, each caught by
// its own test.
const record = `commit: 0 — 2026-11-16T00:00Z
mutant-verdicts: 2 mutant(s), commit 0, 4 workers

baseline ./... ... green
[1/2] CAUGHT   06_docs/mutants/mF.py — TestF (0.00s)
[2/2] CAUGHT   06_docs/mutants/mG.py — TestG (0.00s)

mutant-verdicts: 2 mutant(s) — 2 CAUGHT, 0 SURVIVED, 0 NO EVIDENCE
`

// TestTheKillerRecordFallsBackToTheNewestByVersion is 0.19.6 A-2 and H-4: after a
// release tag the release's own record does not exist yet, and the sweep
// reads the newest record beside it, by the version in its name, not by
// string order.
func TestTheKillerRecordFallsBackToTheNewestByVersion(t *testing.T) {
	dir := t.TempDir()
	for _, v := range []string{"0.19.3", "0.19.10", "0.19.5"} {
		write(t, dir, "since-v"+v+".log", "x")
	}
	got, err := killerRecord(filepath.Join(dir, "since-v0.19.11.log"))
	if err != nil || got != filepath.Join(dir, "since-v0.19.10.log") {
		t.Errorf("with no record for the tag, the killers come from %q (%v); want since-v0.19.10.log", got, err)
	}
	got, err = killerRecord(filepath.Join(dir, "since-v0.19.5.log"))
	if err != nil || got != filepath.Join(dir, "since-v0.19.5.log") {
		t.Errorf("the tag's own record is %q (%v); want since-v0.19.5.log", got, err)
	}
	got, err = killerRecord(filepath.Join(t.TempDir(), "since-v1.0.0.log"))
	if err != nil || got != "" {
		t.Errorf("with no record at all the killers come from %q (%v); want none", got, err)
	}
}

// TestDeriveTakesWhatTheBatchChanged is 0.19.6 A-11 and 0.19.6 A-30: a batch re-vets the
// mutants whose function changed, whose last killer changed, and every new
// one; a mutant whose function and killer are untouched is left out.
func TestDeriveTakesWhatTheBatchChanged(t *testing.T) {
	dir := fixture(t)
	base := git(t, dir, "rev-parse", "HEAD")
	write(t, dir, "x/x.go", "package x\n\nfunc F() int {\n\treturn 1\n}\n\nfunc G() int { return 2 }\n")
	commit(t, dir, "F reshaped")
	got := derived(t, dir, base)
	if got != "06_docs/mutants/mF.py" {
		t.Errorf("a changed F derives %q; want mF alone", got)
	}
	write(t, dir, "x/x_test.go", "package x\n\nimport \"testing\"\n\nfunc TestF(t *testing.T) { _ = F() }\n\nfunc TestG(t *testing.T) { if G() != 2 { t.Fatal() } }\n")
	write(t, dir, "06_docs/mutants/mN.py", mutant("x/x.go", "return 2", "return 7"))
	commit(t, dir, "TestG rewritten, mN added")
	got = derived(t, dir, base)
	if got != "06_docs/mutants/mF.py 06_docs/mutants/mG.py 06_docs/mutants/mN.py" {
		t.Errorf("a rewritten killer and a new mutant derive %q; want mF, mG and mN", got)
	}
}

func derived(t *testing.T, dir, base string) string {
	t.Helper()
	killers := parseLog(record)
	list, err := derive(dir, base, "HEAD", killers)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Join(list, " ")
}

// TestCompareFailsOnEveryLoss is 0.19.6 A-30 and C-12: a scoped sweep passes only
// when it judged exactly the derived list, every mutant kept its class and its
// killer, and every anchor still sits in the function it sat in, once the
// batch's maps are applied.
func TestCompareFailsOnEveryLoss(t *testing.T) {
	dir := fixture(t)
	base := git(t, dir, "rev-parse", "HEAD")
	same := "[1/2] CAUGHT   06_docs/mutants/mF.py — TestF (0.00s)\n[2/2] CAUGHT   06_docs/mutants/mG.py — TestG (0.00s)\n"
	cases := []struct {
		name, log, want string
	}{
		{"the same verdicts", same, ""},
		{"a mutant not judged", "[1/1] CAUGHT   06_docs/mutants/mF.py — TestF (0.00s)\n", "mG.py: derived but not judged"},
		{"a mutant judged but not derived", same + "[3/3] CAUGHT   06_docs/mutants/mZ.py — TestF (0.00s)\n", "mZ.py: judged but not derived"},
		{"a catch lost", "[1/2] SURVIVED 06_docs/mutants/mF.py — NOTHING FAILS.\n[2/2] CAUGHT   06_docs/mutants/mG.py — TestG (0.00s)\n", "mF.py: was CAUGHT, now SURVIVED"},
		{"another killer", "[1/2] CAUGHT   06_docs/mutants/mF.py — TestG (0.00s)\n[2/2] CAUGHT   06_docs/mutants/mG.py — TestG (0.00s)\n", "mF.py: its killer was TestF, now TestG"},
	}
	for _, c := range cases {
		probs, err := compare(dir, base, "HEAD", parseLog(record), parseLog(c.log), []string{"06_docs/mutants/mF.py", "06_docs/mutants/mG.py"}, maps{})
		if err != nil {
			t.Fatal(c.name, err)
		}
		got := strings.Join(probs, "; ")
		if (c.want == "") != (got == "") || !strings.Contains(got, c.want) {
			t.Errorf("%s: the comparator says %q; want %q", c.name, got, c.want)
		}
	}
}

// TestCompareFollowsTheBatchsMaps is 0.19.6 A-30: an anchor that moved to another
// function, or a killer that moved to another package, fails unless the
// batch's re-home or move map says where it went.
func TestCompareFollowsTheBatchsMaps(t *testing.T) {
	dir := fixture(t)
	base := git(t, dir, "rev-parse", "HEAD")
	write(t, dir, "x/x.go", "package x\n\nfunc F() int { return 0 }\n\nfunc G() int { return 2 }\n\nfunc H() int { return 1 }\n")
	write(t, dir, "y/y_test.go", "package y\n\nimport \"testing\"\n\nfunc TestF(t *testing.T) {}\n")
	write(t, dir, "x/x_test.go", "package x\n\nimport \"testing\"\n\nfunc TestG(t *testing.T) { _ = G() }\n")
	commit(t, dir, "mF's line now sits in H; TestF moved to y")
	log := parseLog("[1/2] CAUGHT   06_docs/mutants/mF.py — TestF (0.00s)\n[2/2] CAUGHT   06_docs/mutants/mG.py — TestG (0.00s)\n")
	list := []string{"06_docs/mutants/mF.py", "06_docs/mutants/mG.py"}
	probs, err := compare(dir, base, "HEAD", parseLog(record), log, list, maps{})
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(probs, "; ")
	if !strings.Contains(got, "mF.py: its anchor sat in F, now in H") || !strings.Contains(got, "mF.py: its killer TestF was in x, now in y") {
		t.Errorf("with no maps the comparator says %q; want the anchor and the killer reported", got)
	}
	m := maps{funcs: map[string]string{"F": "H"}, paths: map[string]string{"x/x_test.go": "y/y_test.go"}}
	probs, err = compare(dir, base, "HEAD", parseLog(record), log, list, m)
	if err != nil || len(probs) != 0 {
		t.Errorf("with the maps the comparator says %v (%v); want nothing", probs, err)
	}
}

// TestTheMapsReadAsGcprobesDo is 0.19.6 A-44: a map's comment and blank lines
// read as nothing, as gcprobe reads them, so the release's one renames file
// serves both; a line with no tab is still refused.
func TestTheMapsReadAsGcprobesDo(t *testing.T) {
	file := filepath.Join(t.TempDir(), "renames.tsv")
	write(t, filepath.Dir(file), filepath.Base(file), "# a header\n\nF\tH\n")
	m, err := readMap(file)
	if err != nil || len(m) != 1 || m["F"] != "H" {
		t.Errorf("a commented map read as %v (%v); want F to H alone", m, err)
	}
	write(t, filepath.Dir(file), filepath.Base(file), "# a header\nF H\n")
	if _, err := readMap(file); err == nil {
		t.Error("a line with no tab was read; want it refused")
	}
}

// TestARenameScopedToAPackageMapsThatPackageAlone holds 0.19.6 A-49: a line
// that names a package renames that package's function, so modes/tty's `tick`
// becoming `Tick` leaves another package's `tick` where it was. Seen failing
// with the scope read as part of the new name.
func TestARenameScopedToAPackageMapsThatPackageAlone(t *testing.T) {
	file := filepath.Join(t.TempDir(), "renames.tsv")
	write(t, filepath.Dir(file), filepath.Base(file), "tick\tTick\tmodes/tty\nF\tH\n")
	m, err := readMaps("", file)
	if err != nil {
		t.Fatal(err)
	}
	if got := m.fn("tick", "modes/tty"); got != "Tick" {
		t.Errorf("modes/tty's tick maps to %q; want Tick", got)
	}
	if got := m.fn("tick", "app"); got != "tick" {
		t.Errorf("app's tick maps to %q; want it left alone", got)
	}
	if got := m.fn("F", "app"); got != "H" {
		t.Errorf("an unscoped rename maps app's F to %q; want H", got)
	}
}
