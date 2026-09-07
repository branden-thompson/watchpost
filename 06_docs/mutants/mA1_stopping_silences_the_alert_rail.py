import pathlib
# The asymmetry removed: a stopped radio stops the hazards too, so a listener
# who paused the programme stops hearing warnings.
p = pathlib.Path("platform/lineup/power.go"); s = p.read_text()
old = """	if t == AlertRail {
		return true
	}
"""
assert old in s, "mA1"
p.write_text(s.replace(old, "", 1))
