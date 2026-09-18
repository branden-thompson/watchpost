import pathlib
# The pool gives up ZIP before WX STN, which is the HUM LEAD's order reversed:
# "WX STN should be the first col to get hidden if something doesnt fit" (D-106).
p = pathlib.Path("platform/render/table.go"); s = p.read_text()
old = """		func() { l.noWxStn = l.pool },
		func() { l.spaced = false },
		func() { l.zip = false },"""
assert old in s, "mAC3"
p.write_text(s.replace(old, """		func() { l.zip = false },
		func() { l.spaced = false },
		func() { l.noWxStn = l.pool },""", 1))
