package render

import (
	"reflect"
	"testing"
)

// EVERY GLYPH FIELD CAN BE PAIRED WITH ITS ASCII FALLBACK (F-113).
//
// `asciiMarks` derives the Unicode -> ASCII replacer by walking `Glyphs` with
// reflection, precisely so a glyph added to one set and forgotten in the other
// cannot slip through. Its switch handles `String` and `Array` — and a field of
// any OTHER kind was silently skipped, which is the forgotten glyph arriving by
// the one route the derivation exists to close.
//
// THE CHECK IS HERE RATHER THAN IN A `default` ARM because the walk runs on a
// render path through `sync.OnceValue`, which has no error channel: the only
// runtime answer available there is a panic, in front of a listener. The set of
// field kinds is fixed by the struct, so a test sees all of it and fails at
// build time instead.
//
// IF THIS FAILS, ADD THE CASE IN BOTH PLACES: the pairing arm in `asciiMarks`
// AND the accepted-kinds list below, which is the set of kinds the walk can
// pair. The
// consequence of not doing so is a missing character on exactly the terminals
// that cannot render the Unicode alternative, which is invisible to anyone
// developing on a capable one.
func TestEveryGlyphFieldCanBePaired(t *testing.T) {
	g := reflect.ValueOf(Opts{}.Glyphs())
	if g.NumField() == 0 {
		t.Fatal("Glyphs has no fields; the pairing walk has lost its subject")
	}
	for i := range g.NumField() {
		f := g.Type().Field(i)
		switch g.Field(i).Kind() {
		case reflect.String, reflect.Array:
		default:
			t.Errorf("Glyphs.%s is a %s, which asciiMarks cannot pair — it is skipped "+
				"silently, so this glyph has no --ascii fallback and renders as a missing "+
				"character on the terminals that cannot show the Unicode one. "+
				"Add a case to asciiMarks.", f.Name, g.Field(i).Kind())
		}
	}
}
