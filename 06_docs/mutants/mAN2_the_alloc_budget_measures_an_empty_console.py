import pathlib
# The allocation budget goes back to measuring a console with NO POOL and NO
# SNAPSHOT, so no row joins its weather and the pool table draws nothing — the
# budget guards the cheap path while the expensive one is never in it (D-120).
p = pathlib.Path("modes/tty/broadcaster_alloc_test.go"); s = p.read_text()
old = """	b := loadedConsole(t, ttytest.LoadedPoolSize)"""
assert old in s, "mAN2"
p.write_text(s.replace(old, """	b := bcWith(t, ttytest.Card(t, "a", "OCEANSIDE"), ttytest.Card(t, "b", "BONSALL"))""", 1))
