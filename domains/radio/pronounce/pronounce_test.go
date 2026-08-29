package pronounce

import (
	"strings"
	"testing"
)

// Every rule file follows the convention: a lower-case name, "KEY<TAB>value"
// (or a bare KEY for a set), no duplicate keys, no empty halves.
func TestRuleFilesFollowTheConvention(t *testing.T) {
	names := Names()
	if strings.Join(names, ",") != "abbreviations,states,states-ambiguous,words,zones" {
		t.Fatalf("tables: %v", names)
	}
	for _, name := range names {
		if strings.ToLower(name) != name || strings.ContainsAny(name, " _.") {
			t.Errorf("%q: table names are lower-case words and hyphens", name)
		}
		seen := map[string]bool{}
		for _, line := range lines(name) {
			k, v, tabbed := strings.Cut(line, "\t")
			if k == "" || seen[k] {
				t.Errorf("%s: empty or duplicate key in %q", name, line)
			}
			seen[k] = true
			if name != "states-ambiguous" && (!tabbed || strings.TrimSpace(v) == "") {
				t.Errorf("%s: %q needs KEY<TAB>spoken form", name, line)
			}
			if name == "states-ambiguous" && tabbed {
				t.Errorf("%s: a set has bare keys: %q", name, line)
			}
		}
		if len(seen) == 0 {
			t.Errorf("%s: no rules", name)
		}
	}
}

// The rules the synth relies on are on file — the contract (a renamed file
// or a dropped line fails here, not on the air).
func TestTheSynthsRulesExist(t *testing.T) {
	for _, c := range []struct{ table, key, want string }{
		{"zones", "PDT", "Pacific Daylight Time"}, {"zones", "MDT", "Mountain Daylight Time"}, {"zones", "CDT", "Central Daylight Time"},
		{"zones", "EDT", "Eastern Daylight Time"}, {"zones", "UTC", "Coordinated Universal Time"},
		{"states", "CA", "California"}, {"abbreviations", "NWS", "National Weather Service"}, {"words", "NOAA", "Noah"},
	} {
		if got := Table(c.table)[c.key]; got != c.want {
			t.Errorf("%s/%s = %q, want %q", c.table, c.key, got, c.want)
		}
	}
	amb := Set("states-ambiguous")
	if !amb["IN"] || !amb["OR"] || amb["CA"] {
		t.Errorf("ambiguous states: %v", amb)
	}
	for k := range amb { // every ambiguous code is a state
		if Table("states")[k] == "" {
			t.Errorf("ambiguous %q is not in states", k)
		}
	}
}

func TestUnknownAndUnsafeNamesAreEmpty(t *testing.T) {
	if len(Table("nope")) != 0 || len(Set("../rules/zones")) != 0 || len(Table("zones.txt")) != 0 {
		t.Fatal("an unknown or unsafe name reads as an empty table")
	}
}
