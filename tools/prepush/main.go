// Command prepush is the pre-push hook (0.19.0 W0.1, FR-10.2, REFLECT L5):
// it runs the tests that read the record - seconds, so a batch whose record
// trails its code stops before the push, not in the hosted gate. It is no
// gate: make verify still is. `make hooks` builds it into this clone's git
// folder and points git at it.
package main

import (
	"errors"
	"fmt"
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
	err := run()
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, "prepush:", err)
	os.Exit(1)
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
