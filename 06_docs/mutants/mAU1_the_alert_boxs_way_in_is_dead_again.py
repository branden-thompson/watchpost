import pathlib
# `[A]` stops opening the takeover box's card. The control is still DRAWN at the
# bottom of the box — `burstBody` appends it unconditionally — so the box goes on
# advertising a way in that does nothing, which is how this shipped and what the
# HUM LEAD found in UAT: "[A] Details / Full Read / Manage in the alert window
# doesn't currently work".
#
# A PAINTED CONTROL THAT DOES NOTHING IS WORSE THAN AN ABSENT ONE: the operator
# presses it, sees no window, and reasonably concludes the report has nothing to
# show them about a hazard that is about to interrupt the programme.
p = pathlib.Path("modes/tty/router.go"); s = p.read_text()
# THE WHOLE CASE, NOT ITS FIRST TWO LINES. The block gained a third when D-127
# taught the window to wear the card's ground, and deleting a prefix of a case
# left the remaining statement orphaned before the next `case` — uncompilable,
# which `mutant-check` reports as no evidence either way. The ANCHOR still
# matched, so the fast gate passed it: an anchor that matches is not the same as
# a mutation that is still valid, and that is why the slow gate is not optional.
old = """	case key == "A":
		id, rows, ok = r.broadcaster.alertDetail()
		ground = r.broadcaster.alertWindowGround()
"""
assert old in s, "mAU1"
p.write_text(s.replace(old, "", 1))
