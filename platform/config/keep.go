package config

import (
	"errors"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"github.com/branden-thompson/watchpost/platform/plaintext"
)

// Preserving keys this build does not know (NFR-5).
//
// The hole this closes: load → edit → save goes through the typed struct, so
// any key the struct has no field for is silently dropped on the next write.
// Before 0.14.0 that was harmless — there were no other builds. From 0.14.0 on
// it is not: a 0.13.0 binary opening a 0.14.0 file would delete the whole cast
// on its next save, and the listener would lose every assignment by doing
// nothing more than running an older build once.
//
// So Save merges: the OLD file's unknown keys are copied into the newly
// marshalled document before it is written. This build cannot know what they
// mean, and does not need to — it only needs to not destroy them.

// maxUnknownReported bounds Config.Unknown. The file is user-controlled and a
// hostile or hand-mangled one can name thousands of keys; [S] shows a list a
// person reads, not a dump (Task 1.11).
const maxUnknownReported = 32

// unknownKeys strict-decodes raw and returns each unknown key AS SEGMENTS.
//
// Segments, never a joined string. A quoted top-level key like "radio.mode" and
// the nested key radio.mode both render as `radio.mode` when joined, so joining
// and re-splitting would let a hand-crafted quoted key alias a real nested one
// and overwrite it on save. Path identity lives in the slice, and the dotted
// form exists only for display.
//
// This is a DIAGNOSTIC. It may never fail a Load: a decoder that cannot make
// sense of the old file simply reports no unknown keys, and Save then writes
// the typed document unchanged. The recover is scoped to this function alone —
// a panic in a caller's loop must never be swallowed here and read as "no
// unknown keys" (red-team; go-toml v2.2.4 panicked on a quoted key carrying an
// escape, which is why v2.4.3 is a required bump, not an optional one).
func unknownKeys(raw []byte) (paths [][]string) {
	defer func() {
		if r := recover(); r != nil {
			paths = nil
		}
	}()
	var probe Config
	dec := toml.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	var strict *toml.StrictMissingError
	if err := dec.Decode(&probe); err == nil || !errors.As(err, &strict) {
		return nil // no unknown keys, or an old file this decoder cannot read
	}
	for _, e := range strict.Errors {
		if key := append([]string(nil), e.Key()...); len(key) > 0 {
			paths = append(paths, key)
		}
	}
	return paths
}

// keepUnknown copies every path in paths from the old document into the new
// one, returning how many were kept.
//
// Three rules, each earned:
//
//   - PARENTS FIRST. An unknown table is copied whole, and its children are
//     then already present; walking children first would copy a leaf into a
//     table that the parent copy later replaces.
//   - A KNOWN SCALAR IS NEVER TURNED INTO A TABLE. If the new document holds a
//     scalar where a path wants to descend, the path is abandoned. Otherwise a
//     stray `voice.something` in the old file would replace the real `voice`
//     string with a table and corrupt the config this build DOES understand.
//   - ARRAYS OF TABLES ARE NOT FOLLOWED. A path through [[locations]] or
//     [[recent]] is dropped: indices are not stable across a save that
//     reorders them, and copying by index could attach a key to the wrong
//     entry. This is the recorded limitation in NFR-5.
//
// It is one flat loop over the paths with a bounded inner walk — no recursion
// and no reflection (P10-01/P10-09).
func keepUnknown(newDoc, oldDoc map[string]any, paths [][]string) int {
	sort.SliceStable(paths, func(i, j int) bool { return len(paths[i]) < len(paths[j]) })
	kept := 0
	for _, path := range paths {
		if copyPath(newDoc, oldDoc, path) {
			kept++
		}
	}
	return kept
}

// copyPath copies one path's value. It reports whether anything was copied.
func copyPath(newDoc, oldDoc map[string]any, path []string) bool {
	if len(path) == 0 {
		return false
	}
	src, ok := descend(oldDoc, path[:len(path)-1])
	if !ok {
		return false
	}
	value, ok := src[path[len(path)-1]]
	if !ok {
		return false
	}
	dst, ok := makePath(newDoc, path[:len(path)-1])
	if !ok {
		return false
	}
	if _, taken := dst[path[len(path)-1]]; taken {
		return false // a parent table already carried it in
	}
	dst[path[len(path)-1]] = value
	return true
}

// descend walks an existing map along path, refusing to follow anything that
// is not a table (which is what excludes arrays of tables).
func descend(doc map[string]any, path []string) (map[string]any, bool) {
	cur := doc
	for _, seg := range path {
		next, ok := cur[seg].(map[string]any)
		if !ok {
			return nil, false
		}
		cur = next
	}
	return cur, true
}

// makePath walks path in the new document, creating tables as needed, and
// refuses to replace a scalar this build understands with a table.
func makePath(doc map[string]any, path []string) (map[string]any, bool) {
	cur := doc
	for _, seg := range path {
		existing, present := cur[seg]
		if !present {
			next := map[string]any{}
			cur[seg] = next
			cur = next
			continue
		}
		next, ok := existing.(map[string]any)
		if !ok {
			return nil, false // a known scalar (or an array of tables) is in the way
		}
		cur = next
	}
	return cur, true
}

// radioUnknowns renders the unknown [radio.*] keys for [S] (Task 1.11).
//
// Display strings, dotted, and through plaintext.Line because a quoted TOML key
// may carry escapes, control characters or terminal sequences — this is the ONE
// place that rendering happens, so nothing downstream has to remember. Capped,
// and never persisted: Config.Unknown carries no toml tag.
//
// Only [radio.*] keys are listed. An unknown key elsewhere in the file is still
// PRESERVED — that is keepUnknown's job and it is unconditional — but it is not
// this feature's business to explain.
func radioUnknowns(paths [][]string) []string {
	var out []string
	for _, path := range paths {
		if len(path) < 2 || path[0] != "radio" {
			continue
		}
		out = append(out, plaintext.Line(strings.Join(path, ".")))
		if len(out) == maxUnknownReported {
			break
		}
	}
	return out
}
