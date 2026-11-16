package tty

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/modes/tty/internal/ttytest"
	"github.com/branden-thompson/watchpost/platform/declset"
)

// TestMain pins the package's zone to UTC ONCE, before any goroutine exists.
//
// FOUR TESTS SET time.Local THEMSELVES, and restored it after: a write to a
// global that time.Now reads, while timers left sleeping by earlier tests (a
// tea.Tick's command, which msgsOf does not wait out) called time.Now on their
// own goroutines. -race caught it (W14, 2026-09-29), in
// TestAVoiceNoteFromTheDeckIsDrawnUnderTheRowThatAskedForIt, one run in three.
// Every one of them wanted UTC - the goldens' stamps the same on every machine -
// so the whole package gets it, and nothing writes it mid-run.
//
// THE PIN IS ttytest.Main's (0.19.6 TG-3), the one every package under
// modes/tty calls, so a package a split creates stands on the same ground.
func TestMain(m *testing.M) {
	ttytest.Main(m)
}

// localWrite is a test assigning the process's zone.
var localWrite = regexp.MustCompile(`time\.Local\s*=[^=]`)

// TestNoTestSetsTheZoneMidRun holds the rule: only TestMain assigns
// time.Local, before the tests run.
func TestNoTestSetsTheZoneMidRun(t *testing.T) {
	if !localWrite.MatchString("time.Local = time.UTC") || localWrite.MatchString("time.Local == x") {
		t.Fatal("control: the pattern no longer tells a write from a comparison")
	}
	paths, err := declset.TestPaths(".") // the whole modes/tty tree (0.19.6 TG-2)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) < 200 { // modes/tty's test files at 0.19.6 B0: 213
		t.Fatalf("the rule read %d test files; a moved tree must not pass it empty", len(paths))
	}
	for _, n := range paths {
		if filepath.Base(n) == "main_test.go" {
			continue
		}
		src, err := os.ReadFile(n)
		if err != nil {
			t.Fatal(err)
		}
		for i, line := range strings.Split(string(src), "\n") {
			if localWrite.MatchString(line) && !strings.HasPrefix(strings.TrimSpace(line), "//") {
				t.Errorf("%s:%d sets time.Local mid-run - a race with any timer's time.Now; TestMain pins it once", n, i+1)
			}
		}
	}
}
