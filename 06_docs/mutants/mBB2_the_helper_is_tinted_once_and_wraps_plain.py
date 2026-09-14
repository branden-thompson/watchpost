import pathlib
# The caveat is tinted as ONE string again, so the wrap puts its tail on a second
# line with no colour on it — "Broadcast Radius" in plain grey under a red
# sentence, which is what the HUM LEAD screenshotted on 2026-09-14.
#
# A TINT IS TWO ESCAPE CODES AT THE ENDS OF A STRING. Anything that splits the
# string between them drops the styling on everything after the break, and the
# window wraps whatever it is given.
p = pathlib.Path("modes/tty/request.go"); s = p.read_text()
old = """		for _, l := range render.WrapText(aside, requestHelperWidth(o)) { // bounded by the text (P10-02)
			out = append(out, "    "+render.Italic(render.Tint(l, tone)))
		}"""
assert old in s, "mBB2"
p.write_text(s.replace(old, """		out = append(out, "    "+render.Italic(render.Tint(aside, tone)))""", 1))
