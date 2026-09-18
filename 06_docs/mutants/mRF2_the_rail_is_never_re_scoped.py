import pathlib
# D-154. `onRefenced` installs the fence and never re-tests the rail against it,
# so everything already admitted keeps the admission the OLD fence gave it. The
# settings change appears to land — the Director holds the new fence — and the
# rail it governs is untouched.
p = pathlib.Path("platform/lineup/air.go"); s = p.read_text()
old = """	d.settings.Fence = ev.Fence
	d = d.refence()
	return d.settle()"""
new = """	d.settings.Fence = ev.Fence
	return d.settle()"""
assert old in s, "mRF2"
p.write_text(s.replace(old, new, 1))
