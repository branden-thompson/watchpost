package globalfeed

// supersede.go — the guarded superseded rule (0.13.0, NFR-12; red-team S3):
// an alert is superseded only by a NEWER message from the SAME sender for the
// SAME product that references it. Before the guard, any feature's references
// marked any id, so a crafted low-grade alert could hide a live warning.
// Shared by the national feed (nws.go) and, through domains/severe, the
// tracked-location path.

import "time"

// Ref is what the rule needs to know about one CAP message.
type Ref struct {
	ID       string
	Sender   string
	Product  string
	Sent     time.Time
	Replaces []string // the ids this message updates/replaces (CAP references)
}

// Supersedes reports whether newer legitimately replaces older.
func Supersedes(newer, older Ref) bool {
	if newer.ID == "" || older.ID == "" || newer.ID == older.ID {
		return false
	}
	if newer.Sender != older.Sender || newer.Product != older.Product {
		return false
	}
	if !newer.Sent.After(older.Sent) {
		return false
	}
	for _, r := range newer.Replaces {
		if r == older.ID {
			return true
		}
	}
	return false
}

// SupersededBy returns the set of ids in refs that a sibling legitimately
// supersedes.
func SupersededBy(refs []Ref) map[string]bool {
	byID := make(map[string]Ref, len(refs))
	for _, r := range refs {
		if r.ID != "" {
			byID[r.ID] = r
		}
	}
	out := make(map[string]bool)
	for _, newer := range refs {
		for _, target := range newer.Replaces {
			if older, ok := byID[target]; ok && Supersedes(newer, older) {
				out[target] = true
			}
		}
	}
	return out
}
