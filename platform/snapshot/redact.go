package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
)

// keyPattern is the shape Key prints: two signed decimals with four places,
// comma-joined. A card's ID, a tune's ref and a trace line all carry it.
var keyPattern = regexp.MustCompile(`-?\d+\.\d{4},-?\d+\.\d{4}`)

// HasKey reports whether s carries a location key — a coordinate pair.
func HasKey(s string) bool { return keyPattern.MatchString(s) }

// ReplaceKeys rewrites every location key in s through name. THE KEY IS A
// POSITION AT METRE PRECISION, and the transmitter is a pool member, so any
// text that reaches a file or a screen through a key names where the operator
// is (FR-9.4). Callers that write such text pass the naming they can afford:
// a label lookup where one exists, Opaque where one does not.
func ReplaceKeys(s string, name func(LocationKey) string) string {
	return keyPattern.ReplaceAllStringFunc(s, func(m string) string { return name(LocationKey(m)) })
}

// Opaque is a stable name for a key that is not a position: "place:" and the
// first eight hex digits of the key's SHA-256. Two places stay two names; a
// reader of the diagnostic can follow one place across lines; nobody can walk
// the name back to the pair.
func Opaque(key LocationKey) string {
	sum := sha256.Sum256([]byte(key))
	return "place:" + hex.EncodeToString(sum[:4])
}
