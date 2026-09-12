import pathlib
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
old = '\t\toff = min(b.queueOff, len(lines)-room)'
new = '\t\toff = 0'
assert old in s, "mS5"
p.write_text(s.replace(old, new, 1))
