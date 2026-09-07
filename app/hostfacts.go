package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/branden-thompson/watchpost/domains/radio/cast"
	"github.com/branden-thompson/watchpost/domains/radio/synth"
	"github.com/branden-thompson/watchpost/platform/config"
	"github.com/branden-thompson/watchpost/platform/plaintext"
)

// hostFacts is a cast.Host WITHOUT a deck (Task 4.8).
//
// `watchpost report --verbose` must answer the same questions [S] answers —
// who reads what on this machine, and why the requested voice lost — from a
// shell, with no dashboard and no audio engine running. Two implementations of
// "which voices does this host have" is how a report and a screen start
// disagreeing about the same machine, so the deck and the report share this
// one, and the macOS discovery runs ONCE, under one ceiling.
type hostFacts struct {
	platform   string
	voiceDir   string
	discovered []string
}

// newHostFacts gathers the host's facts once. The context bounds the macOS
// `say -v ?` call; a wedged one leaves the curated list standing rather than
// hanging a report.
func newHostFacts(ctx context.Context) hostFacts {
	h := hostFacts{platform: runtimeGOOS, voiceDir: voiceDir()}
	if runtimeGOOS == cast.PlatformDarwin {
		h.discovered = discoverMacVoices(ctx)
	}
	return h
}

func (h hostFacts) Platform() string { return h.platform }

// Discovered is the same CLOSED ALLOWLIST the deck applies: the curated list
// until `say -v ?` has answered, the intersection afterwards.
func (h hostFacts) Discovered(name string) bool {
	if name == "" {
		return false
	}
	if name == systemVoice {
		return true
	}
	list := h.discovered
	if len(list) == 0 {
		list = macVoices()
	}
	for _, v := range list {
		if v == name {
			return true
		}
	}
	return false
}

// Installed is find-only, exactly as the deck's is.
func (h hostFacts) Installed(key string) bool {
	if key == "" {
		return false
	}
	_, ok := piperInstallFor(h.voiceDir, key)
	return ok
}

func (h hostFacts) Default() string {
	if h.platform == cast.PlatformDarwin {
		return systemVoice
	}
	if installed := synth.InstalledVoices(h.voiceDir); len(installed) > 0 {
		return installed[0].Name
	}
	return ""
}

// CastRow is one role's answer, ready to print: who was asked for, who speaks,
// which link won, and why the requested one lost.
type CastRow struct {
	Role       string
	Requested  string
	Spoken     string
	Link       string
	Reason     string
	Installing bool // a named Piper key that is not on disk yet
}

// CastReport is the whole answer — the cast rows, the tone state and the
// problems — built from a config and a host, with no deck anywhere.
type CastReport struct {
	Rows     []CastRow
	ToneMode string
	Muted    []string
	Problems []cast.Problem
}

// castReport builds it. [S], the diagnostic dump and `report --verbose` all
// read this one function, so the three cannot drift.
func castReport(cfg config.Config, host cast.Host) CastReport {
	c := castLoaded(cfg)
	out := CastReport{ToneMode: c.Tones.Mode, Muted: append([]string(nil), c.Tones.Muted...), Problems: cast.Validate(c, host)}
	for _, r := range append([]cast.Role{cast.All}, cast.Assignable()...) {
		res := cast.Resolve(r, c, host)
		out.Rows = append(out.Rows, CastRow{
			Role: r.Key(), Requested: res.Requested, Spoken: res.Spoken,
			Link: res.Link.String(), Reason: res.Reason,
			Installing: cast.WantsInstall(r, c, host) != "",
		})
	}
	return out
}

// ToneSummary words the tone state for a listener: "all tones on", "muted",
// or "muted: watches, advisories".
func (r CastReport) ToneSummary() string {
	if r.ToneMode != cast.ModeTonesMute {
		return "all tones on"
	}
	if len(r.Muted) == 0 {
		return "muted (nothing ticked: every class)"
	}
	// Only classes this build KNOWS are listed. An unknown key mutes nothing,
	// so naming it here would say a class is silenced when it is not — CONFIG
	// reports the key itself, which is where a listener can act on it.
	var names []string
	for _, k := range r.Muted {
		if c, ok := cast.ClassByKey(k); ok {
			names = append(names, c.String())
		}
	}
	if len(names) == 0 {
		return "muted (no recognised class ticked: every class)"
	}
	return "muted: " + strings.Join(names, ", ")
}

// RenderCast is `watchpost report --verbose`'s cast block: who reads what on
// this machine and why, and the tone state — the same answers [S] gives, with
// no deck and no dashboard (Task 4.8).
//
// A config that will not load is reported as such rather than skipped: a
// listener running --verbose because something is wrong needs to be told when
// the file itself is the something.
func RenderCast(ctx context.Context) string {
	var b strings.Builder
	b.WriteString("\nCAST\n")
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(&b, "  config unreadable: %s\n", plaintext.Line(err.Error()))
		return b.String()
	}
	ctx, cancel := context.WithTimeout(ctx, discoverTimeout)
	defer cancel()
	rep := castReport(cfg, newHostFacts(ctx))

	w := 0
	for _, r := range rep.Rows {
		w = max(w, len(r.Role))
	}
	for _, r := range rep.Rows {
		spoken := plaintext.Line(r.Spoken)
		if spoken == "" {
			spoken = "(silent)"
		}
		why := plaintext.Line(r.Reason)
		if r.Requested != "" && r.Requested != r.Spoken {
			why = plaintext.Line(r.Requested) + ": " + why
		}
		if r.Installing {
			why = strings.TrimSpace(why + " (installing)")
		}
		fmt.Fprintf(&b, "  %-*s  %-24s via %-8s %s\n", w, r.Role, spoken, r.Link, why)
	}
	fmt.Fprintf(&b, "\nTONES\n  %s\n", plaintext.Line(rep.ToneSummary()))
	if len(rep.Problems) > 0 {
		b.WriteString("\nCONFIG\n")
		for _, p := range rep.Problems {
			fmt.Fprintf(&b, "  %s: %s\n", plaintext.Line(p.Key), plaintext.Line(p.Message))
		}
	}
	return b.String()
}
