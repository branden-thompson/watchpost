import pathlib
# The relay selector takes the bare arrows back, so the two controls the reference
# draws with arrows — the bed and the card's PRESENTER — are one key again, and
# the one the operator reaches depends on which case the router hits first
# (D-111).
p = pathlib.Path("modes/tty/router.go"); s = p.read_text()
old = """		actBedPrev: {Keys: []string{"shift+left"}, Help: "Previous Relay"},
		actBedNext: {Keys: []string{"shift+right"}, Help: "Next Relay"},"""
assert old in s, "mAE1"
new = """		actBedPrev: {Keys: []string{"left"}, Help: "Previous Relay"},
		actBedNext: {Keys: []string{"right"}, Help: "Next Relay"},"""
p.write_text(s.replace(old, new, 1))
