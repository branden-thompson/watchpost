package cast

import "fmt"

// Problem is one thing wrong with a cast, in the words [S] prints. Key names
// the config key a listener would edit — the whole point is that a problem is
// actionable, not merely detected.
type Problem struct {
	Key     string
	Message string
}

// Validate is the ONE validator of the cast's semantics: assigned names that
// cannot speak on this host, the cast mode, and the tone-class set.
//
// The split with platform/config is deliberate and load-bearing. Config
// validates table SHAPES — is this a string where a string belongs — because
// that is all a decoder can know. Everything that needs to know what a role IS,
// or which class keys exist, or whether a voice can speak here, is this
// function's, so the rule has one implementation and cannot drift between the
// loader, the screen and the deck (RS-2, FR-8).
//
// It CALLS INTO THE HOST (Discovered, Installed), so a caller holding the
// deck's lock must not call it: snapshot under the lock, validate outside it,
// store under the lock (Task 1.12). Calling it under the lock re-enters the
// deck through Discovered and deadlocks.
//
// A nil host validates everything that does not need one — the mode and the
// class keys — and skips the voice checks rather than reporting false
// problems.
func Validate(cfg Config, host Host) []Problem {
	var out []Problem
	out = append(out, validateMode(cfg)...)
	out = append(out, validateTones(cfg.Tones)...)
	out = append(out, validateVoices(cfg, host)...)
	return out
}

// validateMode checks the two closed sets of mode words. A hand-edited file is
// the only way to get one wrong, and the message names the value it found so
// the listener can see their own typo.
func validateMode(cfg Config) []Problem {
	var out []Problem
	if cfg.Mode != ModeSingle && cfg.Mode != ModeCast {
		out = append(out, Problem{
			Key:     "radio.cast",
			Message: fmt.Sprintf("%q is not a cast mode — use %q for a single voice or %q for a correspondent cast; reading it as a single voice", cfg.Mode, ModeSingle, ModeCast),
		})
	}
	if cfg.Tones.Mode != ModeTonesOn && cfg.Tones.Mode != ModeTonesMute {
		out = append(out, Problem{
			Key:     "radio.tones.mode",
			Message: fmt.Sprintf("%q is not a tone mode — use %q for all tones on or %q to mute; reading it as all tones on", cfg.Tones.Mode, ModeTonesOn, ModeTonesMute),
		})
	}
	return out
}

// validateTones reports unknown class keys ONCE, with a count.
//
// The array is file-controlled: a hand-edited or hostile config can name a
// thousand classes, and one problem per entry would bury every other problem
// on the page. The count is what a listener needs — "six of these are not
// classes" — plus the first one by way of an example (NFR-5, InfoSec SEC3-2).
func validateTones(t Tones) []Problem {
	var unknown int
	var first string
	for _, key := range t.Muted {
		if _, ok := ClassByKey(key); ok {
			continue
		}
		if unknown == 0 {
			first = key
		}
		unknown++
	}
	if unknown == 0 {
		return nil
	}
	msg := fmt.Sprintf("%d muted entries are not alert classes (the first is %q); they are ignored. The classes are: %v", unknown, first, ClassKeys())
	if unknown == 1 {
		msg = fmt.Sprintf("%q is not an alert class; it is ignored. The classes are: %v", first, ClassKeys())
	}
	return []Problem{{Key: "radio.tones.muted", Message: msg}}
}

// validateVoices reports every assigned name that cannot speak here, naming
// where it falls back to instead — a warning, never an error: falling back is
// the designed behaviour (FR-7), and the listener's file is not wrong just
// because they also use another machine.
//
// Roles are walked in registry order so [S] lists them the way Setup draws
// them, and the root is checked first because a root that cannot speak is the
// one problem that affects every other row.
func validateVoices(cfg Config, host Host) []Problem {
	if host == nil {
		return nil
	}
	var out []Problem
	for _, r := range append([]Role{All}, Assignable()...) {
		if r != All && !cfg.CastOn() {
			break // the pairs are kept but unread in Single Voice (MVS-D-25)
		}
		requested := requestedAt(r, cfg, host)
		if requested == "" || speakable(requested, host) {
			continue
		}
		res := Resolve(r, cfg, host)
		out = append(out, Problem{Key: keyOf(r), Message: fallbackMessage(requested, res, host)})
	}
	return out
}

// keyOf is the config key a listener edits for role r: the bare `voice` for the
// root, the platform's half of the role's table otherwise.
func keyOf(r Role) string {
	if r.IsRoot() {
		return r.Key()
	}
	return "radio.voices." + r.Key()
}

// fallbackMessage says what could not speak, why, and who reads instead.
func fallbackMessage(requested string, res Resolution, host Host) string {
	why := ReasonNotInstalled
	if host.Platform() == PlatformDarwin {
		why = ReasonUnknownHere
	}
	if res.Silent() {
		return fmt.Sprintf("%q is %s, and nothing else on this host can read — this role is silent", requested, why)
	}
	return fmt.Sprintf("%q is %s; %q reads instead (%s)", requested, why, res.Spoken, res.Link)
}
