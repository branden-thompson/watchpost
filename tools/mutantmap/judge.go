package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// derive is the batch's re-vet list (0.19.6 A-11, 0.19.6 A-30): every mutant whose anchor's
// file or function changed between the batch's base and its head, whose last
// killer's declaration changed, whose own file changed, or which is new. A
// mutant judged against code nothing touched is left out; the closing sweep
// judges the whole corpus.
func derive(root, base, head string, killers map[string]verdict) ([]string, error) {
	b, h := newTree(root, base), newTree(root, head)
	list, err := h.mutants()
	if err != nil {
		return nil, err
	}
	was, err := b.sites(list)
	if err != nil {
		return nil, err
	}
	now, err := h.sites(list)
	if err != nil {
		return nil, err
	}
	names := map[string]bool{}
	for _, v := range killers {
		names[v.killer] = v.killer != ""
	}
	tb, err := b.tests(names)
	if err != nil {
		return nil, err
	}
	th, err := h.tests(names)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, m := range list {
		changed, err := mutantChanged(b, h, m)
		if err != nil {
			return nil, err
		}
		w, wasThere := was[m]
		if changed || !wasThere || w != now[m] || !sameDefs(tb[killers[m].killer], th[killers[m].killer]) {
			out = append(out, m)
		}
	}
	return out, nil
}

// mutantChanged reports whether a mutant's own file differs between the trees.
func mutantChanged(b, h *tree, m string) (bool, error) {
	x, err := b.read(m)
	if err != nil {
		return false, err
	}
	y, err := h.read(m)
	if err != nil {
		return false, err
	}
	return string(x) != string(y), nil
}

// sameDefs reports whether a test's declarations match in path and source.
func sameDefs(a, b []testDef) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// maps is a batch's re-home and move maps: a function's new name (Recv.Name or
// Name) and a path's new place, a file or a directory and everything under it.
type maps struct {
	funcs, paths map[string]string
	// scope is the package directory a function's rename is for, when its
	// line names one (0.19.6 A-49): B2's `tick` is modes/tty's alone.
	scope map[string]string
}

// path is a path's place after the batch, by its longest mapped prefix.
func (m maps) path(p string) string {
	best := ""
	for from := range m.paths {
		if (p == from || strings.HasPrefix(p, from+"/")) && len(from) > len(best) {
			best = from
		}
	}
	if best == "" {
		return p
	}
	return m.paths[best] + strings.TrimPrefix(p, best)
}

// fn is the name after the batch of a function in the package at dir.
func (m maps) fn(f, dir string) string {
	if to, ok := m.funcs[f]; ok && (m.scope[f] == "" || m.scope[f] == dir) {
		return to
	}
	return f
}

// compare judges a scoped sweep (C-12, 0.19.6 A-30). It fails a mutant the list names
// and the log does not, a mutant the log judged and the list does not name, a
// class that changed, a killer that changed name or package, and an anchor
// that no longer sits where the batch's maps say it went.
func compare(root, base, head string, record, log map[string]verdict, list []string, m maps) ([]string, error) {
	var probs []string
	want := map[string]bool{}
	for _, l := range list {
		want[l] = true
		if _, ok := log[l]; !ok {
			probs = append(probs, filepath.Base(l)+": derived but not judged")
		}
	}
	for l := range log {
		if !want[l] {
			probs = append(probs, filepath.Base(l)+": judged but not derived")
		}
	}
	more, err := structural(root, base, head, record, list, m)
	if err != nil {
		return nil, err
	}
	probs = append(probs, more...)
	for _, l := range list {
		probs = append(probs, classLoss(l, record, log)...)
	}
	sort.Strings(probs)
	return probs, nil
}

// classLoss compares one mutant's verdict with the record's. A mutant the
// record lacks is new, and a new mutant must be caught.
func classLoss(l string, record, log map[string]verdict) []string {
	got, judged := log[l]
	if !judged {
		return nil
	}
	name := filepath.Base(l)
	was, known := record[l]
	if !known {
		if got.class != "CAUGHT" {
			return []string{name + ": a new mutant, " + got.class}
		}
		return nil
	}
	if was.class != got.class {
		return []string{fmt.Sprintf("%s: was %s, now %s", name, was.class, got.class)}
	}
	if was.killer != got.killer {
		return []string{fmt.Sprintf("%s: its killer was %s, now %s", name, was.killer, got.killer)}
	}
	return nil
}

// structural checks where each mutant's anchor and killer sit, base against
// head, through the batch's maps.
func structural(root, base, head string, record map[string]verdict, list []string, m maps) ([]string, error) {
	b, h := newTree(root, base), newTree(root, head)
	was, err := b.sites(list)
	if err != nil {
		return nil, err
	}
	now, err := h.sites(list)
	if err != nil {
		return nil, err
	}
	names := map[string]bool{}
	for _, v := range record {
		names[v.killer] = v.killer != ""
	}
	tb, err := b.tests(names)
	if err != nil {
		return nil, err
	}
	th, err := h.tests(names)
	if err != nil {
		return nil, err
	}
	var probs []string
	for _, l := range list {
		w, ok := was[l]
		if !ok {
			continue
		}
		n := now[l]
		if p := m.path(w.target); p != n.target || m.fn(w.fn, filepath.Dir(w.target)) != n.fn {
			probs = append(probs, anchorMoved(filepath.Base(l), w, n, p != n.target))
		}
		probs = append(probs, killerMoved(l, record[l].killer, tb, th, m)...)
	}
	return probs, nil
}

// anchorMoved says where a mutant's anchor sat and where it sits now: by
// function, with the files too when the file is what the maps do not carry, so
// a function moved whole to a file no map names reads as a move and not as
// "sat in F, now in F" (0.19.6 D-69).
func anchorMoved(name string, was, now site, fileMoved bool) string {
	if fileMoved {
		return fmt.Sprintf("%s: its anchor sat in %s:%s, now in %s:%s", name, was.target, was.fn, now.target, now.fn)
	}
	return fmt.Sprintf("%s: its anchor sat in %s, now in %s", name, was.fn, now.fn)
}

// killerMoved fails a killer declared at head neither where it was nor where
// the batch's move map says its file went. A map names files, and a file can
// keep some of its tests while others leave it.
func killerMoved(l, k string, tb, th map[string][]testDef, m maps) []string {
	if k == "" || len(tb[k]) == 0 {
		return nil
	}
	at := map[string]bool{}
	for _, d := range th[k] {
		at[filepath.Dir(d.file)] = true
	}
	for _, d := range tb[k] {
		if at[filepath.Dir(m.path(d.file))] || at[filepath.Dir(d.file)] {
			return nil
		}
	}
	var now []string
	for d := range at {
		now = append(now, d)
	}
	sort.Strings(now)
	return []string{fmt.Sprintf("%s: its killer %s was in %s, now in %s", filepath.Base(l), k, filepath.Dir(tb[k][0].file), strings.Join(now, ","))}
}

// runDerive prints the batch's re-vet list, one mutant per line.
func runDerive(args []string) error {
	fs := flag.NewFlagSet("derive", flag.ContinueOnError)
	base := fs.String("base", "", "the commit the batch began from")
	rec := fs.String("record", "", "the verdict record, as rev:path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *base == "" || *rec == "" {
		return errors.New("derive needs -base and -record")
	}
	root, record, err := setup(*rec)
	if err != nil {
		return err
	}
	list, err := derive(root, *base, "HEAD", record)
	if err != nil {
		return err
	}
	for _, l := range list {
		fmt.Println(l)
	}
	fmt.Fprintf(os.Stderr, "mutantmap: %d mutant(s) derived since %s\n", len(list), *base)
	return nil
}

// runCompare judges a scoped sweep's log; any problem fails it.
func runCompare(args []string) error {
	fs := flag.NewFlagSet("compare", flag.ContinueOnError)
	base := fs.String("base", "", "the commit the batch began from")
	rec := fs.String("record", "", "the verdict record, as rev:path")
	logf := fs.String("log", "", "the scoped sweep's log")
	listf := fs.String("list", "", "the derived list")
	pathsf := fs.String("paths", "", "the batch's move map, old<TAB>new per line")
	funcsf := fs.String("funcs", "", "the batch's re-home map, old<TAB>new per line")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *base == "" || *rec == "" || *logf == "" || *listf == "" {
		return errors.New("compare needs -base, -record, -log and -list")
	}
	root, record, err := setup(*rec)
	if err != nil {
		return err
	}
	logText, err := os.ReadFile(*logf)
	if err != nil {
		return err
	}
	listText, err := os.ReadFile(*listf)
	if err != nil {
		return err
	}
	m, err := readMaps(*pathsf, *funcsf)
	if err != nil {
		return err
	}
	list := strings.Fields(string(listText))
	probs, err := compare(root, *base, "HEAD", record, parseLog(string(logText)), list, m)
	if err != nil {
		return err
	}
	for _, p := range probs {
		fmt.Println("  " + p)
	}
	if len(probs) != 0 {
		return fmt.Errorf("the scoped sweep lost %d verdict(s) of %d", len(probs), len(list))
	}
	fmt.Printf("mutantmap: %d mutant(s), every verdict, killer and anchor kept\n", len(list))
	return nil
}

// setup finds the repository's root and reads the verdict record from git.
func setup(rec string) (string, map[string]verdict, error) {
	root, err := repoRoot()
	if err != nil {
		return "", nil, err
	}
	rev, path, ok := strings.Cut(rec, ":")
	if !ok {
		return "", nil, fmt.Errorf("-record %q is not rev:path", rec)
	}
	text, err := newTree(root, rev).read(path)
	if err != nil || text == nil {
		return "", nil, fmt.Errorf("the record %s is missing: %v", rec, err)
	}
	return root, parseLog(string(text)), nil
}

// readMaps reads the two optional maps.
func readMaps(pathsFile, funcsFile string) (maps, error) {
	p, err := readMap(pathsFile)
	if err != nil {
		return maps{}, err
	}
	f, err := readMap(funcsFile)
	if err != nil {
		return maps{}, err
	}
	scope := map[string]string{}
	for from, to := range f { // bounded by the map (P10-02)
		if name, pkg, ok := strings.Cut(to, "\t"); ok {
			f[from], scope[from] = name, pkg
		}
	}
	return maps{paths: p, funcs: f, scope: scope}, nil
}

// readMap reads old<TAB>new lines, a blank line or one starting with # read
// as nothing, so one file can be both this map and gcprobe's; no file is an
// empty map. A third field, the package a rename is for, stays in the new
// name's text, which readMaps splits off for the function map.
func readMap(file string) (map[string]string, error) {
	out := map[string]string{}
	if file == "" {
		return out, nil
	}
	b, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	for i, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		from, to, ok := strings.Cut(line, "\t")
		if !ok {
			return nil, fmt.Errorf("%s:%d: not old<TAB>new", file, i+1)
		}
		out[from] = to
	}
	return out, nil
}

// repoRoot is the repository's top directory.
func repoRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("not in a repository: %v", err)
	}
	return strings.TrimSpace(string(out)), nil
}
