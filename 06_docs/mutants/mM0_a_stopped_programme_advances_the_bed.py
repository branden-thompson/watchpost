import pathlib
# The dwell ignores the power state, so a station the listener STOPPED moves its
# bed on five minutes later and starts playing the next location. Nothing follows
# a stop — that is the deck's own rule (d.mode = "") and the Director's, through
# advances(MainTrack).
p = pathlib.Path("platform/lineup/bed.go"); s = p.read_text()
old = """	if !d.advances(MainTrack) {
		return false
	}
"""
assert old in s, "mM0"
p.write_text(s.replace(old, "", 1))
