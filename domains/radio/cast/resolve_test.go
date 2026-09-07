package cast

import (
	"slices"
	"testing"
)

// SystemVoice is the macOS sentinel: it always resolves, and the identity
// lines say "your correspondent" when it has no name (data-shape §4).
const SystemVoice = "System Voice"

// fakeHost is the whole machine, as resolution sees it. Every row of the M5
// matrix is this struct with different fields — no synthesizer, no subprocess,
// no filesystem.
type fakeHost struct {
	platform   string
	discovered []string // macOS: the closed allowlist AT THIS MOMENT
	installed  []string // Piper: what is on disk
	deflt      string   // the host's own last resort
}

func (h fakeHost) Platform() string { return h.platform }
func (h fakeHost) Default() string  { return h.deflt }

func (h fakeHost) Discovered(name string) bool {
	return name == SystemVoice || slices.Contains(h.discovered, name)
}

func (h fakeHost) Installed(key string) bool { return slices.Contains(h.installed, key) }

// mac is a Mac with two voices on its list and the sentinel as its default.
func mac(names ...string) fakeHost {
	return fakeHost{platform: PlatformDarwin, discovered: names, deflt: SystemVoice}
}

// linux is a Linux box with the given Piper keys on disk; its default is the
// first of them, or nothing at all when none are installed.
func linux(keys ...string) fakeHost {
	h := fakeHost{platform: "linux", installed: keys}
	if len(keys) > 0 {
		h.deflt = keys[0]
	}
	return h
}

// castOf builds a cast-mode Config from a root and role assignments.
func castOf(root string, pairs map[Role]Pair) Config {
	return Config{Root: root, Mode: ModeCast, Pairs: pairs}
}

// TestResolveFallbackMatrix is M5. Every row asserts Spoken, Link and Reason:
// a fallback that happens quietly is the failure this feature exists to
// prevent, so "it fell back" is never enough — [S] must be able to say why.
func TestResolveFallbackMatrix(t *testing.T) {
	for _, tc := range []struct {
		name   string
		role   Role
		cfg    Config
		host   Host
		spoken string
		link   Link
		reason string
	}{
		{
			name:   "assigned and discovered: the role's own voice speaks",
			role:   Breaking,
			cfg:    castOf("Samantha", map[Role]Pair{Breaking: {MacOS: "Rishi"}}),
			host:   mac("Samantha", "Rishi"),
			spoken: "Rishi", link: LinkRole, reason: ReasonAssigned,
		},
		{
			name:   "assigned and installed: the Piper key speaks",
			role:   Fire,
			cfg:    castOf("en_US-lessac-medium", map[Role]Pair{Fire: {Piper: "en_US-ryan-medium"}}),
			host:   linux("en_US-lessac-medium", "en_US-ryan-medium"),
			spoken: "en_US-ryan-medium", link: LinkRole, reason: ReasonAssigned,
		},
		{
			name:   "assigned but NOT installed: the ancestor speaks and says so",
			role:   Fire,
			cfg:    castOf("en_US-lessac-medium", map[Role]Pair{Fire: {Piper: "en_US-ryan-medium"}}),
			host:   linux("en_US-lessac-medium"),
			spoken: "en_US-lessac-medium", link: LinkRoot, reason: ReasonNotInstalled,
		},
		{
			name:   "assigned but UNKNOWN here: the ancestor speaks and says so",
			role:   Breaking,
			cfg:    castOf("Samantha", map[Role]Pair{Breaking: {MacOS: "Nobody"}}),
			host:   mac("Samantha"),
			spoken: "Samantha", link: LinkRoot, reason: ReasonUnknownHere,
		},
		{
			name: "assigned in the OTHER OS's table only: this host inherits",
			role: Fire,
			// A file written on a Mac, opened on Linux: the Piper half is empty.
			cfg:    castOf("en_US-lessac-medium", map[Role]Pair{Fire: {MacOS: "Rishi"}}),
			host:   linux("en_US-lessac-medium"),
			spoken: "en_US-lessac-medium", link: LinkRoot, reason: ReasonInherited,
		},
		{
			name: "not yet discovered: the curated list answers, with no trust window",
			role: Breaking,
			cfg:  castOf("Samantha", map[Role]Pair{Breaking: {MacOS: "Rishi"}}),
			// `say -v ?` has not answered; the curated list has Samantha but not
			// Rishi. Rishi does NOT get spoken on the hope that it lands
			// (D-R2-2) — it falls back and says why until the list catches up.
			host:   mac("Samantha"),
			spoken: "Samantha", link: LinkRoot, reason: ReasonUnknownHere,
		},
		{
			name:   "the sentinel always resolves",
			role:   Weather,
			cfg:    castOf("Samantha", map[Role]Pair{Weather: {MacOS: SystemVoice}}),
			host:   mac(), // an empty list: the sentinel still speaks
			spoken: SystemVoice, link: LinkRole, reason: ReasonAssigned,
		},
		{
			name:   "inherited from the GROUP",
			role:   Breaking,
			cfg:    castOf("Samantha", map[Role]Pair{Alerts: {MacOS: "Rishi"}}),
			host:   mac("Samantha", "Rishi"),
			spoken: "Rishi", link: LinkGroup, reason: ReasonInherited,
		},
		{
			name:   "inherited from the ROOT",
			role:   Seismic,
			cfg:    castOf("Samantha", nil),
			host:   mac("Samantha"),
			spoken: "Samantha", link: LinkRoot, reason: ReasonInherited,
		},
		{
			name:   "the root itself, assigned",
			role:   All,
			cfg:    castOf("Samantha", nil),
			host:   mac("Samantha"),
			spoken: "Samantha", link: LinkRole, reason: ReasonAssigned,
		},
		{
			// Weather named nothing, so from ITS point of view it inherits —
			// what failed is the root's name, and Link = default is what says
			// so. The row below asks the root itself and gets the "why".
			name:   "root unresolvable: the platform default speaks, the role still reads as inheriting",
			role:   Weather,
			cfg:    castOf("A Voice From Another Mac", nil),
			host:   mac("Samantha"),
			spoken: SystemVoice, link: LinkDefault, reason: ReasonInherited,
		},
		{
			name:   "root unresolvable, asked OF the root: the default speaks and says why",
			role:   All,
			cfg:    castOf("A Voice From Another Mac", nil),
			host:   mac("Samantha"),
			spoken: SystemVoice, link: LinkDefault, reason: ReasonUnknownHere,
		},
		{
			name:   "nothing named at all: the platform default speaks",
			role:   Weather,
			cfg:    Config{},
			host:   linux("en_US-lessac-medium"),
			spoken: "en_US-lessac-medium", link: LinkDefault, reason: ReasonInherited,
		},
		{
			name:   "Linux with nothing installed: the ONE legitimately silent row",
			role:   Breaking,
			cfg:    castOf("en_US-lessac-medium", map[Role]Pair{Breaking: {Piper: "en_US-ryan-medium"}}),
			host:   linux(), // nothing on disk, no default
			spoken: "", link: LinkNone, reason: ReasonNothingResolvable,
		},
		{
			name:   "the cast is OFF: an assignment is not read, the root speaks",
			role:   Breaking,
			cfg:    Config{Root: "Samantha", Pairs: map[Role]Pair{Breaking: {MacOS: "Rishi"}}},
			host:   mac("Samantha", "Rishi"),
			spoken: "Samantha", link: LinkRoot, reason: ReasonInherited,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := Resolve(tc.role, tc.cfg, tc.host)
			if got.Spoken != tc.spoken {
				t.Errorf("Spoken = %q, want %q", got.Spoken, tc.spoken)
			}
			if got.Link != tc.link {
				t.Errorf("Link = %s, want %s", got.Link, tc.link)
			}
			if got.Reason != tc.reason {
				t.Errorf("Reason = %q, want %q", got.Reason, tc.reason)
			}
			if got.Role != tc.role {
				t.Errorf("Role = %v, want %v", got.Role, tc.role)
			}
			if got.Silent() != (tc.spoken == "") {
				t.Errorf("Silent() = %v with Spoken %q", got.Silent(), got.Spoken)
			}
		})
	}
}

// Spoken is the only string that ever reaches a voice or the screen (RS-18):
// a Requested name that lost must never be mistaken for the one speaking.
func TestResolveKeepsRequestedAndSpokenApart(t *testing.T) {
	got := Resolve(Breaking, castOf("Samantha", map[Role]Pair{Breaking: {MacOS: "Nobody"}}), mac("Samantha"))
	if got.Requested != "Nobody" {
		t.Errorf("Requested = %q, want the name that lost", got.Requested)
	}
	if got.Spoken != "Samantha" {
		t.Errorf("Spoken = %q, want the name that won", got.Spoken)
	}
}

func TestResolveIsSafeWithNoHostAndOutOfRangeRoles(t *testing.T) {
	if got := Resolve(Breaking, castOf("Samantha", nil), nil); !got.Silent() || got.Link != LinkNone {
		t.Errorf("a nil host must resolve to silence, got %+v", got)
	}
	// A Role from a corrupt config walks to the root rather than off the table.
	for _, r := range []Role{-1, numRoles, 1 << 20} {
		got := Resolve(r, castOf("Samantha", nil), mac("Samantha"))
		if got.Spoken != "Samantha" {
			t.Errorf("Resolve(Role(%d)) = %q, want the root's voice", r, got.Spoken)
		}
	}
}

// Every role resolves on a host with a working root: no node of the tree can
// go silent while the root speaks.
func TestNoRoleGoesSilentWhileTheRootSpeaks(t *testing.T) {
	cfg := castOf("Samantha", map[Role]Pair{Alerts: {MacOS: "Nobody"}, Fire: {MacOS: "Nobody either"}})
	host := mac("Samantha")
	for _, r := range append([]Role{All}, Assignable()...) {
		if got := Resolve(r, cfg, host); got.Silent() {
			t.Errorf("%s resolved to silence while the root speaks: %+v", r.Key(), got)
		}
	}
}

// --- WantsInstall ---

func TestWantsInstallNamesOnlyTheMissingPiperKeys(t *testing.T) {
	cfg := castOf("en_US-lessac-medium", map[Role]Pair{
		Alerts:  {Piper: "en_US-ryan-medium"},   // named, missing -> wanted
		Fire:    {Piper: "en_US-lessac-medium"}, // named, present -> not wanted
		Weather: {MacOS: "Rishi"},               // the other OS's half -> nothing here
	})
	host := linux("en_US-lessac-medium")

	if got := WantsInstall(Alerts, cfg, host); got != "en_US-ryan-medium" {
		t.Errorf("WantsInstall(Alerts) = %q, want the missing key", got)
	}
	for _, r := range []Role{Fire, Weather, Seismic} {
		if got := WantsInstall(r, cfg, host); got != "" {
			t.Errorf("WantsInstall(%s) = %q, want nothing", r.Key(), got)
		}
	}
	// macOS installs nothing.
	if got := WantsInstall(Alerts, cfg, mac("Samantha")); got != "" {
		t.Errorf("WantsInstall on macOS = %q, want nothing to install", got)
	}
	if got := WantsInstall(Alerts, cfg, nil); got != "" {
		t.Errorf("WantsInstall with no host = %q, want nothing", got)
	}
}

// The deck asks once and applies ONE per-session cap to the list (MVS-D-44),
// so the list must be de-duplicated and ordered, not one answer per role.
func TestWantedInstallsIsDeduplicatedAndOrdered(t *testing.T) {
	cfg := castOf("en_US-missing-root", map[Role]Pair{
		Alerts:  {Piper: "en_US-ryan-medium"},
		Fire:    {Piper: "en_US-ryan-medium"}, // the same key twice
		Weather: {Piper: "en_US-amy-medium"},
	})
	got := WantedInstalls(cfg, linux())
	want := []string{"en_US-missing-root", "en_US-ryan-medium", "en_US-amy-medium"}
	if !slices.Equal(got, want) {
		t.Fatalf("WantedInstalls() = %v, want %v (root first, then registry order, de-duplicated)", got, want)
	}
	if got := WantedInstalls(cfg, mac("Samantha")); got != nil {
		t.Errorf("WantedInstalls on macOS = %v, want nothing", got)
	}
	if got := WantedInstalls(Config{}, linux("en_US-lessac-medium")); got != nil {
		t.Errorf("WantedInstalls with nothing named = %v, want nothing", got)
	}
}

func TestLinkNamesItselfForTheDiagnosticsPage(t *testing.T) {
	for _, tc := range []struct {
		link Link
		want string
	}{{LinkRole, "role"}, {LinkGroup, "group"}, {LinkRoot, "root"}, {LinkDefault, "default"}, {LinkNone, "none"}, {Link(99), "none"}} {
		if got := tc.link.String(); got != tc.want {
			t.Errorf("Link(%d).String() = %q, want %q", tc.link, got, tc.want)
		}
	}
}
