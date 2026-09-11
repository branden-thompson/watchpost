import pathlib
# The card stops drawing the masthead's box and gets a set of its own, so the
# console and the header disagree about what a border looks like — which is the
# state D-85 closed, and the reason the glyph set carried four corner marks with
# exactly one user.
#
# RE-ANCHORED the same day: the `dupes` gate collapsed HeavyBox and LightBox onto
# one builder, so the marks are ARGUMENTS now rather than a literal body — which
# is a better anchor, because it is the marks themselves rather than the shape of
# the function around them.
p = pathlib.Path("platform/render/panel.go"); s = p.read_text()
old = '\treturn boxGlyphs(ascii, "\u250f", "\u2513", "\u2517", "\u251b", "\u2501", "\u2503")'
new = '\treturn boxGlyphs(ascii, "\u256d", "\u256e", "\u2570", "\u256f", "\u2500", "\u2502")'
assert old in s, "mQ1"
p.write_text(s.replace(old, new, 1))
