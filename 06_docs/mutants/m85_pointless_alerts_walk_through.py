import pathlib
# The tracked tie dropped: every zone-only alert reaches a scoped surface, so
# the burst and the tape disagree about one hazard.
p = pathlib.Path("platform/lineup/fence.go"); s = p.read_text()
old = "		return a.Tracked"
new = "		return true"
assert old in s, "m85"
p.write_text(s.replace(old, new, 1))
