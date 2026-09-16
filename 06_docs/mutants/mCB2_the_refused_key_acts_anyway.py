import pathlib
# D-129, the other half. The window still DRAWS the refusal and still greys the
# chip, and enter acts regardless — a control that says it is unavailable and
# then does the thing, which is worse than one that is simply wrong: the operator
# has been told, in the window, that this cannot happen.
#
# Re-pointed 2026-09-16 (D-157): the ordered run of `if`s became a switch over
# `onSubmit`, the ONE owner of what [enter] means, after the Line-Up Request
# window was found carrying two of the four states. Deleting the refuse ARM is
# the same deletion as before — a definite NO now falls to `default` and ASKS
# AGAIN, so the key the operator was told is unavailable goes to the network.
p = pathlib.Path("modes/tty/modal_location.go"); s = p.read_text()
old = """			case submitRefuse:
				return d, nil
"""
assert old in s, "mCB2"
p.write_text(s.replace(old, "", 1))
