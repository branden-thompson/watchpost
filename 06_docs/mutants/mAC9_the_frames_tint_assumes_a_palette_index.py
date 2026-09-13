import pathlib
# The frame's base foreground goes back through the hard-coded `38;5;` prefix, so
# a truecolor TextBase — which is exactly what Watchpost Light has — is emitted as
# a palette index and the theme the finding was reported against is the one theme
# it does not serve (D-108).
p = pathlib.Path("modes/tty/broadcaster.go"); s = p.read_text()
old = """		out = render.TintKeeping(out, render.FgSGR(render.Tok(render.TextBase)))"""
assert old in s, "mAC9"
p.write_text(s.replace(old, """		out = render.TintDefault(out)""", 1))
