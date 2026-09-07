package cast

import "github.com/branden-thompson/watchpost/platform/invariant"

// Host is what resolution needs to know about the machine it is running on.
// It is an interface so the rule can be tested without a synthesizer, a
// subprocess or a filesystem: the whole fallback matrix (M5) is a table test
// over fakes (plan §2.2).
type Host interface {
	// Platform is "darwin" for the macOS half of a Pair, anything else for the
	// Piper half. It is a value, not runtime.GOOS, so the tests can walk both
	// namespaces on one machine.
	Platform() string

	// Discovered reports whether a macOS voice name may be spoken. It is a
	// CLOSED ALLOWLIST AT EVERY MOMENT: the curated list until `say -v ?` has
	// answered, the intersection afterwards. There is no window in which an
	// unknown name is trusted because discovery has not finished (D-R2-2) —
	// constructing a SayVoice from a name the host does not have is RS-2, and
	// the cost of being wrong is silence on an alert.
	Discovered(name string) bool

	// Installed reports whether a Piper catalogue key is present on disk. It
	// is FIND-ONLY — a stat, never an install (FR-9): the alert path may not
	// block on a 63 MB download.
	Installed(key string) bool

	// Default is this host's own last resort — the macOS System Voice
	// sentinel, the first installed Piper voice, or the catalogue default —
	// used when even the root did not resolve. Empty means the host truly has
	// nothing, which is the one legitimately silent row (AM-19).
	//
	// It is a host fact of the same kind as Discovered and Installed, so it
	// lives here rather than being re-derived at each call site. (The PLAN
	// named three methods; this is the fourth, and P1's build log records why.)
	Default() string
}

// Link says which node of the tree supplied the voice — the answer to "why is
// this correspondent reading?" that [S] prints.
type Link int

const (
	// LinkNone is the zero value: nothing resolved at all. On Linux with
	// nothing installed this is the one legitimately silent row (AM-19,
	// ratified MVS-D-41).
	LinkNone Link = iota
	// LinkRole is the role's own assignment.
	LinkRole
	// LinkGroup is an assignment inherited from Alerts or Standard.
	LinkGroup
	// LinkRoot is the 0.13.0 `voice` key.
	LinkRoot
	// LinkDefault is the platform's own default, when even the root did not
	// resolve.
	LinkDefault
)

// String names the link for [S] and `report --verbose`.
func (l Link) String() string {
	switch l {
	case LinkRole:
		return "role"
	case LinkGroup:
		return "group"
	case LinkRoot:
		return "root"
	case LinkDefault:
		return "default"
	default:
		return "none"
	}
}

// Reasons a requested voice lost. They are the [S] wording, so they are
// sentences a listener can act on rather than error codes.
const (
	// ReasonAssigned means nothing lost: the requested voice is speaking.
	ReasonAssigned = ""
	// ReasonInherited means the role names no voice of its own.
	ReasonInherited = "inherits"
	// ReasonNotInstalled means a Piper key is named but not on disk. The
	// background install is somebody else's job (FR-9); resolution only
	// reports it.
	ReasonNotInstalled = "not installed on this host"
	// ReasonUnknownHere means a macOS name is not on the host's list — a name
	// from the other operating system, or a typo.
	ReasonUnknownHere = "unknown on this host"
	// ReasonNothingResolvable means the walk reached the platform default and
	// even that did not resolve.
	ReasonNothingResolvable = "no voice is available on this host"
)

// Resolution is the whole answer: who was asked for, who speaks, where the
// voice came from and why the requested one lost. Spoken is the ONLY string
// that ever reaches a voice or the screen (RS-18) — a Requested name that lost
// must never be displayed as if it were speaking.
type Resolution struct {
	Role      Role
	Requested string // what the config asked for at Role, "" when it asked for nothing
	Spoken    string // what actually speaks; "" only on the legitimately silent row
	Link      Link
	Reason    string
}

// Silent reports the one legitimately silent outcome: nothing resolved
// anywhere on this host (AM-19). It is a state [S] explains, not a bug.
func (r Resolution) Silent() bool { return r.Spoken == "" }

// maxDepth bounds the walk. The tree is three levels — role, group, root — so
// four links including the platform default. The walk is counter-bounded
// rather than trusting the tree's shape (P10-02): a malformed registry must
// terminate, not hang the broadcast.
const maxDepth = 3

// Resolve walks role → group → root → platform default and returns the first
// voice that exists on this host, together with why the requested one lost.
//
// It reads only THIS host's half of each pair, so a config file written on a
// Mac and opened on Linux resolves through the Piper keys and falls back where
// there are none — the file is never rewritten and never "wrong", it simply
// has nothing to say on this platform (data-shape §3).
//
// Resolve sits at the P10-04 decision ceiling; anything further goes in a
// helper, not here.
func Resolve(role Role, cfg Config, host Host) Resolution {
	if host == nil {
		return Resolution{Role: role, Link: LinkNone, Reason: ReasonNothingResolvable}
	}
	requested := requestedAt(role, cfg, host)
	out := Resolution{Role: role, Requested: requested}

	cur, steps := role, 0
	for ; steps <= maxDepth; steps++ {
		if name := requestedAt(cur, cfg, host); name != "" && speakable(name, host) {
			out.Spoken, out.Link = name, linkFor(role, cur)
			out.Reason = reasonFor(requested, name, host)
			return out
		}
		if cur.IsRoot() {
			break
		}
		cur = cur.Parent()
	}
	// The walk must have reached the root: a tree that did not is a registry
	// bug, and finding out here beats resolving to the wrong correspondent.
	if err := invariant.Check(cur.IsRoot(), "cast: the resolution walk reaches the root"); err != nil {
		return Resolution{Role: role, Requested: requested, Link: LinkNone, Reason: ReasonNothingResolvable}
	}
	if def := host.Default(); def != "" {
		out.Spoken, out.Link = def, LinkDefault
		out.Reason = reasonFor(requested, def, host)
		return out
	}
	out.Link, out.Reason = LinkNone, ReasonNothingResolvable
	return out
}

// requestedAt is the name this host's namespace asks for at one node: the root
// is the bare `voice` key, every other node is its half of the pair.
func requestedAt(r Role, cfg Config, host Host) string {
	if r.IsRoot() {
		return cfg.Root
	}
	p := cfg.PairFor(r)
	if host.Platform() == PlatformDarwin {
		return p.MacOS
	}
	return p.Piper
}

// speakable is the host question, asked in this host's namespace. macOS names
// go through the closed allowlist; Piper keys through a find-only stat.
func speakable(name string, host Host) bool {
	if name == "" {
		return false
	}
	if host.Platform() == PlatformDarwin {
		return host.Discovered(name)
	}
	return host.Installed(name)
}

// linkFor names where the winning voice came from, relative to the role that
// was asked about.
func linkFor(asked, won Role) Link {
	switch {
	case won == asked:
		return LinkRole
	case won.IsRoot():
		return LinkRoot
	default:
		return LinkGroup
	}
}

// reasonFor explains what happened to the requested name. It is the only place
// "why" is decided, so [S], `report --verbose` and the Setup note cannot drift
// apart.
func reasonFor(requested, spoken string, host Host) string {
	switch {
	case requested == "":
		return ReasonInherited
	case requested == spoken:
		return ReasonAssigned
	case host.Platform() == PlatformDarwin:
		return ReasonUnknownHere
	default:
		return ReasonNotInstalled
	}
}

// PlatformDarwin is the Platform() value that selects the macOS half of a Pair.
const PlatformDarwin = "darwin"

// WantsInstall names the Piper catalogue key worth fetching in the background
// for this role: one that the config asks for, that is not on disk, and that
// therefore made the role fall back. It is empty on macOS (nothing to install),
// empty when the role asks for nothing, and empty when the key is already here.
//
// It is a QUESTION, not an action: whoever calls it decides whether to start a
// download, subject to the per-session cap (MVS-D-44).
func WantsInstall(role Role, cfg Config, host Host) string {
	if host == nil || host.Platform() == PlatformDarwin {
		return ""
	}
	key := requestedAt(role, cfg, host)
	if key == "" || host.Installed(key) {
		return ""
	}
	return key
}

// WantedInstalls is every key WantsInstall names across the root and the
// assignable roles, de-duplicated and in registry order. It is what the deck
// asks after a cast change, so the caller applies one cap to one list rather
// than a cap per role.
func WantedInstalls(cfg Config, host Host) []string {
	var out []string
	seen := make(map[string]bool)
	for _, r := range append([]Role{All}, Assignable()...) {
		key := WantsInstall(r, cfg, host)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, key)
	}
	return out
}
