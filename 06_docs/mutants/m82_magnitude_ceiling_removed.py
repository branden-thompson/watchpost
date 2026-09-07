import pathlib
# The magnitude clamp removed, so a malformed feed row claiming M12 buys 2,896
# miles of reach and admits the ruling's own counter-example.
p = pathlib.Path("platform/lineup/fence.go"); s = p.read_text()
old = """	if mag > quakeReachTo {
		mag = quakeReachTo
	}
"""
assert old in s, "m82"
p.write_text(s.replace(old, "", 1))
