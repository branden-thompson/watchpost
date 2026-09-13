import pathlib
# The pointer's `enter` stops working past slot [9] — which is the ceiling the
# digits already had and the whole reason the pointer became an address (D-105,
# HUM LEAD: "this is more important now for positions [10-14]").
p = pathlib.Path("modes/tty/router.go"); s = p.read_text()
old = """	at := r.broadcaster.lineupSelection()
	if at < 0 {"""
assert old in s, "mAB4"
new = """	at := r.broadcaster.lineupSelection()
	if at+bcScheduledFrom > 9 {
		return r, false
	}
	if at < 0 {"""
p.write_text(s.replace(old, new, 1))
