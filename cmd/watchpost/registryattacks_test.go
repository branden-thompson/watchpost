package main

// registryattacks_test.go — section F of 06_docs/gate-attack-list.md: the
// registry over synthetic tables, so the CAUGHT direction of every registry
// rule is proved rather than asserted by prose.

import (
	"strings"
	"testing"
)

// ---- F. the registry, over synthetic tables ------------------------------------------

func TestTheRegistryAttackList(t *testing.T) {
	yes := func(*testing.T, string) bool { return true }
	// honest answers a known-absent subject and a known-satisfied one correctly.
	honestExists := func(_ *testing.T, s string) bool {
		return s != "absent-thing" && !strings.HasSuffix(s, "/no/such/subject")
	}
	honestNeeded := func(_ *testing.T, s string) bool { return s != "satisfied-thing" }
	good := func(rows map[string]string) *exemptionTable {
		return &exemptionTable{name: "specimen", rows: rows, exists: honestExists, stillNeeded: honestNeeded,
			satisfied: "satisfied-thing"}
	}
	realReason := "a real sentence explaining why this row is true today"

	specimens := []struct {
		name   string
		table  *exemptionTable
		caught bool
	}{
		{"C-ok1 a well-formed table with an honest row", good(map[string]string{"thing": realReason}), false},
		{"A28 an empty reason", good(map[string]string{"thing": ""}), true},
		{"A29 a shrug for a reason", good(map[string]string{"thing": "n/a"}), true},
		{"A29 a reason under the floor", good(map[string]string{"thing": "because"}), true},
		{"A30 a subject that no longer exists", good(map[string]string{"absent-thing": realReason}), true},
		{"A31 a subject the rule already accepts", good(map[string]string{"satisfied-thing": realReason}), true},
		{"F1 exists that cannot return false", &exemptionTable{name: "s", rows: map[string]string{"thing": realReason},
			exists: yes, stillNeeded: honestNeeded, satisfied: "satisfied-thing"}, true},
		{"L1 exists honest only for a subject the table would have chosen", &exemptionTable{name: "s", rows: map[string]string{"thing": realReason},
			exists: func(_ *testing.T, s string) bool { return s != "absent-thing" }, stillNeeded: honestNeeded, satisfied: "satisfied-thing"}, true},
		{"N3 exists that matches the nonce by SHAPE (a fixed prefix)", &exemptionTable{name: "s", rows: map[string]string{"thing": realReason},
			exists: func(_ *testing.T, s string) bool { return !strings.HasPrefix(s, "__registry") }, stillNeeded: honestNeeded, satisfied: "satisfied-thing"}, true},
		{"F2 stillNeeded that cannot return false", &exemptionTable{name: "s", rows: map[string]string{"thing": realReason},
			exists: honestExists, stillNeeded: yes, satisfied: "satisfied-thing"}, true},
		{"F1 a table with no satisfied subject declared", &exemptionTable{name: "s", rows: map[string]string{"thing": realReason},
			exists: honestExists, stillNeeded: honestNeeded}, true},
		{"a table with no functions at all", &exemptionTable{name: "s", rows: map[string]string{"thing": realReason}}, true},
		{"the no functions never called on a good table", good(map[string]string{"thing": realReason}), false},
	}
	for _, sp := range specimens { // bounded by the specimen table (P10-02)
		t.Run(sp.name, func(t *testing.T) {
			fired, said := verdictOf(func(r reporter) { assertRegistry(r, []*exemptionTable{sp.table}) })
			switch {
			case sp.caught && !fired:
				t.Errorf("SURVIVED — the registry accepted a row it must refuse")
			case !sp.caught && fired:
				t.Errorf("FALSE POSITIVE — a correct table was refused:\n  %s", strings.Join(said, "\n  "))
			}
		})
	}
}
