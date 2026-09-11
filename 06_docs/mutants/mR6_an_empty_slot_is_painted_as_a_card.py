import pathlib
# A slot holding no card is painted anyway, so the waiting placeholder and the
# LIVE slot on a station at rest are drawn as cards that are not there.
p = pathlib.Path("modes/tty/broadcaster.go"); s = p.read_text()
old = "\tif c.ID == \"\" {\n\t\treturn \"\"\n\t}\n"
assert old in s, "mR6"
p.write_text(s.replace(old, "", 1))
