# The fail-closed guard inverted: a power outside the registry starts putting
# programme to air.
# RE-ANCHORED AT T3.9. The guard MOVED, and moving it was the fix: it used to
# sit BELOW the alert rail's exemption, so an undeclared power still advanced the
# rail — the one track the "fail closed" comment was written to protect. Found by
# a test that walks the power registry rather than a hand-written list.
import pathlib
p = pathlib.Path("platform/lineup/power.go"); s = p.read_text()
old = """	if d.power < 0 || d.power >= numPowers {
		return false
	}
	if d.power == OffAir {"""
new = """	if false {
		return false
	}
	if d.power == OffAir {"""
assert old in s, "mA7"
p.write_text(s.replace(old, new, 1))
