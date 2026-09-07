package app

import (
	"os"
	"strings"
	"testing"
)

// THE INJECTION SEAM IS ABSENT FROM A RELEASE BINARY (F-21b).
//
// A screenshot of a fabricated tornado warning is indistinguishable from a real
// one, so the capability is BUILD-TAGGED rather than runtime-gated: a runtime
// gate is one config mistake from being reachable; a build tag is not in the
// file the linker reads.
//
// This asserts the property structurally, because it cannot be asserted
// behaviourally — a release build has no Inject to call, so no test can call it
// and observe nothing happening. It reads the source the way lint-imports and
// lint-watermark do, which is the shape a rule about WHICH CODE EXISTS has to
// take (F-22).
func TestInjectionIsAbsentFromAReleaseBuild(t *testing.T) {
	debug, release := readSource(t, "inject_debug.go"), readSource(t, "inject_release.go")

	if !strings.HasPrefix(strings.TrimSpace(debug), "//go:build watchpost_debug") {
		t.Error("the injector must be behind the debug tag, or it ships")
	}
	if !strings.HasPrefix(strings.TrimSpace(release), "//go:build !watchpost_debug") {
		t.Error("the release file must exclude the debug tag, or both compile at once")
	}
	// The capability itself: only the debug file may define a way to put an
	// event into the queue.
	if !strings.Contains(debug, "func (t *tickerDeck) Inject(") {
		t.Error("the debug build defines Inject")
	}
	if strings.Contains(release, "func (t *tickerDeck) Inject(") {
		t.Error("a release build must define no Inject at all")
	}
	// And the call site is SHARED, so the two builds differ in what exists
	// rather than in what the tick does.
	if !strings.Contains(release, "func (t *tickerDeck) takeInjected()") ||
		!strings.Contains(debug, "func (t *tickerDeck) takeInjected()") {
		t.Error("both builds provide takeInjected, so the tick has one call site")
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
