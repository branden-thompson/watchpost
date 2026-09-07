import pathlib
# The busy check dropped: whatever is next takes the air over whatever is
# already reading, so two voices speak at once.
p = pathlib.Path("platform/lineup/director.go"); s = p.read_text()
old = """	if _, busy := d.lineup.OnAir(); busy {
		return d, nil, false
	}
"""
assert old in s, "m94"
p.write_text(s.replace(old, "", 1))
