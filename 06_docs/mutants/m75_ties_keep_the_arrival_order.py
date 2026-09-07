import pathlib
# The final tie-break removed, so the order stops being total and the order the
# alerts happened to arrive in reaches the listener.
p = pathlib.Path("platform/lineup/plan.go"); s = p.read_text()
old = "		return p[i].arrival.ID < p[j].arrival.ID"
new = "		return false"
assert old in s, "m75"
p.write_text(s.replace(old, new, 1))
