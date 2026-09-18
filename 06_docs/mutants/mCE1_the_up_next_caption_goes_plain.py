import pathlib
# D-132. The UP NEXT caption loses its bold white and goes back to plain text on
# the box's blue ground (D-114), where at fourteen cells it reads as part of the
# fill rather than as the box's name (HUM LEAD, 2026-09-14).
#
# THE GROUND IS WHY THIS NEEDS A MUTANT AT ALL. D-114 paints every row of the
# box, borders included, so the caption row HAS escape codes either way — a test
# asking whether it "has colour" passes against this mutation. Only the bold,
# which the ground never sets, tells the two apart.
p = pathlib.Path("modes/tty/broadcaster_upnext.go"); s = p.read_text()
old = """			cell = render.Bold(render.Tint(
				render.PadTo(centerText(bcUpNextLabel, bcUpNextLabelW), bcUpNextLabelW),
				render.Tok(render.TextBright)))"""
new = """			cell = render.PadTo(centerText(bcUpNextLabel, bcUpNextLabelW), bcUpNextLabelW)"""
assert old in s, "mCE1"
p.write_text(s.replace(old, new, 1))
