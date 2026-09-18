import pathlib
# The card window ignores the ground it was handed and floats on the standard
# modal tone. The takeover box stays [w]-orange and the window it opens goes back
# to blue — the same hazard, two colours, one keypress apart. That is what the HUM
# LEAD reported: "that modal should MATCH the tone, not be the blue that is
# currently is."
p = pathlib.Path("modes/tty/view.go"); s = p.read_text()
old = """		if d.cardGround != "" {
			bg = d.cardGround
		}
"""
assert old in s, "mAV1"
p.write_text(s.replace(old, "", 1))
