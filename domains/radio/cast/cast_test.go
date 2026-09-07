package cast

import (
	"slices"
	"testing"
)

// --- Task 1.3: the registry ---

func TestRegistryIsWellFormed(t *testing.T) {
	if err := checkRegistry(); err != nil {
		t.Fatalf("the registry is malformed: %v", err)
	}
}

func TestEveryRoleHasItsKeyLabelAndParent(t *testing.T) {
	for _, tc := range []struct {
		role   Role
		key    string
		label  string
		parent Role
	}{
		{All, "voice", "All reads", All},
		{Alerts, "alerts", "Alerts & Notification reads", All},
		{Breaking, "breaking", "Breaking takeover", Alerts},
		{SevereRead, "severe_read", "Severe-event read", Alerts},
		{Standard, "standard", "Standard reports", All},
		{Weather, "weather", "Location weather", Standard},
		{Maritime, "maritime", "Location marine", Standard},
		{Fire, "fire", "Location fire & hotspots", Standard},
		{Seismic, "seismic", "Location seismic", Standard},
		{Station, "station", "Station", Standard},
	} {
		t.Run(tc.key, func(t *testing.T) {
			if got := tc.role.Key(); got != tc.key {
				t.Errorf("Key() = %q, want %q", got, tc.key)
			}
			if got := tc.role.String(); got != tc.label {
				t.Errorf("String() = %q, want %q", got, tc.label)
			}
			if got := tc.role.Parent(); got != tc.parent {
				t.Errorf("Parent() = %v, want %v", got, tc.parent)
			}
			if r, ok := RoleByKey(tc.key); !ok || r != tc.role {
				t.Errorf("RoleByKey(%q) = %v, %v; want %v, true", tc.key, r, ok, tc.role)
			}
		})
	}
}

// The root's parent is itself: that is what terminates the resolution walk, and
// a walk that did not terminate would hang the broadcast rather than fall back.
func TestTheRootIsItsOwnParentSoTheWalkTerminates(t *testing.T) {
	if All.Parent() != All {
		t.Fatal("All.Parent() must be All — the walk needs a fixed point")
	}
	if !All.IsRoot() || Alerts.IsRoot() {
		t.Fatal("IsRoot names exactly the root")
	}
	// Every role reaches the root within the tree's depth.
	for _, r := range Assignable() {
		steps := 0
		for cur := r; !cur.IsRoot(); cur = cur.Parent() {
			steps++
			if steps > int(numRoles) {
				t.Fatalf("walking up from %v does not reach the root", r)
			}
		}
	}
}

func TestAssignableIsEveryRoleBelowTheRootInDrawOrder(t *testing.T) {
	want := []Role{Alerts, Breaking, SevereRead, Standard, Weather, Maritime, Fire, Seismic, Station}
	got := Assignable()
	if !slices.Equal(got, want) {
		t.Fatalf("Assignable() = %v, want %v", got, want)
	}
	if len(got) != int(numRoles)-1 {
		t.Fatalf("Assignable() has %d roles, the registry has %d below the root", len(got), int(numRoles)-1)
	}
	if slices.Contains(got, All) {
		t.Error("the root is assigned by the `voice` key, not by a role table — it must not be listed")
	}
	// Mutating the returned slice must not reach the registry.
	got[0] = Station
	if Assignable()[0] != Alerts {
		t.Error("Assignable() must return a fresh slice")
	}
}

// An out-of-range Role arrives from a config file, a key press or a Segment.
// It must read as empty, never panic — a hand-edited file cannot crash the app.
func TestOutOfRangeRolesReadEmptyAndNeverPanic(t *testing.T) {
	for _, r := range []Role{-1, -99, numRoles, numRoles + 1, 1 << 20} {
		if got := r.Key(); got != "" {
			t.Errorf("Role(%d).Key() = %q, want empty", r, got)
		}
		if got := r.String(); got != "" {
			t.Errorf("Role(%d).String() = %q, want empty", r, got)
		}
		if got := r.Parent(); got != All {
			t.Errorf("Role(%d).Parent() = %v, want All so a walk from garbage terminates", r, got)
		}
	}
	if r, ok := RoleByKey("not_a_role"); ok || r != All {
		t.Errorf("RoleByKey(unknown) = %v, %v; an unknown key must never be minted as a role", r, ok)
	}
	if r, ok := RoleByKey(""); ok || r != All {
		t.Errorf("RoleByKey(\"\") = %v, %v; want All, false", r, ok)
	}
}

// --- Task 1.3: Config ---

func TestZeroConfigIsTodaysStation(t *testing.T) {
	var c Config
	if c.CastOn() {
		t.Error("the zero Config is Single Voice — the cast is off")
	}
	for _, r := range Assignable() {
		if got := c.PairFor(r); !got.Empty() {
			t.Errorf("PairFor(%v) = %+v on a zero Config, want the zero pair", r, got)
		}
	}
	if Muted(ClassWarning, c.Tones) {
		t.Error("the zero Config hears every class")
	}
}

func TestPairsAreOnlyReadWhenTheCastIsOn(t *testing.T) {
	c := Config{Root: "Samantha", Pairs: map[Role]Pair{Alerts: {MacOS: "Rishi"}}}
	if got := c.PairFor(Alerts); !got.Empty() {
		t.Errorf("PairFor with the cast OFF = %+v, want the zero pair — the pairs are kept but unread (MVS-D-25)", got)
	}
	c.Mode = ModeCast
	if got := c.PairFor(Alerts); got.MacOS != "Rishi" {
		t.Errorf("PairFor with the cast ON = %+v, want the assigned pair", got)
	}
	// Switching back keeps the assignment for the next switch.
	c.Mode = ModeSingle
	if len(c.Pairs) != 1 {
		t.Error("switching to Single Voice must keep the pairs, not clear them")
	}
	if got := c.PairFor(numRoles + 5); !got.Empty() {
		t.Error("an out-of-range role reads the zero pair")
	}
}

func TestWithPairCopiesAndNeverMintsTheRoot(t *testing.T) {
	base := Config{Mode: ModeCast, Pairs: map[Role]Pair{Alerts: {MacOS: "Rishi"}}}
	next := base.WithPair(Fire, Pair{Piper: "en_US-ryan-medium"})
	if got := base.PairFor(Fire); !got.Empty() {
		t.Error("WithPair mutated the receiver's map — a stored Config must not change under a caller")
	}
	if got := next.PairFor(Fire); got.Piper != "en_US-ryan-medium" {
		t.Errorf("WithPair did not assign: %+v", got)
	}
	// An empty pair clears the assignment rather than storing a blank one.
	cleared := next.WithPair(Fire, Pair{})
	if _, ok := cleared.Pairs[Fire]; ok {
		t.Error("assigning an empty pair must delete the entry, not store a blank")
	}
	// The root and out-of-range roles are ignored, never minted.
	for _, r := range []Role{All, -1, numRoles} {
		if got := base.WithPair(r, Pair{MacOS: "X"}); len(got.Pairs) != len(base.Pairs) {
			t.Errorf("WithPair(%v, …) minted an entry", r)
		}
	}
}

func TestCloneIsDeep(t *testing.T) {
	c := Config{Mode: ModeCast, Pairs: map[Role]Pair{Alerts: {MacOS: "Rishi"}},
		Tones: Tones{Mode: ModeTonesMute, Muted: []string{"watch"}}}
	clone := c.Clone()
	clone.Pairs[Alerts] = Pair{MacOS: "Someone else"}
	clone.Tones.Muted[0] = "advisory"
	if c.PairFor(Alerts).MacOS != "Rishi" {
		t.Error("Clone shares the pairs map")
	}
	if c.Tones.Muted[0] != "watch" {
		t.Error("Clone shares the muted slice")
	}
	// A nil map and a nil slice clone without allocating one.
	if got := (Config{}).Clone(); got.Pairs != nil || got.Tones.Muted != nil {
		t.Error("cloning a zero Config must not materialise its reference fields")
	}
}

// --- Task 1.6: the classifier, the presets and the mute rule ---

func TestClassifyFollowsTheRuleOfRecord(t *testing.T) {
	for _, tc := range []struct {
		product string
		want    Class
	}{
		// disaster: quakes and tsunami products, ahead of the suffix rules —
		// a Tsunami Warning is a disaster, not a warning.
		{"Tsunami Warning", ClassDisaster},
		{"Tsunami Advisory", ClassDisaster},
		{"Significant Earthquake", ClassDisaster},
		{"M 7.1 Earthquake", ClassDisaster},

		// storm wins over warning and watch (MVS-D-15), and Blizzard sits here
		// (RAT-3, ratified MVS-D-35).
		{"Hurricane Warning", ClassStorm},
		{"Hurricane Watch", ClassStorm},
		{"Tropical Storm Warning", ClassStorm},
		{"Winter Storm Warning", ClassStorm},
		{"Winter Storm Watch", ClassStorm},
		{"Blizzard Warning", ClassStorm},
		{"Storm Surge Warning", ClassStorm},
		{"Typhoon Warning", ClassStorm},

		// the suffix rules
		{"Special Weather Statement", ClassStatement},
		{"Heat Advisory", ClassAdvisory},
		{"Winter Weather Advisory", ClassAdvisory},
		{"Wind Advisory", ClassAdvisory},
		{"Tornado Watch", ClassWatch},
		{"Severe Thunderstorm Watch", ClassWatch},
		{"Tornado Warning", ClassWarning},
		{"Severe Thunderstorm Warning", ClassWarning},
		{"Flash Flood Warning", ClassWarning},

		// the loud default: anything unmatched, including nothing at all
		{"Air Quality Alert", ClassWarning},
		{"Beach Hazards Statement ", ClassStatement}, // trimmed before matching
		{"", ClassWarning},
		{"   ", ClassWarning},
		{"something nobody has ever issued", ClassWarning},

		// case does not matter: products arrive upper-cased from raw text
		{"TORNADO WARNING", ClassWarning},
		{"hurricane warning", ClassStorm},
	} {
		t.Run(tc.product, func(t *testing.T) {
			if got := Classify(tc.product); got != tc.want {
				t.Errorf("Classify(%q) = %v (%s), want %v (%s)", tc.product, got, got.Key(), tc.want, tc.want.Key())
			}
		})
	}
}

func TestToneNameNamesThePresetAndIsLoudOutOfRange(t *testing.T) {
	for _, tc := range []struct {
		class  Class
		preset string
	}{
		{ClassDisaster, PresetDualTone},
		{ClassWarning, PresetDualTone}, // disaster and warning share the loudest
		{ClassWatch, PresetTone1050},
		{ClassAdvisory, PresetClassic},
		{ClassStatement, PresetSoftChime},
		{ClassStorm, PresetLowSweep},
	} {
		if got := ToneName(tc.class); got != tc.preset {
			t.Errorf("ToneName(%s) = %q, want %q", tc.class.Key(), got, tc.preset)
		}
	}
	for _, c := range []Class{-1, numClasses, 1 << 20} {
		if got := ToneName(c); got != PresetDualTone {
			t.Errorf("ToneName(%d) = %q, want the loudest preset — never silence", c, got)
		}
		if c.Key() != "warning" || c.String() != "Warnings" {
			t.Errorf("an out-of-range class must still be nameable, got %q/%q", c.Key(), c.String())
		}
		// EVERY ACCESSOR GIVES THE SAME FALLBACK, and ToneRank joins the row it
		// was added beside rather than answering the question differently. It
		// returned Disaster's rank once — inaudible only because Warning and
		// Disaster share the dual-tone, which is the accidental agreement
		// MVS-D-73 was written to stop relying on.
		if got := c.ToneRank(); got != ClassWarning.ToneRank() {
			t.Errorf("Class(%d).ToneRank() = %d, want ClassWarning's %d — one fallback, not two",
				c, got, ClassWarning.ToneRank())
		}
	}
}

func TestClassKeysIsTheOneOwnerOfTheConfigWords(t *testing.T) {
	want := []string{"disaster", "warning", "watch", "advisory", "statement", "storm"}
	if got := ClassKeys(); !slices.Equal(got, want) {
		t.Fatalf("ClassKeys() = %v, want %v", got, want)
	}
	if len(Classes()) != len(ClassKeys()) {
		t.Fatal("Classes() and ClassKeys() must stay in step")
	}
	for i, c := range Classes() {
		if c.Key() != ClassKeys()[i] {
			t.Errorf("Classes()[%d].Key() = %q, ClassKeys()[%d] = %q", i, c.Key(), i, ClassKeys()[i])
		}
		if r, ok := ClassByKey(c.Key()); !ok || r != c {
			t.Errorf("ClassByKey(%q) = %v, %v; want %v, true", c.Key(), r, ok, c)
		}
	}
	if c, ok := ClassByKey("not_a_class"); ok || c != ClassWarning {
		t.Errorf("ClassByKey(unknown) = %v, %v; an unknown key is never minted", c, ok)
	}
}

func TestMutedTreatsAnEmptySetAsEveryClass(t *testing.T) {
	// Tones on: nothing is muted, whatever the set says.
	on := Tones{Mode: ModeTonesOn, Muted: []string{"warning", "watch"}}
	for _, c := range Classes() {
		if Muted(c, on) {
			t.Errorf("%s is muted with the mode off — the set survives the flip but does not act", c.Key())
		}
	}

	// Mute with nothing ticked means EVERY class: the screen shows "Mute:"
	// with an empty list, and what the screen says is what the ear gets.
	for _, empty := range []Tones{{Mode: ModeTonesMute}, {Mode: ModeTonesMute, Muted: []string{}}} {
		for _, c := range Classes() {
			if !Muted(c, empty) {
				t.Errorf("%s sounds under Mute with nothing ticked — an empty set means every class", c.Key())
			}
		}
	}

	// Mute with a set: exactly those classes.
	some := Tones{Mode: ModeTonesMute, Muted: []string{"watch", "advisory"}}
	for _, c := range Classes() {
		want := c == ClassWatch || c == ClassAdvisory
		if got := Muted(c, some); got != want {
			t.Errorf("Muted(%s) = %v, want %v", c.Key(), got, want)
		}
	}
	// An unknown key in the set mutes nothing extra and is not an error here —
	// Validate reports it once, with a count.
	junk := Tones{Mode: ModeTonesMute, Muted: []string{"not_a_class"}}
	for _, c := range Classes() {
		if Muted(c, junk) {
			t.Errorf("%s is muted by an unknown key", c.Key())
		}
	}
}

// Config.Clone is the one deep-copy owner: the muted-class slice is copied
// there, not by a method on Tones (a Clone calling a Clone reads as recursion
// to the safety gate, and one owner is clearer regardless).
func TestConfigCloneCopiesTheMutedSet(t *testing.T) {
	c := Config{Tones: Tones{Mode: ModeTonesMute, Muted: []string{"watch"}}}
	clone := c.Clone()
	clone.Tones.Muted[0] = "storm"
	if c.Tones.Muted[0] != "watch" {
		t.Error("Clone shares the muted slice")
	}
	if got := (Config{}).Clone(); got.Tones.Muted != nil {
		t.Error("cloning zero Tones must not materialise the slice")
	}
}
