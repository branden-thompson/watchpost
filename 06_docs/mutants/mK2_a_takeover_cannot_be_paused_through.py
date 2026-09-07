import pathlib
# A read displaced by a takeover cannot be paused: the enumeration never looks at
# the SUSPENDED stack. [space] while an alert speaks does nothing, and when the
# alert ends settle promotes the read back, because nothing marked it as the
# listener's hold.
#
# RE-ANCHORED (round 3): the three places a read can be are enumerated once in
# reads(), which four callers ask. This deletes the suspended third; mK5 deletes
# the waiting third.
p = pathlib.Path("app/director.go"); s = p.read_text()
old = """	for i := len(d.suspended) - 1; i >= 0; i-- { // bounded by the stack (P10-02)
		if j := d.suspended[i]; live(j) && j.class == narrateRead {
			out = append(out, j)
		}
	}
"""
assert old in s, "mK2"
p.write_text(s.replace(old, "", 1))
