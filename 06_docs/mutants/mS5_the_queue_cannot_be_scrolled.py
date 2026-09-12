import pathlib
# The queue's window stops following the operator's scrolling, so the bottom of
# the line-up is unreachable — and a line-up the operator cannot reach the bottom
# of is one they cannot manage, which is the whole reason the console exists.
#
# HUM LEAD, 2026-09-11: "that's why we have the vertical scroll bar so that works
# like Observer — that section just needs to be able to scroll up and down."
p = pathlib.Path("modes/tty/broadcaster.go"); s = p.read_text()
old = "\t\toff = min(b.queueOff, len(scroll)-room)"
new = "\t\toff = 0"
assert old in s, "mS5"
p.write_text(s.replace(old, new, 1))
