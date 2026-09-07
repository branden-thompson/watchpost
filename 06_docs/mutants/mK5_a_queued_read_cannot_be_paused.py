import pathlib
# A read asked for WHILE an alert is speaking is queued rather than suspended,
# and the enumeration never looks there — so [space] does nothing and the read
# starts on its own when the alert ends. The same defect as mK2, in the third of
# the three places a read can be.
#
# RE-ANCHORED (round 3) to reads(), the one enumeration four callers ask.
p = pathlib.Path("app/director.go"); s = p.read_text()
old = """	for _, j := range d.waiting { // bounded by the queue (P10-02)
		if live(j) && j.class == narrateRead {
			out = append(out, j)
		}
	}
"""
assert old in s, "mK5"
p.write_text(s.replace(old, "", 1))
