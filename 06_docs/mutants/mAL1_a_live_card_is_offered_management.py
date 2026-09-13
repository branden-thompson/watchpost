import pathlib
# The card window offers [P] and [k] on a card that is ON THE AIR. D-45 rules a
# live card "Management Locked" and the card's own STATUS line says so — so the
# window would show that line and offer the two keys it rules out (D-118).
p = pathlib.Path("modes/tty/broadcaster_detail.go"); s = p.read_text()
old = """	return c.ID != "" && c.State != lineup.OnAir"""
assert old in s, "mAL1"
p.write_text(s.replace(old, """	return c.ID != \"\"""", 1))
