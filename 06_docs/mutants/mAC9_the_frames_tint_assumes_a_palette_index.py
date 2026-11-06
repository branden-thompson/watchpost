import pathlib
# The frame's base foreground goes back through the hard-coded `38;5;` prefix, so
# a truecolor TextBase — which is exactly what Watchpost Light has — is emitted as
# a palette index and the theme the finding was reported against is the one theme
# it does not serve (D-108).
p = pathlib.Path("modes/tty/broadcaster.go"); s = p.read_text()
old = """	if params := render.FgSGR(render.Tok(render.TextBase)); render.ColorOn() && params != "" {"""
assert old in s, "mAC9"
p.write_text(s.replace(old, """	if params := "38;5;" + render.Tok(render.TextBase); render.ColorOn() && params != "" {""", 1))
