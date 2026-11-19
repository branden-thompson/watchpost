package main

import (
	"bytes"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// arm writes one arm's benchmark file: each named benchmark ten times at its
// nanoseconds, with 1 % of noise from a fixed seed, under its package.
func arm(t *testing.T, name string, seed uint64, benches map[string]float64, pkgOf map[string]string) string {
	t.Helper()
	r := rand.New(rand.NewPCG(seed, 1))
	var b strings.Builder
	b.WriteString("goos: darwin\ngoarch: arm64\ncpu: planted\n")
	byPkg := map[string][]string{}
	for n := range benches { // bounded by the planted benchmarks (P10-02)
		byPkg[pkgOf[n]] = append(byPkg[pkgOf[n]], n)
	}
	for _, pkg := range sortedPkgs(byPkg) { // bounded by the packages (P10-02)
		fmt.Fprintf(&b, "pkg: %s\n", pkg)
		for range 10 { // ten counts (P10-02)
			for _, n := range byPkg[pkg] { // bounded by the package (P10-02)
				fmt.Fprintf(&b, "%s-4   \t   10000\t  %.0f ns/op\n", n, benches[n]*(0.99+0.02*r.Float64()))
			}
		}
	}
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// sortedPkgs is a map's keys in order.
func sortedPkgs(m map[string][]string) []string {
	var out []string
	for k := range m { // bounded by the map (P10-02)
		out = append(out, k)
	}
	for i := range out { // a short sort (P10-02)
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

// TestThePairsAreJudgedByD28 is 0.19.6 D-28 on planted arms, through the
// pinned benchstat: a 5 % slowdown in both pairs fails; in one pair alone it
// passes; a benchmark that moved package keeps its pair; one with no pair
// fails; a name two packages declare is refused.
func TestThePairsAreJudgedByD28(t *testing.T) {
	if testing.Short() {
		t.Skip("runs benchstat")
	}
	base := map[string]float64{"BenchmarkFrame": 70_000, "BenchmarkSteady": 1_000}
	tty := map[string]string{"BenchmarkFrame": "m/modes/tty", "BenchmarkSteady": "m/modes/tty", "BenchmarkGone": "m/app"}
	moved := map[string]string{"BenchmarkFrame": "m/modes/tty/mapview", "BenchmarkSteady": "m/modes/tty"}
	slow := map[string]float64{"BenchmarkFrame": 73_500, "BenchmarkSteady": 1_000}
	judged := func(files ...string) (int, string) {
		var b bytes.Buffer
		code := run(append([]string{"judge"}, files...), &out{w: &b})
		return code, b.String()
	}
	b1, b2 := arm(t, "b1", 1, base, tty), arm(t, "b2", 2, base, tty)
	if code, said := judged(b1, arm(t, "n1", 3, base, moved), b2, arm(t, "n2", 4, base, moved)); code != 0 {
		t.Errorf("the same times, one benchmark moved package: exit %d, %s", code, said)
	}
	if code, said := judged(b1, arm(t, "n1", 3, slow, tty), b2, arm(t, "n2", 4, slow, tty)); code == 0 || !strings.Contains(said, "BenchmarkFrame is slower in both pairs") && !strings.Contains(said, "Frame is slower in both pairs") {
		t.Errorf("5 %% slower in both pairs: exit %d, %s", code, said)
	}
	if code, said := judged(b1, arm(t, "n1", 3, slow, tty), b2, arm(t, "n2", 4, base, tty)); code != 0 {
		t.Errorf("slower in one pair alone: exit %d, %s", code, said)
	}
	withGone := map[string]float64{"BenchmarkFrame": 70_000, "BenchmarkSteady": 1_000, "BenchmarkGone": 500}
	g1, g2 := arm(t, "g1", 1, withGone, tty), arm(t, "g2", 2, withGone, tty)
	if code, said := judged(g1, arm(t, "n1", 3, base, tty), g2, arm(t, "n2", 4, base, tty)); code == 0 || !strings.Contains(said, "Gone has no pair") {
		t.Errorf("a base benchmark with no pair: exit %d, %s", code, said)
	}
	twice := arm(t, "twice", 5, map[string]float64{"BenchmarkFrame": 70_000}, map[string]string{"BenchmarkFrame": "m/a"})
	if err := os.WriteFile(twice, append(must(os.ReadFile(twice)), []byte("pkg: m/b\nBenchmarkFrame-4 \t 100 \t 70000 ns/op\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, said := judged(twice, b1, b2, b2); code == 0 || !strings.Contains(said, "declared in two packages") {
		t.Errorf("a name two packages declare: exit %d, %s", code, said)
	}
}

// must is a value with its error dropped, for planted files the test wrote.
func must(b []byte, err error) []byte {
	if err != nil {
		panic(err)
	}
	return b
}

// TestARowIsRead holds benchstat's CSV row forms: a change, no change, and
// no new figure.
func TestARowIsRead(t *testing.T) {
	for _, c := range []struct {
		rec  []string
		want row
	}{
		{[]string{"Frame-4", "7e-05", "1%", "7.3e-05", "1%", "+5.23%", "p=0.000 n=10"}, row{delta: 5.23, p: 0, paired: true}},
		{[]string{"Steady-4", "1e-06", "1%", "1e-06", "1%", "~", "p=1.000 n=10"}, row{p: 1, paired: true, same: true}},
		{[]string{"Gone-4", "5e-07", "1%"}, row{}},
	} {
		if got := parseRow(c.rec); got != c.want {
			t.Errorf("%v read as %+v; want %+v", c.rec, got, c.want)
		}
	}
	if benchName("Frame_133x44-4") != "Frame_133x44" || benchName("Frame") != "Frame" {
		t.Error("the GOMAXPROCS suffix is not dropped")
	}
}
