package main

import (
	"fmt"
	"os"
	"path"
	"slices"
	"strings"
)

// planned is the planned package of each app file and of each
// livePipelines method, a method row overriding its file (0.19.6 A-23,
// design AP; app/testdata/sharedlock/planned.tsv).
type planned struct {
	file, method map[string]string
}

// readPlanned reads the planned map: "file<TAB>path<TAB>package" and
// "method<TAB>Type.Name<TAB>package" rows.
func readPlanned(p string) (*planned, error) {
	raw, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	pl := &planned{file: map[string]string{}, method: map[string]string{}}
	for _, line := range strings.Split(string(raw), "\n") { // bounded by the file (P10-02)
		f := strings.Split(line, "\t")
		if len(f) != 3 {
			continue
		}
		switch f[0] {
		case "file":
			pl.file[f[1]] = f[2]
		case "method":
			pl.method[f[1]] = f[2]
		}
	}
	if len(pl.file) == 0 {
		return nil, fmt.Errorf("%s plans no file: it is not the planned map", p)
	}
	return pl, nil
}

// crossings is every use, in a non-test app file, of an unexported object
// whose planned package differs from the using file's: each one a name the
// move must export, re-home or pass. A method's planned package is its own
// row's when it has one. Each crossing is listed once: user file, object,
// its file.
func (ix *index) crossings(pl *planned) []string {
	set := map[string]bool{}
	for _, u := range ix.uses { // bounded by the uses (P10-02)
		if u.test || u.obj.Exported() || !strings.HasPrefix(u.file, "app/") || !strings.HasPrefix(u.decl, "app/") {
			continue
		}
		from, to := pl.file[u.file], pl.file[u.decl]
		if m, ok := pl.method[u.encl]; ok {
			from = m
		}
		if m, ok := pl.method[ix.symName(u.obj)]; ok {
			to = m
		}
		if from == "" || to == "" || from == to || path.Ext(u.decl) != ".go" {
			continue
		}
		set[u.file+"\t"+ix.symName(u.obj)+"\t"+u.decl] = true
	}
	out := make([]string, 0, len(set))
	for k := range set { // bounded by the crossings (P10-02)
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}
