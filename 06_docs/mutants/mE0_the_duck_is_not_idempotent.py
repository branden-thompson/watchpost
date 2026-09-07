import pathlib
# The bed dips every time it is asked. The narration arbiter gives way when a
# sequence takes the air and the Director will do the same around a whole rail
# drain (MVS-D-67), so between them the listener hears the broadcast step down
# again under a read already in progress.
p = pathlib.Path("app/mastercontrol.go"); s = p.read_text()
old = """\tif m.ducked {
\t\treturn
\t}
\tm.v.duck()"""
new = """\tm.v.duck()"""
assert old in s, "mE0"
p.write_text(s.replace(old, new, 1))
