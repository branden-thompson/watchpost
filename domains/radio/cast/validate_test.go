package cast

import (
	"strings"
	"testing"
)

func problemFor(ps []Problem, key string) (Problem, bool) {
	for _, p := range ps {
		if p.Key == key {
			return p, true
		}
	}
	return Problem{}, false
}

func TestValidateFindsNothingWrongWithAWorkingCast(t *testing.T) {
	cfg := castOf("Samantha", map[Role]Pair{Alerts: {MacOS: "Rishi"}, Fire: {MacOS: "Samantha"}})
	cfg.Tones = Tones{Mode: ModeTonesMute, Muted: []string{"watch", "advisory"}}
	if got := Validate(cfg, mac("Samantha", "Rishi")); len(got) != 0 {
		t.Fatalf("a clean config yielded %d problems: %+v", len(got), got)
	}
	// The zero config is 0.13.0 and is clean by construction.
	if got := Validate(Config{}, mac("Samantha")); len(got) != 0 {
		t.Fatalf("the zero config yielded %+v", got)
	}
}

func TestValidateNamesAnUnspeakableVoiceAndWhoReadsInstead(t *testing.T) {
	cfg := castOf("Samantha", map[Role]Pair{Breaking: {MacOS: "Nobody"}})
	got := Validate(cfg, mac("Samantha"))
	p, ok := problemFor(got, "radio.voices.breaking")
	if !ok {
		t.Fatalf("no problem for the unspeakable role: %+v", got)
	}
	for _, want := range []string{`"Nobody"`, ReasonUnknownHere, `"Samantha"`, "root"} {
		if !strings.Contains(p.Message, want) {
			t.Errorf("message %q does not say %q — a warning nobody can act on is not a warning", p.Message, want)
		}
	}
}

func TestValidateSaysNotInstalledOnLinuxAndUnknownOnMac(t *testing.T) {
	linuxProblems := Validate(castOf("en_US-lessac-medium", map[Role]Pair{Fire: {Piper: "en_US-missing"}}), linux("en_US-lessac-medium"))
	p, ok := problemFor(linuxProblems, "radio.voices.fire")
	if !ok || !strings.Contains(p.Message, ReasonNotInstalled) {
		t.Errorf("Linux should report a missing key as not installed, got %+v", linuxProblems)
	}
	macProblems := Validate(castOf("Samantha", map[Role]Pair{Fire: {MacOS: "Nobody"}}), mac("Samantha"))
	p, ok = problemFor(macProblems, "radio.voices.fire")
	if !ok || !strings.Contains(p.Message, ReasonUnknownHere) {
		t.Errorf("macOS should report an off-list name as unknown here, got %+v", macProblems)
	}
}

func TestValidateReportsTheSilentRowAsSilent(t *testing.T) {
	cfg := castOf("en_US-missing-root", map[Role]Pair{Breaking: {Piper: "en_US-also-missing"}})
	got := Validate(cfg, linux()) // nothing installed, no default
	p, ok := problemFor(got, "voice")
	if !ok {
		t.Fatalf("the unresolvable root produced no problem: %+v", got)
	}
	if !strings.Contains(p.Message, "silent") {
		t.Errorf("message %q must say the role is silent — the one legitimately silent row is still worth stating", p.Message)
	}
}

// In Single Voice the pairs are kept but unread (MVS-D-25), so an assignment
// that cannot speak here is not a problem: nothing is trying to speak it.
func TestValidateIgnoresThePairsWhenTheCastIsOff(t *testing.T) {
	cfg := Config{Root: "Samantha", Pairs: map[Role]Pair{Breaking: {MacOS: "Nobody"}}}
	if got := Validate(cfg, mac("Samantha")); len(got) != 0 {
		t.Fatalf("Single Voice reported %+v — the pairs are kept but unread", got)
	}
	cfg.Mode = ModeCast
	if got := Validate(cfg, mac("Samantha")); len(got) != 1 {
		t.Fatalf("switching the cast on must surface the problem, got %+v", got)
	}
}

func TestValidateRejectsModesOutsideTheirClosedSets(t *testing.T) {
	cfg := Config{Mode: "casr", Tones: Tones{Mode: "muted"}} // two plausible typos
	got := Validate(cfg, mac("Samantha"))
	p, ok := problemFor(got, "radio.cast")
	if !ok || !strings.Contains(p.Message, `"casr"`) {
		t.Errorf("the cast mode problem must quote the value found, got %+v", got)
	}
	p, ok = problemFor(got, "radio.tones.mode")
	if !ok || !strings.Contains(p.Message, `"muted"`) {
		t.Errorf("the tone mode problem must quote the value found, got %+v", got)
	}
}

// A file-controlled array must not be able to fill the diagnostics page: one
// problem, with a count, however many unknown keys it names.
func TestValidateReportsUnknownClassKeysOnceWithACount(t *testing.T) {
	many := []string{"watch"} // one real key, then a flood of junk
	for i := range 200 {
		many = append(many, string(rune('a'+i%26))+"-not-a-class")
	}
	got := Validate(Config{Tones: Tones{Mode: ModeTonesMute, Muted: many}}, mac("Samantha"))
	if len(got) != 1 {
		t.Fatalf("200 unknown keys produced %d problems, want exactly 1: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Message, "200") {
		t.Errorf("the problem must carry the count, got %q", got[0].Message)
	}
	if got[0].Key != "radio.tones.muted" {
		t.Errorf("Key = %q, want radio.tones.muted", got[0].Key)
	}

	// A single unknown key reads as a sentence, not as "1 entries".
	one := Validate(Config{Tones: Tones{Mode: ModeTonesMute, Muted: []string{"warnings"}}}, mac("Samantha"))
	if len(one) != 1 || strings.Contains(one[0].Message, "1 muted entries") {
		t.Errorf("a single unknown key should read naturally, got %+v", one)
	}
	if !strings.Contains(one[0].Message, `"warnings"`) {
		t.Errorf("the message must quote the key, got %q", one[0].Message)
	}
}

// Validate calls into the host; a nil one must degrade rather than lie.
func TestValidateWithoutAHostChecksOnlyWhatItCan(t *testing.T) {
	cfg := castOf("Nobody At All", map[Role]Pair{Fire: {MacOS: "Also Nobody"}})
	cfg.Mode = "wrong"
	cfg.Tones = Tones{Mode: ModeTonesMute, Muted: []string{"not-a-class"}}
	got := Validate(cfg, nil)
	if len(got) != 2 {
		t.Fatalf("a nil host should still check the mode and the class keys, got %+v", got)
	}
	for _, p := range got {
		if strings.HasPrefix(p.Key, "radio.voices.") || p.Key == "voice" {
			t.Errorf("a nil host must not produce voice problems, got %+v", p)
		}
	}
}
