package tty

// memo_positions_test.go — 0.19.0 W9.1 (FR-8.1, #12): a memo key's field
// that is a position - an index, an offset, a row - carries why a position
// is enough there: the identity of what it indexes is keyed beside it.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strings"
	"testing"
)

// positionName is a field named as a position; only an integer one is.
var positionName = regexp.MustCompile(`^(selected|scroll)$|(Idx|Row|At|Off|Focus)$`)

// unreasonedPositions are the position fields of a file's named structs
// whose comment gives no "position:" reason, and how many position fields
// there were.
func unreasonedPositions(t *testing.T, file string, src any, keys map[string]bool) (missing []string, seen int) {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), file, src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	ast.Inspect(f, func(n ast.Node) bool {
		spec, ok := n.(*ast.TypeSpec)
		if !ok || !keys[spec.Name.Name] {
			return true
		}
		st, ok := spec.Type.(*ast.StructType)
		if !ok {
			return false
		}
		for _, field := range st.Fields.List {
			reason := strings.Contains(field.Comment.Text()+field.Doc.Text(), "position:")
			typ, _ := field.Type.(*ast.Ident)
			integer := typ != nil && (typ.Name == "int" || typ.Name == "int64")
			for _, name := range field.Names {
				if !integer || !positionName.MatchString(name.Name) {
					continue
				}
				seen++
				if !reason {
					missing = append(missing, spec.Name.Name+"."+name.Name)
				}
			}
		}
		return false
	})
	return missing, seen
}

// TestEveryPositionFieldHasAReason is FR-8.1 (#12): every position field of
// every window memo's key says why a position is enough - "position:" and
// what keys the identity of what it indexes.
func TestEveryPositionFieldHasAReason(t *testing.T) {
	seen := 0
	for file, keys := range map[string]map[string]bool{"memo.go": {"bodyKey": true, "modalKey": true}, "broadcaster_memo.go": {"consoleKey": true}} {
		missing, n := unreasonedPositions(t, file, nil, keys)
		seen += n
		if len(missing) > 0 {
			t.Errorf("%s: position fields with no reason: %v", file, missing)
		}
	}
	if seen < 12 {
		t.Errorf("%d position fields found in the memo keys; the audit found 13 - the reading is broken", seen)
	}
}

// TestAPositionWithoutAReasonIsCaught is FR-8.1's planted gap: a position
// field with no reason is named; with one, it passes; a field that is no
// position needs none.
func TestAPositionWithoutAReasonIsCaught(t *testing.T) {
	src := "package tty\n\ntype k struct {\n\tselected int\n\twidth int\n\tmapsOff bool\n\trowAt int // position: a fixed row\n}\n"
	missing, seen := unreasonedPositions(t, "planted.go", src, map[string]bool{"k": true})
	if seen != 2 || len(missing) != 1 || missing[0] != "k.selected" {
		t.Errorf("planted: %d positions, missing %v", seen, missing)
	}
}
