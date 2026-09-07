import pathlib
# takeBack lifts whether or not anything was given way, so a stray restore
# raises a broadcast nobody dipped — and, worse, lifts one that another sequence
# is still reading over.
p = pathlib.Path("app/mastercontrol.go"); s = p.read_text()
old = """\tif !m.ducked {
\t\treturn
\t}
\tm.v.restore()"""
new = """\tm.v.restore()"""
assert old in s, "mE1"
p.write_text(s.replace(old, new, 1))
