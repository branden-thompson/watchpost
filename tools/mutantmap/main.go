// mutantmap says where each mutant edits, which mutants a batch must re-vet,
// and whether a scoped sweep kept every verdict (0.19.6 A-2, 0.19.6 A-11, 0.19.6 A-30, D-30).
//
// WHY IT EXISTS. 0.19.6 moves code that 692 mutants anchor in. A sweep's own
// verdicts say a mutant was caught; they do not say it still mutates the code
// it was written for, or that the batch re-vetted every mutant it could have
// disturbed. This tool answers both against git, so the answer does not depend
// on anyone's list.
//
// WHERE A MUTANT EDITS is found by running the mutant itself, with its write
// caught, as `scripts/quality/mutant-anchors.sh` does: one parser for the
// corpus, the harness's own. The first line the write changes names the
// function the mutant lives in.
//
// usage:
//
//	go run ./tools/mutantmap killers <record>
//	go run ./tools/mutantmap derive -base <rev> -record <rev:path>
//	go run ./tools/mutantmap compare -base <rev> -record <rev:path> -log <file> -list <file> [-paths <file>] [-funcs <file>]
//	go run ./tools/mutantmap sites <rev>
package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "mutantmap: say killers, derive or compare")
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "killers":
		err = runKillers(os.Args[2:])
	case "derive":
		err = runDerive(os.Args[2:])
	case "compare":
		err = runCompare(os.Args[2:])
	case "sites":
		err = runSites(os.Args[2:])
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "mutantmap:", err)
		os.Exit(1)
	}
}

// runKillers prints the record a sweep reads its last killers from, or
// nothing when no record exists.
func runKillers(args []string) error {
	if len(args) != 1 {
		return errors.New("killers takes one record path")
	}
	got, err := killerRecord(args[0])
	if err != nil {
		return err
	}
	if got != "" {
		fmt.Println(got)
	}
	return nil
}

// sinceV is a release record's name: since-v<major>.<minor>.<patch>.log.
var sinceV = regexp.MustCompile(`^since-v(\d+)\.(\d+)\.(\d+)\.log$`)

// killerRecord is the release's own record when it exists, and otherwise the
// newest record beside it by the version in its name (0.19.6 A-2). After a tag the
// release's record does not exist until its first full sweep, and starting
// every mutant at its whole package costs about 19 hours instead of one.
func killerRecord(record string) (string, error) {
	if _, err := os.Stat(record); err == nil {
		return record, nil
	}
	dir := filepath.Dir(record)
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	best, bestV := "", []int(nil)
	for _, e := range entries {
		v := versionOf(e.Name())
		if v == nil || (bestV != nil && !lessVersion(bestV, v)) {
			continue
		}
		best, bestV = filepath.Join(dir, e.Name()), v
	}
	return best, nil
}

// versionOf reads the three numbers of a record's name, or nil.
func versionOf(name string) []int {
	m := sinceV.FindStringSubmatch(name)
	if m == nil {
		return nil
	}
	v := make([]int, 3)
	for i := range v {
		n, err := strconv.Atoi(m[i+1])
		if err != nil {
			return nil
		}
		v[i] = n
	}
	return v
}

// lessVersion orders two versions number by number, so 0.19.10 follows 0.19.5.
func lessVersion(a, b []int) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

// verdict is one mutant's line in a sweep's log: its class and, when it was
// caught by a named test, that test.
type verdict struct {
	class, killer string
}

// verdictLine is "[i/n] CLASS  path — rest".
var verdictLine = regexp.MustCompile(`^\[\d+/\d+\] ([A-Z]+) +(\S+) — (.*)$`)

// parseLog reads every verdict line of a sweep's log, by mutant path.
func parseLog(text string) map[string]verdict {
	out := map[string]verdict{}
	for _, line := range strings.Split(text, "\n") {
		m := verdictLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		v := verdict{class: m[1]}
		if f := strings.Fields(m[3]); m[1] == "CAUGHT" && len(f) > 0 && strings.HasPrefix(f[0], "Test") {
			v.killer = f[0]
		}
		out[m[2]] = v
	}
	return out
}

// runSites prints where every mutant edits at one commit: mutant, file,
// function, hash. The baseline's copy is committed beside the verdict records
// (0.19.6 A-30), so a later batch can say what each anchor sat in before 0.19.6.
func runSites(args []string) error {
	if len(args) != 1 {
		return errors.New("sites takes one commit")
	}
	root, err := repoRoot()
	if err != nil {
		return err
	}
	t := newTree(root, args[0])
	list, err := t.mutants()
	if err != nil {
		return err
	}
	found, err := t.sites(list)
	if err != nil {
		return err
	}
	if len(found) != len(list) {
		return fmt.Errorf("%d of %d mutants located at %s", len(found), len(list), args[0])
	}
	for _, m := range list {
		s := found[m]
		fmt.Printf("%s\t%s\t%s\t%s\n", m, s.target, s.fn, s.hash[:12])
	}
	return nil
}
