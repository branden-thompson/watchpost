import pathlib
# RE-AIMED 2026-09-12 (D-101): the window follows the POINTER now, and the offset
# is computed at render time where the room is known.  The rule is unchanged — an
# offset that never advances is a list whose bottom the operator cannot reach.
# RE-AIMED 2026-09-12 (D-97): the queue is the TABLE now.  The rule is unchanged —
# an offset pinned to zero is a scroll control the operator can press for ever
# without reaching the bottom of their own line-up.
# The queue's window stops following the operator's scrolling, so the bottom of
# the line-up is unreachable — and a line-up the operator cannot reach the bottom
# of is one they cannot manage, which is the whole reason the console exists.
#
# HUM LEAD, 2026-09-11: "that's why we have the vertical scroll bar so that works
# like Observer — that section just needs to be able to scroll up and down."
p = pathlib.Path("modes/tty/broadcaster_lineup.go"); s = p.read_text()
old = '\t\t\tif at >= room {\n\t\t\t\toff = at - room + 1\n\t\t\t}'
new = '\t\t\t_ = at'
assert old in s, "mS5"
p.write_text(s.replace(old, new, 1))
