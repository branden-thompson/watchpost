import pathlib
# The recover removed. A panicking executor takes the process with it, which
# under Approach C means both tracks and the bed.
p = pathlib.Path("app/pump.go"); s = p.read_text()
old = """		r := recover()
		if r == nil {
			return
		}
"""
new = """		var r any
		if r == nil {
			return
		}
"""
assert old in s, "mB3"
p.write_text(s.replace(old, new, 1))
