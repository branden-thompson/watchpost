package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
)

// movePath is a package's place after the batch: the longest mapped prefix of
// its repository path, or the path itself.
func movePath(pkg string, paths map[string]string) string {
	best := ""
	for from := range paths {
		if (pkg == from || strings.HasPrefix(pkg, from+"/")) && len(from) > len(best) {
			best = from
		}
	}
	if best == "" {
		return pkg
	}
	return paths[best] + strings.TrimPrefix(pkg, best)
}

// compare judges a batch's inventory against the baseline (0.19.6 A-30). A
// test's identity is its name and package, the package mapped through the
// batch's move map; accepted names a reviewed renaming, baseline key to new
// key, as a split's renamed half is. It fails a test that is gone, one whose
// outcome changed, any test that failed, and two baseline tests mapped onto
// one key; a new test is reported, not failed.
func compare(base, now map[string]string, paths, accepted map[string]string) (probs, added []string) {
	seen := map[string]bool{}
	for key, was := range base {
		pkg, test, _ := strings.Cut(key, "\t")
		want := movePath(pkg, paths) + "\t" + test
		if to, ok := accepted[key]; ok {
			want = to
		}
		if seen[want] {
			// TWO BASELINE TESTS ON ONE KEY: the second would read as kept, so a
			// move map or a renaming that merges two names hides a lost test.
			probs = append(probs, fmt.Sprintf("%s: maps onto %s, which another baseline test maps onto too", strings.ReplaceAll(key, "\t", " "), strings.ReplaceAll(want, "\t", " ")))
			continue
		}
		got, ok := now[want]
		seen[want] = true
		switch {
		case !ok:
			probs = append(probs, fmt.Sprintf("%s: was %s, and is gone", strings.ReplaceAll(key, "\t", " "), was))
		case got != was:
			probs = append(probs, fmt.Sprintf("%s: was %s, now %s", strings.ReplaceAll(want, "\t", " "), was, got))
		}
	}
	for key, got := range now {
		if got == "FAIL" && !seen[key] {
			probs = append(probs, strings.ReplaceAll(key, "\t", " ")+": FAIL")
		}
		if !seen[key] {
			added = append(added, strings.ReplaceAll(key, "\t", " ")+" "+got)
		}
	}
	sort.Strings(probs)
	sort.Strings(added)
	return probs, added
}

// runCompare prints the comparison and fails on any problem.
func runCompare(args []string) error {
	fs := flag.NewFlagSet("compare", flag.ContinueOnError)
	baseFile := fs.String("base", "", "the baseline inventory")
	nowFile := fs.String("now", "", "the batch's inventory")
	pathsFile := fs.String("paths", "", "the batch's move map, old<TAB>new per line")
	acceptFile := fs.String("accept", "", "reviewed renamings, base key<TAB>new key per line")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *baseFile == "" || *nowFile == "" {
		return errors.New("compare needs -base and -now")
	}
	base, err := read(*baseFile)
	if err != nil {
		return err
	}
	now, err := read(*nowFile)
	if err != nil {
		return err
	}
	paths, err := readPairs(*pathsFile, 2)
	if err != nil {
		return err
	}
	accepted, err := readPairs(*acceptFile, 4)
	if err != nil {
		return err
	}
	probs, added := compare(base, now, paths, accepted)
	for _, a := range added {
		fmt.Println("  new: " + a)
	}
	for _, p := range probs {
		fmt.Println("  " + p)
	}
	if len(probs) != 0 {
		return fmt.Errorf("%d test(s) lost or changed of %d in the baseline", len(probs), len(base))
	}
	fmt.Printf("testinv: all %d baseline test(s) kept with their outcomes; %d new\n", len(base), len(added))
	return nil
}

// readPairs reads tab-separated lines into from → to: two fields for a move
// map (old, new), four for a renaming (base package, base test, new package,
// new test). No file is an empty map.
func readPairs(file string, fields int) (map[string]string, error) {
	out := map[string]string{}
	if file == "" {
		return out, nil
	}
	b, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	for i, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		f := strings.Split(line, "\t")
		if len(f) != fields {
			return nil, fmt.Errorf("%s:%d: want %d tab-separated fields", file, i+1, fields)
		}
		half := fields / 2
		out[strings.Join(f[:half], "\t")] = strings.Join(f[half:], "\t")
	}
	return out, nil
}
