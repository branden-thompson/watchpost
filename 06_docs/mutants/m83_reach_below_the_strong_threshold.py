import pathlib
# Every quake buys reach, however ordinary. The listener's fence stops meaning
# anything for the whole Disasters category.
p = pathlib.Path("platform/lineup/fence.go"); s = p.read_text()
old = """	if mag < QuakeReachFrom {
		return 0
	}
"""
assert old in s, "m83"
p.write_text(s.replace(old, "", 1))
