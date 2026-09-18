import pathlib
# The console stops arming the theme's foreground, so every glyph nothing tints
# takes the TERMINAL's default instead — white on a dark terminal, whatever theme
# the app is wearing. It looks right under the dark themes and makes the UP NEXT
# box vanish on Watchpost Light (D-108).
p = pathlib.Path("modes/tty/broadcaster.go"); s = p.read_text()
old = """	if render.ColorOn() {
		out = render.TintKeeping(out, render.FgSGR(render.Tok(render.TextBase)))
	}"""
assert old in s, "mAC8"
p.write_text(s.replace(old, "", 1))
