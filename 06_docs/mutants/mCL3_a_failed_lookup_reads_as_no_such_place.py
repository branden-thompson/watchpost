import pathlib
# D-151. A lookup that could not be MADE — a timeout, a cancelled context —
# reports as a genuine no-match again. The window then tells the operator a real
# location does not exist AND disables the key that would retry it.
p = pathlib.Path("app/pool.go"); s = p.read_text()
old = "			return snapshot.LocationRef{}, false, false, !isLookupFailure(err)"
new = "			return snapshot.LocationRef{}, false, false, true"
assert old in s, "mCL3"
p.write_text(s.replace(old, new, 1))
