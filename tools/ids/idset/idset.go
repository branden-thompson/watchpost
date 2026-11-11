// Package idset names the ID families the record's tables define and the
// code's comments cite: a ruling (D), a minor choice (A), a requirement (FR),
// a follow-up (F), a finding (PF, DR and the other lenses' prefixes). Each
// release numbers its own from 1, so a bare ID can mean a different row in
// each (0.19.5 D-11).
//
// tools/ids indexes the rows that define them, and tools/authoring asks new
// citations to name their release. Both read the families here, so the index
// and the rule cannot disagree about what an ID is.
package idset

import (
	"regexp"
	"strings"
)

// Families are the prefixes an ID may carry, longest first so a pattern built
// from them prefers NFR over FR and DEF over D.
var Families = []string{"NFR", "DEF", "FR", "DR", "PF", "RN", "DC", "JR", "DV", "JM", "RR", "RS", "OQ", "D", "A", "F"}

// ID matches one ID: a family, a hyphen and its number, with an optional
// decimal (0.18.0 FR-3.2) or letter (0.19.5 DR-7a). It does not look at what comes before,
// which is the caller's to judge: `MVS-D-80` and `0.19.5 D-4` are qualified.
var ID = regexp.MustCompile(`\b(` + strings.Join(Families, "|") + `)-(\d+)((?:\.\d+)*[a-z]?)\b`)

// Release matches a release number such as 0.19.5, which qualifies the IDs
// cited after it.
var Release = regexp.MustCompile(`\b\d+\.\d+\.\d+\b`)
