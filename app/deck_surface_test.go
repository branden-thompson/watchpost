package app

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"github.com/branden-thompson/watchpost/modes/tty"
	"github.com/branden-thompson/watchpost/platform/declset"
)

// deckSurface is the deck's exported surface on its unexported types, tiers
// 2 and 3 of design AP-1.5, by receiver: what the deck hands out through an
// interface or a method value (tier 2), and what is exported but used only
// inside the deck (tier 3). The deck's prep commit adds tier 1, the
// package-level names, when the deck is a package.
var deckSurface = map[string][]string{
	// tier 2: tty.Radio, the six method values Dress hands tty.Dashboard's
	// With… setters, cast.Host; tier 3: the three with test users only
	"radioDeck": {"Default", "Discovered", "Installed", "InstallsRemaining", "Platform", "PreviewVoice", "Problems", "Resolutions", "SetMode", "SetRepeat", "SetVoice", "SetVolume", "Spectrum", "Stop", "Tune", "VoiceName", "Voices"},
	// tier 2: tty.Station; tier 3: the six the pool calls, which move into the deck
	"mastercontrol": {"CutBed", "DropCard", "GoOnAir", "GoToStandby", "HandAir", "MoveCard", "Refence", "RequestCard", "StopMonitor"},
	"hostFacts":     {"Default", "Discovered", "Installed", "Platform"}, // tier 2: cast.Host
	// tier 3
	"severeDeck":     {"AlertKeysWithin", "LaneRows", "Row", "SetFeed", "SetLocations"},
	"eventReader":    {"Cancel", "End", "Read", "Toggle"},
	"tickerDeck":     {"Inject"},
	"arbiter":        {"Run"},
	"bedReach":       {"Line"},
	"mainTrackStage": {"String"},
}

// outsideTheDeck are the other exported methods on unexported app types:
// interfaces met by types that stay at the root or go to the map data, which
// AP-1.5 does not count. Listed so a new one is seen.
var outsideTheDeck = map[string][]string{
	"answeredFor":       {"Fetch"},
	"byteCounter":       {"Write"},
	"countingTransport": {"RoundTrip"},
	"dumper":            {"Dump"},
	"publisher":         {"Trigger"},
	"releaseWatch":      {"Enabled", "Status"},
}

// TestTheDecksExportedSurfaceIsHeld is 0.19.6 B0.12 (design AP-1.5, tiers 2
// and 3): every exported method on an unexported type in app's non-test files
// is the deck's 43 or one of the seven outside it, exactly; and radioDeck's
// and mastercontrol's exported methods are no more than the interfaces they
// are handed out as and the names AP-1.5 lists beside them. Seen failing on a
// planted method.
func TestTheDecksExportedSurfaceIsHeld(t *testing.T) {
	decls, err := declset.Set(".")
	if err != nil {
		t.Fatal(err)
	}
	if got := surfaceProblems(append(slices.Clone(decls), "func (*radioDeck).Planted")); len(got) != 1 || !strings.Contains(got[0], "radioDeck.Planted") {
		t.Fatalf("COULD NOT RUN: a planted exported method was reported as %q", got)
	}
	for _, p := range surfaceProblems(decls) { // bounded by the findings (P10-02)
		t.Error(p)
	}
	beyond := func(recv string, allowed []string, ifaces ...reflect.Type) {
		for _, it := range ifaces { // bounded by the interfaces (P10-02)
			for i := range it.NumMethod() { // bounded by the interface (P10-02)
				allowed = append(allowed, it.Method(i).Name)
			}
		}
		for _, m := range deckSurface[recv] { // bounded by the receiver's methods (P10-02)
			if !slices.Contains(allowed, m) {
				t.Errorf("%s.%s is exported and is neither in the interfaces %s is handed out as nor among AP-1.5's names beside them", recv, m, recv)
			}
		}
	}
	beyond("radioDeck", []string{"Spectrum", "Voices", "VoiceName", "SetVoice", "PreviewVoice", "Installed", "InstallsRemaining", "Resolutions", "Problems"},
		reflect.TypeFor[tty.Radio](), reflect.TypeFor[cast.Host]())
	beyond("mastercontrol", []string{"HandAir", "MoveCard", "DropCard", "RequestCard", "Refence", "StopMonitor"}, reflect.TypeFor[tty.Station]())
}

// surfaceProblems is every way declset's lines differ from the held lists:
// an exported method on an unexported type that is not listed, and a listed
// one that is gone.
func surfaceProblems(decls []string) []string {
	got := map[string]bool{}
	for _, d := range decls { // bounded by the declarations (P10-02)
		recv, name, ok := method(d)
		if ok && !isExported(recv) && isExported(name) {
			got[recv+"."+name] = true
		}
	}
	want := map[string]bool{}
	for _, list := range []map[string][]string{deckSurface, outsideTheDeck} { // the two lists (P10-02)
		for recv, names := range list { // bounded by the list (P10-02)
			for _, n := range names { // bounded by the receiver (P10-02)
				want[recv+"."+n] = true
			}
		}
	}
	var out []string
	for _, k := range sortedKeys(got) { // bounded by the methods (P10-02)
		if !want[k] {
			out = append(out, k+" is an exported method on an unexported app type and is on no list: the deck's (AP-1.5) or outsideTheDeck")
		}
	}
	for _, k := range sortedKeys(want) { // bounded by the lists (P10-02)
		if !got[k] {
			out = append(out, k+" is listed and no longer declared: take it off its list")
		}
	}
	return out
}

// method reads declset's "func (Recv).Name" line, a pointer receiver's star
// dropped.
func method(d string) (recv, name string, ok bool) {
	rest, ok := strings.CutPrefix(d, "func (")
	if !ok {
		return "", "", false
	}
	recv, name, ok = strings.Cut(rest, ").")
	return strings.TrimPrefix(recv, "*"), name, ok
}

// isExported says whether a name starts with an upper-case letter.
func isExported(s string) bool { return s != "" && s[0] >= 'A' && s[0] <= 'Z' }

// sortedKeys is a set's members in order.
func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m { // bounded by the set (P10-02)
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}
