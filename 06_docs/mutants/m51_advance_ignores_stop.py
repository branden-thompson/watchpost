import pathlib
# A cycle that ends after the listener pressed STOP still moves the bed on, so
# the station starts itself again playing the next location. Nothing follows a
# stop — that was the deck's rule (d.mode = "") and it is the Director's now.
#
# RE-ANCHORED (T3.2b) to the CYCLE-END path; mM0 carries the same rule on the
# dwell path. Both are needed: a stopped station can be reached either by a tick
# or by a cycle that was already in flight when the stop landed.
p = pathlib.Path("platform/lineup/bed.go"); s = p.read_text()
old = """	if !d.advances(MainTrack) {
		return d, nil // stopped: nothing follows a listener's stop
	}
"""
assert old in s, "m51"
p.write_text(s.replace(old, "", 1))
