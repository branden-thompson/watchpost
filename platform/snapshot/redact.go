package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
)

// keyPattern is a coordinate pair in any spelling a writer is likely to reach
// for: two signed decimals of at least two places, each with a latitude- or
// longitude-sized whole part, joined by a comma and/or whitespace. Key's own
// form (four places, comma-joined) is the common case; `%.6f,%.6f` and the
// `%v` of a LocationRef (`33.2887 -117.2253`) are inside the bound too, so a
// pair that reaches a writer through free text — an error, a panic value — is
// still rewritten. The bound is stated, not assumed: a pair spelled with one
// decimal place, or in degrees-minutes, is outside it.
var keyPattern = regexp.MustCompile(`-?\d{1,3}\.\d{2,}(?:,\s*|\s+)-?\d{1,3}\.\d{2,}`)

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
