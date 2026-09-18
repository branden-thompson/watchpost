import pathlib
# D-137. The age ladder is filled from the DEFAULT theme's palette instead of
# each theme's own, so every theme wears Watchpost's blue/green/yellow/orange —
# including Monochrome, whose whole purpose is to have none.
#
# THIS IS THE BUG THE FIRST DRAFT SHIPPED and Monochrome's guard caught: a
# registered theme starts from a COPY of the default, so testing the slot for
# EMPTINESS never re-derives it. The mutation restores that emptiness test.
p = pathlib.Path("platform/render/theme.go"); s = p.read_text()
old = "		if _, ok := named[f.dst]; !ok {"
new = "		if t[f.dst] == \"\" {"
assert old in s, "mCJ3"
p.write_text(s.replace(old, new, 1))
