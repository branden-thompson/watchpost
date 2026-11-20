// testinv records which tests ran under the two trees 0.19.6 splits, and how
// each ended, and compares a batch's record with the baseline (0.19.6 D-36,
// RR F5, A-30).
//
// WHY IT EXISTS. A move can lose a test without failing anything: a file left
// behind, a build tag the new package lacks, a fixture path that now resolves
// to nothing so the test skips. `make verify` is green over fewer tests, and
// says so nowhere. The inventory names every test, subtests included, with its
// outcome, in each build mode, and the comparison fails a test that is gone or
// that ends differently.
//
// usage:
//
//	go run ./tools/testinv run -mode plain|property|race -out <file>
//	go run ./tools/testinv compare -base <file> -now <file> [-paths <file>] [-accept <file>]
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
)

// module is the import path the inventory strips, so a key reads as a
// repository path.
const module = "github.com/branden-thompson/watchpost/"

// trees are the packages the inventory runs.
var trees = []string{"./app/...", "./modes/tty/..."}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "testinv: say run or compare")
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "run":
		err = runInventory(os.Args[2:])
	case "compare":
		err = runCompare(os.Args[2:])
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "testinv:", err)
		os.Exit(1)
	}
}

// modeArgs is the go test flags of one build mode.
func modeArgs(mode string) ([]string, error) {
	switch mode {
	case "plain":
		return nil, nil
	case "property":
		return []string{"-tags", "property"}, nil
	case "race":
		return []string{"-race"}, nil
	}
	return nil, fmt.Errorf("mode %q is not plain, property or race", mode)
}

// event is the part of a `go test -json` line the inventory reads.
type event struct {
	Action, Package, Test string
}

// outcomes reads a `go test -json` stream into package<TAB>test → PASS, SKIP
// or FAIL, subtests included.
func outcomes(r io.Reader) (map[string]string, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, line := range strings.Split(string(b), "\n") { // bounded by the stream's lines (P10-02)
		var e event
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue // a build error's text, which go test prints raw
		}
		if e.Test == "" {
			continue
		}
		switch e.Action {
		case "pass", "skip", "fail":
			out[strings.TrimPrefix(e.Package, module)+"\t"+e.Test] = strings.ToUpper(e.Action)
		}
	}
	return out, nil
}

// runInventory runs one mode over the two trees and writes its inventory,
// sorted. A failing test is recorded as FAIL and fails the run.
func runInventory(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	mode := fs.String("mode", "", "plain, property or race")
	outFile := fs.String("out", "", "where to write the inventory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	extra, err := modeArgs(*mode)
	if err != nil {
		return err
	}
	if *outFile == "" {
		return errors.New("run needs -out")
	}
	// -timeout is make race's, for the same reason: modes/tty under the race
	// detector, on the efficiency cores, runs past go test's default 10
	// minutes, and a timed-out package records none of the tests after it.
	cmd := exec.Command("go", append(append([]string{"test", "-json", "-count=1", "-timeout", "30m"}, extra...), trees...)...)
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	got, err := outcomes(stdout)
	if err != nil {
		return err
	}
	waitErr := cmd.Wait()
	if len(got) == 0 {
		return fmt.Errorf("%s mode recorded no test: the run did not build or did not start (%v)", *mode, waitErr)
	}
	if err := write(*outFile, got); err != nil {
		return err
	}
	fmt.Printf("testinv: %s mode, %d test(s) and subtest(s) recorded in %s\n", *mode, len(got), *outFile)
	if waitErr != nil {
		return fmt.Errorf("go test failed in %s mode: %v", *mode, waitErr)
	}
	return nil
}

// write saves an inventory as sorted lines.
func write(path string, inv map[string]string) error {
	lines := make([]string, 0, len(inv))
	for k, v := range inv {
		lines = append(lines, k+"\t"+v)
	}
	sort.Strings(lines)
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}

// read loads an inventory written by write.
func read(path string) (map[string]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for i, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		f := strings.Split(line, "\t")
		if len(f) != 3 {
			return nil, fmt.Errorf("%s:%d: not package<TAB>test<TAB>outcome", path, i+1)
		}
		if f[2] != "PASS" && f[2] != "SKIP" && f[2] != "FAIL" {
			return nil, fmt.Errorf("%s:%d: outcome %q is not PASS, SKIP or FAIL", path, i+1, f[2])
		}
		if _, dup := out[f[0]+"\t"+f[1]]; dup {
			return nil, fmt.Errorf("%s:%d: %s %s is listed twice", path, i+1, f[0], f[1])
		}
		out[f[0]+"\t"+f[1]] = f[2]
	}
	return out, nil
}
