import pathlib
# The no-origin rule inverted into the silent fallback it exists to prevent:
# a scoped surface quietly showing the global stack.
p = pathlib.Path("platform/lineup/fence.go"); s = p.read_text()
old = """	if !f.HasOrigin {
		return false
	}
"""
new = """	if !f.HasOrigin {
		return true
	}
"""
assert old in s, "m84"
p.write_text(s.replace(old, new, 1))
