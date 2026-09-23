// Package trees defines the "internal project tree" identity class: the names
// of the private workspace this repository is developed in, which a public tree
// must never carry.
//
// IT WAS A LIST OF TWO DIRECTORY NAMES, and one reorganisation of the workspace
// moved every internal repository out from under both: a path in the new layout
// passed every gate while the gates still refused a tree that no longer existed.
// A list of names protects the layout it was written on. So the class is
// described two ways, and every consumer asks this package for both:
//
//   - STATIC shapes that hold on any machine, CI included: the names retired by a
//     reorganisation, and the numbered-bucket convention the workspace uses.
//   - DERIVED names read from the disk at run time: the workspace root, every
//     directory between it and the checkout, and every tree beside them. A
//     renamed workspace is described by its new names on the next run, with
//     nobody editing a list. Where the checkout does not sit in such a
//     workspace (a CI runner) this half is empty and the static half applies.
//
// Every expression stays inside the subset POSIX ERE, Go RE2 and Python `re`
// read identically, because lint-ledger.sh and p10-ledger-mirror.py apply the
// same string: groups, alternation, bracket classes, {n}. No \b, no \d.
package trees

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// static are the shapes of the class that need no workspace to evaluate.
var static = []string{
	// Names retired by a reorganisation. Notes written before the move still
	// carry them, and text copied from those notes leaks the old tree as well.
	`LI_PROJECTS|DESIGN_FOUNDATIONS`,
	// Numbered bucket folders: 01__SOME_BUCKET, 02__TEAM_TOOLS.
	`[0-9]{2}__[A-Z][A-Za-z0-9_]+`,
	// Dash-numbered folders used as a path segment: 09-ARCHIVE/.
	`(^|[^A-Za-z0-9._-])[0-9]{2}-[A-Z][A-Za-z0-9_-]*/`,
}

// segment anchors a derived name as a PATH SEGMENT ("NAME/"): a leak is a
// path, and the bare word may be ordinary prose.
const segment = `(^|[^A-Za-z0-9._-])(%s)/`

// minNameLen drops names too short to mean anything on their own.
const minNameLen = 3

// bucketShape is the shape a derived name must have: the workspace's own
// convention of shouting names - capitals, digits, underscores and dashes.
//
// LENGTH ALONE IS NOT A FILTER. A workspace whose directories are named
// `docs`, `domains` or `platform` - all real top-level directories of this
// repository - would otherwise put those words in the rule, and the gate would
// refuse hundreds of clean files on one developer's machine and none on
// another's. The only remedy left would be exemption rows, which is what this
// package exists to avoid. A name that does not shout is not a workspace
// bucket, and the static half still covers the conventions that do.
var bucketShape = regexp.MustCompile(`^[A-Z0-9][A-Z0-9_-]*$`)

// Expr returns the one expression for the class, for a checkout at repoRoot
// under the home directory home. An empty or unrelated home yields the static
// shapes alone.
func Expr(repoRoot, home string) (string, error) {
	// The empty-root refusal lives in Derived, which this calls on the next
	// line: a second copy here is a check no test can make fail.
	names, err := Derived(repoRoot, home)
	if err != nil {
		return "", err
	}
	expr := strings.Join(static, "|")
	if len(names) > 0 {
		quoted := make([]string, 0, len(names))
		for _, n := range names { // bounded by the derived names
			quoted = append(quoted, regexp.QuoteMeta(n))
		}
		// Replace, not Sprintf: a directory name may contain a per-cent sign,
		// and Sprintf would read it as a verb.
		expr += "|" + strings.Replace(segment, "%s", strings.Join(quoted, "|"), 1)
	}
	return expr, nil
}

// Derived returns the workspace names around a checkout, sorted and unique.
//
// THE WORKSPACE ROOT is the second directory below home on the way to the
// checkout ($HOME/<first level>/<root>/...). The first level (Documents, src)
// is left out on purpose: its neighbours are generic words, and a gate that
// fires on "Library/" teaches authors to add exemptions. The checkout's own
// name is left out too — the repository may name itself.
func Derived(repoRoot, home string) ([]string, error) {
	if repoRoot == "" {
		return nil, errors.New("trees: no repository root")
	}
	if home == "" {
		return nil, nil
	}
	root, err := resolve(repoRoot)
	if err != nil {
		return nil, err
	}
	// AN UNREADABLE HOME IS AN ERROR, NOT AN EMPTY RULE. A gate handed the
	// static half alone scans with a whole class missing and still prints a
	// pass, which is the fail-open shape this package exists to remove. An
	// empty HOME is different and means "derive nothing", above: that is a
	// caller saying there is no workspace, not a disk refusing to answer.
	base, err := resolve(home)
	if err != nil {
		return nil, err
	}
	rel, err := filepath.Rel(base, root)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return nil, nil
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) < 3 {
		return nil, nil
	}
	repo := parts[len(parts)-1]
	names := append([]string{}, parts[1:len(parts)-1]...)
	// The workspace root's own children - the trees that sit beside this
	// checkout, which is what a leak would name.
	children, err := os.ReadDir(filepath.Join(base, parts[0], parts[1]))
	if err != nil {
		return nil, err
	}
	for _, e := range children { // bounded by the workspace's entries
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return keep(names, repo), nil
}

// keep filters, sorts and de-duplicates the derived names.
func keep(names []string, repo string) []string {
	seen := make(map[string]bool, len(names))
	out := make([]string, 0, len(names))
	for _, n := range names { // bounded by the derived names
		if len(n) < minNameLen || n == repo || seen[n] || !bucketShape.MatchString(n) {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// resolve returns the absolute, symlink-free form of dir, so a home reached
// through a link (macOS /var -> /private/var) still contains its checkouts.
func resolve(dir string) (string, error) {
	if dir == "" {
		return "", errors.New("trees: empty path")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(abs)
}
