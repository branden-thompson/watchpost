import pathlib
# D-137. The ladder's 5-minute threshold moves to 10, so everything under ten
# minutes reads as GREEN — "fresh" — and the yellow rung disappears. The
# operator is told data is current for twice as long as the ruling allows.
#
# A THRESHOLD SHIFT, NOT A DELETION: every duration still maps to a rung and the
# card still paints one, so only the LADDER is wrong. That is the failure this
# rule is about.
p = pathlib.Path("modes/tty/broadcaster_manifest.go"); s = p.read_text()
old = """	case d < 5*time.Minute:
		return render.DataFresh"""
new = """	case d < 10*time.Minute:
		return render.DataFresh"""
assert old in s, "mCJ2"
p.write_text(s.replace(old, new, 1))
