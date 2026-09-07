import pathlib
# MVS-D-67 undone: a per-sequence take-back lifts a bed the rail is holding, so
# the broadcast surges back up between two alerts of one burst — the pumping the
# ruling forbids, and what the code did before the hold existed.
p = pathlib.Path("app/mastercontrol.go"); s = p.read_text()
old = """\tif m.held {
\t\treturn // the rail owns the bed until its tail has played (MVS-D-67)
\t}
"""
assert old in s, "mE4"
p.write_text(s.replace(old, "", 1))
