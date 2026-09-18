import pathlib
# The alert rail's window stops going through the one builder and composes its
# own title. Today that is the same string, which is the point: the two windows
# agree by COINCIDENCE rather than by construction, and the next change to
# `cardTitle` moves one and not the other. The operator then reads a different
# heading depending on which key opened the same card.
p = pathlib.Path("modes/tty/broadcaster_detail.go"); s = p.read_text()
old = """	return b.cardWindowFor(rail[0])
}"""
new = """	c := rail[0]
	return c.ID, func(o render.Opts) (string, []string) {
		return "LOCATION REPORT", b.detailBody(o, c)
	}, true
}"""
assert old in s, "mAU2"
p.write_text(s.replace(old, new, 1))
