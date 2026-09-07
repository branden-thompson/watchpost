package tty

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// THE PTY JOURNEY WAITS FOR WORDS. THIS CHECKS THE APP STILL SAYS THEM.
//
// scripts/quality/validate-journey.expect drives the real binary and asserts on
// what appears. When 0.14.0 renamed the UI, its patterns went stale — it waited
// for "Correspondent Cast", "Sig. Quakes", a "ºF" that is really "°F" — and
// every one of those became a FAIL that looked like the app breaking. Nothing
// ran the journey, so it stayed that way for the release (F-6, F-D3).
//
// A three-minute run against live feeds is the wrong instrument for "does this
// string still exist". The goldens are: they are real rendered frames, they run
// in the suite, and they are regenerated whenever the UI legitimately changes.
//
// WHAT THIS CANNOT SEE, stated rather than implied: there are goldens for the
// frame, Settings and the severe window only. A pattern belonging to Help, the
// Status window, the details window or the radio panel has no frame to be
// checked against, so it is exempted BY NAME below with where it comes from. An
// exemption is a debt, not a pass — each one is a window that would earn a
// golden.
var journeyUncovered = map[string]string{
	"NAVIGATE":                 "the Help window has no golden",
	"EVENT · |PLAYING|Reading": "the radio panel mid-read has no golden",
	"PROVIDERS|REQUESTS":       "the Status window has no golden",
	"SEVERE INDEX":             "the Status window has no golden",
	"Back|Recorded|Declared|Magnitude|No active":  "a severe record's detail view has no golden",
	"Hourly|HOURLY|Extended|EXTENDED|Marine|FIRE": "the location details window has no golden",
	"default location|NASA FIRMS":                 "the FIRST-RUN Setup form has no golden (the setup goldens are the grown window)",
	"Setup|Quit|Q u i t":                          "the masthead before first data has no golden",
	"Vista":                                       "a lookup result, which depends on live data",
	"Oceanside":                                   "the journey's own typed location, which depends on live data",
	"R E C E N T":                                 "drawn only once a second location exists",
	"Search:":                                     "the lookup modal has no golden",
	"ALERT|ALERTS|No active alerts":               "the alert-details window has no golden",
	"Setup|Theme":                                 "superseded by the 'Theme -' assertion; kept until the step is retired",
	"(?s).*":                                      "the drain helper's catch-all, not an assertion",
	"› *$label":                                   "tabTo's own construction inside the proc, not an assertion; its callers are checked",
	// The severe goldens all open on Warnings, so the SELECTED form of any other
	// tab appears in no frame. The labels themselves are pinned — severe_test.go's
	// tab-order test names all eight — and severeTabRow puts the pointer before
	// whichever tab is open, so what is unchecked here is the golden, not the word.
	// The debt is one golden rendered on a non-default tab.
	"› *Watch":  "no golden pins a severe tab other than the default (every severe golden opens on Warnings)",
	"› *Disast": "no golden pins a severe tab other than the default (every severe golden opens on Warnings)",
}

func TestTheJourneyWaitsForWordsTheAppStillSays(t *testing.T) {
	root := filepath.Join("..", "..")
	src, err := os.ReadFile(filepath.Join(root, "scripts", "quality", "validate-journey.expect"))
	if err != nil {
		t.Fatal(err)
	}
	frames := goldenCorpus(t, root)

	// CONTROLS. Without these a broken reader reports everything fine, which is
	// exactly how the first version of this check behaved when written by hand.
	if !strings.Contains(frames, "Lookup Location") {
		t.Fatal("control: a string certainly on screen is absent from the goldens; the corpus is not being read")
	}
	if strings.Contains(frames, "zzz-not-a-real-label") {
		t.Fatal("control: an impossible string was found; the search is not discriminating")
	}

	// BOTH SHAPES THE SCRIPT USES. Most steps go through seeAfter now, and when
	// they were converted this extractor silently fell to two patterns — caught
	// only by the count check below, which is why that check is here.
	// A FOURTH SHAPE, AND THE REASON IT IS HERE. tabTo was added on 2026-09-06 and
	// the extractor could not see it, which would have dropped three patterns out
	// of this guard silently — the same way the seeAfter conversion dropped
	// sixteen. tabTo builds its own regex from the label, so the pattern this
	// checks is the one expect actually receives, not the bare label.
	pat := regexp.MustCompile(`expect [^\n]*?-re "([^"]+)"|expect -timeout [0-9]+ "([^"]+)"|seeAfter "[^"]*" "([^"]+)"|tabTo "([^"]+)"`)
	// EXTRACTED counts what the reader FOUND; checked counts what it went on to
	// verify. They were one number, and that was wrong: every exemption lowered
	// the count that exists to detect a blind extractor, so adding honest
	// exemptions eventually forces the floor down and quietly disarms the
	// tripwire. An exempted pattern still proves the reader is reading.
	extracted, checked := 0, 0
	for _, m := range pat.FindAllStringSubmatch(string(src), -1) {
		p := m[1]
		if p == "" {
			p = m[2]
		}
		if p == "" {
			p = m[3]
		}
		if p == "" && m[4] != "" {
			p = "› *" + m[4] // tabTo's own construction, severe.go's pointer before the label
		}
		extracted++
		if _, ok := journeyUncovered[p]; ok {
			continue
		}
		checked++
		// MATCHED AS A REGEX, the way expect matches it. Several patterns are
		// not literals — `\d+°F` is the temperature — and a Contains check
		// reports those stale when they are fine. Tcl's escaping doubles the
		// backslashes in the file; undo that first.
		re, err := regexp.Compile(wordBounded(strings.ReplaceAll(p, `\\`, `\`)))
		if err != nil {
			t.Errorf("the journey's pattern %q does not compile: %v", p, err)
			continue
		}
		if !re.MatchString(frames) {
			t.Errorf("the journey waits for %q and no rendered frame contains it — that step will FAIL and "+
				"read as the app breaking. Retarget it, or exempt it in journeyUncovered with the window it belongs to.", p)
		}
	}
	if extracted < 24 {
		t.Errorf("only %d patterns were extracted; the reader has stopped matching the script (checked %d)", extracted, checked)
	}
}

// goldenCorpus is every rendered frame the package pins, with the escapes taken
// out so a pattern is matched against what a reader would see.
func goldenCorpus(t *testing.T, root string) string {
	t.Helper()
	dir := filepath.Join(root, "modes", "tty", "testdata")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	sgr := regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)
	var b strings.Builder
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".golden") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		b.WriteString(sgr.ReplaceAllString(string(raw), ""))
		b.WriteByte('\n')
	}
	return b.String()
}

// wordBounded stops a bare word matching INSIDE a longer one.
//
// It was written after the guard passed "CAST" — which appears in the goldens
// only as the tail of "NOTABLE EVENTS AND FORECASTS". The journey was waiting
// for a CAST block the Status window does not draw, and the check that exists to
// catch exactly that said it was fine. A guard with a false positive is worse
// than none: it converts an unchecked thing into a checked-looking one.
//
// Only whole-word patterns are wrapped. Anything carrying regex syntax is left
// alone, because \b around a pattern that already anchors itself changes what it
// means.
func wordBounded(p string) string {
	isWord := func(r byte) bool {
		return r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
	}
	for _, r := range p {
		if !(r == ' ' || r == '|' || r == '/' || r == '.' || r == '-' ||
			(r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return p
		}
	}
	alts := strings.Split(p, "|")
	for i, a := range alts {
		if a == "" {
			continue
		}
		// THE BOUNDARY GOES ONLY WHERE A WORD ENDS. Wrapping unconditionally put
		// \b after the hyphen in "Theme -", which asks for a word character that
		// is a space in the frame — turning a correct pattern into a failure.
		pre, post := "", ""
		if isWord(a[0]) {
			pre = `\b`
		}
		if isWord(a[len(a)-1]) {
			post = `\b`
		}
		alts[i] = pre + regexp.QuoteMeta(a) + post
	}
	return strings.Join(alts, "|")
}
