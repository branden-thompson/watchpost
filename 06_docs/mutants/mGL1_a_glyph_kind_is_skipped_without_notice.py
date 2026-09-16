import pathlib
# F-113. The glyph-pairing guard stops checking that every `Glyphs` field is a
# kind the reflection walk can pair, so a field it cannot handle is skipped
# silently — the forgotten glyph arriving by the exact route the derivation was
# bought to close. The consequence is a missing character on the terminals that
# cannot render the Unicode alternative, which is invisible to anyone developing
# on a capable one.
p = pathlib.Path("platform/render/glyph_pairing_test.go"); s = p.read_text()
old = """		case reflect.String, reflect.Array:
		default:"""
new = """		case reflect.String, reflect.Array:
		case reflect.Invalid:"""
assert old in s, "mGL1"
p.write_text(s.replace(old, new, 1))
