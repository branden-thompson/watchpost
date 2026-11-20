// Package idset names the ID families the record's tables define and the
// code's comments cite: a ruling (D), a minor choice (A), a requirement (FR),
// a follow-up (F), a finding (PF, DR and the other lenses' prefixes), a UAT
// round's finding (0.18.0 U2-61), a plan's task (0.18.0 W1.17). Each release
// numbers its own from 1, so a bare ID can mean a different row in each
// (0.19.5 D-11).
//
// tools/ids indexes the rows that define them, and tools/authoring asks new
// citations to name their release. Both read the families here, so the index
// and the rule cannot disagree about what an ID is.
package idset

import (
	"regexp"
	"strings"
)

// Families are the prefixes an ID written with a hyphen may carry, each a
// pattern, longest first so a pattern built from them prefers NFR over FR and
// DEF over D. U[0-9]+ is a UAT round's finding, 0.18.0 U2-61: each round
// numbers its own from 1 (0.19.6 D-50).
var Families = []string{"NFR", "DEF", "FR", "DR", "PF", "RN", "DC", "JR", "DV", "JM", "RR", "RS", "OQ", "PD", "U[0-9]+", "D", "A", "F", "S"}

// Task is the family of a plan's task, written with no hyphen and always with
// its decimal: 0.18.0 W1.17 is task 17 of wave 1, and a wave alone (W14) is
// no ID (0.19.6 D-50).
const Task = "W"

// ID matches one ID: a family, a hyphen and its number, with an optional
// decimal (0.18.0 FR-3.2) or letter (0.19.5 DR-7a); or a task (0.18.0
// W1.17). It does not look at what comes before, which is the caller's to
// judge: `MVS-D-80` and `0.19.5 D-4` are qualified. Parts reads a match of
// either form.
var ID = regexp.MustCompile(`\b(?:(` + strings.Join(Families, "|") + `)-(\d+)((?:\.\d+)*[a-z]?)|(` + Task + `)(\d+)((?:\.\d+)+))\b`)

// Grep is ID's pattern in the extended syntax git grep -E reads.
var Grep = "(" + strings.Join(Families, "|") + ")-[0-9]+|" + Task + "[0-9]+([.][0-9]+)+"

// Parts is an ID match's family, number and decimal or letter, whichever form
// it took.
func Parts(m []string) (family, num, suffix string) {
	if m[4] != "" {
		return m[4], m[5], m[6]
	}
	return m[1], m[2], m[3]
}

// Written is an ID as the record writes it: a task's family and number run
// together (0.18.0 W1.17), every other family's joined by a hyphen (0.18.0
// U2-61).
func Written(family, num, suffix string) string {
	if family == Task {
		return family + num + suffix
	}
	return family + "-" + num + suffix
}

// Release matches a release number such as 0.19.5, which qualifies the IDs
// cited after it.
var Release = regexp.MustCompile(`\b\d+\.\d+\.\d+\b`)
