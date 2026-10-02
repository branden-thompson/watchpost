import pathlib
# The fence boundary flipped to exclusive, so an alert exactly on the radius is
# on the tape and not in the burst.
p = pathlib.Path("platform/lineup/fence.go"); s = p.read_text()
old = "	if km <= units.KmOf(f.RadiusMi) {"
new = "	if km < units.KmOf(f.RadiusMi) {"
assert old in s, "m86"
p.write_text(s.replace(old, new, 1))
