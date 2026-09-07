import pathlib
# A tick out of order drags the clock back, re-ageing every disaster and
# quietly re-sorting the next burst.
p = pathlib.Path("platform/lineup/director.go"); s = p.read_text()
old = """	if ev.Now.After(d.now) {
		d.now = ev.Now
	}"""
new = """	d.now = ev.Now"""
assert old in s, "m97"
p.write_text(s.replace(old, new, 1))
