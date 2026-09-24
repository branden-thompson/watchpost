package app

// maps_fixtures_test.go — the recorded scenarios the map is judged on
// (observer-maps W0.2, FR-8.6; M1 and M1b's protocol in problem-statement.md).
//
// M1 and M1b are scored over a RECORDED set so that anyone can repeat a run:
// live weather never comes back. The set lives in testdata/maps/m1, one folder
// a scenario, captured from api.weather.gov with the zones each alert names.
// This test holds the set whole: every file the manifest lists is present, and
// the protocol's required kinds are all there.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

const m1Dir = "testdata/maps/m1"

type fixtureManifest struct {
	Captured string   `json:"captured"`
	Files    []string `json:"files"`
}

type m1Scenario struct {
	Name  string `json:"name"`
	Place struct {
		Name string  `json:"name"`
		Lat  float64 `json:"lat"`
		Lon  float64 `json:"lon"`
	} `json:"place"`
	Failure       *string                   `json:"failure"`
	WithheldZones []string                  `json:"withheld_zones"`
	AnswerKey     map[string]map[string]any `json:"answer_key"`
}

func TestEveryRecordedFixtureTheManifestNamesIsPresent(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(m1Dir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m fixtureManifest
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if len(m.Files) == 0 || m.Captured == "" {
		t.Fatal("the manifest names no file or no capture time — a pass would prove nothing")
	}
	for _, f := range m.Files {
		info, err := os.Stat(filepath.Join(m1Dir, f))
		if err != nil || info.Size() == 0 {
			t.Errorf("%s is named by the manifest but missing or empty: %v", f, err)
		}
	}
}

// TestTheScenarioSetMeetsTheProtocol: at least eight scenarios, including
// one that covers, one that stops short, one to one side, one partial, one
// marine and one failure (problem-statement.md, M1's protocol).
func TestTheScenarioSetMeetsTheProtocol(t *testing.T) {
	dirs, err := filepath.Glob(filepath.Join(m1Dir, "*", "scenario.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(dirs) < 8 {
		t.Fatalf("%d scenarios; the protocol asks for at least eight", len(dirs))
	}
	seen := map[string]bool{}
	for _, p := range dirs {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		var s m1Scenario
		if err := json.Unmarshal(b, &s); err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		if s.Place.Name == "" || len(s.AnswerKey) == 0 || s.Place.Lat == 0 || s.Place.Lon == 0 {
			t.Errorf("%s: a scenario needs a named place, its position and an answer key", p)
		}
		for _, a := range s.AnswerKey {
			if answer, ok := a["answer"].(string); ok {
				seen[answer] = true
			}
		}
		if len(s.WithheldZones) > 0 {
			seen["partial"] = true
		}
		if s.Failure != nil {
			seen["failure"] = true
		}
		if _, err := os.Stat(filepath.Join(filepath.Dir(p), "alerts.json")); err != nil {
			t.Errorf("%s: no recorded alerts", p)
		}
	}
	for _, kind := range []string{"covers", "stops short", "lies to one side", "partial", "failure"} {
		if !seen[kind] {
			t.Errorf("no scenario is %q", kind)
		}
	}
	if !marineIn(dirs) {
		t.Error("no marine scenario")
	}
}

// marineZone matches an NWS marine zone id: coastal and offshore waters
// (ANZ, AMZ, GMZ, PZZ, PKZ, PHZ, PMZ, PSZ) and the Great Lakes (LCZ … LSZ).
var marineZone = regexp.MustCompile(`^(AN|AM|GM|PZ|PK|PH|PM|PS|LC|LE|LH|LM|LO|LS)Z\d{3}\.json$`)

// marineIn reports whether any scenario's recorded alerts name a marine zone.
func marineIn(scenarios []string) bool {
	for _, p := range scenarios {
		entries, _ := os.ReadDir(filepath.Join(filepath.Dir(p), "zones"))
		for _, e := range entries {
			if marineZone.MatchString(e.Name()) {
				return true
			}
		}
	}
	return false
}
