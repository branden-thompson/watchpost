import pathlib
# D-143 (0.19.0 W1.2): the byte bound removes the oldest files first.
# Killed by TestADatasetIsHeldToItsByteBound.
p = pathlib.Path("platform/history/history.go"); s = p.read_text()
old = "\t\t\treturn out[i].at.Before(out[j].at)\n"
assert old in s, "m116"
p.write_text(s.replace(old, "\t\t\treturn out[i].at.After(out[j].at)\n"))
