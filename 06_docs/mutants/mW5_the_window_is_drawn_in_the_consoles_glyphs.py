import pathlib
# The card window's title and body are spelt in the CONSOLE's glyph vocabulary
# rather than the window's own, which is the escape the --ascii parity gate found
# when this window was first built: under --ascii the title carries a bullet and the
# chips carry two arrows, none of which has an ASCII form. The renderer takes the
# window's Opts for exactly this reason (Dashboard.cardRows).
p = pathlib.Path("modes/tty/broadcaster_detail.go"); s = p.read_text()
old = """	return c.ID, func(o render.Opts) (string, []string) {
		return cardTitle(c, o.Glyphs()), b.detailBody(o, c)
	}, true"""
new = """	return c.ID, func(render.Opts) (string, []string) {
		return cardTitle(c, b.opts().Glyphs()), b.detailBody(b.opts(), c)
	}, true"""
assert old in s, "mW5"
p.write_text(s.replace(old, new, 1))
