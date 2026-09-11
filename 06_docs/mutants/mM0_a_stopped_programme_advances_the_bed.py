import pathlib
# The dwell ignores the monitor's power, so a rotation the operator STOPPED moves
# its bed on five minutes later and starts playing the next location. Nothing
# follows a stop — that is the deck's own rule (d.mode = "") and the Director's,
# through advancesMonitor().
#
# RE-ANCHORED AT D-74. The rule is unchanged and the gate it lives on is not:
# the bed is the MONITOR's rotation, so it asks `advancesMonitor()` — the
# OPERATOR's own power — rather than the STATION's. It asked
# `advances(MainTrack)` when the two powers were one field, which is the
# conflation D-74 removed.
p = pathlib.Path("platform/lineup/bed.go"); s = p.read_text()
old = """	if !d.advancesMonitor() {
		return false
	}
"""
assert old in s, "mM0"
p.write_text(s.replace(old, "", 1))
