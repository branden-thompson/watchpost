import pathlib
# The overrun removed: the emergency orders become five of the five, and a
# listener in a seven-order emergency stops hearing two of them.
p = pathlib.Path("platform/lineup/plan.go"); s = p.read_text()
old = """		emergency := c.arrival.Category == category.Emergency
		if !emergency && spent >= max {
			continue
		}"""
new = """		if spent >= max {
			continue
		}"""
assert old in s, "m71"
s = s.replace(old, new, 1)
# category is still imported for the ladder, so nothing else has to move.
p.write_text(s)
