import pathlib
# D-148 through D-159's extraction. The swap ignores an open text field again, so
# typing a place name loses every capital O and B and `O` swaps the surface
# MID-WORD: "Oceanside" -> "ceanside", "Bonsall" -> "onsall". Those are the HUM
# LEAD's own station and the hyper-local case D-130 was ruled for.
p = pathlib.Path("modes/tty/router.go"); s = p.read_text()
old = """	case actSwapObserver:
		if r.typingInAWindow(k) {
			break
		}"""
new = """	case actSwapObserver:
		if false {
			break
		}"""
assert old in s, "mRT2"
p.write_text(s.replace(old, new, 1))
