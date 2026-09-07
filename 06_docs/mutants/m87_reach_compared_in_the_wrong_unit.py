import pathlib
# The unit conversion dropped from the reach comparison — kilometres against
# miles, so every disaster's reach silently shrinks by a factor of 1.6.
p = pathlib.Path("platform/lineup/fence.go"); s = p.read_text()
old = "	return km <= a.ReachMi*kmPerMi"
new = "	return km <= a.ReachMi"
assert old in s, "m87"
p.write_text(s.replace(old, new, 1))
