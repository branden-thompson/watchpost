import pathlib
# The helper under the location field loses Observer's caveat tone and reads as
# ordinary support text. "Location not found in Pool" then looks like a label
# rather than a reason the request cannot be made — and the operator's eye has
# nothing to catch on in a window whose other lines are all instructions.
p = pathlib.Path("modes/tty/request.go"); s = p.read_text()
old = '''		out = append(out, "  "+o.Glyphs().Alert+" "+render.Tint(fact, render.Tok(render.NameWarning)))'''
assert old in s, "mBB1"
p.write_text(s.replace(old, '''		out = append(out, "  "+o.Glyphs().Alert+" "+fact)''', 1))
