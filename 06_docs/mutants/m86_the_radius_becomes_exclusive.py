import pathlib
# The fence boundary flipped to exclusive, so an alert exactly on the radius is
# on the tape and not in the burst.
p = pathlib.Path("platform/lineup/fence.go"); s = p.read_text()
old = "	if km <= f.RadiusMi*kmPerMi {"
new = "	if km < f.RadiusMi*kmPerMi {"
assert old in s, "m86"
p.write_text(s.replace(old, new, 1))
