import pathlib
# The helper under the location field loses Observer's caveat tone and reads as
# ordinary support text. "Outside the station's service radius" then looks like a
# label rather than a reason the request cannot be made — and the operator's eye
# has nothing to catch on in a window whose other lines are all instructions.
#
# Re-pointed 2026-09-14: the helper is WRAPPED and tinted line by line now, so
# the tint sits inside the loop rather than on a single append. Same rule, and
# mBB2 guards the other half — that the wrap cannot drop it.
p = pathlib.Path("modes/tty/request.go"); s = p.read_text()
old = """			out = append(out, "  "+o.Glyphs().Alert+" "+render.Tint(l, tone))"""
assert old in s, "mBB1"
p.write_text(s.replace(old, """			out = append(out, "  "+o.Glyphs().Alert+" "+l)""", 1))
