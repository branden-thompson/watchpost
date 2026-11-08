package main

// The A2DH code-authoring rules this tool can decide, and the reason each one is
// decided on the SYNTAX TREE rather than on the text.
//
// A GREP CANNOT TELL A COMMENT FROM A STRING, cannot tell a doc comment from a
// trailing one, and cannot tell WHICH declaration a comment belongs to. The last
// of those is not a nicety: a doc comment that fuses with the one below it binds
// to the wrong declaration and leaves the function above undocumented, and every
// grep in the world reads that as two healthy comments.

import (
	"go/ast"
	"go/token"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Finding is one violation, addressed the way a reader would address it.
type Finding struct {
	Rule string // the catalogue ID
	File string
	Line int
	Text string // the offending line, trimmed
	Why  string // what a reader should do about it
}

// histPhrases is AP-HIST-01's detector.
//
// THE FIRST GROUP IS THE CATALOGUE'S OWN ANTI-SPECIMENS — "used to", "legacy
// compatibility", "the old API", "backward compat". The second is the shape a
// REMEDIATION produces: an account of its own correction, written while the
// correction is what the author has in mind. That group is not in the catalogue
// and is the one this repository generates.
//
// IT MATCHES NARRATION, NOT REASONING. "the boundary errs towards telling the
// listener" is the code as it stands and must pass; "corrected at D-160" is a
// past the reader does not have and must not.
//
// THE WORD BOUNDARY ON `used to` IS LOAD-BEARING: without it "refUSED TO be
// built" is a finding, and the author is asked to rewrite a sentence that
// narrates nothing.
var histPhrases = regexp.MustCompile(`(?i)` + strings.Join([]string{
	`\bused to \w+`,
	`(^|[^a-z])legacy( compatibility|:)`,
	`for backwards? compat`,
	`the old (api|behaviour|behavior|field|name|way|rule)`,
	`an earlier version`,
	`previously (it|this|we|the)`,
	`corrected at [a-z]+-\d+`,
	`(found|caught) by (a |the )?(blind|red[- ]team|newcomer)`,
	`red[- ]?team'?s (round|second|third|final)`,
	`said the opposite`,
	`the first fix`,
	`for a fortnight`,
	`until [a-z]+-\d+ (it|this|the)`,
	`stayed in place after`,
	`was \d+, bumped`,
	`this (used|was) to`,
	// AN ACCOUNT OF THE FIX'S OWN DEFECT, the shape a bug report's comment takes:
	// what the screen showed before the change, and how long it took.
	`\bwas blank\b`,
	`\bwere never (seen|drawn|shown|said)\b`,
	`\bdrew nothing until\b`,
	`\btook half a minute\b`,
	`\bbefore the upgrade\b`,
	`\bbatch \d+('s)? (made|replaced|changed|removed|denser)`,
	// A REVIEW ROUND'S FINDING CITED AS THE REASON: "(round 4, B-07)" means
	// nothing without that round's file, and the sentence states the reason.
	`\bround \d+('s)?,? [A-Z]+\d*-\d+`,
	`\b(was|were) made (denser|twice|coarser)`,
}, "|"))

// histExempt lets a comment say the word without narrating.
//
// `Deprecated:` IS THE LANGUAGE'S OWN DIRECTIVE and the catalogue names it as
// the correct form, so a godoc deprecation is never a finding. A line inside a
// MUTANT is exempt too: those files describe a defect deliberately, and their
// whole purpose is to say what the code would do if it were wrong.
var histExempt = regexp.MustCompile(`(?i)^\s*//\s*Deprecated:`)

// histEmployed is the OTHER sense of "used to", and it is the common one.
//
// "a set USED TO decide what is tracked" is present tense — the thing is
// employed in order to do something. "this USED TO be a field" is narration.
// Go's regexp has no lookbehind, so the employed sense is matched separately and
// wins: a false positive here would train authors to delete a working sentence.
var histEmployed = regexp.MustCompile(`(?i)\b(is|are|was|were|be|been|being|can be|to be|and)\s+used to\b`)

// checkHistory reports comments that narrate the past (AP-HIST-01).
func checkHistory(fset *token.FileSet, f *ast.File, path string) []Finding {
	var out []Finding
	for _, g := range f.Comments { // bounded by the file's comments (P10-02)
		for _, c := range g.List { // bounded by the group (P10-02)
			if !narrates(c.Text) {
				continue
			}
			out = append(out, Finding{
				Rule: "AP-HIST-01", File: path, Line: fset.Position(c.Pos()).Line,
				Text: strings.TrimSpace(c.Text),
				Why:  "state what the code does and why NOW; the change is already in git log and the record documents",
			})
		}
	}
	return out
}

// narrates reports whether a comment narrates the past: it carries one of the
// catalogue's phrases, and is neither a deprecation nor "used to" in its
// employed sense.
func narrates(text string) bool {
	if histExempt.MatchString(text) || !histPhrases.MatchString(text) {
		return false
	}
	return !histEmployed.MatchString(text) || otherHistPhrase(text)
}

// hashCommented reports whether a file's comments are `#` lines: the
// Makefile, shell scripts and .gitignore, where the gates and their reasons
// are written.
func hashCommented(path string) bool {
	base := filepath.Base(path)
	return base == "Makefile" || base == ".gitignore" || strings.HasSuffix(base, ".sh")
}

// checkHashHistory is AP-HIST-01 over a `#`-commented file: each line whose
// first character past the indent is `#`, a script's shebang aside.
func checkHashHistory(src, path string) []Finding {
	var out []Finding
	for n, line := range strings.Split(src, "\n") { // bounded by the file's lines (P10-02)
		t := strings.TrimSpace(line)
		if !strings.HasPrefix(t, "#") || (n == 0 && strings.HasPrefix(t, "#!")) || !narrates(t) {
			continue
		}
		out = append(out, Finding{
			Rule: "AP-HIST-01", File: path, Line: n + 1,
			Text: t,
			Why:  "state what the target or script does and why NOW; the change is already in git log and the record documents",
		})
	}
	return out
}

// checkBlankKeepAlive reports a declaration kept alive by `_ = x` (AP-DEAD-01).
//
// THAT ASSIGNMENT IS HOW UNUSED CODE SURVIVES THE COMPILER, and a checker that
// counts uses reads it as one — which is how a dead closure sat in a window for
// two weeks with a gate watching the file.
//
// ONLY A BARE IDENTIFIER COUNTS. `_ = f()` calls something and may be a
// deliberate discard; `_ = x` names a thing and does nothing with it.
func checkBlankKeepAlive(fset *token.FileSet, f *ast.File, path string) []Finding {
	var out []Finding
	ast.Inspect(f, func(n ast.Node) bool {
		as, ok := n.(*ast.AssignStmt)
		if !ok || len(as.Lhs) != 1 || len(as.Rhs) != 1 {
			return true
		}
		if id, ok := as.Lhs[0].(*ast.Ident); !ok || id.Name != "_" {
			return true
		}
		rhs, ok := as.Rhs[0].(*ast.Ident)
		if !ok || rhs.Name == "_" {
			return true
		}
		// A REASON ON THE LINE IS THE ANSWER THIS RULE ASKS FOR. The finding's own
		// remedy is "delete the declaration, or say why on the line", so a trailing
		// comment is the author having answered it — a fuzz test discarding an
		// expected error, a helper keeping a parameter its caller still passes.
		// Demanding deletion regardless would make the rule unanswerable.
		if trailingComment(fset, f, as.Pos()) {
			return true
		}
		out = append(out, Finding{
			Rule: "AP-DEAD-01", File: path, Line: fset.Position(as.Pos()).Line,
			Text: "_ = " + rhs.Name,
			Why:  "delete the declaration, or use it — suppressing the compiler hides it from every other check too",
		})
		return true
	})
	return out
}

// otherHistPhrase reports whether a comment narrates for a reason other than
// the word "used", so the employed-sense exemption cannot excuse it.
func otherHistPhrase(text string) bool {
	return histOther.MatchString(text)
}

var histOther = regexp.MustCompile(`(?i)` + strings.Join([]string{
	`(^|[^a-z])legacy( compatibility|:)`, `for backwards? compat`,
	`the old (api|behaviour|behavior|field|name|way|rule)`, `an earlier version`,
	`previously (it|this|we|the)`, `corrected at [a-z]+-\d+`,
	`(found|caught) by (a |the )?(blind|red[- ]team|newcomer)`,
	`red[- ]?team'?s (round|second|third|final)`, `said the opposite`,
	`the first fix`, `for a fortnight`, `stayed in place after`, `was \d+, bumped`,
}, "|"))

// checkDocAttached reports a doc comment carrying another declaration's
// contract as well as its own (SN-02).
//
// GO BINDS A CONTIGUOUS COMMENT BLOCK TO THE DECLARATION THAT FOLLOWS IT. Two
// doc comments with no blank line between them are ONE block: the declaration
// above loses its documentation and the one below acquires a contract that
// describes something else. It is invisible in review — both comments are
// present and both read well — and it happened four times in one release here,
// three of them in the commit that fixed the other three.
//
// THE SIGNAL IS WHERE THE SUBJECT IS NAMED. Godoc's convention is that a doc
// comment OPENS by naming its subject, so a block whose "Name does ..." sentence
// arrives on line five has four lines above it that belong to something else.
// Counting two subjects in a block does not work: ordinary prose starts lines
// with words like "and" or "that", and the fusion this exists to catch has only
// one opener — the helper's — with the parent's prose stacked above it.
func checkDocAttached(fset *token.FileSet, f *ast.File, path string) []Finding {
	var out []Finding
	for _, d := range docSubjects(f) { // bounded by the file's declarations (P10-02)
		first, again := ownOpeners(d)
		switch {
		case first > 0:
			out = append(out, Finding{
				Rule: "SN-02", File: path, Line: fset.Position(d.doc.Pos()).Line,
				Text: d.name + "'s doc begins " + itoa(first) + " line(s) in",
				Why:  "the lines above it are another declaration's doc, fused for want of a blank line — separate them so each binds to its own declaration",
			})
		case again > 0:
			out = append(out, Finding{
				Rule: "SN-02", File: path, Line: fset.Position(d.doc.Pos()).Line,
				Text: d.name + "'s doc opens twice, again " + itoa(again) + " line(s) in",
				Why:  "two docs for one declaration, and a reader cannot tell which is true — keep the one that matches the code and delete the other",
			})
		}
	}
	return out
}

// docSubject is a declaration and the doc comment Go binds to it.
type docSubject struct {
	name string
	doc  *ast.CommentGroup
}

// docSubjects are the file's documented declarations: functions, types,
// constants and variables, and the fields of its struct types. A GenDecl's
// doc belongs to its spec when the GenDecl holds just one.
func docSubjects(f *ast.File) []docSubject {
	var out []docSubject
	add := func(names []*ast.Ident, doc *ast.CommentGroup) {
		if doc != nil && len(doc.List) >= 2 && len(names) > 0 {
			out = append(out, docSubject{name: names[0].Name, doc: doc})
		}
	}
	for _, d := range f.Decls { // bounded by the file's declarations (P10-02)
		switch x := d.(type) {
		case *ast.FuncDecl:
			add([]*ast.Ident{x.Name}, x.Doc)
		case *ast.GenDecl:
			for _, sp := range x.Specs { // bounded by the declaration's specs (P10-02)
				add(specNames(sp), specDoc(sp, x))
			}
		}
	}
	ast.Inspect(f, func(n ast.Node) bool {
		if st, ok := n.(*ast.StructType); ok {
			for _, fl := range st.Fields.List { // bounded by the struct's fields (P10-02)
				add(fl.Names, fl.Doc)
			}
		}
		return true
	})
	return out
}

// specNames are the names a type or value spec declares.
func specNames(sp ast.Spec) []*ast.Ident {
	switch x := sp.(type) {
	case *ast.TypeSpec:
		return []*ast.Ident{x.Name}
	case *ast.ValueSpec:
		return x.Names
	}
	return nil
}

// specDoc is the doc Go binds to a spec: its own, or its GenDecl's when the
// GenDecl holds only it.
func specDoc(sp ast.Spec, g *ast.GenDecl) *ast.CommentGroup {
	var own *ast.CommentGroup
	switch x := sp.(type) {
	case *ast.TypeSpec:
		own = x.Doc
	case *ast.ValueSpec:
		own = x.Doc
	}
	if own == nil && len(g.Specs) == 1 {
		return g.Doc
	}
	return own
}

// ownOpeners finds where a doc names its own subject in godoc's opening
// form after line 0: first when line 0 does not, again when it does. -1 means
// none.
//
// A LINE THAT STARTS WITH THE NAME IS NOT ALWAYS AN OPENER. Prose wraps, so
// "a nil" can end one line and "frameChecks checks every frame" begin the
// next; and a new paragraph may open with the name, as godoc's own examples
// do. A later line counts when the one above it ends a sentence and is not a
// paragraph break, or when the block opens by naming another declaration,
// which is a fused doc whatever follows.
func ownOpeners(d docSubject) (first, again int) {
	first, again = -1, -1
	lines := d.doc.List
	m := godocOpener.FindStringSubmatch(lines[0].Text)
	own := m != nil && m[1] == d.name
	foreign := m != nil && !own
	for n := 1; n < len(lines); n++ { // bounded by the block (P10-02)
		m := godocOpener.FindStringSubmatch(lines[n].Text)
		if m == nil || m[1] != d.name || !foreign && !endsSentence(lines[n-1].Text) {
			continue
		}
		if own {
			return -1, n
		}
		return n, -1
	}
	return first, again
}

// endsSentence reports whether a comment line ends a sentence: it carries
// words, and its last one ends in a full stop, a colon, or a question or
// exclamation mark.
func endsSentence(line string) bool {
	t := strings.TrimSpace(strings.TrimPrefix(line, "//"))
	return t != "" && strings.ContainsAny(t[len(t)-1:], ".:!?")
}

// packageDoc is a non-test file's package comment: the package it is in
// (its directory and name) and the file.
type packageDoc struct {
	pkg, file string
}

// packageDocOf is the file's package comment, if it has one. A test file's
// does not count: go doc reads only the package's own files.
func packageDocOf(f *ast.File, rel string) (packageDoc, bool) {
	if f.Doc == nil || strings.HasSuffix(rel, "_test.go") {
		return packageDoc{}, false
	}
	return packageDoc{pkg: path.Dir(rel) + " " + f.Name.Name, file: rel}, true
}

// checkPackageDocs reports a package with more than one package comment
// (SN-03). go doc joins them all, in file order, so a package's front door
// reads as whatever fragments its files carry; the one doc belongs in one
// file, and the rest become file headers, a blank line above `package`.
func checkPackageDocs(docs map[string][]packageDoc) []Finding {
	var out []Finding
	for _, ds := range docs { // bounded by the tree's packages (P10-02)
		if len(ds) < 2 {
			continue
		}
		files := make([]string, 0, len(ds))
		for _, d := range ds { // bounded by the package's files (P10-02)
			files = append(files, path.Base(d.file))
		}
		sort.Strings(files)
		for _, d := range ds { // bounded by the package's files (P10-02)
			out = append(out, Finding{
				Rule: "SN-03", File: d.file, Line: 1,
				Text: "one of " + itoa(len(ds)) + " package comments in " + path.Dir(d.file) + " (" + strings.Join(files, ", ") + ")",
				Why:  "go doc joins every package comment into one — keep the package's doc in one file and make the others file headers, with a blank line above `package`",
			})
		}
	}
	return out
}

// itoa keeps the finding's text free of a fmt dependency in this file.
func itoa(n int) string { return strconv.Itoa(n) }

// godocOpener matches the convention a doc comment opens with: the subject's own
// name, then a present-tense verb ("is", "has", or any word ending in s).
var godocOpener = regexp.MustCompile(`^//\s+([A-Za-z_]\w*) (?:is|are|has|does|[a-z]+s)\b`)

// trailingComment reports whether a comment sits on the same line as pos — the
// author's reason, written where the rule asks for it.
func trailingComment(fset *token.FileSet, f *ast.File, pos token.Pos) bool {
	line := fset.Position(pos).Line
	for _, g := range f.Comments { // bounded by the file's comments (P10-02)
		for _, c := range g.List { // bounded by the group (P10-02)
			if fset.Position(c.Pos()).Line == line {
				return true
			}
		}
	}
	return false
}

// checkShell reports Go that carries a program in shell (AP-SHELL-01): a string
// literal that is a shebang line FOLLOWED BY A BODY (a shebang alone is a file
// header the oracle's fixtures write), and `exec.Command("sh"|"bash", …, "-c", …)`.
// Everything is Go unless absolutely necessary, and "necessary" is a ruling,
// not a comment — so there is no exemption marker.
func checkShell(fset *token.FileSet, f *ast.File, path string) []Finding {
	var out []Finding
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.BasicLit:
			if x.Kind == token.STRING && shellProgram(x.Value) {
				out = append(out, Finding{
					Rule: "AP-SHELL-01", File: path, Line: fset.Position(x.Pos()).Line,
					Text: "a shell program in a Go string literal",
					Why:  "write it in Go; a stub is a small tools/ main built at test time, a checker is a tools/ main",
				})
			}
		case *ast.CallExpr:
			if shellDashC(x) {
				out = append(out, Finding{
					Rule: "AP-SHELL-01", File: path, Line: fset.Position(x.Pos()).Line,
					Text: "exec.Command runs a shell with -c",
					Why:  "run the program directly with exec.Command(prog, args...), or write it in Go",
				})
			}
		}
		return true
	})
	return out
}

// shellProgram: the literal's first line is a shebang and at least one more
// non-blank line follows it.
func shellProgram(lit string) bool {
	body, err := strconv.Unquote(lit)
	if err != nil {
		return false
	}
	lines := strings.Split(body, "\n")
	if !strings.HasPrefix(lines[0], "#!") {
		return false
	}
	for _, l := range lines[1:] { // bounded by the literal (P10-02)
		if strings.TrimSpace(l) != "" {
			return true
		}
	}
	return false
}

// shellDashC: exec.Command(<"sh"|"bash" literal>, …, "-c", …).
func shellDashC(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Command" || len(call.Args) < 2 {
		return false
	}
	if id, ok := sel.X.(*ast.Ident); !ok || id.Name != "exec" {
		return false
	}
	prog, ok := call.Args[0].(*ast.BasicLit)
	if !ok || (prog.Value != `"sh"` && prog.Value != `"bash"`) {
		return false
	}
	for _, a := range call.Args[1:] { // bounded by the arguments (P10-02)
		if lit, ok := a.(*ast.BasicLit); ok && lit.Value == `"-c"` {
			return true
		}
	}
	return false
}
