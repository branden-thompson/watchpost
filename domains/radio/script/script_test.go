package script

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Every built-in file follows the convention, and every one renders with
// the data its first comment line names.
func TestBuiltinScriptsFollowTheConvention(t *testing.T) {
	files := BuiltinFiles()
	if len(files) < 28 {
		t.Fatalf("built-in scripts: %v", files)
	}
	lib := New("")
	// Every field any built-in script names (each file's first line documents its own).
	data := map[string]any{
		"Product": "Tornado Warning", "Location": "Olathe, KS", "Items": "Extreme, Immediate, Observed", "Window": "45 minutes", "Text": "TAKE COVER NOW!", "Line": "Tornado Warning has been declared for Olathe, Kansas",
		"Live": "A version of this forecast is also broadcast live.", "Callsign": "KEC62", "Where": "San Diego, California", "From": "Monday, August 24", "Until": "Sunday, August 30",
		"Headline": "Heat Advisory", "Description": "Hot.", "Voice": "Samantha", "Sources": "FIRMS and HMS", "Count": 2, "Ring": "16 mile", "FRP": "62", "Detected": "2 hours", "Satellite": "GOES-West",
		"Name": "Timber", "HasDistance": true, "Inside": true, "Distance": "12 miles", "Direction": "east", "Facts": "is 26 percent contained",
		"Mag": "5.1", "Depth": "9 kilometers", "Ago": "3 days", "Felt": "A quake of this magnitude has a strong likelihood of being felt when it occurs.", "Likelihood": "strong", "Rest": 2, "Noun": "quakes",
	}
	for _, f := range files {
		report, file, ok := strings.Cut(f, "/")
		part, isTxt := strings.CutSuffix(file, Ext)
		if !ok || !isTxt || !nameRE.MatchString(report) || !nameRE.MatchString(part) || strings.Contains(part, "/") {
			t.Errorf("%q does not follow scripts/<report>/<part>.txt", f)
			continue
		}
		out, err := lib.Text(report, part, data)
		if err != nil || out == "" || strings.Contains(out, "{{") || strings.HasSuffix(out, "\n") {
			t.Errorf("%s: %q %v", f, out, err)
		}
	}
	if got := lib.Reports(); strings.Join(got, ",") != "breaking,event-report,fire-report,global,seismic-report,voice-preview,weather-radio" {
		t.Errorf("reports: %v", got)
	}
	if got := lib.Parts("global"); strings.Join(got, ",") != "head,tail" {
		t.Errorf("global parts: %v", got)
	}
}

// The parts the app relies on exist — the contract between the library and
// its callers (a renamed file fails here, not at runtime).
func TestTheAppsPartsExist(t *testing.T) {
	lib := New("")
	for _, want := range []string{
		"event-report/head", "event-report/tail", "event-report/opening", "event-report/meta", "event-report/window", "event-report/instructions",
		"breaking/single", "breaking/burst-line", "breaking/burst-closing",
		"weather-radio/head", "weather-radio/live", "weather-radio/span", "weather-radio/conditions", "weather-radio/alert", "weather-radio/tail",
		"fire-report/head", "fire-report/count", "fire-report/strongest", "fire-report/incident", "fire-report/outside",
		"seismic-report/head", "seismic-report/count", "seismic-report/quake", "seismic-report/felt", "seismic-report/more", "seismic-report/link",
		"voice-preview/sample",
	} {
		r, p, _ := strings.Cut(want, "/")
		if !lib.Has(r, p) {
			t.Errorf("missing %s", want)
		}
	}
}

// A report without a head or tail of its own inherits global's; a report
// with its own keeps it.
func TestGlobalHeadAndTailAreInherited(t *testing.T) {
	lib := New("")
	head, err := lib.Text("breaking", "head", nil) // breaking has no head file
	if err != nil || !strings.HasPrefix(head, "This is a Watchpost Notification Report.") {
		t.Fatalf("inherited head: %q %v", head, err)
	}
	own, _ := lib.Text("event-report", "head", nil)
	if !strings.HasPrefix(own, "This is a Watchpost Severe Weather Notification Report.") {
		t.Fatalf("the report's own head wins: %q", own)
	}
	if !lib.Has("anything-new", "tail") || lib.Has("anything-new", "opening") {
		t.Fatal("global lends head/tail to any report, nothing else")
	}
	if _, err := lib.Text("global", "opening", nil); !errors.Is(err, ErrNoScript) {
		t.Errorf("global has no opening and inherits from nobody: %v", err)
	}
}

func TestOverrideFileWinsPhraseByPhrase(t *testing.T) {
	dir := t.TempDir()
	must := func(rel, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Join(dir, filepath.Dir(rel)), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, rel), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	must("event-report/head.txt", "Custom head for {{.Product}}.\n")
	must("my-report/line.txt", "A new report: {{.Line}}")
	must("README.txt", "not a script")
	lib := New(dir)
	head, err := lib.Text("event-report", "head", map[string]string{"Product": "Tornado Warning"})
	if err != nil || head != "Custom head for Tornado Warning." {
		t.Fatalf("override: %q %v", head, err)
	}
	if tail, _ := lib.Text("event-report", "tail", nil); !strings.HasPrefix(tail, "This concludes this Watchpost Severe") {
		t.Fatalf("the other parts stay built in: %q", tail)
	}
	if line, err := lib.Text("my-report", "line", map[string]string{"Line": "x"}); err != nil || line != "A new report: x" {
		t.Fatalf("a new report is a new folder: %q %v", line, err)
	}
	if tail, err := lib.Text("my-report", "tail", nil); err != nil || !strings.HasPrefix(tail, "This concludes this Watchpost Notification") {
		t.Fatalf("a new report inherits global's tail: %q %v", tail, err)
	}
	if got := lib.Reports(); strings.Join(got, ",") != "breaking,event-report,fire-report,global,my-report,seismic-report,voice-preview,weather-radio" {
		t.Errorf("reports with the override: %v", got)
	}
}

func TestMissingAndBadScripts(t *testing.T) {
	lib := New("")
	if _, err := lib.Text("event-report", "nope", nil); !errors.Is(err, ErrNoScript) {
		t.Errorf("missing part: %v", err)
	}
	if _, err := lib.Text("Bad Name", "head", nil); !errors.Is(err, ErrNoScript) {
		t.Errorf("a name outside the convention: %v", err)
	}
	if _, err := lib.Text("../global", "head", nil); !errors.Is(err, ErrNoScript) {
		t.Errorf("a path outside the tree is not a name: %v", err)
	}
	if _, err := lib.Text("event-report", "opening", map[string]string{"Location": "x"}); err == nil {
		t.Error("a missing data field is an error, not silence")
	}
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "breaking"), 0o700)
	_ = os.WriteFile(filepath.Join(dir, "breaking", "single.txt"), []byte("{{.Line"), 0o600)
	if _, err := New(dir).Text("breaking", "single", map[string]string{"Line": "x"}); err == nil {
		t.Error("a malformed override is an error")
	}
}

// An override part is bounded (REVIEW R5-C-15): a 20 MB tail is refused
// (silence, an error), a normal one serves.
func TestOverrideFilesAreBounded(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "event-report"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "event-report", "tail.txt"), []byte(strings.Repeat("x", maxOverrideBytes+1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := New(dir).Text("event-report", "tail", nil); err == nil {
		t.Fatal("an oversized override must be an error")
	}
	if err := os.WriteFile(filepath.Join(dir, "event-report", "tail.txt"), []byte("Short."), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := New(dir).Text("event-report", "tail", nil); err != nil || got != "Short." {
		t.Fatalf("a bounded override serves: %q %v", got, err)
	}
}
