import pathlib
# The exception widened past disasters — the condition kept but made
# unreachable, which is how a rule usually stops applying. A warning a thousand
# miles away comes back, which is what the hard fence exists to stop.
p = pathlib.Path("platform/lineup/fence.go"); s = p.read_text()
old = "	if a.Category != category.Disasters {"
new = "	if a.Category == category.Count {"
assert old in s, "m81"
p.write_text(s.replace(old, new, 1))
