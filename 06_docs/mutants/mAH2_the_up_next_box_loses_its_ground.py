import pathlib
# The UP NEXT box goes back to no ground of its own, so the one card the operator
# READS sits on whatever the frame is painted on (D-114, HUM LEAD 2026-09-13:
# "the UP Next Box probably needs a bkg color other than none").
p = pathlib.Path("modes/tty/broadcaster_upnext.go"); s = p.read_text()
old = """	for i, r := range out { // bounded by the box's height (P10-02)
		out[i] = render.TintKeeping(r, ground)
	}
"""
assert old in s, "mAH2"
p.write_text(s.replace(old, "", 1))
