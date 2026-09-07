import pathlib
# The zero value flipped, so a station puts a report to air on launch that
# nobody asked for.
p = pathlib.Path("platform/lineup/power.go"); s = p.read_text()
old = """	Stopped Power = iota
	Running"""
new = """	Running Power = iota
	Stopped"""
assert old in s, "mA4"
p.write_text(s.replace(old, new, 1))
