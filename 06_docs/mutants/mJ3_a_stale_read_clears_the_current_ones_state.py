import pathlib
# A read that is winding down clears the bookkeeping of the read that replaced
# it. Fast input then stacks reads: [space], [esc], reopen, [space] on another
# row — the first read's cleanup lands after the second has started, sets
# busy=false while it is playing and takes its mark down, so the next press
# launches a third and the window no longer knows which one to pause.
#
# Reported at UAT 2026-09-03: "It will then be playing two reports - and the
# program gets confused when I hit space which report to pause."
p = pathlib.Path("app/severe_read.go"); s = p.read_text()
old = """		if r.current(gen) { // a stale read clears nothing: the current one owns this state
			r.free()
		}"""
new = """		r.free()"""
assert old in s, "mJ3"
p.write_text(s.replace(old, new, 1))
