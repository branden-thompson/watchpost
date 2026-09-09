package cast

import (
	"slices"
	"strings"

	"github.com/branden-thompson/watchpost/platform/category"
)

// The alert tone's class: what a listener hears BEFORE the words, saying which
// kind of alert is coming (FR-11, tones.md §2). Six classes over five presets —
// disaster and warning share the loudest, because there is no useful moment in
// which a listener needs to tell those two apart by ear before the words.
//
// Class is a plain int and arrives from a config file and from product strings,
// so every accessor reads an out-of-range value as ClassWarning rather than
// panicking or falling silent: an unknown alert is the one you least want to
// under-announce, and Warning is this package's loud default — it carries the
// loudest PRESET, which is what a listener actually hears.
//
// "the loudest class" is the older wording and it was wrong once ToneRank
// existed: Warning does not hold the highest RANK, Disaster does, and the two
// only sound alike because they share the dual-tone. One fallback, said one way.
type Class int

const (
	// ClassWarning is the zero value AND the default for any product string no
	// rule matches — the loud default (tones.md §2).
	ClassWarning Class = iota
	ClassDisaster
	ClassWatch
	ClassAdvisory
	ClassStatement
	ClassStorm
	// ClassEmergency is the Weather Service's highest-urgency product — leave,
	// now. It reuses the dual tone and is told apart by COUNT: three, where a
	// warning is one (#18, HUM LEAD 2026-09-07). Repetition is the one
	// dimension this taxonomy did not use, so it costs nothing to design, tune,
	// or teach a listener.
	ClassEmergency

	numClasses
)

// Preset names, as domains/radio/synth's PresetByName resolves them (P3 Task
// 3.6). They are identifiers, not labels: the Setup window draws the class, not
// the preset.
const (
	PresetDualTone  = "dual-tone"
	PresetTone1050  = "1050hz"
	PresetClassic   = "classic"
	PresetSoftChime = "soft-chime"
	PresetLowSweep  = "low-sweep"
)

// class is one row of the class registry: the config word, the Setup label and
// the preset it always sounds. A class's preset is not configurable — MVS-D-26
// dropped the "share the Warnings tone" switch; a class is either muted or it
// sounds its ratified preset.
type class struct {
	toneRank int // how much attention this class's tone demands; higher is more severe
	repeats  int // how many times the preset sounds; 0 means once
	key      string
	label    string
	preset   string
}

var classes = [numClasses]class{
	ClassWarning:   {key: "warning", label: "Warnings", preset: PresetDualTone, toneRank: 5},
	ClassDisaster:  {key: "disaster", label: "Disaster Events", preset: PresetDualTone, toneRank: 6},
	ClassWatch:     {key: "watch", label: "Watches", preset: PresetTone1050, toneRank: 4},
	ClassAdvisory:  {key: "advisory", label: "Advisories", preset: PresetClassic, toneRank: 3},
	ClassStatement: {key: "statement", label: "Special Statements", preset: PresetSoftChime, toneRank: 2},
	ClassStorm:     {key: "storm", label: "Marine", preset: PresetLowSweep, toneRank: 1},
	// THREE, and the count is the signal. Same preset as a warning; a listener
	// with no screen hears how many and knows before a word is spoken.
	ClassEmergency: {key: "emergency", label: "Emergency Orders", preset: PresetDualTone, toneRank: 7, repeats: 3},
}

// toneClassOf is the ONE declared mapping from a hazard category to the tone it
// sounds. #18 happened because category.Emergency was added at C-2 and this
// relationship existed nowhere — the feed, the window and the read ladder were
// all taught, and the tone was not.
//
// A function, not a global, per the codebase's table convention (P10-06).
// A category absent here fails TestEveryCategoryThatReachesAReadHasATone rather
// than falling through to ClassWarning, which is what made the omission silent.
func toneClassOf() map[category.Category]Class {
	return map[category.Category]Class{
		category.Emergency:  ClassEmergency,
		category.Warnings:   ClassWarning,
		category.Watches:    ClassWatch,
		category.Advisories: ClassAdvisory,
		category.Statements: ClassStatement,
		category.Disasters:  ClassDisaster,
		category.Marine:     ClassStorm,
		// category.Forecasts is deliberately absent: a forecast is not an alert
		// and never reaches the read ladder. The test carries that reason too.
	}
}

// ClassFor is the tone class a hazard category sounds; ok is false for a
// category no read can carry.
//
// PREFER THIS OVER Classify WHERE A CATEGORY IS IN HAND. Classify reads a
// product's own words, which is all the breaking path has; a caller holding a
// category already knows the answer more precisely, and the civil-emergency
// family is exactly the case where the words do not carry it.
func ClassFor(c category.Category) (Class, bool) {
	cls, ok := toneClassOf()[c]
	return cls, ok
}

// ToneRepeats is how many times c's preset sounds. One, unless the class earns
// more — today only ClassEmergency does.
func ToneRepeats(c Class) int {
	if !c.valid() || classes[c].repeats == 0 {
		return 1
	}
	return classes[c].repeats
}

// ToneRank is how much attention this class's tone demands — HIGHER IS MORE
// SEVERE (MVS-D-73). It exists so a burst carrying two hazards of equal severity
// sounds the more serious of their two tones instead of whichever the schedule
// happened to put first.
//
// IT IS NOT THE READ ORDER AND NOT THE SETUP ORDER. `Classes()` is the order
// Setup draws its checkboxes; `category.ReadRank` is what gets read first, which
// the HUM LEAD ruled is about PROXIMITY as much as hazard (MVS-D-71). This is a
// third question — which SOUND says "listen now" — and it gets its own carrier
// rather than borrowing one that means something else.
//
// The order follows the ratified presets (tones.md §1), loudest first: the
// dual-tone EAS signal, then NOAA's 1050 Hz alarm, the classic pulses, the soft
// chime, and the low sweep. **Disaster and Warning share the dual-tone**, so the
// gap between their ranks is INAUDIBLE — it exists only so the choice is
// deterministic rather than positional, which is the whole point of the rule.
//
// An out-of-range class reads as ClassWarning, which is what `Key`, `String` and
// `ToneName` all return — an unknown alert is the one you least want to
// under-announce, and Warning is this package's loud default.
//
// IT RETURNED DISASTER'S RANK ONCE, and the difference was inaudible only
// because Warning and Disaster share the dual-tone — the accidental coupling
// MVS-D-73 exists to remove, reintroduced in the fallback. Two answers for one
// question, agreeing by accident, is the shape D-1 is about.
func (c Class) ToneRank() int {
	if !c.valid() {
		return classes[ClassWarning].toneRank
	}
	return classes[c].toneRank
}

func (c Class) valid() bool { return c >= 0 && c < numClasses }

// Key is the config word for c. An out-of-range class reads as the loud
// default's key, so a corrupt value is still nameable.
func (c Class) Key() string {
	if !c.valid() {
		return classes[ClassWarning].key
	}
	return classes[c].key
}

// String is the Setup label for c (MVS-D-28 as drawn; the storm class keeps the
// mock's word "Maritime" — MVS-D-33).
func (c Class) String() string {
	if !c.valid() {
		return classes[ClassWarning].label
	}
	return classes[c].label
}

// ToneName is the preset c always sounds. Out of range returns the loudest,
// never a panic and never silence.
func ToneName(c Class) string {
	if !c.valid() {
		return classes[ClassWarning].preset
	}
	return classes[c].preset
}

// ClassKeys is the one owner of the config words, in the order Setup draws the
// checkboxes and [S] lists them. A fresh slice: the registry is not writable
// through a caller.
func ClassKeys() []string {
	return []string{
		ClassDisaster.Key(), ClassWarning.Key(), ClassWatch.Key(),
		ClassAdvisory.Key(), ClassStatement.Key(), ClassStorm.Key(),
	}
}

// Classes is every class in the same order as ClassKeys.
func Classes() []Class {
	return []Class{ClassDisaster, ClassWarning, ClassWatch, ClassAdvisory, ClassStatement, ClassStorm}
}

// ClassByKey resolves a config word. An unknown word is NOT minted as a class:
// it is reported once, with a count, by Validate (NFR-5).
func ClassByKey(key string) (Class, bool) {
	for c := range numClasses {
		if classes[c].key == key {
			return c, true
		}
	}
	return ClassWarning, false
}

// classRule is one row of the rule of record (tones.md §2), evaluated in order.
// A rule matches when the product contains any of `contains`, or ends with
// `suffix` — both lowercased before the comparison.
type classRule struct {
	class    Class
	contains []string
	suffix   string
}

// rules IS the classification rule. Order is precedence, and the order is the
// ruling: disaster products first (a Tsunami Warning is a disaster, not a
// warning), then storm over everything that would otherwise read as a warning
// or a watch (MVS-D-15), then the suffix rules from most specific to least.
// Anything unmatched falls through to ClassWarning — the loud default.
var rules = []classRule{
	{class: ClassDisaster, contains: []string{"tsunami", "earthquake", "significant quake"}},
	{class: ClassStorm, contains: []string{"hurricane", "tropical storm", "tropical depression", "typhoon", "winter storm", "blizzard", "storm surge"}},
	{class: ClassStatement, suffix: "statement"},
	{class: ClassAdvisory, suffix: "advisory"},
	{class: ClassWatch, suffix: "watch"},
	{class: ClassWarning, suffix: "warning"},
}

// Classify reads a product string — an NWS product type, an NHC name, the
// ticker's quake class — and returns the class whose tone it sounds. It must
// serve both the breaking path (which holds a globalfeed.Event) and the severe
// read (which holds a tty.SevereRow), so it takes the one thing both have: the
// product's own words (voice-architecture.md A-3.2).
//
// An empty or unrecognised product is a WARNING, not silence: an alert nobody
// classified is the last one to announce quietly.
func Classify(product string) Class {
	p := strings.TrimSpace(product)
	if p == "" {
		return ClassWarning
	}
	// Matched case-insensitively WITHOUT lowering the input first. Lowering
	// allocates, and this runs on the path a listener is waiting on — the one
	// M4 measures. The rules' own words are already lower-case, so folding at
	// the comparison costs nothing and allocates nothing.
	for _, r := range rules {
		if r.suffix != "" && hasSuffixFold(p, r.suffix) {
			return r.class
		}
		if containsAnyFold(p, r.contains) {
			return r.class
		}
	}
	return ClassWarning
}

// hasSuffixFold is strings.HasSuffix, case-insensitively and without copying.
func hasSuffixFold(s, suffix string) bool {
	return len(s) >= len(suffix) && strings.EqualFold(s[len(s)-len(suffix):], suffix)
}

// containsAnyFold reports whether s contains any of subs, case-insensitively.
//
// A window scan rather than a lowered copy: products are a few words and the
// needles a few characters, so the quadratic worst case is nothing next to an
// allocation, and there is no garbage left on the alert path.
func containsAnyFold(s string, subs []string) bool {
	for _, sub := range subs {
		if containsFold(s, sub) {
			return true
		}
	}
	return false
}

func containsFold(s, sub string) bool {
	if len(sub) == 0 || len(s) < len(sub) {
		return len(sub) == 0
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if strings.EqualFold(s[i:i+len(sub)], sub) {
			return true
		}
	}
	return false
}

// Tones is the per-class mute state (MVS-D-26). Every class always sounds its
// ratified preset; the only choice is whether it sounds at all. The zero value
// is "All Tones On" — a fresh install hears every class.
type Tones struct {
	// Mode is ModeTonesOn or ModeTonesMute. [M] flips it and persists it; the
	// muted set survives the flip, so turning tones back off restores exactly
	// the classes the listener had ticked (the mock's "All Tones On (Default) /
	// Mute:").
	Mode string

	// Muted holds class KEYS, not labels. An unknown key is ignored and
	// reported once with a count by Validate.
	Muted []string
}

// Tone modes.
const (
	// ModeTonesOn is the zero value: every class sounds.
	ModeTonesOn = ""
	// ModeTonesMute silences the classes named in Muted — or every class when
	// Muted is empty.
	ModeTonesMute = "mute"
)

// Muted reports whether c's tone is silent under t.
//
// The rule has one non-obvious half, and it is deliberate: "Mute:" with NOTHING
// ticked means EVERY class. The Setup window shows a mute mode with an empty
// checkbox list, and what the screen says is what the ear must get — a listener
// who chose "Mute" and ticked nothing asked for silence, not for a no-op.
func Muted(c Class, t Tones) bool {
	if t.Mode != ModeTonesMute {
		return false
	}
	if len(t.Muted) == 0 {
		return true
	}
	return slices.Contains(t.Muted, c.Key())
}
