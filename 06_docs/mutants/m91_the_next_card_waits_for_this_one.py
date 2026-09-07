import pathlib
# The pre-build removed from the settle: the next card is only prepared once the
# air is free, so every cutover pays the 1.03 s build in silence.
p = pathlib.Path("platform/lineup/director.go"); s = p.read_text()
old = """	d, air := d.takeTheAir()
	d, prep := d.prepareNext()"""
new = """	d, air := d.takeTheAir()
	var prep []Effect
	if _, busy := d.lineup.OnAir(); !busy {
		d, prep = d.prepareNext()
	}"""
assert old in s, "m91"
p.write_text(s.replace(old, new, 1))
