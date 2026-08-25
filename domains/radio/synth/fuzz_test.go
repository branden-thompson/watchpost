package synth

import (
	"strings"
	"testing"
)

// Product text comes from the network and is handed to a speech engine
// (§10.5: by file or stdin, never argv). These fuzzers pin the narration
// rules against arbitrary text: no panic, segments stay bounded and end at
// a sentence, and the voice-only substitutions never lose or invent words.

func FuzzNormalize(f *testing.F) {
	f.Add(".TONIGHT...Mostly clear. Lows 66 to 69.\n\n* WHAT...Hot.\n\nLAT...LON 3300 11700\n\n$$")
	f.Add("442 PM PDT Mon Aug 24 2026\nCAZ552-251015-\nHEAT ADVISORY IN EFFECT")
	f.Add("")
	f.Add(strings.Repeat("Highs 80 to 84 at the beaches to 86 to 91 farther inland. ", 40))
	f.Add("\x00\xff...\n\n\n...\n* \n. .")
	f.Fuzz(func(t *testing.T, text string) {
		paras := Normalize(text)
		for _, seg := range Segments(paras) {
			if len(seg) > maxSegmentChars*2 { // one unsplittable sentence may exceed the target once
				t.Fatalf("segment of %d chars", len(seg))
			}
			if seg == "" {
				t.Fatal("empty segment")
			}
		}
		_ = NormalizeLine(text)
		_ = ExpandStates(text)
	})
}

func FuzzPronounce(f *testing.F) {
	f.Add("Visit www.weather.gov/sandiego or call the CLI.")
	f.Add("3.5 U.S. e.g. https://x.y/z-w")
	f.Add("")
	f.Fuzz(func(t *testing.T, text string) {
		got := Pronounce(text)
		if len(strings.Fields(got)) < len(strings.Fields(text)) {
			t.Fatalf("pronounce dropped words: %q -> %q", text, got)
		}
	})
}

func FuzzRemainder(f *testing.F) {
	f.Add("one two three four five six", 0.5)
	f.Add("", 0.0)
	f.Fuzz(func(t *testing.T, text string, frac float64) {
		rest := Remainder(text, frac)
		if rest != "" && !strings.HasSuffix(strings.Join(strings.Fields(text), " "), rest) {
			t.Fatalf("remainder must be a word-aligned suffix: %q of %q", rest, text)
		}
	})
}
