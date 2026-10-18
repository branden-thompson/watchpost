// Command prepush is the pre-push hook (0.19.0 W0.1, FR-10.2, REFLECT L5):
// it runs the tests that read the record - seconds, so a batch whose record
// trails its code stops before the push, not in the hosted gate. It is no
// gate: make verify still is. `make hooks` builds it into this clone's git
// folder and points git at it.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// runs are the record's tests, by package.
var runs = []struct {
	pkg   string
	tests []string
}{
	{"./cmd/watchpost", []string{
		"TestWhereThingsHappenNamesRealSymbols", "TestAcceptedCostsNamesRealSymbols", "TestFollowUpsCiteTestsThatExist",
		"TestTheRosterCitesTestsThatExist", "TestEveryRatifiedP10RowNamesCodeThatExists", "TestAMethodRowNamesItsReceiver",
		"TestEveryRequirementsTestIsInAPlanTask", "TestEveryRulingNamingCodeCitesItsLine",
	}},
	{"./app", []string{"TestTheAsBuiltMapIsKeptInStep"}},
	{"./tools/atlas", []string{"TestTheAtlasMatchesAFreshRender"}},
}

func main() {
	run := run
	if len(os.Args) > 1 && os.Args[1] == "-install" {
		run = install // make hooks
	}
	err := run()
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, "prepush:", err)
	os.Exit(1)
}

// zeroSHA is the object name git gives a ref being deleted.
const zeroSHA = "0000000000000000000000000000000000000000"

// treeIsPushed says whether the working tree the tests read is what is being
// pushed (BUILD red team H-9): a tree with changes to tracked files is not, and
// is refused; a push of commits other than the one checked out is let through
// with a note that its record is not read here - make verify is the gate. It
// takes git's pre-push lines: local ref, local name, remote ref, remote name.
func treeIsPushed(changed, head, pushed string) (note string, err error) {
	if changed != "" {
		return "", errors.New("the working tree has changes the push does not carry, and the tests would read them: commit or stash them, then push")
	}
	for _, line := range strings.Split(strings.TrimSpace(pushed), "\n") {
		f := strings.Fields(line)
		if len(f) == 4 && f[1] != head && f[1] != zeroSHA {
			return f[0] + " is not the checked-out commit, so its record is not read here; make verify is the gate", nil
		}
	}
	return "", nil
}

// run runs each package's record tests from the working tree's root, and
// says which failed.
func run() error {
	if len(runs) == 0 {
		return errors.New("no tests to run: a hook that runs nothing passes every push")
	}
	root, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return fmt.Errorf("not in a git working tree: %w", err)
	}
	changed, err := exec.Command("git", "status", "--porcelain", "--untracked-files=no").Output()
	if err != nil {
		return fmt.Errorf("the working tree could not be read: %w", err)
	}
	head, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		return fmt.Errorf("the checked-out commit could not be read: %w", err)
	}
	pushed, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("the refs being pushed could not be read: %w", err)
	}
	if note, err := treeIsPushed(strings.TrimSpace(string(changed)), strings.TrimSpace(string(head)), string(pushed)); err != nil || note != "" {
		if note != "" {
			fmt.Fprintln(os.Stderr, "prepush:", note)
		}
		return err
	}
	for _, r := range runs {
		cmd := exec.Command("go", "test", "-count=1", "-run", "^("+strings.Join(r.tests, "|")+")$", r.pkg)
		cmd.Dir, cmd.Stdout, cmd.Stderr = strings.TrimSpace(string(root)), os.Stdout, os.Stderr
		err = cmd.Run()
		if err != nil {
			return fmt.Errorf("the record's tests in %s failed: fix the record, then push again", r.pkg)
		}
	}
	return nil
}
