import pathlib
# The pool's row stops carrying the fire and seismic marks — the two that
# `fillPoolWeather` never copied, so they were blank by construction on the table
# an operator decides from. A blank mark reads as "nothing is happening there",
# which is the wrong answer told confidently (D-113).
p = pathlib.Path("modes/tty/body.go"); s = p.read_text()
old = """		Fire:       fireCount(loc.Fire),              // B5 / UAT 110: n◆
		FireHot:    fireHot(loc.Fire.Hotspots, fireBoldMW), // B5
		Seismic:    seismicRowLevel(loc.Seismic),     // 0.11.0: the strongest quake's felt-band glyph"""
assert old in s, "mAG1"
p.write_text(s.replace(old, "", 1))
