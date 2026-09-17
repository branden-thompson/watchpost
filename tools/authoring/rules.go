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
	"regexp"
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
			if histExempt.MatchString(c.Text) || !histPhrases.MatchString(c.Text) {
				continue
			}
			if histEmployed.MatchString(c.Text) && !otherHistPhrase(c.Text) {
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
	for _, d := range f.Decls { // bounded by the file's declarations (P10-02)
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Doc == nil || len(fn.Doc.List) < 2 {
			continue
		}
		at := -1
		for n, c := range fn.Doc.List { // bounded by the block (P10-02)
			m := godocOpener.FindStringSubmatch(c.Text)
			if m != nil && m[1] == fn.Name.Name {
				at = n
				break
			}
		}
		if at <= 0 {
			continue // opens with its own name, or never names itself: not this rule
		}
		out = append(out, Finding{
			Rule: "SN-02", File: path, Line: fset.Position(fn.Doc.Pos()).Line,
			Text: fn.Name.Name + "'s doc begins " + itoa(at) + " line(s) in",
			Why:  "the lines above it are another declaration's doc, fused for want of a blank line — separate them so each binds to its own declaration",
		})
	}
	return out
}

// itoa keeps the finding's text free of a fmt dependency in this file.
func itoa(n int) string { return strconv.Itoa(n) }

// godocOpener matches the convention a doc comment opens with: the subject's own
// name, then a verb.
var godocOpener = regexp.MustCompile(`^//\s+([A-Za-z]\w*) (?:is|are|does|reports|returns|builds|draws|takes|copies|names|decides|holds|walks|turns|gives|answers|removes|routes|hands|puts|says|sets|tells|wraps|adds|counts)\b`)

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
