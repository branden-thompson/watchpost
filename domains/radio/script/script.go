// Package script is the library of spoken report scripts — what the
// correspondent says, kept as editable text, not as strings in the app
// (HUM LEAD 2026-08-28: "configurable, extendable, editable; picked up by
// file name, not wired").
//
// Convention — the tree IS the wiring:
//
//	scripts/<report>/<part>.txt   one phrase of one report
//	scripts/global/<part>.txt     the phrase any report inherits when it has none of its own (head, tail)
//
// A file is a text/template; its data is whatever the caller passes (the
// fields are documented in each file's first comment line, "{{/* … */}}").
// The app asks for text by (report, part) — Library.Text("event-report",
// "head", data) — and never names a file: the report's own file is used,
// else global's, else the part is not spoken. Adding a phrase is adding a
// file; adding a report is adding a folder.
//
// Overrides: the same tree under the user's config dir, scripts/ — a file
// there replaces the built-in one, phrase by phrase, the same names and the
// same data.
package script

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"text/template"

	"github.com/branden-thompson/watchpost/platform/invariant"
	"github.com/branden-thompson/watchpost/platform/render"
)

//go:embed scripts
var builtin embed.FS

const (
	root   = "scripts" // the embedded tree's root
	Ext    = ".txt"    // the script file extension
	Global = "global"  // the report every other report inherits missing parts from
)

// ErrNoScript is returned for a (report, part) no file names.
var ErrNoScript = errors.New("script: no such report part")

// nameRE is the name convention for a report or a part: lower-case words
// and hyphens.
var nameRE = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// Library resolves report parts to text: the built-in tree, overridden by
// the same tree under an optional directory.
type Library struct {
	override string // "" = built-in only
	mu       sync.Mutex
	cache    map[string]*template.Template
}

// New builds a library; overrideDir may be "" (built-in scripts only) or a
// directory whose <report>/<part>.txt files replace the built-in ones.
func New(overrideDir string) *Library {
	return &Library{override: overrideDir, cache: map[string]*template.Template{}}
}

// Text renders one part of a report with data: the report's own phrase,
// else global's. ErrNoScript when neither exists; a template error is
// returned as is (the caller decides whether silence or a fallback is right).
func (l *Library) Text(report, part string, data any) (string, error) {
	if !nameRE.MatchString(report) || !nameRE.MatchString(part) {
		return "", fmt.Errorf("%w: %q/%q is not <report>/<part>", ErrNoScript, report, part)
	}
	t, err := l.load(report, part)
	if errors.Is(err, ErrNoScript) && report != Global {
		t, err = l.load(Global, part)
	}
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	if err := t.Execute(&b, data); err != nil {
		return "", fmt.Errorf("script %s/%s: %w", report, part, err)
	}
	return strings.TrimSpace(b.String()), nil // data is never re-parsed: "{{" in provider prose is prose (round 4, A-09)
}

// Say is Text for the air: the rendered part with any terminal escapes
// stripped (provider prose never addresses the terminal — NFR-6), or "" when
// the part is missing or broken — a sentence of silence rather than a crash;
// the built-in tree is pinned by this package's tests, so that only happens
// with a broken override. A nil Library speaks from the built-in tree (a
// fresh loader — the deck and the ticker hold a real one; nil is the tests'
// and a memo would be a package global, P10-06). The one owner of "missing
// script = silence" (round 4, A-13).
func (l *Library) Say(report, part string, data any) string {
	if l == nil {
		l = New("")
	}
	s, err := l.Text(report, part, data)
	if err != nil {
		return ""
	}
	return render.Plain(s)
}

// Has reports whether a (report, part) resolves — the report's own file or
// global's.
func (l *Library) Has(report, part string) bool {
	if !nameRE.MatchString(report) || !nameRE.MatchString(part) {
		return false
	}
	if _, err := l.load(report, part); err == nil {
		return true
	}
	if report == Global {
		return false
	}
	_, err := l.load(Global, part)
	return err == nil
}

// Reports lists the report folders on file (built-in and overrides), sorted;
// global among them.
func (l *Library) Reports() []string {
	seen := map[string]bool{}
	for _, n := range l.files() {
		seen[filepath.Dir(n)] = true
	}
	return sortedKeys(seen)
}

// Parts lists a report's own parts on file, sorted (global's are not
// repeated here — Has answers for the inherited ones).
func (l *Library) Parts(report string) []string {
	seen := map[string]bool{}
	for _, n := range l.files() {
		if filepath.Dir(n) == report {
			seen[strings.TrimSuffix(filepath.Base(n), Ext)] = true
		}
	}
	return sortedKeys(seen)
}

// load parses (and caches) the template for a part: the override file when
// present, else the built-in.
func (l *Library) load(report, part string) (*template.Template, error) {
	if !nameRE.MatchString(report) || !nameRE.MatchString(part) {
		return nil, fmt.Errorf("%w: %q/%q is not <report>/<part>", ErrNoScript, report, part)
	}
	name := report + "/" + part + Ext
	l.mu.Lock()
	defer l.mu.Unlock()
	if t, ok := l.cache[name]; ok {
		return t, nil
	}
	src, err := l.read(name)
	if err != nil {
		return nil, err
	}
	t, err := template.New(name).Option("missingkey=error").Parse(string(src))
	if err != nil {
		return nil, fmt.Errorf("script %s: %w", name, err)
	}
	l.cache[name] = t
	return t, nil
}

// read finds the file: the override tree first, then the built-in tree.
func (l *Library) read(name string) ([]byte, error) {
	if err := invariant.Check(!strings.Contains(name, "..") && strings.Count(name, "/") == 1, "a script name is <report>/<part> and never leaves the tree"); err != nil {
		return nil, err
	}
	if l.override != "" {
		path := filepath.Join(l.override, filepath.FromSlash(name))
		// A regular file in the override tree only: a symlink could read any
		// file the user can into the voice (round 4, A-10).
		if fi, err := os.Lstat(path); err == nil && fi.Mode().IsRegular() {
			if fi.Size() > maxOverrideBytes { // a phrase, not a file dump: a 20 MB tail would be a 20 MB spoken line (R5-C-15)
				return nil, fmt.Errorf("script %s: override exceeds %d bytes", name, maxOverrideBytes)
			}
			if b, err := os.ReadFile(path); err == nil {
				return b, nil
			}
		}
	}
	b, err := fs.ReadFile(builtin, root+"/"+name)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrNoScript, name)
	}
	return b, nil
}

// maxOverrideBytes bounds one override part: the longest built-in part is
// under a kilobyte; 64 KB leaves room for a long instruction block.
const maxOverrideBytes = 64 << 10

// files lists every script file on file as "<report>/<part>.txt", built-in
// and override, without duplicates.
func (l *Library) files() []string {
	seen := map[string]bool{}
	for _, n := range BuiltinFiles() {
		seen[n] = true
	}
	if l.override != "" {
		dirs, _ := os.ReadDir(l.override)
		for _, d := range dirs {
			if !d.IsDir() || !nameRE.MatchString(d.Name()) {
				continue
			}
			entries, _ := os.ReadDir(filepath.Join(l.override, d.Name()))
			for _, e := range entries {
				if part, ok := strings.CutSuffix(e.Name(), Ext); ok && nameRE.MatchString(part) {
					seen[d.Name()+"/"+e.Name()] = true
				}
			}
		}
	}
	return sortedKeys(seen)
}

// BuiltinFiles lists the built-in script files as "<report>/<part>.txt"
// (the convention's test and the README's listing).
func BuiltinFiles() []string {
	var out []string
	_ = fs.WalkDir(builtin, root, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			out = append(out, strings.TrimPrefix(p, root+"/"))
		}
		return nil
	})
	sort.Strings(out)
	return out
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
