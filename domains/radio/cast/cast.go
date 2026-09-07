// Package cast is the station's role registry: who speaks for what, how a
// role inherits its voice from the level above, and the words the config file
// and the screen use for each. It knows nothing about audio, configuration
// files, or the terminal — it takes host facts and returns decisions, so the
// resolution rule has exactly one implementation and can be tested without a
// synthesizer (plan §2.2, data-shape.md §1).
//
// It imports nothing above platform/invariant.
package cast

import (
	"maps"

	"github.com/branden-thompson/watchpost/platform/invariant"
)

// Role is a node of the registry. The tree is
//
//	All ─┬─ Alerts ──┬─ Breaking
//	     │           └─ SevereRead
//	     └─ Standard ┬─ Weather
//	                 ├─ Maritime
//	                 ├─ Fire
//	                 ├─ Seismic
//	                 └─ Station
//
// A role with no voice of its own speaks in its parent's; the root is the
// 0.13.0 `voice` key, so a file that names nothing below it composes exactly
// today's broadcast (FR-2).
//
// Role is a plain int and crosses package boundaries — from a config file, a
// key press, a Segment. Every accessor therefore reads an out-of-range value
// as empty rather than panicking: a hand-edited file must never crash the app.
type Role int

// The registry. All is the zero value so an unset Segment.Role means "the root
// reads it", which is what an un-migrated caller wants.
const (
	All Role = iota
	Alerts
	Breaking
	SevereRead
	Standard
	Weather
	Maritime
	Fire
	Seismic
	Station

	// numRoles bounds every table below; it is not itself a role.
	numRoles
)

// role is one row of the registry. The table IS the registry: adding a role is
// one constant and one row, and nothing else in the package changes.
type role struct {
	key    string // the config word, and the [S] / report --verbose identifier
	label  string // the human name, as the Setup mock draws it
	parent Role
	self   bool // false only for All, whose parent is itself
}

var registry = [numRoles]role{
	All:        {key: "voice", label: "All reads", parent: All},
	Alerts:     {key: "alerts", label: "Alerts & Notification reads", parent: All, self: true},
	Breaking:   {key: "breaking", label: "Breaking takeover", parent: Alerts, self: true},
	SevereRead: {key: "severe_read", label: "Severe-event read", parent: Alerts, self: true},
	Standard:   {key: "standard", label: "Standard reports", parent: All, self: true},
	Weather:    {key: "weather", label: "Location weather", parent: Standard, self: true},
	Maritime:   {key: "maritime", label: "Location marine", parent: Standard, self: true},
	Fire:       {key: "fire", label: "Location fire & hotspots", parent: Standard, self: true},
	Seismic:    {key: "seismic", label: "Location seismic", parent: Standard, self: true},
	Station:    {key: "station", label: "Station", parent: Standard, self: true},
}

// valid reports whether r names a row of the registry. Everything that reads a
// Role goes through it, so an out-of-range value has exactly one meaning.
func (r Role) valid() bool { return r >= 0 && r < numRoles }

// Key is the word the config file, [S] and `report --verbose` use. Empty for a
// role outside the registry.
func (r Role) Key() string {
	if !r.valid() {
		return ""
	}
	return registry[r].key
}

// String is the label the screen draws. Empty for a role outside the registry,
// so a corrupt value renders as nothing rather than as "Role(37)".
func (r Role) String() string {
	if !r.valid() {
		return ""
	}
	return registry[r].label
}

// Parent is the role a voice is inherited from. The root's parent is itself,
// which is what ends the resolution walk; an invalid role also reports All, so
// a walk started from garbage terminates at the root instead of running off.
func (r Role) Parent() Role {
	if !r.valid() || !registry[r].self {
		return All
	}
	return registry[r].parent
}

// IsRoot reports whether r is the tree's root (the 0.13.0 `voice` key).
func (r Role) IsRoot() bool { return r == All }

// RoleByKey looks a role up by its config word, reporting whether the word is
// one the registry knows. An unknown key is NOT minted as a role: a typo in a
// hand-edited file is ignored and listed once in [S] (NFR-5).
func RoleByKey(key string) (Role, bool) {
	for r := range numRoles {
		if registry[r].key == key {
			return r, true
		}
	}
	return All, false
}

// Assignable is every role below the root, in the order the Setup window draws
// them and [S] lists them: the two alert roles under their group, then the five
// report roles under theirs. The root is excluded — it is assigned by the
// `voice` key, not by a role table.
//
// It returns a fresh slice: the registry is not writable through a caller.
func Assignable() []Role {
	return []Role{Alerts, Breaking, SevereRead, Standard, Weather, Maritime, Fire, Seismic, Station}
}

// Pair is one role's assignment: the macOS `say -v` name and the Piper
// catalogue KEY (never a display name). Each OS reads and writes only its own
// half, so a config file synced between a Mac and a Linux box keeps both
// (data-shape §2). An empty half means "inherit on this OS".
type Pair struct {
	MacOS string
	Piper string
}

// Empty reports whether the pair assigns nothing on either platform.
func (p Pair) Empty() bool { return p.MacOS == "" && p.Piper == "" }

// Cast modes (MVS-D-25). Switching to Single Voice keeps the pairs — they are
// simply not read — so switching back restores the whole cast.
const (
	// ModeSingle is the zero value: the root reads everything.
	ModeSingle = ""
	// ModeCast reads the per-role pairs.
	ModeCast = "cast"
)

// Config is the whole station's assignment, as the config file carries it and
// as the deck holds it. The ZERO VALUE is 0.13.0: single-voice, no pairs, no
// tone mutes — everything inherits the root (FR-2).
type Config struct {
	// Root is the 0.13.0 `voice` key: a single string, not a pair, for
	// compatibility (AM-11). On macOS it is a `say -v` name; elsewhere a Piper
	// catalogue key.
	Root string

	// Pairs is one entry per assignable role, keyed by Role. A role with no
	// entry — or an empty pair — inherits.
	Pairs map[Role]Pair

	// Mode is ModeSingle or ModeCast.
	Mode string

	// Tones is the per-class mute state.
	Tones Tones
}

// CastOn reports whether the per-role pairs are read at all.
func (c Config) CastOn() bool { return c.Mode == ModeCast }

// PairFor returns the pair assigned to r, or the zero pair when the role has
// none, is out of range, or the cast is off. Reading through one accessor is
// what keeps "the cast is off" from being re-implemented at every call site.
func (c Config) PairFor(r Role) Pair {
	if !c.CastOn() || !r.valid() || c.Pairs == nil {
		return Pair{}
	}
	return c.Pairs[r]
}

// WithPair returns a copy of c with r assigned p, copying the map so a stored
// Config is never mutated through a caller's reference. An invalid role is
// ignored rather than minted.
func (c Config) WithPair(r Role, p Pair) Config {
	if !r.valid() || r.IsRoot() {
		return c
	}
	pairs := make(map[Role]Pair, len(c.Pairs)+1)
	maps.Copy(pairs, c.Pairs)
	if p.Empty() {
		delete(pairs, r)
	} else {
		pairs[r] = p
	}
	c.Pairs = pairs
	return c
}

// Clone returns a deep copy: the map and the muted-class slice are the only
// reference types a Config holds.
func (c Config) Clone() Config {
	out := c
	if c.Pairs != nil {
		out.Pairs = make(map[Role]Pair, len(c.Pairs))
		maps.Copy(out.Pairs, c.Pairs)
	}
	// The muted-class slice is copied here rather than through a Tones method:
	// a Clone that calls a Clone reads as recursion to the safety gate
	// (P10-01), and one deep-copy owner is clearer than two anyway.
	if c.Tones.Muted != nil {
		out.Tones.Muted = append([]string(nil), c.Tones.Muted...)
	}
	return out
}

// checkRegistry is a build-time-ish assertion the tests run: every row of the
// registry has a key and a label, and every non-root parent is a real role.
// It is a function rather than an init so the package has no mutable state and
// no start-up cost (P10-06).
func checkRegistry() error {
	for r := range numRoles {
		if err := invariant.Check(registry[r].key != "" && registry[r].label != "", "cast: every role has a key and a label"); err != nil {
			return err
		}
		if err := invariant.Check(registry[r].parent.valid(), "cast: every parent is a role"); err != nil {
			return err
		}
	}
	return nil
}
