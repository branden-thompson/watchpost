package app

import (
	"testing"

	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"github.com/branden-thompson/watchpost/platform/config"
)

// Reconciling a stored cast against the Settings window (HUM LEAD, UAT
// 2026-08-30).
//
// A file can name roles the window has no row for. The one that bit was
// `standard`: the parent of the four report roles AND of the station.

// anyHost resolves everything, so these tests measure the TREE rather than a
// machine's voice list.
type anyHost struct{}

func (anyHost) Platform() string       { return cast.PlatformDarwin }
func (anyHost) Discovered(string) bool { return true }
func (anyHost) Installed(string) bool  { return true }
func (anyHost) Default() string        { return "System Voice" }

// The reported bug, from the config that produced it: alerts and weather set to
// System Voice, and a leftover `standard = Daniel` that no row could show or
// clear. Maritime, fire, seismic AND the station's lead read in Daniel while
// every one of those rows said System Voice.
func TestAStoredGroupVoiceIsFoldedIntoTheRowsThatShowIt(t *testing.T) {
	cfg := config.Default()
	cfg.Voice = "System Voice"
	cfg.Radio.Cast = "cast"
	cfg.Radio.Voices.Alerts = config.RoleVoice{MacOS: "System Voice"}
	cfg.Radio.Voices.Standard = config.RoleVoice{MacOS: "Daniel"}
	cfg.Radio.Voices.Weather = config.RoleVoice{MacOS: "System Voice"}

	c := castLoaded(cfg)

	// Nothing a listener HEARS for a drawn row changes: the four reports keep
	// the voice the group was giving them.
	for _, tc := range []struct {
		role cast.Role
		want string
	}{
		{cast.Alerts, "System Voice"},
		{cast.Weather, "System Voice"},
		{cast.Maritime, "Daniel"},
		{cast.Fire, "Daniel"},
		{cast.Seismic, "Daniel"},
	} {
		if got := cast.Resolve(tc.role, c, anyHost{}); got.Spoken != tc.want {
			t.Errorf("%s reads in %q, want %q", tc.role.Key(), got.Spoken, tc.want)
		}
	}
	// And the group itself is GONE, so it can never override a row invisibly
	// again. The station goes with it and follows the root — which is what its
	// row would have said if it had one, and what the listener was already being
	// shown.
	for _, r := range []cast.Role{cast.Standard, cast.Breaking, cast.SevereRead, cast.Station} {
		if !c.Pairs[r].Empty() {
			t.Errorf("%s has no row in Settings; it must not survive reconciliation, got %+v", r.Key(), c.Pairs[r])
		}
	}
	if got := cast.Resolve(cast.Station, c, anyHost{}); got.Spoken != "System Voice" {
		t.Errorf("the station's lead follows the root, got %q", got.Spoken)
	}
}

// THE INVARIANT, and the reason the bug was invisible: every row must SAY what
// it will sound. The row shows its own name when it has one and the root when it
// does not (pickerName), while the broadcast walks the whole tree — so any role
// between a row and the root is a chance for the two to disagree.
func TestEveryDrawnRowSaysWhatItWillSound(t *testing.T) {
	// PIN THE PLATFORM, because this test compares two things that resolve on
	// one. The fixtures set only MacOS names and anyHost reports darwin, while
	// castView goes through halfFor, which reads the RUNNING platform's half.
	// On a Mac the two agree and the test passes; on Linux halfFor reads the
	// empty Piper half, the row shows the inherited root, and every row is
	// reported as a mismatch. It asserted a platform-specific property without
	// ever saying which platform — so it only held on the machine it was
	// written on, and the first Linux run said so.
	asPlatform(t, "darwin")
	for _, tc := range []struct {
		name string
		set  func(*config.Config)
	}{
		{"nothing assigned", func(*config.Config) {}},
		{"a group voice", func(c *config.Config) { c.Radio.Voices.Standard = config.RoleVoice{MacOS: "Daniel"} }},
		{"a group and one child", func(c *config.Config) {
			c.Radio.Voices.Standard = config.RoleVoice{MacOS: "Daniel"}
			c.Radio.Voices.Fire = config.RoleVoice{MacOS: "Karen"}
		}},
		{"an alerts child", func(c *config.Config) {
			c.Radio.Voices.Alerts = config.RoleVoice{MacOS: "Karen"}
			c.Radio.Voices.Breaking = config.RoleVoice{MacOS: "Daniel"}
		}},
		{"every undrawn role at once", func(c *config.Config) {
			c.Radio.Voices.Standard = config.RoleVoice{MacOS: "Daniel"}
			c.Radio.Voices.Breaking = config.RoleVoice{MacOS: "Karen"}
			c.Radio.Voices.SevereRead = config.RoleVoice{MacOS: "Samantha"}
			c.Radio.Voices.Station = config.RoleVoice{MacOS: "Rishi"}
		}},
	} {
		cfg := config.Default()
		cfg.Voice = "System Voice"
		tc.set(&cfg)
		c := castLoaded(cfg)
		view := castView(c)
		for _, r := range settingsRoles() {
			// What the ROW shows: its own name, or the root it inherits.
			shown := view.Names[r.Key()]
			if shown == "" {
				shown = view.Names[cast.All.Key()]
			}
			if got := cast.Resolve(r, c, anyHost{}); got.Spoken != shown {
				t.Errorf("%s: the %s row shows %q and sounds %q", tc.name, r.Key(), shown, got.Spoken)
			}
		}
	}
}

// Reconciliation must SURVIVE a save: the window writes what it holds, and what
// it holds is the reconciled cast, so the leftover cannot come back.
func TestReconciliationSurvivesASave(t *testing.T) {
	cfg := config.Default()
	cfg.Voice = "System Voice"
	cfg.Radio.Voices.Standard = config.RoleVoice{MacOS: "Daniel"}

	saved := castToConfig(castFromView(castView(castLoaded(cfg)), castLoaded(cfg)), cfg)
	if saved.Radio.Voices.Standard.MacOS != "" {
		t.Errorf("the group voice is written back out: %+v", saved.Radio.Voices.Standard)
	}
	for _, got := range []struct {
		role string
		rv   config.RoleVoice
	}{
		{"maritime", saved.Radio.Voices.Maritime},
		{"fire", saved.Radio.Voices.Fire},
		{"seismic", saved.Radio.Voices.Seismic},
	} {
		if got.rv.MacOS != "Daniel" {
			t.Errorf("%s keeps the voice it was sounding, got %+v", got.role, got.rv)
		}
	}
}
