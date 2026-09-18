import pathlib
# The running order goes back to naming every card "Location Report" whatever it
# actually carries. The operator who asked for FIRE and QUAKE at slot 4 sees the
# same words as a full report — so the one column that says what a card WILL READ
# says nothing, and the request they just made is invisible in the table they made
# it from (R5).
p = pathlib.Path("modes/tty/broadcaster_lineup.go"); s = p.read_text()
old = """	if named := c.Reports.Describe(); named != "" {
		return named
	}
"""
assert old in s, "mBA1"
p.write_text(s.replace(old, "", 1))
