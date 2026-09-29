package tty

import (
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
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
func TestMain(m *testing.M) {
	time.Local = time.UTC
	os.Exit(m.Run())
}

// localWrite is a test assigning the process's zone.
var localWrite = regexp.MustCompile(`time\.Local\s*=[^=]`)

// TestNoTestSetsTheZoneMidRun holds the rule: only TestMain assigns
// time.Local, before the tests run.
func TestNoTestSetsTheZoneMidRun(t *testing.T) {
	if !localWrite.MatchString("time.Local = time.UTC") || localWrite.MatchString("time.Local == x") {
		t.Fatal("control: the pattern no longer tells a write from a comparison")
	}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		n := e.Name()
		if !strings.HasSuffix(n, "_test.go") || n == "main_test.go" {
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
