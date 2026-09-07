import pathlib
# The "All" case inverted. Every fresh install, which is the default path,
# hears nothing at all.
p = pathlib.Path("platform/lineup/fence.go"); s = p.read_text()
old = """	if !f.InForce() {
		return true
	}
"""
new = """	if !f.InForce() {
		return false
	}
"""
assert old in s, "m88"
p.write_text(s.replace(old, new, 1))
