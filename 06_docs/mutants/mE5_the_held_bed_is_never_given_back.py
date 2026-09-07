import pathlib
# The hold is never cleared, so the broadcast never comes back after a rail
# drain — worse than pumping, because nothing recovers it.
p = pathlib.Path("app/mastercontrol.go"); s = p.read_text()
old = "\tm.held = false\n"
assert old in s, "mE5"
p.write_text(s.replace(old, "\tm.held = true\n", 1))
