import pathlib
# D-129, the other half, re-pointed for D-130. The window still DRAWS the refusal
# and still greys the chip, and enter acts regardless — a control that says it is
# unavailable and then does the thing, which is worse than one that is simply
# wrong: the operator has been told, in the window, that this cannot happen.
#
# THE SETTLED CHECK IS WHAT IS DELETED, not the whole branch. "Not yet known" must
# still let enter through (D-130) — the mutation keeps that and drops only the
# definite NO, which is exactly the refusal the operator was shown.
p = pathlib.Path("modes/tty/modal_location.go"); s = p.read_text()
# Re-pointed 2026-09-16 (D-151): the gate gained a fourth state — a check that
# could not be MADE is no longer a refusal — so the definite-no arm now asks
# `asked` too. Same rule, same refusal.
old = """			if d.addLocate.settled() && d.addLocate.asked && !d.addLocate.reachable() {
				return d, nil
			}"""
new = """			if false {
				return d, nil
			}"""
assert old in s, "mCB2"
p.write_text(s.replace(old, new, 1))
