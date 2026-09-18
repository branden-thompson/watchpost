import pathlib
# The console goes back to a constant of its own: whatever it inherited, it
# reads the default. The rule is stated, the value is ignored — which is the
# exact shape that shipped, and the reason a setting can look wired and not be.
p = pathlib.Path("modes/tty/broadcaster_pool.go"); s = p.read_text()
old = """	if b.fireBoldMW > 0 {
		return b.fireBoldMW
	}
	return fireBoldDefaultMW"""
assert old in s, "mAQ2"
p.write_text(s.replace(old, "	return fireBoldDefaultMW", 1))
