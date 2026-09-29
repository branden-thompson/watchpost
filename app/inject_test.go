package app

import (
	"os"
	"strings"
	"testing"
)

// THE INJECTOR IS IN EVERY BUILD (0.18.0 D-152, overturning F-21b).
//
// It was build-tagged out of release binaries because a screenshot of a
// fabricated tornado warning was indistinguishable from a real one. Since then
// every surface a test event reaches says it is a test, it lives two minutes,
// and ctrl+d asks ARE YOU SURE first. A station operator tests their alerts as
// a radio station does (HUM LEAD, #9), so the capability ships, and the rule is
// that it can fabricate nothing UNMARKED (TestEveryFabricatedEventIsMarkedAsOne,
// TestATestEventIsMarkedOnEverySurface).
//
// This is the guard against the tag coming back: it runs in the plain build, so
// a release path that loses the injector fails here, not in a listener's hands.
func TestTheInjectorIsInEveryBuild(t *testing.T) {
	if (&livePipelines{}).injectHook() == nil {
		t.Error("the normal build supplies no injector: ctrl+d cannot test an alert (D-152)")
	}
	if len(debugScenarios()) == 0 {
		t.Error("the normal build offers no scenarios, so ctrl+d has no question to ask (D-152)")
	}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".go") && strings.Contains(readSource(t, e.Name()), "watchpost_"+"debug") { // split, so this file does not name it
			t.Errorf("%s names the retired debug tag: one build (D-152)", e.Name())
		}
	}

	// THE SEAM IS WHERE A REAL ALERT ENTERS. An injector that shortcut the
	// pipeline would validate the machinery and say nothing about the wiring —
	// the defect this tool exists to make findable.
	tick := readSource(t, "ticker.go")
	at := strings.Index(tick, "t.takeInjected()")
	active := strings.Index(tick, "globalfeed.Active(events, now)")
	if at < 0 || active < 0 {
		t.Fatal("the tick must drain the queue and then filter the active window")
	}
	if at > active {
		t.Error("injection must happen BEFORE the active-window filter, or it skips the stages a real alert crosses")
	}
}

// readSource reads a file of this package. The rule under test is about what
// the SOURCE contains, so the source is what it reads.
func readSource(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("reading %s: %v", name, err)
	}
	return string(b)
}
