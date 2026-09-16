import pathlib
# F-113. A `Glyphs` field of a kind the reflection walk cannot pair — here a map —
# is added to the struct. `asciiMarks` skips it SILENTLY, which is the forgotten
# glyph arriving by the exact route the derivation was bought to close: the
# consequence is a missing character on the terminals that cannot render the
# Unicode alternative, invisible to anyone developing on a capable one.
#
# IT MUTATES THE STRUCT, NOT THE GUARD. The guard is a build-time test, and a
# mutation that deleted it could not be caught BY it — the first version of this
# mutant did exactly that and survived, which is the tautology INST/P-1 warns
# about: a test cannot be the instrument and the subject at once.
p = pathlib.Path("platform/render/units.go"); s = p.read_text()
old = """type Glyphs struct {
	Pointer, Play, Repeat, Fire, Alert string"""
new = """type Glyphs struct {
	Lanes map[string]string
	Pointer, Play, Repeat, Fire, Alert string"""
assert old in s, "mGL1"
p.write_text(s.replace(old, new, 1))
