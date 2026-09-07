import pathlib
# find stops caring whether the lineup holds the card, so a completion for one
# discarded while its build was in flight starts acting on the head of the rail.
p = pathlib.Path("platform/lineup/director.go"); s = p.read_text()
old = """	if c.ID == id {
				return c, true
			}"""
new = """	if c.ID == id || id != "" {
				return c, true
			}"""
assert old in s, "m98"
p.write_text(s.replace(old, new, 1))
