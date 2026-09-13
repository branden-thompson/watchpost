import pathlib
# The pool's row stops carrying the fire and seismic marks — the two that
# `fillPoolWeather` never copied, so they were blank by construction on the table
# an operator decides from. A blank mark reads as "nothing is happening there",
# which is the wrong answer told confidently (D-113).
#
# WRITTEN AGAINST THE FORMATTED SOURCE. The first draft matched the lines as
# typed and gofmt had aligned the trailing comments into a column, so it applied
# to nothing — UNAPPLIED, which is run.sh refusing to report a rule as measured
# when nothing was measured.
p = pathlib.Path("modes/tty/body.go"); s = p.read_text()
old = "\t\tFire:       fireCount(loc.Fire),                    // B5 / UAT 110: n◆\n\t\tFireHot:    fireHot(loc.Fire.Hotspots, fireBoldMW), // B5\n\t\tSeismic:    seismicRowLevel(loc.Seismic),           // 0.11.0: the strongest quake's felt-band glyph\n"
assert old in s, "mAG1"
p.write_text(s.replace(old, "", 1))
