package main

import (
	"go/types"
	"path"
	"slices"
	"strings"
)

// area is a part of the code whose reach into the model is counted: its
// readers (non-test files whose path starts with one of the prefixes, in the
// directory named), the files whose declarations are the model, and the
// carrier types whose every field it reads counts as reach.
//
// A BATCH THAT MOVES AN AREA'S FILES EDITS ITS ROW: the prefixes follow the
// files, and a carrier the move adds (the map's Env and Hooks, B4) joins the
// row in the same commit, so the count is of the same thing before and after.
type area struct {
	name     string
	readers  []string // path prefixes of the area's non-test files
	model    []string // the model's files
	carriers []string // carrier type names, "pkgpath.Name"
}

// modelFiles are the files whose declarations JN's P2-3 calls the model.
var modelFiles = []string{"modes/tty/dashboard.go", "modes/tty/dashboard_choices.go"}

// areas are M3's reach figures (JN P2-3, MS-9, CK), keyed as 0.19.6 D-44 rules.
var areas = []area{
	{name: "map reach (symbols of the model's files)", readers: []string{"modes/tty/map_"}, model: modelFiles},
	{name: "Settings reach (symbols of the model's files)", readers: []string{"modes/tty/setup"}, model: modelFiles},
	{name: "console reach (symbols of dashboard.go)", readers: []string{"modes/tty/console/broadcaster"}, model: []string{"modes/tty/dashboard.go"}},
}

// byType is a reach counted by type: members of the model's types, declared
// outside the area's own files, read by the area (MS-9's 93).
type byType struct {
	readers []string
	types   []string
}

// mapByType is the map's reach by type (MS-9).
var mapByType = byType{readers: []string{"modes/tty/map_"}, types: []string{"Dashboard", "Config", "mapChoices", "uiChoices", "radioChoices", "setupState"}}

// reads says whether a file is one of an area's non-test readers. A prefix
// names files in its own directory only: modes/tty/map_ is not
// modes/tty/mapview/x.go.
func reads(readers []string, file string) bool {
	if strings.HasSuffix(file, "_test.go") {
		return false
	}
	for _, r := range readers { // bounded by the area's prefixes (P10-02)
		if strings.HasPrefix(file, r) && path.Dir(file) == path.Dir(r+"x") {
			return true
		}
	}
	return false
}

// reach is the distinct model symbols an area reads: those declared in the
// model's files, and every field of a carrier type. A field is keyed by its
// bare name and anything else as itself, as JN's table keyed them (0.19.6 D-44):
// Config.Theme and UIPrefs.Theme are one reach.
func (ix *index) reach(a area) map[string]bool {
	out := map[string]bool{}
	for _, u := range ix.uses { // bounded by the uses (P10-02)
		if !reads(a.readers, u.file) {
			continue
		}
		if slices.Contains(a.model, u.decl) || ix.carried(a, u.obj) {
			out[reachKey(u.obj, u.decl)] = true
		}
	}
	return out
}

// reachKey is a symbol's key in a reach figure (0.19.6 D-44): ".name" for a field,
// and for anything else its name where it is declared.
func reachKey(o types.Object, decl string) string {
	if v, ok := o.(*types.Var); ok && v.IsField() {
		return "." + v.Name()
	}
	return decl + ":" + o.Name()
}

// carried says whether an object is a field of one of the area's carriers.
func (ix *index) carried(a area, o types.Object) bool {
	v, ok := o.(*types.Var)
	if !ok || !v.IsField() || v.Pkg() == nil {
		return false
	}
	return slices.Contains(a.carriers, v.Pkg().Path()+"."+ix.fieldOwner[v])
}

// reachByType is the distinct members of the types an area reads, those
// declared in the area's own files left out.
func (ix *index) reachByType(b byType) map[types.Object]bool {
	out := map[types.Object]bool{}
	for _, u := range ix.uses { // bounded by the uses (P10-02)
		if reads(b.readers, u.file) && slices.Contains(b.types, ix.ownerOf(u.obj)) && !reads(b.readers, u.decl) {
			out[u.obj] = true
		}
	}
	return out
}

// isDashboard says whether an object is the hub's type or one of its members.
func (ix *index) isDashboard(o types.Object) bool {
	if tn, ok := o.(*types.TypeName); ok {
		return tn.Name() == "Dashboard" && tn.Pkg().Path() == module+"/modes/tty"
	}
	return ix.ownerOf(o) == "Dashboard" && o.Pkg().Path() == module+"/modes/tty"
}

// fanIn is the non-test files, in any package, that use the hub's type or a
// member of it, or declare a method on it (0.19.6 D-41: across packages).
func (ix *index) fanIn() int {
	files := map[string]bool{}
	for _, u := range ix.uses { // bounded by the uses (P10-02)
		if !u.test && ix.isDashboard(u.obj) {
			files[u.file] = true
		}
	}
	for f := range ix.methodDecl[hub] { // bounded by the method files (P10-02)
		files[f] = true
	}
	return len(files)
}

// dashboardMethods is the methods declared on Dashboard or *Dashboard, and
// the files that declare them.
func (ix *index) dashboardMethods() (methods, files int) {
	for _, n := range ix.methodDecl[hub] { // bounded by the method files (P10-02)
		methods += n
	}
	return methods, len(ix.methodDecl[hub])
}

// hub is the hub's type, as methodDecl keys it.
const hub = module + "/modes/tty.Dashboard"
